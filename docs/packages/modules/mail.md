# `mail`

Import: `github.com/krewire/krewire/packages/mail`

## Purpose

A modular, multi-driver transactional email delivery system. Abstracts email dispatch across SMTP, Resend, SendGrid, Postmark, and in-memory mock drivers. Features a fluent message builder, standard RFC 5322 MIME multipart encoding (HTML, plain text, and base64 attachments), and fail-closed validation.

## Main API

- **Core**:
  - `Message` and fluent builder: `NewMessage().SetFrom(...).AddTo(...).SetSubject(...).SetHTML(...).Attach(...)`
  - `BuildMIME(msg *Message) ([]byte, error)`
  - `Driver` interface (`Name() string`, `Send(ctx context.Context, msg *Message) error`, `Close() error`)
  - `Mailer`: `NewMailer(driver Driver) *Mailer`, `Mailer.Send(ctx context.Context, msg *Message) error`
- **Supported Drivers**:
  - **SMTP**: `NewSMTPDriver(cfg SMTPConfig)` (supports TLS, STARTTLS, Plain auth, custom ports)
  - **Resend**: `NewResendDriver(cfg ResendConfig)` (Resend REST API)
  - **SendGrid**: `NewSendGridDriver(cfg SendGridConfig)` (SendGrid v3 API)
  - **Postmark**: `NewPostmarkDriver(cfg PostmarkConfig)` (Postmark transactional email API)
  - **Memory**: `NewMemoryDriver()` (captures sent messages in memory for unit testing)

## Example

```go
package main

import (
	"context"

	"github.com/krewire/krewire/packages/mail"
)

func main() {
	ctx := context.Background()

	// Initialize Resend driver (or SMTP, SendGrid, Postmark)
	driver, err := mail.NewResendDriver(mail.ResendConfig{
		APIKey: "re_live_secret_key",
	})
	if err != nil {
		panic(err)
	}

	mailer := mail.NewMailer(driver)
	defer mailer.Close()

	msg := mail.NewMessage().
		SetFrom("Krewire Cloud <billing@krewire.com>").
		AddTo("client@example.com").
		SetSubject("Your Krewire Invoice #1024").
		SetHTML("<h1>Invoice Paid</h1><p>Thank you for using Krewire Cloud.</p>").
		Attach("invoice.pdf", "application/pdf", []byte("%PDF..."))

	if err := mailer.Send(ctx, msg); err != nil {
		panic(err)
	}
}
```

## Security & Architecture Boundaries

- **MIME & Injection Protection**: Rejects CRLF injection attempts in subject and header lines.
- **Pure Go**: Standard library `net/smtp` and `net/http` only; zero external CGO dependencies.
- **Downward Dependency Pyramid**: Depends only on Level 1 (`packages/errs`, `packages/model`).
