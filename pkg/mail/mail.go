package mail

import (
	"context"
	"strings"

	"github.com/krewire/krewire/pkg/errs"
)

// Attachment represents a file or inline media included with an email.
type Attachment struct {
	Filename    string
	ContentType string
	Content     []byte
	Inline      bool
	ContentID   string
}

// Message represents an email to be sent.
type Message struct {
	From        string
	To          []string
	Cc          []string
	Bcc         []string
	ReplyTo     string
	Subject     string
	TextBody    string
	HTMLBody    string
	Attachments []Attachment
	Headers     map[string]string
}

// NewMessage initializes an empty Message with a fluent builder.
func NewMessage() *Message {
	return &Message{
		Headers: make(map[string]string),
	}
}

func (m *Message) SetFrom(from string) *Message {
	m.From = strings.TrimSpace(from)
	return m
}

func (m *Message) AddTo(to ...string) *Message {
	for _, addr := range to {
		if a := strings.TrimSpace(addr); a != "" {
			m.To = append(m.To, a)
		}
	}
	return m
}

func (m *Message) AddCc(cc ...string) *Message {
	for _, addr := range cc {
		if a := strings.TrimSpace(addr); a != "" {
			m.Cc = append(m.Cc, a)
		}
	}
	return m
}

func (m *Message) AddBcc(bcc ...string) *Message {
	for _, addr := range bcc {
		if a := strings.TrimSpace(addr); a != "" {
			m.Bcc = append(m.Bcc, a)
		}
	}
	return m
}

func (m *Message) SetReplyTo(replyTo string) *Message {
	m.ReplyTo = strings.TrimSpace(replyTo)
	return m
}

func (m *Message) SetSubject(subject string) *Message {
	m.Subject = strings.TrimSpace(subject)
	return m
}

func (m *Message) SetText(text string) *Message {
	m.TextBody = text
	return m
}

func (m *Message) SetHTML(html string) *Message {
	m.HTMLBody = html
	return m
}

func (m *Message) Attach(filename, contentType string, content []byte) *Message {
	m.Attachments = append(m.Attachments, Attachment{
		Filename:    filename,
		ContentType: contentType,
		Content:     content,
	})
	return m
}

func (m *Message) SetHeader(key, val string) *Message {
	if m.Headers == nil {
		m.Headers = make(map[string]string)
	}
	m.Headers[key] = val
	return m
}

// Validate checks that mandatory fields (From, To, Subject, Body) are populated.
func (m *Message) Validate() error {
	if m.From == "" {
		return errs.UsageError("mail: 'from' address is required")
	}
	if len(m.To) == 0 {
		return errs.UsageError("mail: at least one 'to' recipient is required")
	}
	if m.Subject == "" {
		return errs.UsageError("mail: subject is required")
	}
	if m.TextBody == "" && m.HTMLBody == "" {
		return errs.UsageError("mail: either text or html body must be provided")
	}
	return nil
}

// Driver defines the interface for all email delivery backends.
type Driver interface {
	Name() string
	Send(ctx context.Context, msg *Message) error
	Close() error
}

// Mailer manages an email delivery driver.
type Mailer struct {
	driver Driver
}

// NewMailer creates a Mailer with the chosen Driver.
func NewMailer(driver Driver) *Mailer {
	return &Mailer{driver: driver}
}

// Send validates and dispatches a message using the active driver.
func (m *Mailer) Send(ctx context.Context, msg *Message) error {
	if msg == nil {
		return errs.UsageError("mail: message is nil")
	}
	if err := msg.Validate(); err != nil {
		return err
	}
	return m.driver.Send(ctx, msg)
}

// Close closes the underlying mailer driver.
func (m *Mailer) Close() error {
	if m.driver != nil {
		return m.driver.Close()
	}
	return nil
}
