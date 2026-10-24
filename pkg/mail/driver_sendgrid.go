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

	"github.com/krewire/krewire/pkg/errs"
)

// SendGridConfig contains options for SendGrid v3 API.
type SendGridConfig struct {
	APIKey     string
	Endpoint   string // default: "https://api.sendgrid.com/v3/mail/send"
	HTTPClient *http.Client
}

// SendGridDriver delivers email via SendGrid.
type SendGridDriver struct {
	apiKey   string
	endpoint string
	client   *http.Client
}

// NewSendGridDriver initializes a SendGrid delivery driver.
func NewSendGridDriver(cfg SendGridConfig) (*SendGridDriver, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errs.UsageError("mail: sendgrid api_key is required")
	}

	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = "https://api.sendgrid.com/v3/mail/send"
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	return &SendGridDriver{
		apiKey:   cfg.APIKey,
		endpoint: endpoint,
		client:   client,
	}, nil
}

func (d *SendGridDriver) Name() string {
	return "sendgrid"
}

func (d *SendGridDriver) Close() error {
	return nil
}

type sgEmail struct {
	Email string `json:"email"`
}

type sgPersonalization struct {
	To  []sgEmail `json:"to"`
	Cc  []sgEmail `json:"cc,omitempty"`
	Bcc []sgEmail `json:"bcc,omitempty"`
}

type sgContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type sgAttachment struct {
	Content  string `json:"content"` // base64
	Type     string `json:"type"`
	Filename string `json:"filename"`
}

type sgRequest struct {
	Personalizations []sgPersonalization `json:"personalizations"`
	From             sgEmail             `json:"from"`
	Subject          string              `json:"subject"`
	Content          []sgContent         `json:"content"`
	Attachments      []sgAttachment      `json:"attachments,omitempty"`
}

func (d *SendGridDriver) Send(ctx context.Context, msg *Message) error {
	p := sgPersonalization{}
	for _, to := range msg.To {
		p.To = append(p.To, sgEmail{Email: to})
	}
	for _, cc := range msg.Cc {
		p.Cc = append(p.Cc, sgEmail{Email: cc})
	}
	for _, bcc := range msg.Bcc {
		p.Bcc = append(p.Bcc, sgEmail{Email: bcc})
	}

	reqBody := sgRequest{
		Personalizations: []sgPersonalization{p},
		From:             sgEmail{Email: msg.From},
		Subject:          msg.Subject,
	}

	if msg.TextBody != "" {
		reqBody.Content = append(reqBody.Content, sgContent{
			Type:  "text/plain",
			Value: msg.TextBody,
		})
	}
	if msg.HTMLBody != "" {
		reqBody.Content = append(reqBody.Content, sgContent{
			Type:  "text/html",
			Value: msg.HTMLBody,
		})
	}

	for _, att := range msg.Attachments {
		cType := att.ContentType
		if cType == "" {
			cType = "application/octet-stream"
		}
		reqBody.Attachments = append(reqBody.Attachments, sgAttachment{
			Content:  base64.StdEncoding.EncodeToString(att.Content),
			Type:     cType,
			Filename: att.Filename,
		})
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("mail: serialize sendgrid payload: %v", err))
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, d.endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Authorization", "Bearer "+d.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	res, err := d.client.Do(httpReq)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("mail: sendgrid request failed: %v", err))
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return errs.FailureError(fmt.Sprintf("mail: sendgrid api error (status %d): %s", res.StatusCode, string(body)))
	}

	return nil
}

var _ Driver = (*SendGridDriver)(nil)
