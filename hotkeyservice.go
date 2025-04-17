package main

import (
	"fmt"
	"golang.design/x/hotkey"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type HotkeyService struct {
	app           *application.App
	windowService *WindowService
}

func NewHotkeyService(windowService *WindowService) *HotkeyService {
	return &HotkeyService{
		windowService: windowService,
	}
}

func (s *HotkeyService) SetApp(app *application.App) {
	s.app = app
}

func (s *HotkeyService) StartHotkeyListener() {
	fmt.Println("Hotkey Startup Called")
	// Register hotkeys
	// visHk is the hotkey for toggling main window visibility
	visHk := hotkey.New([]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeySpace)
	if err := visHk.Register(); err != nil {
		fmt.Println("❌ Failed to register hotkey:", err)
		return
	}

	fmt.Println("✅ Hotkey registered (Ctrl+Shift+Space)")

	go func() {
		for {
			select {
			case <-visHk.Keydown():
				window := "main"
				s.windowService.ToggleVisibility(window)
				s.app.EmitEvent("Backend:GlobalHotkeyEvent", time.Now().String())
			}
		}
	}()
}
