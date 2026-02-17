# NATS pubsub extension examples

This folder contains copy/paste examples for the custom `//encore:pubsub` extension.

## Quick checklist

- Handler directive:

```go
//encore:pubsub orders.created
func Handle(ctx context.Context, evt *OrderCreated) error { ... }
```

- Signature required: `func(context.Context, *T) error`
- Subject rules:
  - `orders.created` ✅
  - `orders.*` ✅
  - `orders.>` ✅
  - `orders/created` ❌

## Runtime helper examples

See `docs/go/how-to/custom-nats-pubsub-directive.md` for full snippets:

- at-least-once topic
- at-most-once topic
- custom stream name/subjects
- partitioned topics
- bucketed topics

## Validation commands

```bash
ENCORE_GOROOT="$(go env GOROOT)" go test ./v2/parser/...
go test ./v2/parser/plugin/natspubsub/...
go build ./cli/cmd/encore
```
