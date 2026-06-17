package dicomweb

import (
	"context"
	"io"
	"net/http"
	"time"
)

// RetryPolicy configures automatic retry of a DICOMweb request on a transient failure, matching
// the dicomweb-client set_http_retry_params behaviour. A retry covers a transport error (a
// connection reset, a DNS hiccup) and a retryable HTTP status (429 Too Many Requests, 503 Service
// Unavailable, 502 Bad Gateway, 504 Gateway Timeout). It deliberately never retries a 4xx other
// than 429: a 400/404/409 is a deterministic client-side fault that a replay cannot fix. Retry is
// applied only to idempotent reads (GET, OPTIONS); a STOW-RS POST is never auto-retried, because a
// non-transactional store could double-store an instance on a retried request (PS3.18 §10.5).
type RetryPolicy struct {
	// MaxRetries is the number of retries after the first attempt (so MaxRetries=3 makes up to
	// four attempts). Zero disables retry.
	MaxRetries int
	// BaseDelay is the initial back-off before the first retry. Each subsequent retry doubles the
	// delay (exponential back-off), capped at MaxDelay.
	BaseDelay time.Duration
	// MaxDelay caps the back-off between retries. Zero leaves it uncapped.
	MaxDelay time.Duration
	// RetryStatuses, when non-empty, replaces the default retryable status set. A status not in
	// the effective set is returned to the caller without a retry.
	RetryStatuses []int
}

// defaultRetryStatuses are the HTTP statuses a retry covers by default: the transient overload
// and gateway statuses. A 4xx other than 429 is a deterministic fault and is never retried.
var defaultRetryStatuses = []int{
	http.StatusTooManyRequests,    // 429
	http.StatusBadGateway,         // 502
	http.StatusServiceUnavailable, // 503
	http.StatusGatewayTimeout,     // 504
}

// WithRetry enables automatic retry of idempotent reads on a transient failure (PS3.18 §8 leaves
// retry to the implementation; this matches the dicomweb-client retry knob). The policy bounds the
// retry count and the exponential back-off. A STOW-RS store is never retried regardless of this
// option, since a replayed non-transactional store risks a double-store.
func WithRetry(p RetryPolicy) ClientOption {
	return func(c *Client) {
		policy := p
		if len(policy.RetryStatuses) == 0 {
			policy.RetryStatuses = defaultRetryStatuses
		}
		c.retry = &policy
	}
}

// do executes an idempotent request through the client's transport, applying the configured retry
// policy on a transient failure. With no policy it is a single httpClient.Do plus the package's
// transport-error wrapping. A request carrying a body is not retried (its body is consumed on the
// first attempt); the package only calls do for GET and OPTIONS, which have nil bodies, so this is
// never reached with a non-replayable body. method and path are the redacted descriptors for the
// transport error (PRD §9.1).
func (c *Client) do(req *http.Request, method, path string) (*http.Response, error) {
	if c.retry == nil || c.retry.MaxRetries <= 0 || req.Body != nil {
		resp, err := c.httpClient.Do(req) // #nosec G704 -- the URL is joined from the caller-configured base URL; requesting the configured service is the client's purpose
		if err != nil {
			return nil, c.transportError(method, path, err)
		}
		return resp, nil
	}
	return c.doWithRetry(req, method, path)
}

// doWithRetry runs the request up to MaxRetries+1 times, retrying a transport error or a retryable
// status with exponential back-off. The back-off respects the request context: a cancelled context
// aborts the wait and returns the context error rather than sleeping out the back-off. On the final
// attempt the last result (response or error) is returned as-is, so the caller sees the real
// failure rather than a synthetic one.
func (c *Client) doWithRetry(req *http.Request, method, path string) (*http.Response, error) {
	ctx := req.Context()
	var lastErr error
	for attempt := 0; attempt <= c.retry.MaxRetries; attempt++ {
		if attempt > 0 {
			if err := sleepBackoff(ctx, c.retry, attempt); err != nil {
				if lastErr != nil {
					return nil, lastErr
				}
				return nil, c.transportError(method, path, err)
			}
		}
		resp, err := c.httpClient.Do(req) // #nosec G704 -- the URL is joined from the caller-configured base URL; requesting the configured service is the client's purpose
		if err != nil {
			lastErr = c.transportError(method, path, err)
			if attempt == c.retry.MaxRetries {
				return nil, lastErr
			}
			continue
		}
		if attempt < c.retry.MaxRetries && c.retry.isRetryable(resp.StatusCode) {
			// Drain and close so the connection can be reused for the retry, then try again.
			drainAndClose(resp)
			continue
		}
		return resp, nil
	}
	return nil, lastErr
}

// isRetryable reports whether a status is in the policy's retryable set.
func (p *RetryPolicy) isRetryable(status int) bool {
	for _, s := range p.RetryStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// sleepBackoff waits the exponential back-off for the given attempt (1-based among retries),
// returning early with the context error if the context is cancelled during the wait. BaseDelay
// doubles per attempt, capped at MaxDelay; a zero BaseDelay means no wait.
func sleepBackoff(ctx context.Context, p *RetryPolicy, attempt int) error {
	delay := p.BaseDelay
	for i := 1; i < attempt; i++ {
		delay *= 2
		if p.MaxDelay > 0 && delay >= p.MaxDelay {
			delay = p.MaxDelay
			break
		}
	}
	if p.MaxDelay > 0 && delay > p.MaxDelay {
		delay = p.MaxDelay
	}
	if delay <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// drainAndClose discards a response body and closes it so the underlying connection returns to the
// pool for reuse on the retry. A small bounded read is enough since the body is discarded.
func drainAndClose(resp *http.Response) {
	if resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	_ = resp.Body.Close()
}
