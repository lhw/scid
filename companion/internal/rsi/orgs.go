package rsi

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

const (
	// DefaultOrgsBaseURL is the RSI citizen organizations page base URL.
	DefaultOrgsBaseURL = "https://robertsspaceindustries.com/en/citizens/"
	orgsPathSuffix     = "/organizations"
)

// OrgInfo holds the fields we extract from a user's RSI organizations page.
type OrgInfo struct {
	// SID is the organization's Spectrum Identification code (e.g. "SPAWO").
	SID string
	// Name is the human-readable organization name (e.g. "Star Pilgrim Alliance").
	Name string
	// LogoURL is the absolute URL of the org logo image (may be empty).
	LogoURL string
	// RankName is the member's rank within the org (e.g. "Member", "Officer").
	RankName string
	// IsMain indicates whether this is the member's primary (main) org.
	IsMain bool
}

// FetchOrgs fetches and parses the organizations page for the given RSI handle.
// It returns an empty slice (not an error) when the handle has no org memberships.
func (s *Scraper) FetchOrgs(ctx context.Context, handle string) ([]OrgInfo, error) {
	orgsURL := s.baseURL + handle + orgsPathSuffix
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, orgsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch orgs page: %w", err)
	}
	defer resp.Body.Close()

	// A 404 simply means no org page exists for this handle — treat as empty.
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RSI orgs page responded %d for handle %q", resp.StatusCode, handle)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	return parseOrgs(doc), nil
}

// parseOrgs extracts org info from the parsed HTML tree.
//
// RSI's org page structure (as of 2026):
//
//	<div class="box-content org main visibility-V">   <!-- or "affiliation" -->
//	  <div class="inner-bg clearfix">
//	    <div class="left-col">
//	      <div class="inner clearfix">
//	        <div class="thumb">
//	          <a href="/orgs/AEON"><img src="...logo..."/></a>
//	        </div>
//	        <div class="info">
//	          <p class="entry"><a href="/orgs/AEON" class="value">Æon</a></p>
//	          <p class="entry">
//	            <span class="label">Spectrum Identification (SID)</span>
//	            <strong class="value">AEON</strong>
//	          </p>
//	          <p class="entry">
//	            <span class="label">Organization rank</span>
//	            <strong class="value">Vice President</strong>
//	          </p>
//	        </div>
//	      </div>
//	    </div>
//	  </div>
//	</div>
//
// The outer div has class "org" along with "main" or "affiliation".
func parseOrgs(doc *html.Node) []OrgInfo {
	var orgs []OrgInfo

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && hasCSSClass(n, "org") &&
			(hasCSSClass(n, "main") || hasCSSClass(n, "affiliation")) {
			if org, ok := parseOrgBox(n); ok {
				orgs = append(orgs, org)
				return // don't descend into children of this box
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return orgs
}

// parseOrgBox extracts org info from a <div class="box-content org main/affiliation"> node.
func parseOrgBox(n *html.Node) (OrgInfo, bool) {
	org := OrgInfo{
		IsMain: hasCSSClass(n, "main"),
	}

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			switch {
			case hasCSSClass(node, "thumb"):
				// Logo and SID from the <a href="/orgs/SID"><img src="..."/></a> inside thumb.
				if a := findFirst(node, "a"); a != nil {
					if org.SID == "" {
						org.SID = sidFromHref(getAttr(a, "href"))
					}
					if org.LogoURL == "" {
						if src := findFirstImgSrc(node); src != "" {
							org.LogoURL = absoluteURL(src)
						}
					}
				}
				return // don't descend further into thumb

			case hasCSSClass(node, "info"):
				// Iterate <p class="entry"> children.
				parseInfoEntries(node, &org)
				return // don't descend further into info
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)

	// Fallback: check any /orgs/ link in the whole box if we still have no SID.
	if org.SID == "" {
		org.SID = findOrgSIDAnywhere(n)
	}
	if org.SID == "" {
		return OrgInfo{}, false
	}
	return org, true
}

// parseInfoEntries processes the <p class="entry"> elements inside a <div class="info">.
func parseInfoEntries(infoNode *html.Node, org *OrgInfo) {
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && hasCSSClass(n, "entry") {
			// Get the label text, if any.
			labelText := ""
			if label := findFirstWithClass(n, "span", "label"); label != nil {
				labelText = strings.ToLower(extractText(label))
			}

			if a := findFirstWithClass(n, "a", "value"); a != nil {
				// <a class="value" href="/orgs/SID">Org Name</a>
				if org.Name == "" {
					org.Name = strings.TrimSpace(extractText(a))
				}
				if org.SID == "" {
					org.SID = sidFromHref(getAttr(a, "href"))
				}
			} else if strong := findFirstWithClass(n, "strong", "value"); strong != nil {
				val := strings.TrimSpace(extractText(strong))
				switch {
				case strings.Contains(labelText, "sid") || strings.Contains(labelText, "spectrum"):
					org.SID = val
				case strings.Contains(labelText, "rank"):
					org.RankName = val
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(infoNode)
}

// findFirstWithClass returns the first descendant element matching tag and CSS class.
func findFirstWithClass(n *html.Node, tag, class string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag && hasCSSClass(n, class) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findFirstWithClass(c, tag, class); found != nil {
			return found
		}
	}
	return nil
}

// sidFromHref extracts the org SID from hrefs like "/orgs/SPAWO" or "/en/orgs/SPAWO".
func sidFromHref(href string) string {
	parts := strings.Split(href, "/orgs/")
	if len(parts) < 2 {
		return ""
	}
	sid := strings.Split(strings.TrimSpace(parts[len(parts)-1]), "/")[0]
	return strings.ToUpper(strings.TrimSpace(sid))
}

// findOrgSIDAnywhere searches descendant <a> tags for any /orgs/ href.
func findOrgSIDAnywhere(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "a" {
		if sid := sidFromHref(getAttr(n, "href")); sid != "" {
			return sid
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if sid := findOrgSIDAnywhere(c); sid != "" {
			return sid
		}
	}
	return ""
}

// findFirst returns the first descendant element with the given tag name.
func findFirst(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findFirst(c, tag); found != nil {
			return found
		}
	}
	return nil
}
