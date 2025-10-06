package settingsservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/imjamesonzeller/tasklight-v3/startupservice"
	"os"
	"path/filepath"
	"runtime/debug"
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
	settingsFileName    = "settings.json"
)

// ====== Structs ======

type SettingsService struct {
	App               *application.App
	AppSettings       ApplicationSettings
	FrontendOverrides FrontendSettings
	StartupService    *startupservice.StartupService
	settingsPath      string
	appVersion        string
}

func keychainDisabled() bool {
	return os.Getenv("TASKLIGHT_SKIP_KEYCHAIN") == "1"
}

type ApplicationSettings struct {
	NotionDBID string       `json:"notion_db_id"`
	UseOpenAI  bool         `json:"use_open_ai"`
	Theme      string       `json:"theme"`
	Hotkey     hotkeyConfig `json:"hotkey"`

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

func NewSettingsService(startup *startupservice.StartupService) *SettingsService {
	service := &SettingsService{StartupService: startup}
	service.settingsPath = resolveSettingsPath()
	service.AppSettings = defaultApplicationSettings()
	service.appVersion = detectAppVersion()
	service.LoadSettings()
	return service
}

func resolveSettingsPath() string {
	if override := os.Getenv("TASKLIGHT_SETTINGS_PATH"); override != "" {
		return override
	}

	configDir, err := os.UserConfigDir()
	if err != nil || configDir == "" {
		return settingsFileName
	}

	appDir := filepath.Join(configDir, "Tasklight")
	if mkErr := os.MkdirAll(appDir, 0o755); mkErr != nil {
		return settingsFileName
	}

	return filepath.Join(appDir, settingsFileName)
}

func (s *SettingsService) SetApp(app *application.App) {
	s.App = app
}

func defaultApplicationSettings() ApplicationSettings {
	config, err := parseHotkeyString("ctrl+space")
	if err != nil {
		config = hotkeyConfig{}
	}

	return ApplicationSettings{
		Theme:  "light",
		Hotkey: config,
	}
}

func detectAppVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info != nil {
		version := strings.TrimSpace(info.Main.Version)
		if version != "" && version != "(devel)" {
			return version
		}

		var revision string
		var modified bool
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				revision = strings.TrimSpace(setting.Value)
			case "vcs.modified":
				modified = setting.Value == "true"
			}
		}

		if revision != "" {
			if len(revision) > 8 {
				revision = revision[:8]
			}
			if modified {
				return "dev-" + revision + "*"
			}
			return "dev-" + revision
		}
	}

	if envVersion := strings.TrimSpace(os.Getenv("TASKLIGHT_APP_VERSION")); envVersion != "" {
		return envVersion
	}

	return "development"
}

func (s *SettingsService) GetAppVersion() string {
	if s.appVersion == "" {
		s.appVersion = detectAppVersion()
	}
	return s.appVersion
}

func (s *SettingsService) ClearLocalCache() (bool, error) {
	var errs []error

	if keychainDisabled() {
		s.AppSettings.NotionAccessToken = ""
		s.AppSettings.OpenAIAPIKey = ""
	} else {
		if err := clearSecret(keychainNotionToken); err != nil && !errors.Is(err, keychain.ErrorItemNotFound) {
			errs = append(errs, err)
		}
		if err := clearSecret(keychainOpenAIKey); err != nil && !errors.Is(err, keychain.ErrorItemNotFound) {
			errs = append(errs, err)
		}
	}

	if s.settingsPath == "" {
		s.settingsPath = resolveSettingsPath()
	}

	if s.settingsPath != "" {
		if err := os.Remove(s.settingsPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
		}
	}

	if s.StartupService != nil {
		if s.StartupService.IsEnabled() {
			if err := s.StartupService.DisableLaunchAtLogin(); err != nil {
				errs = append(errs, err)
			}
		}
	}

	s.AppSettings = defaultApplicationSettings()
	s.FrontendOverrides = FrontendSettings{}

	if len(errs) > 0 {
		return false, errors.Join(errs...)
	}

	return true, nil
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
	_ = clearSecret(label)
	return saveSecret(label, value)
}

