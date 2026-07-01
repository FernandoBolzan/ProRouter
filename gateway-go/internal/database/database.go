package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
	"github.com/FernandoBolzan/ProRouter/internal/models"
)

type DB struct {
	*sql.DB
}

func Open(dbPath string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)

	db := &DB{conn}

	// Enable WAL mode and set busy timeout
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA busy_timeout=5000")

	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return db, nil
}

func (db *DB) migrate() error {
	migrations := []struct {
		name string
		sql  string
	}{
		{
			"001_initial_schema",
			`CREATE TABLE IF NOT EXISTS api_keys (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL DEFAULT 'default',
				key_prefix TEXT NOT NULL,
				key_hash TEXT NOT NULL UNIQUE,
				is_revoked INTEGER NOT NULL DEFAULT 0,
				monthly_budget REAL NOT NULL DEFAULT 0,
				monthly_spent REAL NOT NULL DEFAULT 0,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				last_used_at DATETIME
			);
			CREATE TABLE IF NOT EXISTS audit_logs (
				id TEXT PRIMARY KEY,
				api_key_id TEXT NOT NULL,
				model TEXT NOT NULL,
				provider TEXT NOT NULL,
				prompt_tokens INTEGER NOT NULL DEFAULT 0,
				completion_tokens INTEGER NOT NULL DEFAULT 0,
				cached_tokens INTEGER NOT NULL DEFAULT 0,
				duration_ms INTEGER NOT NULL DEFAULT 0,
				cost_usd REAL NOT NULL DEFAULT 0,
				status_code INTEGER NOT NULL DEFAULT 0,
				streamed INTEGER NOT NULL DEFAULT 0,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			);
			CREATE TABLE IF NOT EXISTS provider_configs (
				id TEXT PRIMARY KEY,
				provider TEXT NOT NULL,
				label TEXT NOT NULL DEFAULT '',
				base_url TEXT NOT NULL DEFAULT '',
				models TEXT NOT NULL DEFAULT '[]',
				is_active INTEGER NOT NULL DEFAULT 1,
				priority INTEGER NOT NULL DEFAULT 0
			);
			CREATE TABLE IF NOT EXISTS recipes (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				pipeline_json TEXT NOT NULL DEFAULT '{}',
				is_active INTEGER NOT NULL DEFAULT 1,
				is_default INTEGER NOT NULL DEFAULT 0
			);
			CREATE TABLE IF NOT EXISTS _migrations (
				version TEXT PRIMARY KEY,
				applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			);
			CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at);
			CREATE INDEX IF NOT EXISTS idx_audit_logs_api_key ON audit_logs(api_key_id);`,
		},
		{
			"002_add_api_key_encrypted",
			`ALTER TABLE provider_configs ADD COLUMN api_key_encrypted TEXT NOT NULL DEFAULT '';`,
		},
		{
			"003_add_auth_type",
			`ALTER TABLE provider_configs ADD COLUMN auth_type TEXT NOT NULL DEFAULT 'api_key';`,
		},
		{
			"004_add_access_token",
			`ALTER TABLE provider_configs ADD COLUMN access_token TEXT NOT NULL DEFAULT '';`,
		},
		{
			"005_add_refresh_token",
			`ALTER TABLE provider_configs ADD COLUMN refresh_token TEXT NOT NULL DEFAULT '';`,
		},
		{
			"006_add_id_token",
			`ALTER TABLE provider_configs ADD COLUMN id_token TEXT NOT NULL DEFAULT '';`,
		},
		{
			"007_add_token_expires_at",
			`ALTER TABLE provider_configs ADD COLUMN token_expires_at DATETIME;`,
		},
		{
			"008_add_provider_meta",
			`ALTER TABLE provider_configs ADD COLUMN provider_meta TEXT NOT NULL DEFAULT '{}';`,
		},
	}

	for _, m := range migrations {
		var count int
		db.QueryRow("SELECT COUNT(*) FROM _migrations WHERE version = ?", m.name).Scan(&count)
		if count > 0 {
			continue
		}

		_, err := db.Exec(m.sql)
		if err != nil {
			return fmt.Errorf("migration %s failed: %w", m.name, err)
		}

		_, err = db.Exec("INSERT INTO _migrations (version) VALUES (?)", m.name)
		if err != nil {
			return fmt.Errorf("recording migration %s: %w", m.name, err)
		}
	}

	return nil
}

