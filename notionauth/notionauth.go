package notionauth

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/imjamesonzeller/tasklight-v3/config"
	"github.com/imjamesonzeller/tasklight-v3/settingsservice"
)

type NotionOAuthResponse struct {
	AccessToken          string  `json:"access_token"`
	BotID                string  `json:"bot_id"`
	DuplicatedTemplateID *string `json:"duplicated_template_id,omitempty"`
	Owner                Owner   `json:"owner"`
	WorkspaceIcon        *string `json:"workspace_icon,omitempty"`
	WorkspaceID          string  `json:"workspace_id"`
	WorkspaceName        *string `json:"workspace_name,omitempty"`
}

type Owner struct {
	Workspace *bool   `json:"workspace,omitempty"`
	Object    *string `json:"object,omitempty"`
	ID        *string `json:"id,omitempty"`
	Type      *string `json:"type,omitempty"`
	Name      *string `json:"name,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

type NotionOAuthRequest struct {
	GrantType   string `json:"grant_type"`
	Code        string `json:"code"`
	RedirectUri string `json:"redirect_uri"`
}

// StartLocalOAuthListener starts http server and handles shutting it down
func StartLocalOAuthListener(settings *settingsservice.SettingsService) {
	log.Printf("main: starting HTTP server")

	httpServerExitDone := &sync.WaitGroup{}

	httpServerExitDone.Add(1)
	srv := startHttpServer(httpServerExitDone, settings)

	log.Printf("main: serving for 120 seconds")

	time.Sleep(120 * time.Second)

	log.Printf("main: stopping HTTP server")

	if err := srv.Shutdown(context.TODO()); err != nil {
		log.Printf("⚠️ Failed to stop Notion OAuth listener: %v", err)
	}

	httpServerExitDone.Wait()
	log.Printf("main: done. exiting")
}

// startHttpServer starts listener for redirect from api.jamesonzeller.com for Notion code, and then it handles it
// by converting to token and then saving it.
func startHttpServer(wg *sync.WaitGroup, s *settingsservice.SettingsService) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/oauth", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing code", http.StatusBadRequest)
			s.App.EmitEvent("Backend:NotionAccessToken", false)
			return
		}

		// Handle the code
		log.Println("Received OAuth code:", code)

		token, err := exchangeCodeForToken(code)
		if err != nil {
			http.Error(w, "Token exchange failed: "+err.Error(), http.StatusInternalServerError)
			log.Println("Token exchange failed:", err)
			s.App.EmitEvent("Backend:NotionAccessToken", false)
			return
		}

		// Handle the token, saving etc.
		err = s.SaveNotionToken(token.AccessToken)
		if err != nil {
			s.App.EmitEvent("Backend:NotionAccessToken", false)
			return
		}
		s.AppSettings.NotionAccessToken = token.AccessToken

		// Derive stable Notion id (bot id) and set as current user id for this session
		if id, err := fetchNotionBotID(token.AccessToken); err != nil {
			log.Printf("⚠️ Failed to fetch Notion bot id: %v", err)
			config.SetCurrentUserId("")
		} else {
			config.SetCurrentUserId(id)
		}

		// Emit event to notify frontend to refresh
		s.App.EmitEvent("Backend:NotionAccessToken", true)

		// Respond to user
		fmt.Fprintln(w, "<html><body><h2>✅ Linked! You may close this tab.</h2></body></html>")
	})

	srv := &http.Server{
		Addr:    "localhost:5173",
		Handler: mux,
	}

	go func() {
		defer wg.Done()

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("⚠️ Notion OAuth listener exited unexpectedly: %v", err)
		}
	}()

	return srv
}

func fetchNotionBotID(accessToken string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.notion.com/v1/users/me", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Notion-Version", "2022-06-28")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("/v1/users/me failed: status %d, body: %s", resp.StatusCode, body)
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}

	idVal, ok := payload["id"].(string)
	if !ok || idVal == "" {
		return "", fmt.Errorf("no id in /v1/users/me response")
	}
	return idVal, nil
}

func exchangeCodeForToken(code string) (*NotionOAuthResponse, error) {
	clientID := config.GetEnv("NOTION_CLIENT_ID")
	clientSecret := config.GetEnv("NOTION_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("notion OAuth client credentials are not configured")
	}

	redirectURI := config.GetEnv("NOTION_REDIRECT_URI")
	if redirectURI == "" {
		redirectURI = "http://localhost:5173/oauth"
	}

	const notionAPIURL = "https://api.notion.com/v1/oauth/token"

	encoded := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))

	data := NotionOAuthRequest{
		GrantType:   "authorization_code",
		Code:        code,
		RedirectUri: redirectURI,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequest("POST", notionAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Basic "+encoded)

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to exchange token, status %d, body: %s", resp.StatusCode, body)
	}

	var tokenResp NotionOAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}
