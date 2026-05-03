package downloader

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestPickTikhubURL(t *testing.T) {
	r := &tikhubResponse{}
	r.NoWatermarkVideoURL = "https://example.test/top.mp4"
	if got := pickTikhubURL(r); got != "https://example.test/top.mp4" {
		t.Fatalf("expected top-level url, got %q", got)
	}

	r = &tikhubResponse{}
	r.Data.NoWatermarkVideoURL = "https://example.test/data.mp4"
	if got := pickTikhubURL(r); got != "https://example.test/data.mp4" {
		t.Fatalf("expected data url, got %q", got)
	}

	r = &tikhubResponse{}
	r.Data.Aweme.Video.PlayAddr.URLList = []string{"https://example.test/play.mp4"}
	if got := pickTikhubURL(r); got != "https://example.test/play.mp4" {
		t.Fatalf("expected play_addr url, got %q", got)
	}

	r = &tikhubResponse{}
	r.Data.Aweme.Video.DownloadAddr.URLList = []string{"https://example.test/dl.mp4"}
	if got := pickTikhubURL(r); got != "https://example.test/dl.mp4" {
		t.Fatalf("expected download_addr url, got %q", got)
	}

	r = &tikhubResponse{}
	if got := pickTikhubURL(r); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestTikhubDownloadIntegration(t *testing.T) {
	const payload = "FAKE_VIDEO_BYTES_PAYLOAD"

	video := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, payload)
	}))
	defer video.Close()

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"no_watermark_video_url": video.URL,
		})
	}))
	defer api.Close()

	dir := t.TempDir()
	tk := &Tikhub{
		Endpoint: api.URL,
		Client:   &http.Client{Timeout: 5 * time.Second},
	}
	res, err := tk.Download(context.Background(), "https://www.tiktok.com/@x/video/123", dir)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if res.Strategy != "tikhub" {
		t.Errorf("strategy = %q", res.Strategy)
	}
	if !strings.HasPrefix(res.Path, dir) {
		t.Errorf("path %q not under workdir %q", res.Path, dir)
	}
	contents, err := os.ReadFile(res.Path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(contents) != payload {
		t.Errorf("expected %q, got %q", payload, string(contents))
	}
}

func TestTikhubGeoBlock(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer api.Close()

	tk := &Tikhub{Endpoint: api.URL, Client: &http.Client{Timeout: time.Second}}
	_, err := tk.Download(context.Background(), "https://www.tiktok.com/x", t.TempDir())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "geo-blocked") {
		t.Errorf("expected geo-blocked error, got %v", err)
	}
}
