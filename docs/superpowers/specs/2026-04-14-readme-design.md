# README Design Spec

**Date:** 2026-04-14
**Issue:** [#5 — Create robust documentation](https://github.com/laforcem/immich-mc-screencap-sync/issues/5)
**Branch:** `docs/readme`

## Goal

Add a comprehensive `README.md` to the repository root that covers everything a new user or contributor needs to get started, configure, and build the tool.

## Sections (in order)

### 1. Project Overview
One-paragraph description: what the tool does, which launchers it supports, and how it integrates with Immich. Include a brief mention of the three run modes.

### 2. Quick Start (End User)
Step-by-step for a non-developer user:
1. Download the release binary for your platform
2. Copy `config.toml.example` → `config.toml` next to the binary
3. Fill in `immich.url`, `immich.api_key`, and optionally `sources.prism_dir`
4. Run `screenshot-sync tray` (or `daemon` for headless)

### 3. Configuration Reference
A table of every config key with: key name, required/optional, default value, description.

| Key | Required | Default | Description |
|-----|----------|---------|-------------|
| `immich.url` | Yes | — | Base URL of your Immich instance |
| `immich.api_key` | Yes | — | Immich API key (see API Key Scope section) |
| `immich.album` | No | `"Minecraft"` | Album name; created automatically if missing |
| `sources.prism_dir` | No | `%APPDATA%\PrismLauncher` | Prism Launcher root directory |
| `sources.vanilla` | No | `false` | Also watch vanilla launcher screenshots |

Config is loaded from next to the binary first, then from `%APPDATA%\screenshot-sync\config.toml`.

### 4. Supported Paths
Exact paths the tool auto-discovers, per source type:

**Prism Launcher**
- Configured: `<prism_dir>/instances/<instance>/minecraft/screenshots/`
- Legacy fallback: `<prism_dir>/instances/<instance>/.minecraft/screenshots/`
- Default root: `%APPDATA%\PrismLauncher`

**Vanilla Launcher**
- `%APPDATA%\.minecraft\screenshots\`
- Only watched when `sources.vanilla = true`

Note: `%APPDATA%` path discovery is Windows-only. macOS/Linux users must set `sources.prism_dir` explicitly.

### 5. Immich API Key Scope
In Immich: **Account Settings → API Keys → New API Key**

Required permissions:
- **Asset** — Read, Upload
- **Album** — Read, Create, Update (add members)
- **Tag** — Read, Create, Apply

No admin permissions required.

### 6. Running Modes

| Command | Description |
|---------|-------------|
| `screenshot-sync sync` | One-shot: upload all unsynced screenshots and exit |
| `screenshot-sync daemon` | Headless watcher: runs in foreground, no tray icon (good for servers/services) |
| `screenshot-sync tray` | System tray: background watcher with tray icon (recommended for desktop use) |

### 7. Dev Environment Setup
Prerequisites:
- Go 1.21+ (`go version`)
- Git

```sh
git clone https://github.com/laforcem/immich-mc-screencap-sync.git
cd immich-mc-screencap-sync
go mod download
go test ./internal/...
```

### 8. Building

**Linux / macOS / Git Bash (via Make):**
```sh
make build
```

**Windows (no Make required):**
```bat
build.bat
```
or in PowerShell:
```powershell
.\build.ps1
```

**Cross-compile for Windows from Linux:**
```sh
make build-windows
```

Output binary: `screenshot-sync` (Linux/macOS) or `screenshot-sync.exe` (Windows).

## Out of Scope

- Installation as a Windows service (not currently supported by the tool)
- macOS tray icon support (tray binary builds on macOS but untested)
- Linux vanilla launcher paths (APPDATA-based discovery is Windows-only)
