# GitHub Actions Release Workflow — Design Spec

**Issue:** [#12](https://github.com/laforcem/immich-mc-screencap-sync/issues/12)  
**Date:** 2026-04-14

---

## Problem

The README links to the GitHub Releases page for binary downloads, but no release pipeline exists. Users must compile from source.

---

## Goal

When a release is published on GitHub, automatically build binaries for Linux and Windows and attach them to that release.

---

## Trigger

`on: release: [published]`

Tag pushes alone do not trigger the workflow. Only publishing a release in the GitHub UI starts a build.

---

## Approach

A single workflow file at `.github/workflows/release.yml`. One job runs on `ubuntu-latest` and performs these steps in order:

1. **Checkout** the repository at the release tag.
2. **Set up Go** at the version specified in `go.mod` (`1.25.5`).
3. **Run tests** — `go test ./internal/... -v`. The job fails here if any test fails.
4. **Build Linux binary** — `go build -o screenshot-sync .`
5. **Build Windows binary** — `GOOS=windows GOARCH=amd64 go build -o screenshot-sync.exe .`
6. **Upload artifacts** to the release using `softprops/action-gh-release`. Uploads `screenshot-sync` and `screenshot-sync.exe`.

Authentication uses the automatically-provided `GITHUB_TOKEN`. No extra secrets are required.

---

## Artifacts

| File | Platform |
|------|----------|
| `screenshot-sync` | Linux (amd64) |
| `screenshot-sync.exe` | Windows (amd64) |

---

## Out of Scope

- macOS builds (requires a native runner due to CGo in `getlantern/systray`; tracked in [#13](https://github.com/laforcem/immich-mc-screencap-sync/issues/13))
- Checksum files, archive formats, or changelog generation (consider goreleaser if these become desirable, also tracked in [#13](https://github.com/laforcem/immich-mc-screencap-sync/issues/13))
- Manual workflow dispatch trigger
