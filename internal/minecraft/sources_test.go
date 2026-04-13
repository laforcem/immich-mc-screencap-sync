package minecraft_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/malc/screenshot-sync/internal/config"
	"github.com/malc/screenshot-sync/internal/minecraft"
)

func makePrismInstance(t *testing.T, prismDir, instanceName string) string {
	t.Helper()
	screenshotsDir := filepath.Join(prismDir, "instances", instanceName, ".minecraft", "screenshots")
	os.MkdirAll(screenshotsDir, 0755)
	return screenshotsDir
}

func TestDiscoverPrismInstances(t *testing.T) {
	prismDir := t.TempDir()
	makePrismInstance(t, prismDir, "Survival")
	makePrismInstance(t, prismDir, "Creative")

	cfg := &config.Config{
		Sources: config.SourcesConfig{PrismDir: prismDir, Vanilla: false},
	}
	sources, err := minecraft.Discover(cfg)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(sources) != 2 {
		t.Fatalf("want 2 sources, got %d", len(sources))
	}
	names := map[string]bool{}
	for _, s := range sources {
		names[s.InstanceName] = true
		if s.Type != minecraft.SourcePrism {
			t.Errorf("expected Prism type")
		}
	}
	if !names["Survival"] || !names["Creative"] {
		t.Errorf("missing instance names: %v", names)
	}
}

func TestDiscoverSkipsInstancesWithNoScreenshotsDir(t *testing.T) {
	prismDir := t.TempDir()
	// Instance without screenshots dir
	os.MkdirAll(filepath.Join(prismDir, "instances", "Empty", ".minecraft"), 0755)
	// Instance with screenshots dir
	makePrismInstance(t, prismDir, "HasScreenshots")

	cfg := &config.Config{
		Sources: config.SourcesConfig{PrismDir: prismDir, Vanilla: false},
	}
	sources, _ := minecraft.Discover(cfg)
	if len(sources) != 1 || sources[0].InstanceName != "HasScreenshots" {
		t.Errorf("expected only HasScreenshots, got %v", sources)
	}
}

func TestDiscoverVanilla(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)
	screenshotsDir := filepath.Join(dir, ".minecraft", "screenshots")
	os.MkdirAll(screenshotsDir, 0755)

	cfg := &config.Config{
		Sources: config.SourcesConfig{Vanilla: true},
	}
	sources, _ := minecraft.Discover(cfg)
	found := false
	for _, s := range sources {
		if s.Type == minecraft.SourceVanilla {
			found = true
			if s.InstanceName != "vanilla" {
				t.Errorf("vanilla instance name = %q, want vanilla", s.InstanceName)
			}
		}
	}
	if !found {
		t.Error("vanilla source not discovered")
	}
}
