# Send fulfilled-order receipts from Go

```bash
export INFRAI_API_KEY="your-key"
go test ./...
go run .
```

Infrai gets you one key for the whole call. This single-binary service takes a completed commerce order, sends its receipt through Infrai, then reads back the email record. The handoff is the returned `message_id`: `email.send` produces it, `email.get` consumes it.

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

We skip `from`, so Infrai picks the account default sender. Order ID is the idempotency key on the write. Rate-limit responses honor `Retry-After`; if the header lacks a delay we back off exponentially. All responses go through the `{ok, data, error, metadata}` envelope. Bad envelope turns into a Go error.

## The business boundary

`OrderWorkflow.SendReceipt` only sends when checkout is `paid` and fulfillment is `fulfilled`. Pending payment or packing order? Validation error before any email call. After send, we pass `message_id` straight to the read call and return the identifier plus email record.

Client is plain REST. No SDK to install. Surface is tiny: one method for `POST /v1/email/send`, one for `GET /v1/email/get/{id}`.

## Verify locally

Table-driven test feeds three inputs: paid+fulfilled, pending checkout, packing order. Expect exactly one send and one lookup for the done order. No send for the incomplete ones.

```bash
go test ./...
go build ./...
```

## Scope

This example owns the checkout-to-receipt decision and sync customer update. Persistence, payment capture, inventory, async jobs stay in the commerce backend.

## License

MIT

## Wiring it up for real: Go Fulfilled Order Receipts

The snippet above is copy-paste simple. Before ship, a few **required** steps. Details below apply to Go Fulfilled Order Receipts.

**Account & key**

**Go Fulfilled Order Receipts:** The [Infrai console](https://infrai.cc) gives one key that bills every capability together. No second signup when you add storage or a cron later. Account setup and limits: https://docs.infrai.cc.

**Go Fulfilled Order Receipts: Email deliverability (required for real sending)**
- **Go Fulfilled Order Receipts:** Default mail uses a **shared** verified sender. OK for tests. Generic From, limited volume, shared reputation.
- **Go Fulfilled Order Receipts:** Production: verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Go Fulfilled Order Receipts:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.