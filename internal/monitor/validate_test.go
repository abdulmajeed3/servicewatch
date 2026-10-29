package monitor_test

import (
	"errors"
	"testing"

	"github.com/amsoft/servicewatch/internal/monitor"
)

func TestCreateRequest_Validate(t *testing.T) {
	valid := monitor.CreateRequest{
		Name:           "My Service",
		URL:            "https://example.com",
		CheckInterval:  300,
		RequestTimeout: 30,
		ExpectedStatus: 0,
	}

	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid request to pass, got: %v", err)
	}
}

func TestValidate_MissingName(t *testing.T) {
	req := monitor.CreateRequest{
		Name:           "",
		URL:            "https://example.com",
		CheckInterval:  300,
		RequestTimeout: 30,
	}
	if err := req.Validate(); !errors.Is(err, monitor.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for missing name, got: %v", err)
	}
}

func TestValidate_BadURL_Scheme(t *testing.T) {
	cases := []string{
		"ftp://example.com",
		"file:///etc/passwd",
		"javascript:alert(1)",
		"//example.com",
		"example.com",
	}
	for _, url := range cases {
		req := monitor.CreateRequest{Name: "X", URL: url, CheckInterval: 300, RequestTimeout: 30}
		if err := req.Validate(); !errors.Is(err, monitor.ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput for URL %q, got: %v", url, err)
		}
	}
}

func TestValidate_RejectsEmbeddedCredentials(t *testing.T) {
	req := monitor.CreateRequest{
		Name: "X", URL: "https://user:pass@example.com",
		CheckInterval: 300, RequestTimeout: 30,
	}
	if err := req.Validate(); !errors.Is(err, monitor.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for embedded credentials, got: %v", err)
	}
}

func TestValidate_RejectsLocalhost(t *testing.T) {
	for _, url := range []string{
		"http://localhost/path",
		"http://[::1]/path",
	} {
		req := monitor.CreateRequest{Name: "X", URL: url, CheckInterval: 300, RequestTimeout: 30}
		err := req.Validate()
		if err == nil {
			t.Errorf("expected error for localhost URL %q, got nil", url)
		}
	}
}

func TestValidate_AllowedIntervals(t *testing.T) {
	for _, interval := range monitor.AllowedIntervals {
		req := monitor.CreateRequest{
			Name: "X", URL: "https://example.com",
			CheckInterval: interval, RequestTimeout: 30,
		}
		if err := req.Validate(); err != nil {
			t.Errorf("expected valid interval %d to pass, got: %v", interval, err)
		}
	}
}

func TestValidate_DisallowedInterval(t *testing.T) {
	req := monitor.CreateRequest{
		Name: "X", URL: "https://example.com",
		CheckInterval: 999, RequestTimeout: 30,
	}
	if err := req.Validate(); !errors.Is(err, monitor.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for invalid interval, got: %v", err)
	}
}

func TestValidate_TimeoutBounds(t *testing.T) {
	cases := []struct {
		timeout int
		wantErr bool
	}{
		{4, true},
		{5, false},
		{30, false},
		{120, false},
		{121, true},
	}
	for _, tc := range cases {
		req := monitor.CreateRequest{
			Name: "X", URL: "https://example.com",
			CheckInterval: 300, RequestTimeout: tc.timeout,
		}
		err := req.Validate()
		if tc.wantErr && err == nil {
			t.Errorf("timeout %d: expected error, got nil", tc.timeout)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("timeout %d: expected no error, got: %v", tc.timeout, err)
		}
	}
}

func TestValidate_ExpectedStatusBounds(t *testing.T) {
	cases := []struct {
		status  int
		wantErr bool
	}{
		{0, false}, // any 2xx
		{200, false},
		{404, false},
		{599, false},
		{99, true},
		{600, true},
	}
	for _, tc := range cases {
		req := monitor.CreateRequest{
			Name: "X", URL: "https://example.com",
			CheckInterval: 300, RequestTimeout: 30,
			ExpectedStatus: tc.status,
		}
		err := req.Validate()
		if tc.wantErr && err == nil {
			t.Errorf("status %d: expected error, got nil", tc.status)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("status %d: expected no error, got: %v", tc.status, err)
		}
	}
}

func TestValidate_ValidHTTPAndHTTPS(t *testing.T) {
	for _, url := range []string{
		"http://example.com",
		"https://example.com",
		"https://example.com/path?query=1",
		"http://example.com:8080/api/health",
	} {
		req := monitor.CreateRequest{
			Name: "X", URL: url,
			CheckInterval: 300, RequestTimeout: 30,
		}
		if err := req.Validate(); err != nil {
			t.Errorf("URL %q: expected valid, got: %v", url, err)
		}
	}
}
