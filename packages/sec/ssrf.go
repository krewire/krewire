package sec

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

var (
	ErrOutboundURLInvalid = errors.New("sec: invalid outbound URL")
	ErrOutboundHostDenied = errors.New("sec: outbound host denied")
)

// ValidateOutboundURL validates a URL before an application performs an
// outbound request. The host must be explicitly listed in allowedHosts and
// literal private, loopback, link-local, and unspecified IP addresses are
// rejected. This function does not perform a network request.
func ValidateOutboundURL(raw string, allowedHosts ...string) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil || u.Scheme == "" || u.Host == "" || u.User != nil {
		return fmt.Errorf("%w: malformed URL", ErrOutboundURLInvalid)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: scheme %q is not allowed", ErrOutboundURLInvalid, u.Scheme)
	}
	if u.Hostname() == "" || strings.ContainsAny(u.Hostname(), "\r\n") {
		return fmt.Errorf("%w: invalid host", ErrOutboundURLInvalid)
	}

	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	allowed := false
	for _, candidate := range allowedHosts {
		candidate = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(candidate), "."))
		if candidate != "" && host == candidate {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("%w: %q is not allowlisted", ErrOutboundHostDenied, host)
	}

	if ip := net.ParseIP(host); ip != nil && isPrivateAddress(ip) {
		return fmt.Errorf("%w: private address %q", ErrOutboundHostDenied, host)
	}
	return nil
}

func isPrivateAddress(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}
