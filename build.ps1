# Native Windows build script (no make required)
# Usage: .\build.ps1

$ErrorActionPreference = "Stop"

Write-Host "Building screenshot-sync.exe..."
go build -o screenshot-sync.exe .

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful: screenshot-sync.exe"
} else {
    Write-Host "Build failed." -ForegroundColor Red
    exit $LASTEXITCODE
}
