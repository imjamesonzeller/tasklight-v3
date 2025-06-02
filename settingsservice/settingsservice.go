package settingsservice

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/keybase/go-keychain"
	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.design/x/hotkey"
)

// ====== Constants ======
const (
	keychainService     = "com.tasklight.app"
	keychainNotionToken = "NotionAccessToken"
	keychainOpenAIKey   = "OpenAIAPISecret"
	settingsFilePath    = "settings.json"
)

// ====== Structs ======

type SettingsService struct {
	App               *application.App
	AppSettings       ApplicationSettings
	FrontendOverrides FrontendSettings
}

type ApplicationSettings struct {
	NotionDBID      string       `json:"notion_db_id"`
	UseOpenAI       bool         `json:"use_open_ai"`
	Theme           string       `json:"theme"`
	LaunchOnStartup bool         `json:"launch_on_startup"`
	Hotkey          hotkeyConfig `json:"hotkey"`

	// ====== Date Property Shiznit ======
	DatePropertyID   string `json:"date_property_id"`
	DatePropertyName string `json:"date_property_name"`

	// ====== Secrets ======
	NotionAccessToken string `json:"notion_access_token,omitempty"`
	OpenAIAPIKey      string `json:"openai_api_key,omitempty"`
}

type FrontendSettings struct {
	NotionDBID         string `json:"notion_db_id"`
	UseOpenAI          bool   `json:"use_open_ai"`
	Theme              string `json:"theme"`
	LaunchOnStartup    bool   `json:"launch_on_startup"`
	Hotkey             string `json:"hotkey"`
	HasConnectedNotion bool   `json:"has_notion_secret"`
	HasOpenAIAPIKey    bool   `json:"has_openai_key"`

	DatePropertyID   string `json:"date_property_id"`
	DatePropertyName string `json:"date_property_name"`
}

// ====== Initializers ======

func NewSettingsService() *SettingsService {
	service := &SettingsService{}
	service.LoadSettings()
	return service
}

func (s *SettingsService) SetApp(app *application.App) {
	s.App = app
}

// ====== Hotkey ======

type hotkeyConfig struct {
	Modifiers []hotkey.Modifier `json:"Modifiers"`
	Key       hotkey.Key        `json:"Key"`
}

func (h *hotkeyConfig) MarshalJSON() ([]byte, error) {
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

func parseHotkeyString(input string) (hotkeyConfig, error) {
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
			return hotkeyConfig{}, fmt.Errorf("unknown hotkey part: %s", part)
		}
	}

	if !keyFound {
		return hotkeyConfig{}, fmt.Errorf("no valid key in hotkey string")
	}

	return hotkeyConfig{
		Modifiers: mods,
		Key:       key,
	}, nil
}

// ====== Keychain Helpers ======
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

func LoadSecret(label string) (string, error) {
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

func UpdateSecret(label, value string) error {
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(keychainService)
	item.SetAccount(label)
	_ = keychain.DeleteItem(item)
	return saveSecret(label, value)
}

// ====== Secret Setters ======

func (s *SettingsService) SaveNotionToken(token string) error {
	println("Saving Notion Token: ", token)
	return UpdateSecret(keychainNotionToken, token)
}

func (s *SettingsService) SaveOpenAIKey(key string) error {
	return UpdateSecret(keychainOpenAIKey, key)
}

// ====== Settings Load/Save ======

func (s *SettingsService) LoadSettings() {
	data, err := os.ReadFile(settingsFilePath)
	if err == nil {
		_ = json.Unmarshal(data, &s.AppSettings)
	}

	s.AppSettings.NotionAccessToken, _ = LoadSecret(keychainNotionToken)
	s.AppSettings.OpenAIAPIKey, _ = LoadSecret(keychainOpenAIKey)
}

func (s *SettingsService) SaveSettings() {
	// Make a shallow copy and scrub secrets
	sanitized := s.AppSettings
	sanitized.NotionAccessToken = ""
	sanitized.OpenAIAPIKey = ""

	data, err := json.MarshalIndent(sanitized, "", "  ")
	if err == nil {
		_ = os.WriteFile(settingsFilePath, data, 0644)
	}

	if s.AppSettings.NotionAccessToken != "" {
		_ = UpdateSecret(keychainNotionToken, s.AppSettings.NotionAccessToken)
	}
	if s.AppSettings.OpenAIAPIKey != "" {
		_ = UpdateSecret(keychainOpenAIKey, s.AppSettings.OpenAIAPIKey)
	}
}

// ====== Frontend Integration ======

func (s *SettingsService) GetSettings() (FrontendSettings, error) {
	var frontend FrontendSettings
	frontend.UseOpenAI = s.AppSettings.UseOpenAI
	frontend.Theme = s.AppSettings.Theme
	frontend.LaunchOnStartup = s.AppSettings.LaunchOnStartup
	frontend.NotionDBID = s.AppSettings.NotionDBID
	frontend.DatePropertyID = s.AppSettings.DatePropertyID
	frontend.DatePropertyName = s.AppSettings.DatePropertyName

	hotkeyJSON, err := s.AppSettings.Hotkey.MarshalJSON()
	if err != nil {
		frontend.Hotkey = "invalid hotkey"
	} else {
		_ = json.Unmarshal(hotkeyJSON, &frontend.Hotkey)
	}

	// Load secret flags
	if token, err := LoadSecret(keychainNotionToken); err == nil && token != "" {
		frontend.HasConnectedNotion = true
	}
	if key, err := LoadSecret(keychainOpenAIKey); err == nil && key != "" {
		frontend.HasOpenAIAPIKey = true
	}

	s.FrontendOverrides = frontend
	return frontend, nil
}

func (s *SettingsService) UpdateSettingsFromFrontend(raw map[string]interface{}) error {
	hotkeyStr, ok := raw["hotkey"].(string)
	if !ok {
		return fmt.Errorf("hotkey must be a string")
	}
	hotkeyCfg, err := parseHotkeyString(hotkeyStr)
	if err != nil {
		return fmt.Errorf("invalid hotkey: %w", err)
	}

	data, _ := json.Marshal(raw)
	var newSettings ApplicationSettings
	_ = json.Unmarshal(data, &newSettings)
	newSettings.Hotkey = hotkeyCfg

	if newSettings.NotionAccessToken == "" {
		newSettings.NotionAccessToken, _ = LoadSecret(keychainNotionToken)
	}
	if newSettings.OpenAIAPIKey == "" {
		newSettings.OpenAIAPIKey, _ = LoadSecret(keychainOpenAIKey)
	}

	s.AppSettings = newSettings
	s.SaveSettings()
	return nil
}
