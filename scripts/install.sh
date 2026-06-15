#!/usr/bin/env bash
# ───────────────────────────────────────────────────────────────
# Talon — Install & Setup
#   - Fresh install: full setup (prerequisites, build, start)
#   - Already installed: rebuild + refresh identity
#
# Usage: ./scripts/install.sh
# ───────────────────────────────────────────────────────────────
set -euo pipefail

# ─── Paths ────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DOCKER_DIR="$PROJECT_DIR/docker"
DASHBOARD_DIR="$PROJECT_DIR/dashboard"
TALON_BIN=""
for p in "$PROJECT_DIR/talon" "$HOME/.local/bin/talon" "$HOME/.local/bin/talon-proxy"; do
  [ -f "$p" ] && TALON_BIN="$p" && break
done
IS_INSTALLED=false
TALON_CONFIG_DIR="${HOME}/.talon"
TALON_IDENTITY_DIR="${HOME}/.local/share/talon"
if [ -n "$TALON_BIN" ]; then
  if [ -f "$PROJECT_DIR/.env" ] || [ -d "$TALON_CONFIG_DIR" ] || [ -f "${TALON_IDENTITY_DIR}/TALON.md" ]; then
    IS_INSTALLED=true
  fi
fi

# ─── Colors ───────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

info()  { echo -e "${BLUE}⋅${NC} $1"; }
ok()    { echo -e "${GREEN}✓${NC} $1"; }
warn()  { echo -e "${YELLOW}⚠${NC} $1"; }
fail()  { echo -e "${RED}✗${NC} $1"; }
header(){ echo -e "\n${CYAN}${BOLD}$1${NC}"; echo -e "${CYAN}──────────────────────────────${NC}"; }

# ─── Helpers ──────────────────────────────────────────────────
check_cmd() {
  if ! command -v "$1" &>/dev/null; then
    fail "$1 is not installed. Please install it first."
    return 1
  fi
  ok "$1 found: $(command -v "$1")"
}

