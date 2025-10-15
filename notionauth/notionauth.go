package notionauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/imjamesonzeller/tasklight-v3/config"
	"github.com/imjamesonzeller/tasklight-v3/notionapi"
	"github.com/imjamesonzeller/tasklight-v3/settingsservice"
)

const (
	clientHeaderName     = "X-Tasklight-Client"
	defaultRedirectURI   = "http://localhost:5173/oauth"
	oauthStartPath       = "/tasklight/notion/oauth/start"
	oauthCompletePath    = "/tasklight/notion/oauth/complete"
	listenerLifetime     = 120 * time.Second
	requestTimeout       = 10 * time.Second
	htmlSuccessResponse  = "<html><body><h2>✅ Linked! You may close this tab.</h2></body></html>"
	errUnsupportedScheme = "redirect URI scheme %q is not supported; expected http or https"
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

var errOAuthTokenMissing = errors.New("notion OAuth token missing")

// ResolveAuthorizeURL asks the Tasklight API for the Notion authorize URL.
func ResolveAuthorizeURL(apiBase, clientHeader string) (string, error) {
	startURL := strings.TrimRight(apiBase, "/") + oauthStartPath

	req, err := http.NewRequest(http.MethodGet, startURL, nil)
	if err != nil {
		return "", err
	}
	if clientHeader != "" {
		req.Header.Set(clientHeaderName, clientHeader)
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: requestTimeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode >= 300 && resp.StatusCode < 400:
		location := resp.Header.Get("Location")
		if location == "" {
			return "", fmt.Errorf("oauth start returned %d without Location header", resp.StatusCode)
		}
		return location, nil
	case resp.StatusCode == http.StatusOK:
		var payload struct {
			AuthURL string `json:"auth_url"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return "", err
		}
		if payload.AuthURL == "" {
			return "", fmt.Errorf("oauth start returned empty auth_url")
		}
		return payload.AuthURL, nil
	default:
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("oauth start failed: status %d, body: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}

// StartLocalOAuthListener starts http server and handles shutting it down.
func StartLocalOAuthListener(settings *settingsservice.SettingsService, apiBase, clientHeader string) {
	log.Printf("notionauth: starting callback listener")

	httpServerExitDone := &sync.WaitGroup{}
	httpServerExitDone.Add(1)

	srv, err := startHTTPServer(httpServerExitDone, settings, apiBase, clientHeader)
	if err != nil {
		log.Printf("⚠️ Failed to start Notion OAuth listener: %v", err)
		settings.App.EmitEvent("Backend:NotionAccessToken", false)
		return
	}

	time.Sleep(listenerLifetime)

	log.Printf("notionauth: stopping callback listener")

	if err := srv.Shutdown(context.TODO()); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("⚠️ Failed to stop Notion OAuth listener cleanly: %v", err)
	}

	httpServerExitDone.Wait()
	log.Printf("notionauth: listener stopped")
}

func startHTTPServer(wg *sync.WaitGroup, s *settingsservice.SettingsService, apiBase, clientHeader string) (*http.Server, error) {
	if clientHeader == "" {
		clientHeader = clientHeaderValue(s)
	}
	redirectURI := config.GetEnv("NOTION_REDIRECT_URI")
	if redirectURI == "" {
		redirectURI = defaultRedirectURI
	}

	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return nil, fmt.Errorf("invalid NOTION_REDIRECT_URI: %w", err)
	}

	switch parsed.Scheme {
	case "http", "https":
	default:
		return nil, fmt.Errorf(errUnsupportedScheme, parsed.Scheme)
	}

	path := parsed.Path
	if path == "" {
		path = "/"
	}

	addr := parsed.Host
	if addr == "" {
		addr = "localhost:5173"
	}

	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		handoff := strings.TrimSpace(r.URL.Query().Get("handoff"))
		if handoff == "" {
			http.Error(w, "Missing handoff", http.StatusBadRequest)
			s.App.EmitEvent("Backend:NotionAccessToken", false)
			return
		}

		log.Println("Received OAuth handoff:", handoff)

		token, err := completeHandoff(apiBase, handoff, clientHeader)
		if err != nil {
			http.Error(w, "Token exchange failed: "+err.Error(), http.StatusInternalServerError)
			log.Println("Token exchange failed:", err)
			s.App.EmitEvent("Backend:NotionAccessToken", false)
			return
		}

		if err := s.SaveNotionToken(token.AccessToken); err != nil {
			http.Error(w, "Failed to persist credentials", http.StatusInternalServerError)
			log.Println("Failed to save Notion token:", err)
			s.App.EmitEvent("Backend:NotionAccessToken", false)
			return
		}
		s.AppSettings.NotionAccessToken = token.AccessToken

		if token.BotID != "" {
			config.SetCurrentUserId(token.BotID)
		} else if id, err := fetchNotionBotID(token.AccessToken); err != nil {
			log.Printf("⚠️ Failed to fetch Notion bot id: %v", err)
			config.SetCurrentUserId("")
		} else {
			config.SetCurrentUserId(id)
		}

		s.App.EmitEvent("Backend:NotionAccessToken", true)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintln(w, htmlSuccessResponse)
	})

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		defer wg.Done()

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("⚠️ Notion OAuth listener exited unexpectedly: %v", err)
		}
	}()

	return srv, nil
}

func clientHeaderValue(settings *settingsservice.SettingsService) string {
	if settings == nil {
		return ""
	}
	version := strings.TrimSpace(settings.GetAppVersion())
	if version == "" {
		version = "development"
	}
	return fmt.Sprintf("tasklight-desktop/%s", version)
}

func completeHandoff(apiBase, handoff, clientHeader string) (*NotionOAuthResponse, error) {
	endpoint := strings.TrimRight(apiBase, "/") + oauthCompletePath

	payload := struct {
		Handoff string `json:"handoff"`
	}{
		Handoff: handoff,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if clientHeader != "" {
		req.Header.Set(clientHeaderName, clientHeader)
	}

	resp, err := (&http.Client{Timeout: requestTimeout}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("handoff completion failed: status %d, body: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tokenResp NotionOAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

func fetchNotionBotID(accessToken string) (string, error) {
	req, err := notionapi.NewRequest(http.MethodGet, "https://api.notion.com/v1/users/me", accessToken)
	if err != nil {
		return "", err
	}

	resp, err := (&http.Client{Timeout: requestTimeout}).Do(req)
	if err != nil {
		return "", err
	}

	var payload struct {
		ID string `json:"id"`
	}
	if err := notionapi.ParseResponse(resp, &payload, errOAuthTokenMissing); err != nil {
		return "", err
	}

	if payload.ID == "" {
		return "", fmt.Errorf("no id in /v1/users/me response")
	}
	return payload.ID, nil
}

// ClientHeader provides the identifier used for backend requests.
func ClientHeader(settings *settingsservice.SettingsService) string {
	return clientHeaderValue(settings)
}
