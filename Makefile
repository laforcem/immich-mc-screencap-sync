.PHONY: build build-windows test clean

# Native build (Linux)
build:
	go build -o screenshot-sync .

# Cross-compile for Windows (requires: apt install gcc-mingw-w64-x86-64)
build-windows:
	CGO_ENABLED=1 \
	CC=x86_64-w64-mingw32-gcc \
	GOOS=windows \
	GOARCH=amd64 \
	go build -o screenshot-sync.exe .

test:
	go test ./internal/... -v

clean:
	rm -f screenshot-sync screenshot-sync.exe
