package dul

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/codeninja55/go-radx/dimse/pdu"
)

// Connection wraps a TCP connection and handles PDU communication
type Connection struct {
	conn          net.Conn
	maxPDULength  uint32
	sm            *StateMachine
	mu            sync.Mutex
	readDeadline  time.Duration
	writeDeadline time.Duration
}

// NewConnection creates a new connection from a net.Conn
func NewConnection(conn net.Conn) *Connection {
	return &Connection{
		conn:          conn,
		maxPDULength:  pdu.DefaultMaxPDULength,
		sm:            NewStateMachine(),
		readDeadline:  30 * time.Second,
		writeDeadline: 30 * time.Second,
	}
}

// SetMaxPDULength sets the maximum PDU length for this connection
func (c *Connection) SetMaxPDULength(length uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if length > pdu.MaxPDULength {
		length = pdu.MaxPDULength
	}
	c.maxPDULength = length
}

// GetMaxPDULength returns the maximum PDU length
func (c *Connection) GetMaxPDULength() uint32 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.maxPDULength
}

// SendPDU sends a PDU on the connection
func (c *Connection) SendPDU(ctx context.Context, p pdu.PDU) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Set write deadline
	if c.writeDeadline > 0 {
		if err := c.conn.SetWriteDeadline(time.Now().Add(c.writeDeadline)); err != nil {
			return fmt.Errorf("set write deadline: %w", err)
		}
		defer c.conn.SetWriteDeadline(time.Time{})
	}

	// Encode and send PDU
	if err := p.Encode(c.conn); err != nil {
		return fmt.Errorf("encode PDU: %w", err)
	}

	return nil
}

// ReadPDU reads a PDU from the connection
func (c *Connection) ReadPDU(ctx context.Context) (pdu.PDU, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Set read deadline
	if c.readDeadline > 0 {
		if err := c.conn.SetReadDeadline(time.Now().Add(c.readDeadline)); err != nil {
			return nil, fmt.Errorf("set read deadline: %w", err)
		}
		defer c.conn.SetReadDeadline(time.Time{})
	}

	// Read PDU
	p, err := pdu.ReadPDU(c.conn)
	if err != nil {
		if err == io.EOF {
			// Connection closed
			_, _ = c.sm.ProcessEvent(AE17)
		}
		return nil, err
	}

	return p, nil
}

// Close closes the connection
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		_, _ = c.sm.ProcessEvent(AE17)
		return err
	}
	return nil
}

// RemoteAddr returns the remote network address
func (c *Connection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// LocalAddr returns the local network address
func (c *Connection) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// SetReadDeadline sets the read timeout duration
func (c *Connection) SetReadDeadline(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readDeadline = d
}

// SetWriteDeadline sets the write timeout duration
func (c *Connection) SetWriteDeadline(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writeDeadline = d
}

// StateMachine returns the underlying state machine
func (c *Connection) StateMachine() *StateMachine {
	return c.sm
}

// TriggerTransportIndication triggers the AE-2 event (Transport connection indication)
// This should be called by SCP after accepting a TCP connection
func (c *Connection) TriggerTransportIndication(ctx context.Context) error {
	_, err := c.sm.ProcessEvent(AE2)
	if err != nil {
		return fmt.Errorf("trigger transport indication: %w", err)
	}
	return nil
}

// Dial establishes a new connection to the specified address
func Dial(ctx context.Context, network, address string) (*Connection, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, network, address)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	c := NewConnection(conn)
	// Trigger transport connect confirmation event
	_, _ = c.sm.ProcessEvent(AE1)

	return c, nil
}
