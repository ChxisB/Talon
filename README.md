<div align="center">

<p>
  <img src="assets/logo.svg" alt="" width="80">
</p>

<h1 style="border-bottom: none; margin-bottom: 20px;">Talon</h1>

**A terminal-based AI coding assistant with multi-provider support, screenshot analysis, and a full-featured TUI.**

Run any AI model from your terminal — with tools for file editing, bash, search, fetch, vision analysis, LSP, MCP, and more. Built for developers who want a fast, local AI coding experience.

[Quick Start](#quick-start) · [Features](#features) · [CLI Usage](#cli-usage) · [Configuration](#configuration) · [Contributing](#contributing)

</div>

---

## Quick Start

```bash
git clone https://github.com/chrisbeckett/talon.git
cd talon

# Set at least one API key
export ANTHROPIC_API_KEY=sk-ant-...
# or
export OPENAI_API_KEY=sk-...
# or
export OPENROUTER_API_KEY=sk-or-v1-...

# Build and run
bash scripts/install.sh
talon
```

---

## Features

### AI & Models

| Feature | Description |
|---------|-------------|
| **Multi-provider** | 30+ providers — Anthropic, OpenAI, Google Gemini, Groq, Mistral, xAI, OpenRouter, Ollama, AWS Bedrock, Azure, and more |
| **Vision analysis** | Native support for vision-capable models (Claude, GPT-4o, Gemini) + local MiniCPM-V via Ollama |
| **Screenshot capture** | Capture your screen and analyze it with AI |
| **Model routing** | Configure `model` and `small_model` per-session, with automatic fallback |

### Developer Tools

| Feature | Description |
|---------|-------------|
| **File editing** | View, write, edit, multi-edit, and apply patches with LSP-backed diagnostics |
| **Bash tool** | Run shell commands with permission management and sandboxing |
| **Interactive shell** | Persistent interactive shell sessions |
| **Search** | Glob, grep, ripgrep, AST-grep, web search, and Sourcegraph code search |
| **Fetch** | Fetch web pages and convert to markdown |
| **Diagram generation** | Generate architecture diagrams from code |
| **Todo management** | Track tasks and progress during a session |
| **Workflow orchestration** | Multi-step agent workflows with parallel and sequential steps |
| **Wisdom system** | Persistent knowledge base — save and query learnings across sessions |
| **CVE database** | Built-in CVE vulnerability search and tracking |
| **Policy generation** | Generate security compliance policies (ISO 27001, GDPR, NIS2) |

### Architecture & Extensibility

| Feature | Description |
|---------|-------------|
| **Sub-agents** | Delegate tasks to specialized sub-agents with custom instructions and tools |
| **Skills system** | Extensible skill-based tool loading |
| **Plugin system** | Plugin architecture with auth flows for custom providers (Azure, GitHub Copilot, Cloudflare, Snowflake, xAI, DigitalOcean, OpenAI Codex) |
| **MCP server support** | Connect Model Context Protocol servers — local (command) and remote (with OAuth) |
| **LSP integration** | Full Language Server Protocol client — diagnostics, references, and language support |
| **Hooks system** | 25+ lifecycle hooks — gate, rewrite, or intercept tool calls, messages, permissions, and provider requests |
| **Agent Communication Protocol** | Agent-to-agent protocol for multi-agent coordination |
| **Team collaboration** | Create and manage AI agent teams with messaging and task assignment |

### Session & Context

| Feature | Description |
|---------|-------------|
| **Session management** | Persistent conversation history with automatic summarization and compaction |
| **Context-aware** | Automatic context management with token tracking and memory tree compression |
| **Snapshot system** | Filesystem state tracking for undo/revert |
| **Token optimization** | Response caching, token optimization, and context compression |

### Deployment & Operations

| Feature | Description |
|---------|-------------|
| **Terminal UI** | Full-featured TUI with command palette, session sidebar, and file tree |
| **Client/Server mode** | Run as a daemon on a Unix socket with HTTP/SSE streaming |
| **Headless server** | `talon serve` — HTTP API server for remote access |

| **Account & Cloud** | Talon Cloud account management with OAuth flow |
| **Auto-update** | Built-in version management and self-update (`talon upgrade`) |
| **Background jobs** | Asynchronous background task execution |

---

## CLI Usage

```bash
talon                  # Launch the TUI
talon run <prompt>     # Non-interactive prompt mode
talon serve            # Start headless HTTP server
talon mcp              # Manage MCP servers (list, add, auth)
talon agent            # Create and manage custom agents
talon providers        # List and configure providers
talon models           # List and query available models
talon session          # Manage sessions (list, delete)
talon attach           # Attach to a running session
talon doctor           # System diagnostics
talon plug             # Plugin management (install, uninstall)
talon upgrade          # Self-update
talon generate         # Generate files
talon github           # GitHub integration
talon pr               # Pull request operations
talon export/import    # Export and import sessions
talon db               # SQLite shell or query
talon stats            # Usage statistics
talon acp              # Agent Communication Protocol
talon uninstall        # Uninstall Talon
```

Once in the TUI:
- **`Ctrl+P`** — Open command palette
- **`Ctrl+D`** — Toggle details panel
- **`Ctrl+N`** — New session
- **`Tab`** — Switch focus between editor and chat

---

## Configuration

Configuration is stored in `~/.talon/talon.json` or `~/.talon/talon.jsonc` (supports comments and trailing commas). The first run will guide you through provider setup.

The full config schema supports: `provider`, `model`, `vision_model`, `agent`, `mcp`, `lsp`, `permission`, `shell`, `server`, `plugin`, `skills`, `snapshot`, `autoupdate`, and more. Variable substitution (`{env:VAR}`, `{file:path}`) is supported throughout.

### Vision Model Setup (Optional)

For image/document analysis with models that don't support vision natively:

```bash
ollama run minicpm-v
```

Configure in `talon.json`:
```json
{
  "tools": {
    "vision": {
      "endpoint": "http://localhost:11434/v1/chat/completions",
      "model": "minicpm-v"
    }
  }
}
```

No configuration needed for natively vision-capable models (Claude, GPT-4o, Gemini).

## Supported Providers

| Provider | Env Variable |
|----------|-------------|
| **Anthropic** (Claude) | `ANTHROPIC_API_KEY` |
| **OpenAI** (GPT-4o, o-series) | `OPENAI_API_KEY` |
| **Google Gemini** | `GEMINI_API_KEY` |
| **Google Vertex AI** | `GOOGLE_VERTEX_CREDENTIALS` |
| **AWS Bedrock** | AWS credentials |
| **Azure OpenAI** | `AZURE_OPENAI_API_KEY` (with endpoint) |
| **Mistral / Codestral** | `MISTRAL_API_KEY` |
| **Groq** | `GROQ_API_KEY` |
| **xAI (Grok)** | `XAI_API_KEY` |
| **OpenRouter** | `OPENROUTER_API_KEY` |
| **Together AI** | `TOGETHER_API_KEY` |
| **Fireworks AI** | `FIREWORKS_API_KEY` |
| **DeepInfra** (incl. DeepSeek) | `DEEPINFRA_API_KEY` |
| **Perplexity** | `PERPLEXITY_API_KEY` |
| **Cerebras** | `CEREBRAS_API_KEY` |
| **Cohere** | `COHERE_API_KEY` |
| **Alibaba Cloud** | `ALIBABA_API_KEY` |
| **GitLab AI** | `GITLAB_API_KEY` |
| **GitHub Copilot** | GitHub auth |
| **Venice.ai** | `VENICE_API_KEY` |
| **NVIDIA NIM** | `NVIDIA_NIM_API_KEY` |
| **Talon Cloud** | `TALON_API_KEY` |
| **Ollama** (local) | None |
| **llama.cpp / LM Studio** (local) | None |
| **Any OpenAI-compatible** | Custom endpoint |
| — | — |
| **Plugin providers** (Azure, Cloudflare, DigitalOcean, Snowflake, OpenAI Codex, GitHub Copilot, xAI) | Plugin-specific auth |

Set API keys in `~/.talon/.env`:
```env
ANTHROPIC_API_KEY=sk-ant-...
OPENAI_API_KEY=sk-...
OPENROUTER_API_KEY=sk-or-v1-...
```

---

## Project Structure

```
talon/
├── ai/                   # AI application (14 packages)
│   ├── packages/
│   │   ├── talon/        # Main application (TypeScript/Effect)
│   │   ├── cli/          # CLI command definitions
│   │   ├── core/         # Core data layer & config
│   │   ├── server/       # HTTP API library
│   │   ├── tui/          # TUI framework integration
│   │   ├── ui/           # Shared UI utilities
│   │   ├── llm/          # LLM utilities
│   │   ├── plugin/       # Plugin system & auth hooks
│   │   ├── sdk/          # OpenAPI-based SDK
│   │   ├── script/       # Scripting support
│   │   ├── team-core/    # Team collaboration
│   │   ├── effect-drizzle-sqlite/  # Drizzle ORM for Effect
│   │   ├── effect-sqlite-node/     # SQLite Node.js bindings
│   │   └── http-recorder/          # HTTP traffic recording
│   └── install           # Installer
├── tui/                  # OpenTUI rendering framework (11 packages)
│   ├── packages/
│   │   ├── core/         # TypeScript TUI library on Zig native core
│   │   ├── react/        # React renderer for TUI
│   │   ├── solid/        # SolidJS renderer for TUI
│   │   ├── web/          # Web renderer
│   │   ├── ssh/          # Serve TUIs over SSH
│   │   ├── three/        # Three.js WebGPU renderer
│   │   ├── keymap/       # Keymap system
│   │   ├── qrcode/       # QR code renderable
│   │   ├── spinner/      # Spinner component
│   │   └── examples/     # Example apps
│   └── scripts/          # TUI build scripts
├── scripts/              # Build and install scripts
└── .claude/              # Project configuration
```

---

## Contributing

Talon is in active development. Contributions are welcome.

### Development

```bash
cd ai/packages/talon && bun run src/index.ts  # Run from source
```

### Areas to Help

- 🧪 **Test providers** — Configure different providers and report issues
- 🐛 **Bug reports** — Include model, error message, and reproduction steps
- 🔧 **New tools** — The tool system is extensible; add your own tools
- 🎨 **UI improvements** — The TUI is built with OpenTUI (Solid.js); contributions welcome
- 📖 **Documentation** — Improve docs, add examples

---

## License

MIT License. See [LICENSE](LICENSE) for details.
