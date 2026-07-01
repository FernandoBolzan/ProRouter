# Implementation Checklists: ProRouter

These checklists define the sequence of implementation from the ground up, organized in distinct validation milestones.

---

## Phase 1: Core Scaffolding & Setup
- [ ] Initialize git repository and directory structures.
- [ ] Set up Go `go.mod` file in `/gateway-go` with basic dependencies (`gin`, `resty`, `sqlite3`, `tiktoken-go`).
- [ ] Set up Next.js app in `/dashboard-zen` with Tailwind CSS and layout components.
- [ ] Design Docker Compose file to orchestrate gateway, DB (Postgres/SQLite), and dashboard.

---

## Phase 2: Gateway core & OpenAI Compatibility Proxy
- [ ] Implement Go HTTP Server routing `/v1/chat/completions` and `/v1/models`.
- [ ] Implement SSE (Server-Sent Events) streaming engine in Go using `http.Flusher`.
- [ ] Build the `ProviderAdapter` interface.
- [ ] Implement the **OpenAI-Compatible Adapter** (for OpenAI, DeepSeek, Groq, local vLLM).
- [ ] Implement the **Anthropic Adapter** (Claude payload mapping & stream translation).
- [ ] Implement the **Google Gemini Adapter** (Gemini API format & streaming).
- [ ] Implement local environment variables mapping for API keys (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, etc.).
- [ ] **Verification:** Send basic and streaming requests using `curl` and confirm correct forwarding and responses.

---

## Phase 3: DB Integration & Key Management
- [ ] Build database connection layer using SQLite3 with WAL (Write-Ahead Logging) active.
- [ ] Implement an asynchronous audit log queue via Go channels to avoid SQLite database locking.
- [ ] Define DB Schemas:
  - `users` / `organizations`
  - `api_keys` (token hash, permissions, spend limit, usage)
  - `recipes` (routing configurations)
  - `audit_logs` (model, provider, duration, tokens, cost)
- [ ] Implement key authentication middleware in Go gateway (`Authorization: Bearer pr-...`).
- [ ] Create basic CLI command or API route to generate/revoke api keys (`pr-...`).
- [ ] **Verification:** Confirm unauthorized requests are rejected and authenticated requests are correctly logged.

---

## Phase 4: OAuth PKCE Implementation
- [ ] Implement OAuth 2.0 PKCE auth state machine `/auth` & `/api/v1/auth/keys`.
- [ ] Support local loopback callback registration (`http://localhost:XXXX/callback`).
- [ ] Configure secure CORS rules and auto-HTTPS exceptions for CLI local endpoints.
- [ ] **Verification:** Emulate authorization requests from CLI client and verify key generation exchange.

---

## Phase 5: Local LLM Auto-Discovery
- [ ] Build automated network scanning worker in Go.
- [ ] Scan standard local ports (`11434` for Ollama, `8000` for vLLM, `1234` for LM Studio).
- [ ] Query local `/v1/models` or native Ollama API `/api/tags`.
- [ ] Register discovered models automatically into the active model catalog.
- [ ] **Verification:** Run a local Ollama instance and confirm ProRouter detects and exposes its models automatically.

---

## Phase 6: Roteamento Inteligente & Combos
- [ ] Implement the Recipe Pipeline Engine (evaluating pipeline steps on request entry).
- [ ] Implement **Cache Lookup Step** (exact cache on DB and semantic cache using pgvector/sqlite-vss).
- [ ] Implement **Prompt Compression Step** (removing conversational clutter).
- [ ] Implement **Execution Chain with Fallbacks** (retrying next model in order if L1 returns error).
- [ ] **Verification:** Test failover behavior by simulating a dead provider and verifying it routes to the secondary provider.

---

## Phase 7: Dashboard Zen
- [ ] Connect dashboard pages to the Go API database.
- [ ] Build API Key creation, revocation, and limit management UI.
- [ ] Build Model Arena (dual-pane playground) with streaming support.
- [ ] Build analytics graphs (latency, spend, model distributions).
- [ ] **Verification:** Perform an end-to-end flow from dashboard configuration to tool integration.
