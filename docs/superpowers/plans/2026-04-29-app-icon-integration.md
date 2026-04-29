# App Icon Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Embed the Immich pixel-art logo throughout the Windows app (system tray + exe icon) and replace solid-color tray squares with purposeful state icons.

**Architecture:** Pre-generate all icons as PNG/ICO files committed to the repo; embed them via `//go:embed` in `internal/assets`; generate `resource.syso` for the Windows exe icon using `rsrc`. The public `IconIdle()` / `IconSyncing()` / `IconError()` API is unchanged — only the implementation changes.

**Tech Stack:** Python 3 + Pillow + numpy (icon processing), ImageMagick `magick` CLI (SVG rendering + ICO packaging), `github.com/akavel/rsrc` (exe resource generation), Go `//go:embed`

---

## File Map

| File | Action | Purpose |
|---|---|---|
| `assets/icons/process_icon.py` | Create | Generic icon processor: optional colorize + border treatment |
| `assets/icons/immich_{16,20,22,24,32,48,256}.png` | Create | Immich logo at each needed size |
| `assets/icons/upload_{16,20,22,24,32}.png` | Create | Upload-progress icon at each needed size |
| `assets/icons/error_{16,20,22,24,32}.png` | Create | Error icon at each needed size |
| `assets/icons/immich_app.ico` | Create | Exe resource ICO (16, 32, 48, 256) |
| `assets/icons/immich_tray.ico` | Create | Idle tray ICO (16, 20, 24, 32) |
| `assets/icons/upload_tray.ico` | Create | Syncing tray ICO (16, 20, 24, 32) |
| `assets/icons/error_tray.ico` | Create | Error tray ICO (16, 20, 24, 32) |
| `resource.syso` | Create | Windows PE icon resource (auto-linked by go build) |
| `internal/assets/immich_tray.ico` | Create | Embedded idle tray icon (Windows) |
| `internal/assets/upload_tray.ico` | Create | Embedded syncing tray icon (Windows) |
| `internal/assets/error_tray.ico` | Create | Embedded error tray icon (Windows) |
| `internal/assets/immich_22_linux.png` | Create | Embedded idle tray icon (Linux) |
| `internal/assets/upload_22_linux.png` | Create | Embedded syncing tray icon (Linux) |
| `internal/assets/error_22_linux.png` | Create | Embedded error tray icon (Linux) |
| `internal/assets/icon.go` | Modify | Replace runtime generators with `IconIdle/Syncing/Error` returning embed vars |
| `internal/assets/icon_windows.go` | Modify | `//go:embed` ICO files into platform vars |
| `internal/assets/icon_other.go` | Modify | `//go:embed` PNG files into platform vars |
| `internal/assets/icon_test.go` | Modify | Replace ICO format tests with magic-byte smoke tests |

---

## Task 1: Write the generic icon processing script

**Files:**
- Create: `assets/icons/process_icon.py`

- [ ] **Step 1: Write `process_icon.py`**

```python
#!/usr/bin/env python3
"""
Apply pixel-art border treatment to an icon PNG, with optional colorization.

Usage: python process_icon.py <input.png> <output.png> [#RRGGBB] [factor]

  #RRGGBB  Optional — recolors all pixels to this color before treatment
  factor   Border darkening factor, default 0.70

Semi-transparent edge pixels (5 < alpha < 250) are darkened by `factor`
and made fully opaque, giving a crisp pixel-art outline.
"""
import sys
from PIL import Image
import numpy as np

input_path = sys.argv[1]
output_path = sys.argv[2]
color_hex = None
factor = 0.70

for arg in sys.argv[3:]:
    if arg.startswith('#'):
        color_hex = arg
    else:
        factor = float(arg)

img = Image.open(input_path).convert("RGBA")
data = np.array(img, dtype=np.float32)

if color_hex:
    data[:, :, 0] = int(color_hex[1:3], 16)
    data[:, :, 1] = int(color_hex[3:5], 16)
    data[:, :, 2] = int(color_hex[5:7], 16)

r, g, b, a = data[:, :, 0], data[:, :, 1], data[:, :, 2], data[:, :, 3]
border = (a > 5) & (a < 250)
interior = a >= 250

out = data.copy()
out[border, 0] = np.clip(r[border] * factor, 0, 255)
out[border, 1] = np.clip(g[border] * factor, 0, 255)
out[border, 2] = np.clip(b[border] * factor, 0, 255)
out[border, 3] = 255
out[interior, 3] = 255

Image.fromarray(out.astype(np.uint8), "RGBA").save(output_path)
print(f"Wrote {output_path}")
```

- [ ] **Step 2: Set up the Python venv**

```bash
cd assets/icons
python3 -m venv venv
venv/bin/pip install Pillow numpy
```

