# Active Specification: ProRouter Architecture

This document serves as the canonical technical spec for the ProRouter codebase.

## 1. Technical Stack
*   **Gateway Backend:** Go 1.21+ (using `gin-gonic/gin` or `gofiber/fiber`, and raw HTTP proxy engines).
*   **Database:** SQLite 3 (local) or PostgreSQL (for multi-tenant production).
*   **Frontend Dashboard:** Next.js 14+ (App Router, Tailwind CSS, shadcn/ui).
*   **API Standard:** Compatibility with OpenAI API `/v1/chat/completions`, `/v1/embeddings`, and `/v1/models`.

---

## 2. Component Design & Interfaces

### 2.1 The Adapter Layer (`adapters`)
To connect any LLM provider, we define a common interface in Go:

```go
type ProviderAdapter interface {
    // Translate standard OpenAI request payload to provider-specific payload
    TransformRequest(req *OpenAIChatRequest) (interface{}, error)
    
    // Translate non-stream provider response back to OpenAI response format
    TransformResponse(resp interface{}) (*OpenAIChatResponse, error)
    
    // Translate a single streaming Server-Sent Event chunk to OpenAI SSE format
    TransformStreamChunk(chunk []byte) (*OpenAIChatStreamResponse, error)
    
    // Returns the provider's specific base URL
    GetBaseURL() string
}
```

#### Adaptadores Específicos:
1.  **OpenAI-Compatible:** Direct proxy mapping for OpenAI, DeepSeek, Groq, Llama.cpp, vLLM, Antigravity.
2.  **Anthropic (Claude):** Maps `messages` array, extracts system prompts, converts tool schemas, translates SSE payloads (`message_start`, `content_block_delta`).
3.  **Google Gemini:** Translates `contents`/`parts` arrays, handles JSON stream variations.
4.  **Ollama:** Custom client parsing or fallback to Ollama's native `/v1` endpoint.

---

### 2.2 The Routing & Recipe Engine (`routing`)
A **Recipe** defines how a request is processed before execution and how fallbacks are handled.

#### Recipe JSON Schema:
```json
{
  "id": "smart-coder-recipe",
  "name": "Intelligent Code Optimizer",
  "pipeline": [
    {
      "step": "cache_lookup",
      "params": { "semantic": true, "threshold": 0.96 }
    },
    {
      "step": "context_compress",
      "params": { "max_history_tokens": 8000, "strategy": "summarize" }
    },
    {
      "step": "execution_chain",
      "params": {
        "models": [
          "ollama/llama3:70b",
          "deepseek-chat",
          "claude-3-5-sonnet"
        ],
        "fallback_on_status": [429, 500, 503],
        "validation": {
          "json_schema": true
        }
      }
    }
  ]
}
```

---

### 2.3 Token Measurement & Cost Management
*   **Tokenizer Engine:** Integration of a lightweight tiktoken client in Go (`pkoukk/tiktoken-go`) for local count estimation.
*   **Audit Logger:** Write request metadata (tokens, duration, cost, status, provider) to a structured log file or DB.
*   **Async Logging Queue:** Uses a Go channel buffer and batch operations to prevent SQLite lockups (`database is locked`) in concurrent writes.
*   **Budget Guard:** Pre-flight hook checking if the requester has exceeded their daily/weekly budget limit.

---

### 2.4 OAuth PKCE & local loopback server
*   **OAuth Server:** Implements standard OAuth 2.0 PKCE flow in the gateway.
*   **Local CLI Handshake:** Explicit support for arbitrary port callback URLs (`http://localhost:XXXX/callback`) to allow local IDEs/agents (e.g. Cursor, OpenCode) to sign in safely.
*   **Auto-HTTPS Bypass:** Handlers to support secure browser redirects to non-secure localhost endpoints with appropriate CORS rules.

---

### 2.5 Interface & Control (`dashboard`)
*   **API Key Minting:** Generates and validates cryptographically random bearer tokens (`pr-...`).
*   **Observability Charts:** Plots throughput (t/s), latencies (TTFT), and aggregated dollar spend.
*   **Playground:** Real-time dual/triple pane prompt debugger to visually compare outputs and speeds from selected recipes or providers.

---

## 3. Distribution & Package System

### 3.1 Installation Methods Matrix

| Method | Command | Best For |
|---|---|---|
| **Homebrew (macOS/Linux)** | `brew install prorouter/tap/prorouter` | Dev machines |
| **Scoop (Windows)** | `scoop install prorouter` | Dev machines (Win) |
| **Go Install** | `go install github.com/prorouter/gateway-go/cmd/prorouter@latest` | Go developers |
| **NPM / npx** | `npx prorouter serve` or `npm i -g @prorouter/cli` | JS-centric users |
| **Docker** | `docker run -p 8080:8080 prorouter/gateway` | Production / Cloud |
| **Binary Download** | `curl -fsSL https://get.prorouter.dev | sh` | Quick start |
| **Source Build** | `git clone && make build` | Contributors |

### 3.2 CLI Tool (`prorouter`) Unified Command Structure

```text
prorouter                       # Status overview of running instance
prorouter serve                 # Start the gateway server
prorouter init                  # Scaffold config file (~/.prorouter/config.yaml)
prorouter doctor                # Diagnostics and health check
prorouter key list              # List API keys
prorouter key generate          # Create new API key
prorouter key revoke <id>       # Revoke an API key
prorouter provider list         # Show connected providers
prorouter provider add          # Add provider credentials
prorouter recipe list           # List routing recipes
prorouter recipe apply <file>   # Apply a recipe from file
prorouter playground            # Open model arena in browser
prorouter proxy start           # Start proxy mode (reverse proxy to dashboard)
prorouter update                # Self-update to latest version
prorouter version               # Show version
```

