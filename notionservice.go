package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/imjamesonzeller/tasklight-v3/config"
	"github.com/imjamesonzeller/tasklight-v3/notionauth"
	"github.com/imjamesonzeller/tasklight-v3/settingsservice"
	"io"
	"log"
	"net/http"
	"os/exec"
	"runtime"
)

type NotionService struct {
	settingsservice *settingsservice.SettingsService
}

var ErrNotionTokenMissing = errors.New("notion access token unavailable")

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

type Filter struct {
	Value    string `json:"value"`
	Property string `json:"property"`
}

type NotionSearchRequest struct {
	Filter Filter `json:"filter"`
}

type NotionDBResponse struct {
	Results []DatabaseMinimal `json:"results"`
}

type DatabaseMinimal struct {
	ID                   string                 `json:"id"`
	Title                []RichTextObj          `json:"title"`
	Properties           map[string]PropertyObj `json:"properties"`
	HasMultipleDateProps bool                   `json:"has_multiple_date_props"`
}

type RichTextObj struct {
	Text TextContent `json:"text"`
}

type TextContent struct {
	Content string `json:"content"`
}

type PropertyObj struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

func (n *NotionService) GetNotionDatabases() (*NotionDBResponse, error) {
	NotionSecret := config.AppConfig.NotionAccessToken
	NotionSearchURL := "https://api.notion.com/v1/search"

	println("Notion secret:", NotionSecret)

	data := NotionSearchRequest{Filter{
		Value:    "database",
		Property: "object",
	}}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequest("POST", NotionSearchURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Authorization", "Bearer "+NotionSecret)
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Notion-Version", "2022-06-28")

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("%w: status %d, body: %s", ErrNotionTokenMissing, resp.StatusCode, body)
		}
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get databases, status %d, body: %s", resp.StatusCode, body)
	}

	var dbResp NotionDBResponse
	if err := json.NewDecoder(resp.Body).Decode(&dbResp); err != nil {
		return nil, err
	}

	for i := range dbResp.Results {
		db := &dbResp.Results[i]

		var dateProps []PropertyObj
		for _, prop := range db.Properties {
			if prop.Type == "date" {
				dateProps = append(dateProps, prop)
			}
		}

		if len(dateProps) == 1 {
			if len(dateProps) == 1 && db.ID == n.settingsservice.AppSettings.NotionDBID {
				n.settingsservice.AppSettings.DatePropertyID = dateProps[0].ID
				n.settingsservice.AppSettings.DatePropertyName = dateProps[0].Name
				n.settingsservice.SaveSettings()
			}
		} else if len(dateProps) > 1 {
			db.HasMultipleDateProps = true
		}
	}

	return &dbResp, nil
}

func (n *NotionService) GetNotionWorkspaceId() (string, error) {
	notionToken := config.AppConfig.NotionAccessToken
	if notionToken == "" {
		return "", ErrNotionTokenMissing
	}

	req, err := http.NewRequest("GET", "https://api.notion.com/v1/users/me", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+notionToken)
	req.Header.Add("Notion-Version", "2022-06-28")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("%w: status %d, body: %s", ErrNotionTokenMissing, resp.StatusCode, body)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("/v1/users/me failed: status %d, body: %s", resp.StatusCode, body)
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}

	idValue, ok := payload["id"]
	if !ok {
		return "", fmt.Errorf("id not found in response")
	}

	idStr, ok := idValue.(string)
	if !ok {
		return "", fmt.Errorf("id field is not a string")
	}

	return idStr, nil
}
