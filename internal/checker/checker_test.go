package checker_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/amsoft/servicewatch/internal/checker"
)

func newChecker() *checker.Checker {
	return checker.New(5*time.Second, true, 1<<20) // allowPrivateIPs=true for test servers
}

func TestCheck_Success_2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newChecker()
	result := c.Check(context.Background(), srv.URL, 0, 5*time.Second)

	if !result.IsSuccess {
		t.Fatalf("expected success, got failure: %s", result.FailureReason)
	}
	if result.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", result.StatusCode)
	}
	if result.ResponseMs < 0 {
		t.Errorf("negative response time: %d", result.ResponseMs)
	}
}

func TestCheck_ExactStatusMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newChecker()

	// Expect 201 — should succeed.
	result := c.Check(context.Background(), srv.URL, http.StatusCreated, 5*time.Second)
	if !result.IsSuccess {
		t.Fatalf("expected success for exact match, got: %s", result.FailureReason)
	}

	// Expect 200 — should fail.
	result = c.Check(context.Background(), srv.URL, http.StatusOK, 5*time.Second)
	if result.IsSuccess {
		t.Fatal("expected failure for wrong exact match, got success")
	}
}

func TestCheck_500_FailsDefaultPolicy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newChecker()
	result := c.Check(context.Background(), srv.URL, 0, 5*time.Second)

	if result.IsSuccess {
		t.Fatal("expected failure for 500, got success")
	}
	if result.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", result.StatusCode)
	}
}

func TestCheck_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hang.
		<-r.Context().Done()
	}))
	defer srv.Close()

	c := newChecker()
	result := c.Check(context.Background(), srv.URL, 0, 100*time.Millisecond)

	if result.IsSuccess {
		t.Fatal("expected failure for timeout, got success")
	}
	if result.FailureReason == "" {
		t.Error("expected non-empty failure reason")
	}
}

func TestCheck_InvalidURL(t *testing.T) {
	c := newChecker()
	result := c.Check(context.Background(), "not-a-url://bad", 0, 5*time.Second)

	if result.IsSuccess {
		t.Fatal("expected failure for invalid URL")
	}
}

func TestCheck_Redirect_NotFollowed(t *testing.T) {
	// Redirect target.
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	// Redirect source.
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer src.Close()

	c := newChecker()
	result := c.Check(context.Background(), src.URL, 0, 5*time.Second)

	// 302 is not 2xx, so it should fail with any-2xx policy.
	if result.IsSuccess {
		t.Fatal("expected failure: redirect should not be followed")
	}
	if result.StatusCode != http.StatusFound {
		t.Errorf("expected 302, got %d", result.StatusCode)
	}
}

func TestCheck_UserAgentHeader(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newChecker()
	_ = c.Check(context.Background(), srv.URL, 0, 5*time.Second)

	if gotUA == "" {
		t.Error("expected User-Agent header to be set")
	}
}

func TestSSRFTransport_BlocksLoopback(t *testing.T) {
	// Use allowPrivateIPs=false for SSRF checks.
	c := checker.New(5*time.Second, false, 1<<20)
	// Attempt to reach localhost directly — should be blocked.
	result := c.Check(context.Background(), "http://127.0.0.1:9999/", 0, 5*time.Second)
	if result.IsSuccess {
		t.Fatal("expected SSRF block for loopback address")
	}
	if result.FailureReason == "" {
		t.Error("expected non-empty failure reason for SSRF block")
	}
}

func TestSSRFTransport_BlocksPrivateRange(t *testing.T) {
	c := checker.New(5*time.Second, false, 1<<20)
	// 10.0.0.1 is in the private range.
	result := c.Check(context.Background(), "http://10.0.0.1:80/", 0, 2*time.Second)
	if result.IsSuccess {
		t.Fatal("expected SSRF block for private IP")
	}
}

func TestCheck_MeasuresResponseTime(t *testing.T) {
	delay := 50 * time.Millisecond
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newChecker()
	result := c.Check(context.Background(), srv.URL, 0, 5*time.Second)

	if result.ResponseMs < int(delay.Milliseconds()) {
		t.Errorf("response time %d ms is less than sleep %d ms", result.ResponseMs, delay.Milliseconds())
	}
}

func TestCheck_ManyStatuses(t *testing.T) {
	cases := []struct {
		status int
		want   bool
	}{
		{200, true},
		{201, true},
		{204, true},
		{301, false},
		{400, false},
		{404, false},
		{500, false},
		{503, false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(fmt.Sprintf("status_%d", tc.status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
			}))
			defer srv.Close()

			c := newChecker()
			result := c.Check(context.Background(), srv.URL, 0, 5*time.Second)
			if result.IsSuccess != tc.want {
				t.Errorf("status %d: expected isSuccess=%v, got %v (reason: %s)",
					tc.status, tc.want, result.IsSuccess, result.FailureReason)
			}
		})
	}
}
