// Package retry provides exponential-backoff retry logic for transient failures.
package retry

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"

	"go.uber.org/zap"

	"github.com/smcsoluciones/backup-system/agent/internal/config"
)

// Permanent wraps an error to signal it should NOT be retried.
// Use this for errors like "file not found" or "permission denied" where
// retrying would never succeed.
type Permanent struct{ Err error }

func (p Permanent) Error() string { return p.Err.Error() }
func (p Permanent) Unwrap() error { return p.Err }

// IsPermanent returns true if err is marked as non-retryable.
func IsPermanent(err error) bool {
	var p Permanent
	return errors.As(err, &p)
}

// Do runs op up to cfg.MaxAttempts times with exponential backoff + ±20% jitter.
//
// Rules:
//   - op returning nil → success, returns immediately.
//   - op returning Permanent{} → stops and returns the wrapped error without retrying.
//   - op returning any other error → retries after a delay, up to MaxAttempts.
//   - ctx cancellation → stops and returns ctx.Err().
//
// Backoff formula: delay = InitialDelay * 2^attempt, capped at 5 minutes.
func Do(ctx context.Context, cfg config.RetryConfig, log *zap.Logger, op func() error) error {
	maxAttempts := cfg.MaxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	baseDelay := cfg.InitialDelay
	if baseDelay <= 0 {
		baseDelay = time.Second
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		lastErr = op()
		if lastErr == nil {
			return nil
		}

		// Permanent error — surface immediately, no retry.
		if IsPermanent(lastErr) {
			var p Permanent
			errors.As(lastErr, &p)
			return p.Err
		}

		// Last attempt — don't sleep, just return.
		if attempt+1 >= maxAttempts {
			break
		}

		// Exponential backoff: baseDelay * 2^attempt, max 5 min.
		backoff := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt)))
		const maxBackoff = 5 * time.Minute
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
		// ±20% jitter to avoid thundering herd.
		jitter := time.Duration(float64(backoff) * (0.8 + 0.4*rand.Float64()))

		log.Warn("retry: attempt failed, backing off",
			zap.Int("attempt", attempt+1),
			zap.Int("max_attempts", maxAttempts),
			zap.Duration("retry_in", jitter.Round(time.Millisecond)),
			zap.Error(lastErr),
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(jitter):
		}
	}

	return lastErr
}