// API Key operations
func (db *DB) CreateAPIKey(key models.APIKey) error {
	_, err := db.Exec(
		`INSERT INTO api_keys (id, name, key_prefix, key_hash, is_revoked, monthly_budget, monthly_spent, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		key.ID, key.Name, key.KeyPrefix, key.KeyHash, boolToInt(key.IsRevoked),
		key.MonthlyBudget, key.MonthlySpent, key.CreatedAt,
	)
	return err
}

func (db *DB) GetAPIKeyByHash(hash string) (*models.APIKey, error) {
	var k models.APIKey
	var isRevoked int
	var lastUsed sql.NullTime
	err := db.QueryRow(
		`SELECT id, name, key_prefix, key_hash, is_revoked, monthly_budget, monthly_spent, created_at, last_used_at
		FROM api_keys WHERE key_hash = ?`, hash,
	).Scan(&k.ID, &k.Name, &k.KeyPrefix, &k.KeyHash, &isRevoked,
		&k.MonthlyBudget, &k.MonthlySpent, &k.CreatedAt, &lastUsed)
	if err != nil {
		return nil, err
	}
	k.IsRevoked = isRevoked == 1
	if lastUsed.Valid {
		k.LastUsedAt = &lastUsed.Time
	}
	return &k, nil
}

func (db *DB) ListAPIKeys() ([]models.APIKey, error) {
	keys := make([]models.APIKey, 0)
	rows, err := db.Query(
		`SELECT id, name, key_prefix, key_hash, is_revoked, monthly_budget, monthly_spent, created_at, last_used_at
		FROM api_keys ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var k models.APIKey
		var isRevoked int
		var lastUsed sql.NullTime
		if err := rows.Scan(&k.ID, &k.Name, &k.KeyPrefix, &k.KeyHash, &isRevoked,
			&k.MonthlyBudget, &k.MonthlySpent, &k.CreatedAt, &lastUsed); err != nil {
			return nil, err
		}
		k.IsRevoked = isRevoked == 1
		if lastUsed.Valid {
			k.LastUsedAt = &lastUsed.Time
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (db *DB) RevokeAPIKey(id string) error {
	_, err := db.Exec("UPDATE api_keys SET is_revoked = 1 WHERE id = ?", id)
	return err
}

func (db *DB) UpdateAPIKeyUsage(id string, spent float64) error {
	_, err := db.Exec(
		"UPDATE api_keys SET monthly_spent = monthly_spent + ?, last_used_at = ? WHERE id = ?",
		spent, time.Now(), id,
	)
	return err
}

// Audit log operations
func (db *DB) InsertAuditLog(log models.AuditLog) error {
	_, err := db.Exec(
		`INSERT INTO audit_logs (id, api_key_id, model, provider, prompt_tokens, completion_tokens,
		cached_tokens, duration_ms, cost_usd, status_code, streamed, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.ID, log.APIKeyID, log.Model, log.Provider,
		log.PromptTokens, log.CompletionTokens, log.CachedTokens,
		log.DurationMs, log.CostUSD, log.StatusCode, boolToInt(log.Streamed), log.CreatedAt,
	)
	return err
}

func (db *DB) GetAuditLogs(limit int) ([]models.AuditLog, error) {
	logs := make([]models.AuditLog, 0)
	rows, err := db.Query(
		`SELECT id, api_key_id, model, provider, prompt_tokens, completion_tokens,
		cached_tokens, duration_ms, cost_usd, status_code, streamed, created_at
		FROM audit_logs ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var l models.AuditLog
		var streamed int
		if err := rows.Scan(&l.ID, &l.APIKeyID, &l.Model, &l.Provider,
			&l.PromptTokens, &l.CompletionTokens, &l.CachedTokens,
			&l.DurationMs, &l.CostUSD, &l.StatusCode, &streamed, &l.CreatedAt); err != nil {
			return nil, err
		}
		l.Streamed = streamed == 1
		logs = append(logs, l)
	}
	return logs, nil
}

func scanProviderConfig(scanner interface {
	Scan(dest ...interface{}) error
}) (models.ProviderConfig, error) {
	var c models.ProviderConfig
	var isActive int
	var tokenExpiresAt sql.NullTime
	err := scanner.Scan(
		&c.ID, &c.Provider, &c.Label, &c.BaseURL, &c.APIKeyEncrypted,
		&c.Models, &isActive, &c.Priority,
		&c.AuthType, &c.AccessToken, &c.RefreshToken, &c.IDToken,
		&tokenExpiresAt, &c.ProviderMeta,
	)
	if err != nil {
		return c, err
	}
	c.IsActive = isActive == 1
	if tokenExpiresAt.Valid {
		c.TokenExpiresAt = &tokenExpiresAt.Time
	}
	return c, nil
}

const providerColumns = `id, provider, label, base_url, api_key_encrypted, models, is_active, priority,
auth_type, access_token, refresh_token, id_token, token_expires_at, provider_meta`

func (db *DB) ListProviderConfigs() ([]models.ProviderConfig, error) {
	configs := make([]models.ProviderConfig, 0)
	rows, err := db.Query(
		`SELECT ` + providerColumns + ` FROM provider_configs ORDER BY priority ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		c, err := scanProviderConfig(rows)
		if err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}
	return configs, nil
}

func (db *DB) UpsertProviderConfig(c models.ProviderConfig) error {
	var expiresAt interface{}
	if c.TokenExpiresAt != nil {
		expiresAt = c.TokenExpiresAt.Format(time.RFC3339)
	}
	_, err := db.Exec(
		`INSERT INTO provider_configs (`+providerColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			provider=excluded.provider, label=excluded.label, base_url=excluded.base_url,
			api_key_encrypted=excluded.api_key_encrypted, models=excluded.models,
			is_active=excluded.is_active, priority=excluded.priority,
			auth_type=excluded.auth_type, access_token=excluded.access_token,
			refresh_token=excluded.refresh_token, id_token=excluded.id_token,
			token_expires_at=excluded.token_expires_at, provider_meta=excluded.provider_meta`,
		c.ID, c.Provider, c.Label, c.BaseURL, c.APIKeyEncrypted, c.Models,
		boolToInt(c.IsActive), c.Priority,
		c.AuthType, c.AccessToken, c.RefreshToken, c.IDToken,
		expiresAt, c.ProviderMeta,
	)
	return err
}

func (db *DB) GetProviderConfig(id string) (*models.ProviderConfig, error) {
	c, err := scanProviderConfig(db.QueryRow(
		`SELECT `+providerColumns+` FROM provider_configs WHERE id = ?`, id,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (db *DB) DeleteProviderConfig(id string) error {
	_, err := db.Exec("DELETE FROM provider_configs WHERE id = ?", id)
	return err
}

func (db *DB) ListOAuthProviders() ([]models.ProviderConfig, error) {
	configs := make([]models.ProviderConfig, 0)
	rows, err := db.Query(
		`SELECT `+providerColumns+` FROM provider_configs WHERE auth_type = 'oauth' AND is_active = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		c, err := scanProviderConfig(rows)
		if err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}
	return configs, nil
}

func (db *DB) UpdateOAuthTokens(id, accessToken, refreshToken, idToken string, expiresAt *time.Time) error {
	var expiresAtArg interface{}
	if expiresAt != nil {
		expiresAtArg = expiresAt.Format(time.RFC3339)
	}
	_, err := db.Exec(
		`UPDATE provider_configs SET access_token=?, refresh_token=?, id_token=?, token_expires_at=? WHERE id=?`,
		accessToken, refreshToken, idToken, expiresAtArg, id,
	)
	return err
}

func (db *DB) GetProviderStats() (map[string]interface{}, error) {
	var total, active int
	db.QueryRow("SELECT COUNT(*) FROM provider_configs").Scan(&total)
	db.QueryRow("SELECT COUNT(*) FROM provider_configs WHERE is_active = 1").Scan(&active)
	return map[string]interface{}{
		"total_providers":  total,
		"active_providers": active,
	}, nil
}

// Stats
type Stats struct {
	TotalRequests   int     `json:"total_requests"`
	TotalTokens     int     `json:"total_tokens"`
	TotalCostUSD    float64 `json:"total_cost_usd"`
	ActiveKeys      int     `json:"active_keys"`
	AvgLatencyMs    float64 `json:"avg_latency_ms"`
}

func (db *DB) GetStats() (*Stats, error) {
	s := &Stats{}
	db.QueryRow("SELECT COUNT(*), COALESCE(SUM(prompt_tokens + completion_tokens), 0), COALESCE(SUM(cost_usd), 0), COALESCE(AVG(duration_ms), 0) FROM audit_logs").
		Scan(&s.TotalRequests, &s.TotalTokens, &s.TotalCostUSD, &s.AvgLatencyMs)
	db.QueryRow("SELECT COUNT(*) FROM api_keys WHERE is_revoked = 0").Scan(&s.ActiveKeys)
	return s, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
