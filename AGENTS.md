# Talon — Agent Guide

A terminal-based AI coding assistant with multi-provider support. Go binary (`talon`), with a Next.js dashboard, vendored TUI stack, SQLite persistence, and client/server modes.

## Quick Reference

| Command | Description |
|---------|-------------|
| `task build` | Build binary into `./talon |
| `task run` | Build + run with optional `-- CLI_ARGS` |
| `task test` | `go test -race -failfast ./...` |
| `task lint` | `golangci-lint` with `.golangci.yml` |
| `task fmt` | `gofumpt -w .` (all Go) |
| `task fmt:html` | `prettier` on HTML/CSS/JS |
| `go run .` | Quick dev (sets `CGO_ENABLED=0`; use `env CGO_ENABLED=0 go run .` from outside task) |
| `./talon | Launch TUI |
| `echo "prompt" \| ./talon | Non-interactive mode |
| `./talonrun "prompt"` | Non-interactive (alias) |
| `./talon--session <id>` | Continue a session |
| `./talon--continue` | Continue most recent session |
| `./talon--yolo` | Auto-accept all permission prompts |
| `./talon--cwd /path` | Set working directory |
| Dashboard | `cd dashboard && npm run dev` (port 3000) |
| Docker | `cd docker && docker compose up -d` (agent + dashboard + cache) |

## Architecture

```
main.go
  └─ internal/cmd/           # Cobra CLI entry points
       ├── root.go           # TUI mode (default), flags, workspace setup
       ├── server.go         # Server mode (HTTP/SSE on Unix socket)
       ├── run.go            # Non-interactive prompt mode
       ├── login.go          # OAuth flows
       ├── session.go        # Session management
       └── ...
  └─ internal/
       ├── app/              # Wires services, lifecycle, provider setup
       ├── agent/            # Core agent orchestration
       │    ├── coordinator.go   # Coordinator interface (Run, Cancel, Queue)
       │    ├── agent.go         # SessionAgent — per-session agent loop
       │    ├── prompts.go       # System prompts (coder, task, init)
       │    ├── tools/           # Tool implementations (bash, edit, view, etc.)
       │    │    ├── tools.go    # Context helpers, permission responses
       │    │    ├── bash.go     # "bash" — shell execution (mvdan/sh)
       │    │    ├── edit.go     # "edit" — find-replace edits
       │    │    ├── view.go     # "view" — read file with line numbers
       │    │    ├── multiedit.go # "multiedit" — batch find-replace
       │    │    ├── write.go    # "write" — create/overwrite files
       │    │    └── ...         # glob, grep, ls, fetch, screenshot, etc.
       │    ├── templates/       # Embedded system prompt templates (.tpl)
       │    └── hyper/           # Provider capability definitions
       ├── backend/          # Transport-agnostic workspace/agent management
       ├── server/           # HTTP/SSE server over Unix socket
       ├── client/           # Client for connecting to server
       ├── config/           # ~/.talon/talon.json config
       ├── db/               # SQLite (sqlc-generated queries)
       ├── session/          # Session CRUD service
       ├── message/          # Message CRUD (debounced writes to SQLite)
       ├── pubsub/           # Typed pub/sub event broker
       ├── hooks/            # PreToolUse hook runner (shell commands gate tool calls)
       ├── skills/           # Agent Skills open standard discovery
       ├── shell/            # Cross-platform shell (mvdan/sh interpreter)
       ├── lsp/              # LSP client manager (diagnostics, references)
       ├── lsp/              # LSP client manager
       ├── ui/               # Bubble Tea TUI
       │    ├── model/           # Application models (chat, sidebar, header, etc.)
       │    ├── chat/            # Chat message rendering
       │    ├── dialog/          # Dialog components
       │    ├── styles/          # Theme and style definitions
       │    └── ...
       ├── workspace/        # Workspace abstraction (local vs client/server)
       ├── permissions/      # Permission requests (user approval dialogs)
       ├── event/            # Telemetry events (PostHog)
       └── ...
  └─ dashboard/              # Next.js web dashboard
  └─ deps/                   # Vendored Charm ecosystem + tools & skills
       ├── glade/                # Memory tree indexing skill (SKILL.md + scripts)
       ├── frugal/               # Token/context optimization (Go: detectors, delta, estimation)
       ├── cache/                # SQLite-backed cache server (Go, Docker)
       ├── cvefree/              # CVE vulnerability database (download + search)
  └─ docker/                 # Dockerfiles and docker-compose.yml
