package downloader

import "testing"

func TestIsGeoBlocked(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"ERROR: HTTP Error 403: Forbidden", true},
		{"This video is geo restricted", true},
		{"Video not available in your country", true},
		{"ERROR: Unable to extract video data", true},
		{"some other error: connection reset", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isGeoBlocked(tt.in)
		if got != tt.want {
			t.Errorf("isGeoBlocked(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
