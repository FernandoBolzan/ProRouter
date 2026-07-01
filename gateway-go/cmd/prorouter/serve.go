package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/FernandoBolzan/ProRouter/internal/adapters"
	"github.com/FernandoBolzan/ProRouter/internal/config"
	"github.com/FernandoBolzan/ProRouter/internal/dashboard"
	"github.com/FernandoBolzan/ProRouter/internal/database"
	"github.com/FernandoBolzan/ProRouter/internal/middleware"
	"github.com/FernandoBolzan/ProRouter/internal/models"
	"github.com/FernandoBolzan/ProRouter/internal/oauth"
	"github.com/FernandoBolzan/ProRouter/internal/proxy"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the ProRouter gateway server",
	Long: `Starts the ProRouter HTTP gateway with OpenAI-compatible API,
dashboard, and all configured provider adapters.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		configPath, _ := cmd.Flags().GetString("config")
		host, _ := cmd.Flags().GetString("host")

		// Load config
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		if host != "" {
			cfg.Server.Host = host
		}
		if port != 0 {
			cfg.Server.Port = port
		}

		// Open database
		db, err := database.Open(cfg.Database.Path)
		if err != nil {
			return fmt.Errorf("opening database: %w", err)
		}
		defer db.Close()

		// Setup proxy
		p := proxy.NewProxy(db)
		setupAdapters(cfg, p)
		loadDBProviders(db, p)

		addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		startTokenRefresher(db)
		server := &http.Server{
			Addr:    addr,
			Handler: middleware.CORS(mainRouter(db, p)),
		}

		// Graceful shutdown
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			log.Printf("ProRouter gateway listening on %s", addr)
			log.Printf("OpenAI API: http://%s/v1", addr)
			log.Printf("Dashboard:  http://%s/dashboard/", addr)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Server error: %v", err)
			}
		}()

		<-quit
		log.Println("Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	},
}

func mainRouter(db *database.DB, p *proxy.Proxy) http.Handler {
	bypassPaths := []string{
		"/", "/dashboard/", "/dashboard/index.html",
		"/api/stats", "/api/keys", "/api/providers", "/api/logs",
		"/api/recipes", "/api/playground",
		"/api/oauth/",
	}

	root := http.NewServeMux()

	// Dashboard routes (no auth)
	root.Handle("/dashboard/", http.StripPrefix("/dashboard/", dashboard.Handler()))
	root.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/dashboard/", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})

	// API routes (no auth)
	root.HandleFunc("/api/stats", handleStats(db))
	root.HandleFunc("/api/keys", handleKeys(db))
	root.HandleFunc("/api/keys/", handleKeyByID(db))
	root.HandleFunc("/api/providers/test", handleProviderTest())
	root.HandleFunc("/api/providers", handleProviders(db))
	root.HandleFunc("/api/logs", handleLogs(db))
	root.HandleFunc("/api/recipes", handleRecipes(db))
	root.HandleFunc("/api/playground", handlePlayground(db, p))
	root.HandleFunc("/api/oauth/google", handleGoogleOAuth(db))
	root.HandleFunc("/api/oauth/", handleOAuthRouter(db))

	// Proxy routes (with auth middleware)
	proxyMux := http.NewServeMux()
	proxyMux.HandleFunc("/v1/chat/completions", p.HandleChatCompletions)
	proxyMux.HandleFunc("/v1/models", p.HandleListModels)
	authHandler := middleware.AuthMiddleware(db, bypassPaths)(proxyMux)
	root.Handle("/v1/", authHandler)

	return root
}

func setupAdapters(cfg *config.Config, p *proxy.Proxy) {
	// OpenAI
	p.RegisterAdapter("openai", adapters.NewOpenAICompatible("openai", "https://api.openai.com/v1", cfg.Providers.OpenAI.APIKey))
	log.Printf("  ✓ Provider: OpenAI (%s)", keyStatus(cfg.Providers.OpenAI.APIKey))

	// DeepSeek
	p.RegisterAdapter("deepseek", adapters.NewOpenAICompatible("deepseek", "https://api.deepseek.com/v1", cfg.Providers.DeepSeek.APIKey))
	log.Printf("  ✓ Provider: DeepSeek (%s)", keyStatus(cfg.Providers.DeepSeek.APIKey))

	// Anthropic
	p.RegisterAdapter("anthropic", adapters.NewAnthropicAdapter(cfg.Providers.Anthropic.APIKey))
	log.Printf("  ✓ Provider: Anthropic (%s)", keyStatus(cfg.Providers.Anthropic.APIKey))

	// Gemini
	p.RegisterAdapter("gemini", adapters.NewGeminiAdapter(cfg.Providers.Google.APIKey))
	log.Printf("  ✓ Provider: Google Gemini (%s)", keyStatus(cfg.Providers.Google.APIKey))

	// Ollama (local)
	p.RegisterAdapter("ollama", adapters.NewOpenAICompatible("ollama", "http://localhost:11434/v1", ""))
	log.Println("  ✓ Provider: Ollama (local)")
}

func keyStatus(key string) string {
	if key != "" {
		return "key set"
	}
	return "no key (use env var)"
}

// Dashboard API Handlers

func handleStats(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := db.GetStats()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, stats)
	}
}

func handleKeys(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			keys, err := db.ListAPIKeys()
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, keys)
		case "POST":
			var req struct {
				Name         string  `json:"name"`
				MonthlyBudget float64 `json:"monthly_budget"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
				return
			}
			if req.Name == "" {
				req.Name = "default"
			}

			// Generate key
			bytes := make([]byte, 32)
			rand.Read(bytes)
			token := hex.EncodeToString(bytes)
			hash := sha256.Sum256([]byte(token))
			keyHash := hex.EncodeToString(hash[:])

			key := models.APIKey{
				ID:            uuid.New().String(),
				Name:          req.Name,
				KeyPrefix:     token[:12],
				KeyHash:       keyHash,
				MonthlyBudget: req.MonthlyBudget,
				CreatedAt:     time.Now(),
			}

			if err := db.CreateAPIKey(key); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}

			writeJSON(w, http.StatusCreated, map[string]interface{}{
				"key":  "pr-" + token,
				"hash": keyHash,
				"id":   key.ID,
			})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
	}
}

