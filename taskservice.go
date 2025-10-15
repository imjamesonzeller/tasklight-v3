package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	c "github.com/imjamesonzeller/tasklight-v3/config"
	"github.com/imjamesonzeller/tasklight-v3/notionapi"
	"github.com/imjamesonzeller/tasklight-v3/settingsservice"
	"github.com/openai/openai-go/option"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type TaskInformation struct {
	Title string  `json:"title"`
	Date  *string `json:"date"`
}

type TaskService struct {
	app           *application.App
	windowService *WindowService
	settings      *settingsservice.SettingsService
}

func NewTaskService(windowService *WindowService, settings *settingsservice.SettingsService) *TaskService {
	return &TaskService{
		windowService: windowService,
		settings:      settings,
	}
}

func (ts *TaskService) SetApp(app *application.App) {
	ts.app = app
}

// ProcessMessage Called from frontend
func (ts *TaskService) ProcessMessage(message string) {
	ts.windowService.Hide("main")

	go func() {
		task := ts.ProcessedThroughAI(message)
		status := ts.SendToNotion(task)

		if status != "200 OK" {
			ts.app.EmitEvent("Backend:ErrorEvent", status)
			ts.windowService.Show("main")
		}
	}()
}

// --- Internals ---

func (ts *TaskService) ProcessedThroughAI(input string) TaskInformation {
	key, userProvided := ts.selectOpenAIKey()
	if userProvided {
		prompt := buildParsePrompt(input)
		task, err := ts.callOpenAI(key, prompt)
		if err != nil {
			log.Println("ProcessedThroughAI: AI call failed:", err)
			return TaskInformation{Title: input, Date: nil}
		}
		return task
	}

	// No user key: call server to parse (server owns key, checks & increments usage)
	task, err := ts.callServerParse(input)
	if err != nil {
		log.Println("ProcessedThroughAI: server parse failed:", err)
		return TaskInformation{Title: input, Date: nil}
	}
	return task
}

// selectOpenAIKey decides which key to use and returns (key, userProvided)
func (ts *TaskService) selectOpenAIKey() (string, bool) {
	if !c.AppConfig.UseOpenAI {
		return "", false
	}

	key, err := ts.settings.GetOpenAIKey()
	if err != nil {
		log.Println("selectOpenAIKey: failed to load OpenAI key:", err)
		return "", false
	}
	if key == "" {
		return "", false
	}

	return key, true
}

// buildParsePrompt returns the exact prompt text for parsing
func buildParsePrompt(input string) string {
	today := time.Now().Format("2006-01-02") // ISO 8601
	weekday := time.Now().Weekday().String()
	return fmt.Sprintf(`You are a precise and reliable task parsing assistant. 
						Your job is to convert natural-language task descriptions into clean, structured data.
			
						Today's date is %s. Today is a %s.
			
						When parsing dates:
						- Always interpret dates as referring to the **next upcoming instance in the future** (never in the past) unless the text clearly says “last” or “previous”.
						- Correct common spelling mistakes in weekday or month names (e.g., "firday" -> "Friday", "janury" -> "January").
						- If the intended date is ambiguous, choose the most **reasonable future** date based on context.
						- Use ISO 8601 format (YYYY-MM-DD) for all dates.
			
						Parse the following sentence: "%s".
			
						Ignore phrases like "remind me to", "remind me on", or similar expressions—only focus on the task and the date.
			
						Return only a JSON object in this exact format:
						{ "title": ..., "date": ... }
			
						If no date is mentioned, set "date" to null.`, today, weekday, input)
}

// callOpenAI sends the prompt and parses the returned JSON content into TaskInformation
func (ts *TaskService) callOpenAI(apiKey string, prompt string) (TaskInformation, error) {
	client := openai.NewClient(option.WithAPIKey(apiKey))
	resp, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Model:    openai.ChatModelGPT4oMini,
		Messages: []openai.ChatCompletionMessageParamUnion{openai.UserMessage(prompt)},
	})
	if err != nil {
		return TaskInformation{}, err
	}
	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return TaskInformation{}, fmt.Errorf("empty AI response")
	}
	return parseTaskFromContent(resp.Choices[0].Message.Content)
}

