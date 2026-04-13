# Minecraft Screenshot Sync — Design Spec

**Date:** 2026-04-13
**Status:** Approved

## Overview

A Go binary (`screenshot-sync.exe`) that watches Minecraft screenshot directories on Windows PCs and uploads new screenshots to a self-hosted Immich instance. Each photo is tagged with the Minecraft account name and instance name. Supports automatic sync via a file watcher and manual sync via a CLI command or system tray menu.

The binary is compiled for Windows but buildable from Linux using standard Go cross-compilation with `mingw-w64`.

---

## Architecture

Single binary, three subcommands, shared internal library:

```
screenshot-sync/
├── main.go
├── cmd/
│   ├── tray.go        # systray + embedded file watcher
│   ├── sync.go        # one-shot upload of all unsynced screenshots
│   └── daemon.go      # headless watcher (no tray, for Task Scheduler)
└── internal/
    ├── config/        # TOML config loading
    ├── immich/        # HTTP client for Immich API
    ├── minecraft/     # source path discovery + account detection
    ├── watcher/       # fsnotify wrapper
    └── state/         # JSON state file tracking uploaded files
```

### Cross-compilation

The system tray requires CGO on Windows. Cross-compile from Linux with:

```bash
apt install gcc-mingw-w64-x86-64
CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go build -o screenshot-sync.exe
```

All other components are pure Go.

---

## Configuration

TOML file. Searched first next to the binary (portable), then at `%APPDATA%\screenshot-sync\config.toml`.

```toml
[immich]
url     = "https://photos.example.com"
api_key = "your-api-key-here"
album   = "Minecraft"

[sources]
# Prism Launcher root (portable install). If omitted, falls back to %APPDATA%\PrismLauncher
prism_dir = "D:/Games/Minecraft/Prism Launcher"

# Set to false to disable vanilla launcher screenshots
vanilla = true
```

Only `immich.url` and `immich.api_key` are required. All path fields have sensible defaults.

A `config.toml.example` is shipped alongside the binary for new machine setup.

---

## Source Discovery & Account Detection

Discovered at startup, re-polled every 60 seconds for new instance folders.

### Prism Launcher

- **Screenshots:** `<prism_dir>/instances/<instance_name>/.minecraft/screenshots/`
- **Primary path:** configured `prism_dir`
- **Fallback path:** `%APPDATA%\PrismLauncher\instances\`
- **Account detection:** parse `<prism_dir>/accounts.json` for the active account username

### Vanilla Launcher

- **Screenshots:** `%APPDATA%\.minecraft\screenshots\`
- **Account detection:** parse `%APPDATA%\.minecraft\launcher_profiles.json` for `lastUsedProfile` and its associated account name
- **Instance name:** tagged as `"vanilla"`

If account detection fails (file missing or unparseable), falls back to `"unknown"` and logs a warning. The upload proceeds regardless.

---

## Immich Integration

All requests include `x-api-key: <api_key>` header. Album ID and tag IDs are cached in memory after first lookup. Each step is retried once on transient failure before logging an error and moving on.

### Upload flow per screenshot

1. **Upload asset** — `POST /api/assets` (multipart form)
   - `deviceAssetId`: stable relative path, e.g. `prism/SurvivalWorld/2024-01-15_12.30.00.png`

2. **Set description** — `PUT /api/assets/<id>` with `description: "Minecraft screenshot | Instance: SurvivalWorld | Account: Steve"`

3. **Apply tags** — look up or create via `GET /api/tags` + `POST /api/tags`, apply with `PUT /api/tags/<id>/assets`
   - Tags: `minecraft`, `mc-instance:<name>`, `mc-account:<username>`

4. **Add to album** — look up or create via `GET /api/albums` + `POST /api/albums`, add with `PUT /api/albums/<id>/assets`
   - Album name from config (default: `"Minecraft"`)

---

## File Watcher & Upload Flow

Uses `fsnotify` to watch all discovered screenshot directories.

On `CREATE` event for a `.png` file:
1. Wait 500ms (Minecraft may still be writing the file)
2. Check local state — skip if already uploaded
3. Run upload flow (asset → tags → album)
4. On success, record in local state

The watcher handles directories that don't exist at startup — new instance folders are picked up during the 60-second poll cycle.

On startup, `tray` and `daemon` modes run a one-shot catch-up sync to pick up screenshots taken while the tool was not running.

---

## Subcommands

| Command                   | Behaviour |
|---------------------------|-----------|
| `screenshot-sync tray`    | File watcher + system tray icon. Normal day-to-day mode. Add a shortcut to the Windows Startup folder. |
| `screenshot-sync sync`    | One-shot catch-up sync, exits when done. Good for scripting. |
| `screenshot-sync daemon`  | Headless watcher, no tray. For Task Scheduler or users who prefer no tray icon. |

### Tray icon states

| State    | Indicator      | Notes |
|----------|----------------|-------|
| Idle     | Grey/green     | Watching, nothing in progress |
| Syncing  | Yellow         | Upload in progress |
| Error    | Red            | Last sync had failures; tooltip shows brief message |

**Tray menu:** "Sync Now" · ─── · "Quit"

---

## State Tracking & Deduplication

A `state.json` file stored alongside the config maps `deviceAssetId → immich asset ID`:

```json
{
  "prism/SurvivalWorld/2024-01-15_12.30.00.png": "abc123",
  "vanilla/2024-01-16_09.15.00.png": "def456"
}
```

- Written atomically (write to temp file, rename) to prevent corruption
- On startup, catch-up sync skips any file already in the map
- If `state.json` is missing or corrupt, falls back to Immich's `deviceAssetId` deduplication — duplicate uploads return the existing asset ID rather than creating a second copy

---

## Distribution & Setup

New machine setup:
1. Copy `screenshot-sync.exe` and fill in `config.toml` (from `config.toml.example`)
2. Add a shortcut to `screenshot-sync.exe tray` in the Windows Startup folder (`%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup`)

No installer, no admin rights required.