Expected: `Successfully installed Pillow-... numpy-...`

---

## Task 2: Generate Immich logo icons at all required sizes

**Files:**
- Create: `assets/icons/immich_{16,20,22,24,32,48,256}.png`

Sizes needed: tray Windows (16, 20, 24, 32), tray Linux (22), exe (16, 32, 48, 256). The set is: 16, 20, 22, 24, 32, 48, 256.

- [ ] **Step 1: Render SVG at each size and apply border treatment**

Run from `assets/icons/`:

```bash
cd assets/icons
for SIZE in 16 20 22 24 32 48 256; do
  magick -background none immich.svg -resize ${SIZE}x${SIZE} _base_${SIZE}.png
  venv/bin/python process_icon.py _base_${SIZE}.png immich_${SIZE}.png
  rm _base_${SIZE}.png
done
```

Expected: seven files created — `immich_16.png` through `immich_256.png`.

- [ ] **Step 2: Visually verify spot-check sizes**

```bash
magick immich_16.png -scale 400% /tmp/preview_immich_16.png
magick immich_32.png -scale 400% /tmp/preview_immich_32.png
magick immich_256.png /tmp/preview_immich_256.png
```

Open `/tmp/preview_immich_*.png` and confirm the logo renders correctly at each size.

---

## Task 3: Generate Material icon PNGs at all required sizes

**Files:**
- Create: `assets/icons/upload_{16,20,22,24,32}.png`
- Create: `assets/icons/error_{16,20,22,24,32}.png`

**Prerequisite:** User must place the downloaded SVG files at:
- `assets/icons/arrow_upload_progress.svg`
- `assets/icons/error.svg`

Colors: upload = `#4CAF50` (Material Green 500), error = `#F44336` (Material Red 500).

- [ ] **Step 1: Confirm SVG files are present**

```bash
ls assets/icons/arrow_upload_progress.svg assets/icons/error.svg
```

Expected: both files listed. If not, stop and wait for the user to provide them.

- [ ] **Step 2: Render and colorize upload-progress icon**

Run from `assets/icons/`:

```bash
cd assets/icons
for SIZE in 16 20 22 24 32; do
  magick -background none arrow_upload_progress.svg -resize ${SIZE}x${SIZE} _base_upload_${SIZE}.png
  venv/bin/python process_icon.py _base_upload_${SIZE}.png upload_${SIZE}.png "#4CAF50"
  rm _base_upload_${SIZE}.png
done
```

Expected: five files — `upload_16.png` through `upload_32.png`.

- [ ] **Step 3: Render and colorize error icon**

```bash
cd assets/icons
for SIZE in 16 20 22 24 32; do
  magick -background none error.svg -resize ${SIZE}x${SIZE} _base_error_${SIZE}.png
  venv/bin/python process_icon.py _base_error_${SIZE}.png error_${SIZE}.png "#F44336"
  rm _base_error_${SIZE}.png
done
```

Expected: five files — `error_16.png` through `error_32.png`.

- [ ] **Step 4: Visually verify**

```bash
magick assets/icons/upload_16.png -scale 400% /tmp/preview_upload_16.png
magick assets/icons/upload_32.png -scale 400% /tmp/preview_upload_32.png
magick assets/icons/error_16.png  -scale 400% /tmp/preview_error_16.png
magick assets/icons/error_32.png  -scale 400% /tmp/preview_error_32.png
```

Open previews. Confirm green upload icon and red error icon look correct.

---

## Task 4: USER REVIEW GATE

**Stop here.** Present all final PNGs to the user for approval before proceeding.

- [ ] **Step 1: List all generated icons for review**

```bash
ls -1 assets/icons/immich_*.png assets/icons/upload_*.png assets/icons/error_*.png
```

- [ ] **Step 2: Generate a composite preview sheet**

```bash
magick \
  assets/icons/immich_16.png assets/icons/immich_32.png assets/icons/immich_48.png assets/icons/immich_256.png \
  assets/icons/upload_16.png assets/icons/upload_32.png \
  assets/icons/error_16.png  assets/icons/error_32.png \
  +append -scale 400% /tmp/icon_review_sheet.png
```

Share `/tmp/icon_review_sheet.png` with the user.

- [ ] **Step 3: Wait for explicit user approval before continuing**

Do not proceed to Task 5 until the user confirms the icons are acceptable.

---

## Task 5: Package ICO files

**Files:**
- Create: `assets/icons/immich_app.ico`
- Create: `assets/icons/immich_tray.ico`
- Create: `assets/icons/upload_tray.ico`
- Create: `assets/icons/error_tray.ico`

- [ ] **Step 1: Build exe resource ICO (16, 32, 48, 256)**

