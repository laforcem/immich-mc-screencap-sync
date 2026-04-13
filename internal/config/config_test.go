package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/malc/screenshot-sync/internal/config"
)

func TestLoadFrom(t *testing.T) {
	prismDir := t.TempDir()
	path := filepath.Join(t.TempDir(), "config.toml")
	content := "[immich]\n" +
		"url     = \"https://photos.example.com\"\n" +
		"api_key = \"testkey\"\n" +
		"album   = \"Minecraft\"\n\n" +
		"[sources]\n" +
		"prism_dir = " + fmt.Sprintf("%q", prismDir) + "\n" +
		"vanilla   = true\n"
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
	if cfg.Sources.PrismDir != prismDir {
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

func TestLoadFromRejectsInvalidPrismDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := `
[immich]
url     = "https://photos.example.com"
api_key = "testkey"

[sources]
prism_dir = "/no/such/directory"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	_, err := config.LoadFrom(path)
	if err == nil {
		t.Fatal("expected error for non-existent prism_dir, got nil")
	}
	want := "can't find prism_dir"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want it to contain %q", err.Error(), want)
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

