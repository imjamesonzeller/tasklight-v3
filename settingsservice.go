package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/keybase/go-keychain"
	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.design/x/hotkey"
)

const (
	keychainService   = "Tasklight"
	keychainNotionKey = "NotionSecret"
	keychainOpenAIKey = "OpenAIAPISecret"
)

type SettingsService struct {
	app      *application.App
	Settings ApplicationSettings
}

func NewSettingsService() *SettingsService {
	return &SettingsService{}
}

func (s *SettingsService) SetApp(app *application.App) {
	s.app = app
}

type ApplicationSettings struct {
	NotionDBID      string       `json:"notion_db_id"`
	UseOpenAI       bool         `json:"use_open_ai"`
	Theme           string       `json:"theme"`
	LaunchOnStartup bool         `json:"launch_on_startup"`
	Hotkey          HotkeyConfig `json:"hotkey"`
	HasNotionSecret bool         `json:"has_notion_secret"`
	HasOpenAIAPIKey bool         `json:"has_openai_key"`

	// Allow deserialization but don't force it to be included in responses
	NotionSecret string `json:"notion_secret,omitempty"`
	OpenAIAPIKey string `json:"openai_api_key,omitempty"`
}

type FrontendSettings struct {
	NotionDBID      string `json:"notion_db_id"`
	UseOpenAI       bool   `json:"use_open_ai"`
	Theme           string `json:"theme"`
	LaunchOnStartup bool   `json:"launch_on_startup"`
	Hotkey          string `json:"hotkey"`
	HasNotionSecret bool   `json:"has_notion_secret"`
	HasOpenAIAPIKey bool   `json:"has_openai_key"`
}

type HotkeyConfig struct {
	Modifiers []hotkey.Modifier `json:"Modifiers"`
	Key       hotkey.Key        `json:"Key"`
}

// MarshalJSON converts HotkeyConfig to a "ctrl+space" style string for JSON
func (h *HotkeyConfig) MarshalJSON() ([]byte, error) {
	var parts []string
	for _, mod := range h.Modifiers {
		if modStr, ok := revModMap[mod]; ok {
			parts = append(parts, modStr)
		}
	}
	if keyStr, ok := revKeyMap[h.Key]; ok {
		parts = append(parts, keyStr)
	}
	return json.Marshal(strings.Join(parts, "+"))
}

// UnmarshalJSON parses "ctrl+space" style strings into a HotkeyConfig
//func (h *HotkeyConfig) UnmarshalJSON(data []byte) error {
//	var input string
//	if err := json.Unmarshal(data, &input); err != nil {
//		return err
//	}
//
//	parts := strings.Split(strings.ToLower(input), "+")
//	var mods []hotkey.Modifier
//	foundKey := false
//
//	for _, part := range parts {
//		if mod, ok := modMap[part]; ok {
//			mods = append(mods, mod)
//		} else if k, ok := keyMap[part]; ok {
//			h.Key = k
//			foundKey = true
//		} else {
//			return fmt.Errorf("unknown hotkey part: %s", part)
//		}
//	}
//
//	if !foundKey {
//		return fmt.Errorf("no valid key found in hotkey string")
//	}
//
//	h.Modifiers = mods
//	return nil
//}

// Save to Keychain
func saveSecret(label, value string) error {
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(keychainService)
	item.SetAccount(label)
	item.SetLabel(label)
	item.SetData([]byte(value))
	item.SetAccessible(keychain.AccessibleWhenUnlocked)
	item.SetSynchronizable(keychain.SynchronizableNo)
	return keychain.AddItem(item)
}

// Load from Keychain
func loadSecret(label string) (string, error) {
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(keychainService)
	query.SetAccount(label)
	query.SetReturnData(true)
	query.SetMatchLimit(keychain.MatchLimitOne)

	results, err := keychain.QueryItem(query)
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "", fmt.Errorf("secret not found for %s", label)
	}
	return string(results[0].Data), nil
}

// Update keychain entry if already present
func updateSecret(label, value string) error {
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(keychainService)
	item.SetAccount(label)

	_ = keychain.DeleteItem(item) // delete if exists
	return saveSecret(label, value)
}

func (s *SettingsService) LoadSettings() {
	data, err := os.ReadFile("settings.json")
	if err == nil {
		err := json.Unmarshal(data, &s.Settings)
		if err != nil {
			return
		}
	}

	s.Settings.NotionSecret, _ = loadSecret(keychainNotionKey)
	s.Settings.OpenAIAPIKey, _ = loadSecret(keychainOpenAIKey)
}

func (s *SettingsService) SaveSettings() {
	data, err := json.MarshalIndent(s.Settings, "", "  ")
	if err == nil {
		_ = os.WriteFile("settings.json", data, 0644)
	}

	_ = updateSecret(keychainNotionKey, s.Settings.NotionSecret)
	_ = updateSecret(keychainOpenAIKey, s.Settings.OpenAIAPIKey)
}

func parseHotkeyString(input string) (HotkeyConfig, error) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(input)), "+")
	var mods []hotkey.Modifier
	var key hotkey.Key
	keyFound := false

	for _, part := range parts {
		if mod, ok := modMap[part]; ok {
			mods = append(mods, mod)
		} else if k, ok := keyMap[part]; ok {
			key = k
			keyFound = true
		} else {
			return HotkeyConfig{}, fmt.Errorf("unknown hotkey part: %s", part)
		}
	}

	if !keyFound {
		return HotkeyConfig{}, fmt.Errorf("no valid key in hotkey string")
	}

	return HotkeyConfig{
		Modifiers: mods,
		Key:       key,
	}, nil
}

// ----- FRONTEND FUNCTIONS -----

func (s *SettingsService) UpdateSettings(raw map[string]interface{}) error {
	// Pull hotkey string manually
	rawHotkey, ok := raw["hotkey"].(string)
	if !ok {
		return fmt.Errorf("hotkey must be a string")
	}

	hotkeyConfig, err := parseHotkeyString(rawHotkey)
	if err != nil {
		return fmt.Errorf("invalid hotkey: %w", err)
	}

	// Convert the rest of raw -> ApplicationSettings
	var newSettings ApplicationSettings
	data, _ := json.Marshal(raw)
	_ = json.Unmarshal(data, &newSettings)

	newSettings.Hotkey = hotkeyConfig
	s.Settings = newSettings

	return s.SaveSettingsInternal()
}

// Internal version that only writes to file
func (s *SettingsService) SaveSettingsInternal() error {
	data, err := json.MarshalIndent(s.Settings, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println("Writing settings.json:", string(data)) // <== ADD THIS
	return os.WriteFile("settings.json", data, 0644)
}

func (s *SettingsService) GetSettings() (FrontendSettings, error) {
	settings := s.Settings

	var frontend FrontendSettings

	// Copy shared fields
	frontend.NotionDBID = settings.NotionDBID
	frontend.UseOpenAI = settings.UseOpenAI
	frontend.Theme = settings.Theme
	frontend.LaunchOnStartup = settings.LaunchOnStartup

	// Convert hotkey struct to string
	hotkeyJSON, err := settings.Hotkey.MarshalJSON()
	if err != nil {
		frontend.Hotkey = "invalid hotkey"
	} else {
		_ = json.Unmarshal(hotkeyJSON, &frontend.Hotkey)
	}

	// Check keychain for secrets
	if notion, err := loadSecret(keychainNotionKey); err == nil && notion != "" {
		frontend.HasNotionSecret = true
	}
	if openai, err := loadSecret(keychainOpenAIKey); err == nil && openai != "" {
		frontend.HasOpenAIAPIKey = true
	}

	return frontend, nil
}
