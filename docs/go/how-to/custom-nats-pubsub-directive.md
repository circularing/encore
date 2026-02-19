---
seotitle: Custom NATS pubsub directive examples
seodesc: Examples for using the custom //encore:nats directive and natspubsub runtime helpers.
title: Custom NATS pubsub directive
subtitle: Practical implementation examples
lang: go
---

<Callout type="warning">
This page documents a custom extension available in this fork/branch, not the default upstream Encore API.
</Callout>

This extension adds support for:

- `//encore:nats <subject>` on handler functions
- automatic subscription wiring (no manual bootstrap required)
- runtime helpers in `encr.dev/v2/parser/plugin/natspubsub`

Defaults are designed so this is enough:

```go
//encore:nats orders.created
func HandleOrderCreated(ctx context.Context, evt *OrderCreated) error { return nil }
```

When `stream`/`subjects` are omitted, the generated runtime wiring uses a shared default stream per subject root
(for example `orders.created` -> stream `encore_nats_orders` with subjects `orders.>`), which avoids common
JetStream overlap errors for related subjects.

## Example 1 — End-to-end minimal flow (`orders.created`)

```go
package orders

import (
	"context"
	"fmt"

	"encr.dev/v2/parser/plugin/natspubsub"
)

type OrderCreated struct {
	OrderID string `json:"order_id"`
	UserID  string `json:"user_id"`
}

// Publisher side (NATS subject-aware helper):
var natsClient = natspubsub.NewClient()
var OrdersCreated = natspubsub.NewTopic[OrderCreated](natsClient, "orders.created")

func PublishOrderCreated(ctx context.Context, orderID, userID string) error {
	_, err := OrdersCreated.Publish(ctx, &OrderCreated{
		OrderID: orderID,
		UserID:  userID,
	})
	return err
}

// Subscriber side (auto-wired by the directive):
//encore:nats orders.created
func HandleOrderCreated(ctx context.Context, evt *OrderCreated) error {
	fmt.Println("received order", evt.OrderID, "for user", evt.UserID)
	return nil
}
```

Handler signature must be exactly:

```go
func(context.Context, *T) error
```

If you use `encore.dev/pubsub.NewTopic` directly, note that topic names must be string literals in kebab-case
(for example `orders-created`, not `orders.created`).

## Example 2 — Directive options (`mode`, `ackwait`, `queue`, ...)

```go
package orders

import "context"

type OrderEvent struct {
	EventType string `json:"event_type"` // created, updated, cancelled
	OrderID   string `json:"order_id"`
}

// Optional fields after the subject are parsed and applied to runtime wiring:
// - mode=at-most-once|at-least-once
// - ackwait=30s
// - maxinflight=64
// - queue=orders-workers
// - stream=orders_events
// - subjects=orders.created,orders.updated
//encore:nats orders.created mode=at-least-once ackwait=30s maxinflight=64 queue=orders-workers stream=orders_events subjects=orders.created,orders.updated
func HandleOrderCreatedAdvanced(ctx context.Context, evt *OrderEvent) error {
	return nil
}
```

## Example 3 — Wildcards and routing by event type

```go
package orders

import "context"

type OrderEvent struct {
	EventType string `json:"event_type"` // created|updated|cancelled
	OrderID   string `json:"order_id"`
}

// Match one token: orders.created, orders.updated, ...
//encore:nats orders.*
func HandleOrderEvents(ctx context.Context, evt *OrderEvent) error {
	switch evt.EventType {
	case "created":
		// create side effects
	case "updated":
		// update projections
	default:
		// ignore unknown event types
	}
	return nil
}

// Match full hierarchy: orders.eu.created, orders.us.cancelled, ...
//encore:nats orders.>
func HandleOrderTree(ctx context.Context, evt *OrderEvent) error { return nil }
```

Wildcard validation follows NATS token rules:

- `*` must be a full token
- `>` must be the final token

## Example 4 — Runtime helper for publishing outside `encore.dev/pubsub`

```go
package billing

import (
	"context"
	"time"

	"encr.dev/v2/parser/plugin/natspubsub"
)

type InvoiceIssued struct {
	InvoiceID string
	UserID    string
}

var client = natspubsub.NewClient()

var invoices = natspubsub.NewTopic[InvoiceIssued](
	client,
	"billing.invoice.issued",
	natspubsub.WithAtLeastOnce(),
	natspubsub.WithStreamName("billing_invoice_events"),
	natspubsub.WithStreamSubjects("billing.invoice.>"),
	natspubsub.WithSubscriptionOptions(30*time.Second, 64, "billing-workers"),
)

func PublishInvoiceIssued(ctx context.Context, evt *InvoiceIssued) error {
	_, err := invoices.Publish(ctx, evt)
	return err
}

// Handlers declared with //encore:nats are subscribed automatically.
// Use this helper when you need explicit NATS/JetStream stream control.
```

## Example 5 — Partitioned publish by user

```go
package notifications

import (
	"context"

	"encr.dev/v2/parser/plugin/natspubsub"
)

type UserNotification struct {
	Message string
}

func SendForUser(ctx context.Context, userID string, msg string) error {
	client := natspubsub.NewClient()
	pt := natspubsub.NewPartitionedTopic[UserNotification](client, "user.notifications")

	_, err := pt.PublishForUser(ctx, userID, &UserNotification{Message: msg})
	return err
}
```

## Example 6 — Stable bucket partitioning

```go
package streams

import (
	"context"

	"encr.dev/v2/parser/plugin/natspubsub"
)

type Event struct{ Key string }

func PublishWithStableBucket(ctx context.Context, key string, evt *Event) error {
	client := natspubsub.NewClient()
	bt := natspubsub.NewBucketedTopic[Event](client, "events.partitioned", 32)
	_, err := bt.PublishWithKey(ctx, key, evt)
	return err
}
```

## Example 7 — Test from Dashboard

1. Start app locally.
2. Open the local Encore dashboard.
3. Go to the Pub/Sub topic `orders.created`.
4. Publish a JSON payload matching your message type:

```json
{
  "order_id": "ord_123",
  "user_id": "usr_42"
}
```

5. Verify subscriber execution in traces/logs for `HandleOrderCreated`.

## Troubleshooting

- If parser tests fail with install-root errors, run tests with:

```bash
ENCORE_GOROOT="$(go env GOROOT)" go test ./v2/parser/...
```

- Build check:

```bash
go build ./cli/cmd/encore
```
