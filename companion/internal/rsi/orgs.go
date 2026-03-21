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
// RSI's org page structure (as of 2025):
//
//	<div class="org-row main-org"> or <div class="org-row affiliation">
//	  <div class="logo"> <img src="..."> </div>
//	  <div class="name"> <a href="/orgs/SPAWO">Star Pilgrim Alliance</a> </div>
//	  <div class="rank"> <span class="value">…</span> </div>
//	</div>
//
// The SID comes from the href: /orgs/<SID>.
func parseOrgs(doc *html.Node) []OrgInfo {
	var orgs []OrgInfo

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && hasCSSClass(n, "org-row") {
			if org, ok := parseOrgRow(n); ok {
				orgs = append(orgs, org)
				return // don't descend into children of this row
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return orgs
}

func parseOrgRow(n *html.Node) (OrgInfo, bool) {
	org := OrgInfo{
		IsMain: hasCSSClass(n, "main-org") || hasCSSClass(n, "main"),
	}

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			switch {
			case hasCSSClass(node, "logo") || hasCSSClass(node, "thumb"):
				if src := findFirstImgSrc(node); src != "" {
					org.LogoURL = absoluteURL(src)
				}

			case hasCSSClass(node, "name") || hasCSSClass(node, "org-name"):
				// The name element typically contains an <a href="/orgs/SID">Name</a>.
				if a := findFirst(node, "a"); a != nil {
					org.Name = strings.TrimSpace(extractText(a))
					href := getAttr(a, "href")
					if sid := sidFromHref(href); sid != "" {
						org.SID = sid
					}
				} else {
					org.Name = strings.TrimSpace(extractText(node))
				}

			case hasCSSClass(node, "rank") || hasCSSClass(node, "member-rank"):
				org.RankName = strings.TrimSpace(extractValueText(node))
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)

	// An org row is only valid if we got at least a SID or a name.
	if org.SID == "" && org.Name == "" {
		return OrgInfo{}, false
	}
	// If we have a name but no SID, try extracting SID from any /orgs/ link inside.
	if org.SID == "" {
		if sid := findOrgSIDAnywhere(n); sid != "" {
			org.SID = sid
		}
	}
	// If we still have no SID, skip this row — the union of name+SID is required.
	if org.SID == "" {
		return OrgInfo{}, false
	}
	return org, true
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
