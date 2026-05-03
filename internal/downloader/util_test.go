package downloader

import (
	"strings"
	"testing"
	"time"
)

func TestPickUserAgentNonEmpty(t *testing.T) {
	for i := 0; i < 50; i++ {
		ua := PickUserAgent()
		if ua == "" {
			t.Fatalf("PickUserAgent returned empty string on iteration %d", i)
		}
	}
}

func TestNewHTTPClientNoProxy(t *testing.T) {
	c, err := NewHTTPClient("", 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("client is nil")
	}
	if c.Timeout != 5*time.Second {
		t.Fatalf("expected timeout 5s, got %s", c.Timeout)
	}
}

func TestNewHTTPClientWithHTTPProxy(t *testing.T) {
	c, err := NewHTTPClient("http://user:pass@127.0.0.1:8080", time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("client is nil")
	}
}

func TestNewHTTPClientWithBadProxy(t *testing.T) {
	_, err := NewHTTPClient("ftp://nope", time.Second)
	if err == nil {
		t.Fatal("expected error for unsupported scheme")
	}
	if !strings.Contains(err.Error(), "unsupported proxy scheme") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRandomFileName(t *testing.T) {
	dir := t.TempDir()
	name1, err := RandomFileName(dir, ".mp4")
	if err != nil {
		t.Fatalf("RandomFileName: %v", err)
	}
	name2, err := RandomFileName(dir, ".mp4")
	if err != nil {
		t.Fatalf("RandomFileName: %v", err)
	}
	if name1 == name2 {
		t.Fatal("RandomFileName collision")
	}
	if !strings.HasPrefix(name1, dir) {
		t.Fatalf("expected name within workDir, got %q", name1)
	}
	if !strings.HasSuffix(name1, ".mp4") {
		t.Fatalf("expected .mp4 suffix, got %q", name1)
	}
}
