package watcher

import (
	"context"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/malc/screenshot-sync/internal/config"
	"github.com/malc/screenshot-sync/internal/minecraft"
)

// Watcher watches Minecraft screenshot directories and calls onFile for each new PNG.
type Watcher struct {
	cfg *config.Config
}

// New creates a new Watcher.
func New(cfg *config.Config) *Watcher {
	return &Watcher{cfg: cfg}
}

// Watch starts watching screenshot directories until ctx is cancelled.
// onFile is called in a goroutine for each new .png file, after a 500ms delay.
func (w *Watcher) Watch(ctx context.Context, onFile func(src minecraft.Source, path string)) error {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer fw.Close()

	// watched maps screenshotsDir -> Source
	watched := make(map[string]minecraft.Source)

	addSources := func() {
		sources, err := minecraft.Discover(w.cfg)
		if err != nil {
			log.Printf("watcher: discover sources: %v", err)
		}
		for _, src := range sources {
			if _, ok := watched[src.ScreenshotsDir]; !ok {
				if err := fw.Add(src.ScreenshotsDir); err == nil {
					watched[src.ScreenshotsDir] = src
				}
			}
		}
	}

	addSources()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			addSources()
		case event, ok := <-fw.Events:
			if !ok {
				return nil
			}
			if event.Op&fsnotify.Create != 0 && strings.HasSuffix(event.Name, ".png") {
				src, ok := watched[filepath.Dir(event.Name)]
				if !ok {
					continue
				}
				go func(s minecraft.Source, p string) {
					select {
					case <-time.After(500 * time.Millisecond):
						onFile(s, p)
					case <-ctx.Done():
					}
				}(src, event.Name)
			}
		case err, ok := <-fw.Errors:
			if !ok {
				return nil
			}
			log.Printf("watcher error: %v", err)
		}
	}
}
