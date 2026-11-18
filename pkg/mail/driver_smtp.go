package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/krewire/krewire/pkg/errs"
)

// SMTPEncryption specifies TLS mode for SMTP connections.
type SMTPEncryption string

const (
	SMTPEncryptionNone     SMTPEncryption = "none"
	SMTPEncryptionSTARTTLS SMTPEncryption = "starttls"
	SMTPEncryptionTLS      SMTPEncryption = "tls"

	// DefaultSMTPPortSTARTTLS is the standard submission port with STARTTLS.
	DefaultSMTPPortSTARTTLS = 587

	// DefaultSMTPPortTLS is the standard SMTPS port with explicit TLS.
	DefaultSMTPPortTLS = 465

	// DefaultSMTPPortPlain is the standard unencrypted SMTP relay port.
	DefaultSMTPPortPlain = 25

	// DefaultSMTPTimeout is the default network I/O timeout for SMTP sessions.
	DefaultSMTPTimeout = 30 * time.Second
)

// SMTPConfig contains connection and authentication options for SMTP servers.
type SMTPConfig struct {
	Host               string // e.g. "smtp.gmail.com", "smtp.mailgun.org"
	Port               int    // 587 (STARTTLS), 465 (TLS), 25 (plain)
	Username           string
	Password           string
	Encryption         SMTPEncryption
	InsecureSkipVerify bool
	Timeout            time.Duration
}

// SMTPDriver sends email via standard SMTP protocol.
type SMTPDriver struct {
	config SMTPConfig
}

// NewSMTPDriver creates an SMTP driver.
func NewSMTPDriver(config SMTPConfig) *SMTPDriver {
	if config.Port <= 0 {
		if config.Encryption == SMTPEncryptionTLS {
			config.Port = DefaultSMTPPortTLS
		} else {
			config.Port = DefaultSMTPPortSTARTTLS
		}
	}
	if config.Timeout <= 0 {
		config.Timeout = DefaultSMTPTimeout
	}
	return &SMTPDriver{config: config}
}

func (d *SMTPDriver) Name() string {
	return "smtp"
}

func (d *SMTPDriver) Close() error {
	return nil
}

func (d *SMTPDriver) Send(ctx context.Context, msg *Message) error {
	mimeBytes, err := BuildMIME(msg)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("mail: build mime failed: %v", err))
	}

	addr := fmt.Sprintf("%s:%d", d.config.Host, d.config.Port)

	// Combine recipients: To + Cc + Bcc
	var recipients []string
	recipients = append(recipients, msg.To...)
	recipients = append(recipients, msg.Cc...)
	recipients = append(recipients, msg.Bcc...)

	tlsConfig := &tls.Config{
		ServerName:         d.config.Host,
		InsecureSkipVerify: d.config.InsecureSkipVerify,
	}

	dialer := &net.Dialer{Timeout: d.config.Timeout}
	var conn net.Conn

	if d.config.Encryption == SMTPEncryptionTLS {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return errs.FailureError(fmt.Sprintf("mail: connect to smtp %s failed: %v", addr, err))
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, d.config.Host)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("mail: smtp handshake failed: %v", err))
	}
	defer client.Quit()

	if d.config.Encryption == SMTPEncryptionSTARTTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(tlsConfig); err != nil {
				return errs.FailureError(fmt.Sprintf("mail: starttls failed: %v", err))
			}
		}
	}

	if d.config.Username != "" && d.config.Password != "" {
		auth := smtp.PlainAuth("", d.config.Username, d.config.Password, d.config.Host)
		if err := client.Auth(auth); err != nil {
			return errs.FailureError(fmt.Sprintf("mail: smtp auth failed: %v", err))
		}
	}

	// MAIL FROM
	cleanFrom := extractEmailAddress(msg.From)
	if err := client.Mail(cleanFrom); err != nil {
		return errs.FailureError(fmt.Sprintf("mail: smtp MAIL FROM %s failed: %v", cleanFrom, err))
	}

	// RCPT TO
	for _, rcpt := range recipients {
		cleanRcpt := extractEmailAddress(rcpt)
		if err := client.Rcpt(cleanRcpt); err != nil {
			return errs.FailureError(fmt.Sprintf("mail: smtp RCPT TO %s failed: %v", cleanRcpt, err))
		}
	}

	// DATA
	w, err := client.Data()
	if err != nil {
		return errs.FailureError(fmt.Sprintf("mail: smtp DATA command failed: %v", err))
	}

	if _, err := w.Write(mimeBytes); err != nil {
		return errs.FailureError(fmt.Sprintf("mail: write mime body failed: %v", err))
	}

	if err := w.Close(); err != nil {
		return errs.FailureError(fmt.Sprintf("mail: finish smtp data failed: %v", err))
	}

	return nil
}

func extractEmailAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	if start := strings.Index(addr, "<"); start >= 0 {
		if end := strings.Index(addr[start:], ">"); end >= 0 {
			return strings.TrimSpace(addr[start+1 : start+end])
		}
	}
	return addr
}

var _ Driver = (*SMTPDriver)(nil)
