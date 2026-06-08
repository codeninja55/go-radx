package server

import (
	"net"
	"time"
)

// bindPollInterval is how often waitBound polls a protocol server for its bound address while
// ListenAndServe is racing to publish it. It is short enough that start returns promptly once the
// listener binds, yet not a busy-spin.
const bindPollInterval = time.Millisecond

// waitBound blocks until a protocol server has bound its listener (addr returns non-nil) or its
// ListenAndServe goroutine returned early with a bind error. The protocol servers' ListenAndServe
// blocks for the listener's lifetime, so a successful bind never sends on served; the address
// becoming readable is the success signal. An early send on served is a bind failure (or an
// immediate clean stop if the daemon was shut down before start). On success it records the bound
// address through record and returns nil.
func waitBound(served <-chan error, addr func() net.Addr, record func(net.Addr)) error {
	ticker := time.NewTicker(bindPollInterval)
	defer ticker.Stop()
	for {
		if a := addr(); a != nil {
			record(a)
			return nil
		}
		select {
		case err := <-served:
			// ListenAndServe returned before the address was readable: a bind failure (err != nil) or
			// an immediate clean stop. Re-check the address in case it bound and stopped in the same
			// instant, so a fast shutdown does not look like a bind failure.
			if a := addr(); a != nil {
				record(a)
				return nil
			}
			return err
		case <-ticker.C:
		}
	}
}
