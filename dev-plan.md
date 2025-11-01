# Goaichat Development Plan

## Overview
- **Goal** Build a terminal-based AI chat client in Go that interacts with an OpenAI-compatible API using settings provided via `config.yaml`.
- **Primary outcomes** Provide an interactive chat session with conversational history, robust config handling, and a clean developer experience.
- **Assumptions** Users run the app locally inside a terminal emulator with network access to the configured API endpoint.

## Functional Requirements
- **Interactive chat loop** Allow continuous message exchange with the AI until the user exits.
- **Config-driven startup** Parse `config.yaml` for `api.url`, `api.key`, model defaults, and future options (e.g., system prompts, temperature, streaming flag).
- **Persistent history** Store chats in a SQLite database located at `/var/log/goaichat/<username>/goaichat.db`, capturing session metadata and full message transcripts.
- **History commands** Provide `/list` to enumerate saved chats and allow selecting or reopening them within the session.
- **Model management** Support `/models` command to display configured/available models, persist the user’s choice, and use it for subsequent requests.
- **Graceful exit controls** Support commands like `/exit`, `/reset`, `/help`.
- **Error feedback** Surface authentication and network issues with actionable terminal messages.

## Non-Functional Requirements
- **Portability** Pure Go solution compatible with Linux/macOS/Windows terminals.
- **Security** Keep API key out of logs; support environment variable overrides for secrets; ensure persisted data has restrictive filesystem permissions.
- **Resilience** Retry transient HTTP failures, validate config on load, handle streaming interruptions.
- **Observability** Include structured logging with verbosity toggle.
## Architecture
### High-Level Components
- **CLI entrypoint** `cmd/goaichat/main.go` handles argument parsing and bootstraps the app.
- **Configuration layer** `internal/config` loads, validates, merges `config.yaml` with environment overrides.
- **Application core** `internal/app` wires config, client, history, and UI controller.
- **Chat service** `internal/chat` manages conversation state, prompt assembly, tool commands, and transcripts.
- **Storage layer** `internal/storage` handles SQLite connections, schema migrations, and CRUD helpers for sessions, messages, and model preferences.
- **OpenAI client** `internal/openai` wraps REST calls (Chat Completions or Responses API) with streaming support.
- **Terminal UI** `internal/ui` renders chat in the terminal; leverages `github.com/charmbracelet/bubbletea` + `lipgloss` for layout.
- **Logging/metrics** `internal/telemetry` centralizes loggers and future metrics hooks.

### Data Flow
```mermaid
dflow LR
    UserInput[(User input)] --> UI[Terminal UI]
    UI -->|Submit| ChatCore[Chat service]
    ChatCore -->|Build request| OpenAIClient[OpenAI client]
    OpenAIClient -->|HTTP call| API[(OpenAI-compatible API)]
    API -->|Response/stream| OpenAIClient
    OpenAIClient --> ChatCore
    ChatCore -->|Update state| Storage[(SQLite history store)]
    Storage --> ChatCore
    ChatCore --> UI
    Config[(config.yaml + env)] --> ConfigPkg[Configuration]
    ConfigPkg --> AppCore[Application core]
    AppCore --> UI
    AppCore --> ChatCore
    AppCore --> OpenAIClient
    AppCore --> Storage
```

## Persistence & Data Storage (`internal/storage`)
- **Data directory** Resolve user-specific path `/var/log/goaichat/<username>/` (configurable override) and ensure directories exist with secure permissions.
- **Database** SQLite file `goaichat.db` managed via `modernc.org/sqlite` (pure Go) for portability; allow build tag to switch to `mattn/go-sqlite3` if desired.
- **Schema**
  - **`sessions`** (`id`, `name`, `created_at`, `updated_at`, `model_name`, `summary`).
  - **`messages`** (`id`, `session_id`, `role`, `content`, `token_count`, `created_at`).
  - **`models`** (`id`, `name`, `display_name`, `provider`, `is_default`, `last_used_at`).
- **Migrations** Ship SQL migration files applied on startup; version using goose/atlas or simple in-app migration runner.
- **Data access** Provide repository interfaces for chat service to create sessions, append messages, list chats, update selected model, and fetch transcripts.
- **Performance** Enable write-ahead logging, tune connection settings, and guard concurrent access with a request queue or mutex.

## Configuration Handling (`internal/config`)
- **Load order** Default values → `config.yaml` → environment overrides (`GOAICHAT_API_KEY`, etc.) → CLI flags.
- **Parsing** Use `gopkg.in/yaml.v3` with a typed struct, e.g., `Config{ API struct{ URL string; Key string }; Model struct{ Name string; Temperature float64; Stream bool }; Logging struct{ Level string } }`.
- **Validation** Ensure non-empty `API.URL`, `API.Key`; validate URL format and ranges (temperature 0-2); provide default fallbacks.
- **Hot reload (future)** Structure API to support reloading without restart.
- **Sample `config.yaml`**
```yaml
api:
  url: "https://api.openai.com/v1"
  key: "${OPENAI_API_KEY}"
model:
  name: "gpt-4o-mini"
  temperature: 0.7
  stream: true
ui:
  show_timestamps: true
logging:
  level: "info"
```

