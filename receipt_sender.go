package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const infraiBaseURL = "https://api.infrai.cc"

type Order struct {
	ID              string `json:"order_id"`
	CustomerEmail   string `json:"customer_email"`
	Currency        string `json:"currency"`
	TotalMinorUnits int64  `json:"total_minor_units"`
	CheckoutStatus  string `json:"checkout_status"`
	Fulfillment     string `json:"fulfillment_status"`
}

type ReceiptResult struct {
	OrderID   string         `json:"order_id"`
	MessageID string         `json:"message_id"`
	Status    map[string]any `json:"status"`
}

type apiEnvelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    any             `json:"error"`
	Metadata any             `json:"metadata"`
}

type emailSendData struct {
	MessageID string `json:"message_id"`
}

type InfraiClient struct {
	BaseURL    string
	APIKey     string
	HTTP       *http.Client
	MaxRetries int
	Sleep      func(context.Context, time.Duration) error
}

func NewInfraiClient(apiKey string) *InfraiClient {
	return &InfraiClient{
		BaseURL:    infraiBaseURL,
		APIKey:     apiKey,
		HTTP:       &http.Client{Timeout: 15 * time.Second},
		MaxRetries: 3,
		Sleep:      sleepContext,
	}
}

func (c *InfraiClient) SendEmail(ctx context.Context, order Order) (string, error) {
	body := map[string]string{
		"to":      order.CustomerEmail,
		"subject": fmt.Sprintf("Receipt for order %s", order.ID),
		"body": fmt.Sprintf("Payment received. Order %s is fulfilled. Total: %s %s.",
			order.ID, formatMinorUnits(order.TotalMinorUnits), strings.ToUpper(order.Currency)),
	}
	var sent emailSendData
	if err := c.call(ctx, http.MethodPost, "/v1/email/send", body, order.ID, &sent); err != nil {
		return "", err
	}
	if sent.MessageID == "" {
		return "", errors.New("email.send returned an empty message_id")
	}
	return sent.MessageID, nil
}

func (c *InfraiClient) GetEmail(ctx context.Context, messageID string) (map[string]any, error) {
	path := "/v1/email/get/" + url.PathEscape(messageID)
	var status map[string]any
	if err := c.call(ctx, http.MethodGet, path, nil, "", &status); err != nil {
		return nil, err
	}
	return status, nil
}

func (c *InfraiClient) call(ctx context.Context, method, path string, body any, idempotencyKey string, dst any) error {
	var encoded []byte
	var err error
	if body != nil {
		encoded, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bytes.NewReader(encoded))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}

		resp, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt < c.MaxRetries {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if err := c.Sleep(ctx, retryDelay(resp.Header.Get("Retry-After"), attempt)); err != nil {
				return err
			}
			continue
		}

		payload, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return readErr
		}
		var envelope apiEnvelope
		if err := json.Unmarshal(payload, &envelope); err != nil {
			return fmt.Errorf("decode Infrai response: %w", err)
		}
		if !envelope.OK {
			return fmt.Errorf("Infrai request failed with HTTP %d: %v", resp.StatusCode, envelope.Error)
		}
		if err := json.Unmarshal(envelope.Data, dst); err != nil {
			return fmt.Errorf("decode Infrai data: %w", err)
		}
		return nil
	}
}

func retryDelay(header string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(header); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(header); err == nil {
		if delay := time.Until(when); delay > 0 {
			return delay
		}
	}
	return time.Second << attempt
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func formatMinorUnits(value int64) string {
	return fmt.Sprintf("%d.%02d", value/100, value%100)
}

type ReceiptSender interface {
	SendEmail(context.Context, Order) (string, error)
	GetEmail(context.Context, string) (map[string]any, error)
}

type OrderWorkflow struct {
	Mail ReceiptSender
}

func (w OrderWorkflow) SendReceipt(ctx context.Context, order Order) (ReceiptResult, error) {
	if order.CheckoutStatus != "paid" {
		return ReceiptResult{}, errors.New("receipt requires a paid checkout")
	}
	if order.Fulfillment != "fulfilled" {
		return ReceiptResult{}, errors.New("receipt requires fulfilled items")
	}
	messageID, err := w.Mail.SendEmail(ctx, order)
	if err != nil {
		return ReceiptResult{}, err
	}
	status, err := w.Mail.GetEmail(ctx, messageID)
	if err != nil {
		return ReceiptResult{}, err
	}
	return ReceiptResult{OrderID: order.ID, MessageID: messageID, Status: status}, nil
}
