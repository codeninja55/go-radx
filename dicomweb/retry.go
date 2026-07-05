package dicomweb

import (
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Retry defaults, applied when a RetryPolicy field is zero or negative.
const (
	defaultRetryMaxRetries = 3
	defaultRetryBackoff    = 250 * time.Millisecond
	defaultRetryMaxBackoff = 10 * time.Second
)

// retryDrainLimit bounds how much of a discarded retryable response body is drained
// before the retry, so the connection can be reused without reading an unbounded error
// body (PRD §9.3).
const retryDrainLimit = 64 << 10

// RetryPolicy bounds the opt-in HTTP retry WithRetry enables. Retries apply only to
// idempotent requests (GET and OPTIONS) with no body: a STOW POST is never replayed,
// since a replay could double-store instances. A retried request waits an exponential
// backoff (Backoff doubled per attempt, capped at MaxBackoff); a 429 or 503 carrying a
// Retry-After header waits the server's delay instead, and when that delay exceeds
// MaxBackoff the answer is returned without retrying — the client neither hammers the
// origin earlier than instructed nor stalls past the caller's bound. A zero or negative
// field takes its default.
type RetryPolicy struct {
	// MaxRetries is the number of retries after the initial attempt (default 3).
	MaxRetries int
	// Backoff is the first retry's wait, doubled for each further retry (default 250ms).
	Backoff time.Duration
	// MaxBackoff caps a single wait, including a server-supplied Retry-After
	// (default 10s).
	MaxBackoff time.Duration
}

func (p RetryPolicy) maxRetries() int {
	if p.MaxRetries <= 0 {
		return defaultRetryMaxRetries
	}
	return p.MaxRetries
}

func (p RetryPolicy) backoff() time.Duration {
	if p.Backoff <= 0 {
		return defaultRetryBackoff
	}
	return p.Backoff
}

func (p RetryPolicy) maxBackoff() time.Duration {
	if p.MaxBackoff <= 0 {
		return defaultRetryMaxBackoff
	}
	return p.MaxBackoff
}

// WithRetry enables bounded retries for idempotent requests. The default is off: without
// this option every request is attempted exactly once.
// Retries cover transport errors and transient 5xx/429 answers on GET and OPTIONS only; see
// RetryPolicy for the backoff and Retry-After semantics. The retry layer wraps the
// credential transport, so a retried request is re-authenticated (a refreshed OAuth2
// token is picked up between attempts).
func WithRetry(p RetryPolicy) ClientOption {
	return func(c *Client) { c.retryPolicy = &p }
}

// retryTransport is the http.RoundTripper WithRetry layers over the client's transport
// stack. It holds no per-request state, so it is safe for concurrent use.
type retryTransport struct {
	next   http.RoundTripper
	policy RetryPolicy
}

// RoundTrip attempts the request, retrying an idempotent request on a transport error or
// a retryable status until the budget is spent, the wait policy gives up, or the request
// context ends. A non-idempotent request, or one carrying a body, is passed through
// untouched.
func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !retryableMethod(req.Method) || req.Body != nil {
		return t.next.RoundTrip(req)
	}
	for attempt := 0; ; attempt++ {
		resp, err := t.next.RoundTrip(req)
		if attempt >= t.policy.maxRetries() || req.Context().Err() != nil {
			return resp, err
		}
		wait, retry := t.retryWait(resp, err, attempt)
		if !retry {
			return resp, err
		}
		if resp != nil {
			// Drain a bounded amount so the connection is reusable, then release it.
			_, _ = io.CopyN(io.Discard, resp.Body, retryDrainLimit)
			_ = resp.Body.Close()
		}
		timer := time.NewTimer(wait)
		select {
		case <-req.Context().Done():
			timer.Stop()
			return nil, req.Context().Err()
		case <-timer.C:
		}
	}
}

// retryableMethod reports whether a method is safe to replay: GET and OPTIONS only. POST
// (STOW) is never retried.
func retryableMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodOptions
}

// retryWait decides whether the attempt's outcome is retryable and how long to wait
// before the next attempt. A transport error retries on the backoff schedule. A 429 or
// 503 honours a parseable Retry-After: the server's delay is used, and a delay beyond
// MaxBackoff (or a nonsensical negative one) refuses the retry so the answer is returned
// rather than retried early or waited on past the policy's bound. Of the remaining 5xx
// statuses only the transient ones (500, 502, 504) retry on the backoff schedule: 501,
// 505, and the other deterministic 5xx answers would fail identically on every attempt,
// so they are returned to the caller, as is every non-5xx status.
func (t *retryTransport) retryWait(resp *http.Response, err error, attempt int) (time.Duration, bool) {
	if err != nil {
		return t.backoffWait(attempt), true
	}
	switch resp.StatusCode {
	case http.StatusTooManyRequests, http.StatusServiceUnavailable:
		if d, ok := parseRetryAfter(resp.Header.Get("Retry-After")); ok {
			if d < 0 || d > t.policy.maxBackoff() {
				return 0, false
			}
			return d, true
		}
		return t.backoffWait(attempt), true
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusGatewayTimeout:
		return t.backoffWait(attempt), true
	default:
		return 0, false
	}
}

// backoffWait is the exponential schedule: Backoff doubled per prior attempt, capped at
// MaxBackoff.
func (t *retryTransport) backoffWait(attempt int) time.Duration {
	wait := t.policy.backoff() << attempt // #nosec G115 -- attempt is a small bounded loop counter
	if maxWait := t.policy.maxBackoff(); wait > maxWait || wait <= 0 {
		return maxWait
	}
	return wait
}

// parseRetryAfter parses a Retry-After header in either RFC 9110 form: a non-negative
// delay in seconds, or an HTTP-date (rendered as the remaining delay, floored at zero).
// An absent or unparseable value reports false so the caller falls back to its own
// schedule rather than trusting a malformed delay.
func parseRetryAfter(v string) (time.Duration, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs < 0 {
			return 0, false
		}
		// Clamp before multiplying: a huge delay must saturate, not wrap negative and
		// read as an instant retry.
		if int64(secs) > math.MaxInt64/int64(time.Second) {
			return time.Duration(math.MaxInt64), true
		}
		return time.Duration(secs) * time.Second, true
	}
	if at, err := http.ParseTime(v); err == nil {
		d := time.Until(at)
		if d < 0 {
			d = 0
		}
		return d, true
	}
	return 0, false
}
