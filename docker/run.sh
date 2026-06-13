#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$ROOT_DIR"

# ─── Cross-platform helpers ──────────────────────────────────────────
# open_url opens a URL in the default browser on any platform.
open_url() {
  case "$(uname -s)" in
    Darwin)  open "$1" ;;
    Linux)   xdg-open "$1" ;;
    MINGW*|MSYS*|CYGWIN*) start "$1" ;;
    *)       echo "Open $1 in your browser" ;;
  esac
}

# ─── Help ────────────────────────────────────────────────────────────
usage() {
  cat <<EOF
Usage: ./docker/run.sh [command]

Commands:
  up        Build and start all services    (docker compose up --build -d)
  down      Stop all services               (docker compose down)
  logs      Follow logs                     (docker compose logs -f)
  restart   Restart all services
  status    Show container status
  build     Rebuild images without starting
  shell     Open a shell in the agent container
  dashboard Open the dashboard in your browser
EOF
  exit 0
}

# ─── Config ──────────────────────────────────────────────────────────
# Source API keys from ~/.spectre-proxy/.env if it exists
ENV_FILE="$HOME/.spectre-proxy/.env"
if [ -f "$ENV_FILE" ]; then
  set -a
  source "$ENV_FILE"
  set +a
fi

# Default model
export MODEL="${MODEL:-openrouter/anthropic/claude-sonnet-4}"

# ─── Commands ────────────────────────────────────────────────────────
case "${1:-help}" in
  up)
    echo "🚀 Building and starting Spectre Proxy..."
    docker compose -f "$SCRIPT_DIR/docker-compose.yml" up --build -d
    echo ""
    echo "  Dashboard:  http://localhost:3000"
    echo "  Proxy:      http://localhost:8082"
    echo "  Proxy API:  curl http://localhost:8082/v1/models"
    echo ""
    echo "  Set API keys: export OPENROUTER_API_KEY=sk-..."
    echo "  Or add them to: $ENV_FILE"
    ;;
  down)
    echo "🛑 Stopping Spectre Proxy..."
    docker compose -f "$SCRIPT_DIR/docker-compose.yml" down
    ;;
  logs)
    docker compose -f "$SCRIPT_DIR/docker-compose.yml" logs -f
    ;;
  restart)
    echo "🔄 Restarting..."
    docker compose -f "$SCRIPT_DIR/docker-compose.yml" down
    docker compose -f "$SCRIPT_DIR/docker-compose.yml" up --build -d
    ;;
  status)
    docker compose -f "$SCRIPT_DIR/docker-compose.yml" ps
    ;;
  build)
    echo "🔨 Building images..."
    docker compose -f "$SCRIPT_DIR/docker-compose.yml" build
    ;;
  shell)
    echo "🐚 Opening shell in proxy container..."
    docker exec -it spectre-proxy sh
    ;;
  dashboard)
    echo "📂 Opening dashboard..."
    open_url "http://localhost:3000"
    ;;
  *)
    usage
    ;;
esac
