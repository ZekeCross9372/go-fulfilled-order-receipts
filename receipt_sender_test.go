package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type recordingMailer struct {
	sent      int
	lookedUp  []string
	messageID string
}

func (m *recordingMailer) SendEmail(_ context.Context, _ Order) (string, error) {
	m.sent++
	return m.messageID, nil
}

func (m *recordingMailer) GetEmail(_ context.Context, id string) (map[string]any, error) {
	m.lookedUp = append(m.lookedUp, id)
	return map[string]any{"state": "accepted"}, nil
}

func TestOrderWorkflowReceiptDecision(t *testing.T) {
	tests := []struct {
		name          string
		checkout      string
		fulfillment   string
		wantErr       string
		wantSendCount int
	}{
		{name: "paid and fulfilled sends receipt", checkout: "paid", fulfillment: "fulfilled", wantSendCount: 1},
		{name: "unpaid checkout is blocked", checkout: "pending", fulfillment: "fulfilled", wantErr: "receipt requires a paid checkout"},
		{name: "unfulfilled order is blocked", checkout: "paid", fulfillment: "packing", wantErr: "receipt requires fulfilled items"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mailer := &recordingMailer{messageID: "msg_123"}
			workflow := OrderWorkflow{Mail: mailer}
			result, err := workflow.SendReceipt(context.Background(), Order{
				ID: "ord_42", CustomerEmail: "buyer@example.com", Currency: "usd",
				TotalMinorUnits: 2599, CheckoutStatus: tt.checkout, Fulfillment: tt.fulfillment,
			})

			if tt.wantErr != "" {
				if !errors.Is(err, errors.New(tt.wantErr)) && (err == nil || err.Error() != tt.wantErr) {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("SendReceipt() error = %v", err)
				}
				if result.MessageID != "msg_123" || !reflect.DeepEqual(mailer.lookedUp, []string{"msg_123"}) {
					t.Fatalf("handoff result = %#v, lookups = %#v", result, mailer.lookedUp)
				}
			}
			if mailer.sent != tt.wantSendCount {
				t.Fatalf("send count = %d, want %d", mailer.sent, tt.wantSendCount)
			}
		})
	}
}
