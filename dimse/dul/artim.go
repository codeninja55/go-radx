package dul

import (
	"sync"
	"time"
)

// artim is the Association Request/Reject/Release Timer (PS3.8 §9.1.5). The DUL starts
// it whenever it is waiting for a peer to complete association establishment, rejection,
// or release, and its expiry is Evt18. A zero duration disables the timer (the timeout
// is configured by the association layer); a disabled timer never fires.
type artim struct {
	duration time.Duration
	mu       sync.Mutex
	timer    *time.Timer
	gen      uint64 // bumped on every arm/stop; a fired callback signals only when current
	c        chan struct{}
}

func newARTIM(d time.Duration) *artim {
	return &artim{duration: d, c: make(chan struct{}, 1)}
}

// expired returns the channel that receives one value when the timer fires. The DUL
// run loop selects on it to raise Evt18.
func (a *artim) expired() <-chan struct{} { return a.c }

// start arms (or re-arms) the timer for its configured duration. A zero duration is
// inert. start is idempotent: re-arming stops any pending timer first so a stale firing
// cannot leak into the next wait.
func (a *artim) start() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.stopLocked() // bumps gen, invalidating any in-flight callback from a prior arm
	if a.duration <= 0 {
		return
	}
	myGen := a.gen
	a.timer = time.AfterFunc(a.duration, func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		// time.Timer.Stop does not wait for an already-firing callback, so a callback from
		// a previous arm can run after stop/re-arm. It signals only when its generation is
		// still current, so a stale expiry cannot inject a spurious Evt18 into the next
		// wait. The lock makes this check mutually exclusive with stopLocked's drain.
		if a.gen != myGen {
			return
		}
		// Non-blocking send: the buffered channel holds at most one pending expiry, so a
		// run loop that has not yet consumed the previous one is not blocked.
		select {
		case a.c <- struct{}{}:
		default:
		}
	})
}

// stop cancels a pending timer. It is safe to call when no timer is armed.
func (a *artim) stop() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.stopLocked()
}

func (a *artim) stopLocked() {
	a.gen++ // invalidate any in-flight callback from the current arm
	if a.timer != nil {
		a.timer.Stop()
		a.timer = nil
	}
	// Drain any expiry that already fired but was not yet consumed, so a subsequent wait
	// starts clean. Together with the generation check this closes both the
	// already-sent and the about-to-send windows.
	select {
	case <-a.c:
	default:
	}
}
