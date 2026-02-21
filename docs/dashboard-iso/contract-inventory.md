# Dashboard Local Contract Inventory (Sprint 0)

Generated: 2026-02-20 21:49:21Z

## HTTP routes in `cli/daemon/dash/server.go`

| Route | Handler |
|---|---|
| `/__encore/nats-column.js` | `inline-script:natsColumnPatchJS` |
| `/__encore/status` | `StatusJSON` |
| `/__encore` | `WebSocket` |
| `/__graphql` | `apiProxy` |

## WebSocket

- `/__encore` upgrades to websocket and serves JSON-RPC stream via `WebSocket` handler.

## Known technical debt (from ISO rework plan)

- `natsColumnPatchJS` currently injects DOM patch logic from backend; plan requires replacing this with native frontend implementation and typed contract.
