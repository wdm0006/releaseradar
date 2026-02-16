#!/usr/bin/env zsh
set -euo pipefail

# Generate all screenshots and GIFs for the README.
# This script manages its own test config/cache so you don't need to touch yours.
#
# Usage:
#   ./screenshots/generate.sh              # Generate all screenshots
#   ./screenshots/generate.sh chat summary # Run only specific tapes
#
# Requirements: vhs, go

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BINARY="$ROOT_DIR/releaseradar"

CONFIG="$HOME/.releaseradar.json"
CACHE_DIR="$HOME/Library/Caches/releaseradar"
CACHE="$CACHE_DIR/cache.json"

DEMO_REPOS=(
  "jlowin/fastmcp"
  "pydantic/pydantic-ai"
  "langchain-ai/langchain"
  "run-llama/llama_index"
  "openai/openai-python"
)

# Default tape order: repos first (VHS Screenshot is unreliable as second tape),
# loading last because it clears the cache.
DEFAULT_TAPES=(repos releases summary chat loading)

# --- helpers ---
save_user_state() {
  echo "Saving your config and cache..."
  cp "$CONFIG" "$CONFIG.demo-bak" 2>/dev/null || true
  cp "$CACHE" "$CACHE.demo-bak" 2>/dev/null || true
}

restore_user_state() {
  echo "Restoring your config and cache..."
  mv "$CONFIG.demo-bak" "$CONFIG" 2>/dev/null || true
  mv "$CACHE.demo-bak" "$CACHE" 2>/dev/null || true
  echo "Done. Your original config is restored."
}

trap restore_user_state EXIT

# --- main ---
echo "=== ReleaseRadar Screenshot Generator ==="
echo ""

# Determine which tapes to run
if [ $# -gt 0 ]; then
  TAPES=("$@")
else
  TAPES=("${DEFAULT_TAPES[@]}")
fi

# Build
echo "Building binary..."
(cd "$ROOT_DIR" && go build -o releaseradar ./cmd/releaseradar)

save_user_state

# Set up demo config using the CLI
echo ""
echo "Setting up demo repos..."
echo '{"repos":[]}' > "$CONFIG"
for repo in "${DEMO_REPOS[@]}"; do
  "$BINARY" add "$repo" >/dev/null
done
echo "Tracking: $("$BINARY" list | tr '\n' ' ')"

# Warm cache
echo "Warming cache (this fetches releases from GitHub)..."
"$BINARY" cache warm
echo ""

# Remove outputs for tapes we're about to record
for name in "${TAPES[@]}"; do
  rm -rf "$SCRIPT_DIR/$name.png" "$SCRIPT_DIR/$name.gif"
done

# Record each tape (PNGs are captured via VHS Screenshot command in each tape)
for name in "${TAPES[@]}"; do
  tape="$SCRIPT_DIR/$name.tape"
  if [ ! -f "$tape" ]; then
    echo "SKIP: $tape not found"
    continue
  fi

  # Loading tape needs a cleared cache
  if [ "$name" = "loading" ]; then
    "$BINARY" cache clear
  fi

  # Kill any lingering TUI processes between tapes
  pkill -f "$BINARY" 2>/dev/null || true
  sleep 2

  echo "Recording $name..."
  vhs "$tape"

  # Re-warm after loading tape
  if [ "$name" = "loading" ]; then
    echo "  Re-warming cache..."
    "$BINARY" cache warm 2>&1 | tail -1
  fi

  echo ""
done

echo "=== Screenshots generated ==="
ls -lh "$SCRIPT_DIR"/*.png "$SCRIPT_DIR"/*.gif 2>/dev/null
