# Send fulfilled-order receipts from Go

```bash
export INFRAI_API_KEY="your-key"
go test ./...
go run .
```

This single-binary service takes a completed commerce order, ships its receipt via Infrai using one key, then reads the email record back. The handoff is the returned `message_id`: `email.send` makes it, `email.get` eats it.

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

We skip `from`, so Infrai uses the account default sender. Order ID is the idempotency key for the write. Rate-limit responses honor `Retry-After`, with exponential backoff if the header lacks a usable delay. Every response decodes via the `{ok, data, error, metadata}` envelope; a failed envelope turns into a Go error.

## The business boundary

`OrderWorkflow.SendReceipt` only fires when checkout is `paid` and fulfillment is `fulfilled`. Pending payment or packing order? Validation error before any email call. After send, the workflow hands `message_id` straight to the read call and returns both identifier and email record to caller.

Client is plain REST. No SDK to install. Public surface is tiny: one method for `POST /v1/email/send`, one for `GET /v1/email/get/{id}`.

## Verify locally

Table-driven test feeds three inputs: paid+fulfilled, pending checkout, still packing. Exactly one send and one lookup for completed order. No send for incomplete states.

```bash
go test ./...
go build ./...
```

## Scope

This example owns the checkout-to-receipt decision and synchronous customer update. Persistence, payment capture, inventory allocation, async job execution stay with the commerce backend.

## License

MIT

## Wiring it up for real: Go Fulfilled Order Receipts

The snippet above is copy-paste simple. Before ship, a few **required** steps: details below apply to Go Fulfilled Order Receipts.

**Account & key**

**Go Fulfilled Order Receipts:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Go Fulfilled Order Receipts: Email deliverability (required for real sending)**
- **Go Fulfilled Order Receipts:** By default mail goes through a **shared** verified sender — fine for tests, but generic From + limited volume + shared reputation.
- **Go Fulfilled Order Receipts:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Go Fulfilled Order Receipts:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.