package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TikWM is a free, no-auth-required fallback strategy backed by
// https://www.tikwm.com/api/. The endpoint accepts any TikTok URL
// (incl. vt.tiktok.com short links) and returns a JSON object with
// `data.play` (no-watermark) and `data.hdplay` (HD no-watermark).
//
// Many open-source TikTok downloader bots use this endpoint because it
// works without registration and handles TikTok URLs that yt-dlp's
// extractor sometimes fails on.
type TikWM struct {
	Client *http.Client
	// BaseURL is overridable for tests; defaults to the public endpoint.
	BaseURL string
}

// Name implements Strategy.
func (t *TikWM) Name() string { return "tikwm" }

type tikwmResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Play   string `json:"play"`   // No-watermark URL.
		HDPlay string `json:"hdplay"` // HD no-watermark URL (preferred).
		WMPlay string `json:"wmplay"` // Watermarked fallback.
	} `json:"data"`
}

// Download calls the TikWM API, picks the highest-quality watermark-free
// URL, and downloads the file into workDir.
func (t *TikWM) Download(ctx context.Context, link, workDir string) (*Result, error) {
	if t.Client == nil {
		t.Client = &http.Client{Timeout: 60 * time.Second}
	}
	base := t.BaseURL
	if base == "" {
		base = "https://www.tikwm.com/api/"
	}

	form := url.Values{}
	form.Set("url", link)
	form.Set("hd", "1") // Ask for the HD variant if available.

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://www.tikwm.com")
	req.Header.Set("Referer", "https://www.tikwm.com/")

	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tikwm: request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("%w: tikwm status %d", ErrGeoBlocked, resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("tikwm: status %d: %s", resp.StatusCode, string(body))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("tikwm: read body: %w", err)
	}

	var parsed tikwmResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("tikwm: parse json: %w", err)
	}
	if parsed.Code != 0 {
		return nil, fmt.Errorf("tikwm: api code %d: %s", parsed.Code, parsed.Msg)
	}

	videoURL := parsed.Data.HDPlay
	if videoURL == "" {
		videoURL = parsed.Data.Play
	}
	if videoURL == "" {
		videoURL = parsed.Data.WMPlay
	}
	if videoURL == "" {
		return nil, fmt.Errorf("tikwm: no video url in response: %s", string(body))
	}

	// TikWM CDN URLs often start with `//`; normalise to https.
	if strings.HasPrefix(videoURL, "//") {
		videoURL = "https:" + videoURL
	} else if strings.HasPrefix(videoURL, "/") {
		videoURL = "https://www.tikwm.com" + videoURL
	}

	return downloadDirect(ctx, t.Client, videoURL, workDir, t.Name())
}
