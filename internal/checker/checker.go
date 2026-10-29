package checker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// Result contains the outcome of a single monitor check.
type Result struct {
	CheckedAt     time.Time
	IsSuccess     bool
	StatusCode    int
	ResponseMs    int
	FailureReason string
}

// Checker executes outbound HTTP checks against monitored endpoints.
type Checker struct {
	client          *http.Client
	maxResponseBody int64
}

// New creates a Checker with an SSRF-safe transport.
func New(requestTimeout time.Duration, allowPrivateIPs bool, maxResponseBody int64) *Checker {
	transport := NewTransport(requestTimeout, allowPrivateIPs)
	return &Checker{
		client: &http.Client{
			Transport: transport,
			// Redirect policy: disabled by default.
			// Following redirects would require re-validating each destination URL,
			// which is complex and error-prone. Treat unexpected redirects as failures.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Timeout: requestTimeout,
		},
		maxResponseBody: maxResponseBody,
	}
}

// Check performs a single HTTP GET against the given URL and returns the result.
// expectedStatus: 0 means any 2xx is success. Any other value requires an exact match.
func (c *Checker) Check(ctx context.Context, rawURL string, expectedStatus int, timeout time.Duration) Result {
	start := time.Now()
	result := Result{CheckedAt: start}

	// Apply per-monitor timeout.
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		result.FailureReason = fmt.Sprintf("build request: %v", err)
		return result
	}
	req.Header.Set("User-Agent", "ServiceWatch/1.0 (+https://github.com/amsoft/servicewatch)")

	resp, err := c.client.Do(req)
	result.ResponseMs = int(time.Since(start).Milliseconds())

	if err != nil {
		result.FailureReason = classifyError(err)
		return result
	}
	defer resp.Body.Close()

	// Drain up to maxResponseBody to allow connection reuse and avoid resource leaks.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, c.maxResponseBody))

	result.StatusCode = resp.StatusCode

	if isSuccessStatus(resp.StatusCode, expectedStatus) {
		result.IsSuccess = true
	} else {
		result.FailureReason = fmt.Sprintf("unexpected status %d", resp.StatusCode)
	}

	return result
}

func isSuccessStatus(statusCode, expectedStatus int) bool {
	if expectedStatus == 0 {
		return statusCode >= 200 && statusCode < 300
	}
	return statusCode == expectedStatus
}

func classifyError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) || isTimeoutErr(err) {
		return "timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "cancelled"
	}
	if errors.Is(err, ErrBlockedAddress) {
		return "blocked: SSRF protection"
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		if dnsErr.Temporary() {
			return "DNS error (temporary)"
		}
		return "DNS error (host not found)"
	}

	msg := err.Error()
	if containsAnyStr(msg, "tls", "certificate", "x509") {
		return "TLS error"
	}
	if containsAnyStr(msg, "connection refused", "connection reset") {
		return "connection refused"
	}
	if containsAnyStr(msg, "no route to host", "network unreachable") {
		return "network unreachable"
	}
	if containsAnyStr(msg, "blocked", "ssrf") {
		return "blocked: SSRF protection"
	}

	return fmt.Sprintf("request failed: %v", err)
}

func isTimeoutErr(err error) bool {
	type timeoutErr interface{ Timeout() bool }
	var t timeoutErr
	if errors.As(err, &t) {
		return t.Timeout()
	}
	return false
}

func containsAnyStr(s string, subs ...string) bool {
	sLow := strToLower(s)
	for _, sub := range subs {
		if strContains(sLow, strToLower(sub)) {
			return true
		}
	}
	return false
}

func strToLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		b[i] = c
	}
	return string(b)
}

func strContains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