func clearSecret(label string) error {
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(keychainService)
	item.SetAccount(label)
	return keychain.DeleteItem(item)
}

// ====== Secret Setters ======

func (s *SettingsService) SaveNotionToken(token string) error {
	if keychainDisabled() {
		s.AppSettings.NotionAccessToken = token
		return nil
	}
	return UpdateSecret(keychainNotionToken, token)
}

func (s *SettingsService) SaveOpenAIKey(key string) error {
	if keychainDisabled() {
		s.AppSettings.OpenAIAPIKey = key
		return nil
	}
	err := UpdateSecret(keychainOpenAIKey, key)
	if err != nil {
		return err
	}

	s.AppSettings.OpenAIAPIKey = key

	return nil
}

func (s *SettingsService) ClearOpenAIKey() error {
	if keychainDisabled() {
		s.AppSettings.OpenAIAPIKey = ""
		return nil
	}
	return clearSecret(keychainOpenAIKey)
}

// ====== Settings Load/Save ======

func (s *SettingsService) LoadSettings() {
	if s.settingsPath == "" {
		s.settingsPath = resolveSettingsPath()
	}

	data, err := os.ReadFile(s.settingsPath)
	if err == nil {
		_ = json.Unmarshal(data, &s.AppSettings)
	}

	if !keychainDisabled() {
		s.AppSettings.NotionAccessToken, _ = LoadSecret(keychainNotionToken)
		s.AppSettings.OpenAIAPIKey, _ = LoadSecret(keychainOpenAIKey)
	}
}

func (s *SettingsService) SaveSettings() {
	// Make a shallow copy and scrub secrets
	sanitized := s.AppSettings
	sanitized.NotionAccessToken = ""
	sanitized.OpenAIAPIKey = ""

	if s.settingsPath == "" {
		s.settingsPath = resolveSettingsPath()
	}
	_ = os.MkdirAll(filepath.Dir(s.settingsPath), 0o755)

	data, err := json.MarshalIndent(sanitized, "", "  ")
	if err == nil {
		_ = os.WriteFile(s.settingsPath, data, 0644)
	}

	if !keychainDisabled() {
		if s.AppSettings.NotionAccessToken != "" {
			_ = UpdateSecret(keychainNotionToken, s.AppSettings.NotionAccessToken)
		}
		if s.AppSettings.OpenAIAPIKey != "" {
			_ = UpdateSecret(keychainOpenAIKey, s.AppSettings.OpenAIAPIKey)
		}
	}
}

// ====== Frontend Integration ======

func (s *SettingsService) GetSettings() (FrontendSettings, error) {
	var frontend FrontendSettings
	frontend.UseOpenAI = s.AppSettings.UseOpenAI
	frontend.Theme = s.AppSettings.Theme
	frontend.LaunchOnStartup = s.StartupService.IsEnabled()
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
	if !keychainDisabled() {
		if token, err := LoadSecret(keychainNotionToken); err == nil && token != "" {
			frontend.HasConnectedNotion = true
		}
		if key, err := LoadSecret(keychainOpenAIKey); err == nil && key != "" {
			frontend.HasOpenAIAPIKey = true
		}
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

	if newSettings.NotionAccessToken == "" && !keychainDisabled() {
		newSettings.NotionAccessToken, _ = LoadSecret(keychainNotionToken)
	}
	if newSettings.OpenAIAPIKey == "" && !keychainDisabled() {
		newSettings.OpenAIAPIKey, _ = LoadSecret(keychainOpenAIKey)
	}

	if launchRaw, ok := raw["launch_on_startup"].(bool); ok {
		if launchRaw {
			_ = s.StartupService.EnableLaunchAtLogin()
		} else {
			_ = s.StartupService.DisableLaunchAtLogin()
		}
	}

	s.AppSettings = newSettings
	s.SaveSettings()

	// Emit settings saved event so hotkey can catch it and
	s.App.EmitEvent("Backend:SettingsUpdated", map[string]any{
		"theme": s.AppSettings.Theme,
	})

	return nil
}
