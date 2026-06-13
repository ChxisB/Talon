// Package tools provides web search, web fetch, and other utility tools
// for the Spectre Proxy proxy.
package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"golang.org/x/net/html"
)

const (
	maxFetchChars       = 100000
	maxSearchResults    = 10
	maxResponseBytes    = 2 * 1024 * 1024 // 2MB
	requestTimeout      = 15 * time.Second
	maxRedirectHops     = 5
	redirectCapBytes    = 64 * 1024
)

// EgressPolicy controls web fetch security.
type EgressPolicy struct {
	AllowPrivateNetworks bool
	AllowedSchemes       []string
}

// DefaultEgressPolicy returns a safe default policy.
func DefaultEgressPolicy() EgressPolicy {
	return EgressPolicy{
		AllowPrivateNetworks: false,
		AllowedSchemes:       []string{"https", "http"},
	}
}

// WebSearch performs a web search using DuckDuckGo.
func WebSearch(ctx context.Context, query string) ([]SearchResult, error) {
	reqURL := fmt.Sprintf("https://lite.duckduckgo.com/lite/?q=%s", url.QueryEscape(query))

	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("web_search: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("web_search: read: %w", err)
	}

	return parseSearchResults(string(body)), nil
}

// SearchResult represents a single search result.
type SearchResult struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// WebFetch fetches a URL and extracts text content.
func WebFetch(ctx context.Context, rawURL string, egress EgressPolicy) (*FetchResult, error) {
	if err := validateURL(rawURL, egress); err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: requestTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirectHops {
				return fmt.Errorf("too many redirects")
			}
			return validateURL(req.URL.String(), egress)
		},
	}

	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("web_fetch: %w", err)
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("web_fetch: read: %w", err)
	}

	result := &FetchResult{
		URL:       resp.Request.URL.String(),
		MediaType: "text/plain",
	}

	if strings.Contains(contentType, "html") {
		parsed := parseHTML(string(body))
		result.Title = parsed.Title
		result.Data = truncate(parsed.Text, maxFetchChars)
	} else {
		result.Title = resp.Request.URL.String()
		result.Data = truncate(string(body), maxFetchChars)
	}

	return result, nil
}

// FetchResult represents a web fetch response.
type FetchResult struct {
	URL       string `json:"url"`
	Title     string `json:"title"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// ─── Validation ────────────────────────────────────────────────────

func validateURL(rawURL string, egress EgressPolicy) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme == "" {
		return fmt.Errorf("missing URL scheme")
	}

	allowed := false
	for _, s := range egress.AllowedSchemes {
		if u.Scheme == s {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("scheme %s not allowed", u.Scheme)
	}

	if !egress.AllowPrivateNetworks {
		host := u.Hostname()
		if isPrivateHost(host) {
			return fmt.Errorf("private network target not allowed: %s", host)
		}
	}

	return nil
}

func isPrivateHost(host string) bool {
	// Simple check for localhost and private IPs
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	if strings.HasPrefix(host, "10.") || strings.HasPrefix(host, "192.168.") {
		return true
	}
	if strings.HasPrefix(host, "172.") {
		parts := strings.Split(host, ".")
		if len(parts) >= 2 {
			second := parts[1]
			if len(second) == 2 || (len(second) == 3 && second >= "16" && second <= "31") {
				return true
			}
		}
	}
	return false
}

// ─── HTML Parsing ───────────────────────────────────────────────────

type parsedHTML struct {
	Title string
	Text  string
}

func parseHTML(htmlContent string) parsedHTML {
	var result parsedHTML
	var textParts []string
	inBody := false
	inScript := false
	inStyle := false

	z := html.NewTokenizer(strings.NewReader(htmlContent))
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}

		switch tt {
		case html.StartTagToken, html.EndTagToken:
			name, _ := z.TagName()
			tagName := string(name)
			switch tagName {
			case "title":
				if tt == html.StartTagToken {
					z.Next() // text inside title
					result.Title = string(z.Text())
				}
			case "body":
				inBody = tt == html.StartTagToken
			case "script":
				inScript = tt == html.StartTagToken
			case "style":
				inStyle = tt == html.StartTagToken
			case "br", "p", "div", "tr", "li", "h1", "h2", "h3", "h4", "h5", "h6":
				if inBody && !inScript && !inStyle {
					textParts = append(textParts, "\n")
				}
			}

		case html.TextToken:
			if inBody && !inScript && !inStyle {
				text := strings.TrimSpace(string(z.Text()))
				if text != "" {
					textParts = append(textParts, text)
				}
			}
		}
	}

	result.Text = strings.Join(textParts, " ")
	return result
}

// ─── Search Result Parsing ─────────────────────────────────────────

func parseSearchResults(htmlContent string) []SearchResult {
	var results []SearchResult
	var current SearchResult
	inResult := false

	z := html.NewTokenizer(strings.NewReader(htmlContent))
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}

		if tt == html.StartTagToken || tt == html.SelfClosingTagToken {
			name, _ := z.TagName()
			tagName := string(name)

			if tagName == "a" {
				var href string
				for {
					key, val, more := z.TagAttr()
					if string(key) == "href" {
						href = string(val)
					}
					if !more {
						break
					}
				}
				if strings.HasPrefix(href, "/") && !strings.HasPrefix(href, "/lite/") {
					// Likely a search result link
					inResult = true
					current.URL = "https://lite.duckduckgo.com" + href
				}
			}
		}

		if tt == html.TextToken && inResult {
			text := strings.TrimSpace(string(z.Text()))
			if text != "" && isMostlyPrintable(text) {
				current.Title = text
				results = append(results, current)
				current = SearchResult{}
				inResult = false
				if len(results) >= maxSearchResults {
					break
				}
			}
		}
	}

	return results
}

func isMostlyPrintable(s string) bool {
	printable := 0
	for _, r := range s {
		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			printable++
		}
	}
	return len(s) > 0 && float64(printable)/float64(len(s)) > 0.8
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
