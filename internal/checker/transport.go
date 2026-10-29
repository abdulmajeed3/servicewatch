// Package checker provides SSRF-safe outbound HTTP checking.
//
// SSRF Mitigation Strategy
// ─────────────────────────
// User-supplied URLs are validated at creation time (scheme, no credentials,
// no fragments), but the real defense happens at the network level:
//
//  1. Custom dialer: we resolve the hostname ourselves, then inspect every
//     resolved IP address before opening a TCP connection. If any address is
//     private/loopback/link-local/multicast/reserved, we reject the whole dial.
//
//  2. Redirects are disabled by default. The caller decides whether to follow
//     them; when enabled, each redirect destination is re-validated through
//     the same custom dialer.
//
//  3. Timeouts and response-body limits are always applied to prevent resource
//     exhaustion.
//
//  4. We never forward Authorization headers to monitored endpoints.
//
// Limitations
// ───────────
//   - DNS rebinding: an attacker may control a DNS record that rotates between
//     a public IP (passes validation) and an internal IP after the check. The
//     controlled-dial approach mitigates this because we only connect to the
//     address we validated, but does not eliminate it entirely. Network-level
//     egress controls (firewall, VPC policy) provide the strongest guarantee.
//   - Cloud metadata endpoints (169.254.169.254, fd00:ec2::254 etc.) are
//     blocked by the private-IP check as link-local / RFC-1918 addresses.
//   - AllowPrivateIPs=true disables the IP check and is intended for local
//     development only. It must never be set in production.
package checker

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// ErrBlockedAddress is returned when a resolved IP is not permitted.
var ErrBlockedAddress = errors.New("destination address is not permitted (SSRF protection)")

// ssrfDialer wraps net.Dialer to validate resolved IP addresses before connecting.
type ssrfDialer struct {
	inner           net.Dialer
	allowPrivateIPs bool
}

func (d *ssrfDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("split host/port %q: %w", addr, err)
	}

	addrs, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve %q: %w", host, err)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no addresses resolved for %q", host)
	}

	if !d.allowPrivateIPs {
		for _, raw := range addrs {
			ip := net.ParseIP(raw)
			if ip == nil {
				return nil, fmt.Errorf("could not parse resolved address %q", raw)
			}
			if isBlockedIP(ip) {
				return nil, fmt.Errorf("%w: %s resolves to %s", ErrBlockedAddress, host, ip)
			}
		}
	}

	// Connect only to the first validated address; avoids a TOCTOU window that
	// a different resolution could exploit.
	target := net.JoinHostPort(addrs[0], port)
	return d.inner.DialContext(ctx, network, target)
}

// isBlockedIP returns true if the IP must not be contacted.
func isBlockedIP(ip net.IP) bool {
	// Loopback: 127.0.0.0/8, ::1
	if ip.IsLoopback() {
		return true
	}
	// Link-local: 169.254.0.0/16 (includes AWS metadata 169.254.169.254), fe80::/10
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	// Private: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, fc00::/7
	if ip.IsPrivate() {
		return true
	}
	// Unspecified: 0.0.0.0, ::
	if ip.IsUnspecified() {
		return true
	}
	// Multicast
	if ip.IsMulticast() {
		return true
	}
	return false
}

// NewTransport builds an http.Transport that blocks SSRF destinations.
func NewTransport(requestTimeout time.Duration, allowPrivateIPs bool) *http.Transport {
	dialer := &ssrfDialer{
		inner: net.Dialer{
			Timeout:   requestTimeout,
			KeepAlive: 0, // no keep-alive for one-shot monitoring requests
		},
		allowPrivateIPs: allowPrivateIPs,
	}

	return &http.Transport{
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: requestTimeout,
		ExpectContinueTimeout: 1 * time.Second,
		// DisableKeepAlives saves resources for short-lived check connections.
		DisableKeepAlives: true,
		// No proxy — do not allow proxy environment variables to bypass SSRF checks.
		Proxy: nil,
	}
}
