package main

import (
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

type WindowService struct {
	windows   map[string]*application.WebviewWindow
	factories map[string]func() *application.WebviewWindow
}

func NewWindowService() *WindowService {
	return &WindowService{
		windows:   make(map[string]*application.WebviewWindow),
		factories: make(map[string]func() *application.WebviewWindow),
	}
}

// RegisterWindow registers a window factory under an ID
func (s *WindowService) RegisterWindow(id string, factory func() *application.WebviewWindow) {
	win := factory()
	s.windows[id] = win
	s.factories[id] = factory

	// Remove the window from the map if it gets closed
	win.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
		fmt.Println("Window closed:", id)
		delete(s.windows, id)
	})
}

// Show displays the window by ID or recreates it if it was closed
func (s *WindowService) Show(id string) {
	win, ok := s.windows[id]
	if !ok {
		factory, exists := s.factories[id]
		if !exists {
			fmt.Println("No factory registered for window:", id)
			return
		}
		fmt.Println("Recreating window:", id)
		win = factory()
		s.windows[id] = win
		win.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
			fmt.Println("Window closed:", id)
			delete(s.windows, id)
		})
	}

	application.InvokeSync(func() {
		win.Show()
		_ = win.SetAlwaysOnTop(true)
		if id == "main" {
			focusAppWindow()
		}
	})
}

// Hide hides the window by ID
func (s *WindowService) Hide(id string) {
	win, ok := s.windows[id]
	if ok {
		win.Hide()
	}
}

// ToggleVisibility toggles a window's visibility or recreates it if destroyed
func (s *WindowService) ToggleVisibility(id string) {
	win, ok := s.windows[id]
	if !ok {
		s.Show(id)
		return
	}

	isVisible := win.IsVisible()

	if isVisible {
		win.Hide()
	} else {
		s.Show(id)
	}
}

// IsVisible returns whether the window is currently visible
func (s *WindowService) IsVisible(id string) bool {
	win, ok := s.windows[id]
	if !ok {
		return false
	}

	visible := win.IsVisible()
	
	return visible
}
