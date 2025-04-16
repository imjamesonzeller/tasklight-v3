package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

type WindowService struct {
	windows map[string]*application.WebviewWindow
	state   map[string]bool
}

func NewWindowService() *WindowService {
	return &WindowService{
		windows: make(map[string]*application.WebviewWindow),
		state:   make(map[string]bool),
	}
}

// RegisterWindow Register a window by ID
func (s *WindowService) RegisterWindow(id string, win *application.WebviewWindow) {
	s.windows[id] = win
	s.state[id] = true
}

// ToggleVisibility Toggle visibility of a window by ID
func (s *WindowService) ToggleVisibility(id string) {
	win, ok := s.windows[id]
	if !ok {
		return // silently ignore for now
	}

	isVisible := s.state[id]
	if isVisible {
		win.Hide()
	} else {
		win.Show()
		win.SetAlwaysOnTop(true)
		focusAppWindow()
	}
	s.state[id] = !isVisible
}
