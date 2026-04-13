package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/malc/screenshot-sync/internal/config"
)

func TestLoadFrom(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := `
[immich]
url     = "https://photos.example.com"
api_key = "testkey"
album   = "Minecraft"

[sources]
prism_dir = "D:/Games/Minecraft/Prism Launcher"
vanilla   = true
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cfg, err := config.LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.Immich.URL != "https://photos.example.com" {
		t.Errorf("URL = %q", cfg.Immich.URL)
	}
	if cfg.Immich.APIKey != "testkey" {
		t.Errorf("APIKey = %q", cfg.Immich.APIKey)
	}
	if cfg.Immich.Album != "Minecraft" {
		t.Errorf("Album = %q", cfg.Immich.Album)
	}
	if cfg.Sources.PrismDir != "D:/Games/Minecraft/Prism Launcher" {
		t.Errorf("PrismDir = %q", cfg.Sources.PrismDir)
	}
	if !cfg.Sources.Vanilla {
		t.Error("Vanilla should be true")
	}
	if cfg.Path != path {
		t.Errorf("Path = %q, want %q", cfg.Path, path)
	}
}

func TestLoadFromMissingRequired(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(`[immich]`), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	_, err := config.LoadFrom(path)
	if err == nil {
		t.Fatal("expected error for missing required fields")
	}
}

func TestDefaultAlbumName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("[immich]\nurl=\"https://x.com\"\napi_key=\"k\""), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	cfg, err := config.LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.Immich.Album != "Minecraft" {
		t.Errorf("default album = %q, want Minecraft", cfg.Immich.Album)
	}
}
