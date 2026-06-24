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
  (cd "$TALON_ROOT/tui" && bun install 2>&1 | tail -1)
  (cd "$TALON_ROOT/ai" && bun install 2>&1 | tail -1)
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
    # Remove any stale dylib from the rebranding
    rm -f "$TALON_HOME/bin/libtalon.dylib"
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
  ln -sfn dist/parser.worker.js "$TALON_ROOT/tui/packages/core/parser.worker.js"
  echo "  ✅ parser.worker.js symlink"
  
  echo "🏗️  Building @tui/keymap..."
  (cd "$TALON_ROOT/tui/packages/keymap" && bun run build 2>&1 | tail -1)
  echo "  ✅ @tui/keymap built"
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
  (cd "$TALON_ROOT/ai/packages/talon" && bun install "@tui/core-darwin-arm64@workspace:*" 2>&1 | tail -1)
  CORE_DARWIN_PKG="$TALON_ROOT/ai/packages/talon/node_modules/@tui/core-darwin-arm64"
  if [ -L "$CORE_DARWIN_PKG" ]; then
    rm "$CORE_DARWIN_PKG"
    cp -R "$TALON_ROOT/tui/packages/core-darwin-arm64" "$CORE_DARWIN_PKG"
  fi
  (cd "$TALON_ROOT/ai/packages/talon" && bun run build --single --skip-embed-web-ui 2>&1 | tail -5)
  
  # Find the built binary (dist/talon-{os}-{arch}/bin/talon)
  BUILT_BINARY=$(find "$TALON_ROOT/ai/packages/talon/dist" -name "talon" -type f 2>/dev/null | head -1)
  
  if [ -f "$BUILT_BINARY" ]; then
    cp "$BUILT_BINARY" "$TALON_HOME/bin/talon-ai"
    # Re-sign to avoid macOS "Code Signature Invalid" SIGKILL on ad-hoc signed binaries
    codesign -f -s - "$TALON_HOME/bin/talon-ai" 2>/dev/null || true
    echo "  ✅ talon-ai ($(du -h "$TALON_HOME/bin/talon-ai" | cut -f1))"
  else
    echo "  ⚠️  AI CLI build output not found"
  fi
fi
echo ""

# ── Install talon command ───────────────────────────

echo "🔗 Installing 'talon' command..."

# Remove existing symlink first, if any, so the cat > below creates a regular file
# (otherwise it would follow the symlink and overwrite the git-tracked scripts/talon)
rm -f "$TALON_HOME/bin/talon"

cat > "$TALON_HOME/bin/talon" << TALON_SCRIPT
#!/usr/bin/env bash
TALON_HOME="\$HOME/.talon"
TALON_ROOT="$TALON_ROOT"

# Try the compiled binary first (fastest)
if [ -f "\$TALON_HOME/bin/talon-ai" ]; then
  exec "\$TALON_HOME/bin/talon-ai" "\$@"
fi

# Run from source (works even without a compiled binary)
cd "\$TALON_ROOT/ai/packages/talon"
exec bun run src/index.ts "\$@"
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
  "mcp": { "servers": {} }
}
CONFIG
  echo "  ✅ Default config created"
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
echo "    OPENAI_API_KEY=sk-..."
echo ""
