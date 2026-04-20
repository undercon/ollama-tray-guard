package guard

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/getlantern/systray"
)

type State int

const (
	StateGreen  State = iota // GPU available for Ollama
	StateYellow              // No model loaded
	StateRed                 // GPU busy, model unloaded
)

type modelMenuItem struct {
	name string
	item *systray.MenuItem
}

type Guard struct {
	mu      sync.Mutex
	cfg     Config
	state   State
	stopCh  chan struct{}
	running bool

	mToggle   *systray.MenuItem
	mForce    *systray.MenuItem
	mRefresh  *systray.MenuItem
	mSettings *systray.MenuItem
	mQuit     *systray.MenuItem

	modelItems []modelMenuItem
}

func New() *Guard {
	return &Guard{
		cfg:    LoadConfig(),
		stopCh: make(chan struct{}),
	}
}

func (g *Guard) OnReady() {
	systray.SetIcon(IconGreen)
	systray.SetTitle("Ollama Tray Guard")
	systray.SetTooltip("Ollama Tray Guard - Monitoring VRAM")

	g.mToggle = systray.AddMenuItem("Auto Guard: ON", "Toggle VRAM watchdog")
	if !g.cfg.AutoGuard {
		g.mToggle.SetTitle("Auto Guard: OFF")
	}

	g.mForce = systray.AddMenuItem("Unload All Models", "Unload all Ollama models now")
	systray.AddSeparator()

	g.buildModelMenu()

	systray.AddSeparator()
	g.mRefresh = systray.AddMenuItem("Refresh Model List", "Re-fetch available models from Ollama")
	g.mSettings = systray.AddMenuItem("Settings (edit config)", "Open config file")
	systray.AddSeparator()
	g.mQuit = systray.AddMenuItem("Exit", "Quit application")

	g.stopCh = make(chan struct{})

	if g.cfg.AutoGuard {
		g.startMonitor()
	}

	g.updateModelTitles()
	g.startEventListeners()
}

func (g *Guard) OnExit() {
	if g.running {
		g.stopMonitor()
	}
}

func (g *Guard) buildModelMenu() {
	models, err := OllamaAvailableModels()
	if err != nil || len(models) == 0 {
		item := systray.AddMenuItem("(no models — start Ollama first)", "")
		item.Disable()
		g.modelItems = []modelMenuItem{{name: "", item: item}}
		return
	}

	g.modelItems = make([]modelMenuItem, len(models))
	for i, name := range models {
		item := systray.AddMenuItem("Load: "+name, "Load "+name+" into VRAM")
		g.modelItems[i] = modelMenuItem{name: name, item: item}
	}
}

func (g *Guard) startEventListeners() {
	go func() {
		for range g.mToggle.ClickedCh {
			g.mu.Lock()
			g.cfg.AutoGuard = !g.cfg.AutoGuard
			if g.cfg.AutoGuard {
				g.mToggle.SetTitle("Auto Guard: ON")
				g.startMonitor()
			} else {
				g.mToggle.SetTitle("Auto Guard: OFF")
				g.stopMonitor()
				g.setState(StateGreen)
			}
			_ = SaveConfig(g.cfg)
			g.mu.Unlock()
		}
	}()

	go func() {
		for range g.mForce.ClickedCh {
			go func() {
				unloaded, err := UnloadAllModels()
				if err != nil {
					log.Printf("Force clear error: %v", err)
				} else if len(unloaded) > 0 {
					Toast("Ollama Tray Guard", fmt.Sprintf("Unloaded: %v", unloaded))
				} else {
					Toast("Ollama Tray Guard", "No models were loaded")
				}
			}()
		}
	}()

	go func() {
		for range g.mRefresh.ClickedCh {
			Toast("Ollama Tray Guard", "Restart the app to refresh the model list")
		}
	}()

	go func() {
		for range g.mSettings.ClickedCh {
			_ = SaveConfig(g.cfg)
			_ = noWindow("cmd", "/c", "start", "", configPath()).Start()
		}
	}()

	go func() {
		for range g.mQuit.ClickedCh {
			systray.Quit()
		}
	}()

	g.startModelTitleUpdater()

	for _, mi := range g.modelItems {
		if mi.name == "" {
			continue
		}
		mi := mi
		go func() {
			for range mi.item.ClickedCh {
				name := mi.name
				go func() {
					Toast("Ollama Tray Guard", fmt.Sprintf("Loading %s...", name))
					if err := LoadModel(name); err != nil {
						log.Printf("Load model error: %v", err)
						Toast("Ollama Tray Guard", fmt.Sprintf("Failed to load %s: %v", name, err))
					} else {
						Toast("Ollama Tray Guard", fmt.Sprintf("%s loaded", name))
					}
				}()
			}
		}()
	}
}

func (g *Guard) startModelTitleUpdater() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-g.stopCh:
				return
			case <-ticker.C:
				g.updateModelTitles()
			}
		}
	}()
}

func (g *Guard) updateModelTitles() {
	loaded, err := OllamaRunningModels()
	if err != nil {
		return
	}
	loadedMap := make(map[string]bool)
	for _, name := range loaded {
		loadedMap[name] = true
	}
	for _, mi := range g.modelItems {
		if mi.name == "" {
			continue
		}
		if loadedMap[mi.name] {
			mi.item.SetTitle("[*] " + mi.name)
		} else {
			mi.item.SetTitle("Load: " + mi.name)
		}
	}
}

func (g *Guard) startMonitor() {
	if g.running {
		return
	}
	g.running = true
	g.stopCh = make(chan struct{})
	go g.monitorLoop()
}

func (g *Guard) stopMonitor() {
	if !g.running {
		return
	}
	g.running = false
	close(g.stopCh)
}

func (g *Guard) monitorLoop() {
	ticker := time.NewTicker(time.Duration(g.cfg.PollIntervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-g.stopCh:
			return
		case <-ticker.C:
			g.check()
		}
	}
}

func (g *Guard) check() {
	nonOllamaVRAM, err := NonOllamaVRAM()
	if err != nil {
		log.Printf("VRAM check error: %v", err)
		return
	}

	thresholdMB := g.cfg.VRAMThresholdGB * 1024

	if nonOllamaVRAM >= thresholdMB {
		if g.state != StateRed {
			unloaded, _ := UnloadAllModels()
			if len(unloaded) > 0 {
				Toast("GPU busy — Ollama unloaded",
					fmt.Sprintf("%.1f GB non-Ollama VRAM in use", nonOllamaVRAM/1024))
			}
			g.setState(StateRed)
		}
	} else {
		if g.state == StateRed {
			Toast("GPU available — Ollama ready",
				"VRAM pressure dropped. Models will reload on next request.")
		}
		models, _ := OllamaRunningModels()
		if len(models) > 0 {
			g.setState(StateGreen)
		} else {
			g.setState(StateYellow)
		}
	}
}

func (g *Guard) setState(s State) {
	if g.state == s {
		return
	}
	g.state = s
	switch s {
	case StateGreen:
		systray.SetIcon(IconGreen)
		systray.SetTooltip("Ollama Tray Guard - GPU available")
	case StateYellow:
		systray.SetIcon(IconYellow)
		systray.SetTooltip("Ollama Tray Guard - No model loaded")
	case StateRed:
		systray.SetIcon(IconRed)
		systray.SetTooltip("Ollama Tray Guard - GPU busy, model unloaded")
	}
}
