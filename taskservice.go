package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	c "github.com/imjamesonzeller/tasklight-v3/config"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/imjamesonzeller/tasklight-v3/auth"
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
	identity      *auth.Identity
}

func NewTaskService(windowService *WindowService) *TaskService {
	return &TaskService{
		windowService: windowService,
	}
}

func (ts *TaskService) SetApp(app *application.App) {
	ts.app = app
}

func (ts *TaskService) SetIdentity(identity *auth.Identity) {
	ts.identity = identity
}

func (ts *TaskService) CanUseAI() (bool, int) {
	if ts.identity == nil {
		log.Println("No identity loaded")
		return false, 0
	}

	url := fmt.Sprintf("https://api.jamesonzeller.com/check-usage?user_id=%s&auth_token=%s", ts.identity.UserID, ts.identity.AuthToken)

	resp, err := http.Get(url)
	if err != nil {
		log.Println("Failed to contact usage server: ", err)
		return false, 0
	}
	defer resp.Body.Close()

	var result struct {
		Allowed   bool `json:"allowed"`
		Remaining int  `json:"remaining"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		log.Println("Failed to parse usage response: ", err)
		return false, 0
	}

	return result.Allowed, result.Remaining
}

func (ts *TaskService) IncrementUsage() {
	if ts.identity == nil {
		log.Println("No identity loaded")
		return
	}

	data := map[string]string{
		"user_id":    ts.identity.UserID,
		"auth_token": ts.identity.AuthToken,
	}
	jsonData, err := json.Marshal(data)

	resp, err := http.Post("https://api.jamesonzeller.com/increment-usage", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("Failed to increment usage: ", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Println("Increment usage failed: ", string(body))
	}
}

// ProcessMessage Called from frontend
func (ts *TaskService) ProcessMessage(message string) {
	task := ts.ProcessedThroughAI(message)

	status := ts.SendToNotion(task)

	if status != "200 OK" {
		ts.app.EmitEvent("Backend:ErrorEvent", status)
	} else {
		ts.windowService.ToggleVisibility("main")
	}
}

// --- Internals ---

func (ts *TaskService) ProcessedThroughAI(input string) TaskInformation {
	allowed, _ := ts.CanUseAI()
	if !allowed {
		return TaskInformation{Title: input, Date: nil}
	}

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
		log.Println("OpenAI call failed:", err)
		return TaskInformation{Title: input, Date: nil}
	}

	var task TaskInformation
	err = json.Unmarshal([]byte(resp.Choices[0].Message.Content), &task)
	if err != nil {
		log.Println("Failed to parse AI output:", err)
		return TaskInformation{Title: input, Date: nil}
	}

	ts.IncrementUsage()
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
