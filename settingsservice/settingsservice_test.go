package settingsservice

import (
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
