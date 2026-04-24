package guard

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	VRAMThresholdGB float64 `json:"vram_threshold_gb"`
	PollIntervalSec int     `json:"poll_interval_sec"`
	AutoGuard       bool   `json:"auto_guard"`
	StopOllamaMode  string `json:"stop_ollama_mode"` // "unload", "stop", "both"
}

func DefaultConfig() Config {
	return Config{
		VRAMThresholdGB: 4.0,
		PollIntervalSec: 5,
		AutoGuard:       true,
		StopOllamaMode:  "both",
	}
}

func configPath() string {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		appdata = "."
	}
	return filepath.Join(appdata, "ollama-tray-guard", "config.json")
}

func LoadConfig() Config {
	cfg := DefaultConfig()
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, &cfg)
	return cfg
}

func SaveConfig(cfg Config) error {
	p := configPath()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}
