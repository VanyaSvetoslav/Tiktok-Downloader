package downloader

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// YTDLP is the primary downloader strategy backed by the yt-dlp binary.
type YTDLP struct {
	// Binary is the absolute or PATH-resolvable path to the yt-dlp binary.
	Binary string
	// Cookie is an optional TIKTOK_COOKIE value passed via --add-header.
	Cookie string
	// ProxyURL, when non-empty, is forwarded to yt-dlp via --proxy.
	ProxyURL string
	// UseChromeCookies, when true, asks yt-dlp to read cookies from a
	// local Chrome profile (only useful in environments that have one).
	UseChromeCookies bool
}

// Name implements Strategy.
func (y *YTDLP) Name() string { return "yt-dlp" }

// Download invokes yt-dlp with the flags requested in the spec and
// writes the output to a unique file inside workDir.
func (y *YTDLP) Download(ctx context.Context, link, workDir string) (*Result, error) {
	if y.Binary == "" {
		y.Binary = "yt-dlp"
	}
	if _, err := exec.LookPath(y.Binary); err != nil {
		return nil, fmt.Errorf("%w: yt-dlp binary not found: %v", ErrUnsupported, err)
	}

	out, err := RandomFileName(workDir, ".mp4")
	if err != nil {
		return nil, err
	}

	args := []string{
		"--format", "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best",
		"--no-playlist",
		"--no-warnings",
		"--geo-bypass",
		"--geo-bypass-country", "US",
		"--add-header", "User-Agent: " + PickUserAgent(),
		"--add-header", "Referer: https://www.tiktok.com/",
		"--retries", "5",
		"--fragment-retries", "5",
		"--socket-timeout", "30",
		"--merge-output-format", "mp4",
		"-o", out,
	}
	if y.Cookie != "" {
		args = append(args, "--add-header", "Cookie: "+y.Cookie)
	}
	if y.UseChromeCookies {
		args = append(args, "--cookies-from-browser", "chrome")
	}
	if y.ProxyURL != "" {
		args = append(args, "--proxy", y.ProxyURL)
	}
	args = append(args, link)

	cmd := exec.CommandContext(ctx, y.Binary, args...) //nolint:gosec // args are constructed locally
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stderr

	log.Printf("yt-dlp: starting download for %s", link)
	if err := cmd.Run(); err != nil {
		_ = os.Remove(out)
		combined := stderr.String()
		if isGeoBlocked(combined) {
			return nil, fmt.Errorf("%w: %s", ErrGeoBlocked, combined)
		}
		return nil, fmt.Errorf("yt-dlp failed: %v: %s", err, combined)
	}

	info, err := os.Stat(out)
	if err != nil {
		return nil, fmt.Errorf("yt-dlp finished but output missing: %w", err)
	}
	if info.Size() == 0 {
		_ = os.Remove(out)
		return nil, errors.New("yt-dlp produced empty file")
	}
	return &Result{Path: out, Size: info.Size(), Strategy: y.Name()}, nil
}

func isGeoBlocked(stderr string) bool {
	s := strings.ToLower(stderr)
	switch {
	case strings.Contains(s, "http error 403"):
		return true
	case strings.Contains(s, "geo restrict"):
		return true
	case strings.Contains(s, "not available in your country"):
		return true
	case strings.Contains(s, "this video is not available"):
		return true
	case strings.Contains(s, "unable to extract"):
		return true
	}
	return false
}
