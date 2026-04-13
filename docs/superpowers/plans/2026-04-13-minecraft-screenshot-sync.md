# Minecraft Screenshot Sync — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go binary that watches Minecraft screenshot directories on Windows and uploads new screenshots to Immich with instance/account metadata tags.

**Architecture:** Single binary with `tray`, `sync`, and `daemon` subcommands. Shared internal packages handle config loading, Minecraft source discovery, Immich API calls, file watching, and upload state tracking. Cross-compiled from Linux using mingw-w64 for the CGO tray requirement.

**Tech Stack:** Go 1.22+, `github.com/spf13/cobra` (CLI), `github.com/BurntSushi/toml` (config), `github.com/fsnotify/fsnotify` (file watching), `github.com/getlantern/systray` (tray icon), standard `net/http` (Immich API)

---

## File Map

| File | Purpose |
|------|---------|
| `main.go` | Entry point, calls `cmd.Execute()` |
| `cmd/root.go` | cobra root + `Execute()` + shared `setup()` helper |
| `cmd/sync.go` | `sync` subcommand: one-shot catch-up upload |
| `cmd/daemon.go` | `daemon` subcommand: headless watcher |
| `cmd/tray.go` | `tray` subcommand: watcher + system tray icon |
| `internal/config/config.go` | Config struct, TOML loading, path resolution |
| `internal/minecraft/accounts.go` | Parse active account from Prism/vanilla launcher files |
| `internal/minecraft/sources.go` | Discover screenshot directories, associate accounts |
| `internal/immich/client.go` | HTTP client for Immich REST API with tag/album caching |
| `internal/upload/upload.go` | Uploader interface + Screenshot orchestration function |
| `internal/state/state.go` | JSON state file, atomic writes, deduplication |
| `internal/watcher/watcher.go` | fsnotify wrapper with 60s instance polling |
| `internal/assets/icon.go` | Programmatic tray icon generation (no external files) |
| `Makefile` | Build targets including Windows cross-compilation |
| `config.toml.example` | Documented example config |
| `.gitignore` | Ignore build artifacts |

---

### Task 1: Initialize module and project skeleton

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `cmd/root.go`

- [ ] **Step 1: Initialize module and install dependencies**

```bash
cd /home/malc/repos/screenshot-sync
go mod init github.com/malc/screenshot-sync
go get github.com/spf13/cobra@latest
go get github.com/BurntSushi/toml@latest
go get github.com/fsnotify/fsnotify@latest
go get github.com/getlantern/systray@latest
```

- [ ] **Step 2: Create directory structure**

```bash
mkdir -p cmd internal/config internal/minecraft internal/immich internal/upload internal/state internal/watcher internal/assets
```

- [ ] **Step 3: Write `main.go`**

```go
package main

import "github.com/malc/screenshot-sync/cmd"

func main() {
	cmd.Execute()
}
```

- [ ] **Step 4: Write `cmd/root.go`**

```go
package cmd

import (
	"os"

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
```

- [ ] **Step 5: Verify build**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum main.go cmd/root.go
git commit -m "feat: initialize module and project skeleton"
```

---

### Task 2: Config package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/config/config_test.go
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
	os.WriteFile(path, []byte(content), 0644)

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
	os.WriteFile(path, []byte(`[immich]`), 0644)
	_, err := config.LoadFrom(path)
	if err == nil {
		t.Fatal("expected error for missing required fields")
	}
}

func TestDefaultAlbumName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	os.WriteFile(path, []byte("[immich]\nurl=\"https://x.com\"\napi_key=\"k\""), 0644)
	cfg, err := config.LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.Immich.Album != "Minecraft" {
		t.Errorf("default album = %q, want Minecraft", cfg.Immich.Album)
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/config/...
```

Expected: compile error — package does not exist yet.

- [ ] **Step 3: Write `internal/config/config.go`**

```go
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
	cfg.Path = path
	return &cfg, nil
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./internal/config/... -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add config package"
```

---

### Task 3: State package

**Files:**
- Create: `internal/state/state.go`
- Create: `internal/state/state_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/state/state_test.go
package state_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/malc/screenshot-sync/internal/state"
)

func TestMarkAndCheck(t *testing.T) {
	st, _ := state.Load(filepath.Join(t.TempDir(), "state.json"))
	const id = "prism/Survival/2024-01-15_12.30.00.png"
	if st.IsUploaded(id) {
		t.Fatal("should not be uploaded before marking")
	}
	if err := st.MarkUploaded(id, "asset-abc"); err != nil {
		t.Fatalf("MarkUploaded: %v", err)
	}
	if !st.IsUploaded(id) {
		t.Fatal("should be uploaded after marking")
	}
}

func TestPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	st, _ := state.Load(path)
	st.MarkUploaded("prism/World/a.png", "id1")

	st2, err := state.Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !st2.IsUploaded("prism/World/a.png") {
		t.Fatal("state not persisted")
	}
}

func TestMissingFileStartsEmpty(t *testing.T) {
	st, err := state.Load(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if st.IsUploaded("x") {
		t.Fatal("fresh state should be empty")
	}
}

func TestCorruptFileStartsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	os.WriteFile(path, []byte("not json {{{"), 0644)
	st, err := state.Load(path)
	if err != nil {
		t.Fatalf("corrupt state should not error: %v", err)
	}
	if st.IsUploaded("x") {
		t.Fatal("corrupt state should start empty")
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/state/...
```