semver_compare() {
  local a=(${1//./ }) b=(${2//./ })
  for i in 0 1 2; do
    [[ ${a[$i]} -gt ${b[$i]} ]] && return 0
    [[ ${a[$i]} -lt ${b[$i]} ]] && return 1
  done
  return 0
}

# ─── Shared build steps ───────────────────────────────────────

build_native() {
  header "Building native Go binary"
  cd "$PROJECT_DIR"

  if command -v go-task &>/dev/null; then
    info "Using go-task..."
    go-task build
  elif command -v task &>/dev/null; then
    info "Using task..."
    task build
  else
    info "Running: CGO_ENABLED=0 go build -v -o talon ."
    CGO_ENABLED=0 GOEXPERIMENT=greenteagc go build -v -o talon .
  fi

  if [ -f "$PROJECT_DIR/talon" ]; then
    ok "Native binary built: $PROJECT_DIR/talon"
    mkdir -p "$HOME/.local/bin"
    install -m 755 "$PROJECT_DIR/talon" "$HOME/.local/bin/talon"
    # Also install as talon-proxy for compatibility with go-install users
    install -m 755 "$PROJECT_DIR/talon" "$HOME/.local/bin/talon-proxy"
    ok "Installed to ~/.local/bin/talon and ~/.local/bin/talon-proxy"
  else
    warn "Native binary not found at expected path"
  fi
}

rebuild_docker() {
  header "Rebuilding Docker images"
  cd "$PROJECT_DIR"

  AGENT_PORT=8082
  CACHE_PORT=8083
  if lsof -i :8082 &>/dev/null 2>&1; then
    AGENT_PORT=8092
    warn "Port 8082 in use — mapping agent to :$AGENT_PORT"
  fi
  if lsof -i :8083 &>/dev/null 2>&1; then
    CACHE_PORT=8093
    warn "Port 8083 in use — mapping cache to :$CACHE_PORT"
  fi
  export AGENT_PORT CACHE_PORT

  info "Building images..."
  docker compose -f "$DOCKER_DIR/docker-compose.yml" build
  info "Restarting services..."
  docker compose -f "$DOCKER_DIR/docker-compose.yml" down --remove-orphans
  docker compose -f "$DOCKER_DIR/docker-compose.yml" up -d --force-recreate

  ok "Docker services rebuilt and restarted!"
  echo ""
  echo "  Dashboard:  ${BOLD}http://localhost:3000${NC}"
  echo "  Proxy API:  ${BOLD}http://localhost:${AGENT_PORT}${NC}"
  echo ""
}

docker_is_running() {
  [ -f "$DOCKER_DIR/docker-compose.yml" ] && command -v docker &>/dev/null && \
    docker compose -f "$DOCKER_DIR/docker-compose.yml" ps --services 2>/dev/null | grep -q .
}

setup_identity() {
  header "Setting up identity"
  local talon_dir="${HOME}/.local/share/talon"
  mkdir -p "$talon_dir"
  cat > "${talon_dir}/TALON.md" << EOF
# TALON.md — Auto-generated from system info

## Identity
- Username: $(whoami)
- Hostname: $(hostname -s 2>/dev/null || hostname)
- Platform: $(uname -s)/$(uname -m)
- Shell: ${SHELL##*/}
- Home: ${HOME}

## System
- OS: $(sw_vers -productName 2>/dev/null || echo "Unknown") $(sw_vers -productVersion 2>/dev/null || echo "")
- Timezone: $(TZ=$(readlink /etc/localtime 2>/dev/null || echo "UTC"); basename "$TZ" 2>/dev/null || echo "UTC")
- Locale: ${LANG:-en_US.UTF-8}
EOF
  ok "Identity saved to ${talon_dir}/TALON.md"
}

# ═══════════════════════════════════════════════════════════════
# REBUILD MODE  (already installed)
# ═══════════════════════════════════════════════════════════════

if $IS_INSTALLED; then
  cat << 'REBUILD'

  ╔══════════════════════════════════════════════════╗
  ║         Talon — Rebuild               ║
  ╚══════════════════════════════════════════════════╝

  Talon is already installed. Rebuilding and refreshing...

REBUILD

  build_native

  if docker_is_running; then
    rebuild_docker
  else
    info "Docker services not running — skipping Docker rebuild."
  fi

  setup_identity

  echo ""
  ok "Rebuild complete!"
  echo ""
  echo "  Run the TUI:  ${BOLD}talon${NC}"
  exit 0
fi

# ═══════════════════════════════════════════════════════════════
# FRESH INSTALL
# ═══════════════════════════════════════════════════════════════

cat << 'WELCOME'

  ╔══════════════════════════════════════════════════╗
  ║              Talon — Install             ║
  ╚══════════════════════════════════════════════════╝

  This script will build and start Talon on
  your machine using either Docker containers or
  running directly on the machine.

WELCOME

# ─── Prerequisites ────────────────────────────────────────────
header "Checking prerequisites"

HAS_DOCKER=false
HAS_GO=false
HAS_NODE=false
HAS_TASK=false

if command -v docker &>/dev/null && docker compose version &>/dev/null 2>&1; then
  if docker info &>/dev/null 2>&1; then
    HAS_DOCKER=true
    ok "Docker is running"
  else
    warn "Docker found but not running (docker daemon not available)"
  fi
else
  warn "Docker not found (optional if installing directly)"
fi

MIN_GO="1.26"
INSTALLED_GO=""
if command -v go &>/dev/null; then
  INSTALLED_GO=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+(\.[0-9]+)?' || true)
  if semver_compare "${INSTALLED_GO}.0" "${MIN_GO}.0"; then
    HAS_GO=true
    ok "Go ${INSTALLED_GO} found"
  else
    warn "Go ${INSTALLED_GO} found, but ${MIN_GO}+ is required"
  fi
else
  warn "Go not found (required for direct install)"
fi

MIN_NODE="22"
INSTALLED_NODE=""
if command -v node &>/dev/null; then
  INSTALLED_NODE=$(node --version | grep -oP 'v\K[0-9]+' || true)
  if [ "${INSTALLED_NODE}" -ge 22 ] 2>/dev/null; then
    HAS_NODE=true
    ok "Node $(node --version) found"
  else
    warn "Node $(node --version) found, but v22+ is required"
  fi
else
  warn "Node.js not found (required for direct install)"
fi

if command -v task &>/dev/null; then
  HAS_TASK=true
  ok "go-task found"
else
  info "go-task not found — will use raw go commands instead"
fi

echo ""

# ─── Choose installation method ───────────────────────────────
header "Choose installation method"

if $HAS_DOCKER; then
  echo "  1) Docker (recommended) — containers for proxy + dashboard"
  echo "     Requires: Docker Desktop (✓ detected)"
  echo ""
fi
if $HAS_GO && $HAS_NODE; then
  echo "  2) Direct on machine — run proxy + dashboard natively"
  echo "     Requires: Go ${MIN_GO}+ (${INSTALLED_GO:+✓ $INSTALLED_GO detected})"
  echo "               Node v${MIN_NODE}+ (${INSTALLED_NODE:+✓ v$INSTALLED_NODE detected})"
  echo ""
fi

VALID_CHOICES=""
OPT_DOCKER=""
OPT_DIRECT=""

if $HAS_DOCKER; then
  VALID_CHOICES="1"
  OPT_DOCKER=true
fi
if $HAS_GO && $HAS_NODE; then
  VALID_CHOICES="${VALID_CHOICES}${VALID_CHOICES:+ }2"
  OPT_DIRECT=true
fi

if [ -z "$VALID_CHOICES" ]; then
  fail "No installation method available."
  echo ""
  echo "  To install via Docker, install Docker Desktop first:"
  echo "    https://docs.docker.com/desktop/"
  echo ""
  echo "  To install directly, install prerequisites:"
  echo "    Go ${MIN_GO}+: https://go.dev/dl/"
  echo "    Node v${MIN_NODE}+: https://nodejs.org/"
  echo ""
  exit 1
fi

if [ "$VALID_CHOICES" = "1" ]; then
  INSTALL_MODE="docker"
  echo "  → Only Docker is available (Go or Node missing for direct install)"
  echo ""
  read -p "  Press Enter to continue with Docker install ... "
  echo ""
elif [ "$VALID_CHOICES" = "2" ]; then
  INSTALL_MODE="direct"
  echo "  → Only direct install is available (Docker not found)"
  echo ""
  read -p "  Press Enter to continue with direct install ... "
  echo ""
else
  read -p "  Choice [1/2]: " INSTALL_CHOICE
  echo ""
  case "$INSTALL_CHOICE" in
    1) INSTALL_MODE="docker" ;;
    2) INSTALL_MODE="direct" ;;
    *) fail "Invalid choice. Exiting."; exit 1 ;;
  esac
