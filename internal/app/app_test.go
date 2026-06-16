package app

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name  string
		input string
		def   time.Duration
		want  time.Duration
	}{
		{"empty string returns default", "", 3 * time.Minute, 3 * time.Minute},
		{"valid duration", "30s", 3 * time.Minute, 30 * time.Second},
		{"invalid duration returns default", "not-a-duration", 5 * time.Second, 5 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDuration(tt.input, tt.def)
			if got != tt.want {
				t.Errorf("parseDuration(%q, %v) = %v, want %v", tt.input, tt.def, got, tt.want)
			}
		})
	}
}
