.PHONY: build clean

# Cross-compile from Linux to Windows
# Note: github.com/getlantern/systray requires CGO on Windows.
# For cross-compilation, you need either:
#   1. A Windows machine or VM to build natively
#   2. mingw-w64 cross-compiler: sudo apt install gcc-mingw-w64-x86-64
#      Then: CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go build

build:
	GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui -s -w" -o ollama-tray-guard.exe .

build-cgo:
	CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui -s -w" -o ollama-tray-guard.exe .

build-native:
	go build -ldflags="-H windowsgui -s -w" -o ollama-tray-guard.exe .

clean:
	rm -f ollama-tray-guard.exe
