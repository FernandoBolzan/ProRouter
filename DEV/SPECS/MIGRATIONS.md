# Database Migration System

## 1. Migration Directory Structure

```
gateway-go/internal/migrations/
├── 001_create_users.sql
├── 002_create_api_keys.sql
├── 003_create_audit_logs.sql
├── 004_create_recipes.sql
├── 005_add_cache_table.sql
├── 006_add_oauth_sessions.sql
└── 007_add_provider_credentials.sql
```

## 2. Migration File Format

```sql
-- 001_create_users.sql
-- Description: Initial users and organizations tables
-- Up

CREATE TABLE IF NOT EXISTS organizations (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    email TEXT UNIQUE NOT NULL,
    display_name TEXT NOT NULL,
    avatar_url TEXT,
    role TEXT DEFAULT 'member' CHECK(role IN ('admin', 'member', 'viewer')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_organization ON users(organization_id);
CREATE INDEX idx_users_email ON users(email);

-- Down

DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS organizations;
```

## 3. Migration Engine (Go)

```go
// internal/database/migrations.go
package database

import (
    "embed"
    "fmt"
    "io/fs"
    "sort"
    "strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Migration struct {
    Version     int
    Description string
    UpSQL       string
    DownSQL     string
}

func LoadMigrations() ([]Migration, error) {
    entries, err := fs.ReadDir(migrationsFS, "migrations")
    if err != nil {
        return nil, fmt.Errorf("reading migrations dir: %w", err)
    }

    var migrations []Migration
    for _, entry := range entries {
        if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
            continue
        }

        content, err := fs.ReadFile(migrationsFS, "migrations/"+entry.Name())
        if err != nil {
            return nil, fmt.Errorf("reading %s: %w", entry.Name(), err)
        }

        parts := strings.SplitN(string(content), "-- Down", 2)
        migration := Migration{
            UpSQL:   strings.TrimSpace(parts[0]),
        }
        if len(parts) > 1 {
            migration.DownSQL = strings.TrimSpace(parts[1])
        }

        fmt.Sscanf(entry.Name(), "%d_", &migration.Version)
        migrations = append(migrations, migration)
    }

    sort.Slice(migrations, func(i, j int) bool {
        return migrations[i].Version < migrations[j].Version
    })

    return migrations, nil
}

func (db *DB) RunMigrations() error {
    // Create migrations tracking table
    db.Exec(`
        CREATE TABLE IF NOT EXISTS _migrations (
            version INTEGER PRIMARY KEY,
            applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )
    `)

    currentVersion := db.GetCurrentMigrationVersion()
    migrations, _ := LoadMigrations()

    for _, m := range migrations {
        if m.Version <= currentVersion {
            continue
        }
        if err := db.Exec(m.UpSQL).Error; err != nil {
            return fmt.Errorf("migration %d failed: %w", m.Version, err)
        }
        db.Exec("INSERT INTO _migrations (version) VALUES (?)", m.Version)
        fmt.Printf("  ✔ Applied migration %d: %s\n", m.Version, m.Description)
    }

    return nil
}
```

## 4. Migration Commands (CLI)

```
prorouter migrate            # Apply pending migrations (auto-run on serve)
prorouter migrate --down 3   # Rollback 3 migrations
prorouter migrate --status   # Show migration status

$ prorouter migrate --status
Migration Status:
  ✔ 001 - Initial schema
  ✔ 002 - API keys
  ✔ 003 - Audit logs
  ✘ 004 - Recipes (PENDING)
  ✘ 005 - Cache table (PENDING)
```

## 5. Schema Overview

```sql
-- Users & Organizations
organizations: id, name, slug, created_at, updated_at
users: id, org_id, email, display_name, avatar_url, role, created_at, updated_at

-- API Keys
api_keys: id, org_id, user_id, key_prefix, key_hash, name, permissions,
          spend_limit_daily, spend_limit_monthly, max_tokens_per_min,
          allowed_models, allowed_providers, expires_at,
          last_used_at, is_revoked, created_at

-- Audit Logs
audit_logs: id, org_id, api_key_id, user_id, model, provider, recipe_id,
            prompt_tokens, completion_tokens, cached_tokens,
            duration_ms, cost_usd, status_code, error_message,
            streamed, idempotency_key, created_at

-- Routing Recipes
recipes: id, org_id, name, description, pipeline_json,
         is_active, is_default, created_at, updated_at

-- Provider Credentials (BYOK)
provider_credentials: id, org_id, provider_name, label,
                      key_encrypted, config_json, is_active,
                      priority, created_at, updated_at

-- Cache
response_cache: id, prompt_hash, semantic_hash, response_json,
                model, provider, tokens_cached, hit_count,
                created_at, expires_at

-- OAuth Sessions
oauth_sessions: id, code_challenge, code_challenge_method,
                callback_url, client_state, user_id,
                expires_at, is_used, created_at
```
