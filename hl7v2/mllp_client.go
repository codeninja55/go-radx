package hl7v2

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"
)

// defaultDialTimeout bounds how long NewClient waits to establish the TCP (and,
// under TLS, the handshake) connection. It mirrors a conservative interface
// engine default: long enough for a busy peer, short enough that an
// unreachable address fails fast rather than hanging a caller.
const defaultDialTimeout = 30 * time.Second

// clientConfig holds the resolved Client options. There is no global mutable
// state (PRD §9.4); every Client carries its own configuration, immutable after
// NewClient.
type clientConfig struct {
	dialTimeout  time.Duration
	readTimeout  time.Duration
	maxFrameSize int
	tlsConfig    *tls.Config
}

// ClientOption configures a Client at construction.
type ClientOption func(*clientConfig)

// WithClientDialTimeout sets how long NewClient waits to establish the
// connection (default 30s). A non-positive value restores the default.
func WithClientDialTimeout(d time.Duration) ClientOption {
	return func(c *clientConfig) {
		if d > 0 {
			c.dialTimeout = d
		}
	}
}

// WithClientReadTimeout sets the deadline applied to reading the acknowledgement
// frame on each Send (default none). It bounds how long a Send blocks waiting
// for a peer that accepts the message but never replies. A non-positive value
// clears the deadline.
func WithClientReadTimeout(d time.Duration) ClientOption {
	return func(c *clientConfig) { c.readTimeout = d }
}

// WithClientMaxFrameSize caps the size of the acknowledgement frame the client
// will accumulate before failing, guarding against a hostile peer (PRD §9.3).
// A non-positive value uses DefaultMaxFrameSize.
func WithClientMaxFrameSize(n int) ClientOption {
	return func(c *clientConfig) {
		if n > 0 {
			c.maxFrameSize = n
		}
	}
}

// WithClientTLS dials over TLS using cfg, so the client verifies the server's
// certificate against cfg.RootCAs and cfg.ServerName (PRD §9.7). Passing a
// config with InsecureSkipVerify set disables that verification and is the
// caller's explicit choice. A nil cfg leaves the client on plain TCP.
func WithClientTLS(cfg *tls.Config) ClientOption {
	return func(c *clientConfig) { c.tlsConfig = cfg }
}

// Client is a blocking MLLP sender. It holds one connection to a peer for its
// lifetime: Send frames a message, writes it, and blocks for the
// acknowledgement frame on the same connection, matching the synchronous
// request/response discipline of MLLP. It is NOT safe for concurrent Send calls
// on one Client — MLLP is a single in-flight exchange per connection — so a
// caller needing concurrency uses one Client per goroutine. A mutex serialises
// Send and Close so a concurrent Close does not race the connection field.
type Client struct {
	cfg  clientConfig
	addr string

	mu     sync.Mutex
	conn   net.Conn
	closed bool
}

// NewClient dials addr and returns a Client ready to Send. The connection is
// established eagerly so a dial failure surfaces here rather than on the first
// Send. Under WithClientTLS the TLS handshake (and certificate verification)
// completes before NewClient returns.
func NewClient(addr string, opts ...ClientOption) (*Client, error) {
	cfg := clientConfig{dialTimeout: defaultDialTimeout, maxFrameSize: DefaultMaxFrameSize}
	for _, opt := range opts {
		opt(&cfg)
	}

	conn, err := dial(addr, cfg)
	if err != nil {
		return nil, err
	}
	return &Client{cfg: cfg, addr: addr, conn: conn}, nil
}

// dial opens the transport for addr per cfg: a plain TCP connection, or a TLS
// connection (with the handshake completed) when a TLS config is set.
func dial(addr string, cfg clientConfig) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: cfg.dialTimeout}
	if cfg.tlsConfig == nil {
		conn, err := dialer.Dial("tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("hl7v2: dial %q: %w", addr, err)
		}
		return conn, nil
	}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, cfg.tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("hl7v2: tls dial %q: %w", addr, err)
	}
	return conn, nil
}

// Send marshals m, writes it as one MLLP frame, and blocks for the
// acknowledgement frame, returning it parsed. ctx cancellation interrupts both
// the write and the acknowledgement read: a cancelled ctx sets an immediate
// connection deadline so an in-flight I/O returns promptly, and Send returns
// ctx.Err(). A WithClientReadTimeout deadline bounds the acknowledgement read
// independently of ctx.
func (c *Client) Send(ctx context.Context, m *Message) (*Message, error) {
	raw, err := m.MarshalText()
	if err != nil {
		return nil, fmt.Errorf("hl7v2: marshal message for mllp send: %w", err)
	}
	ackBytes, err := c.SendRaw(ctx, raw)
	if err != nil {
		return nil, err
	}
	ack, err := Parse(ackBytes)
	if err != nil {
		return nil, fmt.Errorf("hl7v2: parse acknowledgement: %w", err)
	}
	return ack, nil
}

// SendRaw frames and sends the already-rendered message bytes and returns the
// raw acknowledgement payload, for a caller that has its own bytes or wants the
// unparsed reply. The same ctx and read-timeout discipline as Send applies.
func (c *Client) SendRaw(ctx context.Context, payload []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.conn == nil {
		return nil, fmt.Errorf("hl7v2: mllp client is closed")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Cancellation watcher: a cancelled ctx sets an immediate deadline so an
	// in-flight write or read returns at once. The watcher stops when the
	// exchange completes so it does not pin the connection past this Send.
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			_ = c.conn.SetDeadline(time.Now())
		case <-stop:
		}
	}()

	if err := WriteFrame(c.conn, payload); err != nil {
		return nil, ctxErrOr(ctx, err)
	}

	if c.cfg.readTimeout > 0 {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.cfg.readTimeout))
		defer func() { _ = c.conn.SetReadDeadline(time.Time{}) }()
	}

	ack, err := ReadFrame(ctx, c.conn, c.cfg.maxFrameSize)
	if err != nil {
		return nil, ctxErrOr(ctx, err)
	}
	return ack, nil
}

// Close closes the underlying connection. It is idempotent and safe to call
// concurrently with a returning Send; a second Close is a no-op.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	if c.conn == nil {
		return nil
	}
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("hl7v2: close mllp client: %w", err)
	}
	return nil
}

// ctxErrOr returns ctx.Err() when the context was cancelled (so a deadline the
// cancellation watcher set surfaces as the real cause), otherwise err.
func ctxErrOr(ctx context.Context, err error) error {
	if cerr := ctx.Err(); cerr != nil {
		return cerr
	}
	return err
}
