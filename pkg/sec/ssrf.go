package sec

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
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
	if isBlockedHostname(host) {
		return fmt.Errorf("%w: blocked hostname %q", ErrOutboundHostDenied, host)
	}

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

func isBlockedHostname(host string) bool {
	h := strings.ToLower(strings.TrimSuffix(host, "."))
	return h == "localhost" || strings.HasSuffix(h, ".localhost") ||
		h == "metadata.google.internal" || h == "instance-data"
}

func isPrivateAddress(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
		if ip4[0] == 127 || ip4[0] == 0 { // Loopback 127.0.0.0/8 & Current net 0.0.0.0/8
			return true
		}
		if ip4[0] == 169 && ip4[1] == 254 { // Link-local & cloud metadata 169.254.0.0/16
			return true
		}
		if ip4[0] == 100 && (ip4[1]&0xC0) == 64 { // Carrier-grade NAT 100.64.0.0/10
			return true
		}
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast()
}

// SafeTransportOptions tunes outbound HTTP transport security.
type SafeTransportOptions struct {
	Timeout        time.Duration
	AllowedHosts   []string
	AllowPrivateIP bool
}

// SafeTransportOption is a functional option for SafeTransportOptions.
type SafeTransportOption func(*SafeTransportOptions)

// WithAllowedHosts restricts outbound connections strictly to allowed domains/hosts.
func WithAllowedHosts(hosts ...string) SafeTransportOption {
	return func(o *SafeTransportOptions) {
		o.AllowedHosts = append(o.AllowedHosts, hosts...)
	}
}

// WithDialTimeout sets the network dial timeout for outbound connections.
func WithDialTimeout(d time.Duration) SafeTransportOption {
	return func(o *SafeTransportOptions) {
		o.Timeout = d
	}
}

// NewSafeTransport returns an http.RoundTripper that guards against SSRF
// and DNS rebinding attacks by validating every resolved IP address before dialing.
func NewSafeTransport(opts ...SafeTransportOption) *http.Transport {
	cfg := &SafeTransportOptions{
		Timeout: 10 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	dialer := &net.Dialer{
		Timeout:   cfg.Timeout,
		KeepAlive: 30 * time.Second,
	}

	return &http.Transport{
		Proxy:                 nil, // no ambient proxy hijacking by default
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid address %q", ErrOutboundURLInvalid, addr)
			}
			cleanHost := strings.ToLower(strings.TrimSuffix(host, "."))

			if isBlockedHostname(cleanHost) {
				return nil, fmt.Errorf("%w: blocked host %q", ErrOutboundHostDenied, cleanHost)
			}

			if len(cfg.AllowedHosts) > 0 {
				matched := false
				for _, h := range cfg.AllowedHosts {
					if strings.EqualFold(strings.TrimSuffix(h, "."), cleanHost) {
						matched = true
						break
					}
				}
				if !matched {
					return nil, fmt.Errorf("%w: host %q not in allowed list", ErrOutboundHostDenied, cleanHost)
				}
			}

			ips, err := net.DefaultResolver.LookupIP(ctx, "ip", cleanHost)
			if err != nil {
				return nil, err
			}
			if len(ips) == 0 {
				return nil, fmt.Errorf("%w: no IP addresses resolved for %q", ErrOutboundHostDenied, cleanHost)
			}

			// Validate all resolved IPs
			if !cfg.AllowPrivateIP {
				for _, ip := range ips {
					if isPrivateAddress(ip) {
						return nil, fmt.Errorf("%w: host %q resolved to private address %s", ErrOutboundHostDenied, cleanHost, ip.String())
					}
				}
			}

			// Dial the first valid resolved IP directly to prevent DNS rebinding TOCTOU
			targetAddr := net.JoinHostPort(ips[0].String(), port)
			return dialer.DialContext(ctx, network, targetAddr)
		},
	}
}

// NewSafeHTTPClient returns an http.Client backed by NewSafeTransport.
func NewSafeHTTPClient(opts ...SafeTransportOption) *http.Client {
	return &http.Client{
		Timeout:   15 * time.Second,
		Transport: NewSafeTransport(opts...),
	}
}
