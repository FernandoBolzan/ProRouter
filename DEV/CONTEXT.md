# ProRouter - Project Context & Vision

## 1. Executive Summary
ProRouter is an open-source, high-performance LLM gateway and routing engine designed as a self-hosted alternative to OpenRouter, 9Router, OmniRouter, and LiteLLM. It allows developers, power users, and enterprises to route, load-balance, cache, and monitor LLM API requests across both cloud providers (OpenAI, Anthropic Claude, Google Gemini, DeepSeek, Antigravity) and local instances (Ollama, vLLM, Llama.cpp) through a single OpenAI-compatible interface.

## 2. Core Pillars

### Performance First (`ProRouter Go`)
*   **Low Overhead:** Written in Go to guarantee sub-millisecond proxy latency and high-throughput streaming (SSE).
*   **Zero-Downtime Fallbacks:** Instantly switches to alternative providers or models when rate limits (429) or failures (5xx) occur.
*   **Zero-Dependency Binary:** Compiled as a single static binary for easy local running.

### Experience Centric (`ProRouter Zen`)
*   **Zero-Config Local Auto-discovery:** Automatically scans local network ports to register Ollama/vLLM instances.
*   **Model Arena Playground:** Visual side-by-side prompt testing interface to compare latency, costs, speed, and quality.
*   **OAuth PKCE for CLI & IDEs:** Direct sign-in from desktop shells, CLI agents, and IDE plugins (e.g. Cursor, Cline).

### Token Discipline & Cost Control (The Combos)
*   **Semantic Cache:** Lightweight vector storage (SQLite/pgvector) to bypass expensive provider calls for semantically similar prompts.
*   **Prompt Caching Alignment:** Rearranging payloads to maximize prompt caching hit rates for providers like Anthropic and DeepSeek (saving up to 90%).
*   **Context Squeezing:** Built-in local prompt compression (via LLMLingua-like heuristics or auto-summarization).
*   **Cascade Rote (Roteamento em Cascata):** Attempting tasks on local, cheaper models first, validating output quality locally, and up-routing to cloud providers only if necessary.
