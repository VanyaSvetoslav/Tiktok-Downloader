package downloader

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
)

// Compress shrinks src to fit under maxBytes by re-encoding it via ffmpeg.
// On success the original src file is replaced with the compressed copy
// and the function returns the new size. The original mp4 is removed
// only if the compression strictly decreased the file size; otherwise
// the original is kept.
func Compress(ctx context.Context, src string, maxBytes int64) (int64, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return 0, fmt.Errorf("ffmpeg not found in PATH: %w", err)
	}
	info, err := os.Stat(src)
	if err != nil {
		return 0, err
	}
	if info.Size() <= maxBytes {
		return info.Size(), nil
	}
	dst := src + ".compressed.mp4"
	args := []string{
		"-y",
		"-i", src,
		"-c:v", "libx264",
		"-crf", "18",
		"-preset", "fast",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		dst,
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", args...) //nolint:gosec
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	log.Printf("ffmpeg: compressing %s (current %d bytes)", src, info.Size())
	if err := cmd.Run(); err != nil {
		_ = os.Remove(dst)
		return 0, fmt.Errorf("ffmpeg: %v: %s", err, stderr.String())
	}
	newInfo, err := os.Stat(dst)
	if err != nil {
		return 0, err
	}
	if newInfo.Size() == 0 {
		_ = os.Remove(dst)
		return 0, errors.New("ffmpeg produced empty file")
	}
	if newInfo.Size() >= info.Size() {
		// No improvement; keep original.
		_ = os.Remove(dst)
		return info.Size(), nil
	}
	if err := os.Rename(dst, src); err != nil {
		_ = os.Remove(dst)
		return 0, fmt.Errorf("rename compressed: %w", err)
	}
	return newInfo.Size(), nil
}
