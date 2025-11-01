# Goaichat Rewrite Plan: Simplicity First

## Current State Analysis

### What We Have
- **Working components**: Config loading, OpenAI client, basic chat service, CLI loop
- **Architecture**: Multi-layer with `internal/app`, `internal/chat`, `internal/config`, `internal/openai`
- **Planned complexity**: SQLite persistence, Bubble Tea UI, migrations, session management, model switching, command palette

### The Problem: Over-Engineering
The current plan adds significant complexity that may not be needed for a fast, simple chat client:

1. **SQLite persistence** (`/var/log/goaichat/<username>/goaichat.db`)
   - Schema migrations
   - Session/message/model tables
   - Repository patterns
   - Concurrent access handling
   
2. **Bubble Tea TUI**
   - State management overhead
   - Viewport/scrolling logic
   - Component lifecycle
   - Event handling complexity

3. **Multiple abstraction layers**
   - App → Chat Service → OpenAI Client
   - Interface definitions for testing
   - Dependency injection patterns

4. **Feature bloat**
   - `/list`, `/models`, `/save`, `/copy-last` commands
   - Model management system
   - Streaming display with incremental rendering
   - Status bars, timestamps, metrics

## Simplified Vision

### Core Goal
**A dead-simple, blazing-fast terminal chat client that does ONE thing well: chat with an AI.**

### What Actually Matters
1. **Fast startup** - No database initialization, no complex UI setup
2. **Simple interaction** - Type message, get response, repeat
3. **Minimal dependencies** - Standard library + HTTP client
4. **Easy configuration** - One YAML file or env vars
5. **Streaming support** - See responses as they arrive

### What We Can Drop
- ❌ SQLite persistence (use simple file-based history if needed)
- ❌ Bubble Tea TUI (use simple ANSI colors + readline)
- ❌ Session management (one session per run)
- ❌ Model switching UI (set in config)
- ❌ Complex command system (just /exit, /reset, /help)
- ❌ Migration system
- ❌ Repository patterns

## Proposed Simplified Architecture

### Single-File Approach (Option A: Ultra-Minimal)
```
goaichat/
├── main.go              # Everything in ~300-400 lines
├── config.yaml          # Simple config
└── go.mod
```

**Pros:**
- Fastest possible startup
- Zero abstraction overhead
- Easy to understand and modify
- Perfect for personal use

**Cons:**
- Harder to test
- Less modular

### Lean Multi-File (Option B: Pragmatic)
```
goaichat/
├── cmd/goaichat/
│   └── main.go          # Entry point (~50 lines)
├── internal/
│   ├── config.go        # Config loading (~100 lines)
│   ├── client.go        # OpenAI HTTP client (~150 lines)
│   └── chat.go          # Chat loop + history (~100 lines)
├── config.yaml
└── go.mod
```

**Pros:**
- Still very simple
- Testable units
- Clear separation of concerns
- Room to grow if needed

**Cons:**
- Slightly more files to navigate

## Recommended Approach: Option B (Lean Multi-File)

### Implementation Details

#### 1. Config (`internal/config.go`)
```go
type Config struct {
    APIKey      string  // From env or file
    APIURL      string  // Default: https://api.openai.com/v1
    Model       string  // Default: gpt-4o-mini
    Temperature float64 // Default: 0.7
    Stream      bool    // Default: true
}

func Load() (*Config, error) {
    // Load from config.yaml, override with env vars
    // No validation beyond "API key exists"
}
```

#### 2. Client (`internal/client.go`)
```go
type Client struct {
    apiKey  string
    baseURL string
    http    *http.Client
}

func (c *Client) Chat(ctx context.Context, messages []Message, model string, temp float64, stream bool) (string, error) {
    // Direct HTTP call to /chat/completions
    // Handle streaming with SSE if enabled
    // Return complete response or error
}
```

#### 3. Chat Loop (`internal/chat.go`)
```go
type Session struct {
    history []Message
    client  *Client
    config  *Config
}

func (s *Session) Run(ctx context.Context) error {
    // Simple readline loop
    // Handle /exit, /reset, /help
    // Print with basic ANSI colors
    // Maintain in-memory history only
}
```

