# AIBrow

AIBrow is an AI-assisted photo editor written in Go. It combines FFmpeg-based LCH color analysis with LLM-driven tool orchestration, exposing both a REST API and a web UI.

## Features

- Upload images and keep track of originals, edits, and conversions.
- Analyze images in the LCH color space (lightness, chroma, hue) using FFmpeg.
- Apply perceptual adjustments: brightness, contrast, saturation, gamma, micro-contrast, color temperature.
- Convert images between `png`, `jpg`, `webp`, `tiff`, `bmp`, and `avif`.
- Chat with an LLM through `POST /api/v1/chat`; the model can invoke image tools.
- Natural-language request endpoint `POST /api/v1/request` that uploads an image and lets the LLM orchestrate tools.
- Export analysis/adjustment presets to Excel via `POST /api/v1/images/{id}/export`.
- Provider-agnostic LLM layer supporting OpenAI, Google Gemini, and Ollama.
- 100% Go statement coverage.

## Requirements

- Go 1.26.4 or later
- FFmpeg 8.x or compatible
- For LLM features, one of:
  - `OPENAI_API_KEY`
  - `GEMINI_API_KEY`
  - a local Ollama server (default: `http://localhost:11434`)

## Quick Start

```bash
# install dependencies
go mod tidy

# run with defaults
go run ./cmd/aibrow
```

The REST server starts on `http://localhost:8080` and Swagger UI on `http://localhost:8081`.

## Configuration

Create `configs/.env` or set environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `REST_PORT` | `8080` | REST API port |
| `SWAGGER_PORT` | `8081` | Swagger UI port |
| `UPLOAD_DIR` | `uploads` | Uploaded image storage |
| `OUTPUT_DIR` | `outputs` | Edited/converted/exported files |
| `LOG_PATH` | `logs/error.log` | Application log file |
| `OPENAI_API_KEY` | - | OpenAI API key |
| `OPENAI_BASE_URL` | - | OpenAI-compatible base URL |
| `OPENAI_MODEL` | `gpt-4o` | OpenAI model |
| `GEMINI_API_KEY` | - | Gemini API key |
| `GEMINI_MODEL` | `gemini-1.5-flash` | Gemini model |
| `OLLAMA_BASE_URL` | `http://localhost:11434` | Ollama server URL |
| `OLLAMA_MODEL` | `llama3.1` | Ollama model |

Provider priority when no explicit provider is selected: OpenAI → Gemini → Ollama.

## API Overview

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| POST | `/api/v1/images/upload` | Upload an image |
| GET | `/api/v1/images/{id}` | Download original image |
| POST | `/api/v1/images/{id}/analyze` | Run LCH analysis |
| POST | `/api/v1/images/{id}/adjust` | Apply adjustments |
| POST | `/api/v1/images/{id}/convert` | Convert format |
| GET | `/api/v1/images/{id}/processed` | Download processed image |
| POST | `/api/v1/images/{id}/export` | Export preset Excel |
| POST | `/api/v1/chat` | LLM chat with tool calling |
| POST | `/api/v1/request` | Natural-language edit request |
| GET | `/swagger/index.html` | Swagger UI |

See Swagger for request/response schemas.

## Architecture

```
cmd/aibrow          main entrypoint
internal/app        bootstrap, port selection, server wiring
internal/config     env/config loading
internal/logging    dual file/stdout logger
internal/ports      free-port scanner
src/chat            LLM chat orchestration with tool execution loop
src/ffmpeg          FFmpeg client, LCH analysis, adjustment filters
src/llm             provider-agnostic LLM interface
src/llm/factory     provider selection from environment
src/llm/openai      OpenAI-compatible driver
src/llm/gemini      Google Gemini driver
src/llm/ollama      Ollama driver
src/server          Gin server, REST handlers, Swagger wiring
src/store           in-memory image record repository
src/tools           MCP-style tool registry (analyze, adjust, convert, export)
web                 static frontend
```

## Testing

```bash
go test ./...
```

All packages are kept at 100% statement coverage. To verify:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Benchmarks

Run on the local development environment (Linux arm64):

```bash
go test -bench=. -benchmem ./src/ffmpeg
```

Example output:

```text
BenchmarkAnalyze-8    6    248185077 ns/op    4462868 B/op    254 allocs/op
BenchmarkAdjust-8     8    183869431 ns/op     18024 B/op     99 allocs/op
```

## License

MIT
