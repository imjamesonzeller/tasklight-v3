package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	c "github.com/imjamesonzeller/tasklight-v3/config"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
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
	task := ts.ProcessedThroughAI(message)

	// API PASS-THROUGH
	// task := TaskInformation{message, nil}

	status := ts.SendToNotion(task)

	if status != "200 OK" {
		ts.app.EmitEvent("Backend:ErrorEvent", status)
	} else {
		ts.windowService.ToggleVisibility("main")
	}
}

// --- Internals ---

func (ts *TaskService) ProcessedThroughAI(input string) TaskInformation {
	// TODO: Add handling of No API key and failure of OpenAI request so it just sends input back.

	client := openai.NewClient(option.WithAPIKey(c.AppConfig.OpenAIAPIKey))

	today := time.Now().Format("2006-01-02") // ISO 8601 format
	prompt := fmt.Sprintf(`You are a helpful task parsing assistant. Your job is to parse natural language
                                  task descriptions into structured data.
                                  Today's date is is %s.
                                  Extract the task title and date from this sentence: "%s".
                                  Return only a JSON object in this exact format: { "title": ..., "date": ... }.
                                  If no date is mentioned, set the "date" value to null.
                                  The date should be in ISO 8601 format when present.`, today, input)

	resp, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4oMini,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	})
	if err != nil {
		panic(err)
	}

	// Extract and parse JSON
	var task TaskInformation
	err = json.Unmarshal([]byte(resp.Choices[0].Message.Content), &task)
	if err != nil {
		panic(err)
	}

	return task
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
		properties["Due Date"] = map[string]interface{}{
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
	req.Header.Add("Authorization", "Bearer "+c.AppConfig.NotionSecret)
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
