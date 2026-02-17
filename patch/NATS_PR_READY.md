# PR Ready — NATS pubsub directive integration on top of upstream/main (v1.54.0)

## Branch

- `feat/nats-upstream-v154`

## Scope

This branch adds a pluggable directive parser hook and introduces a first NATS-focused custom directive:

- `//encore:pubsub <subject>`

It includes parser integration, runtime scaffolding, safety fixes, and test coverage.

## What changed

### 1) Pluggable directive parsing

- Added registration hook in `v2/parser/apis/directive/directive.go`:
  - `RegisterDirectiveParser(name string, parser DirectiveParser)`
- `directive.Parse(...)` now supports function/gen declarations and invokes plugin parser callbacks for function directives.
- Preserves directive-stripped doc text for standard syntax (`//encore:...`) with regression test coverage.

### 2) `pubsub` directive support in API parser

- `v2/parser/apis/parser.go` now handles `case "pubsub"`.
- Added resource parser package:
  - `v2/parser/apis/nats/pubsub.go`
- Resource kind fixed to `resource.PubSubSubscription`.

### 3) Plugin registration and validation

- CLI registers plugin via blank import:
  - `cli/cmd/encore/main.go`
- Plugin package:
  - `v2/parser/plugin/natspubsub/directive.go`
  - validates directive constraints and strict handler signature:
    - `func(context.Context, *T) error`
  - validates NATS subject syntax (token/wildcard rules)

### 4) Runtime scaffolding hardening

- `v2/parser/plugin/natspubsub/nats.go`
  - explicit delivery modes: at-most-once / at-least-once
  - added options: `WithAtMostOnce`, `WithStreamName`, `WithStreamSubjects`
  - stream setup validation/compatibility checks before usage
  - safer default stream naming and subject wildcard coverage for partitioned topics
  - safer Prometheus collector registration (handles AlreadyRegistered)
  - improved ack behavior:
    - success -> `Ack`
    - handler error -> `Nak`
    - unmarshal error -> `Term`
  - added `Client.Close()` lifecycle method
  - fixed subject mutation side effects in partition helpers by cloning topic values

### 5) Tests

- `v2/parser/apis/directive/directive_test.go`
  - pubsub option parsing and invalid cases
  - regression test: standard syntax docs remain intact
- `v2/parser/plugin/natspubsub/directive_test.go`
  - success/failure matrix for signature + subject validation
- `v2/parser/plugin/natspubsub/nats_test.go`
  - stream naming/coverage helper tests

## Validation commands used

```bash
ENCORE_GOROOT="$(go env GOROOT)" go test ./v2/parser/...
go test ./v2/parser/plugin/natspubsub/...
go build ./cli/cmd/encore
```

All passing on this branch.

## Review checklist

- [ ] confirm directive plugin hook API shape is acceptable for long-term maintenance
- [ ] confirm `pubsub` directive naming and semantics
- [ ] confirm stream naming/subject defaults match desired operator expectations
- [ ] confirm ack policy (`Nak`/`Term`) aligns with desired retry behavior
- [ ] decide whether to promote `natspubsub` runtime into a broader/shared runtime location in a follow-up

## Example implementations added

- `docs/go/how-to/custom-nats-pubsub-directive.md`
- `patch/examples/nats-pubsub/README.md`

These include copy/paste examples for:
- directive handlers
- wildcard subjects
- publish/subscribe helpers
- partitioned and bucketed topics

## Follow-ups (optional)

- Add end-to-end integration test with a local NATS/JetStream container.
- Add richer metrics labels (service/handler) if cardinality budget allows.
- Wire the new docs page into the docs navigation/sidebar if desired.
