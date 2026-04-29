//go:build windows

package cmd

import (
	"bytes"
	"testing"
	"time"
)

func TestWriteCountdown(t *testing.T) {
	var buf bytes.Buffer
	writeCountdown(&buf, func(time.Duration) {})
	got := buf.String()
	want := "Started in tray. Closing in 3... 2... 1...\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
