package main

import (
	"fmt"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

type WindowService struct {
	mu        sync.RWMutex
	windows   map[string]*application.WebviewWindow
	factories map[string]func() *application.WebviewWindow
	visible   map[string]bool
}

func NewWindowService() *WindowService {
	return &WindowService{
		windows:   make(map[string]*application.WebviewWindow),
		factories: make(map[string]func() *application.WebviewWindow),
		visible:   make(map[string]bool),
	}
}

// RegisterWindow registers a window factory under an ID
func (s *WindowService) RegisterWindow(id string, factory func() *application.WebviewWindow) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.factories[id] = factory
}

// internal: ensures the window exists, creates if needed
func (s *WindowService) getOrCreateWindow(id string) (*application.WebviewWindow, bool) {
	s.mu.RLock()
	win, exists := s.windows[id]
	if exists {
		s.mu.RUnlock()
		return win, true
	}

	factory, ok := s.factories[id]
	s.mu.RUnlock()
	if !ok {
		fmt.Println("❌ No factory registered for window:", id)
		return nil, false
	}

	win = factory()
	s.mu.Lock()
	if existing, exists := s.windows[id]; exists {
		s.mu.Unlock()
		return existing, true
	}
	s.windows[id] = win
	s.visible[id] = false
	s.mu.Unlock()

	// Cleanup when closed
	win.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
		fmt.Println("🪟 Window closed:", id)
		s.mu.Lock()
		delete(s.windows, id)
		delete(s.visible, id)
		s.mu.Unlock()
	})

	return win, true
}

// Show displays the window by ID, creating it if necessary
func (s *WindowService) Show(id string) {
	win, ok := s.getOrCreateWindow(id)
	if !ok {
		return
	}

	s.mu.Lock()
	s.visible[id] = true
	s.mu.Unlock()
	application.InvokeAsync(func() {
		win.Show()
		win.Focus()
	})
}

// Hide hides the window by ID
func (s *WindowService) Hide(id string) {
	s.mu.Lock()
	win, ok := s.windows[id]
	if ok {
		s.visible[id] = false
	}
	s.mu.Unlock()

	if !ok {
		return
	}

	application.InvokeAsync(func() {
		win.Hide()
	})
}

// ToggleVisibility shows or hides the window
func (s *WindowService) ToggleVisibility(id string) {
	win, ok := s.getOrCreateWindow(id)
	if !ok {
		return
	}

	s.mu.Lock()
	if s.visible[id] {
		s.visible[id] = false
		s.mu.Unlock()
		application.InvokeAsync(func() {
			win.Hide()
		})
		return
	}

	s.visible[id] = true
	s.mu.Unlock()
	application.InvokeAsync(func() {
		println("Showing window:", id)
		win.Show()
		win.Focus()
	})
}

// IsVisible returns whether a window is currently visible
func (s *WindowService) IsVisible(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.visible[id]
}
