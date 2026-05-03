package config

import "testing"

func TestLoadRequiresToken(t *testing.T) {
	t.Setenv("BOT_TOKEN", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error when BOT_TOKEN is missing")
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("BOT_TOKEN", "abc")
	t.Setenv("PORT", "")
	t.Setenv("TIKTOK_COOKIE", "")
	t.Setenv("PROXY_URL", "")
	t.Setenv("TIKHUB_API_KEY", "")
	t.Setenv("WORK_DIR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.BotToken != "abc" {
		t.Errorf("BotToken = %q", cfg.BotToken)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d", cfg.Port)
	}
	if cfg.MaxFileSize != 50*1024*1024 {
		t.Errorf("MaxFileSize = %d", cfg.MaxFileSize)
	}
}

func TestLoadInvalidPort(t *testing.T) {
	t.Setenv("BOT_TOKEN", "abc")
	t.Setenv("PORT", "abc")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for non-int PORT")
	}
}
