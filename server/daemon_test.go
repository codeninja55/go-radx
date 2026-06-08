package server

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dimse"
	"github.com/codeninja55/go-radx/hl7v2"
)

// newTestBackends builds an in-memory catalogue and a temp-dir file store for a daemon test.
func newTestBackends(t *testing.T) (ObjectStore, Catalogue) {
	t.Helper()
	store, err := FileStore(t.TempDir())
	if err != nil {
		t.Fatalf("FileStore: %v", err)
	}
	cat, err := SQLiteCatalogue(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("SQLiteCatalogue: %v", err)
	}
	return store, cat
}

// TestInsecureBindRequiresAuthenticator asserts the fail-closed bind policy at the Daemon: a
// non-loopback WithBind with no explicit Authenticator returns ErrInsecureBind, and the same bind
// with an explicit AllowAll() override is accepted (the deliberate, reviewable opt-in).
func TestInsecureBindRequiresAuthenticator(t *testing.T) {
	t.Parallel()
	store, cat := newTestBackends(t)
	aet, _ := dimse.ParseAETitle("RADX-SCP")
	dimseRole, err := NewDIMSERole(aet, store, cat)
	if err != nil {
		t.Fatalf("NewDIMSERole: %v", err)
	}

	// Non-loopback bind, no authenticator => ErrInsecureBind.
	_, err = New(WithDIMSE(dimseRole), WithBind("0.0.0.0"))
	if !errors.Is(err, ErrInsecureBind) {
		t.Fatalf("New(non-loopback bind, no auth) = %v, want ErrInsecureBind", err)
	}

	// A concrete non-loopback interface IP is equally refused.
	if _, err := New(WithDIMSE(dimseRole), WithBind("192.0.2.1")); !errors.Is(err, ErrInsecureBind) {
		t.Fatalf("New(interface IP, no auth) = %v, want ErrInsecureBind", err)
	}

	// Non-loopback bind WITH an explicit authenticator is accepted.
	if _, err := New(WithDIMSE(dimseRole), WithBind("0.0.0.0"), WithAuthenticator(AllowAll())); err != nil {
		t.Fatalf("New(non-loopback bind, explicit auth) = %v, want nil", err)
	}

	// Loopback bind with no authenticator is accepted (AllowAll is the safe localhost default).
	if _, err := New(WithDIMSE(dimseRole), WithBind("127.0.0.1")); err != nil {
		t.Fatalf("New(loopback bind, no auth) = %v, want nil", err)
	}

	// The default (no WithBind) is loopback and accepted.
	if _, err := New(WithDIMSE(dimseRole)); err != nil {
		t.Fatalf("New(default) = %v, want nil", err)
	}
}

// TestInsecureBindAppliesToEveryRole asserts the bind policy is applied uniformly: a non-loopback
// bind without auth is refused regardless of which role is mounted (DICOMweb, MLLP, DIMSE).
func TestInsecureBindAppliesToEveryRole(t *testing.T) {
	t.Parallel()
	store, cat := newTestBackends(t)

	webRole, err := NewDICOMwebRole(store, cat)
	if err != nil {
		t.Fatalf("NewDICOMwebRole: %v", err)
	}
	mllpRole, err := NewMLLPRole(hl7v2.HandlerFunc(func(_ context.Context, m *hl7v2.Message) (*hl7v2.Message, error) {
		return m.BuildACK(hl7v2.AckAccept)
	}))
	if err != nil {
		t.Fatalf("NewMLLPRole: %v", err)
	}

	if _, err := New(WithDICOMweb(webRole), WithBind("0.0.0.0")); !errors.Is(err, ErrInsecureBind) {
		t.Errorf("DICOMweb non-loopback bind = %v, want ErrInsecureBind", err)
	}
	if _, err := New(WithMLLP(mllpRole), WithBind("0.0.0.0")); !errors.Is(err, ErrInsecureBind) {
		t.Errorf("MLLP non-loopback bind = %v, want ErrInsecureBind", err)
	}
}

