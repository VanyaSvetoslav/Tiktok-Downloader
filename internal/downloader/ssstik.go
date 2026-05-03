package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// SSSTik is the last-resort fallback. It scrapes ssstik.io's public form
// endpoint for a direct download link.
type SSSTik struct {
	Client *http.Client
	// BaseURL allows tests to override the upstream host.
	BaseURL string
}

// Name implements Strategy.
func (s *SSSTik) Name() string { return "ssstik" }

// ssstikTokenRE finds the dynamic anti-bot token rendered into the
// landing page as `tt = "..."` (also seen as `s_tt = "..."`).
var ssstikTokenRE = regexp.MustCompile(`(?:s_)?tt\s*[:=]\s*"([^"]+)"`)

// Download POSTs the user's URL to ssstik.io/abc and parses the
// resulting HTML fragment for a direct CDN link. It accepts links
// without the `.mp4` extension because ssstik often returns tikcdn.io
// URLs that point to a streaming endpoint.
func (s *SSSTik) Download(ctx context.Context, link, workDir string) (*Result, error) {
	if s.Client == nil {
		s.Client = &http.Client{Timeout: 60 * time.Second}
	}
	base := s.BaseURL
	if base == "" {
		base = "https://ssstik.io"
	}

	tt, err := s.fetchToken(ctx, base)
	if err != nil {
		return nil, fmt.Errorf("ssstik: token: %w", err)
	}

	form := url.Values{}
	form.Set("id", link)
	form.Set("locale", "en")
	if tt != "" {
		form.Set("tt", tt)
	}

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost,
		base+"/abc?url=dl", strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, err
	}
	browserUA := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
	req.Header.Set("User-Agent", browserUA)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Origin", base)
	req.Header.Set("Referer", base+"/en")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "target")
	req.Header.Set("HX-Current-URL", base+"/en")

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ssstik: post: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ssstik: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("ssstik: read: %w", err)
	}

	videoURL, err := extractSSSTikDownloadURL(string(body))
	if err != nil {
		return nil, err
	}
	return downloadDirect(ctx, s.Client, videoURL, workDir, s.Name())
}

// extractSSSTikDownloadURL parses the HTML fragment returned by
// /abc?url=dl and returns the first anchor that points to the
// no-watermark download. ssstik renders an `<a download_link without_watermark>`
// pointing to tikcdn.io (no .mp4 extension), so we cannot just regex
// for *.mp4.
func extractSSSTikDownloadURL(body string) (string, error) {
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("ssstik: parse html: %w", err)
	}

	type candidate struct {
		href     string
		priority int // higher = better
	}
	var found []candidate

	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			href, classes := "", ""
			for _, a := range n.Attr {
				switch a.Key {
				case "href":
					href = strings.TrimSpace(a.Val)
				case "class":
					classes = a.Val
				}
			}
			if href != "" && (strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://")) {
				prio := 0
				switch {
				case strings.Contains(classes, "without_watermark"):
					prio = 100
				case strings.Contains(href, "tikcdn"):
					prio = 80
				case strings.Contains(href, "tiktokcdn"):
					prio = 70
				case strings.Contains(href, ".mp4"):
					prio = 50
				default:
					prio = 0
				}
				if prio > 0 {
					found = append(found, candidate{href: href, priority: prio})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)

	if len(found) == 0 {
		return "", errors.New("ssstik: no download link found in response")
	}
	best := found[0]
	for _, c := range found[1:] {
		if c.priority > best.priority {
			best = c
		}
	}
	return best.href, nil
}

func (s *SSSTik) fetchToken(ctx context.Context, base string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/en", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	resp, err := s.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	m := ssstikTokenRE.FindStringSubmatch(string(body))
	if len(m) < 2 {
		return "", nil
	}
	return m[1], nil
}
