.PHONY: build build-windows run-sync run-daemon run-tray test clean

# Detect OS for output binary name
ifeq ($(OS),Windows_NT)
    BINARY = screenshot-sync.exe
else
    BINARY = screenshot-sync
endif

# Native build (Linux/macOS/Windows via Git Bash or MSYS2)
build:
	go build -o $(BINARY) .

# Cross-compile for Windows from Linux
build-windows:
	GOOS=windows \
	GOARCH=amd64 \
	go build -o screenshot-sync.exe .

run-sync: build-windows
	./screenshot-sync.exe sync

run-daemon: build-windows
	./screenshot-sync.exe daemon

run-tray: build-windows
	./screenshot-sync.exe tray

test:
	go test ./internal/... -v

clean:
	rm -f screenshot-sync screenshot-sync.exe