Expected: compile error.

- [ ] **Step 3: Write `internal/state/state.go`**

```go
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// State tracks which screenshots have been uploaded to Immich.
type State struct {
	mu       sync.Mutex
	uploaded map[string]string // deviceAssetID -> immich asset ID
	path     string
}

// Load reads state from path. Returns an empty state if the file is missing or corrupt.
func Load(path string) (*State, error) {
	s := &State{
		uploaded: make(map[string]string),
		path:     path,
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return s, nil
	}
	_ = json.Unmarshal(data, &s.uploaded) // silently ignore corrupt JSON
	return s, nil
}

// IsUploaded reports whether deviceAssetID has already been uploaded.
func (s *State) IsUploaded(deviceAssetID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.uploaded[deviceAssetID]
	return ok
}

// MarkUploaded records a successful upload and saves state atomically to disk.
func (s *State) MarkUploaded(deviceAssetID, assetID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.uploaded[deviceAssetID] = assetID
	return s.save()
}

func (s *State) save() error {
	data, err := json.MarshalIndent(s.uploaded, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write temp state: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("rename state: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./internal/state/... -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/state/
git commit -m "feat: add state package with atomic JSON persistence"
```

---

### Task 4: Minecraft account detection

**Files:**
- Create: `internal/minecraft/accounts.go`
- Create: `internal/minecraft/accounts_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/minecraft/accounts_test.go
package minecraft_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/malc/screenshot-sync/internal/minecraft"
)

func writePrismAccounts(t *testing.T, dir string, accounts []map[string]any, activeLocalId string) {
	t.Helper()
	data, _ := json.Marshal(map[string]any{
		"accounts":             accounts,
		"activeAccountLocalId": activeLocalId,
	})
	os.WriteFile(filepath.Join(dir, "accounts.json"), data, 0644)
}

func TestReadPrismAccount(t *testing.T) {
	dir := t.TempDir()
	writePrismAccounts(t, dir, []map[string]any{
		{
			"localId": "local-1",
			"profile": map[string]any{"name": "Steve"},
		},
	}, "local-1")

	name, err := minecraft.ReadPrismAccount(dir)
	if err != nil {
		t.Fatalf("ReadPrismAccount: %v", err)
	}
	if name != "Steve" {
		t.Errorf("name = %q, want Steve", name)
	}
}

func TestReadPrismAccountFallsBackToFirst(t *testing.T) {
	dir := t.TempDir()
	writePrismAccounts(t, dir, []map[string]any{
		{
			"localId": "local-1",
			"profile": map[string]any{"name": "Alex"},
		},
	}, "no-match")

	name, err := minecraft.ReadPrismAccount(dir)
	if err != nil {
		t.Fatalf("ReadPrismAccount: %v", err)
	}
	if name != "Alex" {
		t.Errorf("name = %q, want Alex", name)
	}
}

func TestReadPrismAccountMissingFile(t *testing.T) {
	_, err := minecraft.ReadPrismAccount(t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing accounts.json")
	}
}

func TestReadVanillaAccountNewFormat(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)
	mcDir := filepath.Join(dir, ".minecraft")
	os.MkdirAll(mcDir, 0755)

	data, _ := json.Marshal(map[string]any{
		"accounts": map[string]any{
			"acc-1": map[string]any{
				"minecraftProfile": map[string]any{"name": "Herobrine"},
			},
		},
		"activeAccountLocalId": "acc-1",
	})
	os.WriteFile(filepath.Join(mcDir, "launcher_accounts.json"), data, 0644)

	name, err := minecraft.ReadVanillaAccount()
	if err != nil {
		t.Fatalf("ReadVanillaAccount: %v", err)
	}
	if name != "Herobrine" {
		t.Errorf("name = %q, want Herobrine", name)
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/minecraft/...
```

Expected: compile error.

- [ ] **Step 3: Write `internal/minecraft/accounts.go`**

