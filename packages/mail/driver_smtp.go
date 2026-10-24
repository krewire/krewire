package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/krewire/krewire/packages/errs"
)

// SMTPEncryption specifies TLS mode for SMTP connections.
type SMTPEncryption string

const (
	SMTPEncryptionNone     SMTPEncryption = "none"
	SMTPEncryptionSTARTTLS SMTPEncryption = "starttls"
	SMTPEncryptionTLS      SMTPEncryption = "tls"
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
	cfg SMTPConfig
}

// NewSMTPDriver creates an SMTP driver.
func NewSMTPDriver(cfg SMTPConfig) *SMTPDriver {
	if cfg.Port <= 0 {
		if cfg.Encryption == SMTPEncryptionTLS {
			cfg.Port = 465
		} else {
			cfg.Port = 587
		}
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &SMTPDriver{cfg: cfg}
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

	addr := fmt.Sprintf("%s:%d", d.cfg.Host, d.cfg.Port)

	// Combine recipients: To + Cc + Bcc
	var recipients []string
	recipients = append(recipients, msg.To...)
	recipients = append(recipients, msg.Cc...)
	recipients = append(recipients, msg.Bcc...)

	tlsConfig := &tls.Config{
		ServerName:         d.cfg.Host,
		InsecureSkipVerify: d.cfg.InsecureSkipVerify,
	}

	dialer := &net.Dialer{Timeout: d.cfg.Timeout}
	var conn net.Conn

	if d.cfg.Encryption == SMTPEncryptionTLS {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return errs.FailureError(fmt.Sprintf("mail: connect to smtp %s failed: %v", addr, err))
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, d.cfg.Host)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("mail: smtp handshake failed: %v", err))
	}
	defer client.Quit()

	if d.cfg.Encryption == SMTPEncryptionSTARTTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(tlsConfig); err != nil {
				return errs.FailureError(fmt.Sprintf("mail: starttls failed: %v", err))
			}
		}
	}

	if d.cfg.Username != "" && d.cfg.Password != "" {
		auth := smtp.PlainAuth("", d.cfg.Username, d.cfg.Password, d.cfg.Host)
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
