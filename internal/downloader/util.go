package downloader

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/net/proxy"
)

// MobileUserAgents is a small pool of recent mobile User-Agent strings
// used to spoof the TikTok mobile app on out-of-process requests.
var MobileUserAgents = []string{
	"TikTok 26.2.0 rv:262018 (iPhone; iOS 14.4.2; en_US) Cronet",
	"com.zhiliaoapp.musically/2022600040 (Linux; U; Android 12; en_US; SM-G973F; Build/SP1A.210812.016; Cronet/TTNetVersion:b4d74d15)",
	"com.ss.android.ugc.trill/2022600040 (Linux; U; Android 11; en_US; Pixel 5; Build/RQ3A.210805.001.A1)",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1",
}

// PickUserAgent returns a pseudo-random mobile User-Agent.
func PickUserAgent() string {
	var b [1]byte
	_, _ = rand.Read(b[:])
	return MobileUserAgents[int(b[0])%len(MobileUserAgents)]
}

// NewHTTPClient returns an *http.Client configured to honour PROXY_URL
// (HTTP(S) or SOCKS5) when set, with a sane default timeout.
func NewHTTPClient(proxyURL string, timeout time.Duration) (*http.Client, error) {
	transport := &http.Transport{
		ResponseHeaderTimeout: 30 * time.Second,
	}
	if proxyURL != "" {
		u, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("parse proxy url: %w", err)
		}
		switch u.Scheme {
		case "http", "https":
			transport.Proxy = http.ProxyURL(u)
		case "socks5", "socks5h":
			dialer, err := proxy.FromURL(u, proxy.Direct)
			if err != nil {
				return nil, fmt.Errorf("socks5 dialer: %w", err)
			}
			transport.Dial = dialer.Dial //nolint:staticcheck // DialContext not exposed by all socks dialers
		default:
			return nil, fmt.Errorf("unsupported proxy scheme %q", u.Scheme)
		}
	}
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}, nil
}

// RandomFileName returns a workdir-relative path with the given extension.
func RandomFileName(workDir, ext string) (string, error) {
	if workDir == "" {
		return "", errors.New("workDir is empty")
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return "", err
	}
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return filepath.Join(workDir, "tiktok_"+hex.EncodeToString(b[:])+ext), nil
}