```

### Control/Data Flow

1. **CLI startup** → `internal/cmd/root.go` → `setupWorkspace()` creates either an in-process `app.App` or connects to a server process via `internal/client` depending on `TALON_CLIENT_SERVER`. Skills discovered here.
2. **Agent loop** → `internal/agent/coordinator.go::Run()` → creates/gets `SessionAgent` → `agent.go::Run()` → sends prompt to LLM → executes tool calls → feeds results back to LLM → loops until done.
3. **Tool execution** → tools receive context with session ID, message ID, permissions service, etc. → return `fantasy.ToolResponse` → coordinator feeds back to LLM.
4. **Event streaming** → agent publishes `notify.RunComplete` events → backend streams via SSE to clients → message updates published via `pubsub` broker.
5. **Message persistence** → messages debounce writes (33ms default) to SQLite + publish events. Terminal-state updates flush synchronously.
6. **Hooks** → `PreToolUse` hooks run shell commands before each tool call, can allow/deny/rewrite input or halt the turn.
7. **Skills** → discovered at session start from config paths, injected into system prompt as instructions.
8. **Client/Server mode** → optional; enabled via `TALON_CLIENT_SERVER=1`. Server runs as daemon on Unix socket.

## Key Packages & Aliases

The project uses consistent import aliases for vendored deps:

```go
llm     "github.com/ChxisB/talon-proxy/deps/llm"         // LLM library
style   "github.com/ChxisB/talon-proxy/deps/style/v2"    // Styling (Lipgloss fork)
bubble  "github.com/ChxisB/talon-proxy/deps/ui/terminal/v2" // Bubble Tea TUI
cfg     "github.com/ChxisB/talon-proxy/deps/config/v2"    // Config (Cobra wrapper)
term    "github.com/ChxisB/talon-proxy/deps/terminal"     // Terminal library
trm     "github.com/ChxisB/talon-proxy/deps/terminal"     // In files that also import deps/util/term
lip     "github.com/ChxisB/talon-proxy/deps/style/v2"     // In files with local `style` variable conflicts
```

## Sidebar Layout (`internal/ui/model/sidebar.go`)

- **Model info** at top: provider, reasoning effort, context %, cost, token breakdown
- **Files, LSP, MCP, Skills** sections in middle (dynamically sized)
- **"Talon (version)"** at the very bottom in the SessionTitle style
- Logo was removed from sidebar (was previously at top)

## Configuration

- **Config file**: `~/.talon/talon.json` (auto-created on first run)
- **Env file**: `~/.talon/.env` (API keys, loaded via godotenv/autoload)
- **Data directory**: `$XDG_DATA_HOME/talon/` or `~/.local/share/talon/`
- Providers configured in talon.json under `providers.*`
- Selected model config under `agents.coder.model`
- Tools, LSP, MCP, hooks, skills all configured under their respective keys
- Token-saving techniques: **memory-tree** (input compression), **token-optimizer** (context optimization), **response-cache** (SQLite cache)
- Security: **cve** tool queries the CVE vulnerability database. Dashboard at `/security` for search, filter, severity tracking.
- Dashboard at `/tools` has toggles for all built-in tools (auto-enabled by default)
- Schema generated via `task schema` → `schema.json`

## Important Patterns & Gotchas

### Module Structure
- The main module is `github.com/ChxisB/talon-proxy`
- All `deps/` packages are vendored (forked Charm ecosystem packages) and referenced as module subpaths
- `go.work` exists with `use .` — likely for the dashboard or local development
- Uses Go 1.26 features (like `context.Context` methods on testing.T: `t.Context()`)
- `GOEXPERIMENT=greenteagc` is set in Taskfile for all builds

### Testing
- Uses **VCR recording** for LLM API tests (see `deps/util/vcr` and `internal/agent/agent_test.go`)
- Uses **catwalk** for snapshot-based testing (forked testing framework in `deps/testing/pkg/catwalk`)
- Uses **testify** (`require`/`assert`) for assertions
- Golden files for TUI snapshot tests live in `testdata/` directories near their tests
- Some tests skip on Windows (`t.Skip("skipping on windows for now")`)
- `TestMain` sets slog to Error level to suppress noise
- Message service updates are debounced; tests must call `Flush()` before reading

### Code Style
- **gofumpt** enforced — imports in groups, trailing blank lines, strict formatting
- **goimports** runs in CI on top of gofumpt
- Formatting via `task fmt` runs `gofumpt -w .`
- Prefer `slog` for logging (configured in `internal/log`)
- No `errcheck` or `unused` linters in CI (disabled in `.golangci.yml`)

### Cross-Platform
- Unix/Windows split files: `*_unix.go` / `*_windows.go`
- Shell uses `mvdan.cc/sh/v3` for POSIX emulation on all platforms including Windows
- Socket path limited to 104 bytes (macOS sun_path limit)
- Windows uses named pipes instead of Unix sockets

### Embedded Assets
- Tool descriptions embedded via `//go:embed`. Some are `.md` (static), others `.md.tpl` (Go templates)
- System prompts embedded as `.tpl` Go templates with variables
- Provider capabilities in `internal/agent/hyper/provider.json` (generated via `task hyper`)

### Version
- Default version is `0.1.0` (`internal/version/version.go`)
- Release builds set version via `-ldflags="-X github.com/ChxisB/talon-proxy/internal/version.Version=vX.Y.Z"`
- `go install` builds get version from build info
- BuildID derived from executable modification time for dev builds

### Notable Gotchas
- **slog discard workaround**: `config.Load` uses slog internally, but the file logger isn't ready yet, so slog is discarded during setup (see `internal/cmd/root.go:171`)
- **`ghAvailable`** is cached at `tools` package init time — won't detect installs mid-session
- **Coordinator** currently only supports the "coder" agent type; `AgentTask` config exists but isn't wired
- **Race detection** via `-race` flag on tests is standard; a `race.log` file at project root enables race flag for builds too
- **Permissions** `yolo` mode skips all permission prompts (dangerous)
- **LSP** integration means edits trigger diagnostics, which stream to the UI
- **Dashboard API routes** proxy to the agent's HTTP server (Unix socket). `dashboard/src/app/api/talon-proxy/` routes exist for tools, tasks, status, models, knowledge, health, cron, chat, diagrams, agents, admin, activity.

### Go Workspace
A `go.work` file exists at root, meaning multiple modules could be developed simultaneously. Currently only `use .` — the deps are referenced as module subpaths via `replace` directives in `go.mod`.
