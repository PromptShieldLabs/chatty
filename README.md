# Chatty

A minimal terminal chat client for OpenAI-compatible APIs, written in Go.

## Why Chatty?

Opening a browser, navigating to ChatGPT or Claude, waiting for the page to load—it all takes time. When you just need a quick answer or want to iterate on an idea, that friction adds up.

Chatty eliminates that friction. It's a terminal-based AI chat client that launches instantly and gets you chatting with AI in milliseconds, not seconds. No Electron, no bloat—just a simple binary that starts instantly and streams responses in real-time. The name emphasizes the TTY (teletypewriter) interface—your terminal.

## What it does

- **Starts instantly** - Compiled Go binary with minimal dependencies
- **Streams responses** - See the text as it arrives, not after the full response
- **Renders markdown** - Code blocks with syntax highlighting, formatted text
- **Supports reasoning models** - Detects and dims thinking tags (`<think>`, `<thinking>`)
- **Works anywhere** - Any OpenAI-compatible API endpoint
- **Simple config** - YAML file with environment variable overrides
- **Persists sessions** - Save and reopen chats with `/list` and `/load`
- **Shell-like editing** - Arrow keys and history recall when running in an interactive terminal
- **Small codebase** - ~1,650 lines of Go, easy to read and modify

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

Or use the provided Makefile for convenience:

```bash
git clone https://github.com/PromptShieldLabs/chatty.git
cd chatty
make build
```

### Configuration

Chatty works with any OpenAI-compatible API provider. Create `config.yaml` in the project root or pass `--config` to specify a path.

#### Using PromptShield

[PromptShield](https://promptshield.io) provides access to multiple AI models through a unified API:

```yaml
api:
  url: "https://api.promptshield.io/v1"
  key: "${CHATTY_API_KEY}"
model:
  name: "openai/gpt-4o-mini"
  temperature: 0.7
  stream: true
```

#### Using OpenAI Directly

```yaml
api:
  url: "https://api.openai.com/v1"
  key: "${OPENAI_API_KEY}"
model:
  name: "gpt-4o-mini"
  temperature: 0.7
  stream: true
```

#### Using Other Compatible Providers

Chatty works with any OpenAI-compatible API. Examples:

**xAI (Grok):**
```yaml
api:
  url: "https://api.x.ai/v1"
  key: "${XAI_API_KEY}"
model:
  name: "grok-beta"
  temperature: 0.7
  stream: true
```

**Anthropic (Direct API):**
```yaml
api:
  url: "https://api.anthropic.com/v1"
  key: "${ANTHROPIC_API_KEY}"
model:
  name: "claude-3-5-sonnet-20241022"
  temperature: 0.7
  stream: true
```

**Local models (Ollama, LM Studio, etc.):**
```yaml
api:
  url: "http://localhost:11434/v1"
  key: "not-needed"
model:
  name: "llama3.2"
  temperature: 0.7
  stream: true
```

#### Environment Variables

Environment variables override config file values:

- `CHATTY_API_URL` - Override the API endpoint
- `CHATTY_API_KEY` or provider-specific keys (e.g., `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`) - Override the API key

### Running

```bash
# Build with version info
make build

# Run
./chatty

# Or build and run directly
make run
```

### Available Commands

Once running, you can use these commands:
- `/help` - Show available commands
- `/exit` or `/quit` - Exit the chat
- `/reset` or `/clear` - Clear conversation history
- `/history` - Show conversation history
- `/markdown` - Toggle markdown rendering on/off
- `/list` or `/sessions` - List saved conversations
- `/load <id>` - Load a saved conversation by its numeric id

## Architecture

Chatty follows a lean, modular layout:

```
chatty/
├── cmd/chatty/
│   └── main.go               # CLI entrypoint & bootstrap (~70 lines)
├── internal/
│   ├── chat.go               # Chat loop, commands, streaming (~670 lines)
│   ├── client.go             # OpenAI-compatible HTTP client (~230 lines)
│   ├── storage/
│   │   └── storage.go        # SQLite persistence layer (~320 lines)
│   └── config/
│       └── config.go         # Config loading & validation (~140 lines)
├── config.yaml               # Sample configuration
└── go.mod                    # Module definition
```

**Total: ~1,650 lines of Go code (tests included)**

## Development

- Run tests: `make test`
- Build: `make build`
- Install: `make install`
- Format: `go fmt ./...`

## License

Released under the [MIT License](LICENSE), which allows you to use, modify, and distribute the code in commercial or personal projects as long as you include the original copyright notice and license text.