fi

# ─── API Keys ─────────────────────────────────────────────────
header "API Keys"

if [ -f "$PROJECT_DIR/.env" ]; then
  ok ".env file already exists at $PROJECT_DIR/.env"
  info "Edit it to add your API keys if you haven't already."
else
  echo "  Talon needs API keys for LLM providers."
  echo ""
  echo "  You can set them now or later by editing the .env file."
  echo ""
  read -p "  Create .env with placeholder keys? [Y/n]: " CREATE_ENV
  if [[ ! "$CREATE_ENV" =~ ^[Nn] ]]; then
    cat > "$PROJECT_DIR/.env" << 'ENV'
# ─── Talon — Environment ──────────────────────────────
# Uncomment and fill in at least one provider's API key.

# Anthropic (Claude)
# ANTHROPIC_API_KEY=sk-ant-...

# OpenAI
# OPENAI_API_KEY=sk-...

# Google (Gemini)
# GEMINI_API_KEY=...

# OpenRouter (multi-provider)
# OPENROUTER_API_KEY=...
ENV
    ok "Created $PROJECT_DIR/.env"
    echo ""
    warn "  Don't forget to edit .env and add your API keys!"
    echo "  Talon won't work without at least one key."
    echo ""
    read -p "  Edit .env now? [y/N]: " EDIT_ENV
    if [[ "$EDIT_ENV" =~ ^[Yy] ]]; then
      ${EDITOR:-vim} "$PROJECT_DIR/.env"
    fi
  else
    info "Skipping .env creation. Create one later if needed."
  fi
fi

# ─── Install ──────────────────────────────────────────────────

