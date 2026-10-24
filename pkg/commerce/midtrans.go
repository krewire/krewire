package commerce

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/krewire/krewire/pkg/errs"
)

const (
	// Midtrans Provider Name
	ProviderMidtrans = "midtrans"

	midtransSandboxSnapURL    = "https://app.sandbox.midtrans.com/snap/v1/transactions"
	midtransProductionSnapURL = "https://app.midtrans.com/snap/v1/transactions"

	midtransSandboxAPIURL    = "https://api.sandbox.midtrans.com/v2"
	midtransProductionAPIURL = "https://api.midtrans.com/v2"
)

// MidtransConfig contains credentials and options for the Midtrans provider.
type MidtransConfig struct {
	ServerKey    string
	ClientKey    string
	IsProduction bool
	HTTPClient   *http.Client
}

// MidtransProvider implements Provider using Midtrans Snap & Core API.
type MidtransProvider struct {
	serverKey    string
	clientKey    string
	isProduction bool
	snapURL      string
	apiURL       string
	client       *http.Client
}

// NewMidtransProvider creates a new Midtrans payment provider.
func NewMidtransProvider(cfg MidtransConfig) (*MidtransProvider, error) {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}

	snapURL := midtransSandboxSnapURL
	apiURL := midtransSandboxAPIURL
	if cfg.IsProduction {
		snapURL = midtransProductionSnapURL
		apiURL = midtransProductionAPIURL
	}

	return &MidtransProvider{
		serverKey:    cfg.ServerKey,
		clientKey:    cfg.ClientKey,
		isProduction: cfg.IsProduction,
		snapURL:      snapURL,
		apiURL:       apiURL,
		client:       cfg.HTTPClient,
	}, nil
}

// Name returns "midtrans".
func (m *MidtransProvider) Name() string {
	return ProviderMidtrans
}

// ClientKey returns the public client key (safe for browser frontends).
func (m *MidtransProvider) ClientKey() string {
	return m.clientKey
}

// IsProduction reports whether the provider targets the Midtrans production environment.
func (m *MidtransProvider) IsProduction() bool {
	return m.isProduction
}

type midtransSnapRequest struct {
	TransactionDetails midtransTxDetails  `json:"transaction_details"`
	CustomerDetails    *CustomerDetail    `json:"customer_details,omitempty"`
	ItemDetails        []ItemDetail       `json:"item_details,omitempty"`
	CustomField1       string             `json:"custom_field1,omitempty"` // e.g. Tier
	Callbacks          *midtransCallbacks `json:"callbacks,omitempty"`
}

type midtransTxDetails struct {
	OrderID     string  `json:"order_id"`
	GrossAmount float64 `json:"gross_amount"`
}

type midtransCallbacks struct {
	Finish string `json:"finish,omitempty"`
}

type midtransSnapResponse struct {
	Token         string   `json:"token"`
	RedirectURL   string   `json:"redirect_url"`
	ErrorMessages []string `json:"error_messages,omitempty"`
}

// CreatePayment initiates a transaction using Midtrans Snap API.
func (m *MidtransProvider) CreatePayment(ctx context.Context, req PaymentRequest) (*PaymentResponse, error) {
	if strings.TrimSpace(req.OrderID) == "" {
		return nil, errs.UsageError("order_id is required")
	}
	if req.GrossAmount <= 0 {
		return nil, errs.UsageError("gross_amount must be greater than zero")
	}

	snapReq := midtransSnapRequest{
		TransactionDetails: midtransTxDetails{
			OrderID:     req.OrderID,
			GrossAmount: req.GrossAmount,
		},
		ItemDetails:  req.Items,
		CustomField1: string(req.Tier),
	}

	if req.Customer.Email != "" || req.Customer.FirstName != "" {
		snapReq.CustomerDetails = &req.Customer
	}
	if req.CallbackURL != "" {
		snapReq.Callbacks = &midtransCallbacks{Finish: req.CallbackURL}
	}

	bodyBytes, err := json.Marshal(snapReq)
	if err != nil {
		return nil, errs.FailureError(fmt.Sprintf("failed to serialize snap request: %v", err))
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, m.snapURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, errs.FailureError(fmt.Sprintf("failed to create http request: %v", err))
	}

	m.applyAuthHeader(httpReq)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	res, err := m.client.Do(httpReq)
	if err != nil {
		return nil, errs.FailureError(fmt.Sprintf("midtrans snap request failed: %v", err))
	}
	defer res.Body.Close()

	respBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errs.FailureError(fmt.Sprintf("failed to read snap response body: %v", err))
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, errs.FailureError(
			fmt.Sprintf("midtrans snap error (status %d): %s", res.StatusCode, string(respBytes)),
		)
	}

	var snapRes midtransSnapResponse
	if err := json.Unmarshal(respBytes, &snapRes); err != nil {
		return nil, errs.FailureError(fmt.Sprintf("failed to decode snap response: %v", err))
	}

	return &PaymentResponse{
		OrderID:    req.OrderID,
		PaymentURL: snapRes.RedirectURL,
		Token:      snapRes.Token,
		Status:     StatusPending,
	}, nil
}