```go
package minecraft

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ReadPrismAccount reads the active Minecraft username from Prism Launcher's accounts.json.
func ReadPrismAccount(prismDir string) (string, error) {
	path := filepath.Join(prismDir, "accounts.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	var f prismAccountsFile
	if err := json.Unmarshal(data, &f); err != nil {
		return "", fmt.Errorf("parse %s: %w", path, err)
	}
	name := f.activeName()
	if name == "" {
		return "", fmt.Errorf("no account name found in %s", path)
	}
	return name, nil
}

// ReadVanillaAccount reads the active Minecraft username from the vanilla launcher.
// Tries launcher_accounts.json (newer format) then launcher_profiles.json (older format).
func ReadVanillaAccount() (string, error) {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		return "", fmt.Errorf("APPDATA not set")
	}
	mcDir := filepath.Join(appdata, ".minecraft")

	if name, err := readLauncherAccounts(filepath.Join(mcDir, "launcher_accounts.json")); err == nil {
		return name, nil
	}
	return readLauncherProfiles(filepath.Join(mcDir, "launcher_profiles.json"))
}

// prism format
type prismAccountsFile struct {
	Accounts             []prismAccount `json:"accounts"`
	ActiveAccountLocalID string         `json:"activeAccountLocalId"`
}

type prismAccount struct {
	LocalID string       `json:"localId"`
	Profile prismProfile `json:"profile"`
}

type prismProfile struct {
	Name string `json:"name"`
}

func (f *prismAccountsFile) activeName() string {
	for _, acc := range f.Accounts {
		if acc.LocalID == f.ActiveAccountLocalID && acc.Profile.Name != "" {
			return acc.Profile.Name
		}
	}
	for _, acc := range f.Accounts {
		if acc.Profile.Name != "" {
			return acc.Profile.Name
		}
	}
	return ""
}

// newer vanilla format (launcher_accounts.json, Microsoft auth)
type launcherAccountsFile struct {
	Accounts             map[string]launcherAccount `json:"accounts"`
	ActiveAccountLocalID string                     `json:"activeAccountLocalId"`
}

type launcherAccount struct {
	MinecraftProfile struct {
		Name string `json:"name"`
	} `json:"minecraftProfile"`
}

func readLauncherAccounts(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var f launcherAccountsFile
	if err := json.Unmarshal(data, &f); err != nil {
		return "", err
	}
	if acc, ok := f.Accounts[f.ActiveAccountLocalID]; ok && acc.MinecraftProfile.Name != "" {
		return acc.MinecraftProfile.Name, nil
	}
	for _, acc := range f.Accounts {
		if acc.MinecraftProfile.Name != "" {
			return acc.MinecraftProfile.Name, nil
		}
	}
	return "", fmt.Errorf("no account in %s", path)
}

// older vanilla format (launcher_profiles.json)
type launcherProfilesFile struct {
	AuthenticationDatabase map[string]struct {
		Profiles map[string]struct {
			DisplayName string `json:"displayName"`
		} `json:"profiles"`
	} `json:"authenticationDatabase"`
	SelectedUser struct {
		Account string `json:"account"`
		Profile string `json:"profile"`
	} `json:"selectedUser"`
}

func readLauncherProfiles(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var f launcherProfilesFile
	if err := json.Unmarshal(data, &f); err != nil {
		return "", err
	}
	acc, ok := f.AuthenticationDatabase[f.SelectedUser.Account]
	if !ok {
		return "", fmt.Errorf("selected account not found in %s", path)
	}
	profile, ok := acc.Profiles[f.SelectedUser.Profile]
	if !ok || profile.DisplayName == "" {
		return "", fmt.Errorf("selected profile not found in %s", path)
	}
	return profile.DisplayName, nil
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./internal/minecraft/... -v -run TestReadPrism
go test ./internal/minecraft/... -v -run TestReadVanilla
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/minecraft/accounts.go internal/minecraft/accounts_test.go
git commit -m "feat: add Minecraft account detection"
```

---

### Task 5: Minecraft source discovery

**Files:**
- Create: `internal/minecraft/sources.go`
- Create: `internal/minecraft/sources_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/minecraft/sources_test.go
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
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/minecraft/... -run TestDiscover
```

Expected: compile error — `Discover`, `SourcePrism`, `SourceVanilla` not defined.

- [ ] **Step 3: Write `internal/minecraft/sources.go`**

```go
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
```

- [ ] **Step 4: Run all minecraft tests**

```bash
go test ./internal/minecraft/... -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/minecraft/sources.go internal/minecraft/sources_test.go
git commit -m "feat: add Minecraft source discovery"
```

---

### Task 6: Immich HTTP client