func handleKeyByID(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/keys/")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing key id"})
			return
		}
		if r.Method == "DELETE" {
			if err := db.RevokeAPIKey(id); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
		} else {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
	}
}

func handleProviders(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			id := r.URL.Query().Get("id")
			if id != "" {
				config, err := db.GetProviderConfig(id)
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
					return
				}
				if config == nil {
					writeJSON(w, http.StatusNotFound, map[string]string{"error": "provider not found"})
					return
				}
				config.APIKeyEncrypted = maskKey(config.APIKeyEncrypted)
				writeJSON(w, http.StatusOK, config)
				return
			}
			configs, err := db.ListProviderConfigs()
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			for i := range configs {
				configs[i].APIKeyEncrypted = maskKey(configs[i].APIKeyEncrypted)
			}
			writeJSON(w, http.StatusOK, configs)

		case "POST":
			var req struct {
				ID       string `json:"id"`
				Provider string `json:"provider"`
				Label    string `json:"label"`
				BaseURL  string `json:"base_url"`
				APIKey   string `json:"api_key"`
				Models   string `json:"models"`
				IsActive *bool  `json:"is_active"`
				Priority int    `json:"priority"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
				return
			}
			if req.Provider == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "provider type is required"})
				return
			}
			if req.ID == "" {
				req.ID = uuid.New().String()
			}
			isActive := true
			if req.IsActive != nil {
				isActive = *req.IsActive
			}
			config := models.ProviderConfig{
				ID:              req.ID,
				Provider:        models.ProviderType(req.Provider),
				Label:           req.Label,
				BaseURL:         req.BaseURL,
				APIKeyEncrypted: req.APIKey,
				Models:          req.Models,
				IsActive:        isActive,
				Priority:        req.Priority,
			}
			if err := db.UpsertProviderConfig(config); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			config.APIKeyEncrypted = maskKey(config.APIKeyEncrypted)
			writeJSON(w, http.StatusOK, config)

		case "DELETE":
			id := r.URL.Query().Get("id")
			if id == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing id query parameter"})
				return
			}
			if err := db.DeleteProviderConfig(id); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})

		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
	}
}

func handleProviderTest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "use POST"})
			return
		}

		var req struct {
			Provider string `json:"provider"`
			BaseURL  string `json:"base_url"`
			APIKey   string `json:"api_key"`
			Model    string `json:"model,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if req.Provider == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "provider type is required"})
			return
		}

		client := &http.Client{Timeout: 5 * time.Second}
		start := time.Now()

		var httpReq *http.Request
		var err error

		switch req.Provider {
		case "openai-compatible":
			url := strings.TrimRight(req.BaseURL, "/") + "/models"
			httpReq, err = http.NewRequest("GET", url, nil)
			if err == nil && req.APIKey != "" {
				httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
			}
		case "anthropic":
			baseURL := strings.TrimRight(req.BaseURL, "/")
			if baseURL == "" {
				baseURL = "https://api.anthropic.com/v1"
			}
			body := `{"model":"claude-3-5-sonnet-20241022","max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`
			httpReq, err = http.NewRequest("POST", baseURL+"/messages", strings.NewReader(body))
			if err == nil {
				httpReq.Header.Set("x-api-key", req.APIKey)
				httpReq.Header.Set("anthropic-version", "2023-06-01")
				httpReq.Header.Set("Content-Type", "application/json")
			}
		case "gemini":
			baseURL := strings.TrimRight(req.BaseURL, "/")
			if baseURL == "" {
				baseURL = "https://generativelanguage.googleapis.com/v1beta"
			}
			httpReq, err = http.NewRequest("GET", baseURL+"/models?key="+req.APIKey, nil)
		case "ollama":
			baseURL := strings.TrimRight(req.BaseURL, "/")
			if baseURL == "" {
				baseURL = "http://localhost:11434"
			}
			httpReq, err = http.NewRequest("GET", baseURL+"/api/tags", nil)
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported provider type"})
			return
		}

		if err != nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"success": false,
				"message": fmt.Sprintf("request error: %v", err),
			})
			return
		}

		resp, err := client.Do(httpReq)
		latency := time.Since(start).Milliseconds()
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"success": false,
				"message": fmt.Sprintf("connection failed: %v", err),
			})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"success":    true,
				"message":    "Connection OK",
				"latency_ms": latency,
			})
			return
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyStr := strings.TrimSpace(string(bodyBytes))
		if len(bodyStr) > 200 {
			bodyStr = bodyStr[:200] + "..."
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("HTTP %d: %s", resp.StatusCode, bodyStr),
		})
	}
}

