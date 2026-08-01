package api

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRetryWithBackoffSkipsClientError(t *testing.T) {
	calls := 0
	cfg := RetryConfig{MaxRetries: 3, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond}
	err := retryWithBackoff(cfg, func() error {
		calls++
		return &MathClientError{StatusCode: 400, Reason: "expresión inválida"}
	})

	var clientErr *MathClientError
	if !errors.As(err, &clientErr) {
		t.Fatalf("expected *MathClientError, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected no retry for client error, got %d calls", calls)
	}
}

func TestRetryWithBackoffRetriesTransientError(t *testing.T) {
	calls := 0
	cfg := RetryConfig{MaxRetries: 2, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond}
	err := retryWithBackoff(cfg, func() error {
		calls++
		return errors.New("upstream down")
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls (2 retries), got %d", calls)
	}
}

func TestExtractMathError(t *testing.T) {
	if got := extractMathError([]byte(`{"success":false,"error":"expected a symbol"}`)); got != "expected a symbol" {
		t.Fatalf("expected reason from JSON, got %q", got)
	}
	if got := extractMathError([]byte(`<html>oops</html>`)); strings.TrimSpace(got) != "<html>oops</html>" {
		t.Fatalf("expected raw body fallback, got %q", got)
	}
}