**Files:**
- Create: `internal/immich/client.go`
- Create: `internal/immich/client_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/immich/client_test.go
package immich_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/malc/screenshot-sync/internal/immich"
)

func fakeServer(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/assets":
			json.NewEncoder(w).Encode(map[string]any{"id": "asset-1", "status": "created"})
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/api/assets/"):
			json.NewEncoder(w).Encode(map[string]any{"id": "asset-1"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/tags":
			json.NewEncoder(w).Encode([]any{})
		case r.Method == http.MethodPost && r.URL.Path == "/api/tags":
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			json.NewEncoder(w).Encode(map[string]any{"id": "tag-" + body["name"], "name": body["name"]})
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/api/tags/"):
			json.NewEncoder(w).Encode(map[string]any{"successIds": []string{"asset-1"}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/albums":
			json.NewEncoder(w).Encode([]any{})
		case r.Method == http.MethodPost && r.URL.Path == "/api/albums":
			json.NewEncoder(w).Encode(map[string]any{"id": "album-1", "albumName": "Minecraft"})
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/api/albums/"):
			json.NewEncoder(w).Encode(map[string]any{"successIds": []string{"asset-1"}})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return srv, &calls
}

func TestUploadAsset(t *testing.T) {
	srv, _ := fakeServer(t)
	defer srv.Close()

	f := filepath.Join(t.TempDir(), "screenshot.png")
	os.WriteFile(f, []byte("PNG"), 0644)

	c := immich.NewClient(srv.URL, "testkey")
	assetID, dup, err := c.UploadAsset(context.Background(), "prism/World/screenshot.png", f)
	if err != nil {
		t.Fatalf("UploadAsset: %v", err)
	}
	if assetID != "asset-1" {
		t.Errorf("assetID = %q", assetID)
	}
	if dup {
		t.Error("should not be duplicate")
	}
}

func TestSetDescription(t *testing.T) {
	srv, _ := fakeServer(t)
	defer srv.Close()

	c := immich.NewClient(srv.URL, "testkey")
	if err := c.SetDescription(context.Background(), "asset-1", "test description"); err != nil {
		t.Fatalf("SetDescription: %v", err)
	}
}

func TestApplyTagsCreatesAndCaches(t *testing.T) {
	srv, calls := fakeServer(t)
	defer srv.Close()

	c := immich.NewClient(srv.URL, "testkey")
	tags := []string{"minecraft", "mc-instance:Survival"}

	if err := c.ApplyTags(context.Background(), "asset-1", tags); err != nil {
		t.Fatalf("ApplyTags: %v", err)
	}
	// Call again — tag cache should prevent duplicate GET /api/tags
	if err := c.ApplyTags(context.Background(), "asset-1", tags); err != nil {
		t.Fatalf("second ApplyTags: %v", err)
	}
	getTagsCalls := 0
	for _, c := range *calls {
		if c == "GET /api/tags" {
			getTagsCalls++
		}
	}
	if getTagsCalls > 1 {
		t.Errorf("GET /api/tags called %d times, want 1 (should be cached)", getTagsCalls)
	}
}

func TestAddToAlbumCreatesAndCaches(t *testing.T) {
	srv, calls := fakeServer(t)
	defer srv.Close()

	c := immich.NewClient(srv.URL, "testkey")
	if err := c.AddToAlbum(context.Background(), "asset-1", "Minecraft"); err != nil {
		t.Fatalf("AddToAlbum: %v", err)
	}
	if err := c.AddToAlbum(context.Background(), "asset-1", "Minecraft"); err != nil {
		t.Fatalf("second AddToAlbum: %v", err)
	}
	getAlbumCalls := 0
	for _, c := range *calls {
		if c == "GET /api/albums" {
			getAlbumCalls++
		}
	}
	if getAlbumCalls > 1 {
		t.Errorf("GET /api/albums called %d times, want 1 (should be cached)", getAlbumCalls)
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/immich/...
```

Expected: compile error.

- [ ] **Step 3: Write `internal/immich/client.go`**

