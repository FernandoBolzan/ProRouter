package models

import "time"

type ProviderType string

const (
	ProviderOpenAI    ProviderType = "openai-compatible"
	ProviderAnthropic ProviderType = "anthropic"
	ProviderGemini    ProviderType = "gemini"
	ProviderOllama    ProviderType = "ollama"
)

type APIKey struct {
	ID            string    `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	KeyPrefix     string    `json:"key_prefix" db:"key_prefix"`
	KeyHash       string    `json:"key_hash" db:"key_hash"`
	IsRevoked     bool      `json:"is_revoked" db:"is_revoked"`
	MonthlyBudget float64   `json:"monthly_budget" db:"monthly_budget"`
	MonthlySpent  float64   `json:"monthly_spent" db:"monthly_spent"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	LastUsedAt    *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
}

type AuditLog struct {
	ID              string    `json:"id" db:"id"`
	APIKeyID        string    `json:"api_key_id" db:"api_key_id"`
	Model           string    `json:"model" db:"model"`
	Provider        string    `json:"provider" db:"provider"`
	PromptTokens    int       `json:"prompt_tokens" db:"prompt_tokens"`
	CompletionTokens int      `json:"completion_tokens" db:"completion_tokens"`
	CachedTokens    int       `json:"cached_tokens" db:"cached_tokens"`
	DurationMs      int64     `json:"duration_ms" db:"duration_ms"`
	CostUSD         float64   `json:"cost_usd" db:"cost_usd"`
	StatusCode      int       `json:"status_code" db:"status_code"`
	Streamed        bool      `json:"streamed" db:"streamed"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

type ProviderConfig struct {
	ID        string       `json:"id" db:"id"`
	Provider  ProviderType `json:"provider" db:"provider"`
	Label     string       `json:"label" db:"label"`
	BaseURL   string       `json:"base_url" db:"base_url"`
	Models    string       `json:"models" db:"models"` // JSON array
	IsActive  bool         `json:"is_active" db:"is_active"`
	Priority  int          `json:"priority" db:"priority"`
}

type Recipe struct {
	ID           string `json:"id" db:"id"`
	Name         string `json:"name" db:"name"`
	PipelineJSON string `json:"pipeline_json" db:"pipeline_json"`
	IsActive     bool   `json:"is_active" db:"is_active"`
	IsDefault    bool   `json:"is_default" db:"is_default"`
}