// TestNilAuthenticatorDoesNotSatisfyBindPolicy asserts that WithAuthenticator(nil) is treated as NOT
// set, so a non-loopback bind paired with a nil authenticator is refused with ErrInsecureBind exactly
// like omitting the authenticator. Were nil to count as set, the daemon would expose an unauthenticated
// server to the network — the very exposure the fail-closed bind policy prevents (PRD §9.1).
func TestNilAuthenticatorDoesNotSatisfyBindPolicy(t *testing.T) {
	t.Parallel()
	store, cat := newTestBackends(t)
	aet, _ := dimse.ParseAETitle("RADX-SCP")
	dimseRole, err := NewDIMSERole(aet, store, cat, WithDIMSEPort(0))
	if err != nil {
		t.Fatalf("NewDIMSERole: %v", err)
	}

	if _, err := New(WithDIMSE(dimseRole), WithBind("0.0.0.0"), WithAuthenticator(nil)); !errors.Is(err, ErrInsecureBind) {
		t.Fatalf("New(non-loopback bind, nil authenticator) = %v, want ErrInsecureBind", err)
	}

	// A loopback bind with a nil authenticator is still accepted; the safe AllowAll default stays in
	// place rather than leaving the daemon with no authenticator at all.
	if _, err := New(WithDIMSE(dimseRole), WithBind("127.0.0.1"), WithAuthenticator(nil)); err != nil {
		t.Fatalf("New(loopback bind, nil authenticator) = %v, want nil", err)
	}
}

// TestShutdownDrainsWithinDeadline asserts a Shutdown(ctx) drains the mounted roles and returns nil
// when they drain cleanly, exercising the SIGINT/SIGTERM-equivalent code path (Run blocks until the
// context is cancelled, then drains).
func TestShutdownDrainsWithinDeadline(t *testing.T) {
	t.Parallel()
	store, cat := newTestBackends(t)
	aet, _ := dimse.ParseAETitle("RADX-SCP")
	dimseRole, err := NewDIMSERole(aet, store, cat, WithDIMSEPort(0))
	if err != nil {
		t.Fatalf("NewDIMSERole: %v", err)
	}
	d, err := New(WithDIMSE(dimseRole))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	runCtx, cancelRun := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()

	waitForAddrs(t, d, "dimse")

	// Cancelling Run's context (the SIGINT/SIGTERM equivalent) drains within the deadline.
	cancelRun()
	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("Run returned %v on a clean cancel, want nil", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return within 10s of context cancellation")
	}
}

// TestShutdownTimeoutNamesRole asserts a role that cannot drain within the deadline yields a
// role-naming ErrShutdownTimeout from Run, an honest report that the stop was not clean (PRD §9.2).
func TestShutdownTimeoutNamesRole(t *testing.T) {
	t.Parallel()
	stubborn := &stubbornRole{roleName: "stubborn", boundAddr: loopbackTCPAddr(), unblock: make(chan struct{})}
	d, err := New(WithShutdownTimeout(50 * time.Millisecond))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Mount the fake role directly; the public option path takes concrete role types, so a test fake
	// is injected through the unexported config (same package).
	d.cfg.roles = append(d.cfg.roles, stubborn)

	runCtx, cancelRun := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()

	waitForAddrs(t, d, "stubborn")
	cancelRun()

	select {
	case err := <-runErr:
		if !errors.Is(err, ErrShutdownTimeout) {
			t.Fatalf("Run drain error = %v, want ErrShutdownTimeout", err)
		}
		if !contains(err.Error(), "stubborn") {
			t.Errorf("drain error %q does not name the role that failed to drain", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after a stubborn drain")
	}
	close(stubborn.unblock)
}

// stubbornRole is a fake role whose shutdown never completes until released, so the daemon's bounded
// drain hits its deadline and reports the role by name.
type stubbornRole struct {
	roleName  string
	boundAddr net.Addr
	unblock   chan struct{}
}

func (r *stubbornRole) name() string { return r.roleName }

func (r *stubbornRole) start(_ context.Context, _ string, _ roleEnv) error { return nil }

func (r *stubbornRole) addr() net.Addr { return r.boundAddr }

func (r *stubbornRole) shutdown(ctx context.Context) error {
	// Block until either the deadline elapses (the case under test) or the test releases us.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.unblock:
		return nil
	}
}

// waitForAddrs blocks until the daemon reports a bound address for role, or fails the test.
func waitForAddrs(t *testing.T, d *Daemon, role string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if addrs := d.Addrs(); addrs != nil {
			if _, ok := addrs[role]; ok {
				return
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("daemon did not bind role %q within 5s", role)
}

// loopbackTCPAddr returns a placeholder loopback address for a fake role's addr().
func loopbackTCPAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
}

// contains reports whether s contains substr.
func contains(s, substr string) bool {
	return len(substr) == 0 || indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
