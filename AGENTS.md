# Talon — Agent Guide

A terminal-based AI coding assistant with multi-provider support. TypeScript monorepo (`ai/`) with an OpenTUI framework (`tui/`), Effect-TS architecture, Bun runtime, and Turborepo orchestration.

## Quick Reference

| Command | Description |
|---------|-------------|
| `bun install` | Install all dependencies (workspace root) |
| `bun run dev` | Start in development mode |
| `cd ai/packages/talon && bun run src/index.ts` | Run from source |
| `bun run build` | Full build via Turborepo |
| `cd ai/packages/talon && bun test` | Run tests (talon package) |
| `bun run typecheck` | TypeScript type checking across all packages |
| `turbo run build` | Build all packages |
| `bun run lint` | Lint all packages (oxlint) |
| `cd ai/packages/talon && bun run build -- --single` | Compile standalone binary |
| `bash scripts/install.sh` | Full install (build + setup) |
| `bash scripts/install.sh --quick` | Just copy binaries |
| `bash scripts/install.sh --force` | Full rebuild everything |

## Architecture

```
talon/
├── ai/                        # Main application (Bun monorepo, 14 packages)
│   ├── packages/
│   │   ├── talon/             # Main entry point — the CLI binary
│   │   ├── core/              # Core layer (AI SDKs, DB, PTY, session loop)
│   │   ├── cli/               # CLI command definitions
│   │   ├── server/            # HTTP/SSE server library
│   │   ├── tui/               # TUI application (Solid.js + OpenTUI)
│   │   ├── ui/                # Shared UI components and utilities
│   │   ├── llm/               # LLM utilities and provider abstractions
│   │   ├── plugin/            # Plugin system with auth hooks
│   │   ├── sdk/               # OpenAPI-based SDK (JS + generated)
│   │   ├── script/            # Scripting support
│   │   ├── team-core/         # Team collaboration core
│   │   ├── effect-drizzle-sqlite/  # Drizzle ORM for Effect-TS
│   │   ├── effect-sqlite-node/     # SQLite Node.js bindings
│   │   └── http-recorder/     # HTTP traffic recording
│   ├── specs/                 # Technical specifications
│   ├── script/                # Build and dev scripts
│   ├── .talon/                # Talon local config (agents, skills, context)
│   ├── turbo.json             # Turborepo task config
│   └── package.json           # Workspace root (name: "talon")
│
├── tui/                       # OpenTUI rendering framework (11 packages)
│   ├── packages/
│   │   ├── core/              # Core TUI engine (Zig native + TypeScript)
│   │   ├── core-darwin-arm64/ # Prebuilt native binary (libopentui.dylib)
│   │   ├── solid/             # Solid.js renderer for TUI
│   │   ├── react/             # React renderer for TUI
│   │   ├── keymap/            # Keymap system
│   │   ├── web/               # Web renderer
│   │   ├── ssh/               # SSH TUI server
│   │   ├── three/             # Three.js WebGPU renderer
│   │   ├── qrcode/            # QR code renderable
│   │   ├── spinner/           # Spinner component
│   │   └── examples/          # Example applications
│   └── package.json           # Workspace root (name: "@tui")
│
├── scripts/                   # Build, install, and dev helper scripts
│   ├── install.sh             # End-user install script
│   └── talon                  # Dev launcher wrapper
│
├── assets/                    # Logo and branding assets
├── .claude/                   # Project configuration
├── .github/                   # CI, issue templates, dependabot
├── AGENTS.md                  # This file
└── README.md                  # Project readme
```

### Control/Data Flow

1. **CLI startup** → `ai/packages/talon/src/index.ts` → loads config, discovers providers and skills, launches TUI or processes prompt
2. **Agent loop** → Session agent receives prompt → sends to LLM → executes tool calls → feeds results back → loops until done
3. **Tool execution** → Tools (edit, bash, search, etc.) run with permission management and return structured results
4. **Event streaming** → Agent publishes events via pub/sub → UI streams updates via SSE
5. **Message persistence** → Messages written to SQLite via Effect-TS + Drizzle ORM, debounced writes
6. **Hooks** → Lifecycle hooks gate, rewrite, or intercept tool calls, messages, and permissions
7. **Skills** → Loaded from `.talon/` config paths or built-in, injected into system prompt
8. **LSP** → Language Server Protocol client provides diagnostics and references during editing
9. **MCP** → Model Context Protocol servers (local commands or remote OAuth) extend tool capabilities

