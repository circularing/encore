# Dashboard ISO — Technical Debt Register

## TD-001: Backend-injected DOM patch (`natsColumnPatchJS`)

- **Location:** `cli/daemon/dash/server.go`
- **Current behavior:** backend injects JS that mutates Service Catalog DOM and patches rows for NATS column data.
- **Why debt:** fragile coupling to frontend DOM shape, contradicts ISO rework constraints (no production DOM patch hacks).
- **Risk:** breakage on frontend updates, hidden regressions, hard-to-test behavior.

### Exit criteria

- NATS column implemented in frontend source with versioned contract from backend.
- Backend serves data only (typed payload), no DOM mutation script injection.
- Automated test coverage:
  - contract test on backend payload,
  - e2e check that NATS column renders in Service Catalog.
