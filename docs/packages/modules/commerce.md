# `commerce`

Import: `github.com/krewire/krewire/packages/commerce`

## Purpose

Payment gateway integration, payment tokenization, and webhook notification verification for Krewire monetization (Model C). Provides provider abstractions with first-class support for Midtrans (Snap API, Core API, and SHA-512 signature verification for QRIS, Virtual Accounts, E-Wallets, and Credit Cards).

## Main API

- `Provider` interface:
  - `CreatePayment(ctx context.Context, req PaymentRequest) (*PaymentResponse, error)`
  - `GetStatus(ctx context.Context, orderID string) (*PaymentResponse, error)`
  - `VerifyWebhook(payload []byte) (*WebhookEvent, error)`
  - `CancelPayment(ctx context.Context, orderID string) error`
- `MidtransProvider` (implements `Provider`)
- `NewMidtransProvider(cfg MidtransConfig) (*MidtransProvider, error)`
- `CalculateMidtransSignature(orderID, statusCode, grossAmount, serverKey string) string`
- Domain types:
  - `PaymentRequest`, `PaymentResponse`, `CustomerDetail`, `ItemDetail`, `WebhookEvent`
  - `PaymentMethod` (`PaymentMethodQRIS`, `PaymentMethodBankTransfer`, `PaymentMethodCreditCard`, etc.)
  - `PaymentStatus` (`StatusPending`, `StatusSettled`, `StatusExpired`, `StatusFailed`, `StatusCancelled`)

## Example

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/krewire/krewire/packages/commerce"
	"github.com/krewire/krewire/packages/model"
)

func main() {
	provider, err := commerce.NewMidtransProvider(commerce.MidtransConfig{
		ServerKey:    os.Getenv("MIDTRANS_SERVER_KEY"),
		ClientKey:    os.Getenv("MIDTRANS_CLIENT_KEY"),
		IsProduction: os.Getenv("MIDTRANS_IS_PRODUCTION") == "true",
	})
	if err != nil {
		panic(err)
	}

	res, err := provider.CreatePayment(context.Background(), commerce.PaymentRequest{
		OrderID:     "INV-202610-001",
		GrossAmount: 150000,
		Currency:    "IDR",
		Tier:        model.TierPro,
		Customer: commerce.CustomerDetail{
			FirstName: "Budi",
			Email:     "budi@example.com",
		},
		CallbackURL: "https://cloud.krewire.com/billing/callback",
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Payment Snap URL: %s\n", res.PaymentURL)
}
```

## Security & Verification

All incoming webhook payloads are verified using HMAC SHA-512 with the merchant's `ServerKey` before processing (`SHA512(order_id + status_code + gross_amount + server_key)`). Never disable signature checks in production.
