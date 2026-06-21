#!/usr/bin/env bash
set -euo pipefail

# ═══════════════════════════════════════════════════════
# Talon — Install / Update Script
#
# First run:  builds everything from scratch
# Subsequent: detects what changed, rebuilds only what's needed
#
# Usage:
#   bash scripts/install.sh              # Install or update
#   bash scripts/install.sh --force      # Full rebuild everything
#   bash scripts/install.sh --quick      # Just copy binaries, no build
# ═══════════════════════════════════════════════════════

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TALON_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TALON_HOME="$HOME/.talon"
ZIG_PATH="/opt/homebrew/opt/zig@0.16/bin/zig"

FORCE=false
QUICK=false
[[ "$*" == *"--force"* ]] && FORCE=true
[[ "$*" == *"--quick"* ]] && QUICK=true

echo "═══════════════════════════════════════════"
echo "     Talon — Installation"
echo "═══════════════════════════════════════════"
echo ""

# ── Prerequisites ───────────────────────────────────

echo "🔍 Checking prerequisites..."

if command -v bun &>/dev/null; then
  echo "  ✅ Bun: $(bun --version)"
else
  echo "  ❌ Bun not found. Install: curl -fsSL https://bun.sh/install | bash"
  exit 1
fi

if command -v go &>/dev/null; then
  echo "  ✅ Go: $(go version | sed 's/go version //')"
else
  echo "  ❌ Go not found. Install: https://go.dev/dl/"
  exit 1
fi

ZIG_CMD=""
if ! $QUICK; then
  if command -v zig &>/dev/null; then
    ZIG_CMD="zig"
    echo "  ✅ Zig: $(zig version)"
  elif [ -f "$ZIG_PATH" ]; then
    ZIG_CMD="$ZIG_PATH"
    echo "  ✅ Zig: $($ZIG_CMD version)"
  else
    echo "  ⚠️  Zig not found (optional, needed for native lib)"
  fi
fi
echo ""

# ── Create directories ──────────────────────────────

mkdir -p "$TALON_HOME/bin" "$TALON_HOME/log" "$TALON_HOME/data"
echo "📁 Directories ready"
echo ""

# ── Install JS dependencies ─────────────────────────

if $FORCE || $QUICK; then
  echo "📦 JavaScript dependencies already installed"
else
  echo "📦 Installing JavaScript dependencies..."
  cd "$TALON_ROOT/tui"
  bun install 2>&1 | tail -1
  cd "$TALON_ROOT/ai"
  bun install 2>&1 | tail -1
  cd "$TALON_ROOT"
  echo ""
fi

# ── Build native library ─────────────────────────────

if $QUICK; then
  echo "⏭️  Skipping native build (--quick)"
elif [ -n "$ZIG_CMD" ]; then
  echo "🏗️  Building native library (libopentui.dylib)..."
  (cd "$TALON_ROOT/tui/packages/core/src/zig" && $ZIG_CMD build install)
  LIB_SRC="$TALON_ROOT/tui/packages/core/src/zig/lib/aarch64-macos/libopentui.dylib"
  if [ -f "$LIB_SRC" ]; then
    cp "$LIB_SRC" "$TALON_HOME/bin/libopentui.dylib"
    echo "  ✅ libopentui.dylib ($(du -h "$LIB_SRC" | cut -f1))"
  fi
else
  echo "⚠️  Skipping native build (Zig not available)"
fi

# Create workspace package so ai/ can resolve @tui/core-darwin-arm64
CORE_DARWIN_SRC="$TALON_ROOT/tui/packages/core/node_modules/@tui/core-darwin-arm64"
CORE_DARWIN_PKG="$TALON_ROOT/tui/packages/core-darwin-arm64"
if [ -d "$CORE_DARWIN_SRC" ] && [ ! -d "$CORE_DARWIN_PKG" ]; then
  cp -R "$CORE_DARWIN_SRC" "$CORE_DARWIN_PKG"
  echo "  ✅ Workspace package: packages/core-darwin-arm64 → @tui/core-darwin-arm64"
