package proxy

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/prorouter/prorouter/internal/adapters"
	"github.com/prorouter/prorouter/internal/database"
	"github.com/prorouter/prorouter/internal/models"
)

type Proxy struct {
	db       *database.DB
	adapters map[string]*adapters.ProviderAdapter
}

func NewProxy(db *database.DB) *Proxy {
	return &Proxy{
		db:       db,
		adapters: make(map[string]*adapters.ProviderAdapter),
	}
}

func (p *Proxy) RegisterAdapter(name string, a *adapters.ProviderAdapter) {
	p.adapters[name] = a
}

func (p *Proxy) HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"reading body: %s"}`, err), http.StatusBadRequest)
		return
	}

	var req struct {
		Model       string          `json:"model"`
		Messages    json.RawMessage `json:"messages"`
		Stream      bool            `json:"stream"`
		Temperature json.Number     `json:"temperature,omitempty"`
		MaxTokens   int             `json:"max_tokens,omitempty"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid request: %s"}`, err), http.StatusBadRequest)
		return
	}

	// Resolve provider and model
	providerName, modelName := resolveProviderModel(req.Model)
	adapter, ok := p.adapters[providerName]
	if !ok {
		http.Error(w, fmt.Sprintf(`{"error":"unknown provider: %s"}`, providerName), http.StatusNotFound)
		return
	}

	// Get API key info for audit
	apiKeyID := r.Context().Value("api_key_id").(string)

	// Transform request if needed
	reqBody := body
	if adapter.TransformRequest != nil {
		transformed, err := adapter.TransformRequest(body)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"transform failed: %s"}`, err), http.StatusInternalServerError)
			return
		}
		reqBody = transformed
	}

	start := time.Now()

	// Determine the API path based on provider
	apiPath := "/chat/completions"
	if providerName == "anthropic" {
		apiPath = "/messages"
	} else if providerName == "gemini" {
		apiPath = "/models/" + modelName + ":streamGenerateContent?alt=sse"
	}

	// Make request to provider
	resp, err := adapter.ProviderRequest("POST", apiPath, reqBody, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"provider error: %s"}`, err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	duration := time.Since(start).Milliseconds()

	// Handle streaming
	if req.Stream && resp.StatusCode == 200 {
		p.handleStreamingResponse(w, r, resp, adapter, apiKeyID, modelName, providerName, duration, start)
		return
	}

	// Handle non-streaming response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"reading response: %s"}`, err), http.StatusBadGateway)
		return
	}

	// For Anthropic, translate response back to OpenAI format
	if providerName == "anthropic" {
		respBody = translateAnthropicResponse(respBody)
	}

	// Log audit
	go p.logAudit(apiKeyID, modelName, providerName, duration, resp.StatusCode, false, respBody)

	// Copy response headers and status
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.Header().Set("X-ProRouter-Provider", providerName)
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}

func (p *Proxy) handleStreamingResponse(w http.ResponseWriter, r *http.Request, resp *http.Response,
	adapter *adapters.ProviderAdapter, apiKeyID, modelName, providerName string,
	duration int64, start time.Time) {

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming not supported"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-ProRouter-Provider", providerName)
	w.WriteHeader(http.StatusOK)

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)

	for scanner.Scan() {
		line := scanner.Bytes()

		// Pass through for OpenAI-compatible providers
		if adapter.TransformStreamChunk == nil {
			w.Write(line)
			w.Write([]byte("\n"))
			flusher.Flush()

			if bytes.Contains(line, []byte(`"finish_reason":"stop"`)) ||
				bytes.Contains(line, []byte(`"finish_reason":"length"`)) ||
				bytes.Equal(line, []byte("data: [DONE]")) {
				break
			}

			// Count tokens roughly
			if bytes.HasPrefix(line, []byte("data: ")) && !bytes.Equal(line, []byte("data: [DONE]")) {
			}
			continue
		}

		// Transform streaming chunks for non-OpenAI providers
		transformed, ok := adapter.TransformStreamChunk(line)
		if !ok {
			continue
		}

		w.Write(transformed)
		flusher.Flush()

		if bytes.Contains(transformed, []byte(`"finish_reason":"stop"`)) ||
			bytes.Contains(transformed, []byte(`"finish_reason":"length"`)) ||
			bytes.Equal(transformed, []byte("data: [DONE]\n\n")) {
			break
		}
	}

	duration = time.Since(start).Milliseconds()
	go p.logAudit(apiKeyID, modelName, providerName, duration, http.StatusOK, true, nil)
}