type midtransStatusResponse struct {
	StatusCode        string `json:"status_code"`
	StatusMessage     string `json:"status_message"`
	TransactionID     string `json:"transaction_id"`
	OrderID           string `json:"order_id"`
	GrossAmount       string `json:"gross_amount"`
	PaymentType       string `json:"payment_type"`
	TransactionTime   string `json:"transaction_time"`
	TransactionStatus string `json:"transaction_status"`
	FraudStatus       string `json:"fraud_status"`
	SignatureKey      string `json:"signature_key"`
}

// GetStatus checks transaction status from Midtrans Core API.
func (m *MidtransProvider) GetStatus(ctx context.Context, orderID string) (*PaymentResponse, error) {
	if strings.TrimSpace(orderID) == "" {
		return nil, errs.UsageError("order_id is required")
	}

	url := fmt.Sprintf("%s/%s/status", m.apiURL, orderID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errs.FailureError(fmt.Sprintf("failed to create http request: %v", err))
	}

	m.applyAuthHeader(httpReq)
	httpReq.Header.Set("Accept", "application/json")

	res, err := m.client.Do(httpReq)
	if err != nil {
		return nil, errs.FailureError(fmt.Sprintf("midtrans status request failed: %v", err))
	}
	defer res.Body.Close()

	respBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errs.FailureError(fmt.Sprintf("failed to read status response body: %v", err))
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, errs.FailureError(
			fmt.Sprintf("midtrans status query failed (status %d): %s", res.StatusCode, string(respBytes)),
		)
	}

	var statusRes midtransStatusResponse
	if err := json.Unmarshal(respBytes, &statusRes); err != nil {
		return nil, errs.FailureError(fmt.Sprintf("failed to decode midtrans status response: %v", err))
	}

	return &PaymentResponse{
		OrderID:       statusRes.OrderID,
		TransactionID: statusRes.TransactionID,
		Status:        mapMidtransStatus(statusRes.TransactionStatus, statusRes.FraudStatus),
	}, nil
}

// CancelPayment cancels a pending payment in Midtrans.
func (m *MidtransProvider) CancelPayment(ctx context.Context, orderID string) error {
	if strings.TrimSpace(orderID) == "" {
		return errs.UsageError("order_id is required")
	}

	url := fmt.Sprintf("%s/%s/cancel", m.apiURL, orderID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("failed to create cancel request: %v", err))
	}

	m.applyAuthHeader(httpReq)
	httpReq.Header.Set("Accept", "application/json")

	res, err := m.client.Do(httpReq)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("midtrans cancel request failed: %v", err))
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(res.Body)
		return errs.FailureError(
			fmt.Sprintf("midtrans cancel failed (status %d): %s", res.StatusCode, string(respBytes)),
		)
	}

	return nil
}

// VerifyWebhook validates incoming webhook payload signatures and deserializes the event.
// Midtrans signature formula: SHA512(order_id + status_code + gross_amount + ServerKey).
func (m *MidtransProvider) VerifyWebhook(payload []byte) (*WebhookEvent, error) {
	var notif midtransStatusResponse
	if err := json.Unmarshal(payload, &notif); err != nil {
		return nil, errs.UsageError(fmt.Sprintf("failed to parse webhook json payload: %v", err))
	}

	// Verify SHA-512 signature key if server key is configured
	if m.serverKey != "" {
		expectedSig := CalculateMidtransSignature(notif.OrderID, notif.StatusCode, notif.GrossAmount, m.serverKey)
		if notif.SignatureKey != expectedSig {
			return nil, errs.FailureError("invalid webhook signature: signature mismatch")
		}
	}

	var amount float64
	if notif.GrossAmount != "" {
		amount, _ = strconv.ParseFloat(notif.GrossAmount, 64)
	}

	return &WebhookEvent{
		OrderID:           notif.OrderID,
		TransactionID:     notif.TransactionID,
		StatusCode:        notif.StatusCode,
		GrossAmount:       amount,
		PaymentType:       notif.PaymentType,
		TransactionStatus: mapMidtransStatus(notif.TransactionStatus, notif.FraudStatus),
		FraudStatus:       notif.FraudStatus,
		SignatureKey:      notif.SignatureKey,
		RawPayload:        payload,
	}, nil
}

// CalculateMidtransSignature computes the canonical SHA-512 signature for Midtrans webhooks.
func CalculateMidtransSignature(orderID, statusCode, grossAmount, serverKey string) string {
	raw := fmt.Sprintf("%s%s%s%s", orderID, statusCode, grossAmount, serverKey)
	hash := sha512.Sum512([]byte(raw))
	return hex.EncodeToString(hash[:])
}

func mapMidtransStatus(txStatus, fraudStatus string) PaymentStatus {
	switch strings.ToLower(txStatus) {
	case "capture":
		if strings.ToLower(fraudStatus) == "challenge" {
			return StatusPending
		}
		return StatusSettled
	case "settlement":
		return StatusSettled
	case "pending":
		return StatusPending
	case "deny":
		return StatusFailed
	case "expire":
		return StatusExpired
	case "cancel":
		return StatusCancelled
	default:
		return StatusPending
	}
}

func (m *MidtransProvider) applyAuthHeader(req *http.Request) {
	if m.serverKey != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(m.serverKey + ":"))
		req.Header.Set("Authorization", "Basic "+auth)
	}
}
