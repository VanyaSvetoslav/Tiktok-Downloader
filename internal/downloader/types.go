// Package downloader implements multiple strategies for fetching TikTok
// videos without watermark.
package downloader

import (
	"context"
	"errors"
)

// Result describes a downloaded video on disk.
type Result struct {
	// Path is the absolute path to the downloaded video file.
	Path string
	// Size is the file size in bytes.
	Size int64
	// Strategy is the human-readable name of the strategy that produced
	// the file (e.g. "yt-dlp", "tikhub", "ssstik").
	Strategy string
}

// Strategy is implemented by every downloader backend. Each strategy
// receives a TikTok URL and writes the resulting video to a file inside
// the working directory.
type Strategy interface {
	// Name returns the strategy's identifier (used for logging).
	Name() string
	// Download attempts to fetch the video. workDir is a directory the
	// strategy may use for temp / output files. The returned Result
	// MUST point to a file inside workDir.
	Download(ctx context.Context, url, workDir string) (*Result, error)
}

// ErrUnsupported is returned by a strategy when it explicitly cannot
// handle the URL (e.g. invalid format) and the manager should move on
// to the next strategy.
var ErrUnsupported = errors.New("downloader: url not supported by this strategy")

// ErrGeoBlocked is returned by a strategy when the upstream service
// indicates a 403 / region-block. The manager treats this as a signal
// to fall back to the next strategy.
var ErrGeoBlocked = errors.New("downloader: geo-blocked or 403")