func loadDBProviders(db *database.DB, p *proxy.Proxy) {
	configs, err := db.ListProviderConfigs()
	if err != nil {
		log.Printf("Error loading providers from DB: %v", err)
		return
	}
	for _, c := range configs {
		if !c.IsActive {
			continue
		}
		cred := c.APIKeyEncrypted
		authType := "api_key"
		if c.AuthType == models.AuthTypeOAuth {
			cred = c.AccessToken
			authType = "oauth"
		}
		switch c.Provider {
		case models.ProviderOpenAI:
			p.RegisterAdapter(c.ID, adapters.NewOpenAICompatible(c.Label, c.BaseURL, cred))
			log.Printf("  ✓ Provider (DB): %s (%s, %s)", c.Label, keyStatus(cred), authType)
		case models.ProviderAnthropic:
			p.RegisterAdapter(c.ID, adapters.NewAnthropicAdapter(cred))
			log.Printf("  ✓ Provider (DB): %s (%s, %s)", c.Label, keyStatus(cred), authType)
		case models.ProviderGemini:
			p.RegisterAdapter(c.ID, adapters.NewGeminiAdapter(cred))
			log.Printf("  ✓ Provider (DB): %s (%s, %s)", c.Label, keyStatus(cred), authType)
		case models.ProviderOllama:
			p.RegisterAdapter(c.ID, adapters.NewOpenAICompatible(c.Label, c.BaseURL, ""))
			log.Printf("  ✓ Provider (DB): %s (no key)", c.Label)
		default:
			log.Printf("  ? Provider (DB): %s (%s) - unknown type, registering as OpenAI-compatible", c.Label, c.Provider)
			p.RegisterAdapter(c.ID, adapters.NewOpenAICompatible(c.Label, c.BaseURL, cred))
		}
	}
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return key
	}
	return key[:8] + "..."
}

