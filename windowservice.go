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
	s.factories[id] = factory
}

// internal: ensures the window exists, creates if needed
func (s *WindowService) getOrCreateWindow(id string) (*application.WebviewWindow, bool) {
	win, exists := s.windows[id]
	if exists {
		return win, true
	}

	factory, ok := s.factories[id]
	if !ok {
		fmt.Println("❌ No factory registered for window:", id)
		return nil, false
	}

	win = factory()
	s.windows[id] = win

	// Cleanup when closed
	win.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
		fmt.Println("🪟 Window closed:", id)
		delete(s.windows, id)
	})

	return win, true
}

// Show displays the window by ID, creating it if necessary
func (s *WindowService) Show(id string) {
	win, ok := s.getOrCreateWindow(id)
	if !ok {
		return
	}

	application.InvokeAsync(func() {
		win.Show()
		win.Focus()
	})
}

// Hide hides the window by ID
func (s *WindowService) Hide(id string) {
	if win, ok := s.windows[id]; ok {
		application.InvokeAsync(func() {
			win.Hide()
		})
	}
}

// ToggleVisibility shows or hides the window
func (s *WindowService) ToggleVisibility(id string) {
	win, ok := s.getOrCreateWindow(id)
	if !ok {
		return
	}

	application.InvokeAsync(func() {
		if win.IsVisible() {
			win.Hide()
		} else {
			println("Showing window:", id)
			win.Show()
			win.Focus()
		}
	})
}

// IsVisible returns whether a window is currently visible
func (s *WindowService) IsVisible(id string) bool {
	if win, ok := s.windows[id]; ok {
		return win.IsVisible()
	}
	return false
}