## Terminal UI (`internal/ui`)
- **Framework** Adopt Bubble Tea for TUI state management; use `textinput` component for prompt entry and a scrolling viewport for conversation.
- **Features**
  - **Message formatting** Differentiate user/assistant/system with colors via `lipgloss`.
  - **Streaming display** Render tokens incrementally if streaming enabled.
  - **Shortcut commands** `/reset`, `/config`, `/copy-last`, `/save`, `/list`, `/models`.
  - **Status bar** Show model name, latency, active session, and config hints.
- **Accessibility** Support basic ANSI-only fallback.

## Chat Service (`internal/chat`)
- **State** Maintain slice of `Message{Role, Content, Timestamp}` plus metadata (tokens, response duration).
- **Prompt assembly** Prepend system prompt from config; limit history by token budget using approximate counts and hydrate from stored transcripts when reopening a session.
- **Commands** Parse `/` prefixed input before sending to API, including `/reset`, `/help`, `/exit`, `/list`, `/models`, `/config`, `/copy-last`, `/save`.
- **Persistence integration** Manage session lifecycle, store messages, list sessions, and update preferred model through `internal/storage`.

## OpenAI Client (`internal/openai`)
- **HTTP layer** Use net/http with context cancellation, timeout, retry (exponential backoff) on 5xx.
- **Endpoints** Start with Chat Completions `POST /chat/completions`; optionally add Responses API abstraction.
- **Streaming** Handle Server-Sent Events (SSE) or chunked responses; emit channel of tokens consumed by UI.
- **Auth** Inject `Authorization: Bearer` header from config/environment.
- **Telemetry hooks** Log request duration; hide sensitive payloads in debug output.
- **Testing** Mock transport implementation for deterministic tests.

## Application Core (`internal/app`)
- **Lifecycle** Initialize logger → load config → instantiate client and chat service → start UI program.
- **Dependency injection** Pass config structs or interfaces to decouple packages for testing.
- **Shutdown** Capture signals (Ctrl+C) to flush logs, close session transcripts.

## Implementation Roadmap
- **Milestone 1: Project bootstrap**
  - **Create module** `go mod init github.com/<user>/goaichat`.
  - **Set up CI** (GitHub Actions) running `go test` and lint.
  - **Add basic logging** using `log/slog` or `zerolog`.

- **Milestone 2: Config foundation**
  - **Implement** `internal/config` loader with validation and tests.
  - **Support** environment overrides and `--config` CLI flag.

- **Milestone 3: API integration**
  - **Build** `internal/openai` client with non-streaming support first.
  - **Add** response parsing, error handling, retries.
  - **Write** integration tests against mock server.

- **Milestone 4: Persistence foundation**
  - **Design** SQLite schema and migrations within `internal/storage`.
  - **Implement** repositories for sessions, messages, and models.
  - **Ensure** path resolution to `/var/log/goaichat/<username>/` with permission checks and tests.

- **Milestone 5: Chat loop MVP**
  - **Implement** CLI loop that reads user input, calls chat client, prints responses.
  - **Integrate** persistence for session creation, message storage, `/list`, and `/models` commands.
  - **Instrument** logging and simple metrics.

- **Milestone 6: Terminal UI**
  - **Integrate** Bubble Tea UI replacing basic loop.
  - **Add** viewport rendering, streaming display, command palette.
  - **Polish** keyboard shortcuts and layout.

- **Milestone 7: Enhancements & polish**
  - **Implement** transcript saving, timestamp display, configurable prompts.
  - **Add** optional plugins (command to open last response in $EDITOR).
  - **Harden** error messages and retries; tune tests and docs.

## Testing & Quality Strategy
- **Unit tests** Cover config parsing, validation, command parsing, token budgeting.
- **Integration tests** Use httptest server to simulate OpenAI API scenarios (success, auth error, rate limit, streaming).
- **UI tests** Leverage Bubble Tea test helpers to simulate model update cycles.
- **Static analysis** `golangci-lint` for vet, errcheck, gofmt.
- **Load testing (future)** Add simple script to replay prompts for latency measurement.

## Developer Tooling & DX
- **Makefile** Targets: `make build`, `make test`, `make lint`, `make run`.
- **Task runner** Optionally use `taskfile` or `mage`.
- **Documentation** Maintain `README.md`, `docs/configuration.md`, `docs/development.md`.
- **Sample configs** Provide `config.example.yaml` without secrets.
- **Issue templates** For bug reports and feature requests once repo public.

## Deployment & Distribution
- **Binary releases** Cross-compile via `goreleaser` for Linux/macOS/Windows; ensure `CGO_ENABLED=0` for static binaries.
- **Packaging** Offer Homebrew tap or `go install` instructions.
- **Versioning** Semantic versioning with tagged releases; changelog via `git-chglog`.

## Future Extensions
- **Plugin commands** Shell hooks, sending to external tools, code interpreter.
- **Multi-provider support** Add adapters for Anthropic, local LLMs via OpenAI-compatible bridges.
- **Conversation persistence** SQLite-backed history with search.
- **Team collaboration** Shared sessions via websockets or tmux integration.
- **Telemetry dashboard** Optional Prometheus exporter for usage stats.

## Risks & Mitigations
- **API drift** Mitigate with client abstraction and compatibility tests.
- **Rate limits** Implement exponential backoff, expose retry counters in UI.
- **Secret leakage** Ensure config accepts env vars, redact logs, document best practices.
- **Terminal portability** Provide fallback non-TUI mode for minimal terminals.
