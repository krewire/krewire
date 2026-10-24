package commerce

import (
	"context"
	"time"

	"github.com/krewire/krewire/packages/model"
)

// PaymentMethod specifies the payment channel requested by or used by the customer.
type PaymentMethod string

const (
	PaymentMethodQRIS         PaymentMethod = "qris"
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
	PaymentMethodCreditCard   PaymentMethod = "credit_card"
	PaymentMethodEWallet      PaymentMethod = "ewallet"
	PaymentMethodConvenience  PaymentMethod = "convenience_store"
)

// PaymentStatus indicates the lifecycle state of a transaction.
type PaymentStatus string

const (
	StatusPending   PaymentStatus = "pending"
	StatusSettled   PaymentStatus = "settled"
	StatusExpired   PaymentStatus = "expired"
	StatusFailed    PaymentStatus = "failed"
	StatusCancelled PaymentStatus = "cancelled"
)

// ItemDetail represents a billable line item in an order.
type ItemDetail struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
	Category string  `json:"category,omitempty"`
}

// CustomerDetail represents the billing contact for an order.
type CustomerDetail struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Email     string `json:"email"`
	Phone     string `json:"phone,omitempty"`
}

// PaymentRequest contains information necessary to initiate a payment transaction.
type PaymentRequest struct {
	OrderID     string            `json:"order_id"`
	GrossAmount float64           `json:"gross_amount"`
	Currency    string            `json:"currency"` // Default: IDR
	Tier        model.Tier        `json:"tier,omitempty"`
	Customer    CustomerDetail    `json:"customer"`
	Items       []ItemDetail      `json:"items,omitempty"`
	Method      PaymentMethod     `json:"method,omitempty"`
	CallbackURL string            `json:"callback_url,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// PaymentResponse contains token and URL for customer payment completion.
type PaymentResponse struct {
	OrderID       string        `json:"order_id"`
	TransactionID string        `json:"transaction_id,omitempty"`
	PaymentURL    string        `json:"payment_url"`
	Token         string        `json:"token,omitempty"`
	Status        PaymentStatus `json:"status"`
	ExpiryTime    time.Time     `json:"expiry_time,omitempty"`
}

// WebhookEvent represents an incoming payment status notification from a payment provider.
type WebhookEvent struct {
	OrderID           string        `json:"order_id"`
	TransactionID     string        `json:"transaction_id"`
	StatusCode        string        `json:"status_code"`
	GrossAmount       float64       `json:"gross_amount"`
	PaymentType       string        `json:"payment_type"`
	TransactionStatus PaymentStatus `json:"transaction_status"`
	FraudStatus       string        `json:"fraud_status,omitempty"`
	SignatureKey      string        `json:"signature_key"`
	RawPayload        []byte        `json:"-"`
}

// Provider defines the interface implemented by payment gateway providers (e.g., Midtrans).
type Provider interface {
	Name() string
	CreatePayment(ctx context.Context, req PaymentRequest) (*PaymentResponse, error)
	GetStatus(ctx context.Context, orderID string) (*PaymentResponse, error)
	VerifyWebhook(payload []byte) (*WebhookEvent, error)
	CancelPayment(ctx context.Context, orderID string) error
}
