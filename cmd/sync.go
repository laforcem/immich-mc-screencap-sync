package cmd

import (
	"context"
	"log"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(syncCmd)
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Upload all unsynced Minecraft screenshots once and exit",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, client, st, err := setup()
		if err != nil {
			return err
		}
		log.Println("Running catch-up sync...")
		runCatchupSync(context.Background(), cfg, client, st)
		log.Println("Done.")
		return nil
	},
}
