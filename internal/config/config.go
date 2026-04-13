package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config is the top-level configuration.
type Config struct {
	Immich  ImmichConfig  `toml:"immich"`
	Sources SourcesConfig `toml:"sources"`
	Path    string        `toml:"-"` // populated after loading
}

// ImmichConfig holds Immich connection settings.
type ImmichConfig struct {
	URL    string `toml:"url"`
	APIKey string `toml:"api_key"`
	Album  string `toml:"album"`
}

// SourcesConfig defines which Minecraft sources to watch.
type SourcesConfig struct {
	PrismDir string `toml:"prism_dir"`
	Vanilla  bool   `toml:"vanilla"`
}

// Load searches for config.toml next to the executable, then in
// %APPDATA%\screenshot-sync\config.toml.
func Load() (*Config, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("locate executable: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return nil, fmt.Errorf("resolve executable path: %w", err)
	}
	candidates := []string{
		filepath.Join(filepath.Dir(exe), "config.toml"),
	}
	if appdata := os.Getenv("APPDATA"); appdata != "" {
		candidates = append(candidates, filepath.Join(appdata, "screenshot-sync", "config.toml"))
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return LoadFrom(p)
		}
	}
	return nil, errors.New("config.toml not found next to binary or in %APPDATA%\\screenshot-sync\\")
}

// LoadFrom loads and validates a config from the given path.
func LoadFrom(path string) (*Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	if cfg.Immich.URL == "" {
		return nil, errors.New("immich.url is required")
	}
	if cfg.Immich.APIKey == "" {
		return nil, errors.New("immich.api_key is required")
	}
	if cfg.Immich.Album == "" {
		cfg.Immich.Album = "Minecraft"
	}
	if cfg.Sources.PrismDir != "" {
		info, err := os.Stat(cfg.Sources.PrismDir)
		if err != nil {
			return nil, fmt.Errorf("can't find prism_dir %q — make sure the folder exists (see %s)", cfg.Sources.PrismDir, path)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("sources.prism_dir: %s is not a directory", cfg.Sources.PrismDir)
		}
	}
	cfg.Path = path
	return &cfg, nil
}

