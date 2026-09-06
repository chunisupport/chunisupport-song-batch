package datasource

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

const downloadRetryAttempts = 3

// googleSheetsGate は同一 API キーへの並行アクセスによる 429/503 を避ける。
var googleSheetsGate = make(chan struct{}, 1)

var retryBackoff = func(attempt int) time.Duration {
	return time.Duration(attempt) * 500 * time.Millisecond
}

type httpStatusError struct {
	status int
}

func (e httpStatusError) Error() string {
	return fmt.Sprintf("unexpected status code: %d", e.status)
}

func isRetryableDownloadError(err error) bool {
	var statusErr httpStatusError
	if !errors.As(err, &statusErr) {
		return false
	}
	switch statusErr.status {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func retryDownload(ctx context.Context, sourceType string, fn func() error) error {
	var err error
	for attempt := 1; attempt <= downloadRetryAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err = fn()
		if err == nil || !isRetryableDownloadError(err) || attempt == downloadRetryAttempts {
			return err
		}
		slog.Warn("retrying datasource download", "type", sourceType, "attempt", attempt, "error", err)
		if wait := retryBackoff(attempt); wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	return err
}

func wrapRequestError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return fmt.Errorf("failed to execute request: %w", urlErr.Err)
	}
	return fmt.Errorf("failed to execute request: %w", err)
}

func withGoogleSheetsGate(ctx context.Context, fn func() error) error {
	select {
	case googleSheetsGate <- struct{}{}:
		defer func() { <-googleSheetsGate }()
		return fn()
	case <-ctx.Done():
		return ctx.Err()
	}
}