### 3.3 Auto-Update Mechanism
*   **Channel-Based Updates:** `stable`, `beta`, `nightly`.
*   **Update Check:** The CLI checks for updates on startup (silently, once every 24h).
*   **Rollback Support:** `prorouter update --rollback` to revert to previous version.
*   **Signature Verification:** All binaries are signed with Cosign/Sigstore for supply chain security.

---

## 4. Configuration System

### 4.1 Config File (YAML) - `~/.prorouter/config.yaml`

```yaml
# ProRouter Configuration
server:
  host: "0.0.0.0"
  port: 8080
  tls_enabled: false
  cert_file: ""
  key_file: ""

database:
  engine: "sqlite"    # "sqlite" | "postgres"
  path: "~/.prorouter/data/prorouter.db"
  wal_mode: true
  max_connections: 100

dashboard:
  enabled: true
  port: 3000
  theme: "system"    # "light" | "dark" | "system"

auth:
  jwt_secret: ""      # Auto-generated on first run
  oauth_providers:
    github:
      enabled: false
      client_id: ""
      client_secret: ""
    google:
      enabled: false
      client_id: ""
      client_secret: ""

providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
  google:
    api_key: "${GEMINI_API_KEY}"
  deepseek:
    api_key: "${DEEPSEEK_API_KEY}"
  local:
    scan_ports: true
    ports: [11434, 8000, 1234]

recipes_file: "~/.prorouter/recipes/*.yaml"
log_level: "info"
update_channel: "stable"
```

---

## 5. Database Migration System
*   **Directory:** `gateway-go/internal/migrations/`
*   **Format:** Sequential timestamped SQL files (e.g. `001_initial_schema.sql`, `002_add_recipe_tables.sql`).
*   **Auto-Migrate:** On `prorouter serve`, pending migrations are applied automatically.
*   **Rollback:** `prorouter migrate --down <step>` for safe downgrades.

---

## 6. Project Repository Structure

```
prorouter/
├── .github/
│   ├── workflows/
│   │   ├── ci.yml              # Test & lint on PR
│   │   ├── release.yml         # Build and publish binaries
│   │   ├── docker.yml          # Build and push Docker images
│   │   └── npm-publish.yml     # Publish @prorouter/cli to npm
│   └── ISSUE_TEMPLATE/
│       ├── bug_report.md
│       └── feature_request.md
│
├── gateway-go/
│   ├── cmd/
│   │   └── prorouter/
│   │       ├── main.go
│   │       ├── serve.go
│   │       ├── init.go
│   │       ├── doctor.go
│   │       ├── key.go
│   │       ├── provider.go
│   │       ├── recipe.go
│   │       ├── playground.go
│   │       └── update.go
│   ├── internal/
│   │   ├── adapters/            # Provider adapters
│   │   │   ├── openai_adapter.go
│   │   │   ├── anthropic_adapter.go
│   │   │   ├── gemini_adapter.go
│   │   │   └── ollama_adapter.go
│   │   ├── cache/               # Cache engine (exact + semantic)
│   │   │   ├── cache.go
│   │   │   └── semantic.go
│   │   ├── config/              # Configuration loader
│   │   │   └── config.go
│   │   ├── database/            # DB connection & migrations
│   │   │   ├── db.go
│   │   │   └── migrations.go
│   │   ├── migrations/          # SQL migration files
│   │   ├── middleware/          # Auth, rate-limit, logging
│   │   │   ├── auth.go
│   │   │   ├── ratelimit.go
│   │   │   └── logger.go
│   │   ├── models/              # Data models
│   │   │   └── models.go
│   │   ├── oauth/               # OAuth PKCE server
│   │   │   └── oauth.go
│   │   ├── proxy/               # HTTP proxy & streaming
│   │   │   ├── proxy.go
│   │   │   └── stream.go
│   │   ├── routing/             # Recipe engine
│   │   │   ├── engine.go
│   │   │   ├── pipeline.go
│   │   │   └── recipe.go
│   │   └── tokenizer/           # Token counting
│   │       └── tokenizer.go
│   ├── go.mod
│   ├── go.sum
│   └── Makefile
│
├── dashboard-zen/
│   ├── src/
│   │   ├── app/
│   │   │   ├── layout.tsx
│   │   │   ├── page.tsx
│   │   │   ├── dashboard/
│   │   │   ├── keys/
│   │   │   ├── providers/
│   │   │   ├── recipes/
│   │   │   ├── playground/
│   │   │   └── settings/
│   │   ├── components/
│   │   │   ├── ui/              # shadcn/ui components
│   │   │   └── prorouter/       # Custom components
│   │   └── lib/
│   │       ├── api.ts
│   │       └── utils.ts
│   ├── public/
│   ├── package.json
│   ├── next.config.js
│   ├── tailwind.config.ts
│   └── tsconfig.json
│
├── cli-npm/
│   ├── bin/
│   │   └── prorouter            # NPM wrapper that downloads Go binary
│   ├── src/
│   │   └── install.ts
│   ├── package.json
│   └── README.md
│
├── scripts/
│   ├── install.sh               # Unix install script
│   ├── install.ps1              # Windows install script
│   └── dev.sh                   # Dev environment setup
│
├── docker-compose.yml
├── Dockerfile.gateway
├── Dockerfile.dashboard
├── Makefile
├── .gitignore
├── README.md
├── LICENSE
└── CONTRIBUTING.md
