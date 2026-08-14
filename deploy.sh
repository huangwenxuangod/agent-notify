#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="$ROOT_DIR/deploy/compose.yaml"
COMPOSE=(docker compose -f "$COMPOSE_FILE")

usage() {
  echo "Usage: ./deploy.sh {up|down|restart|status|logs|upgrade|desktop|uninstall}"
}

sync_token() {
  local state_dir="${AGENT_NOTIFY_STATE_DIR:-$HOME/.agent-notify}"
  local token_path="$state_dir/bridge.token"
  local temp_path="$state_dir/.bridge.token.tmp.$$"
  mkdir -p "$state_dir"
  "${COMPOSE[@]}" exec -T control-plane cat /var/lib/agent-notify/bridge.token >"$temp_path"
  chmod 600 "$temp_path"
  mv "$temp_path" "$token_path"
}

desktop() {
  if [[ "$(uname -s)" != "Darwin" ]]; then
    echo "The packaged desktop launcher currently supports macOS only." >&2
    exit 1
  fi

  local app_path="$HOME/Applications/Agent Notify.app"
  local contents="$app_path/Contents"
  local executable="$contents/MacOS/Agent Notify"
  local hook_binary="${AGENT_NOTIFY_HOOK_BINARY:-$HOME/.agent-notify/agent-notify}"

  (cd "$ROOT_DIR/desktop" && bun install --frozen-lockfile && bun run typecheck && bun run build)
  mkdir -p "$contents/MacOS"
  mkdir -p "$contents/Resources"
  # Keep the launcher on the installed toolchain. With cached dependencies this
  # makes the desktop command deterministic and avoids an implicit toolchain
  # download when the machine is offline.
  if ! GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOTELEMETRY=off \
    go build -tags production -o "$executable" ./cmd/agent-notify-desktop; then
    echo "Desktop build needs Go modules in the local cache. Connect once and run: go mod download" >&2
    exit 1
  fi
  mkdir -p "$(dirname "$hook_binary")"
  if ! GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOTELEMETRY=off \
    go build -tags production -o "$hook_binary" ./cmd/agent-notify; then
    echo "Hook runtime build failed; desktop app was not opened." >&2
    exit 1
  fi
  chmod 755 "$hook_binary"
  cp "$ROOT_DIR/cmd/agent-notify-desktop/build/darwin/Info.plist" "$contents/Info.plist"
  cp "$ROOT_DIR/cmd/agent-notify-desktop/build/darwin/AgentNotify.icns" "$contents/Resources/AgentNotify.icns"
  codesign --force --deep --sign - "$app_path"
  osascript -e 'tell application "Agent Notify" to quit' 2>/dev/null || true
  # Hide-on-close is the normal runtime mode, so AppleScript quit may only
  # hide the window. Stop the exact old executable before replacing its memory.
  pkill -TERM -f -- "$executable" 2>/dev/null || true
  for _ in 1 2 3 4 5; do
    pgrep -f "$executable" >/dev/null || break
    sleep 1
  done
  if pgrep -f "$executable" >/dev/null; then
    pkill -KILL -f -- "$executable" 2>/dev/null || true
    sleep 1
  fi
  open -gj "$app_path" --args --show
}

case "${1:-}" in
  desktop)   desktop ;;
  up|down|restart|status|logs|upgrade|uninstall)
    command -v docker >/dev/null 2>&1 || { echo "Docker is required." >&2; exit 1; }
    docker compose version >/dev/null 2>&1 || { echo "Docker Compose v2 is required." >&2; exit 1; }
    case "$1" in
      up)        "${COMPOSE[@]}" up -d --build --remove-orphans; sync_token ;;
      down)      "${COMPOSE[@]}" down ;;
      restart)   "${COMPOSE[@]}" restart; sync_token ;;
      status)    "${COMPOSE[@]}" ps ;;
      logs)      "${COMPOSE[@]}" logs -f --tail=100 ;;
      upgrade)   "${COMPOSE[@]}" pull --ignore-buildable; "${COMPOSE[@]}" up -d --build --remove-orphans; sync_token ;;
      uninstall) "${COMPOSE[@]}" down --volumes --rmi local ;;
    esac
    ;;
  *)         usage; exit 1 ;;
esac
