package notionauth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/imjamesonzeller/tasklight-v3/config"
	"github.com/imjamesonzeller/tasklight-v3/settingsservice"
	"io"
	"log"
	"net/http"
	//"net/url"
	"encoding/base64"
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

// StartLocalOAuthListener listens for redirect from api.jamesonzeller.com for Notion code,
// it then handles it and turns into OAuth token and saves it.
func StartLocalOAuthListener(s *settingsservice.SettingsService) {
	http.HandleFunc("/oauth", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing code", http.StatusBadRequest)
			return
		}

		// Handle the code
		log.Println("Received OAuth code:", code)

		token, err := exchangeCodeForToken(code)
		if err != nil {
			http.Error(w, "Token exchange failed: "+err.Error(), http.StatusInternalServerError)
			log.Println("Token exchange failed:", err)
			return
		}

		// Handle the token, saving etc.
		err = s.SaveNotionToken(token.AccessToken)
		if err != nil {
			return
		}

		// Respond to user
		fmt.Fprintln(w, "<html><body><h2>✅ Linked! You may close this tab.</h2></body></html>")
	})

	go func() {
		log.Println("Listening on http://localhost:5173/oauth...")
		err := http.ListenAndServe("localhost:5173", nil)
		if err != nil {
			log.Fatal("OAuth listener failed:", err)
		}
	}()
}

type NotionOAuthRequest struct {
	GrantType   string `json:"grant_type"`
	Code        string `json:"code"`
	RedirectUri string `json:"redirect_uri"`
}

func exchangeCodeForToken(code string) (*NotionOAuthResponse, error) {
	ClientId := config.GetEnv("OAUTH_CLIENT_ID")
	ClientSecret := config.GetEnv("OAUTH_CLIENT_SECRET")
	NotionAPIURL := "https://api.notion.com/v1/oauth/token"

	encoded := base64.StdEncoding.EncodeToString([]byte(ClientId + ":" + ClientSecret))

	data := NotionOAuthRequest{
		GrantType:   "authorization_code",
		Code:        code,
		RedirectUri: "https://api.jamesonzeller.com/callback",
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequest("POST", NotionAPIURL, bytes.NewBuffer(jsonData))
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
