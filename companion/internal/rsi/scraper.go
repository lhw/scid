package rsi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const (
	// DefaultProfileBaseURL is the production RSI citizen profile base URL.
	DefaultProfileBaseURL = "https://robertsspaceindustries.com/en/citizens/"
	rsiTimeout            = 10 * time.Second
	userAgent             = "SCID-Companion/1.0 (Star Citizen Identity Provider; unofficial fansite tool)"
)

// RSIScraper is the interface for fetching RSI public profile and org data.
// It is satisfied by *Scraper and can be replaced with a mock in tests.
type RSIScraper interface {
	FetchProfile(ctx context.Context, handle string) (*Profile, error)
	FetchOrgs(ctx context.Context, handle string) ([]OrgInfo, error)
}

// Profile holds the fields we extract from a public RSI profile page.
type Profile struct {
	Handle            string
	Bio               string
	HasDeveloperBadge bool
	CitizenRecord     string // e.g. "#40746"
	Enlisted          string // e.g. "Oct 18, 2012"
	AvatarURL         string // absolute URL to the user's RSI avatar image
}

// Scraper fetches and parses RSI public profile pages.
type Scraper struct {
	baseURL string
	client  *http.Client
}

// New creates a new Scraper targeting the production RSI website.
func New() *Scraper {
	return NewWithBaseURL(DefaultProfileBaseURL)
}

// newWithBaseURL creates a Scraper with an overridden base URL (for testing).
func NewWithBaseURL(baseURL string) *Scraper {
	return &Scraper{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: rsiTimeout,
		},
	}
}

// FetchProfile fetches and parses the RSI profile page for the given handle.
func (s *Scraper) FetchProfile(ctx context.Context, handle string) (*Profile, error) {
	profileURL := s.baseURL + handle
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, profileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("RSI handle %q not found", handle)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RSI responded %d for handle %q", resp.StatusCode, handle)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	return parseProfile(handle, doc), nil
}

// ContainsToken reports whether the profile bio contains the given token string.
func ContainsToken(profile *Profile, token string) bool {
	return strings.Contains(profile.Bio, token)
}

