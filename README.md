# immich-mc-screencap-sync

Automatically sync Minecraft screenshots to your [Immich](https://immich.app) photo server. Watches Prism Launcher and/or the vanilla Minecraft launcher for new screenshots and uploads them — tagged by instance and player — as soon as they appear.

Three run modes:
- **tray** — background watcher with a system tray icon (recommended for desktop)
- **daemon** — headless watcher, no tray icon (good for servers or scripted startup)
- **sync** — one-shot upload of all unsynced screenshots, then exit

---

## Quick Start

1. **Download** the latest binary for your platform from the [Releases](../../releases) page.

2. **Copy the example config** next to the binary and fill it in:

   ```
   cp config.toml.example config.toml
   ```

3. **Edit `config.toml`** — at minimum set `immich.url` and `immich.api_key`:

   ```toml
   [immich]
   url     = "https://photos.example.com"
   api_key = "your-api-key-here"
   album   = "Minecraft"

   [sources]
   prism_dir = "D:/Games/Minecraft/Prism Launcher"
   vanilla   = false
   ```

4. **Run:**

   ```
   screenshot-sync tray
   ```

---

## Configuration Reference

Config is loaded from the first location that exists:

1. `config.toml` next to the binary
2. `%APPDATA%\screenshot-sync\config.toml`

| Key | Required | Default | Description |
|-----|----------|---------|-------------|
| `immich.url` | **Yes** | — | Base URL of your Immich instance (same URL inside and outside your network) |
| `immich.api_key` | **Yes** | — | Immich API key — see [API Key Scope](#immich-api-key-scope) |
| `immich.album` | No | `"Minecraft"` | Album name in Immich; created automatically if it doesn't exist |
| `sources.prism_dir` | No | `%APPDATA%\PrismLauncher` | Path to your Prism Launcher root directory. Required if Prism is installed portably. |
| `sources.vanilla` | No | `false` | Set to `true` to also watch the vanilla launcher's screenshots folder |

---

## Supported Paths

### Prism Launcher

Each instance under your Prism root is discovered automatically.

| Path | Notes |
|------|-------|
| `<prism_dir>/instances/<instance>/minecraft/screenshots/` | Standard path (Prism ≥ 9) |
| `<prism_dir>/instances/<instance>/.minecraft/screenshots/` | Legacy fallback — used only if the modern path does not exist |

Default `prism_dir` when `sources.prism_dir` is not set: `%APPDATA%\PrismLauncher`

> **macOS / Linux:** `%APPDATA%` auto-discovery is Windows-only. Set `sources.prism_dir` explicitly in your config.

### Vanilla Launcher

Enabled with `sources.vanilla = true`.

| Path |
|------|
| `%APPDATA%\.minecraft\screenshots\` |

> **macOS / Linux:** Vanilla auto-discovery is Windows-only.

---

## Immich API Key Scope

In Immich: **Account Settings → API Keys → New API Key**

The following permissions are required:

| Resource | Permissions needed |
|----------|--------------------|
| Asset | Read, Upload |
| Album | Read, Create, Update |
| Tag | Read, Create, Apply |

No admin permissions are required.

---

## Running Modes

| Command | Description |
|---------|-------------|
| `screenshot-sync tray` | Runs as a background watcher with a system tray icon. Recommended for everyday desktop use. |
| `screenshot-sync daemon` | Runs in the foreground with no tray icon. Suitable for headless servers or launching from a script/service. |
| `screenshot-sync sync` | Uploads all unsynced screenshots once and exits. Useful for a manual catch-up or testing your config. |

---

## Dev Environment Setup

**Prerequisites:**
- [Go](https://go.dev/dl/) 1.21 or later
- Git

**Clone and verify:**

```sh
git clone https://github.com/laforcem/immich-mc-screencap-sync.git
cd immich-mc-screencap-sync
go mod download
go test ./internal/... -v
```

All tests should pass with no external dependencies.

---

## Building

### Linux / macOS / Git Bash

```sh
make build
```

Output: `./screenshot-sync`

### Windows (no Make required)

**Command Prompt:**
```bat
build.bat
```

**PowerShell:**
```powershell
.\build.ps1
```

Output: `screenshot-sync.exe`

### Cross-compile for Windows from Linux

```sh
make build-windows
```

Output: `./screenshot-sync.exe`
