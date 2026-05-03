package downloader

import (
	"strings"
	"testing"
)

func TestExtractSSSTikDownloadURLPrefersWithoutWatermark(t *testing.T) {
	body := `
		<a class="pure-button download_link with_watermark" href="https://tikcdn.io/wm/123">WM</a>
		<a class="pure-button download_link without_watermark" href="https://tikcdn.io/nowm/123">No WM</a>
		<a class="pure-button audio" href="https://tikcdn.io/audio/123.mp3">Audio</a>
	`
	got, err := extractSSSTikDownloadURL(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://tikcdn.io/nowm/123" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractSSSTikDownloadURLAcceptsTikcdnWithoutMP4(t *testing.T) {
	body := `<a class="pure-button" href="https://tikcdn.io/ssstik/7606251641339251989">Download</a>`
	got, err := extractSSSTikDownloadURL(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "tikcdn.io") {
		t.Fatalf("got %q", got)
	}
}

func TestExtractSSSTikDownloadURLFallsBackToMP4(t *testing.T) {
	body := `<a class="pure-button" href="https://example.com/foo.mp4">Download</a>`
	got, err := extractSSSTikDownloadURL(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://example.com/foo.mp4" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractSSSTikDownloadURLNoMatch(t *testing.T) {
	body := `<a href="https://ssstik.io/en/share">Share</a><a href="javascript:void(0)">x</a>`
	if _, err := extractSSSTikDownloadURL(body); err == nil {
		t.Fatal("expected error")
	}
}

func TestExtractSSSTikDownloadURLIgnoresInternalLinks(t *testing.T) {
	body := `
		<a href="/en/news">News</a>
		<a href="https://twitter.com/ssstik_io">Follow</a>
		<a class="without_watermark" href="https://tikcdn.io/ssstik/abc">Real one</a>
	`
	got, err := extractSSSTikDownloadURL(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://tikcdn.io/ssstik/abc" {
		t.Fatalf("got %q", got)
	}
}
