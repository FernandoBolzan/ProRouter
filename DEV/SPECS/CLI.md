# CLI Tool Design: `prorouter`

## 1. Philosophy

The CLI is the primary interface for ProRouter. It follows the **"progressive disclosure"** principle:
- New users run `prorouter init` and `prorouter serve` to get started.
- Power users use `prorouter recipe apply`, `prorouter provider add`, `prorouter key generate`.
- The CLI provides rich terminal output with spinners, colors, and progress bars.

## 2. Command Tree

```
prorouter
├── serve          # Start gateway + dashboard
├── init           # Generate config file
├── doctor         # Diagnostics & health check
│
├── key
│   ├── generate   # Create new API key
│   ├── list       # List all keys
│   ├── revoke     # Revoke a key
│   └── validate   # Validate a key
│
├── provider
│   ├── list       # Show connected providers
│   ├── add        # Add provider credentials
│   ├── remove     # Remove a provider
│   └── test       # Test provider connection
│
├── recipe
│   ├── list       # List routing recipes
│   ├── apply      # Apply recipe from file
│   ├── create     # Create a recipe interactively
│   ├── edit       # Edit a recipe in $EDITOR
│   └── test       # Test a recipe with a prompt
│
├── proxy
│   ├── start      # Start reverse proxy to dashboard
│   └── stop       # Stop proxy
│
├── cache
│   ├── clear      # Clear response cache
│   ├── stats      # Cache hit/miss statistics
│   └── warm       # Pre-warm cache with prompts
│
├── playground     # Open model arena in browser
├── update         # Self-update
├── version        # Show version info
├── help           # Show help
└── completion     # Generate shell completions
```

## 3. Technology Stack for CLI

The CLI is written in **Go** (same binary as the gateway) using:
- **Cobra** (`github.com/spf13/cobra`) - Command structure
- **Viper** (`github.com/spf13/viper`) - Config management
- **Charm** (`github.com/charmbracelet/bubbletea`) - TUI components (spinners, progress bars)
- **Gum-style** output with `github.com/fatih/color` for colored output

## 4. Core UX Flows

### 4.1 First Run (`prorouter init`)
```
$ prorouter init

  ╭──────────────────────────────────────╮
  │         ProRouter Setup              │
  │                                      │
  │  ◉ Quick Start (recommended)         │
  │  ○ Advanced Configuration            │
  │  ○ Docker Environment                │
  ╰──────────────────────────────────────╯

  ✔ Created config file at ~/.prorouter/config.yaml
  ✔ Generated JWT secret
  ✔ Initialized SQLite database at ~/.prorouter/data/prorouter.db
  ✔ Applied 3 pending migrations

  Next steps:
    1. Set your API keys:
       export OPENAI_API_KEY="sk-..."
       export ANTHROPIC_API_KEY="sk-ant-..."

    2. Start the gateway:
       prorouter serve

    3. Open dashboard:
       open http://localhost:8080
```

### 4.2 Diagnostics (`prorouter doctor`)
```
$ prorouter doctor

  🔍 ProRouter Diagnostics
  ─────────────────────────
  ✔ Config file: ~/.prorouter/config.yaml (valid)
  ✔ Database: SQLite (WAL mode, 12MB)
  ✔ Port 8080: Available
  ✔ Port 3000: Available

  🌐 Provider Connectivity:
  ✔ OpenAI        [200 OK]  0.3s
  ✔ Anthropic     [200 OK]  0.4s
  ✘ Google Gemini [401]     Check GEMINI_API_KEY
  ✔ Ollama (local)[200 OK]  5ms
  ✔ DeepSeek      [200 OK]  0.6s

  📦 Version: 0.1.0 (stable)
  🔄 Update: 0.1.1 available (prorouter update)
```

### 4.3 Model Arena (`prorouter playground`)
```
$ prorouter playground

  Opens browser at http://localhost:8080/playground

  In the browser:
  ┌─────────────────────────────────────────────────┐
  │  Model Arena                    [Compare] [Send] │
  ├─────────────────────────────────────────────────┤
  │ Prompt: "Explain quantum computing in 3 levels" │
  ├──────────────┬──────────────┬──────────────────┤
  │ GPT-4o       │ Claude 3.5   │ Llama 3 (local)  │
  │──────────────│──────────────│──────────────────│
  │ $0.02        │ $0.03        │ $0.00            │
  │ 1.2s TTFT    │ 0.8s TTFT    │ 3.1s TTFT        │
  │ 45 t/s       │ 62 t/s       │ 28 t/s           │
  │────────────────────────────────────────────────│
  │ ⭐ Best      │              │                  │
  └──────────────┴──────────────┴──────────────────┘
```

## 5. Shell Completions

Supported shells: `bash`, `zsh`, `fish`, `powershell`

```bash
# Bash
source <(prorouter completion bash)

# Zsh
prorouter completion zsh > "${fpath[1]}/_prorouter"

# Fish
prorouter completion fish > ~/.config/fish/completions/prorouter.fish

# PowerShell
prorouter completion powershell > $PROFILE
```

## 6. Exit Codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | General error |
| 2 | Configuration error |
| 3 | Database error |
| 4 | Provider connection error |
| 5 | Authentication error |
| 6 | Update available (non-blocking) |