func (p *Proxy) HandleListModels(w http.ResponseWriter, r *http.Request) {
	type modelEntry struct {
		ID       string `json:"id"`
		Object   string `json:"object"`
		Created  int64  `json:"created"`
		OwnedBy  string `json:"owned_by"`
	}

	models := []modelEntry{}

	for name := range p.adapters {
		switch name {
		case "openai":
			models = append(models, modelEntry{
				ID: "gpt-4o", Object: "model", Created: time.Now().Unix(), OwnedBy: "openai",
			}, modelEntry{
				ID: "gpt-4o-mini", Object: "model", Created: time.Now().Unix(), OwnedBy: "openai",
			})
		case "anthropic":
			models = append(models, modelEntry{
				ID: "claude-3-5-sonnet-20241022", Object: "model", Created: time.Now().Unix(), OwnedBy: "anthropic",
			})
		case "gemini":
			models = append(models, modelEntry{
				ID: "gemini-1.5-pro", Object: "model", Created: time.Now().Unix(), OwnedBy: "google",
			})
		case "deepseek":
			models = append(models, modelEntry{
				ID: "deepseek-chat", Object: "model", Created: time.Now().Unix(), OwnedBy: "deepseek",
			})
		case "ollama":
			models = append(models, modelEntry{
				ID: "llama3", Object: "model", Created: time.Now().Unix(), OwnedBy: "ollama",
			}, modelEntry{
				ID: "mistral", Object: "model", Created: time.Now().Unix(), OwnedBy: "ollama",
			})
		}
	}

	resp := map[string]interface{}{
		"object": "list",
		"data":   models,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (p *Proxy) logAudit(apiKeyID, model, provider string, durationMs int64, statusCode int, streamed bool, respBody []byte) {
	log := models.AuditLog{
		ID:         uuid.New().String(),
		APIKeyID:   apiKeyID,
		Model:      model,
		Provider:   provider,
		DurationMs: durationMs,
		StatusCode: statusCode,
		Streamed:   streamed,
		CreatedAt:  time.Now(),
	}
	p.db.InsertAuditLog(log)
}

func resolveProviderModel(model string) (provider, modelName string) {
	// Model format: "provider/model" or just "model"
	parts := strings.SplitN(model, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}

	// Default mappings
	switch model {
	case "gpt-4o", "gpt-4o-mini", "gpt-5.2":
		return "openai", model
	case "claude-3-5-sonnet-20241022", "claude-3-opus", "claude-sonnet-4":
		return "anthropic", model
	case "gemini-1.5-pro", "gemini-1.5-flash":
		return "gemini", model
	case "deepseek-chat", "deepseek-coder":
		return "deepseek", model
	default:
		return "openai", model
	}
}

func translateAnthropicResponse(body []byte) []byte {
	var anthResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Model  string `json:"model"`
		Usage  struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &anthResp); err != nil {
		return body
	}

	var content string
	if len(anthResp.Content) > 0 {
		content = anthResp.Content[0].Text
	}

	openAIResp := map[string]interface{}{
		"id":      "chatcmpl-" + uuid.New().String()[:8],
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   anthResp.Model,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]string{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]int{
			"prompt_tokens":     anthResp.Usage.InputTokens,
			"completion_tokens": anthResp.Usage.OutputTokens,
			"total_tokens":      anthResp.Usage.InputTokens + anthResp.Usage.OutputTokens,
		},
	}

	result, _ := json.Marshal(openAIResp)
	return result
}

func APIKeyHash(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}
