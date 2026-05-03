# Tiktok-Downloader

A production-ready Telegram bot — written in Go — that delivers the highest-quality, **watermark-free** version of any TikTok video sent to it. Designed to deploy to [Railway](https://railway.app) as a single static binary inside a small Alpine container.

## Features

- **Telegram interface**: send any TikTok URL, get the video back as a file. Concurrent requests are handled in their own goroutines.
- **Three-tier downloader fallback** (in priority order):
  1. **`yt-dlp`** — invoked as a subprocess with TikTok-app User-Agent spoofing, geo-bypass, retries and proxy support.
  2. **Tikhub API** — pulls the `no_watermark_video_url` from `https://api.tikhub.io/api/v1/tiktok/app/v3/fetch_one_video`.
  3. **SSSTik scraper** — POSTs to `https://ssstik.io/abc?lang=en` and parses the response HTML.
- **403 / geo-block bypass**: rotating mobile User-Agents, optional `TIKTOK_COOKIE`, optional HTTP/SOCKS5 proxy, `--geo-bypass-country US`.
- **Quality assurance**: prefers MP4/H.264, picks the best video+audio combo. Files larger than Telegram's 50 MB limit are re-encoded with `ffmpeg -crf 18 -preset fast` and replaced atomically.
- **Railway-ready**: HTTP health-check endpoint on `GET /`, graceful `SIGTERM` handling, `Dockerfile`, and `railway.toml`.
- **Stateless & disposable**: temp files are cleaned up after every download.

## Quick start (local)

```bash
# 1. Install yt-dlp + ffmpeg locally
sudo apt-get install -y yt-dlp ffmpeg     # Debian/Ubuntu
brew install yt-dlp ffmpeg                # macOS

# 2. Create a bot via @BotFather and copy the token
export BOT_TOKEN=123456:ABC...

# 3. Run
go run .
```

## Configuration

All configuration is read from environment variables at start-up. See [`.env.example`](.env.example) for a copy-pasteable template.

| Variable          | Required | Default       | Description |
|-------------------|----------|---------------|-------------|
| `BOT_TOKEN`       | yes      | —             | Telegram Bot API token from `@BotFather`. |
| `TIKTOK_COOKIE`   | no       | —             | A `Cookie:` header value forwarded to every TikTok request. Set this if you see a lot of 403s. |
| `PROXY_URL`       | no       | —             | HTTP, HTTPS, or SOCKS5 proxy URL (e.g. `socks5://user:pass@host:1080`). Forwarded to yt-dlp via `--proxy` and to the Go HTTP client. |
| `TIKHUB_API_KEY`  | no       | —             | Bearer key for the Tikhub fallback. Without it Tikhub may rate-limit aggressively. |
| `PORT`            | no       | `8080`        | Port for the Railway health check server. |
| `WORK_DIR`        | no       | `$TMPDIR`     | Directory used for temporary video downloads. |

## Deploy to Railway

1. Fork this repo (or push a clone of it) to GitHub.
2. Create a new Railway project from the repo. Railway will detect [`railway.toml`](railway.toml) and build the [`Dockerfile`](Dockerfile).
3. In *Variables*, set at minimum `BOT_TOKEN`. Optionally add `TIKTOK_COOKIE`, `PROXY_URL`, `TIKHUB_API_KEY`.
4. Hit **Deploy**. Railway will hit `GET /` for health checks; the bot will start polling Telegram.

## Architecture

```
┌──────────────┐      ┌─────────────────────────────────┐
│ Telegram API │◀────▶│ internal/bot       (long-poll)  │
└──────────────┘      └────────────────┬────────────────┘
                                       │
                                       ▼
                       ┌──────────────────────────────┐
                       │ internal/downloader.Manager  │
                       └────────────────┬─────────────┘
                                        │ tries each in order
       ┌────────────────────────────────┼────────────────────────────┐
       ▼                                ▼                            ▼
 ┌─────────────┐               ┌────────────────┐           ┌────────────────┐
 │  yt-dlp     │               │  Tikhub API    │           │  SSSTik scrape │
 │ (subprocess)│               │  (REST GET)    │           │  (HTML parse)  │
 └──────┬──────┘               └────────┬───────┘           └────────┬───────┘
        │ MP4 file                       │ MP4 URL → GET             │ MP4 URL → GET
        ▼                                ▼                           ▼
                       ┌──────────────────────────────┐
                       │ ffmpeg compress (>50 MB only)│
                       └────────────────┬─────────────┘
                                        ▼
                                Telegram upload
```

| Path | What lives there |
|------|------------------|
| `main.go`                      | Wires up signals, HTTP server, and bot. |
| `internal/config`              | Env-var loader. |
| `internal/server`              | Railway health-check HTTP server. |
| `internal/bot`                 | Telegram long-polling and message handling. |
| `internal/downloader/manager.go` | Chains the three strategies and delegates to ffmpeg compression. |
| `internal/downloader/ytdlp.go` | yt-dlp subprocess wrapper. |
| `internal/downloader/tikhub.go`| Tikhub REST client + direct HTTP downloader. |
| `internal/downloader/ssstik.go`| ssstik.io scraper. |
| `internal/downloader/ffmpeg.go`| Re-encodes oversized videos. |

## Telegram commands

| Command           | Behaviour |
|-------------------|-----------|
| `/start`, `/help` | Sends a short usage hint. |
| any TikTok URL    | Bot replies with `Downloading...`, then sends the watermark-free MP4. |
| anything else     | Bot asks for a TikTok link. |

## Local development

```bash
go vet ./...
go build ./...
go test ./...
```

## License

[MIT](LICENSE)