func handleLogs(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logs, err := db.GetAuditLogs(100)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, logs)
	}
}

func handleRecipes(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Return empty list for now (recipe engine not yet implemented)
		writeJSON(w, http.StatusOK, []models.Recipe{})
	}
}

func handlePlayground(db *database.DB, p *proxy.Proxy) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("Playground panic: %v", rec)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("internal error: %v", rec)})
			}
		}()
		if r.Method != "POST" {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "use POST"})
			return
		}
		// For playground, we make a direct request using a fake API key context
		ctx := context.WithValue(r.Context(), middleware.APIKeyIDKey, "playground")
		p.HandleChatCompletions(w, r.WithContext(ctx))
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func handleGoogleOAuthCallback(w http.ResponseWriter, r *http.Request, db *database.DB, clientID string) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")

	if errorParam != "" {
		w.Write([]byte(fmt.Sprintf(`<html><body><h2>OAuth Error</h2><p>%s</p><p>You can close this window.</p></body></html>`, errorParam)))
		return
	}

	clientSecret := os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET")
	if clientSecret == "" {
		clientSecret = os.Getenv("GEMINI_CLIENT_SECRET")
	}

	redirectURI := fmt.Sprintf("http://%s/api/oauth/google", r.Host)
	tokenURL := "https://oauth2.googleapis.com/token"
	tokenBody := fmt.Sprintf(
		"code=%s&client_id=%s&client_secret=%s&redirect_uri=%s&grant_type=authorization_code",
		code, clientID, clientSecret, redirectURI,
	)

	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", bytes.NewBufferString(tokenBody))
	if err != nil {
		w.Write([]byte(fmt.Sprintf(`<html><body><h2>Error</h2><p>Token exchange failed: %s</p></body></html>`, err)))
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token,omitempty"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil || tokenResp.AccessToken == "" {
		w.Write([]byte(fmt.Sprintf(`<html><body><h2>Error</h2><p>Token exchange failed: %s</p></body></html>`, string(body))))
		return
	}

	var expiresAt *time.Time
	if tokenResp.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	config := models.ProviderConfig{
		ID:             uuid.New().String(),
		Provider:       models.ProviderGemini,
		Label:          "gemini",
		AuthType:       models.AuthTypeOAuth,
		AccessToken:    tokenResp.AccessToken,
		RefreshToken:   tokenResp.RefreshToken,
		TokenExpiresAt: expiresAt,
		ProviderMeta:   "{}",
		IsActive:       true,
		BaseURL:        "https://generativelanguage.googleapis.com/v1beta",
	}

	if err := db.UpsertProviderConfig(config); err != nil {
		w.Write([]byte(fmt.Sprintf(`<html><body><h2>Error</h2><p>Failed to save provider: %s</p></body></html>`, err)))
		return
	}

	_ = state
	w.Write([]byte(`<html><body><h2>Connected!</h2><p>Gemini provider connected via Google OAuth successfully.</p><script>window.close();</script></body></html>`))
}

