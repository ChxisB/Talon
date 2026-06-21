#!/usr/bin/env bash
set -euo pipefail

# ═══════════════════════════════════════════════
# Talon — Build Script
# Builds all components into a distributable CLI
# ═══════════════════════════════════════════════

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TALON_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DIST_DIR="$TALON_ROOT/dist"
ZIG_PATH="/opt/homebrew/opt/zig@0.16/bin/zig"

echo "═══════════════════════════════════════════════"
echo "  Talon Build — v$(node -e "console.log(require('$TALON_ROOT/tui/package.json').version)")"
echo "═══════════════════════════════════════════════"
echo ""

# ── 1. Build the Zig native library ────────────────

echo "🔧 [1/4] Building native library (libopentui.dylib)..."
if command -v zig &>/dev/null; then
  ZIG=zig
elif [ -f "$ZIG_PATH" ]; then
  ZIG="$ZIG_PATH"
else
  echo "  ❌ Zig not found. Install: brew install zig"
  exit 1
fi

(cd "$TALON_ROOT/tui/packages/core/src/zig" && $ZIG build install)
LIB_SRC="$TALON_ROOT/tui/packages/core/src/zig/lib/aarch64-macos/libopentui.dylib"
if [ -f "$LIB_SRC" ]; then
  echo "  ✅ libopentui.dylib ($(du -h "$LIB_SRC" | cut -f1))"
else
  echo "  ❌ Native library build failed"
  exit 1
fi
echo ""

# ── 2. Build the Go backend ────────────────────────

echo "🔧 [2/4] Building Go backend..."
mkdir -p "$DIST_DIR"
(cd "$TALON_ROOT/backend" && go build -o "$DIST_DIR/talon-server" ./cmd/server/)
echo "  ✅ talon-server ($(du -h "$DIST_DIR/talon-server" | cut -f1))"
echo ""

# ── 3. Build the TUI (compiled Bun binary) ─────────

echo "🔧 [3/4] Compiling TUI (bun build --compile)..."
mkdir -p "$DIST_DIR"

# Copy native lib so the binary can find it relative to itself
cp "$LIB_SRC" "$DIST_DIR/libopentui.dylib"

(cd "$TALON_ROOT/tui" && bun build --compile \
  --outfile="$DIST_DIR/talon" \
  ./src/cli.tsx 2>&1)

if [ ! -f "$DIST_DIR/talon" ]; then
  # Try without target flag
  (cd "$TALON_ROOT/tui" && bun build --compile \
    --outfile="$DIST_DIR/talon" \
    ./src/cli.tsx 2>&1)
fi

if [ -f "$DIST_DIR/talon" ]; then
  echo "  ✅ talon binary ($(du -h "$DIST_DIR/talon" | cut -f1))"
else
  echo "  ❌ TUI compilation failed"
  exit 1
fi
echo ""

# ── 4. Create distribution package ─────────────────

echo "📦 [4/4] Creating distribution..."
DIST_VERSION="$(node -e "console.log(require('$TALON_ROOT/tui/package.json').version)")"
DIST_NAME="talon-v$DIST_VERSION"

mkdir -p "$DIST_DIR/$DIST_NAME"

# Copy compiled binary into the dist package
cp "$DIST_DIR/talon" "$DIST_DIR/$DIST_NAME/talon-bin"
cp "$DIST_DIR/talon-server" "$DIST_DIR/$DIST_NAME/talon-server"
cp "$DIST_DIR/libopentui.dylib" "$DIST_DIR/$DIST_NAME/libopentui.dylib"

# Create a convenience launcher
cat > "$DIST_DIR/$DIST_NAME/talon" << 'LAUNCHER'
#!/usr/bin/env bash
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export TALON_NATIVE_LIB="$DIR/libopentui.dylib"
"$DIR/talon-server" &
BACKEND_PID=$!
for i in {1..10}; do
  if curl -s http://localhost:8090/health >/dev/null 2>&1; then break; fi
  sleep 0.3
done
"$DIR/talon-bin" "$@"
kill $BACKEND_PID 2>/dev/null
LAUNCHER
chmod +x "$DIST_DIR/$DIST_NAME/talon"

echo "  ✅ Distribution: $DIST_DIR/$DIST_NAME/"
echo "     ├── talon         (launcher script)"
echo "     ├── talon-server   (Go backend)"
    echo "     ├── libopentui.dylib (Zig native core)"
    echo "     └── talon-bin      (compiled TUI)"
echo ""

# ── Install to ~/.local/bin ────────────────────────

echo "🔗 Installing to ~/.local/bin/..."
mkdir -p "$HOME/.local/bin"
cp "$DIST_DIR/talon" "$HOME/.local/bin/talon-bin"
cp "$DIST_DIR/talon-server" "$HOME/.local/bin/talon-server"
cp "$DIST_DIR/libopentui.dylib" "$HOME/.local/bin/libopentui.dylib"

cat > "$HOME/.local/bin/talon" << 'LAUNCHER'
#!/usr/bin/env bash
DIR="$HOME/.local/bin"
export TALON_NATIVE_LIB="$DIR/libopentui.dylib"
"$DIR/talon-server" &
BACKEND_PID=$!
for i in {1..10}; do
  if curl -s http://localhost:8090/health >/dev/null 2>&1; then break; fi
  sleep 0.3
done
"$DIR/talon-bin" "$@"
kill $BACKEND_PID 2>/dev/null
LAUNCHER
chmod +x "$HOME/.local/bin/talon"
echo "  ✅ Installed to ~/.local/bin/"
echo "     Run 'talon' from anywhere!"
echo "  ✅ Installed to ~/.local/bin/"
echo "     Run 'talon' from anywhere!"
echo ""

# ── Done ───────────────────────────────────────────

echo "═══════════════════════════════════════════════"
echo "  ✅ Talon build complete!"
echo "═══════════════════════════════════════════════"
echo ""
echo "  Binary:  $DIST_DIR/talon"
echo "  Backend: $DIST_DIR/talon-server"
echo "  Lib:     $DIST_DIR/libopentui.dylib"
echo ""
echo "  Run:     talon"
echo "  Or:      $DIST_DIR/talon"
echo ""
