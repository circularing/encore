#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT="${1:-$ROOT/docs/dashboard-iso/contract-inventory.md}"

mkdir -p "$(dirname "$OUT")"

{
  echo "# Dashboard Local Contract Inventory (Sprint 0)"
  echo
  echo "Generated: $(date -u +"%Y-%m-%d %H:%M:%SZ")"
  echo
  echo '## HTTP routes in `cli/daemon/dash/server.go`'
  echo
  echo "| Route | Handler |"
  echo "|---|---|"
  awk '
    /switch req.URL.Path/ { in_switch=1; next }
    in_switch && /case "/ {
      route=$2; gsub(/"|:/, "", route)
      pending=route
      next
    }
    in_switch && pending != "" && /natsColumnPatchJS/ {
      printf("| `%s` | `inline-script:natsColumnPatchJS` |\n", pending)
      pending=""
      next
    }
    in_switch && pending != "" && /s\.[A-Za-z0-9_]+\(/ {
      handler=$0
      sub(/^.*s\./, "", handler)
      sub(/\(.*/, "", handler)
      printf("| `%s` | `%s` |\n", pending, handler)
      pending=""
      next
    }
    in_switch && pending != "" && /ServeHTTP\(w, req\)/ {
      handler=$0
      if (handler ~ /apiProxy/) {
        printf("| `%s` | `apiProxy` |\n", pending)
      } else if (handler ~ /proxy/) {
        printf("| `%s` | `proxy` |\n", pending)
      }
      pending=""
      next
    }
  ' "$ROOT/cli/daemon/dash/server.go"
  echo
  echo "## WebSocket"
  echo
  echo '- `/__encore` upgrades to websocket and serves JSON-RPC stream via `WebSocket` handler.'
  echo
  echo "## Known technical debt (from ISO rework plan)"
  echo
  echo '- `natsColumnPatchJS` currently injects DOM patch logic from backend; plan requires replacing this with native frontend implementation and typed contract.'
} > "$OUT"

echo "Wrote $OUT"