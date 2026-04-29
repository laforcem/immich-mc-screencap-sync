# GUI Tray Auto-Launch Design

**Issue:** [#7 — Run as tray in Windows when running executable via GUI](https://github.com/laforcem/immich-mc-screencap-sync/issues/7)

## Problem

When a user double-clicks `screenshot-sync.exe` in Windows Explorer, cobra's built-in `mousetrap` integration detects the Explorer launch and prints "This is a command line tool. You need to open cmd.exe and run it from there." The app then waits for a keypress before exiting. This is unhelpful — the right behavior is to launch the system tray UI.

## Solution

Use `mousetrap.StartedByExplorer()` to detect the Explorer launch before cobra parses arguments, then redirect to the `tray` subcommand automatically.

## Change

**File:** `cmd/root.go` — `Execute()` function

1. Set `cobra.MousetrapHelpText = ""` to disable cobra's built-in Explorer-launch warning.
2. Call `mousetrap.StartedByExplorer()`. If true, prepend `"tray"` to `os.Args` so cobra routes to the tray command.

`mousetrap` (`github.com/inconshreveable/mousetrap`) is already an indirect dependency via cobra and returns `false` on non-Windows platforms, so no build tags or platform guards are needed.

## Behavior

| Launch method | Before | After |
|---|---|---|
| Double-click in Explorer | "This is a command line tool..." pause | Launches system tray |
| `screenshot-sync` (no args) | cobra help text | cobra help text (unchanged) |
| `screenshot-sync tray` | Launches system tray | Launches system tray (unchanged) |
| `screenshot-sync sync` | Runs one-shot sync | Runs one-shot sync (unchanged) |
| `screenshot-sync daemon` | Runs headless daemon | Runs headless daemon (unchanged) |

## Testing

Manual test on Windows: double-click `screenshot-sync.exe` — tray icon should appear in the system tray without a console window.