case "$INSTALL_MODE" in
  docker)
    header "Installing via Docker"
    cd "$PROJECT_DIR"

    # Detect port conflicts and auto-select alternatives
    AGENT_PORT=8082
    CACHE_PORT=8083
    if lsof -i :8082 &>/dev/null 2>&1; then
      AGENT_PORT=8092
      warn "Port 8082 in use — mapping agent to :$AGENT_PORT"
    fi
    if lsof -i :8083 &>/dev/null 2>&1; then
      CACHE_PORT=8093
      warn "Port 8083 in use — mapping cache to :$CACHE_PORT"
    fi
    export AGENT_PORT CACHE_PORT

    info "Building Docker images..."
    docker compose -f "$DOCKER_DIR/docker-compose.yml" build
    echo ""

    info "Starting services..."
    docker compose -f "$DOCKER_DIR/docker-compose.yml" up -d
    echo ""

    ok "Talon is running!"
    echo ""
    echo "  Dashboard:  ${BOLD}http://localhost:3000${NC}"
    echo "  Proxy API:  ${BOLD}http://localhost:8082${NC}"
    echo ""
    echo "  To see logs:    cd docker && bash run.sh logs"
    echo "  To stop:        cd docker && bash run.sh down"
    echo "  To rebuild:     bash scripts/install.sh"
    ;;

  direct)
    header "Installing directly on machine"
    cd "$PROJECT_DIR"

    # ── Build Go binary ──
    header "Building Go server"
    if $HAS_TASK; then
      info "Using go-task..."
      task build
    else
      info "Running: go build -v -o talon ."
      CGO_ENABLED=0 go build -v -o talon .
    fi
    ok "Go binary built: $PROJECT_DIR/talon-proxy"
    echo ""

    # ── Build Dashboard ──
    header "Building Dashboard (Next.js)"
    cd "$DASHBOARD_DIR"
    info "Installing dependencies..."
    npm ci --ignore-scripts
    info "Building..."
    npm run build
    ok "Dashboard built"
    cd "$PROJECT_DIR"
    echo ""

    # ── Start services ──
    header "Starting services"

    AGENT_PORT=8082
    DASHBOARD_PORT=3000

    if lsof -i :$AGENT_PORT &>/dev/null 2>&1; then
      warn "Port $AGENT_PORT is already in use. Skipping agent server start."
    else
      info "Starting proxy server on port $AGENT_PORT..."
      nohup "$PROJECT_DIR/talon-proxy" server --host "tcp://0.0.0.0:$AGENT_PORT" \
        > "$PROJECT_DIR/.agent.log" 2>&1 &
      AGENT_PID=$!
      ok "Proxy server started (PID: $AGENT_PID)"
    fi

    if lsof -i :$DASHBOARD_PORT &>/dev/null 2>&1; then
      warn "Port $DASHBOARD_PORT is already in use. Skipping dashboard start."
    else
      info "Starting dashboard on port $DASHBOARD_PORT..."
      nohup node "$DASHBOARD_DIR/.next/standalone/server.js" \
        > "$PROJECT_DIR/.dashboard.log" 2>&1 &
      DASHBOARD_PID=$!
      ok "Dashboard started (PID: $DASHBOARD_PID)"
    fi
    echo ""

    ok "Talon is running!"
    echo ""
    echo "  Dashboard:  ${BOLD}http://localhost:3000${NC}"
    echo "  Proxy API:  ${BOLD}http://localhost:8082${NC}"
    echo ""
    echo "  Agent PID:    ${AGENT_PID:-N/A}"
    echo "  Dashboard PID: ${DASHBOARD_PID:-N/A}"
    echo ""
    echo "  To stop:       kill $AGENT_PID $DASHBOARD_PID"
    echo "  To view logs:  tail -f .agent.log"
    echo "  To rebuild:    bash scripts/install.sh"
    ;;
esac

# ─── Identity ─────────────────────────────────────────────────
setup_identity

# ─── Done ─────────────────────────────────────────────────────
header "Install complete"

cat << 'DONE'

  ╔══════════════════════════════════════════════════╗
  ║         Talon is up and running!         ║
  ╚══════════════════════════════════════════════════╝

  • Dashboard:    http://localhost:3000
  • Proxy API:    http://localhost:8082
  • Health check: curl http://localhost:8082/health

  For help:       task --list
  Rebuild:        bash scripts/install.sh

DONE