// callServerParse sends the text to the backend, which handles OpenAI calls and usage accounting
func (ts *TaskService) callServerParse(input string) (TaskInformation, error) {
	userID := c.GetCurrentUserId()
	if userID == "" {
		return TaskInformation{}, fmt.Errorf("no current user id set; connect Notion")
	}
	payload := map[string]string{"text": input}
	body, err := json.Marshal(payload)
	if err != nil {
		return TaskInformation{}, err
	}
	apiBase := c.GetEnv("TASKLIGHT_API_BASE")
	if apiBase == "" {
		apiBase = "https://api.jamesonzeller.com"
	}
	endpoint := strings.TrimRight(apiBase, "/") + "/tasklight/parse"

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return TaskInformation{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Notion-User-Id", userID)

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return TaskInformation{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return TaskInformation{}, fmt.Errorf("server parse %d: %s", resp.StatusCode, string(b))
	}

	var parsed TaskInformation
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return TaskInformation{}, err
	}
	return parsed, nil
}

// parseTaskFromContent converts the model's JSON string into TaskInformation
func parseTaskFromContent(content string) (TaskInformation, error) {
	var task TaskInformation
	if err := json.Unmarshal([]byte(content), &task); err != nil {
		return TaskInformation{}, err
	}
	return task, nil
}

func (ts *TaskService) SendToNotion(task TaskInformation) string {
	token, err := ts.settings.GetNotionToken(true)
	if err != nil || token == "" {
		msg := "Notion token unavailable; reconnect Notion from settings"
		if err != nil {
			log.Println("SendToNotion: failed to load token:", err)
			msg = "Failed to load Notion token"
		}
		return msg
	}

	if c.AppConfig == nil {
		log.Println("SendToNotion: configuration not initialised")
		return "Tasklight configuration not ready; reopen the app."
	}

	if c.AppConfig.NotionDataSourceID == "" {
		log.Println("SendToNotion: data source not selected")
		return "Data source not selected for this Notion database."
	}

	dataSource, err := ts.loadDataSourceDetail(token, c.AppConfig.NotionDataSourceID)
	if err != nil {
		log.Println("SendToNotion: data source load failed:", err)
		return fmt.Sprintf("Failed to load Notion data source: %v", err)
	}

	titlePropName, err := detectTitleProperty(dataSource)
	if err != nil {
		log.Println("SendToNotion:", err)
		return err.Error()
	}

	if c.AppConfig.DatePropertyName != "" {
		prop, ok := dataSource.Properties[c.AppConfig.DatePropertyName]
		if !ok || prop.Type != "date" {
			msg := fmt.Sprintf("Selected Notion property %q is not available on the chosen data source.", c.AppConfig.DatePropertyName)
			log.Println("SendToNotion:", msg)
			return msg
		}
	}

	payload := buildNotionPagePayload(task, c.AppConfig.NotionDataSourceID, titlePropName, c.AppConfig.DatePropertyName)

	req, err := notionapi.NewJSONRequest(http.MethodPost, "https://api.notion.com/v1/pages", token, payload)
	if err != nil {
		log.Println("SendToNotion: failed to create request:", err)
		return err.Error()
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		msg := fmt.Sprintf("Error sending request: %v", err)
		log.Println(msg)
		return msg
	}
	status := resp.Status

	if err := notionapi.ParseResponse(resp, nil, ErrNotionTokenMissing); err != nil {
		log.Println("SendToNotion: Notion API error:", err)
		return err.Error()
	}

	log.Printf("Notion page created (status %s) using data source %s", status, c.AppConfig.NotionDataSourceID)
	return status
}

func (ts *TaskService) loadDataSourceDetail(token, dataSourceID string) (*NotionDataSourceDetail, error) {
	req, err := notionapi.NewRequest(http.MethodGet, fmt.Sprintf("https://api.notion.com/v1/data_sources/%s", dataSourceID), token)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	var detail NotionDataSourceDetail
	if err := notionapi.ParseResponse(resp, &detail, ErrNotionTokenMissing); err != nil {
		return nil, err
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

	return &detail, nil
}

func detectTitleProperty(detail *NotionDataSourceDetail) (string, error) {
	for name, prop := range detail.Properties {
		if prop.Type == "title" {
			if prop.Name != "" {
				return prop.Name, nil
			}
			return name, nil
		}
	}
	return "", fmt.Errorf("Selected Notion data source is missing a title property.")
}

func buildNotionPagePayload(task TaskInformation, dataSourceID, titlePropertyName, datePropertyName string) map[string]any {
	nameProp := map[string]any{
		"type": "title",
		"title": []map[string]any{
			{
				"type": "text",
				"text": map[string]any{
					"content": task.Title,
				},
			},
		},
	}

	properties := map[string]any{
		titlePropertyName: nameProp,
	}

	if task.Date != nil && datePropertyName != "" {
		properties[datePropertyName] = map[string]any{
			"date": map[string]any{
				"start": *task.Date,
			},
		}
	}

	return map[string]any{
		"parent": map[string]any{
			"type":           "data_source_id",
			"data_source_id": dataSourceID,
		},
		"properties": properties,
	}
}
