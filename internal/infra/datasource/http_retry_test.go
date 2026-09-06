package datasource

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestIsRetryableDownloadError(t *testing.T) {
	t.Parallel()

	if !isRetryableDownloadError(httpStatusError{status: http.StatusServiceUnavailable}) {
		t.Fatal("503 should be retryable")
	}
	if !isRetryableDownloadError(fmt.Errorf("failed to batch get sheet data: %w", httpStatusError{status: http.StatusServiceUnavailable})) {
		t.Fatal("wrapped 503 should be retryable")
	}
	if isRetryableDownloadError(httpStatusError{status: http.StatusNotFound}) {
		t.Fatal("404 should not be retryable")
	}
	if isRetryableDownloadError(fmt.Errorf("failed to execute request")) {
		t.Fatal("generic error should not be retryable")
	}
}

func TestRetryDownloadRetriesTransientError(t *testing.T) {
	original := retryBackoff
	retryBackoff = func(int) time.Duration { return 0 }
	t.Cleanup(func() { retryBackoff = original })

	attempts := 0
	err := retryDownload(context.Background(), "additional_songs", func() error {
		attempts++
		if attempts < 3 {
			return httpStatusError{status: http.StatusServiceUnavailable}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected retry to succeed, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestRetryDownloadStopsDuringBackoffWhenCanceled(t *testing.T) {
	original := retryBackoff
	retryBackoff = func(int) time.Duration { return time.Hour }
	t.Cleanup(func() { retryBackoff = original })

	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0
	err := retryDownload(ctx, "official", func() error {
		attempts++
		cancel()
		return httpStatusError{status: http.StatusServiceUnavailable}
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestWrapRequestErrorRedactsURLAndPreservesCause(t *testing.T) {
	cause := errors.New("connection refused")
	err := wrapRequestError(&url.Error{
		Op:  "Get",
		URL: "https://example.invalid/data?key=secret",
		Err: cause,
	})
	if !errors.Is(err, cause) {
		t.Fatalf("error = %v, want wrapped cause", err)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatalf("error contains secret URL: %v", err)
	}
}

func TestWithGoogleSheetsGateReturnsWhenCanceled(t *testing.T) {
	googleSheetsGate <- struct{}{}
	t.Cleanup(func() { <-googleSheetsGate })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := withGoogleSheetsGate(ctx, func() error {
		t.Fatal("canceled waiter must not enter the gate")
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
