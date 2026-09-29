# Project
- Go 1.25.1 local gateway exposing Gemini Web through Gemini-native, OpenAI-compatible, and Claude-compatible HTTP APIs.
- Browser cookies are required; all three API surfaces ultimately use one Gemini Web client.
- The embedded dashboard is the default local entry point; OpenAI-compatible routes are the primary client interface.

# Repository Map
- `cmd/server/main.go`: Fx entry point; loads config/logger, then server and API modules.
- `internal/server/`: Fiber setup, middleware, `/health`, `/docs`, and embedded `/openapi.json`.
- `internal/auth/`: gateway API key generation, private persistence, rotation, and protocol-route authentication.
- `internal/commons/configs/`: environment parsing and validation; `models/` and `utils/`: shared message normalization, prompt, JSON, and SSE helpers.
- `internal/modules/providers/`: Gemini Web cookies/session bootstrap, account-specific model discovery, RPC generation, uploads, image downloads, refresh, and deep research.
- `internal/modules/dashboard/`: local-only control routes and embedded HTML/CSS/JS; UI tests the existing OpenAI API.
- `internal/modules/gemini/`: `/gemini/v1beta` routes, native DTOs, prompt/tool bridge, and in-memory research interactions.
- `internal/modules/openai/`: `/openai/v1` chat, models, and image-generation compatibility routes/DTOs.
- `internal/modules/claude/`: `/claude/v1` messages, models, and token-count compatibility routes/DTOs.
- `tests/e2e/`: live cross-API checks; `docs/` and `examples/`: image guidance and client examples; `.github/workflows/go-checks.yml`: PR test/vet/build gate.

# Key Flows
- Startup: `cmd/server/main.go` -> config/logger/key manager -> Fiber routes + protocol modules -> provider client; provider initialization retries asynchronously after server start.
- Generation: gateway key middleware -> protocol controller -> protocol service/DTO conversion -> `providers.Client` -> Gemini Web RPC; model names resolve from account-specific discovery.
- Research: Gemini deep-research route -> sequential plan/sub-questions/synthesis; `/interactions` runs this in the background and stores jobs only in memory.

# Commands
- Set `GEMINI_COOKIES` with `__Secure-1PSID` and `__Secure-1PSIDTS` before starting; see `.env.example` for options.
- `API_AUTH_ENABLED=true` by default; set `API_KEY` or copy the generated key from the local dashboard. Generated state lives in `.gateway/`.
- Run: `go run cmd/server/main.go`; build: `go build -o gemini-web-to-api cmd/server/main.go` (also `task run`, `task dev`, `task build`).
- Standard suite: `go test ./...`; live suite: `E2E=1 go test ./tests/e2e -v -count=1 -timeout 15m` (or `task e2e`).

# Conventions & Gotchas
- `.env` is loaded if present. `GEMINI_COOKIES` is mandatory at startup; `GEMINI_AUTH_USER` is a nonnegative numeric account slot. Keep credentials out of code and docs.
- Default listener is `127.0.0.1:4981`; Compose exposes the host port on loopback. `APP_ENV=production` selects production log format.
- Dashboard defaults on only for loopback listeners; Compose sets `DASHBOARD_LOCAL_PORT=true` for its loopback host mapping. A Host header is not a network boundary: disable the dashboard or expose only API paths through a proxy for remote deployments.
- Gateway state `.gateway/` and cookie cache `.cookies/` are private credential material ignored by Git; Compose persists both in named volumes.
- `/health` checks only HTTP service availability, not provider readiness. Models may be empty and generation may fail until asynchronous initialization succeeds.
- OpenAI and Claude are adapters over the same Gemini client, not separate providers. Their streaming endpoints and Gemini streaming split a completed response into delayed chunks.
- Gemini tool calls use a prompt/JSON bridge. Some accepted Gemini DTO fields, including safety settings and function-response parts, are not forwarded to the provider.
- Model availability is discovered per account; unavailable Pro requests fail rather than fall back to Flash. Image input uses base64 `data:` URLs, not remote image URLs.
- Deep-research sources come from Gemini output; the workflow does not independently fetch cited pages. Interaction jobs disappear on restart and expire after 24 hours.
- Check route modules and `internal/server/static/openapi.json` together when changing endpoints; keep README and `.env.example` defaults aligned with source config.
- Live e2e tests need cookies/network, can be slow, and use real Gemini quota. Docker Compose also expects a populated root `.env`.

# Engineering Rules
- Search current source before assuming documented behavior. Raise correctness-critical ambiguity only when repository evidence cannot resolve it.
- For non-trivial work, define brief steps and verification criteria; reproduce bugs when practical, and report checks actually run.
- Make the smallest correct change using existing patterns; avoid unrelated refactors, formatting, speculative features, and unused artifacts.

# Multi-Agent Routing
- Root/orchestrator: GPT-5.6 Sol High decomposes, integrates, makes final decisions, and reports results.
- Delegate routine exploration, tracing, config/test inspection, implementation analysis, and verification to GPT-5.6 Luna Max in focused, non-overlapping scopes.
- Use GPT-5.6 Terra High only for materially difficult cross-module reasoning, contradictory findings, subtle state/security behavior, or complex integrations.
- Partition edits by file/module ownership; use independent review/test workers when useful. Root should trust reliable scoped findings without repeating broad reads.
