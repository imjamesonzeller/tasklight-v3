package main

import (
	"errors"
	"fmt"
	"github.com/imjamesonzeller/tasklight-v3/config"
	"github.com/imjamesonzeller/tasklight-v3/notionapi"
	"github.com/imjamesonzeller/tasklight-v3/notionauth"
	"github.com/imjamesonzeller/tasklight-v3/settingsservice"
	"github.com/keybase/go-keychain"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

type NotionService struct {
	settingsservice *settingsservice.SettingsService
	oauthMu         sync.Mutex
	oauthInProgress bool
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
	clientID := config.GetEnv("NOTION_CLIENT_ID")
	redirectURI := config.GetEnv("NOTION_REDIRECT_URI")
	if redirectURI == "" {
		redirectURI = "http://localhost:5173/oauth"
	}

	if clientID == "" {
		log.Println("⚠️ NOTION_CLIENT_ID is not configured; cannot start OAuth flow")
		return
	}

	if !n.beginOAuthListener() {
		log.Println("ℹ️ Notion OAuth already in progress; ignoring duplicate start request")
		return
	}

	go func() {
		defer n.endOAuthListener()
		notionauth.StartLocalOAuthListener(n.settingsservice)
	}()

	authURL := url.URL{
		Scheme: "https",
		Host:   "api.notion.com",
		Path:   "/v1/oauth/authorize",
	}

	query := url.Values{}
	query.Set("client_id", clientID)
	query.Set("response_type", "code")
	query.Set("owner", "user")
	query.Set("redirect_uri", redirectURI)

	authURL.RawQuery = query.Encode()

	log.Println("🔍 Launching Notion OAuth:", authURL.String())
	openBrowser(authURL.String())
}

func (n *NotionService) beginOAuthListener() bool {
	n.oauthMu.Lock()
	defer n.oauthMu.Unlock()

	if n.oauthInProgress {
		return false
	}

	n.oauthInProgress = true
	return true
}

func (n *NotionService) endOAuthListener() {
	n.oauthMu.Lock()
	n.oauthInProgress = false
	n.oauthMu.Unlock()
}

type Filter struct {
	Value    string `json:"value"`
	Property string `json:"property"`
}

type NotionSearchRequest struct {
	Filter Filter `json:"filter"`
}

type NotionDataSourceList struct {
	Results []NotionDataSourceSummary `json:"results"`
}

type RichTextObj struct {
	Text      TextContent `json:"text"`
	PlainText string      `json:"plain_text"`
}

type TextContent struct {
	Content string `json:"content"`
}

type PropertyObj struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type NotionDataSourceSummary struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	ParentDatabaseID string `json:"parent_database_id,omitempty"`
}

type NotionDataSourceDetail struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Properties map[string]PropertyObj `json:"properties"`
}

type notionParent struct {
	Type       string `json:"type"`
	DatabaseID string `json:"database_id"`
}

type notionDataSourcePayload struct {
	ID   string        `json:"id"`
	Name []RichTextObj `json:"name"`
}

type dataSourceSearchResult struct {
	Object      string                  `json:"object"`
	ID          string                  `json:"id"`
	DataSource  notionDataSourcePayload `json:"data_source"`
	DisplayName []RichTextObj           `json:"display_name"`
	Parent      notionParent            `json:"parent"`
	Properties  map[string]any          `json:"properties"`
	Title       []RichTextObj           `json:"title"`
}

type dataSourceSearchResponse struct {
	Results    []dataSourceSearchResult `json:"results"`
	HasMore    bool                     `json:"has_more"`
	NextCursor *string                  `json:"next_cursor"`
}

func richTextPlainText(blocks []RichTextObj) string {
	var b strings.Builder
	for _, block := range blocks {
		if block.PlainText != "" {
			b.WriteString(block.PlainText)
			continue
		}
		if block.Text.Content != "" {
			b.WriteString(block.Text.Content)
		}
	}
	return strings.TrimSpace(b.String())
}

