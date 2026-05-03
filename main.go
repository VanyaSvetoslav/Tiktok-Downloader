// Tiktok-Downloader is a Telegram bot that delivers watermark-free
// TikTok videos. See README.md for runtime configuration.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/VanyaSvetoslav/Tiktok-Downloader/internal/bot"
	"github.com/VanyaSvetoslav/Tiktok-Downloader/internal/config"
	"github.com/VanyaSvetoslav/Tiktok-Downloader/internal/downloader"
	"github.com/VanyaSvetoslav/Tiktok-Downloader/internal/server"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	dl, err := downloader.New(
		cfg.WorkDir,
		cfg.MaxFileSize,
		cfg.TikTokCookie,
		cfg.ProxyURL,
		cfg.TikhubAPIKey,
		cfg.HTTPTimeout,
	)
	if err != nil {
		log.Fatalf("downloader: %v", err)
	}

	b, err := bot.New(cfg.BotToken, dl)
	if err != nil {
		log.Fatalf("bot: %v", err)
	}

	httpSrv := server.New(cfg.Port)

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt, syscall.SIGTERM,
	)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := server.Run(ctx, httpSrv); err != nil {
			log.Printf("http: %v", err)
			cancel()
		}
	}()

	go func() {
		defer wg.Done()
		if err := b.Run(ctx); err != nil {
			log.Printf("bot: %v", err)
			cancel()
		}
	}()

	<-ctx.Done()
	log.Printf("main: shutdown signal received, waiting for components")
	wg.Wait()
	log.Printf("main: bye")
}
