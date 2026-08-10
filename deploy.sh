#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="$ROOT_DIR/deploy/compose.yaml"
COMPOSE=(docker compose -f "$COMPOSE_FILE")

usage() {
  echo "Usage: ./deploy.sh {up|down|restart|status|logs|upgrade|uninstall}"
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

command -v docker >/dev/null 2>&1 || { echo "Docker is required." >&2; exit 1; }
docker compose version >/dev/null 2>&1 || { echo "Docker Compose v2 is required." >&2; exit 1; }

case "${1:-}" in
  up)        "${COMPOSE[@]}" up -d --build --remove-orphans; sync_token ;;
  down)      "${COMPOSE[@]}" down ;;
  restart)   "${COMPOSE[@]}" restart ;;
  status)    "${COMPOSE[@]}" ps ;;
  logs)      "${COMPOSE[@]}" logs -f --tail=100 ;;
  upgrade)   "${COMPOSE[@]}" pull --ignore-buildable; "${COMPOSE[@]}" up -d --build --remove-orphans; sync_token ;;
  uninstall) "${COMPOSE[@]}" down --volumes --rmi local ;;
  *)         usage; exit 1 ;;
esac
