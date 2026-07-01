<div align="center">

# 🚀 ProRouter

**O Gateway Universal para LLMs — Alternativa Open-Source ao OpenRouter, 9Router e LiteLLM**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue)](https://golang.org)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](#contribuindo)
[![GitHub Release](https://img.shields.io/github/v/release/FernandoBolzan/ProRouter)](https://github.com/FernandoBolzan/ProRouter/releases)
[![GitHub Stars](https://img.shields.io/github/stars/FernandoBolzan/ProRouter)](https://github.com/FernandoBolzan/ProRouter/stargazers)

---

**⚠️ Precisamos de você!** Este projeto é mantido pela comunidade e está em evolução ativa. Contribuições de todo tipo são bem-vindas — código, documentação, testes, ideias, report de bugs. Veja [como contribuir](#contribuindo).

*Uma iniciativa do **Grupo IAPro** no WhatsApp — [entre para o grupo](https://fernandobolzan.com/bio) e participe das discussões.*

---

</div>

## 📋 Índice

- [Visão Geral](#-visão-geral)
- [Recursos](#-recursos)
- [Arquitetura](#-arquitetura)
- [Instalação](#-instalação)
- [Quick Start](#-quick-start)
- [Configuração](#-configuração)
- [CLI Completa](#-cli-completa)
- [Provedores Suportados](#-provedores-suportados)
- [Uso com SDKs](#-uso-com-sdks)
- [Receitas de Roteamento (Recipes)](#-receitas-de-roteamento-recipes)
- [OAuth PKCE para Agentes e IDEs](#-oauth-pkce-para-agentes-e-ides)
- [Dashboard Web](#-dashboard-web)
- [Desenvolvimento](#-desenvolvimento)
- [Roadmap](#-roadmap)
- [Contribuindo](#contribuindo)
- [Suporte](#-suporte)
- [Licença](#-licença)

---

## 🎯 Visão Geral

ProRouter é um **gateway de LLM auto-hospedado, performático e open-source** que unifica o acesso a dezenas de provedores de inteligência artificial através de uma única API compatível com OpenAI.

Em vez de gerenciar múltiplas chaves de API, SDKs diferentes e endpoints dispersos, você aponta seu cliente para o ProRouter e ele cuida do roteamento inteligente, fallbacks automáticos, cache semântico, compressão de contexto e auditoria de custos.

### Por que auto-hospedar?

| OpenRouter / 9Router | **ProRouter** |
|---|---|
| Taxas por requisição | **Zero taxa** — use suas próprias chaves |
| Limitado a provedores suportados | **Qualquer provedor** — cloud + local |
| Dados passam por terceiros | **100% self-hosted** — seus dados, seu controle |
| Sem cache local | **Cache semântico embutido** (SQLite/pgvector) |
| Sem fallback customizável | **Receitas de roteamento** com pipeline completo |
| Sem suporte a modelos locais | **Auto-descoberta** de Ollama, vLLM, Llama.cpp |

---

## ✨ Recursos

### 🔌 Compatibilidade OpenAI
API 100% compatível com `/v1/chat/completions`, `/v1/embeddings` e `/v1/models`. Funcione com qualquer SDK ou ferramenta que suporte OpenAI — LangChain, Vercel AI SDK, Cursor, Cline, OpenCode, entre outros.

### 🔀 Roteamento Inteligente
Distribua requisições entre provedores com base em latência, custo, disponibilidade ou regras customizadas. Configure **receitas de roteamento (Recipes)** com pipeline de pré-processamento, execução encadeada e fallbacks.

### 🛡️ Fallback Automático
Se um provedor retornar erro (5xx), rate limit (429) ou timeout, o ProRouter tenta automaticamente o próximo provedor na cadeia — sem interrupção para sua aplicação.

### 💰 Controle de Custos
- Orçamentos por chave de API (diário/semanal/mensal)
- Log de custo por requisição (tokens de entrada + saída)
- Cache semântico para evitar chamadas redundantes
- Compressão de contexto para reduzir tokens

### 🧠 Cache Semântico
Armazena embeddings de prompts anteriores e reutiliza respostas para consultas semanticamente similares (threshold configurável). Economia de até 90% em chamadas repetitivas.

### 📡 Suporte a Streaming (SSE)
Proxy de streaming com latência submilissegundo. Suporte completo a Server-Sent Events para todos os provedores compatíveis.

### 🖥️ Dashboard Embutido
Interface web elegante construída com Next.js 14, Tailwind CSS e shadcn/ui para monitoramento, gerenciamento de chaves, playground de modelos e visualização de métricas.

### 🔑 Gerenciamento de API Keys
Crie, revogue e defina orçamentos para chaves de API no formato `pr-...`. Ideal para fornecer acesso a múltiplos usuários ou aplicações.

### 🤖 Auto-Descoberta Local
Escaneie automaticamente a rede local para detectar instâncias de Ollama, vLLM e Llama.cpp — sem configuração manual.

### 🔐 OAuth PKCE para Agentes e IDEs
Autenticação direta via OAuth 2.0 PKCE para agentes de linha de comando e IDEs como Cursor, Cline e OpenCode. Permite login seguro sem expor chaves de API.

### 🏠 Suporte a Modelos Locais
Conecte-se a modelos rodando localmente via Ollama, vLLM, Llama.cpp ou LM Studio e use-os como provedores de primeira classe no roteamento.

### 📦 Binário Único Static
Compilado em Go como um único binário estático sem dependências externas. SQLite embutido. Pronto para usar em qualquer lugar.

---

## 🏗️ Arquitetura

```
                     ┌──────────────────────────────────────┐
                     │        Sua Aplicação / Cliente        │
                     │  (OpenAI SDK, LangChain, cURL, etc)   │
                     └──────────────┬───────────────────────┘
                                    │ API compatível OpenAI
                                    ▼
┌─────────────────────────────────────────────────────────────────┐
│                     ProRouter Gateway (Go)                       │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │ Auth Middleware│  │  Rate Limit  │  │  Audit Logger        │  │
│  │ (API Key JWT) │  │  (token/sec) │  │  (SQLite assíncrono) │  │
│  └──────┬───────┘  └──────┬───────┘  └──────────┬───────────┘  │
│         │                 │                      │              │
│  ┌──────┴─────────────────┴──────────────────────┴───────────┐  │
│  │                    Recipe Engine                           │  │
│  │  ┌──────────┐  ┌───────────┐  ┌──────────┐  ┌─────────┐  │  │
│  │  │Cache     │→ │Context    │→ │Execution │→ │Validation│  │  │
│  │  │Lookup    │  │Compress   │  │Chain     │  │(opcional)│  │  │
│  │  └──────────┘  └───────────┘  └──────────┘  └─────────┘  │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │                  Provider Adapters                         │  │
│  │  ┌──────────┐ ┌──────────┐ ┌────────┐ ┌────────┐ ┌─────┐  │  │
│  │  │ OpenAI   │ │ Anthropic│ │ Gemini │ │DeepSeek│ │Ollama│  │  │
│  │  │ (nativo) │ │ (Claude) │ │ (Google)│ │        │ │      │  │  │
│  │  └──────────┘ └──────────┘ └────────┘ └────────┘ └─────┘  │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────┬──────────────────────────────────────────┬───────────┘
           │                                          │
           ▼                                          ▼
┌──────────────────────┐              ┌──────────────────────────┐
│   Provedores Cloud   │              │   Provedores Locais      │
│  ┌────────────────┐  │              │  ┌─────────────────────┐ │
│  │ OpenAI (GPT-4o) │  │              │  │ Ollama              │ │
│  │ Anthropic Claude│  │              │  │ vLLM               │ │
│  │ Google Gemini   │  │              │  │ Llama.cpp          │ │
│  │ DeepSeek V3/R1  │  │              │  │ LM Studio          │ │
│  │ Antigravity     │  │              │  └─────────────────────┘ │
│  │ Groq            │  │              └──────────────────────────┘
│  └────────────────┘  │
└──────────────────────┘
```

### Componentes Principais

**Adapter Layer** — Interface Go que padroniza requisições/respostas entre provedores. Cada provedor implementa `TransformRequest`, `TransformResponse` e `TransformStreamChunk` para traduzir do formato OpenAI para o formato nativo do provedor e vice-versa.

**Recipe Engine** — Motor de roteamento que executa pipelines configuráveis: cache lookup → compressão de contexto → chain de execução com fallbacks → validação de saída.

**Cache Engine** — Cache exato (por hash da requisição) e semântico (por similaridade de embedding via SQLite/pgvector).

**OAuth PKCE Server** — Implementa OAuth 2.0 PKCE para autenticação direta de agentes CLI e IDEs.

---

## 📦 Instalação

### Linux / macOS (via script)

```bash
curl -fsSL https://raw.githubusercontent.com/FernandoBolzan/ProRouter/main/scripts/install.sh | sh
```

### Windows (via PowerShell)

```powershell
iwr https://raw.githubusercontent.com/FernandoBolzan/ProRouter/main/scripts/install.ps1 | iex
```

### Go Install

```bash
go install github.com/FernandoBolzan/ProRouter/gateway-go/cmd/prorouter@latest
```

### Docker

```bash
docker run -p 8080:8080 ghcr.io/FernandoBolzan/prorouter:latest
```

Ou com docker-compose:

```bash
git clone https://github.com/FernandoBolzan/ProRouter.git
cd ProRouter
docker compose up -d
```

### Build da Fonte

```bash
git clone https://github.com/FernandoBolzan/ProRouter.git
cd ProRouter
make build
```

### Homebrew (macOS / Linux)

```bash
brew install prorouter/tap/prorouter
```

### Scoop (Windows)

```bash
scoop bucket add prorouter https://github.com/FernandoBolzan/scoop-bucket
scoop install prorouter
```

---

## ⚡ Quick Start

```bash
# 1. Inicie a configuração
prorouter init

# 2. Defina suas chaves de API
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export GEMINI_API_KEY="..."
export DEEPSEEK_API_KEY="sk-..."

# 3. Inicie o gateway
prorouter serve

# 4. Abra o dashboard
open http://localhost:8080/dashboard/
```

Assim que o servidor iniciar, você já pode fazer requisições:

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer pr-<sua-chave>" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

---

## ⚙️ Configuração

O arquivo de configuração fica em `~/.prorouter/config.yaml` e é gerado automaticamente pelo `prorouter init`:

```yaml
# ProRouter Configuration
server:
  host: "0.0.0.0"
  port: 8080

database:
  engine: "sqlite"        # "sqlite" | "postgres"
  path: "~/.prorouter/data/prorouter.db"
  wal_mode: true

dashboard:
  enabled: true
  port: 3000
  theme: "system"         # "light" | "dark" | "system"

auth:
  jwt_secret: ""           # Auto-gerado no primeiro init
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

### Variáveis de Ambiente

| Variável | Descrição |
|---|---|
| `OPENAI_API_KEY` | Chave de API OpenAI |
| `ANTHROPIC_API_KEY` | Chave de API Anthropic Claude |
| `GEMINI_API_KEY` | Chave de API Google Gemini |
| `DEEPSEEK_API_KEY` | Chave de API DeepSeek |
| `ANTIGRAVITY_API_KEY` | Chave de API Antigravity |
| `GROQ_API_KEY` | Chave de API Groq |

---

## 🖥️ CLI Completa

```text
prorouter                       # Status geral da instância rodando
prorouter serve                 # Inicia o servidor gateway
prorouter init                  # Gera arquivo de configuração (~/.prorouter/)
prorouter doctor                # Diagnóstico completo do sistema
prorouter version               # Exibe a versão

Gerenciamento de Chaves:
prorouter key list              # Lista todas as chaves de API
prorouter key generate          # Cria nova chave de API
prorouter key revoke <id>       # Revoga uma chave de API
prorouter key budget <id>       # Define orçamento para uma chave

Gerenciamento de Provedores:
prorouter provider list         # Lista provedores configurados
prorouter provider add          # Adiciona novo provedor
prorouter provider remove <id>  # Remove um provedor

Receitas de Roteamento:
prorouter recipe list           # Lista receitas cadastradas
prorouter recipe apply <file>   # Aplica uma receita de um arquivo

Ferramentas:
prorouter playground            # Abre o Model Arena no navegador
prorouter proxy start           # Inicia modo proxy reverso
prorouter update                # Auto-update para última versão
```

---

## 🌐 Provedores Suportados

### Cloud

| Provedor | Modelos | Adapter | Status |
|---|---|---|---|
| **OpenAI** | GPT-4o, GPT-4, GPT-3.5, o1, o3 | Nativo | ✅ |
| **Anthropic** | Claude 3.5 Sonnet, Claude 3 Opus, Claude 3 Haiku | Nativo | ✅ |
| **Google** | Gemini 2.5 Pro, Gemini 2.0 Flash | Nativo | ✅ |
| **DeepSeek** | DeepSeek V3, DeepSeek R1 | OpenAI-Compatible | ✅ |
| **Antigravity** | Antigravity QwQ, Antigravity DeepSeek | OpenAI-Compatible | ✅ |
| **Groq** | Mixtral, Llama 3, Gemma | OpenAI-Compatible | ✅ |
| **Qualquer OpenAI-Compatible** | Qualquer API compatível | Genérico | ✅ |

### Local

| Provedor | Descoberta | Adapter | Status |
|---|---|---|---|
| **Ollama** | Auto-scan (porta 11434) | Nativo | ✅ |
| **vLLM** | Auto-scan (porta 8000) | OpenAI-Compatible | ✅ |
| **Llama.cpp** | Auto-scan (porta 1234) | OpenAI-Compatible | ✅ |
| **LM Studio** | Auto-scan | OpenAI-Compatible | ✅ |

---

## 🔌 Uso com SDKs

### Python (OpenAI SDK)

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="pr-<sua-chave>",
)

# Chat Completion
response = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "Explique computação quântica"}],
    stream=True,
)

for chunk in response:
    print(chunk.choices[0].delta.content, end="")
```

### Node.js (OpenAI SDK)

```javascript
import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: 'http://localhost:8080/v1',
  apiKey: 'pr-<sua-chave>',
});

const stream = await client.chat.completions.create({
  model: 'claude-sonnet-4',
  messages: [{ role: 'user', content: 'Hello!' }],
  stream: true,
});

for await (const chunk of stream) {
  process.stdout.write(chunk.choices[0]?.delta?.content || '');
}
```

### LangChain

```python
from langchain_openai import ChatOpenAI

llm = ChatOpenAI(
    base_url="http://localhost:8080/v1",
    api_key="pr-<sua-chave>",
    model="gpt-4o",
)
```

### cURL

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer pr-<sua-chave>" \
  -d '{
    "model": "deepseek-chat",
    "messages": [
      {"role": "system", "content": "Você é um assistente útil."},
      {"role": "user", "content": "Escreva um poema sobre IA."}
    ],
    "temperature": 0.7,
    "stream": true
  }'
```

---

## 📜 Receitas de Roteamento (Recipes)

Recipes são pipelines configuráveis que definem como cada requisição é processada. Você pode criar arquivos YAML/JSON em `~/.prorouter/recipes/`:

```json
{
  "id": "economico-inteligente",
  "name": "Econômico e Inteligente",
  "pipeline": [
    {
      "step": "cache_lookup",
      "params": { "semantic": true, "threshold": 0.95 }
    },
    {
      "step": "context_compress",
      "params": { "max_tokens": 6000, "strategy": "summarize" }
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
        "timeout": 30000
      }
    }
  ]
}
```

Aplique com:

```bash
prorouter recipe apply ~/.prorouter/recipes/economico-inteligente.json
```

---

## 🔐 OAuth PKCE para Agentes e IDEs

O ProRouter implementa OAuth 2.0 PKCE para permitir que agentes de CLI e IDEs (Cursor, Cline, OpenCode) façam login diretamente no gateway sem expor chaves de API:

```
Agente/IDE                    ProRouter Gateway              Provedor OAuth
    │                              │                              │
    │─── Init PKCE ───────────────→│                              │
    │←── code_verifier + URL ─────│                              │
    │                              │─── Authorization Request ──→│
    │←── Authorization Code ──────│                              │
    │                              │─── Token Exchange ─────────→│
    │                              │←── Access + Refresh Token ──│
    │←── Session Token ───────────│                              │
    │                              │                              │
```

**Provedores OAuth suportados:** GitHub, Google, Claude, Codex.

---

## 🖥️ Dashboard Web

O dashboard embutido (ProRouter Zen) oferece:

- **Métricas em tempo real** — Throughput (t/s), latências (TTFT), custo agregado
- **Gerenciamento de chaves** — Criação, revogação e orçamentos
- **Model Arena** — Comparação lado a lado de respostas de diferentes modelos/provedores
- **Histórico de requisições** — Log completo com tokens, custo, provedor, status
- **Configuração visual** — Adicionar/remover provedores, gerenciar receitas

---

## 💻 Desenvolvimento

```bash
# Clonar
git clone https://github.com/FernandoBolzan/ProRouter.git
cd ProRouter

# Build
make build

# Testes unitários
make test

# Testes E2E (Playwright)
cd e2e && npm install && npx playwright install chromium && npx playwright test

# Desenvolvimento com hot-reload
make dev

# Verificar cobertura
make coverage
```

### Stack Tecnológica

| Camada | Tecnologia |
|---|---|
| **Backend** | Go 1.21+ |
| **Database** | SQLite 3 (local) / PostgreSQL (produção) |
| **Frontend** | Next.js 14, Tailwind CSS, shadcn/ui |
| **API** | OpenAI-compatible (`/v1/chat/completions`) |
| **OAuth** | PKCE (Proof Key for Code Exchange) |
| **Distribuição** | GitHub Releases, GoReleaser, Docker |

---

## 🗺️ Roadmap

### 🔜 Em Breve

- [ ] **Provedor Cloudflare AI** — Workers AI e modelos serverless
- [ ] **Provedor AWS Bedrock** — Acesso a modelos via AWS
- [ ] **Provedor Azure OpenAI** — Integração com Azure
- [ ] **Plugin System** — Middleware customizável via plugins

### 🔮 Futuro

- [ ] **Multi-tenant** — Isolamento completo entre organizações
- [ ] **Rate Limiting Avançado** — Por usuário, por modelo, por provedor
- [ ] **Alertas** — Notificações via webhook, email e WhatsApp
- [ ] **Painel Administrativo** — Gestão completa de usuários, planos e faturas
- [ ] **OpenTelemetry** — Exportação de traces e métricas para Prometheus/Grafana

---

## Contribuindo

**Toda contribuição é bem-vinda!** Este é um projeto comunitário e cresce com a ajuda de pessoas como você.

Formas de contribuir:

- **🐛 Reportar bugs** — Abra uma [issue](https://github.com/FernandoBolzan/ProRouter/issues)
- **💡 Sugerir funcionalidades** — Conte-nos sua ideia
- **📝 Melhorar documentação** — README, exemplos, tutoriais
- **🔧 Enviar PRs** — Correções, novos adaptadores, otimizações
- **🧪 Escrever testes** — Unitários, integração, E2E
- **🌍 Traduções** — Ajudar a alcançar mais pessoas

Veja [CONTRIBUTING.md](CONTRIBUTING.md) para guia completo.

---

## 💬 Suporte

- 📱 **Grupo WhatsApp IAPro** — [Entre aqui](https://fernandobolzan.com/bio) para discussões, dúvidas e novidades
- 🐙 **GitHub Issues** — [Reporte bugs](https://github.com/FernandoBolzan/ProRouter/issues) e sugira melhorias
- ⭐ **GitHub Stars** — Ajude o projeto a crescer deixando uma estrela

---

## 📄 Licença

Distribuído sob licença **MIT**. Veja [LICENSE](LICENSE) para mais informações.

---

<div align="center">
  <strong>Feito com ❤️ pela comunidade</strong><br>
  <a href="https://fernandobolzan.com/bio">Grupo IAPro no WhatsApp</a>
</div>