## Package Overview

### `ai/packages/talon` — Main Application
- **Entry**: `src/index.ts` — CLI bootstrap
- **Build**: `bun run build` — compiles standalone binary via `bun build --single`
- **Version**: `1.17.8` (current)
- **Key deps**: Effect-TS, Solid.js, @ai-sdk/* providers, @tui/core, @tui/keymap, @tui/solid
- **Subdirectories**: `agent/`, `cli/`, `config/`, `session/`, `tool/`, `skill/`, `server/`, `lsp/`, `mcp/`, `plugin/`

### `ai/packages/core` — Core Data Layer & Config
- **Exports**: `@talon-ai/core` — public API, session runner, intent, system context, repomap, hashline, config, tool executors
- **Key deps**: Effect-TS, AI SDKs, Drizzle ORM, node-pty
- **Subdirectories**: `config/`, `session/`, `tool/`, `plugin/`, `evidence/`, `wisdom/`, `repomap/`, `loop/`

### `ai/packages/tui` — TUI Application
- **Framework**: Solid.js rendering on OpenTUI
- **Key features**: Command palette, session sidebar, chat view, dialogs, permission prompts

## Key Packages & Patterns

### Runtime
- **Bun** (`bun@1.3.14`) — JavaScript/TypeScript runtime and bundler
- **Effect-TS** — Functional programming with structured concurrency, SQL, and STM
- **Solid.js** — Reactive UI framework for the TUI
- **OpenTUI** — Custom TUI engine with Zig native core + TypeScript bindings

### Build System
- **Turborepo** (`turbo.json`) — Orchestrates `typecheck`, `build`, `test`, `lint` across packages
- **oxlint** — Fast linter (no `eslint` dependency)
- **tsgo** — TypeScript type checker used alongside `tsc`
- **Bun** — Package manager and bundler

### Database
- **SQLite** via `effect/sql-sqlite-bun` (runtime) + **Drizzle ORM** (migrations)
- Conditional imports via `#sqlite` and `#pty` in `package.json` for platform-specific implementations

### Testing
- **bun test** — Built-in test runner
- VCR-style recording for LLM API tests
- Snapshot tests for TUI components
- Tests use `bun test --timeout 30000 --only-failures`

### Code Style
- **oxfmt** — Code formatting for TUI packages
- **prettier** — Code formatting for AI packages
- **oxlint** — Linting (configured in `.oxlintrc.json` in both `ai/` and `tui/`)
- **TypeScript** — Strict mode across all packages

### Configuration
- **Config file**: `~/.talon/config.json` (auto-created on first run)
- **Config directory**: `~/.talon/` — also holds `.env` for API keys
- **Data directory**: `~/.local/share/talon/` — identity, session data
- **Talon local config**: `ai/.talon/` — agents, sub-agents, skills, context, commands
- **Schema**: Full config schema supports `provider`, `model`, `vision_model`, `agent`, `mcp`, `lsp`, `permission`, `shell`, `server`, `plugin`, `skills`, `snapshot`, `autoupdate`, and more

## Important Notes

### Cross-Workspace References
- `ai/packages/talon/` and `ai/packages/tui/` reference `tui/` packages via `workspace:*` protocol:
  ```
  "@tui/core": "workspace:*",
  "@tui/core-darwin-arm64": "workspace:*",
  "@tui/keymap": "workspace:*",
  "@tui/solid": "workspace:*",
  "@tui/spinner": "workspace:*"
  ```
- When building the standalone binary, Bun needs real directories (not symlinks) for native `.dylib` embedding — `scripts/install.sh` handles this

### Native Library
- `tui/packages/core/src/zig/` — Zig source for `libopentui.dylib`
- Built with `zig build install` (Zig v0.16.x)
- Copied to `tui/packages/core-darwin-arm64/` for workspace consumers
- Prebuilt binary shipped in `tui/packages/core-darwin-arm64/`

### Version
- **Current**: `1.17.8` (`ai/packages/talon/package.json`)
- Monorepo packages are versioned together via changesets

### Gotchas
- **SST config**: `sst.config.ts` at `ai/` root for Serverless Stack deployment
- **Husky**: Git hooks via `ai/.husky/` directory
- **Nix flake**: `flake.nix` + `flake.lock` at `ai/` for reproducible dev environments
- **Platform detection**: Conditional imports (`#sqlite`, `#pty`, `#fff`) in `package.json` for Bun vs Node.js compatibility
