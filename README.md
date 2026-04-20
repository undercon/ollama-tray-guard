# Ollama Tray Guard

A lightweight Windows system tray utility that monitors GPU VRAM usage and automatically unloads Ollama models when something else needs the GPU (gaming, VR, etc).

## Features

- **VRAM Monitoring**: Polls nvidia-smi to detect non-Ollama GPU usage
- **Auto Unload**: Unloads Ollama models when VRAM pressure exceeds threshold
- **Auto Reload**: Ollama naturally reloads models on next API request when GPU is free
- **Tray Icon**: Color-coded status (Green/Yellow/Red)
- **Toast Notifications**: Windows notifications on state changes
- **Force Clear**: Panic button to immediately free GPU for VR/gaming

## Tray Icon States

- 🟢 **Green** — Auto Guard active, GPU available for Ollama
- 🟡 **Yellow** — Auto Guard active, no Ollama model loaded
- 🔴 **Red** — GPU busy, model was unloaded due to VRAM pressure

## Configuration

Config file: `%APPDATA%\ollama-tray-guard\config.json`

```json
{
  "vram_threshold_gb": 4.0,
  "poll_interval_sec": 5,
  "auto_guard": true
}
```

- `vram_threshold_gb`: Non-Ollama VRAM usage (in GB) that triggers model unload (default: 4)
- `poll_interval_sec`: How often to check nvidia-smi (default: 5 seconds)
- `auto_guard`: Whether monitoring starts automatically (default: true)

## Building

### On Windows (native, recommended)

```
go build -ldflags="-H windowsgui -s -w" -o ollama-tray-guard.exe .
```

### Cross-compile from Linux

Requires mingw-w64:
```
sudo apt install gcc-mingw-w64-x86-64
make build
```

Note: `github.com/getlantern/systray` requires CGO for Windows (it uses Win32 APIs). Cross-compilation needs the mingw-w64 toolchain.

## Requirements

- Windows 10/11
- NVIDIA GPU with nvidia-smi in PATH
- Ollama running on localhost:11434

## Usage

1. Build the exe
2. Run `ollama-tray-guard.exe`
3. Icon appears in system tray
4. Right-click for menu options
5. Launch your game/VR — models auto-unload when GPU gets busy
6. When done gaming, Ollama reloads on next request automatically
