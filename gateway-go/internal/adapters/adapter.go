package adapters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ProviderAdapter translates between OpenAI standard format and provider-specific formats.
type ProviderAdapter struct {
	Name    string
	BaseURL string
	APIKey  string

	// TransformRequest adapts the request body for the provider.
	// Return nil to use the standard OpenAI format as-is.
	TransformRequest func(body []byte) ([]byte, error)

	// TransformStreamChunk adapts a single SSE data line from the provider back to OpenAI format.
	// Return nil to pass through unchanged.
	TransformStreamChunk func(line []byte) ([]byte, bool)
}

func NewOpenAICompatible(name, baseURL, apiKey string) *ProviderAdapter {
	return &ProviderAdapter{
		Name:    name,
		BaseURL: baseURL,
		APIKey:  apiKey,
	}
}

func NewAnthropicAdapter(apiKey string) *ProviderAdapter {
	return &ProviderAdapter{
		Name:    "anthropic",
		BaseURL: "https://api.anthropic.com/v1",
		APIKey:  apiKey,
		TransformRequest: func(body []byte) ([]byte, error) {
			var req struct {
				Model       string          `json:"model"`
				Messages    []chatMsg       `json:"messages"`
				Stream      bool            `json:"stream"`
				MaxTokens   int             `json:"max_tokens"`
				Temperature float64         `json:"temperature,omitempty"`
				Tools       json.RawMessage `json:"tools,omitempty"`
			}
			if err := json.Unmarshal(body, &req); err != nil {
				return nil, err
			}

			if req.MaxTokens == 0 {
				req.MaxTokens = 4096
			}

			// Extract system prompt if present
			var systemPrompt string
			var filtered []chatMsg
			for _, m := range req.Messages {
				if m.Role == "system" {
					systemPrompt = m.Content
				} else {
					filtered = append(filtered, m)
				}
			}

			anthropicReq := struct {
				Model       string          `json:"model"`
				Messages    []chatMsg       `json:"messages"`
				System      string          `json:"system,omitempty"`
				Stream      bool            `json:"stream"`
				MaxTokens   int             `json:"max_tokens"`
				Temperature float64         `json:"temperature,omitempty"`
				Tools       json.RawMessage `json:"tools,omitempty"`
			}{
				Model:       req.Model,
				Messages:    filtered,
				System:      systemPrompt,
				Stream:      req.Stream,
				MaxTokens:   req.MaxTokens,
				Temperature: req.Temperature,
				Tools:       req.Tools,
			}

			return json.Marshal(anthropicReq)
		},
		TransformStreamChunk: func(line []byte) ([]byte, bool) {
			if !bytes.HasPrefix(line, []byte("data: ")) {
				return nil, false
			}

			data := bytes.TrimPrefix(line, []byte("data: "))
			if string(data) == "[DONE]" {
				return []byte("data: [DONE]\n\n"), true
			}

			var event struct {
				Type  string `json:"type"`
				Delta struct {
					Text string `json:"text"`
				} `json:"delta,omitempty"`
				ContentBlock struct {
					Text string `json:"text"`
				} `json:"content_block,omitempty"`
			}
			if err := json.Unmarshal(data, &event); err != nil {
				return nil, false
			}

			switch event.Type {
			case "content_block_delta":
				if event.Delta.Text != "" {
					openAI := fmt.Sprintf(`{"choices":[{"index":0,"delta":{"content":"%s"},"finish_reason":null}]}`, jsonEscape(event.Delta.Text))
					return []byte("data: " + openAI + "\n\n"), true
				}
			case "message_stop":
				openAI := `{"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`
				return []byte("data: " + openAI + "\n\n"), true
			}

			return nil, false
		},
	}
}

func NewGeminiAdapter(apiKey string) *ProviderAdapter {
	return &ProviderAdapter{
		Name:    "gemini",
		BaseURL: "https://generativelanguage.googleapis.com/v1beta",
		APIKey:  apiKey,
		TransformRequest: func(body []byte) ([]byte, error) {
			var req struct {
				Model    string    `json:"model"`
				Messages []chatMsg `json:"messages"`
			}
			if err := json.Unmarshal(body, &req); err != nil {
				return nil, err
			}

			var contents []map[string]interface{}
			for _, m := range req.Messages {
				parts := []map[string]interface{}{
					{"text": m.Content},
				}
				contents = append(contents, map[string]interface{}{
					"role":  m.Role,
					"parts": parts,
				})
			}

			geminiReq := map[string]interface{}{
				"contents": contents,
			}

			return json.Marshal(geminiReq)
		},
	}
}

func NewMistralAdapter(apiKey string) *ProviderAdapter {
	return &ProviderAdapter{
		Name:    "mistral",
		BaseURL: "https://api.mistral.ai/v1",
		APIKey:  apiKey,
	}
}

func NewCohereAdapter(apiKey string) *ProviderAdapter {
	return &ProviderAdapter{
		Name:    "cohere",
		BaseURL: "https://api.cohere.com/v2",
		APIKey:  apiKey,
	}
}

func NewTogetherAdapter(apiKey string) *ProviderAdapter {
	return &ProviderAdapter{
		Name:    "together",
		BaseURL: "https://api.together.xyz/v1",
		APIKey:  apiKey,
	}
}

func NewGroqAdapter(apiKey string) *ProviderAdapter {
	return &ProviderAdapter{
		Name:    "groq",
		BaseURL: "https://api.groq.com/openai/v1",
		APIKey:  apiKey,
	}
}

func NewPerplexityAdapter(apiKey string) *ProviderAdapter {
	return &ProviderAdapter{
		Name:    "perplexity",
		BaseURL: "https://api.perplexity.ai",
		APIKey:  apiKey,
	}
}

func NewFireworksAdapter(apiKey string) *ProviderAdapter {
	return &ProviderAdapter{
		Name:    "fireworks",
		BaseURL: "https://api.fireworks.ai/inference/v1",
		APIKey:  apiKey,
	}
}

func NewDeepInfraAdapter(apiKey string) *ProviderAdapter {
	return &ProviderAdapter{
		Name:    "deepinfra",
		BaseURL: "https://api.deepinfra.com/v1/openai",
		APIKey:  apiKey,
	}
}

func NewReplicateAdapter(apiKey string) *ProviderAdapter {
	return &ProviderAdapter{
		Name:    "replicate",
		BaseURL: "https://api.replicate.com/v1",
		APIKey:  apiKey,
	}
}

func NewXiaomiAdapter(apiKey string) *ProviderAdapter {
	return &ProviderAdapter{
		Name:    "xiaomi",
		BaseURL: "https://api.azure.cn/v1/openai",
		APIKey:  apiKey,
	}
}

type chatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	return string(b[1 : len(b)-1])
}

// ProviderRequest sends a request to the provider and returns the response.
func (p *ProviderAdapter) ProviderRequest(method, path string, body []byte, headers map[string]string) (*http.Response, error) {
	url := p.BaseURL + path
	if p.Name == "gemini" {
		// Gemini uses query param for API key
		url += "?key=" + p.APIKey
	}

	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set auth header (Gemini uses query param instead)
	if p.Name != "gemini" {
		switch p.Name {
		case "anthropic":
			req.Header.Set("x-api-key", p.APIKey)
			req.Header.Set("anthropic-version", "2023-06-01")
		default:
			req.Header.Set("Authorization", "Bearer "+p.APIKey)
		}
	}

	if p.Name == "deepinfra" || p.Name == "replicate" {
		req.Header.Set("User-Agent", "ProRouter/0.1")
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("provider request failed: %w", err)
	}

	return resp, nil
}