func (n *NotionService) GetNotionDatabases() (*NotionDataSourceList, error) {
	notionSecret, err := n.settingsservice.GetNotionToken(false)
	if err != nil {
		if errors.Is(err, keychain.ErrorItemNotFound) {
			return nil, ErrNotionTokenMissing
		}
		return nil, err
	}

	if notionSecret == "" {
		return nil, ErrNotionTokenMissing
	}
	NotionSearchURL := "https://api.notion.com/v1/search"

	seen := make(map[string]struct{})
	var results []NotionDataSourceSummary
	var cursor *string

	for {
		payload := map[string]any{
			"filter": map[string]any{
				"value":    "data_source",
				"property": "object",
			},
			"page_size": 100,
		}
		if cursor != nil {
			payload["start_cursor"] = *cursor
		}

		req, err := notionapi.NewJSONRequest(http.MethodPost, NotionSearchURL, notionSecret, payload)
		if err != nil {
			return nil, err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		var search dataSourceSearchResponse
		if err := notionapi.ParseResponse(resp, &search, ErrNotionTokenMissing); err != nil {
			return nil, err
		}

		for _, result := range search.Results {
			dbID := result.Parent.DatabaseID
			ds := result.DataSource
			if ds.ID == "" {
				ds.ID = result.ID
			}

			if ds.ID == "" {
				continue
			}

			if _, ok := seen[ds.ID]; ok {
				continue
			}
			seen[ds.ID] = struct{}{}

			name := richTextPlainText(ds.Name)
			if name == "" {
				name = richTextPlainText(result.DisplayName)
			}
			if name == "" {
				name = richTextPlainText(result.Title)
			}

			results = append(results, NotionDataSourceSummary{
				ID:               ds.ID,
				Name:             name,
				ParentDatabaseID: dbID,
			})
		}

		if !search.HasMore || search.NextCursor == nil || *search.NextCursor == "" {
			break
		}
		cursor = search.NextCursor
	}

	return &NotionDataSourceList{Results: results}, nil
}

func (n *NotionService) GetDataSourceDetail(dataSourceID string) (*NotionDataSourceDetail, error) {
	if dataSourceID == "" {
		return nil, fmt.Errorf("data source id is required")
	}

	token, err := n.settingsservice.GetNotionToken(false)
	if err != nil {
		if errors.Is(err, keychain.ErrorItemNotFound) {
			return nil, ErrNotionTokenMissing
		}
		return nil, err
	}
	if token == "" {
		return nil, ErrNotionTokenMissing
	}

	return n.fetchDataSourceDetail(token, dataSourceID)
}

func (n *NotionService) GetNotionWorkspaceId() (string, error) {
	notionToken, err := n.settingsservice.GetNotionToken(false)
	if err != nil {
		if errors.Is(err, keychain.ErrorItemNotFound) {
			return "", ErrNotionTokenMissing
		}
		return "", err
	}

	if notionToken == "" {
		return "", ErrNotionTokenMissing
	}

	req, err := notionapi.NewRequest(http.MethodGet, "https://api.notion.com/v1/users/me", notionToken)
	if err != nil {
		return "", err
	}

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return "", err
	}

	var payload struct {
		ID string `json:"id"`
	}
	if err := notionapi.ParseResponse(resp, &payload, ErrNotionTokenMissing); err != nil {
		return "", err
	}

	if payload.ID == "" {
		return "", fmt.Errorf("id not found in response")
	}

	return payload.ID, nil
}

func (n *NotionService) fetchDataSourceDetail(token, dataSourceID string) (*NotionDataSourceDetail, error) {
	req, err := notionapi.NewRequest(http.MethodGet, fmt.Sprintf("https://api.notion.com/v1/data_sources/%s", dataSourceID), token)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	var payload struct {
		ID         string                 `json:"id"`
		Name       []RichTextObj          `json:"name"`
		Properties map[string]PropertyObj `json:"properties"`
	}
	if err := notionapi.ParseResponse(resp, &payload, ErrNotionTokenMissing); err != nil {
		return nil, err
	}

	detail := NotionDataSourceDetail{
		ID:         payload.ID,
		Name:       richTextPlainText(payload.Name),
		Properties: payload.Properties,
	}

	if detail.Properties == nil {
		detail.Properties = map[string]PropertyObj{}
	}

	for key, prop := range detail.Properties {
		if prop.Name == "" {
			prop.Name = key
			detail.Properties[key] = prop
		}
	}

	if detail.ID == n.settingsservice.AppSettings.NotionDataSourceID &&
		n.settingsservice.AppSettings.DatePropertyID == "" {
		var dateProps []PropertyObj
		for _, prop := range detail.Properties {
			if prop.Type == "date" {
				dateProps = append(dateProps, prop)
			}
		}

		if len(dateProps) == 1 {
			n.settingsservice.AppSettings.DatePropertyID = dateProps[0].ID
			n.settingsservice.AppSettings.DatePropertyName = dateProps[0].Name
			n.settingsservice.SaveSettings()
		}
	}

	return &detail, nil
}
