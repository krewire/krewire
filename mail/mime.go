package mail

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"strings"
	"time"
)

// BuildMIME creates an RFC 5322 multipart MIME email message byte stream.
func BuildMIME(msg *Message) ([]byte, error) {
	var buf bytes.Buffer

	// Headers
	buf.WriteString(fmt.Sprintf("From: %s\r\n", msg.From))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))
	if len(msg.Cc) > 0 {
		buf.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(msg.Cc, ", ")))
	}
	if msg.ReplyTo != "" {
		buf.WriteString(fmt.Sprintf("Reply-To: %s\r\n", msg.ReplyTo))
	}
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	buf.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	buf.WriteString(fmt.Sprintf("Message-ID: <%s@krewire.mail>\r\n", randomID()))

	for k, v := range msg.Headers {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}

	buf.WriteString("MIME-Version: 1.0\r\n")

	// If no attachments, simple or multipart/alternative
	if len(msg.Attachments) == 0 {
		if msg.HTMLBody != "" && msg.TextBody != "" {
			altWriter := multipart.NewWriter(&buf)
			buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", altWriter.Boundary()))

			// Plain text part
			textHeader := make(textproto.MIMEHeader)
			textHeader.Set("Content-Type", "text/plain; charset=UTF-8")
			textPart, err := altWriter.CreatePart(textHeader)
			if err != nil {
				return nil, err
			}
			_, _ = textPart.Write([]byte(msg.TextBody))

			// HTML part
			htmlHeader := make(textproto.MIMEHeader)
			htmlHeader.Set("Content-Type", "text/html; charset=UTF-8")
			htmlPart, err := altWriter.CreatePart(htmlHeader)
			if err != nil {
				return nil, err
			}
			_, _ = htmlPart.Write([]byte(msg.HTMLBody))

			_ = altWriter.Close()
			return buf.Bytes(), nil
		}

		if msg.HTMLBody != "" {
			buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
			buf.WriteString(msg.HTMLBody)
			return buf.Bytes(), nil
		}

		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		buf.WriteString(msg.TextBody)
		return buf.Bytes(), nil
	}

	// Multipart mixed (body + attachments)
	mixedWriter := multipart.NewWriter(&buf)
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", mixedWriter.Boundary()))

	// Body part
	bodyHeader := make(textproto.MIMEHeader)
	if msg.HTMLBody != "" {
		bodyHeader.Set("Content-Type", "text/html; charset=UTF-8")
		bodyPart, err := mixedWriter.CreatePart(bodyHeader)
		if err != nil {
			return nil, err
		}
		_, _ = bodyPart.Write([]byte(msg.HTMLBody))
	} else {
		bodyHeader.Set("Content-Type", "text/plain; charset=UTF-8")
		bodyPart, err := mixedWriter.CreatePart(bodyHeader)
		if err != nil {
			return nil, err
		}
		_, _ = bodyPart.Write([]byte(msg.TextBody))
	}

	// Attachments
	for _, att := range msg.Attachments {
		attHeader := make(textproto.MIMEHeader)
		cType := att.ContentType
		if cType == "" {
			cType = "application/octet-stream"
		}
		attHeader.Set("Content-Type", fmt.Sprintf("%s; name=\"%s\"", cType, att.Filename))
		attHeader.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", att.Filename))
		attHeader.Set("Content-Transfer-Encoding", "base64")

		part, err := mixedWriter.CreatePart(attHeader)
		if err != nil {
			return nil, err
		}

		encoded := base64.StdEncoding.EncodeToString(att.Content)
		// Split lines into 76 chars
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			_, _ = part.Write([]byte(encoded[i:end] + "\r\n"))
		}
	}

	_ = mixedWriter.Close()
	return buf.Bytes(), nil
}

func randomID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
