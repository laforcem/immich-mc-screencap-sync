@echo off
REM Native Windows build script (no make required)
REM Usage: build.bat

echo Building screenshot-sync.exe...
go build -o screenshot-sync.exe .

if %ERRORLEVEL% EQU 0 (
    echo Build successful: screenshot-sync.exe
) else (
    echo Build failed.
    exit /b %ERRORLEVEL%
)
