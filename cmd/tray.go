package cmd

import (
	"context"
	"log"

	"github.com/getlantern/systray"
	"github.com/malc/screenshot-sync/internal/assets"
	"github.com/malc/screenshot-sync/internal/config"
	"github.com/malc/screenshot-sync/internal/immich"
	"github.com/malc/screenshot-sync/internal/minecraft"
	"github.com/malc/screenshot-sync/internal/state"
	"github.com/malc/screenshot-sync/internal/upload"
	"github.com/malc/screenshot-sync/internal/watcher"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(trayCmd)
}

var trayCmd = &cobra.Command{
	Use:   "tray",
	Short: "Watch for new screenshots and upload them (with system tray icon)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, client, st, err := setup()
		if err != nil {
			return err
		}
		systray.Run(
			onReady(cfg, client, st),
			func() {},
		)
		return nil
	},
}

func onReady(cfg *config.Config, client *immich.Client, st *state.State) func() {
	return func() {
		systray.SetIcon(assets.IconIdle())
		systray.SetTooltip("Minecraft Screenshot Sync")

		mSync := systray.AddMenuItem("Sync Now", "Upload all unsynced screenshots")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Stop screenshot-sync")

		ctx, cancel := context.WithCancel(context.Background())

		// Initial catch-up sync
		go func() {
			systray.SetIcon(assets.IconSyncing())
			runCatchupSync(ctx, cfg, client, st)
			systray.SetIcon(assets.IconIdle())
		}()

		// File watcher
		go func() {
			w := watcher.New(cfg)
			w.Watch(ctx, func(src minecraft.Source, path string) {
				systray.SetIcon(assets.IconSyncing())
				if err := upload.Screenshot(ctx, client, st, cfg.Immich.Album, src, path); err != nil {
					log.Printf("upload %s: %v", path, err)
					systray.SetIcon(assets.IconError())
					systray.SetTooltip("Error: " + err.Error())
					return
				}
				systray.SetIcon(assets.IconIdle())
				systray.SetTooltip("Minecraft Screenshot Sync")
			})
		}()

		for {
			select {
			case <-mSync.ClickedCh:
				go func() {
					systray.SetIcon(assets.IconSyncing())
					runCatchupSync(ctx, cfg, client, st)
					systray.SetIcon(assets.IconIdle())
				}()
			case <-mQuit.ClickedCh:
				cancel()
				systray.Quit()
				return
			}
		}
	}
}
