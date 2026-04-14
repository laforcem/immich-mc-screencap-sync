# immich-mc-screencap-sync

Automatically sync Minecraft screenshots to your [Immich](https://immich.app) photo server. Watches Prism Launcher and/or the vanilla Minecraft launcher for new screenshots and uploads them as soon as they appear. Uploaded files are tagged by instance and player.

The app has three run modes:

- **tray**: background watcher with a system tray icon

- **daemon**: background watcher, no tray icon

- **sync**: one-shot upload of all unsynced screenshots, then exit

This app is Windows-first until better support for Linux/Mac is added.

---

## Quick Start

1. **Download** the latest binary for your platform from the [Releases](https://github.com/laforcem/immich-mc-screencap-sync/releases) page.

2. **Generate** an API key from Immich. [More on that in their docs](https://api.immich.app/getting-started).

3. **Copy the example config** next to the binary and fill it in:

   **Unix/Git Bash:**

   ```sh
   cp config.toml.example config.toml
   ```

   **Windows PowerShell:**

   ```pwsh
   Copy-Item config.toml.example config.toml
   ```

4. **Edit `config.toml`**. Minimum required fields are `immich.url` and `immich.api_key`:

   ```toml
   [immich]
   url     = "https://photos.example.com"
   api_key = "your-api-key-here"
   album   = "Minecraft"

   [sources]
   prism_dir = "D:/Games/Minecraft/Prism Launcher"
   vanilla   = false
   ```

5. **Run:**

   ```sh
   screenshot-sync tray
   ```

---

## Configuration Reference

Config is loaded from the first location that exists:

1. `config.toml` next to the binary
2. `%APPDATA%\screenshot-sync\config.toml`

| Key | Required | Default | Description |
| ----- | ---------- | --------- | ------------- |
| `immich.url` | **Yes** | - | Base URL of your Immich instance |
| `immich.api_key` | **Yes** | - | Immich API key (see [API Key Scope](#immich-api-key-scope)) |
| `immich.album` | No | `"Minecraft"` | Album name in Immich; created automatically if it doesn't exist |
| `sources.prism_dir` | No | `%APPDATA%\PrismLauncher` | Path to your Prism Launcher root directory. Must be set for portable installs; otherwise the default is used. |
| `sources.vanilla` | No | `false` | Set to `true` to also watch the vanilla launcher's screenshots folder |

---

## Supported Paths

### Prism Launcher

Each instance under your Prism root is discovered automatically.

| Path | Notes |
|------|-------|
| `<prism_dir>/instances/<instance>/minecraft/screenshots/` | Standard path (Prism ≥ 9) |
| `<prism_dir>/instances/<instance>/.minecraft/screenshots/` | Legacy fallback (used only if the modern path does not exist) |

### Vanilla Launcher

Enabled with `sources.vanilla = true`.

| Path |
|------|
| `%APPDATA%\.minecraft\screenshots\` |

> [!NOTE]
> **macOS / Linux:** Vanilla auto-discovery is Windows-only.

---

## Immich API Key Scope

See the [Immich docs](https://api.immich.app/getting-started) for how to create an API key.

The following scopes are required as of Immich v2:

| Resource | Scope |
| ---------- | -------------------- |
| `asset` | `read`, `upload` |
| `album` | `read`, `create`, `update` |
| `tag` | `read`, `create`, `asset` |

---

## Running Modes

| Command | Description |
| --------- | ------------- |
| `screenshot-sync tray` | Runs as a background watcher with a system tray icon. |
| `screenshot-sync daemon` | Runs in the foreground with no tray icon. |
| `screenshot-sync sync` | Uploads all unsynced screenshots once and exits. |

---

## Dev Environment Setup

**Prerequisites:**

- [`go`](https://go.dev/dl/) ≥1.25.5
- `git`

**Checkup:**

Clone the repo, then:

```sh
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

### Windows (no `make`)

**Command Prompt:**

```bat
build.bat
```

**PowerShell:**

```pwsh
.\build.ps1
```

Output: `screenshot-sync.exe`

### Cross-compile for Windows from Linux

```sh
make build-windows
```

Output: `./screenshot-sync.exe`
