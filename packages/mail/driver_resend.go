package mail

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/krewire/krewire/packages/errs"
)

// ResendConfig contains options for the Resend email platform.
type ResendConfig struct {
	APIKey     string
	Endpoint   string // default: "https://api.resend.com/emails"
	HTTPClient *http.Client
}

// ResendDriver delivers email using the Resend REST API.
type ResendDriver struct {
	apiKey   string
	endpoint string
	client   *http.Client
}

// NewResendDriver creates a Resend delivery driver.
func NewResendDriver(cfg ResendConfig) (*ResendDriver, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errs.UsageError("mail: resend api_key is required")
	}

	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = "https://api.resend.com/emails"
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	return &ResendDriver{
		apiKey:   cfg.APIKey,
		endpoint: endpoint,
		client:   client,
	}, nil
}

func (d *ResendDriver) Name() string {
	return "resend"
}

func (d *ResendDriver) Close() error {
	return nil
}

type resendAttachment struct {
	Filename string `json:"filename"`
	Content  string `json:"content"` // base64
}

type resendRequest struct {
	From        string             `json:"from"`
	To          []string           `json:"to"`
	Cc          []string           `json:"cc,omitempty"`
	Bcc         []string           `json:"bcc,omitempty"`
	ReplyTo     string             `json:"reply_to,omitempty"`
	Subject     string             `json:"subject"`
	HTML        string             `json:"html,omitempty"`
	Text        string             `json:"text,omitempty"`
	Attachments []resendAttachment `json:"attachments,omitempty"`
}

func (d *ResendDriver) Send(ctx context.Context, msg *Message) error {
	reqBody := resendRequest{
		From:    msg.From,
		To:      msg.To,
		Cc:      msg.Cc,
		Bcc:     msg.Bcc,
		ReplyTo: msg.ReplyTo,
		Subject: msg.Subject,
		HTML:    msg.HTMLBody,
		Text:    msg.TextBody,
	}

	for _, att := range msg.Attachments {
		reqBody.Attachments = append(reqBody.Attachments, resendAttachment{
			Filename: att.Filename,
			Content:  base64.StdEncoding.EncodeToString(att.Content),
		})
	}

	payloadBytes, err := json.Marshal(reqBody)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("mail: serialize resend payload: %v", err))
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, d.endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Authorization", "Bearer "+d.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	res, err := d.client.Do(httpReq)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("mail: resend api call failed: %v", err))
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return errs.FailureError(fmt.Sprintf("mail: resend api error (status %d): %s", res.StatusCode, string(body)))
	}

	return nil
}

var _ Driver = (*ResendDriver)(nil)