```go
package immich

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Client is an Immich REST API client.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	hostname   string

	mu         sync.Mutex
	tagCache   map[string]string // tag name -> tag ID
	albumCache map[string]string // album name -> album ID
}

// NewClient creates a new Immich client.
func NewClient(baseURL, apiKey string) *Client {
	hostname, _ := os.Hostname()
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 60 * time.Second},
		hostname:   hostname,
		tagCache:   make(map[string]string),
		albumCache: make(map[string]string),
	}
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("x-api-key", c.apiKey)
	return c.httpClient.Do(req)
}


// UploadAsset uploads a file to Immich. Returns the asset ID and whether it was a duplicate.
func (c *Client) UploadAsset(ctx context.Context, deviceAssetID, filePath string) (string, bool, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", false, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", false, fmt.Errorf("stat file: %w", err)
	}
	modTime := info.ModTime().UTC().Format(time.RFC3339)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("deviceAssetId", deviceAssetID)
	mw.WriteField("deviceId", c.hostname)
	mw.WriteField("fileCreatedAt", modTime)
	mw.WriteField("fileModifiedAt", modTime)
	mw.WriteField("isFavorite", "false")

	part, err := mw.CreateFormFile("assetData", filepath.Base(filePath))
	if err != nil {
		return "", false, err
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", false, err
	}
	mw.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/assets", &body)
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", false, fmt.Errorf("upload: status %d: %s", resp.StatusCode, b)
	}

	var result struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", false, fmt.Errorf("decode upload response: %w", err)
	}
	return result.ID, result.Status == "duplicate", nil
}

// SetDescription sets the description on an existing Immich asset.
func (c *Client) SetDescription(ctx context.Context, assetID, description string) error {
	body, _ := json.Marshal(map[string]string{"description": description})
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/api/assets/"+assetID, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("set description: status %d: %s", resp.StatusCode, b)
	}
	return nil
}

// ApplyTags looks up or creates each tag, then applies them to the asset.
func (c *Client) ApplyTags(ctx context.Context, assetID string, tags []string) error {
	if err := c.ensureTagsExist(ctx, tags); err != nil {
		return err
	}
	c.mu.Lock()
	tagIDs := make([]string, 0, len(tags))
	for _, t := range tags {
		if id, ok := c.tagCache[t]; ok {
			tagIDs = append(tagIDs, id)
		}
	}
	c.mu.Unlock()

	for _, tagID := range tagIDs {
		body, _ := json.Marshal(map[string][]string{"ids": {assetID}})
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/api/tags/"+tagID+"/assets", bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
	}
	return nil
}

func (c *Client) ensureTagsExist(ctx context.Context, tags []string) error {
	c.mu.Lock()
	missing := []string{}
	for _, t := range tags {
		if _, ok := c.tagCache[t]; !ok {
			missing = append(missing, t)
		}
	}
	c.mu.Unlock()

	if len(missing) == 0 {
		return nil
	}

	// Fetch existing tags once
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/tags", nil)
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var existing []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	json.NewDecoder(resp.Body).Decode(&existing)

	c.mu.Lock()
	for _, t := range existing {
		c.tagCache[t.Name] = t.ID
	}
	// Recalculate missing after fetching
	stillMissing := []string{}
	for _, t := range missing {
		if _, ok := c.tagCache[t]; !ok {
			stillMissing = append(stillMissing, t)
		}
	}
	c.mu.Unlock()

	for _, name := range stillMissing {
		body, _ := json.Marshal(map[string]string{"name": name})
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/tags", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.do(req)
		if err != nil {
			return err
		}
		var created struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		json.NewDecoder(resp.Body).Decode(&created)
		resp.Body.Close()
		if created.ID != "" {
			c.mu.Lock()
			c.tagCache[created.Name] = created.ID
			c.mu.Unlock()
		}
	}
	return nil
}

// AddToAlbum looks up or creates the named album, then adds the asset to it.
func (c *Client) AddToAlbum(ctx context.Context, assetID, albumName string) error {
	albumID, err := c.ensureAlbumExists(ctx, albumName)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string][]string{"ids": {assetID}})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/api/albums/"+albumID+"/assets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) ensureAlbumExists(ctx context.Context, albumName string) (string, error) {
	c.mu.Lock()
	if id, ok := c.albumCache[albumName]; ok {
		c.mu.Unlock()
		return id, nil
	}
	c.mu.Unlock()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/albums", nil)
	resp, err := c.do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var albums []struct {
		ID        string `json:"id"`
		AlbumName string `json:"albumName"`
	}
	json.NewDecoder(resp.Body).Decode(&albums)

	c.mu.Lock()
	for _, a := range albums {
		c.albumCache[a.AlbumName] = a.ID
	}
	if id, ok := c.albumCache[albumName]; ok {
		c.mu.Unlock()
		return id, nil
	}
	c.mu.Unlock()

	// Create the album
	body, _ := json.Marshal(map[string]any{"albumName": albumName, "description": "", "assetIds": []string{}})
	createReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/albums", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := c.do(createReq)
	if err != nil {
		return "", err
	}
	defer createResp.Body.Close()
	var created struct {
		ID string `json:"id"`
	}
	json.NewDecoder(createResp.Body).Decode(&created)
	if created.ID == "" {
		return "", fmt.Errorf("create album returned empty ID")
	}
	c.mu.Lock()
	c.albumCache[albumName] = created.ID
	c.mu.Unlock()
	return created.ID, nil
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./internal/immich/... -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/immich/
git commit -m "feat: add Immich HTTP client"
```

---

### Task 7: Upload orchestrator

