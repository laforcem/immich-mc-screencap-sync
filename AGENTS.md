# Dev Workflow

## Running from WSL

WSL interop handles `.exe` execution natively — no extra tools needed.

| Command | What it does |
|---|---|
| `make run-sync` | Build Windows binary, run one-shot sync |
| `make run-daemon` | Build Windows binary, run headless watcher |
| `make run-tray` | Build Windows binary, run with system tray icon |

stdout/stderr flow back to the WSL terminal for all three modes. The tray icon appears in the Windows taskbar; quit via the tray menu.

## Superpowers Docs

`docs/superpowers/` is gitignored — specs and plans written there do not need to be committed.

## Config

`config.toml` lives in the repo root (gitignored). Use Windows-format paths for `prism_dir` and any other paths — the binary runs as a Windows process.
