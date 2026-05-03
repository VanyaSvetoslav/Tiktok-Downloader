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
)

// SSSTik is the tertiary fallback. It scrapes ssstik.io's public form
// endpoint for a direct download link.
type SSSTik struct {
	Client *http.Client
	// BaseURL allows tests to override the upstream host.
	BaseURL string
}

// Name implements Strategy.
func (s *SSSTik) Name() string { return "ssstik" }

var (
	ssstikTokenRE = regexp.MustCompile(`tt\s*[:=]\s*"([^"]+)"`)
	ssstikHrefRE  = regexp.MustCompile(`href="(https?://[^"]+\.mp4[^"]*)"`)
)

// Download POSTs the user's URL to ssstik.io/abc and parses the
// resulting HTML fragment for a direct .mp4 link.
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
		base+"/abc?lang=en", strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Origin", base)
	req.Header.Set("Referer", base+"/en")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("HX-Request", "true")
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
	matches := ssstikHrefRE.FindAllStringSubmatch(string(body), -1)
	if len(matches) == 0 {
		return nil, errors.New("ssstik: no mp4 link found in response")
	}
	// The first link is generally the no-watermark variant.
	videoURL := matches[0][1]
	return downloadDirect(ctx, s.Client, videoURL, workDir, s.Name())
}

func (s *SSSTik) fetchToken(ctx context.Context, base string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/en", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
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