fi
echo ""

# ── Build TUI packages (needed by AI CLI) ───────────

if $QUICK; then
  echo "⏭️  Skipping TUI package builds (--quick)"
else
  echo "🏗️  Building @tui/core..."
  (cd "$TALON_ROOT/tui/packages/core" && bun run build 2>&1 | tail -1)
  echo "  ✅ @tui/core built"
  
  # Symlink parser.worker.js into core root for workspace consumers
  if [ ! -L "$TALON_ROOT/tui/packages/core/parser.worker.js" ]; then
    ln -sfn dist/parser.worker.js "$TALON_ROOT/tui/packages/core/parser.worker.js"
    echo "  ✅ parser.worker.js symlink"
  fi
  
  echo "🏗️  Building @tui/keymap..."
  (cd "$TALON_ROOT/tui/packages/keymap" && bun run build 2>&1 | tail -1)
  echo "  ✅ @tui/keymap built"
fi
echo ""

# ── Build Go backend ────────────────────────────────

if $QUICK && [ -f "$TALON_HOME/bin/talon-server" ]; then
  echo "⏭️  Skipping Go backend (--quick)"
elif $FORCE || $QUICK || [ ! -f "$TALON_HOME/bin/talon-server" ]; then
  echo "🏗️  Building Go backend..."
  (cd "$TALON_ROOT/backend" && go build -o "$TALON_HOME/bin/talon-server" ./cmd/server/)
  echo "  ✅ talon-server ($(du -h "$TALON_HOME/bin/talon-server" | cut -f1))"
else
  echo "⏭️  Go backend already built (use --force to rebuild)"
fi
echo ""

# ── Build AI CLI ────────────────────────────────────

if $QUICK && [ -f "$TALON_HOME/bin/talon-ai" ]; then
  echo "⏭️  Skipping AI CLI build (--quick)"
else
  echo "🏗️  Building talon AI CLI..."
  export PATH="$HOME/.bun/bin:$PATH"
  # Pre-install native addon so bun compile embeds the dylib.
  # NOTE: bun install creates a symlink for workspace:* packages, but bun compile
  # needs a real directory to embed the .dylib file, so we replace the symlink.
  (cd "$TALON_ROOT/ai/packages/opencode" && bun install "@tui/core-darwin-arm64@workspace:*" 2>&1 | tail -1)
  CORE_DARWIN_PKG="$TALON_ROOT/ai/packages/opencode/node_modules/@tui/core-darwin-arm64"
  if [ -L "$CORE_DARWIN_PKG" ]; then
    rm "$CORE_DARWIN_PKG"
    cp -R "$TALON_ROOT/tui/packages/core-darwin-arm64" "$CORE_DARWIN_PKG"
  fi
  (cd "$TALON_ROOT/ai/packages/opencode" && bun run build --single 2>&1 | tail -5)
  
  # Find the built binary (dist/opencode-{os}-{arch}/bin/opencode)
  BUILT_BINARY=$(find "$TALON_ROOT/ai/packages/opencode/dist" -name "opencode" -type f 2>/dev/null | head -1)
  if [ -z "$BUILT_BINARY" ]; then
    BUILT_BINARY=$(find "$TALON_ROOT/ai/packages/opencode/dist" -name "talon" -type f 2>/dev/null | head -1)
  fi
  
  if [ -f "$BUILT_BINARY" ]; then
    cp "$BUILT_BINARY" "$TALON_HOME/bin/talon-ai"
    echo "  ✅ talon-ai ($(du -h "$TALON_HOME/bin/talon-ai" | cut -f1))"
  else
    echo "  ⚠️  AI CLI build output not found"
  fi
fi
echo ""

# ── Install talon command ───────────────────────────

echo "🔗 Installing 'talon' command..."

cat > "$TALON_HOME/bin/talon" << 'TALON_SCRIPT'
#!/usr/bin/env bash
TALON_HOME="$HOME/.talon"
TALON_ROOT="$HOME/development/projects/talon"

