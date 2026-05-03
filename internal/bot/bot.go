// Package bot wires up the Telegram bot interface around the downloader.
package bot

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/VanyaSvetoslav/Tiktok-Downloader/internal/downloader"
)

// Bot wraps a tgbotapi.BotAPI and a downloader.Manager.
type Bot struct {
	api     *tgbotapi.BotAPI
	dl      *downloader.Manager
	once    sync.Once
	stopped chan struct{}
}

// New creates a Telegram bot client.
func New(token string, dl *downloader.Manager) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("telegram: new bot: %w", err)
	}
	api.Debug = false
	log.Printf("bot: authorized as @%s", api.Self.UserName)
	return &Bot{
		api:     api,
		dl:      dl,
		stopped: make(chan struct{}),
	}, nil
}

// Run starts the long-polling loop. It returns when ctx is cancelled.
func (b *Bot) Run(ctx context.Context) error {
	cfg := tgbotapi.NewUpdate(0)
	cfg.Timeout = 30
	updates := b.api.GetUpdatesChan(cfg)

	defer b.once.Do(func() {
		b.api.StopReceivingUpdates()
		close(b.stopped)
	})

	var wg sync.WaitGroup
	for {
		select {
		case <-ctx.Done():
			log.Printf("bot: context cancelled, waiting for in-flight handlers")
			wg.Wait()
			return nil
		case update, ok := <-updates:
			if !ok {
				wg.Wait()
				return errors.New("telegram updates channel closed")
			}
			if update.Message == nil {
				continue
			}
			wg.Add(1)
			go func(u tgbotapi.Update) {
				defer wg.Done()
				b.handleMessage(ctx, u.Message)
			}(update)
		}
	}
}

var tiktokURLRE = regexp.MustCompile(`https?://[^\s]*tiktok[^\s]*`)

func extractURL(text string) string {
	if m := tiktokURLRE.FindString(text); m != "" {
		return strings.TrimRight(m, ".,)")
	}
	return ""
}

func (b *Bot) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return
	}
	if strings.HasPrefix(text, "/start") || strings.HasPrefix(text, "/help") {
		welcome := "Send me any TikTok link and I'll send back the highest-quality watermark-free video."
		_, _ = b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, welcome))
		return
	}

	link := extractURL(text)
	if link == "" {
		_, _ = b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "Please send a TikTok link (must contain tiktok.com)."))
		return
	}

	statusMsg, _ := b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "Downloading..."))

	dlCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	res, err := b.dl.Download(dlCtx, link)
	if err != nil {
		log.Printf("bot: download failed for %s: %v", link, err)
		b.editStatus(statusMsg, msg.Chat.ID, "Sorry — couldn't download that video. (All download strategies failed.)")
		return
	}
	defer func() {
		_ = os.Remove(res.Path)
	}()

	b.editStatus(statusMsg, msg.Chat.ID, fmt.Sprintf("Got it (%s). Uploading...", res.Strategy))

	video := tgbotapi.NewVideo(msg.Chat.ID, tgbotapi.FilePath(res.Path))
	video.Caption = "via " + res.Strategy
	video.SupportsStreaming = true
	video.ReplyToMessageID = msg.MessageID
	if _, err := b.api.Send(video); err != nil {
		log.Printf("bot: upload failed for %s: %v", link, err)
		b.editStatus(statusMsg, msg.Chat.ID, "Download succeeded but upload failed: "+err.Error())
		return
	}

	if statusMsg.MessageID != 0 {
		_, _ = b.api.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, statusMsg.MessageID))
	}
}

func (b *Bot) editStatus(prev tgbotapi.Message, chatID int64, text string) {
	if prev.MessageID == 0 {
		_, _ = b.api.Send(tgbotapi.NewMessage(chatID, text))
		return
	}
	edit := tgbotapi.NewEditMessageText(chatID, prev.MessageID, text)
	_, _ = b.api.Send(edit)
}
