package minecraft

import (
	"log"
	"os"
	"path/filepath"

	"github.com/malc/screenshot-sync/internal/config"
)

// SourceType identifies the launcher that produced a screenshot directory.
type SourceType string

const (
	SourcePrism   SourceType = "prism"
	SourceVanilla SourceType = "vanilla"
)

// Source represents a discovered Minecraft screenshot directory.
type Source struct {
	Type           SourceType
	InstanceName   string
	ScreenshotsDir string
	AccountName    string // "unknown" if detection fails
}

// Discover returns all Minecraft screenshot sources based on cfg.
func Discover(cfg *config.Config) ([]Source, error) {
	var sources []Source

	for _, prismDir := range prismRoots(cfg.Sources.PrismDir) {
		instancesDir := filepath.Join(prismDir, "instances")
		entries, err := os.ReadDir(instancesDir)
		if err != nil {
			continue
		}
		account, err := ReadPrismAccount(prismDir)
		if err != nil {
			log.Printf("account detection failed for %s: %v — using 'unknown'", prismDir, err)
			account = "unknown"
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			screenshotsDir := filepath.Join(instancesDir, entry.Name(), ".minecraft", "screenshots")
			if _, err := os.Stat(screenshotsDir); err != nil {
				continue
			}
			sources = append(sources, Source{
				Type:           SourcePrism,
				InstanceName:   entry.Name(),
				ScreenshotsDir: screenshotsDir,
				AccountName:    account,
			})
		}
		break // use first valid Prism root (configured dir takes priority)
	}

	if cfg.Sources.Vanilla {
		appdata := os.Getenv("APPDATA")
		if appdata != "" {
			screenshotsDir := filepath.Join(appdata, ".minecraft", "screenshots")
			if _, err := os.Stat(screenshotsDir); err == nil {
				account, err := ReadVanillaAccount()
				if err != nil {
					log.Printf("vanilla account detection failed: %v — using 'unknown'", err)
					account = "unknown"
				}
				sources = append(sources, Source{
					Type:           SourceVanilla,
					InstanceName:   "vanilla",
					ScreenshotsDir: screenshotsDir,
					AccountName:    account,
				})
			}
		}
	}

	return sources, nil
}

// prismRoots returns candidate Prism Launcher root directories to try, in priority order.
func prismRoots(configuredDir string) []string {
	var roots []string
	if configuredDir != "" {
		roots = append(roots, configuredDir)
	}
	if appdata := os.Getenv("APPDATA"); appdata != "" {
		roots = append(roots, filepath.Join(appdata, "PrismLauncher"))
	}
	return roots
}
