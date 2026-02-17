---
seotitle: Custom NATS pubsub directive examples
seodesc: Examples for using the custom //encore:pubsub directive and natspubsub runtime helpers.
title: Custom NATS pubsub directive
subtitle: Practical implementation examples
lang: go
---

<Callout type="warning">
This page documents a custom extension available in this fork/branch, not the default upstream Encore API.
</Callout>

This extension adds support for:

- `//encore:pubsub <subject>` on handler functions
- runtime helpers in `encr.dev/v2/parser/plugin/natspubsub`

## Example 1 — Minimal handler with directive

```go
package orders

import (
	"context"
)

type OrderCreated struct {
	OrderID string
	UserID  string
}

//encore:pubsub orders.created
func HandleOrderCreated(ctx context.Context, evt *OrderCreated) error {
	// process event
	return nil
}
```

Handler signature must be exactly:

```go
func(context.Context, *T) error
```

## Example 2 — Valid wildcard subjects

```go
//encore:pubsub orders.*
func HandleOrderEvents(ctx context.Context, evt *OrderEvent) error { return nil }

//encore:pubsub orders.>
func HandleOrderTree(ctx context.Context, evt *OrderEvent) error { return nil }
```

Subject validation follows NATS token/wildcard rules:

- `*` must be a full token
- `>` must be the final token

## Example 3 — Runtime publish/subscribe helper

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

func StartInvoiceWorker() error {
	return invoices.Subscribe("invoice-worker", natspubsub.SubscriptionConfig[InvoiceIssued]{
		Handler: func(ctx context.Context, evt *InvoiceIssued) error {
			// process evt
			return nil
		},
	})
}
```

## Example 4 — Partitioned topics

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

## Example 5 — Bucketed partitioning

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

## Troubleshooting

- If parser tests fail with install-root errors, run tests with:

```bash
ENCORE_GOROOT="$(go env GOROOT)" go test ./v2/parser/...
```

- Build check:

```bash
go build ./cli/cmd/encore
```