```bash
magick \
  assets/icons/immich_16.png \
  assets/icons/immich_32.png \
  assets/icons/immich_48.png \
  assets/icons/immich_256.png \
  assets/icons/immich_app.ico
```

- [ ] **Step 2: Build idle tray ICO (16, 20, 24, 32)**

```bash
magick \
  assets/icons/immich_16.png \
  assets/icons/immich_20.png \
  assets/icons/immich_24.png \
  assets/icons/immich_32.png \
  assets/icons/immich_tray.ico
```

- [ ] **Step 3: Build syncing tray ICO (16, 20, 24, 32)**

```bash
magick \
  assets/icons/upload_16.png \
  assets/icons/upload_20.png \
  assets/icons/upload_24.png \
  assets/icons/upload_32.png \
  assets/icons/upload_tray.ico
```

- [ ] **Step 4: Build error tray ICO (16, 20, 24, 32)**

```bash
magick \
  assets/icons/error_16.png \
  assets/icons/error_20.png \
  assets/icons/error_24.png \
  assets/icons/error_32.png \
  assets/icons/error_tray.ico
```

- [ ] **Step 5: Verify ICO magic bytes**

```bash
for f in assets/icons/immich_app.ico assets/icons/immich_tray.ico assets/icons/upload_tray.ico assets/icons/error_tray.ico; do
  python3 -c "
data = open('$f','rb').read()
assert data[:4] == b'\\x00\\x00\\x01\\x00', f'Bad ICO magic in $f: {data[:4]!r}'
print('OK:', '$f')
"
done
```

Expected: four `OK:` lines.

- [ ] **Step 6: Commit icon source files**

```bash
git add assets/icons/process_icon.py \
        assets/icons/immich_*.png \
        assets/icons/upload_*.png \
        assets/icons/error_*.png \
        assets/icons/immich_app.ico \
        assets/icons/immich_tray.ico \
        assets/icons/upload_tray.ico \
        assets/icons/error_tray.ico
git commit -m "feat: add generated icon assets (immich logo + material state icons)"
```

---

## Task 6: Generate Windows exe resource (`resource.syso`)

**Files:**
- Create: `resource.syso` (repo root)

- [ ] **Step 1: Install rsrc**

```bash
go install github.com/akavel/rsrc@latest
```

Expected: tool installed to `$(go env GOPATH)/bin/rsrc`.

- [ ] **Step 2: Generate `resource.syso`**

Run from the repo root:

```bash
$(go env GOPATH)/bin/rsrc -ico assets/icons/immich_app.ico -o resource.syso
```

Expected: `resource.syso` created in the repo root (the `main` package directory). No output on success.

- [ ] **Step 3: Verify the file exists and is non-empty**

```bash
ls -lh resource.syso
```

Expected: file present, several kilobytes.

- [ ] **Step 4: Commit**

```bash
git add resource.syso
git commit -m "feat: add resource.syso for Windows exe icon"
```

---

## Task 7: Copy embedded icons to `internal/assets/`

**Files:**
- Create: `internal/assets/immich_tray.ico`
- Create: `internal/assets/upload_tray.ico`
- Create: `internal/assets/error_tray.ico`
- Create: `internal/assets/immich_22_linux.png`
- Create: `internal/assets/upload_22_linux.png`
- Create: `internal/assets/error_22_linux.png`

- [ ] **Step 1: Copy ICO files**

```bash
cp assets/icons/immich_tray.ico internal/assets/immich_tray.ico
cp assets/icons/upload_tray.ico internal/assets/upload_tray.ico
cp assets/icons/error_tray.ico  internal/assets/error_tray.ico
```

- [ ] **Step 2: Copy Linux PNG files**

```bash
cp assets/icons/immich_22.png internal/assets/immich_22_linux.png
cp assets/icons/upload_22.png internal/assets/upload_22_linux.png
cp assets/icons/error_22.png  internal/assets/error_22_linux.png
```

- [ ] **Step 3: Verify all six files present**

```bash
ls -lh internal/assets/*.ico internal/assets/*_linux.png
```

Expected: six files listed.

---

## Task 8: Rewrite `internal/assets` Go source

**Files:**
- Modify: `internal/assets/icon.go`
- Modify: `internal/assets/icon_windows.go`
- Modify: `internal/assets/icon_other.go`

- [ ] **Step 1: Write failing tests first**

Replace the contents of `internal/assets/icon_test.go` with:

