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
- Watermill in-process event bus for future async expansion.
- Optional MCP endpoint at `POST /mcp` (`ENABLE_MCP=true`).

## Quick Start

```bash
go mod tidy
go run ./cmd/api
```

Server defaults to `:8080`.

## Environment Variables

- `HTTP_ADDR` (default `:8080`)
- `HTTP_READ_TIMEOUT` (default `15s`)
- `HTTP_WRITE_TIMEOUT` (default `60s`)
- `LOG_LEVEL` (default `info`, supported: `debug|info|warn|error`)
- `HTTP_ACCESS_LOG_ENABLED` (default `true`)
- `HTTP_SLOW_REQUEST_THRESHOLD` (default `500ms`)
- `OPENAI_BASE_URL` (default `http://localhost:11434/v1`)
- `OPENAI_API_KEY` (optional)
- `OPENAI_MODEL` (default `qwen2.5:7b`)
- `POKEMON_TCG_BASE_URL` (default `https://api.pokemontcg.io/v2`)
- `POKEMON_TCG_API_KEY` (optional)
- `POKEMON_TCG_FALLBACK_RPM` (default `15`)
- `ENABLE_MCP` (default `false`)

All operational logs are emitted via `slog` in JSON format. HTTP request logging is tunable with the logging env vars above.

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

Use `deploy/docker-compose.yml` for a local stack (`api` + `ollama` + `open-webui`).

In Open WebUI, add a custom tool that calls:

- URL: `http://host.docker.internal:8080/v1/grade` (from containerized Open WebUI on Windows)
- Method: `POST`
- JSON body: same as `POST /v1/grade`

## Optional Self-Hosted OpenAI-Compatible Backends

Set `OPENAI_BASE_URL` and `OPENAI_MODEL` to one of:

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
