# Pokemon AI Grading API (Go)

Windows-first local API server for Pokemon card grading with deterministic image analysis first, optional OpenAI-compatible AI fallback, and PokemonTCG enrichment.

## Features

- REST API for grading and card data:
  - `POST /v1/grade`
  - `GET /v1/cards/search?q=...`
  - `GET /v1/cards/pricing/{id}`
  - `GET /healthz`
- Deterministic-first grading:
  - centering, corners, edges, surface heuristic subscores
  - dual output: numeric proxy (`1-10`) and seller condition (`Mint/NM/LP/MP/HP/Damaged`)
- OpenAI-compatible AI fallback when deterministic confidence is low.
- PokemonTCG integration with:
  - `POKEMON_TCG_API_KEY` support
  - automatic client-side rate-limited fallback when key is absent.
- Optional MCP endpoint at `POST /mcp` (`ENABLE_MCP=true`).

## Quick Start

```bash
go mod tidy
go run ./cmd/api
```

Server defaults to `:8080`.

## Configuration

Configuration is loaded from YAML.

- Optional env var: `APP_CONFIG_FILE`
  - If set, specifies the explicit config file path to load (overrides search).
  - If omitted, the app searches for a config file (in this order):
    - current working directory: `pokemon-ai.yaml`, `.pokemon-ai.yaml`, `.pokemon-ai/config.yaml`
    - user home directory: `pokemon-ai.yaml`, `.pokemon-ai.yaml`, `.pokemon-ai/config.yaml`
    - executable directory: `pokemon-ai.yaml`, `.pokemon-ai.yaml`, `.pokemon-ai/config.yaml`
  - First readable config file found is used, else start fails with error.

Copy `configs/config.example.yaml` to one of those locations and edit values.

All operational logs are emitted via `slog` in JSON format. HTTP logging and level are configured in YAML under `logging`.
Prometheus metrics are available at `GET /metrics`.

## AI Gating Rules

AI usage is controlled by three expression strings in YAML:

- `ai.price_rule` (evaluated first)
- `ai.confidence_rule`
- `ai.score_rule`

Supported operators: `<`, `<=`, `>`, `>=`.
Rules are written without variable names (operator + value only).

Implicit metrics:

- `price_rule` uses `market_value_usd`
- `confidence_rule` uses `confidence`
- `score_rule` uses `overall_proxy`

Evaluation order:

1. If `price_rule` is false, AI is skipped and response includes `skipped_reason=low_value`.
2. If `price_rule` is true, AI is used only when `confidence_rule AND score_rule` are both true.

## API Examples

### Grade a card

```bash
curl -X POST http://localhost:8080/v1/grade \
  -H "Content-Type: application/json" \
  -d '{
    "front_image_path":"assets/pokemon_card_front.png",
    "back_image_path":"assets/pokemon_card_back.png",
    "card_name_hint":"Pikachu",
    "set_code_hint":"base1",
    "card_number_hint":"58"
  }'
```

### Search cards

```bash
curl "http://localhost:8080/v1/cards/search?q=charizard"
```

### Get pricing

```bash
curl "http://localhost:8080/v1/cards/pricing/base1-4"
```

## Optional Open WebUI Integration

Use `deploy/docker-compose.yml` for a fully provisioned local stack (`api` + `ollama` + `open-webui`).

This compose setup now:

- mounts a YAML config file into API (`deploy/config.compose.yaml`)
- bootstraps Ollama and pre-pulls the configured model
- starts Open WebUI after API health checks pass

In Open WebUI, add a custom tool that calls the grading endpoint:

- URL: `http://host.docker.internal:8080/v1/grade` (from containerized Open WebUI on Windows)
- Method: `POST`
- JSON body: same as `POST /v1/grade`

## Optional Self-Hosted OpenAI-Compatible Backends

Set `openai.base_url` and `openai.model` in YAML to one of:

- Ollama (`http://localhost:11434/v1`)
- LM Studio local server (`http://localhost:1234/v1`)
- Jan local API (`http://localhost:1337/v1`)
- LocalAI (`http://localhost:8080/v1`)

## Optional MCP Endpoint

Enable with:

```bash
set ENABLE_MCP=true
go run ./cmd/api
```

MCP JSON-RPC methods currently supported:

- `tools/list`
- `tools/call` with tool `grade_card`

## Development

```bash
go test ./...
```
