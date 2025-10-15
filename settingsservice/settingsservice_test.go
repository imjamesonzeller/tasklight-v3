package settingsservice

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/imjamesonzeller/tasklight-v3/startupservice"
	"golang.design/x/hotkey"
)

func TestSaveAndLoadSettings(t *testing.T) {
	t.Setenv("TASKLIGHT_SKIP_KEYCHAIN", "1")

	tempDir := t.TempDir()
	settingsPath := filepath.Join(tempDir, "settings.json")
	t.Setenv("TASKLIGHT_SETTINGS_PATH", settingsPath)

	startup := startupservice.NewStartupService()
	svc := NewSettingsService(startup)

	svc.AppSettings.NotionDataSourceID = "ds-42"
	svc.AppSettings.Theme = "dark"
	svc.AppSettings.Hotkey = hotkeyConfig{
		Modifiers: []hotkey.Modifier{hotkey.ModCmd},
		Key:       hotkey.KeyA,
	}
	svc.AppSettings.NotionAccessToken = "secret-token"
	svc.AppSettings.OpenAIAPIKey = "ai-secret"

	svc.SaveSettings()

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read settings file: %v", err)
	}
	if strings.Contains(string(data), "secret-token") || strings.Contains(string(data), "ai-secret") {
		t.Fatalf("secrets should be scrubbed from settings file: %s", string(data))
	}

	svcReload := NewSettingsService(startupservice.NewStartupService())
	if svcReload.AppSettings.NotionDataSourceID != "ds-42" {
		t.Fatalf("expected notion data source id to persist, got %s", svcReload.AppSettings.NotionDataSourceID)
	}
	if svcReload.AppSettings.Theme != "dark" {
		t.Fatalf("expected theme to persist, got %s", svcReload.AppSettings.Theme)
	}
	if svcReload.AppSettings.NotionAccessToken != "" || svcReload.AppSettings.OpenAIAPIKey != "" {
		t.Fatalf("secrets should not be loaded when keychain is skipped")
	}
}

func TestLoadSettingsRecoversFromCorruptFile(t *testing.T) {
	t.Setenv("TASKLIGHT_SKIP_KEYCHAIN", "1")

	tempDir := t.TempDir()
	settingsPath := filepath.Join(tempDir, "settings.json")
	t.Setenv("TASKLIGHT_SETTINGS_PATH", settingsPath)

	// Create a corrupt file that will fail to unmarshal into ApplicationSettings.
	corrupt := `{"theme": "midnight", "hotkey": "cmd+space"}`
	if err := os.WriteFile(settingsPath, []byte(corrupt), 0o644); err != nil {
		t.Fatalf("failed to write corrupt settings: %v", err)
	}

	startup := startupservice.NewStartupService()
	svc := NewSettingsService(startup)

	defaults := defaultApplicationSettings()
	if svc.AppSettings.Theme != defaults.Theme {
		t.Fatalf("expected theme to fall back to default %q, got %q", defaults.Theme, svc.AppSettings.Theme)
	}

	backups, err := filepath.Glob(settingsPath + ".corrupt-*")
	if err != nil {
		t.Fatalf("failed to glob for corrupt backups: %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("expected 1 corrupt settings backup, got %d (files: %v)", len(backups), backups)
	}

	contents, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read regenerated settings: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(contents, &decoded); err != nil {
		t.Fatalf("expected regenerated settings to be valid json: %v (data: %s)", err, string(contents))
	}

	svcReload := NewSettingsService(startupservice.NewStartupService())
	if svcReload.AppSettings.Theme != defaults.Theme {
		t.Fatalf("expected reload to keep default theme %q, got %q", defaults.Theme, svcReload.AppSettings.Theme)
	}

	backupsAfterReload, err := filepath.Glob(settingsPath + ".corrupt-*")
	if err != nil {
		t.Fatalf("failed to glob for corrupt backups after reload: %v", err)
	}
	if len(backupsAfterReload) != 1 {
		t.Fatalf("expected no additional corrupt backups after reload, got %d (files: %v)", len(backupsAfterReload), backupsAfterReload)
	}
}
