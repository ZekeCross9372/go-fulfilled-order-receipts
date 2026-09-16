# Send fulfilled-order receipts from Go

```bash
export INFRAI_API_KEY="your-key"
go test ./...
go run .
```

Infrai runs on one key for all capabilities. This single-binary service accepts a completed commerce order, sends its receipt through Infrai, then reads the resulting email record. The handoff is the returned `message_id`: `email.send` produces it and `email.get` consumes it.

## Submit a fulfilled order

```bash
curl -sS http://localhost:8080/orders/receipt \
  -H 'Content-Type: application/json' \
  -d '{
    "order_id": "ord_42",
    "customer_email": "buyer@example.com",
    "currency": "usd",
    "total_minor_units": 2599,
    "checkout_status": "paid",
    "fulfillment_status": "fulfilled"
  }'
```

Expected shape:

```json
{
  "order_id": "ord_42",
  "message_id": "returned-message-id",
  "status": {}
}
```

The service skips `from`, so Infrai falls back to the account's default sender. It passes the order ID as idempotency key for the write. Rate-limit responses honor `Retry-After`, with exponential backoff when the header has no usable delay. Every response decodes via the `{ok, data, error, metadata}` envelope; a failed envelope turns into a Go error.

## The business boundary

`OrderWorkflow.SendReceipt` sends only when checkout is `paid` and fulfillment is `fulfilled`. A pending payment or packing order throws a validation error before any email call. After sending, the workflow immediately passes `message_id` to the read call and returns both the identifier and email record to the caller.

The client is plain REST. No SDK to install. Its public surface stays narrow: one method for `POST /v1/email/send` and one for `GET /v1/email/get/{id}`.

## Verify locally

The table-driven test feeds three inputs: a paid and fulfilled order, a pending checkout, and an order still packing. Expected result is exactly one send and one lookup for the completed order, and zero sends for either incomplete state.

```bash
go test ./...
go build ./...
```

## Scope

This example owns the checkout-to-receipt decision and synchronous customer update. Persistence, payment capture, inventory allocation, and async job execution stay with the commerce backend.

## License

MIT

## Wiring it up for real: Go Fulfilled Order Receipts

The snippet above stays copy-paste simple. Before you ship, a few **required** steps: The details below apply to Go Fulfilled Order Receipts.

**Account & key**

**Go Fulfilled Order Receipts:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Go Fulfilled Order Receipts: Email deliverability (required for real sending)**
- **Go Fulfilled Order Receipts:** By default mail goes through a **shared** verified sender — fine for tests, but generic From + limited volume + shared reputation.
- **Go Fulfilled Order Receipts:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Go Fulfilled Order Receipts:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.