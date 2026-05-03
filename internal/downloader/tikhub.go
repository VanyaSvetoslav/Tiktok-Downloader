package downloader

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Tikhub is the secondary downloader strategy. It calls the public
// Tikhub API, parses the no-watermark URL out of the JSON response and
// downloads the file directly.
type Tikhub struct {
	APIKey   string
	Client   *http.Client
	Endpoint string // override for tests; defaults to the official URL
}

// Name implements Strategy.
func (t *Tikhub) Name() string { return "tikhub" }

type tikhubResponse struct {
	Data struct {
		Aweme struct {
			Video struct {
				PlayAddr struct {
					URLList []string `json:"url_list"`
				} `json:"play_addr"`
				DownloadAddr struct {
					URLList []string `json:"url_list"`
				} `json:"download_addr"`
			} `json:"video"`
		} `json:"aweme_detail"`
		NoWatermarkVideoURL string `json:"no_watermark_video_url"`
	} `json:"data"`
	NoWatermarkVideoURL string `json:"no_watermark_video_url"`
}

// Download fetches metadata from Tikhub then downloads the no-watermark
// stream into workDir. It returns ErrGeoBlocked on auth/quota failures
// so the manager can move on to the next strategy.
func (t *Tikhub) Download(ctx context.Context, link, workDir string) (*Result, error) {
	if t.Client == nil {
		t.Client = &http.Client{Timeout: 60 * time.Second}
	}
	endpoint := t.Endpoint
	if endpoint == "" {
		endpoint = "https://api.tikhub.io/api/v1/tiktok/app/v3/fetch_one_video"
	}

	q := url.Values{}
	q.Set("url", link)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", PickUserAgent())
	if t.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+t.APIKey)
	}

	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tikhub: request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("%w: tikhub status %d", ErrGeoBlocked, resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("tikhub: status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("tikhub: read body: %w", err)
	}

	var parsed tikhubResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("tikhub: parse json: %w", err)
	}
	videoURL := pickTikhubURL(&parsed)
	if videoURL == "" {
		return nil, errors.New("tikhub: no video url in response")
	}

	return downloadDirect(ctx, t.Client, videoURL, workDir, t.Name())
}

func pickTikhubURL(r *tikhubResponse) string {
	if r.NoWatermarkVideoURL != "" {
		return r.NoWatermarkVideoURL
	}
	if r.Data.NoWatermarkVideoURL != "" {
		return r.Data.NoWatermarkVideoURL
	}
	if len(r.Data.Aweme.Video.PlayAddr.URLList) > 0 {
		return r.Data.Aweme.Video.PlayAddr.URLList[0]
	}
	if len(r.Data.Aweme.Video.DownloadAddr.URLList) > 0 {
		return r.Data.Aweme.Video.DownloadAddr.URLList[0]
	}
	return ""
}

// downloadDirect performs an HTTP GET on rawURL and writes the response
// body into a fresh file inside workDir. The returned Result.Strategy is
// set to the supplied label.
func downloadDirect(ctx context.Context, client *http.Client, rawURL, workDir, label string) (*Result, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", PickUserAgent())
	req.Header.Set("Referer", "https://www.tiktok.com/")
	if strings.Contains(rawURL, "tiktok") {
		req.Header.Set("Accept", "video/mp4,*/*;q=0.8")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%w: status 403 from %s", ErrGeoBlocked, rawURL)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("download: status %d from %s", resp.StatusCode, rawURL)
	}

	out, err := RandomFileName(workDir, ".mp4")
	if err != nil {
		return nil, err
	}
	f, err := os.Create(out)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	written, err := io.Copy(f, resp.Body)
	if err != nil {
		_ = os.Remove(out)
		return nil, fmt.Errorf("download: copy: %w", err)
	}
	if written == 0 {
		_ = os.Remove(out)
		return nil, errors.New("download: empty body")
	}
	return &Result{Path: out, Size: written, Strategy: label}, nil
}
