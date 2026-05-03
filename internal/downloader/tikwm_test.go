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

func TestTikWMHappyPathPrefersHDPlay(t *testing.T) {
	const payload = "FAKE_HD_VIDEO_BYTES"

	video := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, payload)
	}))
	defer video.Close()

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		_ = r.ParseForm()
		if r.Form.Get("url") == "" {
			t.Errorf("missing url field")
		}
		if r.Form.Get("hd") != "1" {
			t.Errorf("expected hd=1, got %q", r.Form.Get("hd"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"msg":  "success",
			"data": map[string]any{
				"play":   video.URL + "/sd",
				"hdplay": video.URL,
				"wmplay": video.URL + "/wm",
			},
		})
	}))
	defer api.Close()

	dir := t.TempDir()
	tw := &TikWM{BaseURL: api.URL, Client: &http.Client{Timeout: 5 * time.Second}}
	res, err := tw.Download(context.Background(), "https://vt.tiktok.com/foo", dir)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if res.Strategy != "tikwm" {
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

func TestTikWMFallsBackToPlay(t *testing.T) {
	const payload = "FAKE_SD_VIDEO_BYTES"
	video := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, payload)
	}))
	defer video.Close()

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"play": video.URL,
			},
		})
	}))
	defer api.Close()

	tw := &TikWM{BaseURL: api.URL, Client: &http.Client{Timeout: 5 * time.Second}}
	res, err := tw.Download(context.Background(), "https://vt.tiktok.com/foo", t.TempDir())
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if res.Size == 0 {
		t.Fatal("expected non-empty file")
	}
}

func TestTikWMNonZeroCode(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": -1,
			"msg":  "video not found",
		})
	}))
	defer api.Close()

	tw := &TikWM{BaseURL: api.URL, Client: &http.Client{Timeout: time.Second}}
	_, err := tw.Download(context.Background(), "https://vt.tiktok.com/foo", t.TempDir())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "video not found") {
		t.Errorf("expected msg in error, got %v", err)
	}
}

func TestTikWMRateLimit(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer api.Close()

	tw := &TikWM{BaseURL: api.URL, Client: &http.Client{Timeout: time.Second}}
	_, err := tw.Download(context.Background(), "https://vt.tiktok.com/foo", t.TempDir())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "geo-blocked") {
		t.Errorf("expected geo-block-style fallback signal, got %v", err)
	}
}