func handleGoogleOAuth(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientID := os.Getenv("GOOGLE_OAUTH_CLIENT_ID")
		if clientID == "" {
			clientID = os.Getenv("GEMINI_CLIENT_ID")
		}

		switch r.Method {
		case "GET":
			// Handle OAuth callback (code in query params)
			code := r.URL.Query().Get("code")
			if code != "" {
				handleGoogleOAuthCallback(w, r, db, clientID)
				return
			}

			// Return OAuth config status
			if clientID == "" {
				writeJSON(w, http.StatusOK, map[string]interface{}{
					"configured": false,
					"message":    "Google OAuth not configured. Set GOOGLE_OAUTH_CLIENT_ID and GOOGLE_OAUTH_CLIENT_SECRET env vars.",
				})
				return
			}

			redirectURI := fmt.Sprintf("http://%s/api/oauth/google", r.Host)
			authURL := fmt.Sprintf(
				"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=https://www.googleapis.com/auth/cloud-platform&access_type=offline&prompt=consent",
				clientID, redirectURI,
			)
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"configured":  true,
				"auth_url":    authURL,
				"redirect_uri": redirectURI,
			})

		case "POST":
			// Handle OAuth callback (code exchange)
			var req struct {
				Code string `json:"code"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
				return
			}
			if req.Code == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "code is required"})
				return
			}

			clientSecret := os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET")
			if clientSecret == "" {
				clientSecret = os.Getenv("GEMINI_CLIENT_SECRET")
			}
			if clientID == "" || clientSecret == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "OAuth not configured on server"})
				return
			}

			redirectURI := fmt.Sprintf("http://%s/api/oauth/google", r.Host)
			tokenURL := "https://oauth2.googleapis.com/token"
			tokenBody := fmt.Sprintf(
				"code=%s&client_id=%s&client_secret=%s&redirect_uri=%s&grant_type=authorization_code",
				req.Code, clientID, clientSecret, redirectURI,
			)

			resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", bytes.NewBufferString(tokenBody))
			if err != nil {
				writeJSON(w, http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("token exchange failed: %s", err)})
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)

			var tokenResp struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token,omitempty"`
				ExpiresIn    int    `json:"expires_in"`
			}
			if err := json.Unmarshal(body, &tokenResp); err != nil || tokenResp.AccessToken == "" {
				writeJSON(w, http.StatusBadGateway, map[string]string{"error": "token exchange failed", "detail": string(body)})
				return
			}

			// Store as a provider connection
			var expiresAt *time.Time
			if tokenResp.ExpiresIn > 0 {
				t := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
				expiresAt = &t
			}

			config := models.ProviderConfig{
				ID:            uuid.New().String(),
				Provider:      models.ProviderGemini,
				Label:         "gemini",
				AuthType:      models.AuthTypeOAuth,
				AccessToken:   tokenResp.AccessToken,
				RefreshToken:  tokenResp.RefreshToken,
				TokenExpiresAt: expiresAt,
				ProviderMeta:  "{}",
				IsActive:      true,
				BaseURL:       "https://generativelanguage.googleapis.com/v1beta",
			}

			var connErr error
			if db != nil {
				connErr = db.UpsertProviderConfig(config)
			}

			if connErr != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("failed to save provider: %s", connErr)})
				return
			}

			log.Printf("Google OAuth successful — provider saved (expires in %ds)", tokenResp.ExpiresIn)
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"success":       true,
				"message":       "Google OAuth successful! Gemini provider connected.",
				"expires_in":    tokenResp.ExpiresIn,
				"has_refresh":   tokenResp.RefreshToken != "",
			})

		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	}
}

func handleOAuthRouter(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/oauth/")
		parts := strings.Split(path, "/")
		if len(parts) < 1 || parts[0] == "" {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown OAuth endpoint"})
			return
		}
		provider := parts[0]
		action := ""
		if len(parts) >= 2 {
			action = parts[1]
		}

		// Google/Gemini OAuth uses the existing handler
		if provider == "google" {
			handleGoogleOAuth(db)(w, r)
			return
		}

		cfg, err := oauth.GetProviderConfig(provider)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}

		switch action {
		case "authorize":
			handleOAuthAuthorize(w, r, cfg, db)
		case "exchange":
			handleOAuthExchange(w, r, cfg, db)
		case "callback":
			handleOAuthCallback(w, r, cfg, db)
		case "poll":
			handleOAuthPoll(w, r)
		case "refresh":
			handleOAuthRefresh(w, r, cfg, db)
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown OAuth action"})
		}
	}
}

