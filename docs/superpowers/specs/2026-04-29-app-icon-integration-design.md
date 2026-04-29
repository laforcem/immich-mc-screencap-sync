# App Icon Integration Design

## Problem

The app ships with no visual identity. The system tray shows solid-color squares (green/yellow/red). The `.exe` has no embedded icon, so Windows displays a generic placeholder in Explorer, the Start menu, and the console window.

## Goal

Embed the Immich pixel-art logo as the app's identity icon throughout Windows, and replace the solid-color tray squares with purposeful state icons.

## Tray Icon States

| State | Icon | Color |
|---|---|---|
| Idle | Immich flower logo | as-is |
| Syncing | Material `arrow_upload_progress` | green |
| Error | Material `error` | red |

Icons are static. No animation.

## Icon Sizes

### Exe resource ICO (embedded via `resource.syso`)

Used by Explorer file views, Start menu, console window title bar, and alt-tab.

| Size | Context |
|---|---|
| 16×16 | Explorer list view, Start menu app list (compact), console title bar |
| 32×32 | Explorer medium view, Start menu pinned (standard DPI), alt-tab |
| 48×48 | Explorer large view, Start menu pinned |
| 256×256 | Explorer extra-large view, high-DPI Start menu (stored as PNG inside ICO) |

### Tray ICO (Windows, per state)

Windows picks the size that best matches the current DPI scaling.

| Size | DPI scale |
|---|---|
| 16×16 | 100% |
| 20×20 | 125% |
| 24×24 | 150% |
| 32×32 | 200% |

### Tray PNG (Linux, per state)

Single 22×22 PNG covers both GNOME (22px) and KDE (24px, scaled).

## Icon Generation Pipeline

### Source material

- **Immich logo**: `assets/icons/immich.svg` → render at each target size via `magick -background none immich.svg -resize NxN base_N.png` → apply `make_icon.py` (border darkening factor 0.70)
- **Material icons**: User downloads `arrow_upload_progress` and `error` SVGs from fonts.google.com/icons and provides them (default black fill). Pipeline colorizes each SVG by rewriting its `fill` attribute to the target color (green / red), then renders at each target size, then applies `make_icon.py`.

### Output files (committed to repo)

```
assets/icons/
  immich_16.png
  immich_20.png
  immich_24.png
  immich_32.png    (generated fresh from SVG; supersedes immich_32_v1_subtle.png)
  immich_48.png
  immich_256.png
  upload_16.png
  upload_20.png
  upload_24.png
  upload_32.png
  upload_22_linux.png
  error_16.png
  error_20.png
  error_24.png
  error_32.png
  error_22_linux.png
  immich_app.ico   (exe resource: 16, 32, 48, 256)
  immich_tray.ico  (idle tray: 16, 20, 24, 32)
  upload_tray.ico  (syncing tray: 16, 20, 24, 32)
  error_tray.ico   (error tray: 16, 20, 24, 32)
```

**User reviews all rendered PNGs before implementation proceeds.**

### Exe resource generation

Run `rsrc` (`github.com/akavel/rsrc`) against `immich_app.ico` to produce `resource.syso` in the root package. Go's linker picks up `.syso` files automatically on `GOOS=windows`. Commit `resource.syso`.

## Go Changes

### `internal/assets`

Replace the runtime solid-color generators with `//go:embed` of the pre-generated ICO/PNG files. Platform split stays:

- `icon_windows.go`: returns ICO bytes
- `icon_other.go`: returns PNG bytes

Public API is unchanged: `IconIdle()`, `IconSyncing()`, `IconError()` still return `[]byte`.

The existing `iconPNG`, `iconICO`, and `iconForTray` helpers are deleted — no longer needed.

### `cmd/tray.go`

No changes. `systray.SetIcon(assets.IconIdle())` etc. continue to work as before.

### `go.mod`

No new runtime dependencies. `rsrc` is a dev tool run once during icon generation, not imported.

## Build and CI

No changes to `Makefile`, `build.bat`, `build.ps1`, or `.github/workflows/release.yml`. The committed `resource.syso` is picked up automatically by `GOOS=windows go build`.

## Testing

Existing `icon_test.go` tests the ICO binary format. After this change those tests become obsolete (we embed pre-built files rather than generating ICO bytes at runtime). Replace with smoke tests that verify the embedded bytes are non-empty and start with the correct magic bytes (PNG: `\x89PNG`; ICO: `\x00\x00\x01\x00`).

## Review Gate

All rendered icon PNGs are presented to the user for approval before any Go code changes.
