package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/malc/screenshot-sync/internal/minecraft"
	"github.com/malc/screenshot-sync/internal/upload"
	"github.com/malc/screenshot-sync/internal/watcher"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(daemonCmd)
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Watch for new screenshots and upload them (headless, no tray icon)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, client, st, err := setup()
		if err != nil {
			return err
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		log.Println("Running catch-up sync...")
		runCatchupSync(ctx, cfg, client, st)
		log.Println("Watching for new screenshots...")

		w := watcher.New(cfg)
		return w.Watch(ctx, func(src minecraft.Source, path string) {
			if err := upload.Screenshot(ctx, client, st, cfg.Immich.Album, src, path); err != nil {
				log.Printf("upload %s: %v", path, err)
			}
		})
	},
}
