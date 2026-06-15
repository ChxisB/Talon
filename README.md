<div align="center">

<p>
  <img src="dashboard/public/favicon.svg" alt="" width="80">
</p>

<h1 style="border-bottom: none; margin-bottom: 20px;">Talon</h1>

**A terminal-based AI coding assistant with multi-provider support, screenshot analysis, and a full-featured TUI.**

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?style=for-the-badge&logo=docker)](https://www.docker.com/)

Run any AI model from your terminal — with tools for file editing, bash, search, fetch, vision analysis, LSP, MCP, and more. Built for developers who want a fast, local AI coding experience.

[Quick Start](#quick-start) · [Features](#features) · [CLI Usage](#cli-usage) · [Configuration](#configuration) · [Dashboard](#dashboard) · [Contributing](#contributing)

</div>

---

## Quick Start

### Native (macOS / Linux)

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
go build -o talon .
./talon
```

### Docker

```bash
cd docker
docker compose up -d
```

---

## Features

| Feature | Description |
|---------|-------------|
| **Terminal UI** | Full-featured Bubble Tea TUI with sidebar, chat view, file tree, and session management |
| **Multi-provider** | OpenAI, Anthropic, OpenRouter, Google Gemini, Groq, DeepSeek, Ollama, and more |
| **File editing tools** | View, write, edit, multi-edit files with LSP integration |
| **Bash tool** | Run shell commands with permission management and sandboxing |
| **Search tools** | Glob, grep, ripgrep, web search, and sourcegraph code search |
| **Fetch tool** | Fetch web pages and convert to markdown |
| **Screenshot capture** | Capture your screen and analyze it with AI |
| **Vision analysis** | Local MiniCPM-V integration for image/document analysis via Ollama |
| **MCP server support** | Connect Model Context Protocol servers for expanded tool capabilities |
| **LSP integration** | Language Server Protocol integration for diagnostics and references |
| **Skills system** | Extensible skill-based tool loading |
| **Sub-agents** | Delegate tasks to specialized sub-agents |
| **Session management** | Persistent conversation history with automatic summarization |
| **Context-aware** | Automatic context management with token tracking and memory tree compression |
| **Client/Server mode** | Run as a daemon on a Unix socket with HTTP/SSE streaming |
| **Hooks system** | Gate, approve, or rewrite tool calls before execution |
| **CVE database** | Built-in CVE vulnerability search and tracking |
| **Token optimization** | Response caching, token optimization, and context compression |
| **Dashboard** | Web dashboard with live status, task stats, and configuration |
| **Docker support** | Containerized deployment with docker-compose |

---

## CLI Usage

```bash
# Start the TUI
./talon

# Direct prompt
echo "list all go files in this project" | ./talon
```

Once in the TUI:
- **`Ctrl+P`** — Open command palette
- **`Ctrl+D`** — Toggle details panel
- **`Ctrl+N`** — New session
- **`Tab`** — Switch focus between editor and chat

---

## Configuration

Configuration is stored in `~/.talon/talon.json`. The first run will guide you through provider setup.

### Vision Model Setup (Optional)

For screenshot analysis and image understanding with models that don't support vision natively:

```bash
# Install MiniCPM-V via Ollama
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

---

## Screenshots

<div align="center">
  <table>
    <tr>
      <td align="center"><strong>Dashboard</strong></td>
      <td align="center"><strong>Tasks / Kanban</strong></td>
    </tr>
    <tr>
      <td><img src="dashboard/public/assets/screenshots/dashboard.png" alt="Dashboard" width="380"></td>
      <td><img src="dashboard/public/assets/screenshots/tasks.png" alt="Task Board" width="380"></td>
    </tr>
    <tr>
      <td align="center"><strong>Memory Vault</strong></td>
      <td align="center"><strong>Model Selection</strong></td>
    </tr>
    <tr>
      <td><img src="dashboard/public/assets/screenshots/memory.png" alt="Memory Vault" width="380"></td>
      <td><img src="dashboard/public/assets/screenshots/model-selection.png" alt="Model Selection" width="380"></td>
    </tr>
    <tr>
      <td align="center"><strong>Settings / Configuration</strong></td>
      <td align="center"><strong>Floating Bottom Bar</strong></td>
    </tr>
    <tr>
      <td><img src="dashboard/public/assets/screenshots/settings.png" alt="Settings" width="380"></td>
      <td><img src="dashboard/public/assets/screenshots/floating-bottom-bar.png" alt="Floating Bottom Bar" width="380"></td>
    </tr>
  </table>
</div>

---

## Supported Providers

| Provider | API Key |
|----------|---------|
| **Anthropic** | `ANTHROPIC_API_KEY` |
| **OpenAI** | `OPENAI_API_KEY` |
| **OpenRouter** | `OPENROUTER_API_KEY` |
| **Google Gemini** | `GEMINI_API_KEY` |
| **Groq** | `GROQ_API_KEY` |
| **DeepSeek** | `DEEPSEEK_API_KEY` |
| **Mistral / Codestral** | `MISTRAL_API_KEY` / `CODESTRAL_API_KEY` |
| **Azure OpenAI** | `AZURE_OPENAI_API_KEY` (with endpoint) |
| **AWS Bedrock** | AWS credentials |
| **Ollama** (local) | None (runs locally) |
| **llama.cpp / LM Studio** | None (runs locally) |
| **OpenCode** | `OPENCODE_API_KEY` |
| **Fireworks AI** | `FIREWORKS_API_KEY` |
| **Cerebras** | `CEREBRAS_API_KEY` |
| **NVIDIA NIM** | `NVIDIA_NIM_API_KEY` |
| **Kimi / Moonshot** | `KIMI_API_KEY` |
| **Z.AI** | `ZAI_API_KEY` |
| **Wafer** | `WAFER_API_KEY` |

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
├── internal/
│   ├── agent/             # Agent orchestration, tools, prompts
│   │   ├── tools/         # Tool implementations (bash, edit, view, etc.)
│   │   ├── hyper/         # Provider capability definitions
│   │   └── templates/     # System prompt templates
│   ├── app/               # Application wiring and lifecycle
│   ├── backend/           # Transport-agnostic workspace/agent management
│   ├── client/            # Client for connecting to server
│   ├── cmd/               # CLI entry points (Cobra)
│   ├── config/            # Configuration management
│   ├── db/                # SQLite (sqlc-generated queries)
│   ├── hooks/             # PreToolUse hook runner
│   ├── lsp/               # LSP client manager
│   ├── message/           # Message CRUD (debounced writes)
│   ├── permissions/       # Permission approval dialogs
│   ├── pubsub/            # Typed pub/sub event broker
│   ├── server/            # HTTP/SSE server over Unix socket
│   ├── session/           # Session CRUD service
│   ├── shell/             # Cross-platform shell (mvdan/sh)
│   ├── skills/            # Agent skills discovery
│   ├── ui/                # Terminal UI (Bubble Tea)
│   │   ├── chat/          # Chat message rendering
│   │   ├── dialog/        # Dialog components
│   │   ├── model/         # UI models (header, sidebar, etc.)
│   │   └── styles/        # Theming and styles
│   ├── version/           # Version information
│   └── workspace/         # Workspace abstraction (local vs client/server)
├── deps/                  # Forked Charm dependencies
├── docker/                # Docker configuration
├── dashboard/             # Next.js web dashboard
└── dashboard/public/      # Assets, favicon, screenshots
```

---

## Contributing

Talon is in active development. Contributions are welcome.

### Development

```bash
go build ./...          # Verify compilation
go test ./internal/...  # Run tests
```

### Areas to Help

- 🧪 **Test providers** — Configure different providers and report issues
- 🐛 **Bug reports** — Include model, error message, and reproduction steps
- 🔧 **New tools** — The tool system is extensible; add your own tools
- 🎨 **UI improvements** — The TUI is built with Bubble Tea; contributions welcome
- 📖 **Documentation** — Improve docs, add examples

---

## License

MIT License. See [LICENSE](LICENSE) for details.
