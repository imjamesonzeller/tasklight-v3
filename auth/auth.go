package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	s "github.com/imjamesonzeller/tasklight-v3/settingsservice"
)

const (
	apiBaseURL   = "https://api.jamesonzeller.com"
	userIDKey    = "user_id"
	authTokenKey = "auth_token"
)

var tokenSecret string

func Init(secret string) {
	if secret == "" {
		panic("TASKLIGHT_TOKEN_SECRET is not set. Cannot compute HMAC.")
	}
	tokenSecret = secret
}

func computeAuthToken(userID string) string {
	mac := hmac.New(sha256.New, []byte(tokenSecret))
	mac.Write([]byte(userID))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

type Identity struct {
	UserID    string
	AuthToken string
}

func LoadOrRegisterIdentity() (*Identity, error) {
	userID, _ := s.LoadSecret(userIDKey)
	authToken, _ := s.LoadSecret(authTokenKey)

	if userID != "" && authToken != "" {
		return &Identity{UserID: userID, AuthToken: authToken}, nil
	}

	userID = uuid.New().String()
	authToken = computeAuthToken(userID)

	if err := registerWithBackend(userID, authToken); err != nil {
		return nil, fmt.Errorf("failed to register with backend: %v", err)
	}

	_ = s.UpdateSecret(userIDKey, userID)
	_ = s.UpdateSecret(authTokenKey, authToken)

	return &Identity{UserID: userID, AuthToken: authToken}, nil
}

func registerWithBackend(userID string, authToken string) error {
	body := map[string]string{
		"user_id":    userID,
		"auth_token": authToken,
	}

	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(apiBaseURL+"/register", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registering with backend returned status %d: %s", resp.StatusCode, respBody)
	}

	return nil
}
