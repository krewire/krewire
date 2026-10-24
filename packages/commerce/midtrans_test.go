package commerce_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krewire/krewire/packages/commerce"
	"github.com/krewire/krewire/packages/model"
)

func TestCalculateMidtransSignature(t *testing.T) {
	orderID := "ORDER-101"
	statusCode := "200"
	grossAmount := "150000.00"
	serverKey := "SB-Mid-server-testkey123"

	sig := commerce.CalculateMidtransSignature(orderID, statusCode, grossAmount, serverKey)
	if sig == "" {
		t.Fatal("expected non-empty signature hash")
	}

	// Calling with exact same inputs must yield exact same deterministic hash
	sig2 := commerce.CalculateMidtransSignature(orderID, statusCode, grossAmount, serverKey)
	if sig != sig2 {
		t.Fatalf("expected deterministic signature, got %s and %s", sig, sig2)
	}

	// Tampered amount must produce different hash
	sigTampered := commerce.CalculateMidtransSignature(orderID, statusCode, "999999.00", serverKey)
	if sig == sigTampered {
		t.Fatal("tampered amount must not match signature")
	}
}

func TestVerifyWebhook_ValidSignature(t *testing.T) {
	serverKey := "SB-Mid-server-validkey"
	provider, err := commerce.NewMidtransProvider(commerce.MidtransConfig{
		ServerKey:    serverKey,
		ClientKey:    "SB-Mid-client-validkey",
		IsProduction: false,
	})
	if err != nil {
		t.Fatalf("failed to init provider: %v", err)
	}

	orderID := "ORD-TEST-200"
	statusCode := "200"
	grossAmount := "250000.00"
	sig := commerce.CalculateMidtransSignature(orderID, statusCode, grossAmount, serverKey)

	payload := map[string]interface{}{
		"order_id":           orderID,
		"transaction_id":     "tx-uuid-1234",
		"status_code":        statusCode,
		"gross_amount":       grossAmount,
		"payment_type":       "qris",
		"transaction_status": "settlement",
		"signature_key":      sig,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}

	event, err := provider.VerifyWebhook(payloadBytes)
	if err != nil {
		t.Fatalf("verify webhook failed: %v", err)
	}

	if event.OrderID != orderID {
		t.Errorf("expected orderID %s, got %s", orderID, event.OrderID)
	}
	if event.TransactionStatus != commerce.StatusSettled {
		t.Errorf("expected status %s, got %s", commerce.StatusSettled, event.TransactionStatus)
	}
	if event.GrossAmount != 250000.00 {
		t.Errorf("expected gross amount 250000, got %f", event.GrossAmount)
	}
}

func TestVerifyWebhook_InvalidSignature(t *testing.T) {
	provider, _ := commerce.NewMidtransProvider(commerce.MidtransConfig{
		ServerKey: "SB-Mid-server-secret",
	})

	payload := map[string]interface{}{
		"order_id":           "ORD-FAIL-1",
		"status_code":        "200",
		"gross_amount":       "10000.00",
		"signature_key":      "invalid-forged-signature",
		"transaction_status": "settlement",
	}

	payloadBytes, _ := json.Marshal(payload)
	_, err := provider.VerifyWebhook(payloadBytes)
	if err == nil {
		t.Fatal("expected error on forged webhook signature, got nil")
	}
}

func TestCreatePayment_MockServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		auth := r.Header.Get("Authorization")
		if auth == "" {
			t.Errorf("expected Authorization header")
		}

		resp := map[string]string{
			"token":        "snap-token-xyz-123",
			"redirect_url": "https://app.sandbox.midtrans.com/snap/v2/vtweb/snap-token-xyz-123",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	req := commerce.PaymentRequest{
		OrderID:     "INV-2026-001",
		GrossAmount: 150000,
		Currency:    "IDR",
		Tier:        model.TierPro,
		Customer: commerce.CustomerDetail{
			FirstName: "Budi",
			Email:     "budi@example.com",
		},
	}

	customClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			resp := map[string]string{
				"token":        "snap-token-xyz-123",
				"redirect_url": "https://app.sandbox.midtrans.com/snap/v2/vtweb/snap-token-xyz-123",
			}
			data, _ := json.Marshal(resp)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(data)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	provider, err := commerce.NewMidtransProvider(commerce.MidtransConfig{
		ServerKey:  "SB-Mid-server-test",
		HTTPClient: customClient,
	})
	if err != nil {
		t.Fatal(err)
	}

	res, err := provider.CreatePayment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreatePayment failed: %v", err)
	}

	if res.Token != "snap-token-xyz-123" {
		t.Errorf("expected token snap-token-xyz-123, got %s", res.Token)
	}
	if res.Status != commerce.StatusPending {
		t.Errorf("expected pending status, got %s", res.Status)
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
