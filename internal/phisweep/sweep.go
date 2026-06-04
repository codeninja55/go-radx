//go:build unix

// Package phisweep is the library-wide PHI-default sanity harness (PRD §11.2). It
// exercises representative library entry points and parsers at default verbosity
// over fixtures carrying known, distinctive PHI sentinel tokens, then scans every
// observable sink for any token. A single appearance is a failure.
//
// The four swept sinks are the process standard output, the process standard error,
// the strings of any returned errors, and the structured-log output captured from
// the logging package. These are the channels through which a careless code path
// could surface a patient value at default verbosity. The harness consolidates the
// per-package no-leak checks into one authoritative sweep so the no-PHI guarantee
// is test-enforced rather than convention-enforced.
//
// The sentinel tokens are deliberately synthetic — never real PHI — yet shaped like
// the patient values they stand in for (a person name, an identifier, a date) so a
// leak through a value-formatting path is caught the same way a real value would be.
package phisweep

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/codeninja55/go-radx/logging"
)

// Sink names the observable channel a sentinel surfaced through. It exists so a
// failure report can point at the exact channel that leaked.
type Sink string

const (
	// SinkStdout is the process standard output stream.
	SinkStdout Sink = "stdout"
	// SinkStderr is the process standard error stream.
	SinkStderr Sink = "stderr"
	// SinkError is the string of an error returned by an exercised entry point.
	SinkError Sink = "error"
	// SinkLog is the structured-log output captured from the logging package.
	SinkLog Sink = "log"
)

// Sinks is the canonical list of channels the sweep scans, in report order. It is
// the documented contract enumerated in docs/conformance/cli-server.md.
var Sinks = []Sink{SinkStdout, SinkStderr, SinkError, SinkLog}

// Capture holds the bytes a single exercised path emitted to each sink. The Run
// helper populates it; Scan reads it.
type Capture struct {
	Stdout string
	Stderr string
	Errors []string
	Logs   string
}

// byID returns the captured content for a sink id, so Scan can iterate Sinks
// uniformly without a per-sink branch at each call site.
func (c Capture) byID(id Sink) string {
	switch id {
	case SinkStdout:
		return c.Stdout
	case SinkStderr:
		return c.Stderr
	case SinkError:
		return strings.Join(c.Errors, "\n")
	case SinkLog:
		return c.Logs
	default:
		return ""
	}
}

// Leak records one sentinel appearance in one sink.
type Leak struct {
	Sentinel string
	Sink     Sink
}

func (l Leak) String() string {
	return fmt.Sprintf("sentinel %q surfaced in %s", l.Sentinel, l.Sink)
}

// Scan reports every sentinel that appears in any swept sink of c. An empty result
// means the path leaked nothing. Tokens are matched literally and case-sensitively:
// the sentinels are distinctive enough that a substring match needs no word
// boundaries, and a literal match cannot be defeated by surrounding punctuation a
// formatter might add.
func Scan(c Capture, sentinels []string) []Leak {
	var leaks []Leak
	for _, id := range Sinks {
		content := c.byID(id)
		if content == "" {
			continue
		}
		for _, token := range sentinels {
			if token == "" {
				continue
			}
			if strings.Contains(content, token) {
				leaks = append(leaks, Leak{Sentinel: token, Sink: id})
			}
		}
	}
	return leaks
}

// Run exercises exercise with a context-injected logger and the process standard
// streams redirected, returning everything those four sinks observed. The exercise
// closure receives the same context the logger is attached to, so any library call
// that logs through logging.FromContext is captured; it returns any errors whose
// strings should be scanned.
//
// The logger is built at the package's default level (info), matching the PRD's
// "default verbosity" requirement: the sweep proves no PHI surfaces when an
// operator runs with no debug flags, which is the posture that actually ships.
//
// Redirection replaces the process file descriptors 1 and 2 — not merely the Go
// os.Stdout/os.Stderr variables — so a raw descriptor write, a cgo write, or a
// writer bound to the descriptor before the run is captured too, not just a Go
// fmt.Print. Because it mutates process-global state, a caller must not run two Run
// invocations concurrently. The streams are always restored, even if exercise
// panics; the panic then propagates after cleanup.
func Run(exercise func(ctx context.Context) []error) (capture Capture, err error) {
	var logBuf bytes.Buffer
	logger, err := logging.NewLogger(&logBuf, logging.Config{})
	if err != nil {
		return Capture{}, fmt.Errorf("phisweep: build logger: %w", err)
	}
	ctx := logging.WithContext(context.Background(), logger)

	restoreOut, err := redirect(&os.Stdout, syscall.Stdout)
	if err != nil {
		return Capture{}, err
	}
	restoreErr, err := redirect(&os.Stderr, syscall.Stderr)
	if err != nil {
		_, outErr := restoreOut()
		return Capture{}, errors.Join(err, outErr)
	}

	var errs []error
	var restoreErrs error
	func() {
		// Restore in a deferred closure so a panic in exercise still rolls the
		// descriptors back before propagating, rather than leaving the process
		// streams pointing at closed pipes for the rest of the test binary.
		defer func() {
			var stderrErr, stdoutErr error
			capture.Stderr, stderrErr = restoreErr()
			capture.Stdout, stdoutErr = restoreOut()
			restoreErrs = errors.Join(stderrErr, stdoutErr)
		}()
		errs = exercise(ctx)
		_ = logger.Sync()
	}()
	if restoreErrs != nil {
		return capture, restoreErrs
	}

	errStrings := make([]string, 0, len(errs))
	for _, e := range errs {
		if e != nil {
			errStrings = append(errStrings, e.Error())
		}
	}
	capture.Errors = errStrings
	capture.Logs = logBuf.String()
	return capture, nil
}

// redirect points both *target (the Go stream variable) and the process descriptor
// fd at the write end of an OS pipe, returning a restore closure. The closure points
// the descriptor and the variable back at the original stream, closes the pipe
// writer, and returns everything written through the pipe. Redirecting the
// descriptor with dup2 — not only the Go variable — means writes that bypass the Go
// variable (a duplicated descriptor, a cgo write, a pre-bound writer) are captured.
func redirect(target **os.File, fd int) (func() (string, error), error) {
	original := *target
	saved, err := syscall.Dup(fd)
	if err != nil {
		return nil, fmt.Errorf("phisweep: save fd %d: %w", fd, err)
	}
	r, w, err := os.Pipe()
	if err != nil {
		_ = syscall.Close(saved)
		return nil, fmt.Errorf("phisweep: open capture pipe: %w", err)
	}
	if err := dup2(int(w.Fd()), fd); err != nil {
		_ = syscall.Close(saved)
		_ = r.Close()
		_ = w.Close()
		return nil, fmt.Errorf("phisweep: redirect fd %d: %w", fd, err)
	}
	*target = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	restore := func() (string, error) {
		*target = original
		restoreErr := dup2(saved, fd)
		// The pipe write end is referenced by both w and fd (the latter via the
		// earlier dup2 onto fd). The reader only sees EOF once every reference is
		// gone, so both must be dropped before reading done — otherwise a failed
		// restore would leave fd holding the writer open and <-done would block
		// forever. A successful restore already replaced fd's reference with saved;
		// a failed one leaves it dangling, so close fd directly in that case.
		if restoreErr != nil {
			_ = syscall.Close(fd)
		}
		_ = syscall.Close(saved)
		_ = w.Close()
		s := <-done
		_ = r.Close()
		if restoreErr != nil {
			return s, fmt.Errorf("phisweep: restore fd %d: %w", fd, restoreErr)
		}
		return s, nil
	}
	return restore, nil
}
