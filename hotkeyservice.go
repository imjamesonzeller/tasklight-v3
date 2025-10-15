package main

import (
	"fmt"
	"github.com/imjamesonzeller/tasklight-v3/settingsservice"
	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.design/x/hotkey"
	"time"
)

type HotkeyService struct {
	app             *application.App
	windowService   *WindowService
	settingsService *settingsservice.SettingsService

	currentHotkey *hotkey.Hotkey
	listenerStop  chan struct{}
}

func NewHotkeyService(windowService *WindowService, settingsService *settingsservice.SettingsService) *HotkeyService {
	return &HotkeyService{
		windowService:   windowService,
		settingsService: settingsService,
	}
}

func (s *HotkeyService) SetApp(app *application.App) {
	s.app = app

	app.OnEvent("Backend:SettingsUpdated", func(_ *application.CustomEvent) {
		fmt.Println("Settings updated event received - updating hotkey")
		_ = s.UpdateHotkey()
	})
}

func (s *HotkeyService) StartHotkeyListener() {
	fmt.Println("🔁 Starting hotkey listener from settings")
	if err := s.UpdateHotkey(); err != nil {
		fmt.Println("❌ Failed to start hotkey listener:", err)
	}
}

// internal: Registers a hotkey and starts its listener goroutine
func (s *HotkeyService) RegisterHotkey(hk *hotkey.Hotkey) error {
	if err := hk.Register(); err != nil {
		return err
	}

	s.currentHotkey = hk

	s.startListener()

	fmt.Println("✅ Hotkey registered:", s.settingsService.AppSettings.Hotkey)
	return nil
}

func (s *HotkeyService) UpdateHotkey() error {
	// Stop the listener before unregister the hotkey
	// If this wasn't done before unregistering then there was a NASTY race condition where it would
	// have a phantom .Keydown() for some reason
	s.stopListener()

	if s.currentHotkey != nil {
		_ = s.currentHotkey.Unregister()
		s.currentHotkey = nil
	}

	// Create new hotkey from saved config
	modifiers := s.settingsService.AppSettings.Hotkey.Modifiers
	key := s.settingsService.AppSettings.Hotkey.Key

	newHk := hotkey.New(modifiers, key)
	return s.RegisterHotkey(newHk)
}

// stopListener stops the existing listener goroutine if it exists
func (s *HotkeyService) stopListener() {
	if stop := s.listenerStop; stop != nil {
		close(stop)
		s.listenerStop = nil
	}
}

func (s *HotkeyService) startListener() {
	hk := s.currentHotkey
	if hk == nil {
		return
	}

	stop := make(chan struct{})
	s.listenerStop = stop
	keydown := hk.Keydown()

	go func(stop <-chan struct{}, keydown <-chan hotkey.Event) {
		for {
			select {
			case <-keydown:
				s.windowService.ToggleVisibility("main")
				s.app.EmitEvent("Backend:GlobalHotkeyEvent", time.Now().String())

			case <-stop:
				return
			}
		}
	}(stop, keydown)
}

func (s *HotkeyService) PauseHotkey() {
	fmt.Println("⏸️ Pausing hotkey listener")
	s.stopListener()
	if s.currentHotkey != nil {
		_ = s.currentHotkey.Unregister()
	}
}

func (s *HotkeyService) ResumeHotkey() {
	fmt.Println("▶️ Resuming hotkey listener")
	if s.currentHotkey != nil {
		_ = s.currentHotkey.Register()
		s.startListener()
	}
}
