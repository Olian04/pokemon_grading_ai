# Agent context: Pokemon AI grading API (Go)

This document summarizes the project’s purpose, layout, design decisions, and conventions so future AI agents (and humans) can work in the codebase without re-deriving context from chat history.

## Coding standards (required references)

Follow idiomatic Go and explicit error handling. Align with:

- [Go Proverbs](https://go-proverbs.github.io/) — clarity, small interfaces, errors as values, avoid cleverness over clarity.
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) — consistency, naming, error handling patterns, package structure discipline.

When in doubt, prefer boring, explicit code over abstraction for its own sake.

## What this project is

A single-machine **REST API** for **Pokemon TCG card grading** from photos. Design goals:

1. **Deterministic-first**: image heuristics and rules produce scores and evidence before any LLM call.
2. **Optional AI**: OpenAI-compatible HTTP APIs (Ollama, LM Studio, Jan, LocalAI, cloud) for surface assist when rules allow it.
3. **Enrichment**: Pokemon TCG API for card search and US-side pricing signals; EU market fields are structured but may be unavailable until Cardmarket integration is complete.
4. **Operability**: structured JSON logging (`slog`), Prometheus metrics, optional MCP endpoint, Docker Compose for local stack.

## Repository layout (high level)

| Path                                | Role                                                                                                               |
| ----------------------------------- | ------------------------------------------------------------------------------------------------------------------ |
| `cmd/api/main.go`                   | Process entry: load YAML config, init JSON `slog`, graceful shutdown on `SIGINT`/`SIGTERM`, bootstrap HTTP server. |
| `internal/app/bootstrap.go`         | Config load (`LoadConfig`, path discovery), dependency wiring, `http.Server`, cleanup.                             |
| `internal/domain/grading/`          | Grading domain: `Service`, AI gating, expression rules, tests.                                                     |
| `internal/domain/imageproc/`        | Deterministic image analysis (paths on disk).                                                                      |
| `internal/integrations/openai/`     | OpenAI-compatible chat client for AI assist.                                                                       |
| `internal/integrations/pokemontcg/` | Pokemon TCG API client (key + rate-limited fallback).                                                              |
| `internal/integrations/market/`     | Market analytics types and US/EU shaping (EU may return explicit unavailable reasons).                             |
| `internal/transport/http/`          | `ServeMux` routes, logging + metrics middleware.                                                                   |
| `internal/transport/http/handlers/` | JSON handlers for REST endpoints.                                                                                  |
| `internal/transport/mcp/`           | Optional JSON-RPC MCP surface for tools (e.g. `grade_card`).                                                       |
| `internal/observability/metrics/`   | Prometheus registry: custom HTTP metrics + `go_*` / `process_*` collectors.                                        |
| `configs/config.example.yaml`       | Documented default YAML shape.                                                                                     |
| `deploy/docker-compose.yml`         | Compose stack: API + Ollama init + Open WebUI; mounts `deploy/config.compose.yaml`.                                |
| `Makefile`                          | Common tasks (`test`, `run`, `compose-*`, etc.).                                                                   |

## Configuration

### Source of truth

All runtime settings are **YAML**. The only environment variable used for config location is:

- `APP_CONFIG_FILE` — optional explicit path to the YAML file.

If unset, the loader searches predictable locations (see `internal/app/bootstrap.go` — e.g. `pokemon-ai.yaml`, `.pokemon-ai.yaml`, `.pokemon-ai/config.yaml` under cwd, home, and executable directory).

### Example files

- `configs/config.example.yaml` — reference for local dev.
- `deploy/config.compose.yaml` — file mounted into containers by Compose.

### AI gating rules (important)

Three **separate** rule strings in YAML under `ai:`:

- `price_rule`
- `confidence_rule`
- `score_rule`

**Syntax**: each rule is **operator + numeric value only** (no variable names in the string). Whitespace-separated token form, e.g. `>= 20`, `< 0.75`.

**Implicit operands** (bound by code, not repeated in YAML):

| Rule key          | Operand                                                    |
| ----------------- | ---------------------------------------------------------- |
| `price_rule`      | `market_value_usd` (from pricing resolution / market step) |
| `confidence_rule` | deterministic analysis `confidence`                        |
| `score_rule`      | `overall_proxy` (weighted subscores before AI merge)       |

**Evaluation order** (must stay consistent if you change code):

