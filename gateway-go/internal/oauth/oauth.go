package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type ProviderConfig struct {
	Name                string
	ClientID            string
	AuthorizeURL        string
	TokenURL            string
	RefreshURL          string
	Scope               string
	CodeChallengeMethod string
	ExtraParams         map[string]string
	TokenContentType    string
	UsePKCE             bool
	CallbackPath        string
	FixedPort           int
	RefreshLead         time.Duration
}

type OAuthSession struct {
	State         string
	CodeVerifier  string
	Provider      string
	RedirectURI   string
	CreatedAt     time.Time
	Done          bool
	ConnectionID  string
	Error         string
	mu            sync.Mutex
}

type OAuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
}

var claudeConfig = ProviderConfig{
	Name:                "claude",
	ClientID:            "9d1c250a-e61b-44d9-88ed-5944d1962f5e",
	AuthorizeURL:        "https://claude.ai/oauth/authorize",
	TokenURL:            "https://api.anthropic.com/v1/oauth/token",
	Scope:               "org:create_api_key user:profile user:inference",
	CodeChallengeMethod: "S256",
	TokenContentType:    "json",
	UsePKCE:             true,
	CallbackPath:        "/api/oauth/claude/callback",
	RefreshLead:         4 * time.Hour,
}

var codexConfig = ProviderConfig{
	Name:                "codex",
	ClientID:            "app_EMoamEEZ73f0CkXaXp7hrann",
	AuthorizeURL:        "https://auth.openai.com/oauth/authorize",
	TokenURL:            "https://auth.openai.com/oauth/token",
	Scope:               "openid profile email offline_access",
	CodeChallengeMethod: "S256",
	ExtraParams: map[string]string{
		"id_token_add_organizations": "true",
		"codex_cli_simplified_flow":  "true",
		"originator":                 "codex_cli_rs",
	},
	TokenContentType: "form",
	UsePKCE:          true,
	CallbackPath:     "/api/oauth/codex/callback",
	RefreshLead:      5 * 24 * time.Hour,
}

var (
	sessionsMu sync.Mutex
	sessions   = make(map[string]*OAuthSession)
)

func GetProviderConfig(name string) (*ProviderConfig, error) {
	switch name {
	case "claude":
		return &claudeConfig, nil
	case "codex":
		return &codexConfig, nil
	default:
		return nil, fmt.Errorf("unknown OAuth provider: %s", name)
	}
}

func GeneratePKCE() (codeVerifier, codeChallenge string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generating random bytes: %w", err)
	}
	codeVerifier = base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(codeVerifier))
	codeChallenge = base64.RawURLEncoding.EncodeToString(h[:])
	return codeVerifier, codeChallenge, nil
}

func BuildAuthURL(cfg *ProviderConfig, redirectURI, state, codeChallenge string) string {
	params := url.Values{}
	params.Set("client_id", cfg.ClientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", redirectURI)
	params.Set("state", state)
	if cfg.UsePKCE {
		params.Set("code_challenge", codeChallenge)
		params.Set("code_challenge_method", cfg.CodeChallengeMethod)
	}
	if cfg.Scope != "" {
		params.Set("scope", cfg.Scope)
	}
	for k, v := range cfg.ExtraParams {
		params.Set(k, v)
	}
	return cfg.AuthorizeURL + "?" + params.Encode()
}

func ExchangeCode(cfg *ProviderConfig, code, redirectURI, codeVerifier string) (*OAuthTokens, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", cfg.ClientID)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	if cfg.UsePKCE {
		data.Set("code_verifier", codeVerifier)
	}

	var body io.Reader
	var contentType string

	if cfg.TokenContentType == "json" {
		jsonBody := make(map[string]string)
		for k, v := range data {
			jsonBody[k] = v[0]
		}
		b, _ := json.Marshal(jsonBody)
		body = strings.NewReader(string(b))
		contentType = "application/json"
	} else {
		body = strings.NewReader(data.Encode())
		contentType = "application/x-www-form-urlencoded"
	}

	resp, err := http.Post(cfg.TokenURL, contentType, body)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading token response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("token exchange failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var tokens OAuthTokens
	if err := json.Unmarshal(respBody, &tokens); err != nil {
		return nil, fmt.Errorf("parsing token response: %w (body: %s)", err, string(respBody))
	}

	if tokens.AccessToken == "" {
		return nil, fmt.Errorf("no access_token in response: %s", string(respBody))
	}

	return &tokens, nil
}

func RefreshTokens(cfg *ProviderConfig, refreshToken string) (*OAuthTokens, error) {
	refreshURL := cfg.RefreshURL
	if refreshURL == "" {
		refreshURL = cfg.TokenURL
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", cfg.ClientID)
	data.Set("refresh_token", refreshToken)

	var body io.Reader
	var contentType string

	if cfg.TokenContentType == "json" {
		jsonBody := make(map[string]string)
		for k, v := range data {
			jsonBody[k] = v[0]
		}
		b, _ := json.Marshal(jsonBody)
		body = strings.NewReader(string(b))
		contentType = "application/json"
	} else {
		body = strings.NewReader(data.Encode())
		contentType = "application/x-www-form-urlencoded"
	}

	resp, err := http.Post(refreshURL, contentType, body)
	if err != nil {
		return nil, fmt.Errorf("token refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading refresh response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("token refresh failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var tokens OAuthTokens
	if err := json.Unmarshal(respBody, &tokens); err != nil {
		return nil, fmt.Errorf("parsing refresh response: %w", err)
	}

	return &tokens, nil
}

func CreateSession(provider string) (*OAuthSession, error) {
	codeVerifier, codeChallenge, err := GeneratePKCE()
	if err != nil {
		return nil, err
	}

	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("generating state: %w", err)
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	session := &OAuthSession{
		State:        state,
		CodeVerifier: codeVerifier,
		Provider:     provider,
		CreatedAt:    time.Now(),
	}
	_ = codeChallenge

	sessionsMu.Lock()
	sessions[state] = session
	sessionsMu.Unlock()

	return session, nil
}

func GetSession(state string) *OAuthSession {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	return sessions[state]
}

func CompleteSession(state, connectionID string, errMsg string) {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	if s, ok := sessions[state]; ok {
		s.mu.Lock()
		s.Done = true
		s.ConnectionID = connectionID
		s.Error = errMsg
		s.mu.Unlock()
	}
}

func CleanupSession(state string) {
	sessionsMu.Lock()
	delete(sessions, state)
	sessionsMu.Unlock()
}

func GetSessionDone(state string) bool {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	if s, ok := sessions[state]; ok {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.Done
	}
	return false
}

func GetSessionResult(state string) (done bool, connectionID string, errMsg string) {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	if s, ok := sessions[state]; ok {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.Done, s.ConnectionID, s.Error
	}
	return false, "", ""
}

func (p *ProviderConfig) RefreshLeadDuration() time.Duration {
	if p.RefreshLead > 0 {
		return p.RefreshLead
	}
	return 5 * time.Minute
}
