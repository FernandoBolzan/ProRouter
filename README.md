# ProRouter

**Universal LLM Gateway** — Open-source alternative to OpenRouter, 9Router, and OmniRouter.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue)](https://golang.org)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

ProRouter is a high-performance, self-hosted LLM gateway that provides a unified OpenAI-compatible API for routing requests to cloud providers (OpenAI, Anthropic, Gemini, DeepSeek) and local instances (Ollama, vLLM) — with intelligent fallbacks, caching, and cost optimization.

## Features

- **Zero-Config Setup** — One binary, no external dependencies. SQLite database built in.
- **OpenAI-Compatible API** — Drop-in replacement for any OpenAI SDK. Just change the base URL.
- **Multi-Provider Routing** — OpenAI, Anthropic Claude, Google Gemini, DeepSeek, Ollama, and more.
- **Intelligent Fallbacks** — Automatic failover when providers return errors or rate limits.
- **Streaming Support** — Full SSE streaming proxy with sub-millisecond overhead.
- **API Key Management** — Create, revoke, and budget API keys with usage tracking.
- **Embedded Dashboard** — Beautiful web UI for monitoring, key management, and model playground.
- **Cost Tracking** — Per-request cost logging and monthly budget enforcement.
- **BYOK (Bring Your Own Key)** — Use your own provider API keys.
- **Local LLM Auto-Discovery** — Automatically detects Ollama and vLLM instances on your network.

## Installation

### Quick Install (macOS/Linux)
```bash
curl -fsSL https://raw.githubusercontent.com/FernandoBolzan/ProRouter/main/scripts/install.sh | sh
```

### Quick Install (Windows PowerShell)
```powershell
iwr https://raw.githubusercontent.com/FernandoBolzan/ProRouter/main/scripts/install.ps1 | iex
```

### NPM
```bash
npx @fernandobolzan/prorouter-cli serve
```

### Go Install
```bash
go install github.com/FernandoBolzan/ProRouter/gateway-go/cmd/prorouter@latest
```

### Docker
```bash
# Build locally
git clone https://github.com/FernandoBolzan/ProRouter
cd ProRouter && docker compose up -d
```

### Build from Source
```bash
git clone https://github.com/FernandoBolzan/ProRouter
cd ProRouter
make build
```

## Quick Start

```bash
# 1. Initialize configuration
prorouter init

# 2. Set your API keys
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."

# 3. Start the gateway
prorouter serve

# 4. Open dashboard
open http://localhost:8080/dashboard/
```

## Usage

### Generate an API Key
```bash
prorouter key generate --name "my-app" --budget 10
```
Or via the dashboard at `http://localhost:8080/dashboard/`.

### Use with any OpenAI SDK
```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="pr-<your-key>",
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "Hello!"}],
)
```

### Use with cURL
```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer pr-<your-key>" \
  -d '{"model": "gpt-4o", "messages": [{"role": "user", "content": "Hello!"}]}'
```

### Environment Variables
| Variable | Description |
|---|---|
| `OPENAI_API_KEY` | OpenAI API key |
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `GEMINI_API_KEY` | Google Gemini API key |
| `DEEPSEEK_API_KEY` | DeepSeek API key |

### CLI Commands
```bash
prorouter serve      # Start the gateway server
prorouter init       # Create configuration
prorouter key        # Manage API keys
prorouter provider   # Configure providers
prorouter doctor     # System diagnostics
prorouter playground # Open model arena
prorouter update     # Self-update
```

## Architecture

```
┌──────────────────────────────────────┐
│           Your Application           │
│  (OpenAI SDK, LangChain, cURL, etc)  │
└──────────────┬───────────────────────┘
               │ OpenAI API format
               ▼
┌──────────────────────────────────────┐
│         ProRouter Gateway (Go)       │
│  ┌────────────────────────────────┐  │
│  │  Auth Middleware               │  │
│  │  API Key Validation            │  │
│  ├────────────────────────────────┤  │
│  │  Provider Adapters             │  │
│  │  OpenAI | Anthropic | Gemini.. │  │
│  ├────────────────────────────────┤  │
│  │  Cache Engine                  │  │
│  │  Exact + Semantic Cache        │  │
│  ├────────────────────────────────┤  │
│  │  Audit Logger                  │  │
│  │  SQLite (embedded)             │  │
│  └────────────────────────────────┘  │
└──────┬─────────────────────────┬─────┘
       │                         │
       ▼                         ▼
┌──────────────┐       ┌────────────────┐
│ Cloud        │       │ Local          │
│ OpenAI       │       │ Ollama         │
│ Anthropic    │       │ vLLM           │
│ Gemini       │       │ Llama.cpp      │
│ DeepSeek     │       │ LM Studio      │
└──────────────┘       └────────────────┘
```

## Development

```bash
# Build
make build

# Run tests
make test

# Run E2E tests
cd e2e && npm install && npx playwright install chromium && npx playwright test
```

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

[MIT](LICENSE)
