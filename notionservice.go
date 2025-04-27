package main

import (
	"github.com/imjamesonzeller/tasklight-v3/config"
	"github.com/imjamesonzeller/tasklight-v3/notionauth"
	"github.com/imjamesonzeller/tasklight-v3/settingsservice"
	"log"
	"os/exec"
	"runtime"
)

type NotionService struct {
	settingsservice *settingsservice.SettingsService
}

func NewNotionService(settingsservice *settingsservice.SettingsService) *NotionService {
	return &NotionService{settingsservice: settingsservice}
}

func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "linux":
		cmd = "xdg-open"
	case "windows":
		cmd = "rundll32"
		args = append(args, "url.dll,FileProtocolHandler")
	}

	args = append(args, url)
	err := exec.Command(cmd, args...).Start()
	if err != nil {
		return
	}
}

func (n *NotionService) StartOAuth() {
	go notionauth.StartLocalOAuthListener(n.settingsservice)

	url := config.GetEnv("NOTION_AUTH_URL")
	log.Println("🔍 NOTION_AUTH_URL:", url)
	openBrowser(url)
}

// TODO: Add Frontend function GetNotionDatabases so that we can display options
// map[string]string is notionDBID to NotionDBName

func (n *NotionService) GetNotionDatabases() map[string]string {
	return nil
}
