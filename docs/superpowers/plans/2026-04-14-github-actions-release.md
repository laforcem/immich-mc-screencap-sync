# GitHub Actions Release Workflow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a GitHub Actions workflow that builds Linux and Windows binaries and attaches them to a GitHub Release when that release is published.

**Architecture:** A single workflow file triggered by `release: [published]`. One Ubuntu job checks out the repo, runs tests, cross-compiles both binaries, and uploads them as release assets using `softprops/action-gh-release`.

**Tech Stack:** GitHub Actions, `actions/checkout@v4`, `actions/setup-go@v5`, `softprops/action-gh-release@v2`

---

### Task 1: Create the release workflow

**Files:**
- Create: `.github/workflows/release.yml`

- [ ] **Step 1: Create the workflows directory and file**

```bash
mkdir -p .github/workflows
```

- [ ] **Step 2: Write the workflow**

Create `.github/workflows/release.yml` with this exact content:

```yaml
name: Release

on:
  release:
    types: [published]

permissions:
  contents: write

jobs:
  build-and-upload:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Run tests
        run: go test ./internal/... -v

      - name: Build Linux binary
        run: go build -o screenshot-sync .

      - name: Build Windows binary
        run: GOOS=windows GOARCH=amd64 go build -o screenshot-sync.exe .

      - name: Upload release artifacts
        uses: softprops/action-gh-release@v2
        with:
          files: |
            screenshot-sync
            screenshot-sync.exe
```

**Notes on key choices:**
- `permissions: contents: write` is required for `softprops/action-gh-release` to upload assets to the release. Without it, the upload step fails with a 403.
- `go-version-file: go.mod` reads the Go version directly from `go.mod` rather than hardcoding it, so the workflow stays in sync automatically.
- When triggered by `release: [published]`, `softprops/action-gh-release` infers the target release from `GITHUB_REF` — no tag input needed.

- [ ] **Step 3: Validate YAML syntax**

Run:
```bash
python3 -c "import yaml, sys; yaml.safe_load(open('.github/workflows/release.yml'))" && echo "YAML valid"
```

Expected output:
```
YAML valid
```

If Python is not available, use:
```bash
cat .github/workflows/release.yml
```
and visually confirm indentation is consistent (2-space throughout, no tabs).

- [ ] **Step 4: Verify the workflow will not trigger on a bare tag push**

Confirm the `on:` block contains only `release: types: [published]` and no `push: tags:` entry. A bare tag push must not trigger this workflow.

- [ ] **Step 5: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: add release workflow to build and upload artifacts (closes #12)"
```