// parseProfile walks the parsed HTML tree and extracts Profile fields.
func parseProfile(handle string, doc *html.Node) *Profile {
	p := &Profile{Handle: handle}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch {
			case !p.HasDeveloperBadge && hasDeveloperBadge(n):
				p.HasDeveloperBadge = true

			case hasCSSClass(n, "bio") || hasDataAttr(n, "bio"):
				// The bio is typically inside a <div class="bio"> or similar.
				p.Bio = strings.TrimSpace(extractText(n))

			case hasCSSClass(n, "citizen-record") || hasCSSClass(n, "record-number"):
				// UEE Citizen Record block — look for a child value element.
				record := extractValueText(n)
				if record != "" && p.CitizenRecord == "" {
					p.CitizenRecord = strings.TrimSpace(record)
				}

			case hasCSSClass(n, "citizen-stat") || hasCSSClass(n, "entry"):
				// Generic stat entries: look for a label child that says "Enlisted"
				// and extract the adjacent value.
				label, value := extractLabelValue(n)
				lower := strings.ToLower(label)
				switch {
				case strings.Contains(lower, "enlisted") || strings.Contains(lower, "enlistment"):
					if p.Enlisted == "" {
						p.Enlisted = strings.TrimSpace(value)
					}
				case strings.Contains(lower, "record") || strings.Contains(lower, "citizen"):
					if p.CitizenRecord == "" && strings.Contains(value, "#") {
						p.CitizenRecord = strings.TrimSpace(value)
					}
				}

			// Profile avatar: look for img tags inside known avatar containers.
			case p.AvatarURL == "" && isAvatarContainer(n):
				if src := findFirstImgSrc(n); src != "" {
					p.AvatarURL = absoluteURL(src)
				}
			}
		}

		// Also look for spans/divs with data-label or aria-label attributes that
		// describe individual fields, covering alternate RSI page layouts.
		if n.Type == html.ElementNode {
			label := getAttr(n, "data-label")
			if label == "" {
				label = getAttr(n, "aria-label")
			}
			lower := strings.ToLower(label)
			if strings.Contains(lower, "uee citizen record") || strings.Contains(lower, "citizen record") {
				if p.CitizenRecord == "" {
					p.CitizenRecord = strings.TrimSpace(extractText(n))
				}
			}
			if strings.Contains(lower, "enlisted") {
				if p.Enlisted == "" {
					p.Enlisted = strings.TrimSpace(extractText(n))
				}
			}

			// Fallback avatar: any <img> whose src contains "/media/" and is
			// plausibly a profile picture (not a banner/background image).
			if p.AvatarURL == "" && n.Type == html.ElementNode && n.Data == "img" {
				src := getAttr(n, "src")
				if strings.Contains(src, "/media/") && !strings.Contains(src, "banner") {
					p.AvatarURL = absoluteURL(src)
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(doc)

	// Normalise CitizenRecord to the integer-only form (strip leading "#").
	p.CitizenRecord = strings.TrimLeft(strings.TrimSpace(p.CitizenRecord), "#")

	return p
}

// hasDeveloperBadge reports whether an element looks like RSI's public
// developer badge for staff accounts.
func hasDeveloperBadge(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}

	if n.Data == "img" {
		if src := strings.ToLower(getAttr(n, "src")); strings.Contains(src, "developer.png") {
			return true
		}
	}

	for _, attr := range []string{"alt", "title", "aria-label", "data-label"} {
		if strings.Contains(strings.ToLower(getAttr(n, attr)), "developer") {
			return true
		}
	}

	return false
}

// hasCSSClass reports whether an element node has the given class among its
// space-separated class attribute values.
func hasCSSClass(n *html.Node, class string) bool {
	for _, a := range n.Attr {
		if a.Key == "class" {
			for _, c := range strings.Fields(a.Val) {
				if c == class {
					return true
				}
			}
		}
	}
	return false
}

// hasDataAttr reports whether an element has a data-<name> attribute.
func hasDataAttr(n *html.Node, name string) bool {
	key := "data-" + name
	for _, a := range n.Attr {
		if a.Key == key {
			return true
		}
	}
	return false
}

// getAttr returns the value of the named attribute, or "".
func getAttr(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}

// extractText recursively collects all text content under a node.
func extractText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

// extractValueText looks for an immediate child element with class "value" or
// "content" and returns its text.
func extractValueText(n *html.Node) string {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode &&
			(hasCSSClass(c, "value") || hasCSSClass(c, "content") || hasCSSClass(c, "number")) {
			return extractText(c)
		}
	}
	return extractText(n)
}

// extractLabelValue finds the first child with class "label"/"name" and the
// first child with class "value"/"content", returning their texts.
func extractLabelValue(n *html.Node) (label, value string) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}
		if hasCSSClass(c, "label") || hasCSSClass(c, "name") || hasCSSClass(c, "label-value") {
			label = extractText(c)
		} else if hasCSSClass(c, "value") || hasCSSClass(c, "content") {
			value = extractText(c)
		}
	}
	return
}

// isAvatarContainer returns true if the element is a known RSI avatar wrapper.
func isAvatarContainer(n *html.Node) bool {
	for _, class := range []string{"thumb", "profile-thumb", "avatar", "profile-pic", "profile-image", "overview-image"} {
		if hasCSSClass(n, class) {
			return true
		}
	}
	return false
}

// findFirstImgSrc returns the src of the first <img> descendant.
func findFirstImgSrc(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "img" {
		return getAttr(n, "src")
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if src := findFirstImgSrc(c); src != "" {
			return src
		}
	}
	return ""
}

// absoluteURL turns a root-relative RSI path into an absolute URL.
func absoluteURL(src string) string {
	if strings.HasPrefix(src, "http") {
		return src
	}
	return "https://robertsspaceindustries.com" + src
}

// allowedImageHosts is the set of hostnames from which SCID will fetch images.
// All URLs scraped from RSI pages should resolve to one of these domains.
var allowedImageHosts = map[string]struct{}{
	"robertsspaceindustries.com":       {},
	"www.robertsspaceindustries.com":   {},
	"media.robertsspaceindustries.com": {},
	"cdn.robertsspaceindustries.com":   {},
}

// IsAllowedImageURL validates that a URL points to a known RSI host over HTTPS.
// Returns false for internal/private networks, non-HTTPS, or unknown hosts.
func IsAllowedImageURL(rawURL string) bool {
	if rawURL == "" {
		return false
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if u.Scheme != "https" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	_, ok := allowedImageHosts[host]
	return ok
}