**Files:**
- Create: `internal/upload/upload.go`
- Create: `internal/upload/upload_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/upload/upload_test.go
package upload_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/malc/screenshot-sync/internal/minecraft"
	"github.com/malc/screenshot-sync/internal/state"
	"github.com/malc/screenshot-sync/internal/upload"
)

type mockUploader struct {
	uploadedID  string
	description string
	tags        []string
	album       string
}

func (m *mockUploader) UploadAsset(_ context.Context, deviceAssetID, _ string) (string, bool, error) {
	m.uploadedID = deviceAssetID
	return "asset-123", false, nil
}
func (m *mockUploader) SetDescription(_ context.Context, _, desc string) error {
	m.description = desc
	return nil
}
func (m *mockUploader) ApplyTags(_ context.Context, _ string, tags []string) error {
	m.tags = tags
	return nil
}
func (m *mockUploader) AddToAlbum(_ context.Context, _, album string) error {
	m.album = album
	return nil
}

func TestScreenshotUploadsPrism(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "2024-01-15_12.30.00.png")
	os.WriteFile(f, []byte("PNG"), 0644)

	st, _ := state.Load(filepath.Join(dir, "state.json"))
	u := &mockUploader{}
	src := minecraft.Source{
		Type:         minecraft.SourcePrism,
		InstanceName: "Survival",
		AccountName:  "Steve",
	}

	if err := upload.Screenshot(context.Background(), u, st, "Minecraft", src, f); err != nil {
		t.Fatalf("Screenshot: %v", err)
	}

	if u.uploadedID != "prism/Survival/2024-01-15_12.30.00.png" {
		t.Errorf("deviceAssetID = %q", u.uploadedID)
	}
	if u.description != "Minecraft screenshot | Instance: Survival | Account: Steve" {
		t.Errorf("description = %q", u.description)
	}
	expectedTags := []string{"minecraft", "mc-instance:Survival", "mc-account:Steve"}
	if len(u.tags) != len(expectedTags) {
		t.Errorf("tags = %v, want %v", u.tags, expectedTags)
	}
	if u.album != "Minecraft" {
		t.Errorf("album = %q", u.album)
	}
	if !st.IsUploaded("prism/Survival/2024-01-15_12.30.00.png") {
		t.Error("state not marked as uploaded")
	}
}

func TestScreenshotSkipsAlreadyUploaded(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.png")
	os.WriteFile(f, []byte("PNG"), 0644)

	st, _ := state.Load(filepath.Join(dir, "state.json"))
	st.MarkUploaded("prism/Survival/a.png", "existing-id")

	u := &mockUploader{}
	src := minecraft.Source{Type: minecraft.SourcePrism, InstanceName: "Survival", AccountName: "Steve"}

	upload.Screenshot(context.Background(), u, st, "Minecraft", src, f)

	if u.uploadedID != "" {
		t.Error("should have skipped upload for already-uploaded file")
	}
}

func TestScreenshotVanillaDeviceAssetID(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "2024-02-01_10.00.00.png")
	os.WriteFile(f, []byte("PNG"), 0644)

	st, _ := state.Load(filepath.Join(dir, "state.json"))
	u := &mockUploader{}
	src := minecraft.Source{Type: minecraft.SourceVanilla, InstanceName: "vanilla", AccountName: "Alex"}

	upload.Screenshot(context.Background(), u, st, "Minecraft", src, f)

	if u.uploadedID != "vanilla/2024-02-01_10.00.00.png" {
		t.Errorf("deviceAssetID = %q", u.uploadedID)
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/upload/...
```

Expected: compile error.

- [ ] **Step 3: Write `internal/upload/upload.go`**

```go
package upload

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/malc/screenshot-sync/internal/minecraft"
	"github.com/malc/screenshot-sync/internal/state"
)

// Uploader is the interface for uploading assets to Immich.
// *immich.Client satisfies this interface.
type Uploader interface {
	UploadAsset(ctx context.Context, deviceAssetID, filePath string) (assetID string, duplicate bool, err error)
	SetDescription(ctx context.Context, assetID, description string) error
	ApplyTags(ctx context.Context, assetID string, tags []string) error
	AddToAlbum(ctx context.Context, assetID, albumName string) error
}

// Screenshot uploads a single Minecraft screenshot to Immich.
// It is a no-op if the file has already been uploaded according to st.
// Each API step is retried once on failure before returning an error.
func Screenshot(ctx context.Context, u Uploader, st *state.State, albumName string, src minecraft.Source, filePath string) error {
	id := deviceAssetID(src, filePath)
	if st.IsUploaded(id) {
		return nil
	}

	var assetID string
	if err := retry(func() error {
		var err error
		assetID, _, err = u.UploadAsset(ctx, id, filePath)
		return err
	}); err != nil {
		return fmt.Errorf("upload asset: %w", err)
	}

	desc := fmt.Sprintf("Minecraft screenshot | Instance: %s | Account: %s", src.InstanceName, src.AccountName)
	if err := retry(func() error {
		return u.SetDescription(ctx, assetID, desc)
	}); err != nil {
		return fmt.Errorf("set description: %w", err)
	}

	tags := []string{
		"minecraft",
		fmt.Sprintf("mc-instance:%s", src.InstanceName),
		fmt.Sprintf("mc-account:%s", src.AccountName),
	}
	if err := retry(func() error {
		return u.ApplyTags(ctx, assetID, tags)
	}); err != nil {
		return fmt.Errorf("apply tags: %w", err)
	}

	if err := retry(func() error {
		return u.AddToAlbum(ctx, assetID, albumName)
	}); err != nil {
		return fmt.Errorf("add to album: %w", err)
	}

	return st.MarkUploaded(id, assetID)
}

// retry calls fn, retrying once on failure.
func retry(fn func() error) error {
	if err := fn(); err != nil {
		return fn()
	}
	return nil
}

func deviceAssetID(src minecraft.Source, filePath string) string {
	filename := filepath.Base(filePath)
	if src.Type == minecraft.SourcePrism {
		return fmt.Sprintf("prism/%s/%s", src.InstanceName, filename)
	}
	return fmt.Sprintf("vanilla/%s", filename)
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./internal/upload/... -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/upload/
git commit -m "feat: add upload orchestration"
```

