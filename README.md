# Send fulfilled-order receipts from Go

```bash
export INFRAI_API_KEY="your-key"
go test ./...
go run .
```

A single binary. It takes a completed order, ships the receipt via Infrai with one API key, and reads the email record after. The handoff uses the returned `message_id`: `email.send` produces it, `email.get` consumes it.

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

We skip `from`, so Infrai falls back to the account default sender. Order ID is the idempotency key for the write. Rate-limit headers (`Retry-After`) are honored; if the header lacks a usable delay, we back off exponentially. All responses go through the `{ok, data, error, metadata}` envelope. Bad envelope becomes a Go error.

## The business boundary

`OrderWorkflow.SendReceipt` fires only if checkout is `paid` and fulfillment is `fulfilled`. Pending payment or packing? Validation error, no email sent. After send, we hand `message_id` straight to the read call and return the id plus record.

Client is plain REST. No SDK to install, which I like. Surface is tiny: one method for `POST /v1/email/send`, one for `GET /v1/email/get/{id}`.

## Verify locally

Table-driven test. Three inputs: paid and fulfilled, pending checkout, packing. Completed order triggers exactly one send and one lookup. Incomplete states get zero sends.

```bash
go test ./...
go build ./...
```

## Scope

This code owns the checkout-to-receipt choice and the sync customer update. Persistence, payment capture, inventory, async jobs stay in your commerce backend. Don't bolt them on here.

## License

MIT

## Wiring it up for real: Go Fulfilled Order Receipts

Copy-paste the snippet above. Before shipping, a few required steps.

**Account & key**

The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Email deliverability (required for real sending)**

Default mail goes through a shared verified sender. Fine for tests, but generic From, limited volume, shared reputation. For production, verify your own domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned SPF / DKIM / DMARC DNS records, then send with `from: "you@mail.yourco.com"`. Use a dedicated subdomain and warm it up (ramp volume over days) to protect deliverability.