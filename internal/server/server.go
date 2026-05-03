// Package server exposes a minimal HTTP server for Railway health checks.
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// New returns an *http.Server configured to listen on the given port and
// answer GET / with HTTP 200 for Railway's health check.
func New(port int) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "ok")
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "ok")
	})

	return &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
}

// Run starts the HTTP server and gracefully shuts it down when ctx is cancelled.
// It blocks until shutdown completes.
func Run(ctx context.Context, srv *http.Server) error {
	errCh := make(chan error, 1)
	go func() {
		log.Printf("http: listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("http shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		return err
	}
}
