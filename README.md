# Chatty

A dead-simple, blazing-fast terminal chat client for OpenAI-compatible APIs written in Go.

## Features

- **Fast startup**: Near-instant launch with minimal overhead (~512 lines of code)
- **Simple & clean**: Intuitive command-line interface with ANSI colors
- **Config-driven**: Loads settings from `config.yaml` with environment overrides
- **Interactive chat**: Real-time conversation with in-memory history
- **Lean architecture**: Just 4 Go files - easy to understand and modify

## Getting Started

### Prerequisites

- Go 1.23 or later
- Access to an OpenAI-compatible API endpoint

### Installation

```bash
go install github.com/PromptShieldLabs/chatty/cmd/chatty@latest
```

Alternatively, clone the repository and build locally:

```bash
git clone https://github.com/PromptShieldLabs/chatty.git
cd chatty
go build ./cmd/chatty
```

### Configuration

Create `config.yaml` in the project root or pass `--config` to specify a path. A minimal configuration looks like:

```yaml
api:
  url: "https://api.openai.com/v1"
  key: "${CHATTY_API_KEY}"
model:
  name: "gpt-4o-mini"
  temperature: 0.7
  stream: true
ui:
  show_timestamps: true
logging:
  level: "info"
```

Environment variables override several fields:

- `CHATTY_API_URL`
- `CHATTY_API_KEY`

### Running

```bash
# Run directly
go run ./cmd/chatty

# Or build and run
go build ./cmd/chatty
./chatty
```

### Available Commands

Once running, you can use these commands:
- `/help` - Show available commands
- `/exit` or `/quit` - Exit the chat
- `/reset` or `/clear` - Clear conversation history
- `/history` - Show conversation history

## Architecture

Chatty follows a lean, simple architecture:

```
chatty/
├── cmd/chatty/
│   └── main.go           # Entry point (~45 lines)
├── internal/
│   ├── config/
│   │   └── config.go     # Config loading (~127 lines)
│   ├── client.go         # OpenAI HTTP client (~122 lines)
│   └── chat.go           # Chat loop + colors (~218 lines)
├── config.yaml
└── go.mod
```

**Total: ~512 lines of production code**

## Development

- Run tests: `go test ./...`
- Build: `go build ./cmd/chatty`
- Format: `go fmt ./...`

## License

TBD
