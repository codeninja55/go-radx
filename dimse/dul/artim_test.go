package dul

import (
	"testing"
	"time"
)

func TestARTIMFiresAfterDuration(t *testing.T) {
	timer := newARTIM(20 * time.Millisecond)
	timer.start()
	select {
	case <-timer.expired():
		// fired as expected
	case <-time.After(time.Second):
		t.Fatal("ARTIM timer did not fire within the timeout")
	}
	timer.stop()
}

func TestARTIMStopPreventsFire(t *testing.T) {
	timer := newARTIM(20 * time.Millisecond)
	timer.start()
	timer.stop()
	select {
	case <-timer.expired():
		t.Fatal("ARTIM timer fired after being stopped")
	case <-time.After(60 * time.Millisecond):
		// no fire — correct
	}
}

func TestARTIMRestart(t *testing.T) {
	timer := newARTIM(20 * time.Millisecond)
	timer.start()
	timer.stop()
	// A second start must arm a fresh timer.
	timer.start()
	select {
	case <-timer.expired():
	case <-time.After(time.Second):
		t.Fatal("restarted ARTIM timer did not fire")
	}
	timer.stop()
}

// TestARTIMZeroDurationNeverFires confirms a zero/disabled timeout is inert rather than
// firing immediately (the association layer disables ARTIM when no timeout is configured).
func TestARTIMZeroDurationNeverFires(t *testing.T) {
	timer := newARTIM(0)
	timer.start()
	select {
	case <-timer.expired():
		t.Fatal("zero-duration ARTIM timer should never fire")
	case <-time.After(40 * time.Millisecond):
	}
	timer.stop()
}

// TestARTIMRestartDropsPriorExpiry guards against a stale expiry leaking across a re-arm:
// after a timer has fired, re-arming then stopping must leave the next wait clean. The
// harder in-flight-callback race (Stop returning while the callback is mid-flight) is
// closed by construction — the callback checks its arm generation under the same mutex
// that drains the channel, so a callback from a superseded arm never signals.
func TestARTIMRestartDropsPriorExpiry(t *testing.T) {
	timer := newARTIM(2 * time.Millisecond)
	timer.start()
	time.Sleep(30 * time.Millisecond) // let the first arm fire; its expiry is now buffered
	timer.start()                     // re-arm: invalidates the prior arm and drains its expiry
	timer.stop()                      // stop the re-armed timer before it can fire
	select {
	case <-timer.expired():
		t.Fatal("a prior arm's expiry leaked after restart + stop")
	case <-time.After(30 * time.Millisecond):
		// no stale fire — correct
	}
}