if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
  echo "Talon — AI-powered development tool"
  echo ""
  echo "Usage:"
  echo "  talon                   Open the AI assistant"
  echo "  talon run <message..>   Run with a prompt"
  echo "  talon --help            Show this help"
  echo ""
  echo "Commands:"
  echo "  talon models       List available models"
  echo "  talon providers    Manage providers"
  echo "  talon session      Manage sessions"
  echo "  talon mcp          Manage MCP servers"
  echo ""
  echo "Services:"
  echo "  launchctl kickstart gui/$(id -u)/com.talon.backend   # Start Go backend"
  echo "  curl http://localhost:8090/health                    # Check server"
  exit 0
fi

# Try the compiled binary first (fastest)
if [ -f "$TALON_HOME/bin/talon-ai" ]; then
  # Run with args — CLI commands work great in compiled mode
  if [ $# -gt 0 ]; then
    exec "$TALON_HOME/bin/talon-ai" "$@"
  fi
  
  # No args — try compiled TUI, fall back to source if it fails
  set +e
  output=$("$TALON_HOME/bin/talon-ai" 2>&1)
  exit_code=$?
  set -e
  
  if [ $exit_code -eq 0 ]; then
    echo "$output"
    exit 0
  fi
  
  # Compiled TUI failed — try running from source
  echo "⚠️  Compiled TUI unavailable, starting from source..." >&2
fi

# Run from source (always works)
cd "$TALON_ROOT/ai/packages/talon"
exec bun run src/index.ts "$@"
TALON_SCRIPT

chmod +x "$TALON_HOME/bin/talon"

# Symlink to PATH
mkdir -p "$HOME/.local/bin"
ln -sf "$TALON_HOME/bin/talon" "$HOME/.local/bin/talon"
if ! echo "$PATH" | grep -q "$HOME/.local/bin"; then
  echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.zshrc"
fi

echo "  ✅ Command installed: talon"
echo ""

# ── Create default config ───────────────────────────

if [ ! -f "$TALON_HOME/config.json" ]; then
  cat > "$TALON_HOME/config.json" << 'CONFIG'
{
  "server": { "port": 8090 },
  "mcp": { "servers": {} }
}
CONFIG
  echo "  ✅ Default config created"
fi
echo ""

# ── Set up launchd service ──────────────────────────

echo "⚙️  Setting up backend service..."
PLIST_SRC="$TALON_ROOT/scripts/com.talon.backend.plist"
PLIST_DST="$HOME/Library/LaunchAgents/com.talon.backend.plist"

sed -e "s|__TALON_BIN_DIR__|$TALON_HOME/bin|g" \
    -e "s|__TALON_HOME__|$TALON_HOME|g" \
    "$PLIST_SRC" > "$PLIST_DST"

launchctl bootout "gui/$(id -u)/com.talon.backend" 2>/dev/null || true
sleep 1
launchctl bootstrap "gui/$(id -u)" "$PLIST_DST"
echo "  ✅ Service running on http://localhost:8090"
echo ""

# ── Verify ───────────────────────────────────────────

echo "⏳ Verifying..."
sleep 2
if curl -s http://localhost:8090/api/health >/dev/null 2>&1; then
  echo "  ✅ Backend health: OK"
  curl -s http://localhost:8090/api/health | python3 -m json.tool 2>/dev/null || true
else
  echo "  ⚠️  Backend not responding. Check: cat $TALON_HOME/log/talon-server.log"
fi
echo ""

# ── Done ─────────────────────────────────────────────

echo "═══════════════════════════════════════════"
echo "     ✅ Talon is ready!"
echo "═══════════════════════════════════════════"
echo ""
echo "  talon               # Open the AI assistant"
echo "  talon run \"msg\"     # Run with a prompt"
echo "  talon --help        # Show commands"
echo ""
echo "  Set API keys in ~/.talon/.env:"
echo "    ANTHROPIC_API_KEY=sk-ant-..."
echo "    OPENAI_API_KEY=sk-..."
echo ""