1. Evaluate `price_rule` against `market_value_usd`.
2. If false → **do not call AI**; set `skipped_reason` to `low_value` (HTTP still returns 200 with a normal JSON body — intentional product behavior).
3. If true → evaluate `confidence_rule` and `score_rule`.
4. Call AI **only if both** confidence and score rules are true (**AND**), and an AI client is configured.

Rules are validated at config load time (`ValidateExpression`).

## HTTP API

Implemented on `net/http` + `http.ServeMux` (Go 1.22+ style patterns):

| Method + pattern             | Handler purpose                               |
| ---------------------------- | --------------------------------------------- |
| `GET /healthz`               | Liveness                                      |
| `POST /v1/grade`             | Grade from image paths + optional card hints  |
| `GET /v1/cards/search`       | Query param `q` → TCG search                  |
| `GET /v1/cards/pricing/{id}` | Pricing summary; `id` via `r.PathValue("id")` |
| `GET /metrics`               | Prometheus scrape                             |
| `POST /mcp`                  | Optional MCP (YAML `mcp.enable`)              |

**Why method-qualified patterns**: `r.Pattern` then reflects stable strings like `GET /v1/cards/pricing/{id}` for logs/metrics, avoiding raw URL path cardinality.

### Metrics label design

HTTP counters/histograms use labels **`route`** (matched pattern from `r.Pattern`, fallback `unmatched`) and **`status`**. Raw `r.URL.Path` is **not** used as a Prometheus label (cardinality risk).

### Logging

- Global **`slog` JSON** to stdout; level from YAML `logging.level`.
- Access and slow-request logs include `method`, `route` (pattern), `status`, `duration_ms`.

### Response capture for status codes

`internal/observability/metrics/http.go` defines `statusWriter` wrapping `http.ResponseWriter` so middleware can read the final status after handlers run (needed for metrics and logs).

## Grading pipeline (conceptual)

1. **Image analysis** (`internal/domain/imageproc`): deterministic subscores + confidence + evidence strings (filesystem image paths).
2. **Market / price context**: TCG search + pricing to derive `market_value_usd` and populate `GradeResponse.market` (US/EU structure).
3. **Rule gating**: parse/evaluate the three AI rules; decide skip vs AI.
4. **AI assist** (optional): adjusts surface-related scoring path when allowed; sets `ai_used`, merges evidence, updates `deterministic_only`.

Domain types live in `internal/domain/grading`; keep HTTP types thin in handlers.

## Integrations

- **Pokemon TCG**: `internal/integrations/pokemontcg` — API key header when set; otherwise client-side rate limiter.
- **OpenAI-compatible**: `internal/integrations/openai` — `POST {base}/chat/completions`, JSON surface assist prompt.
- **Market**: `internal/integrations/market` — builds analytics object; EU may be explicitly unavailable until full Cardmarket wiring exists.

## MCP (optional)

When enabled, `POST /mcp` exposes minimal JSON-RPC: `tools/list`, `tools/call` for `grade_card`. This is a lightweight sidecar to REST, not a replacement for the product API.

## Local development

- `go test ./...`
- `make help` — see `Makefile` targets.
- `make run APP_CONFIG=...` — runs API with `APP_CONFIG_FILE` set.

## Docker

`deploy/docker-compose.yml` mounts `deploy/config.compose.yaml`, sets `APP_CONFIG_FILE`, includes Ollama model pull bootstrap and Open WebUI. Requires Docker CLI on the host to validate (`docker compose config`).

## Git ignore

`.gitignore` excludes build artifacts, local env files, local config copies, coverage, and IDE noise while keeping committed examples under `configs/` and `deploy/`.

## Known gaps / extension points

- **Cardmarket EU**: response shape supports EU analytics; full authenticated API + product ID mapping may still be TODO — preserve explicit `unavailable_reason` rather than silent fallback if policy is “EU required”.
- **Grading rubric**: deterministic scores are heuristic; tune `internal/domain/imageproc` and rubric mapping with tests.
- **MCP spec**: current handler is minimal; align with full MCP transport spec if clients require it.

## Quick orientation checklist for agents

1. Read this file and the two style links above.
2. Read `configs/config.example.yaml` for YAML contract.
3. Trace `cmd/api` → `internal/app/bootstrap.go` → `internal/transport/http/router.go` → handlers.
4. Read `internal/domain/grading/service.go` for gating + response shape.
5. Run `go test ./...` before claiming changes are safe.

---

_Document generated from project evolution and conversation context; update when behavior or config contract changes._