---

### Task 8: File watcher

**Files:**
- Create: `internal/watcher/watcher.go`
- Create: `internal/watcher/watcher_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/watcher/watcher_test.go
package watcher_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/malc/screenshot-sync/internal/config"
	"github.com/malc/screenshot-sync/internal/minecraft"
	"github.com/malc/screenshot-sync/internal/watcher"
)

func TestWatcherDetectsNewPNG(t *testing.T) {
	prismDir := t.TempDir()
	screenshotsDir := filepath.Join(prismDir, "instances", "Test", ".minecraft", "screenshots")
	os.MkdirAll(screenshotsDir, 0755)

	cfg := &config.Config{
		Sources: config.SourcesConfig{PrismDir: prismDir, Vanilla: false},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	received := make(chan string, 1)
	go func() {
		w := watcher.New(cfg)
		w.Watch(ctx, func(src minecraft.Source, path string) {
			received <- path
		})
	}()

	// Give the watcher time to start
	time.Sleep(100 * time.Millisecond)

	pngPath := filepath.Join(screenshotsDir, "2024-01-01_10.00.00.png")
	os.WriteFile(pngPath, []byte("PNG"), 0644)

	select {
	case got := <-received:
		if got != pngPath {
			t.Errorf("got %q, want %q", got, pngPath)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for watcher event")
	}
}
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
go test ./internal/watcher/...
```

Expected: compile error.

- [ ] **Step 3: Write `internal/watcher/watcher.go`**

```go
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
		sources, _ := minecraft.Discover(w.cfg)
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
					time.Sleep(500 * time.Millisecond)
					onFile(s, p)
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
```

- [ ] **Step 4: Run test to confirm it passes**

```bash
go test ./internal/watcher/... -v -timeout 10s
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/watcher/
git commit -m "feat: add file watcher with instance polling"
```

---

### Task 9: Icon assets

**Files:**
- Create: `internal/assets/icon.go`

No tests needed — purely generative code.

- [ ] **Step 1: Write `internal/assets/icon.go`**

```go
// Package assets provides embedded resources for the tray application.
package assets

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

// IconIdle returns a 16x16 green PNG icon for the idle tray state.
func IconIdle() []byte { return iconPNG(color.RGBA{0x5a, 0xac, 0x2b, 0xff}) }

// IconSyncing returns a 16x16 yellow PNG icon for the syncing tray state.
func IconSyncing() []byte { return iconPNG(color.RGBA{0xff, 0xd7, 0x00, 0xff}) }

// IconError returns a 16x16 red PNG icon for the error tray state.
func IconError() []byte { return iconPNG(color.RGBA{0xcc, 0x33, 0x33, 0xff}) }

func iconPNG(c color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/assets/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/assets/
git commit -m "feat: add programmatic tray icon assets"
```

---

### Task 10: CLI — sync and daemon subcommands

**Files:**
- Modify: `cmd/root.go` (add shared `setup()` and `runCatchupSync()`)
- Create: `cmd/sync.go`
- Create: `cmd/daemon.go`

- [ ] **Step 1: Update `cmd/root.go` with shared helpers**

```go
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
```

- [ ] **Step 2: Write `cmd/sync.go`**

```go
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
```

- [ ] **Step 3: Write `cmd/daemon.go`**

```go
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
```

