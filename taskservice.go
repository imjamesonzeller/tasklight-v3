package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	c "github.com/imjamesonzeller/tasklight-v3/config"
	"github.com/openai/openai-go/option"
	"io"
	"log"
	"net/http"
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
}

func NewTaskService(windowService *WindowService) *TaskService {
	return &TaskService{
		windowService: windowService,
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
	userProvided := c.AppConfig.UseOpenAI && c.AppConfig.OpenAIAPIKey != ""
	if userProvided {
		return c.AppConfig.OpenAIAPIKey, true
	}
	return "", false
}

// buildParsePrompt returns the exact prompt text for parsing
func buildParsePrompt(input string) string {
	today := time.Now().Format("2006-01-02") // ISO 8601
	weekday := time.Now().Weekday().String()
	return fmt.Sprintf(`You are a helpful task parsing assistant. Your job is to convert natural language
                                  task descriptions into clean, structured data.
                                  Today's date is is %s. Today is a %s
                                  Parse the following sentence: "%s".
                                  Ignore phrases like "remind me to", "remind me on", or similar expressions,
                                  only focus on the task and date.
                                  Return only a JSON object in this exact format: { "title": ..., "date": ... }.
                                  If no date is mentioned, set the "date" value to null.
                                  The date should be in ISO 8601 format when present.`, today, weekday, input)
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
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/tasklight/parse", bytes.NewBuffer(body))
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
	nameProp := map[string]interface{}{
		"type": "title",
		"title": []map[string]interface{}{
			{
				"type": "text",
				"text": map[string]interface{}{
					"content": task.Title,
				},
			},
		},
	}

	properties := map[string]interface{}{
		"Name": nameProp,
	}

	if task.Date != nil {
		properties[c.AppConfig.DatePropertyName] = map[string]interface{}{
			"date": map[string]interface{}{
				"start": *task.Date,
			},
		}
	}

	postBody := map[string]interface{}{
		"parent": map[string]string{
			"type":        "database_id",
			"database_id": c.AppConfig.NotionDBID,
		},
		"properties": properties,
	}

	jsonData, err := json.Marshal(postBody)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return err.Error()
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.notion.com/v1/pages", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}

	//println("NotionAccessToken: ", c.AppConfig.NotionAccessToken)

	req.Header.Add("Authorization", "Bearer "+c.AppConfig.NotionAccessToken)
	req.Header.Add("Content-Type", "application/json; charset=utf-8")
	req.Header.Add("Notion-Version", "2022-06-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Notion response: %s", string(body))
	return resp.Status
}