```go
package assets

import "testing"

func TestIcons_NonEmpty(t *testing.T) {
	icons := map[string]func() []byte{
		"idle":    IconIdle,
		"syncing": IconSyncing,
		"error":   IconError,
	}
	for name, fn := range icons {
		data := fn()
		if len(data) < 4 {
			t.Errorf("%s: too short (%d bytes)", name, len(data))
		}
	}
}

func TestIcons_MagicBytes(t *testing.T) {
	pngMagic := []byte{0x89, 0x50, 0x4e, 0x47}
	icoMagic := []byte{0x00, 0x00, 0x01, 0x00}

	icons := map[string]func() []byte{
		"idle":    IconIdle,
		"syncing": IconSyncing,
		"error":   IconError,
	}
	for name, fn := range icons {
		data := fn()
		isPNG := len(data) >= 4 && data[0] == pngMagic[0] && data[1] == pngMagic[1] && data[2] == pngMagic[2] && data[3] == pngMagic[3]
		isICO := len(data) >= 4 && data[0] == icoMagic[0] && data[1] == icoMagic[1] && data[2] == icoMagic[2] && data[3] == icoMagic[3]
		if !isPNG && !isICO {
			t.Errorf("%s: unexpected magic bytes: %x", name, data[:4])
		}
	}
}
```

**Note:** Steps 3, 4, and 5 below must all be applied before the package will compile. `icon.go` references `idleIcon`/`syncIcon`/`errorIcon` which are only defined after the platform files are rewritten. Apply all three file rewrites, then run the tests in Step 6.

- [ ] **Step 2: Rewrite `internal/assets/icon.go`**

Replace the entire file with:

```go
// Package assets provides embedded tray icons for the application.
package assets

// IconIdle returns the icon bytes for the idle (ready) tray state.
func IconIdle() []byte { return idleIcon }

// IconSyncing returns the icon bytes for the upload-in-progress tray state.
func IconSyncing() []byte { return syncIcon }

// IconError returns the icon bytes for the error tray state.
func IconError() []byte { return errorIcon }
```

- [ ] **Step 3: Rewrite `internal/assets/icon_windows.go`**

Replace the entire file with:

```go
//go:build windows

package assets

import _ "embed"

//go:embed immich_tray.ico
var idleIcon []byte

//go:embed upload_tray.ico
var syncIcon []byte

//go:embed error_tray.ico
var errorIcon []byte
```

- [ ] **Step 4: Rewrite `internal/assets/icon_other.go`**

Replace the entire file with:

```go
//go:build !windows

package assets

import _ "embed"

//go:embed immich_22_linux.png
var idleIcon []byte

//go:embed upload_22_linux.png
var syncIcon []byte

//go:embed error_22_linux.png
var errorIcon []byte
```

- [ ] **Step 5: Run the tests**

```bash
go test ./internal/assets/... -v
```

Expected:

```
=== RUN   TestIcons_NonEmpty
--- PASS: TestIcons_NonEmpty (0.00s)
=== RUN   TestIcons_MagicBytes
--- PASS: TestIcons_MagicBytes (0.00s)
PASS
```

On Linux the magic check will match PNG. On Windows it would match ICO.

- [ ] **Step 6: Run the full test suite**

```bash
go test ./internal/... -v
```

Expected: all tests pass.

- [ ] **Step 7: Cross-compile for Windows to confirm embed resolves**

```bash
GOOS=windows GOARCH=amd64 go build -o screenshot-sync.exe .
```

Expected: `screenshot-sync.exe` built with no errors. (The `.ico` embeds only resolve under `GOOS=windows`; the Linux embeds resolve in the default build.)

- [ ] **Step 8: Commit**

```bash
git add internal/assets/icon.go \
        internal/assets/icon_windows.go \
        internal/assets/icon_other.go \
        internal/assets/icon_test.go \
        internal/assets/immich_tray.ico \
        internal/assets/upload_tray.ico \
        internal/assets/error_tray.ico \
        internal/assets/immich_22_linux.png \
        internal/assets/upload_22_linux.png \
        internal/assets/error_22_linux.png
git commit -m "feat: embed immich logo and material state icons in tray (replaces solid-color squares)"
```

---

## Task 9: Final verification

- [ ] **Step 1: Full test suite**

```bash
go test ./internal/... -v
```

Expected: all tests pass.

- [ ] **Step 2: Native Linux build**

```bash
go build -o screenshot-sync .
ls -lh screenshot-sync
```

Expected: binary built successfully.

- [ ] **Step 3: Windows cross-compile**

```bash
GOOS=windows GOARCH=amd64 go build -o screenshot-sync.exe .
ls -lh screenshot-sync.exe
```

Expected: binary built. This binary will show the Immich logo as its exe icon in Windows Explorer and carry the embedded tray icons.

- [ ] **Step 4: Clean up build artifacts**

```bash
rm -f screenshot-sync screenshot-sync.exe
```

- [ ] **Step 5: Final commit if anything changed**

```bash
git status
```

If clean: done. If there are stray uncommitted files, commit them with an appropriate message.
