package cmd

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/malc/screenshot-sync/internal/config"
	"github.com/malc/screenshot-sync/internal/immich"
	"github.com/malc/screenshot-sync/internal/minecraft"
	"github.com/malc/screenshot-sync/internal/state"
	"github.com/malc/screenshot-sync/internal/upload"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "screenshot-sync",
	Short: "Sync Minecraft screenshots to Immich",
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func setup() (*config.Config, *immich.Client, *state.State, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, nil, err
	}
	client := immich.NewClient(cfg.Immich.URL, cfg.Immich.APIKey)
	statePath := filepath.Join(filepath.Dir(cfg.Path), "state.json")
	st, err := state.Load(statePath)
	if err != nil {
		return nil, nil, nil, err
	}
	return cfg, client, st, nil
}

// runCatchupSync uploads all unsynced screenshots in all discovered sources.
func runCatchupSync(ctx context.Context, cfg *config.Config, u upload.Uploader, st *state.State) {
	sources, err := minecraft.Discover(cfg)
	if err != nil {
		log.Printf("discover sources: %v", err)
		return
	}
	for _, src := range sources {
		entries, err := os.ReadDir(src.ScreenshotsDir)
		if err != nil {
			log.Printf("read dir %s: %v", src.ScreenshotsDir, err)
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".png") {
				continue
			}
			filePath := filepath.Join(src.ScreenshotsDir, entry.Name())
			if err := upload.Screenshot(ctx, u, st, cfg.Immich.Album, src, filePath); err != nil {
				log.Printf("upload %s: %v", filePath, err)
			}
		}
	}
}
