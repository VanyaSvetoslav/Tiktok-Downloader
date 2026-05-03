package bot

import "testing"

func TestExtractURL(t *testing.T) {
	tests := []struct {
		text string
		want string
	}{
		{"https://www.tiktok.com/@user/video/123", "https://www.tiktok.com/@user/video/123"},
		{"check this out: https://vm.tiktok.com/abcd/", "https://vm.tiktok.com/abcd/"},
		{"https://vm.tiktok.com/abcd/.", "https://vm.tiktok.com/abcd/"},
		{"no urls here", ""},
		{"https://youtube.com/watch?v=1", ""},
	}
	for _, tt := range tests {
		got := extractURL(tt.text)
		if got != tt.want {
			t.Errorf("extractURL(%q) = %q, want %q", tt.text, got, tt.want)
		}
	}
}
