# Title
feat(nats): add pluggable parser hooks and support `//encore:nats` custom directive

## Summary
This PR introduces a custom NATS directive path built on top of the parser plugin mechanism.

It adds:
- pluggable directive parser hooks in `v2/parser/apis/directive`
- `//encore:nats <subject> [field=value ...]` support
- parser/resource wiring for NATS subscriptions
- stricter signature + subject validation
- NATS runtime scaffolding improvements and tests
- docs + implementation examples

`//encore:pubsub` alias has been removed; `//encore:nats` is now canonical.

## Main changes

### Parser plugin hooks
- `directive.RegisterDirectiveParser(name, parser)`
- plugin callback execution from `directive.Parse(...)` for function directives
- regression fix preserving stripped doc text for standard `//encore:...` syntax

### NATS directive parsing
- New directive name: `nats`
- Supported form:
  - `//encore:nats orders.created`
  - `//encore:nats orders.created mode=at-least-once ackwait=30s maxinflight=64 queue=workers stream=orders_events subjects=orders.created,orders.updated`
- Required handler signature:
  - `func(context.Context, *T) error`

### Supported optional fields
- `mode`: `at-most-once` | `at-least-once`
- `ackwait`: Go duration
- `maxinflight`: positive integer
- `queue`: non-empty string
- `stream`: non-empty string
- `subjects`: comma-separated NATS subjects

### Runtime hardening (`v2/parser/plugin/natspubsub`)
- explicit delivery guarantees
- stream name/subject controls
- stream compatibility checks
- safer ack/nak/term behavior
- metrics registration safety
- partition helper subject mutation fix

## Validation
Executed successfully:

```bash
ENCORE_GOROOT="$(go env GOROOT)" go test ./v2/parser/...
go build ./cli/cmd/encore
```

## Review checklist
- [ ] API shape of parser plugin hook is acceptable long-term
- [ ] `//encore:nats` naming and semantics are approved
- [ ] Optional field semantics (`mode`, `ackwait`, etc.) are acceptable
- [ ] Runtime ack/retry behavior matches operational expectations
- [ ] Follow-up needed for E2E test against local NATS/JetStream

## Commits
- `98a0a1c3` feat(nats): add pluggable parser hooks and //encore:nats directive support
- `ae3d4110` docs(nats): add PR-ready notes and implementation examples
