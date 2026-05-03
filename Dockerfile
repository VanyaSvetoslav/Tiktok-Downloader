## ---- builder ----------------------------------------------------------
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install git for go mod and ca-certificates for HTTPS module fetches.
RUN apk add --no-cache git ca-certificates

# Cache deps before copying the rest of the source.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build a fully static binary so the runtime stage can stay minimal.
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -trimpath -ldflags="-s -w" -o /app/bot ./

## ---- runtime ----------------------------------------------------------
FROM alpine:3.20

# yt-dlp from PyPI is dramatically newer than the Alpine package — TikTok
# extractors break often, so a stale yt-dlp is one of the most common
# causes of "Unable to extract webpage video data" errors.
RUN apk add --no-cache \
        ffmpeg \
        python3 \
        py3-pip \
        ca-certificates \
        tzdata \
    && update-ca-certificates \
    && pip3 install --no-cache-dir --break-system-packages --upgrade yt-dlp

# Non-root user for the bot.
RUN addgroup -S bot && adduser -S -G bot bot
WORKDIR /home/bot
USER bot

COPY --from=builder /app/bot /usr/local/bin/bot

ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/bot"]