- [ ] **Step 4: Verify build**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 5: Smoke-test sync subcommand locally** (will fail to connect to Immich — that's expected)

```bash
go run . sync 2>&1 | head -5
```

Expected: error like `config.toml not found` — confirms the CLI is wired up correctly.

- [ ] **Step 6: Commit**

```bash
git add cmd/root.go cmd/sync.go cmd/daemon.go
git commit -m "feat: add sync and daemon subcommands"
```

---

### Task 11: Tray subcommand

**Files:**
- Create: `cmd/tray.go`

Note: `getlantern/systray` requires CGO. On Linux the build requires GTK dev libraries (`apt install libgtk-3-dev libappindicator3-dev`). For the Windows target, use the cross-compilation command in the Makefile (Task 12).

- [ ] **Step 1: Write `cmd/tray.go`**

```go
package cmd

import (
	"context"
	"log"

	"github.com/getlantern/systray"
	"github.com/malc/screenshot-sync/internal/assets"
	"github.com/malc/screenshot-sync/internal/immich"
	"github.com/malc/screenshot-sync/internal/minecraft"
	"github.com/malc/screenshot-sync/internal/state"
	"github.com/malc/screenshot-sync/internal/upload"
	"github.com/malc/screenshot-sync/internal/watcher"
	"github.com/malc/screenshot-sync/internal/config"
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
			setIcon(systray.SetIcon, assets.IconSyncing())
			runCatchupSync(ctx, cfg, client, st)
			setIcon(systray.SetIcon, assets.IconIdle())
		}()

		// File watcher
		go func() {
			w := watcher.New(cfg)
			w.Watch(ctx, func(src minecraft.Source, path string) {
				setIcon(systray.SetIcon, assets.IconSyncing())
				if err := upload.Screenshot(ctx, client, st, cfg.Immich.Album, src, path); err != nil {
					log.Printf("upload %s: %v", path, err)
					setIcon(systray.SetIcon, assets.IconError())
					systray.SetTooltip("Error: " + err.Error())
					return
				}
				setIcon(systray.SetIcon, assets.IconIdle())
				systray.SetTooltip("Minecraft Screenshot Sync")
			})
		}()

		for {
			select {
			case <-mSync.ClickedCh:
				go func() {
					setIcon(systray.SetIcon, assets.IconSyncing())
					runCatchupSync(ctx, cfg, client, st)
					setIcon(systray.SetIcon, assets.IconIdle())
				}()
			case <-mQuit.ClickedCh:
				cancel()
				systray.Quit()
				return
			}
		}
	}
}

// setIcon is a thin wrapper to allow passing systray.SetIcon as a value.
func setIcon(fn func([]byte), icon []byte) {
	fn(icon)
}
```

- [ ] **Step 2: Verify it compiles for the current platform**

```bash
go build ./cmd/... 2>&1 | head -20
```

On a headless Linux build machine without GTK headers, this may fail with a CGO error — that is expected. The binary is cross-compiled for Windows in Task 12. If GTK headers are present, it will succeed.

- [ ] **Step 3: Commit**

```bash
git add cmd/tray.go
git commit -m "feat: add tray subcommand"
```

---

### Task 12: Build config, example config, gitignore

**Files:**
- Create: `Makefile`
- Create: `config.toml.example`
- Create: `.gitignore`

- [ ] **Step 1: Write `Makefile`**

```makefile
.PHONY: build build-windows clean

# Native build (Linux)
build:
	go build -o screenshot-sync .

# Cross-compile for Windows (requires: apt install gcc-mingw-w64-x86-64)
build-windows:
	CGO_ENABLED=1 \
	CC=x86_64-w64-mingw32-gcc \
	GOOS=windows \
	GOARCH=amd64 \
	go build -o screenshot-sync.exe .

test:
	go test ./internal/... -v

clean:
	rm -f screenshot-sync screenshot-sync.exe
```

- [ ] **Step 2: Write `config.toml.example`**

```toml
[immich]
# URL of your Immich instance (same URL used inside and outside your network)
url = "https://photos.example.com"

# Immich API key — create one in Immich under Account Settings > API Keys
api_key = "your-api-key-here"

# Album name in Immich where screenshots will be collected (created automatically if missing)
album = "Minecraft"

[sources]
# Path to your Prism Launcher root directory (portable install).
# If omitted, falls back to %APPDATA%\PrismLauncher (standard install location).
prism_dir = "D:/Games/Minecraft/Prism Launcher"

# Set to true to also watch the vanilla launcher's screenshots folder
# (%APPDATA%\.minecraft\screenshots)
vanilla = true
```

- [ ] **Step 3: Write `.gitignore`**

```gitignore
screenshot-sync
screenshot-sync.exe
state.json
config.toml
```

- [ ] **Step 4: Run all tests one final time**

```bash
go test ./internal/... -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add Makefile config.toml.example .gitignore
git commit -m "feat: add Makefile, example config, and gitignore"
```

---

## Setup on a New Windows Machine

1. Copy `screenshot-sync.exe` to a permanent location (e.g. `C:\Users\<you>\AppData\Local\screenshot-sync\`)
2. Copy `config.toml.example` to the same folder, rename to `config.toml`, and fill in your Immich URL and API key
3. Create a shortcut to `screenshot-sync.exe tray` in:
   `%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup`
4. Run the shortcut once (or restart) — the tray icon will appear and sync will begin
