package mail_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/mail"
)

func TestMessage_ValidationAndFluentBuilder(t *testing.T) {
	msg := mail.NewMessage().
		SetFrom("sender@krewire.com").
		AddTo("recipient1@example.com", "recipient2@example.com").
		SetSubject("Welcome to Krewire Cloud").
		SetText("Hello plain text").
		SetHTML("<h1>Hello HTML</h1>").
		Attach("invoice.pdf", "application/pdf", []byte("%PDF-1.4 dummy"))

	if err := msg.Validate(); err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	if len(msg.To) != 2 {
		t.Errorf("expected 2 recipients, got %d", len(msg.To))
	}
	if len(msg.Attachments) != 1 {
		t.Errorf("expected 1 attachment, got %d", len(msg.Attachments))
	}

	// Missing From must fail validation
	invalidMsg := mail.NewMessage().AddTo("test@example.com").SetSubject("Hi")
	if err := invalidMsg.Validate(); err == nil {
		t.Error("expected validation error on missing From, got nil")
	}
}

func TestBuildMIME(t *testing.T) {
	msg := mail.NewMessage().
		SetFrom("support@krewire.com").
		AddTo("dev@example.com").
		SetSubject("Activation Code").
		SetText("Code: 123456").
		SetHTML("<p>Code: <strong>123456</strong></p>").
		Attach("notes.txt", "text/plain", []byte("sample notes"))

	mimeBytes, err := mail.BuildMIME(msg)
	if err != nil {
		t.Fatalf("BuildMIME failed: %v", err)
	}

	content := string(mimeBytes)
	if !strings.Contains(content, "From: support@krewire.com") {
		t.Error("missing From header in MIME")
	}
	if !strings.Contains(content, "Subject: Activation Code") {
		t.Error("missing Subject header in MIME")
	}
	if !strings.Contains(content, "multipart/mixed") {
		t.Error("missing multipart/mixed header in MIME with attachment")
	}
	if !strings.Contains(content, "filename=\"notes.txt\"") {
		t.Error("missing attachment filename in MIME")
	}
}

func TestMemoryDriver_Operations(t *testing.T) {
	driver := mail.NewMemoryDriver()
	mailer := mail.NewMailer(driver)
	defer mailer.Close()

	ctx := context.Background()

	msg := mail.NewMessage().
		SetFrom("admin@krewire.com").
		AddTo("user@example.com").
		SetSubject("Hello Memory").
		SetText("Body content")

	err := mailer.Send(ctx, msg)
	if err != nil {
		t.Fatalf("Mailer.Send failed: %v", err)
	}

	if driver.Count() != 1 {
		t.Errorf("expected 1 captured message, got %d", driver.Count())
	}
	if driver.LastMessage().Subject != "Hello Memory" {
		t.Errorf("unexpected subject: %s", driver.LastMessage().Subject)
	}
}

func TestResendDriver_MockServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer re_test_key_123" {
			t.Errorf("unexpected auth header: %s", auth)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "resend_email_id_100"}`))
	}))
	defer ts.Close()

	driver, err := mail.NewResendDriver(mail.ResendConfig{
		APIKey:     "re_test_key_123",
		Endpoint:   ts.URL,
		HTTPClient: ts.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}

	mailer := mail.NewMailer(driver)
	err = mailer.Send(context.Background(), mail.NewMessage().
		SetFrom("noreply@krewire.com").
		AddTo("client@example.com").
		SetSubject("Resend Test").
		SetHTML("<b>Sent via Resend</b>"))
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
}

func TestSendGridDriver_MockServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer SG.test_key_xyz" {
			t.Errorf("unexpected auth header: %s", auth)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer ts.Close()

	driver, err := mail.NewSendGridDriver(mail.SendGridConfig{
		APIKey:     "SG.test_key_xyz",
		Endpoint:   ts.URL,
		HTTPClient: ts.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}

	mailer := mail.NewMailer(driver)
	err = mailer.Send(context.Background(), mail.NewMessage().
		SetFrom("support@krewire.com").
		AddTo("dev@example.com").
		SetSubject("SendGrid Test").
		SetText("Text content"))
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
}

func TestPostmarkDriver_MockServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Postmark-Server-Token")
		if token != "postmark-token-789" {
			t.Errorf("unexpected postmark token: %s", token)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ErrorCode": 0, "Message": "OK"}`))
	}))
	defer ts.Close()

	driver, err := mail.NewPostmarkDriver(mail.PostmarkConfig{
		ServerToken: "postmark-token-789",
		Endpoint:    ts.URL,
		HTTPClient:  ts.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}

	mailer := mail.NewMailer(driver)
	err = mailer.Send(context.Background(), mail.NewMessage().
		SetFrom("alerts@krewire.com").
		AddTo("ops@example.com").
		SetSubject("System Alert").
		SetText("High CPU load alert"))
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
}
