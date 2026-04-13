package watcher_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/malc/screenshot-sync/internal/config"
	"github.com/malc/screenshot-sync/internal/minecraft"
	"github.com/malc/screenshot-sync/internal/watcher"
)

func TestWatcherDetectsNewPNG(t *testing.T) {
	prismDir := t.TempDir()
	screenshotsDir := filepath.Join(prismDir, "instances", "Test", ".minecraft", "screenshots")
	os.MkdirAll(screenshotsDir, 0755)

	cfg := &config.Config{
		Sources: config.SourcesConfig{PrismDir: prismDir, Vanilla: false},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	received := make(chan string, 1)
	go func() {
		w := watcher.New(cfg)
		w.Watch(ctx, func(src minecraft.Source, path string) {
			received <- path
		})
	}()

	// Give the watcher time to start
	time.Sleep(100 * time.Millisecond)

	pngPath := filepath.Join(screenshotsDir, "2024-01-01_10.00.00.png")
	os.WriteFile(pngPath, []byte("PNG"), 0644)

	select {
	case got := <-received:
		if got != pngPath {
			t.Errorf("got %q, want %q", got, pngPath)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for watcher event")
	}
}
