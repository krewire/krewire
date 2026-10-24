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

// PostmarkConfig contains options for Postmark transactional email delivery.
type PostmarkConfig struct {
	ServerToken string
	Endpoint    string // default: "https://api.postmarkapp.com/email"
	HTTPClient  *http.Client
}

// PostmarkDriver delivers email via Postmark.
type PostmarkDriver struct {
	serverToken string
	endpoint    string
	client      *http.Client
}

// NewPostmarkDriver creates a Postmark delivery driver.
func NewPostmarkDriver(cfg PostmarkConfig) (*PostmarkDriver, error) {
	if strings.TrimSpace(cfg.ServerToken) == "" {
		return nil, errs.UsageError("mail: postmark server_token is required")
	}

	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = "https://api.postmarkapp.com/email"
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	return &PostmarkDriver{
		serverToken: cfg.ServerToken,
		endpoint:    endpoint,
		client:      client,
	}, nil
}

func (d *PostmarkDriver) Name() string {
	return "postmark"
}

func (d *PostmarkDriver) Close() error {
	return nil
}

type postmarkAttachment struct {
	Name        string `json:"Name"`
	Content     string `json:"Content"` // base64
	ContentType string `json:"ContentType"`
}

type postmarkRequest struct {
	From        string               `json:"From"`
	To          string               `json:"To"`
	Cc          string               `json:"Cc,omitempty"`
	Bcc         string               `json:"Bcc,omitempty"`
	Subject     string               `json:"Subject"`
	TextBody    string               `json:"TextBody,omitempty"`
	HtmlBody    string               `json:"HtmlBody,omitempty"`
	Attachments []postmarkAttachment `json:"Attachments,omitempty"`
}

func (d *PostmarkDriver) Send(ctx context.Context, msg *Message) error {
	reqBody := postmarkRequest{
		From:     msg.From,
		To:       strings.Join(msg.To, ", "),
		Cc:       strings.Join(msg.Cc, ", "),
		Bcc:      strings.Join(msg.Bcc, ", "),
		Subject:  msg.Subject,
		TextBody: msg.TextBody,
		HtmlBody: msg.HTMLBody,
	}

	for _, att := range msg.Attachments {
		cType := att.ContentType
		if cType == "" {
			cType = "application/octet-stream"
		}
		reqBody.Attachments = append(reqBody.Attachments, postmarkAttachment{
			Name:        att.Filename,
			Content:     base64.StdEncoding.EncodeToString(att.Content),
			ContentType: cType,
		})
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("mail: serialize postmark payload: %v", err))
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, d.endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}

	httpReq.Header.Set("X-Postmark-Server-Token", d.serverToken)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	res, err := d.client.Do(httpReq)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("mail: postmark request failed: %v", err))
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return errs.FailureError(fmt.Sprintf("mail: postmark api error (status %d): %s", res.StatusCode, string(body)))
	}

	return nil
}

var _ Driver = (*PostmarkDriver)(nil)
