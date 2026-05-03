package downloader

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// Manager orchestrates the configured Strategy chain.
type Manager struct {
	Strategies  []Strategy
	WorkDir     string
	MaxFileSize int64
}

// New builds a Manager wired with the standard yt-dlp -> Tikhub -> SSSTik
// chain using the supplied configuration.
func New(workDir string, maxFileSize int64, cookie, proxyURL, tikhubKey string, httpTimeout time.Duration) (*Manager, error) {
	httpClient, err := NewHTTPClient(proxyURL, httpTimeout)
	if err != nil {
		return nil, err
	}
	strategies := []Strategy{
		&YTDLP{
			Cookie:           cookie,
			ProxyURL:         proxyURL,
			UseChromeCookies: false,
		},
		&Tikhub{
			APIKey: tikhubKey,
			Client: httpClient,
		},
		&SSSTik{
			Client: httpClient,
		},
	}
	return &Manager{
		Strategies:  strategies,
		WorkDir:     workDir,
		MaxFileSize: maxFileSize,
	}, nil
}

// Download tries each strategy in order until one succeeds or all fail.
// The returned Result is guaranteed to point at a file <= MaxFileSize
// (compressed via ffmpeg if necessary).
func (m *Manager) Download(ctx context.Context, link string) (*Result, error) {
	link = strings.TrimSpace(link)
	if link == "" {
		return nil, errors.New("empty url")
	}

	var lastErr error
	for _, s := range m.Strategies {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		log.Printf("downloader: trying %s for %s", s.Name(), link)
		res, err := s.Download(ctx, link, m.WorkDir)
		if err == nil {
			log.Printf("downloader: %s succeeded (%d bytes)", s.Name(), res.Size)
			if m.MaxFileSize > 0 && res.Size > m.MaxFileSize {
				newSize, cerr := Compress(ctx, res.Path, m.MaxFileSize)
				if cerr != nil {
					return nil, fmt.Errorf("compress: %w", cerr)
				}
				res.Size = newSize
			}
			return res, nil
		}
		log.Printf("downloader: %s failed: %v", s.Name(), err)
		lastErr = err
		// Always try next strategy, but log loudly when it's not a geo block.
		if !errors.Is(err, ErrGeoBlocked) && !errors.Is(err, ErrUnsupported) && !isHTTPClientError(err) {
			// keep the last error and continue regardless
		}
	}
	if lastErr == nil {
		lastErr = errors.New("no strategy succeeded")
	}
	return nil, fmt.Errorf("all strategies failed: %w", lastErr)
}

func isHTTPClientError(err error) bool {
	var herr *http.ProtocolError
	return errors.As(err, &herr)
}
