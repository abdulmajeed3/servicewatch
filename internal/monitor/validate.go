package monitor

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// Validate checks a CreateRequest for correctness.
func (r *CreateRequest) Validate() error {
	return validateFields(r.Name, r.URL, r.Notes, r.CheckInterval, r.RequestTimeout, r.ExpectedStatus)
}

// Validate checks an UpdateRequest for correctness.
func (r *UpdateRequest) Validate() error {
	return validateFields(r.Name, r.URL, r.Notes, r.CheckInterval, r.RequestTimeout, r.ExpectedStatus)
}

func validateFields(name, rawURL, notes string, interval, timeout, expectedStatus int) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if len(name) > 200 {
		return fmt.Errorf("%w: name must be at most 200 characters", ErrInvalidInput)
	}

	if len(notes) > 500 {
		return fmt.Errorf("%w: notes must be at most 500 characters", ErrInvalidInput)
	}

	if err := validateURL(rawURL); err != nil {
		return err
	}

	if !isAllowedInterval(interval) {
		return fmt.Errorf("%w: check_interval must be one of %v", ErrInvalidInput, AllowedIntervals)
	}

	if timeout < 5 || timeout > 120 {
		return fmt.Errorf("%w: request_timeout must be between 5 and 120 seconds", ErrInvalidInput)
	}

	if expectedStatus != 0 && (expectedStatus < 100 || expectedStatus > 599) {
		return fmt.Errorf("%w: expected_status must be a valid HTTP status code or 0 for any 2xx", ErrInvalidInput)
	}

	return nil
}

// validateURL checks that a URL is safe and well-formed.
// It returns ErrInvalidInput for validation failures.
func validateURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("%w: url is required", ErrInvalidInput)
	}
	if len(rawURL) > 2048 {
		return fmt.Errorf("%w: url is too long", ErrInvalidInput)
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: url is malformed: %v", ErrInvalidInput, err)
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("%w: url must use http or https scheme", ErrInvalidInput)
	}

	if u.Host == "" {
		return fmt.Errorf("%w: url must have a host", ErrInvalidInput)
	}

	// Reject embedded credentials — avoid forwarding auth to monitored services.
	if u.User != nil {
		return fmt.Errorf("%w: url must not contain embedded credentials", ErrInvalidInput)
	}

	// Reject fragment identifiers — not meaningful for HTTP monitoring.
	if u.Fragment != "" {
		return fmt.Errorf("%w: url must not contain a fragment", ErrInvalidInput)
	}

	// Additional hostname-level checks; runtime SSRF guard handles IP resolution.
	host := u.Hostname()
	if isObviouslyLocalhost(host) {
		return errors.New("url must not target localhost or loopback addresses; use MONITOR_ALLOW_PRIVATE_IPS=true for local development only")
	}

	return nil
}

func isObviouslyLocalhost(host string) bool {
	lower := strings.ToLower(host)
	if lower == "localhost" {
		return true
	}
	// Check if the hostname is a bare IP address (loopback/link-local/private).
	// Full IP range blocking is done at dial time by the SSRF transport;
	// this catches the obvious cases early.
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
	}
	return false
}

func isAllowedInterval(seconds int) bool {
	for _, a := range AllowedIntervals {
		if a == seconds {
			return true
		}
	}
	return false
}
