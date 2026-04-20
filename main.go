package main

import (
	"github.com/decon/ollama-tray-guard/guard"
	"github.com/getlantern/systray"
)

func main() {
	g := guard.New()
	systray.Run(g.OnReady, g.OnExit)
}