func handleOAuthAuthorize(w http.ResponseWriter, r *http.Request, cfg *oauth.ProviderConfig, db *database.DB) {
	session, err := oauth.CreateSession(cfg.Name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	redirectURI := fmt.Sprintf("http://%s%s", r.Host, cfg.CallbackPath)
	session.RedirectURI = redirectURI

	codeVerifier, codeChallenge, err := oauth.GeneratePKCE()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	session.CodeVerifier = codeVerifier

	authURL := oauth.BuildAuthURL(cfg, redirectURI, session.State, codeChallenge)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"state":       session.State,
		"auth_url":    authURL,
		"redirect_uri": redirectURI,
		"provider":    cfg.Name,
	})
}

func handleOAuthExchange(w http.ResponseWriter, r *http.Request, cfg *oauth.ProviderConfig, db *database.DB) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "use POST"})
		return
	}

	var req struct {
		Code         string `json:"code"`
		State        string `json:"state"`
		RedirectURI  string `json:"redirect_uri"`
		CodeVerifier string `json:"code_verifier"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	session := oauth.GetSession(req.State)
	if session == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid or expired session state"})
		return
	}

	if req.CodeVerifier == "" {
		req.CodeVerifier = session.CodeVerifier
	}

	tokens, err := oauth.ExchangeCode(cfg, req.Code, req.RedirectURI, req.CodeVerifier)
	if err != nil {
		oauth.CompleteSession(req.State, "", err.Error())
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	providerType := models.ProviderOpenAI
	if cfg.Name == "claude" {
		providerType = models.ProviderAnthropic
	}

	var expiresAt *time.Time
	if tokens.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	config := models.ProviderConfig{
		ID:           uuid.New().String(),
		Provider:     providerType,
		Label:        cfg.Name,
		AuthType:     models.AuthTypeOAuth,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		IDToken:      tokens.IDToken,
		TokenExpiresAt: expiresAt,
		ProviderMeta: "{}",
		IsActive:     true,
	}

	if cfg.Name == "codex" {
		config.BaseURL = "https://api.openai.com/v1"
	} else if cfg.Name == "claude" {
		config.BaseURL = "https://api.anthropic.com/v1"
	}

	if err := db.UpsertProviderConfig(config); err != nil {
		oauth.CompleteSession(req.State, "", err.Error())
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	oauth.CompleteSession(req.State, config.ID, "")

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"connection": map[string]interface{}{
			"id":       config.ID,
			"provider": config.Provider,
			"label":    config.Label,
			"auth_type": config.AuthType,
		},
	})
}

func handleOAuthCallback(w http.ResponseWriter, r *http.Request, cfg *oauth.ProviderConfig, db *database.DB) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")

	if errorParam != "" {
		w.Write([]byte(fmt.Sprintf(`<html><body><h2>OAuth Error</h2><p>%s</p><p>You can close this window.</p></body></html>`, errorParam)))
		return
	}

	if code == "" || state == "" {
		w.Write([]byte(`<html><body><h2>Error</h2><p>Missing authorization code or state.</p></body></html>`))
		return
	}

	session := oauth.GetSession(state)
	if session == nil {
		w.Write([]byte(`<html><body><h2>Error</h2><p>Invalid or expired session.</p></body></html>`))
		return
	}

	tokens, err := oauth.ExchangeCode(cfg, code, session.RedirectURI, session.CodeVerifier)
	if err != nil {
		oauth.CompleteSession(state, "", err.Error())
		w.Write([]byte(fmt.Sprintf(`<html><body><h2>Error</h2><p>%s</p></body></html>`, err.Error())))
		return
	}

	providerType := models.ProviderOpenAI
	if cfg.Name == "claude" {
		providerType = models.ProviderAnthropic
	}

	var expiresAt *time.Time
	if tokens.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	config := models.ProviderConfig{
		ID:            uuid.New().String(),
		Provider:      providerType,
		Label:         cfg.Name,
		AuthType:      models.AuthTypeOAuth,
		AccessToken:   tokens.AccessToken,
		RefreshToken:  tokens.RefreshToken,
		IDToken:       tokens.IDToken,
		TokenExpiresAt: expiresAt,
		ProviderMeta:  "{}",
		IsActive:      true,
	}

	if cfg.Name == "codex" {
		config.BaseURL = "https://api.openai.com/v1"
	} else if cfg.Name == "claude" {
		config.BaseURL = "https://api.anthropic.com/v1"
	}

	if err := db.UpsertProviderConfig(config); err != nil {
		oauth.CompleteSession(state, "", err.Error())
		w.Write([]byte(fmt.Sprintf(`<html><body><h2>Error</h2><p>%s</p></body></html>`, err.Error())))
		return
	}

	oauth.CompleteSession(state, config.ID, "")

	w.Write([]byte(`<html><body><h2>Connected!</h2><p>Provider connected successfully. You can close this window.</p><script>window.close();</script></body></html>`))
}

func handleOAuthPoll(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if state == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing state"})
		return
	}

	done, connID, errMsg := oauth.GetSessionResult(state)

	if done {
		if errMsg != "" {
			oauth.CleanupSession(state)
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"status": "error",
				"error":  errMsg,
			})
			return
		}
		oauth.CleanupSession(state)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":        "done",
			"connection_id": connID,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "pending",
	})
}

func handleOAuthRefresh(w http.ResponseWriter, r *http.Request, cfg *oauth.ProviderConfig, db *database.DB) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "use POST"})
		return
	}

	var req struct {
		ConnectionID string `json:"connection_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	config, err := db.GetProviderConfig(req.ConnectionID)
	if err != nil || config == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "connection not found"})
		return
	}

	if config.RefreshToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no refresh token available"})
		return
	}

	tokens, err := oauth.RefreshTokens(cfg, config.RefreshToken)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	var expiresAt *time.Time
	if tokens.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	if err := db.UpdateOAuthTokens(config.ID, tokens.AccessToken, tokens.RefreshToken, tokens.IDToken, expiresAt); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"expires_in": tokens.ExpiresIn,
	})
}

