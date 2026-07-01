# ProRouter Verification Plan

## 1. Unit Testing Strategy
*   **Gateway (Go):** Test adapters (`adapters/*_test.go`) to ensure payload conversions from OpenAI standard format to provider formats are correct.
*   **Routing Logic:** Test fallback mechanisms and recipe pipelines under mock HTTP failures.
*   **Dashboard:** Test API Key forms, layout rendering, and SSE message parsing.

## 2. Integration & End-to-End Tests
*   Verify that `curl` command using a ProRouter key (`pr-...`) returns valid streaming responses from Ollama, Anthropic, and OpenAI.
*   Test Tool Calling (Function Calling) compatibility through a custom local test script that executes tool-based workflows.

## 3. Performance & Latency Benchmarks
*   Measure Time to First Token (TTFT) when proxied through ProRouter vs direct provider calls. Target: < 2ms proxy overhead.
