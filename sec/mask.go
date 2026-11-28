// Package sec provides security primitives for the Krewire ecosystem.
//
// This file implements the real-time stream masking writer (MaskingWriter)
// described in the 2027 Q1 Phase 0 roadmap (Step 06: Dynamic Secret Hygiene
// & Stream Masking). MaskingWriter wraps an io.Writer and redacts any
// registered secret values before they reach the underlying sink, including
// when a secret is split across write boundaries.
package sec

import (
	"io"
	"strings"
	"sync"
)

// MaskingWriter is an io.Writer that redacts registered secrets from byte
// streams in real time. It is safe for concurrent use.
type MaskingWriter struct {
	mu      sync.Mutex
	dst     io.Writer
	secrets []string
	mask    string

	carry []byte // trailing bytes that may be a prefix of a secret
}

// NewMaskingWriter wraps dst, redacting the given secret values.
func NewMaskingWriter(dst io.Writer, secrets ...string) *MaskingWriter {
	filtered := make([]string, 0, len(secrets))
	for _, s := range secrets {
		if strings.TrimSpace(s) != "" {
			filtered = append(filtered, s)
		}
	}
	return &MaskingWriter{dst: dst, secrets: filtered, mask: "[REDACTED]"}
}

// maxSecretLen returns the length of the longest registered secret.
func (m *MaskingWriter) maxSecretLen() int {
	n := 0
	for _, s := range m.secrets {
		if len(s) > n {
			n = len(s)
		}
	}
	return n
}

// Write implements io.Writer.
func (m *MaskingWriter) Write(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(p) == 0 {
		return 0, nil
	}

	// Combine carry from previous write with new data.
	buf := append(append([]byte{}, m.carry...), p...)
	m.carry = m.carry[:0]

	// 1. Redact all full occurrences of secrets currently in buf.
	buf = m.redact(buf)

	// 2. Check if a trailing suffix of buf could be the prefix of any secret.
	hold := 0
	bufStr := string(buf)
	for _, secret := range m.secrets {
		for k := 1; k < len(secret) && k <= len(buf); k++ {
			if strings.HasSuffix(bufStr, secret[:k]) {
				if k > hold {
					hold = k
				}
			}
		}
	}

	toProcess := buf[:len(buf)-hold]
	if hold > 0 {
		m.carry = append(m.carry, buf[len(buf)-hold:]...)
	}

	if len(toProcess) > 0 {
		if _, err := m.dst.Write(toProcess); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

// Flush writes any buffered carry bytes (redacted) to the underlying writer.
func (m *MaskingWriter) Flush() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.carry) == 0 {
		return nil
	}
	out := m.redact(m.carry)
	m.carry = m.carry[:0]
	_, err := m.dst.Write(out)
	return err
}

func (m *MaskingWriter) redact(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	s := string(data)
	for _, secret := range m.secrets {
		s = strings.ReplaceAll(s, secret, m.mask)
	}
	return []byte(s)
}
