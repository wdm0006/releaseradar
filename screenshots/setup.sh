#!/bin/bash
# Temporary config for screenshot generation
# Backs up existing config, writes test config, runs VHS, restores original

set -e

CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/Library/Application Support}/releaseradar"
CONFIG_FILE="$CONFIG_DIR/config.json"
CACHE_DIR="${XDG_CACHE_HOME:-$HOME/Library/Caches}/releaseradar"
CACHE_FILE="$CACHE_DIR/cache.json"
BACKUP_CONFIG="$CONFIG_FILE.bak"
BACKUP_CACHE="$CACHE_FILE.bak"

cleanup() {
    # Restore original config
    if [ -f "$BACKUP_CONFIG" ]; then
        mv "$BACKUP_CONFIG" "$CONFIG_FILE"
    fi
    if [ -f "$BACKUP_CACHE" ]; then
        mv "$BACKUP_CACHE" "$CACHE_FILE"
    elif [ -f "$CACHE_FILE" ]; then
        rm "$CACHE_FILE"
    fi
}
trap cleanup EXIT

# Backup existing files
[ -f "$CONFIG_FILE" ] && cp "$CONFIG_FILE" "$BACKUP_CONFIG"
[ -f "$CACHE_FILE" ] && cp "$CACHE_FILE" "$BACKUP_CACHE"

# Write test config
mkdir -p "$CONFIG_DIR"
cat > "$CONFIG_FILE" << 'EOF'
{
  "repos": [
    "jlowin/fastmcp",
    "pydantic/pydantic-ai",
    "langchain-ai/langchain",
    "run-llama/llama_index",
    "openai/openai-python"
  ]
}
EOF

# Clear cache so we get fresh data for screenshots
rm -f "$CACHE_FILE"

# Run VHS tapes
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR/.."

echo "Recording releases tab..."
vhs "$SCRIPT_DIR/releases.tape"

echo "Recording repos tab..."
vhs "$SCRIPT_DIR/repos.tape"

echo "Recording loading screen..."
vhs "$SCRIPT_DIR/loading.tape"

echo "Done! Screenshots saved to screenshots/"