#### 4. Main (`cmd/goaichat/main.go`)
```go
func main() {
    cfg := config.Load()
    client := NewClient(cfg)
    session := NewSession(client, cfg)
    session.Run(context.Background())
}
```

### Optional: File-Based History (If Needed Later)
Instead of SQLite, use simple append-only JSON files:
```
~/.goaichat/
└── history/
    └── 2024-10-18-12-23.jsonl  # One file per session
```

Each line: `{"role": "user|assistant", "content": "...", "ts": "..."}`

**Benefits:**
- No schema, no migrations
- Easy to grep/search
- Human-readable
- Can implement in ~20 lines

### Streaming Implementation
Keep it simple:
```go
// For streaming: print tokens as they arrive
// For non-streaming: print complete response
// Use a simple spinner while waiting
```

### UI/UX
- **Prompt**: Simple `> ` prefix
- **Colors**: 
  - User input: cyan
  - AI response: green
  - Errors: red
  - Commands: yellow
- **No fancy TUI**: Just ANSI escape codes
- **No viewport**: Terminal's native scrolling

## Migration Path from Current Code

### Phase 1: Consolidate (Keep Working Code)
1. Flatten `internal/openai` → `internal/client.go`
2. Merge `internal/app` + `internal/chat` → `internal/chat.go`
3. Keep `internal/config` as-is (already simple)
4. Remove empty dirs: `storage/`, `telemetry/`, `ui/`

### Phase 2: Simplify
1. Remove interface abstractions (CompletionClient)
2. Remove dependency injection patterns
3. Inline small functions
4. Remove unused options/features

### Phase 3: Polish
1. Add basic ANSI colors
2. Improve error messages
3. Add simple streaming display
4. Write minimal README

## Comparison: Before vs After

| Aspect | Current Plan | Simplified |
|--------|-------------|------------|
| **Files** | ~15-20 Go files | ~4 Go files |
| **Lines of Code** | ~2000-3000 | ~400-600 |
| **Dependencies** | 5-7 external | 1-2 external |
| **Startup Time** | ~100-200ms | ~10-20ms |
| **Features** | 15+ commands | 3 commands |
| **Persistence** | SQLite + migrations | Optional JSONL |
| **UI** | Bubble Tea TUI | ANSI colors |
| **Complexity** | High | Low |

## What We Gain

1. **Speed**: Near-instant startup, no initialization overhead
2. **Simplicity**: Easy to understand, modify, and debug
3. **Reliability**: Fewer moving parts = fewer bugs
4. **Maintainability**: Can read entire codebase in 10 minutes
5. **Focus**: Does one thing exceptionally well

## What We Lose (And Why It's OK)

1. **Persistent history**: Can add simple JSONL later if needed
2. **Fancy UI**: Terminal scrolling works fine
3. **Session management**: One session per run is simpler
4. **Model switching**: Change config file or use env var
5. **Complex commands**: Keep it minimal

## Decision Points

### Do we need history at all?
- **No**: Simplest, fastest
- **Yes**: Add JSONL append-only files later (easy)

### Do we need streaming?
- **Yes**: It's a core feature for good UX
- **Implementation**: Simple SSE parsing, print as we go

### Do we need colors?
- **Yes**: Minimal ANSI codes improve readability
- **Implementation**: ~10 lines of helper functions

### Do we need tests?
- **Yes**: But only for critical paths (config, HTTP client)
- **Keep it minimal**: ~3-4 test files max

## Next Steps

1. **Review this plan** - Does this align with your vision?
2. **Choose approach** - Option A (single file) or Option B (lean multi-file)?
3. **Decide on features** - Which optional features matter most?
4. **Start fresh or refactor?** - Clean slate vs. evolve current code?

## Recommended Action

**Start with Option B (Lean Multi-File):**
1. Keep current working code as reference
2. Create new simplified implementation alongside
3. Test both, compare performance
4. Switch when new version is solid
5. Delete old complexity

This gives us a safety net while moving fast toward simplicity.

---

## Philosophy

> "Perfection is achieved, not when there is nothing more to add, but when there is nothing left to take away." - Antoine de Saint-Exupéry

The best chat client is the one that:
- Starts instantly
- Gets out of your way
- Just works

Everything else is noise.