func startTokenRefresher(db *database.DB) {
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			refreshExpiringTokens(db)
		}
	}()
}

func refreshExpiringTokens(db *database.DB) {
	connections, err := db.ListOAuthProviders()
	if err != nil {
		return
	}

	for _, conn := range connections {
		if conn.RefreshToken == "" {
			continue
		}
		if conn.TokenExpiresAt == nil {
			continue
		}

		cfg, err := oauth.GetProviderConfig(string(conn.Provider))
		if err != nil {
			continue
		}

		lead := cfg.RefreshLeadDuration()
		if time.Until(*conn.TokenExpiresAt) > lead {
			continue
		}

		tokens, err := oauth.RefreshTokens(cfg, conn.RefreshToken)
		if err != nil {
			log.Printf("Token refresh failed for %s (%s): %v", conn.Label, conn.ID, err)
			continue
		}

		var expiresAt *time.Time
		if tokens.ExpiresIn > 0 {
			t := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
			expiresAt = &t
		}

		if err := db.UpdateOAuthTokens(conn.ID, tokens.AccessToken, tokens.RefreshToken, tokens.IDToken, expiresAt); err != nil {
			log.Printf("Token save failed for %s: %v", conn.ID, err)
			continue
		}

		log.Printf("Token refreshed for %s (%s), expires in %ds", conn.Label, conn.ID, tokens.ExpiresIn)
	}
}

func init() {
	serveCmd.Flags().IntP("port", "p", 0, "Port to listen on (overrides config)")
	serveCmd.Flags().StringP("config", "c", "", "Config file path")
	serveCmd.Flags().String("host", "", "Host to bind to (overrides config)")
	rootCmd.AddCommand(serveCmd)
}
