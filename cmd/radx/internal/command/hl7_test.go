package command

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/hl7v2"
)

// syntheticORM is a minimal, fully synthetic HL7 v2 order message. The names and identifiers are
// deliberately fictional test tokens, never a real patient value (PRD §9.1).
const syntheticORM = "MSH|^~\\&|RADX|TEST|PACS|TEST|20260101120000||ORM^O01|RADXTEST01|P|2.4\r" +
	"PID|||TEST-0001^^^TEST^MR||FIXTURE^TESTPATIENT^^^^^L||20000101|O\r" +
	"ORC|NW|PLC-001|FIL-001||||||20260101120000\r" +
	"OBR|1|PLC-001|FIL-001|TESTCODE^TEST PROCEDURE^LOCAL|||20260101120100\r"

// startMLLPServer runs an in-process MLLP server on loopback replying with the given ack code, and
// returns its host and port.
func startMLLPServer(t *testing.T, ack hl7v2.AckCode) (host string, port int) {
	t.Helper()
	handler := hl7v2.HandlerFunc(func(_ context.Context, m *hl7v2.Message) (*hl7v2.Message, error) {
		return m.BuildACK(ack)
	})
	srv := hl7v2.NewServer(handler)

	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && srv.Addr() == nil {
		time.Sleep(time.Millisecond)
	}
	if srv.Addr() == nil {
		t.Fatal("MLLP server did not bind within the deadline")
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		<-served
	})
	tcp := srv.Addr().(*net.TCPAddr)
	return "127.0.0.1", tcp.Port
}

// TestHL7SendPositiveAck is the send golden: a message sent to an accepting MLLP server returns a
// positive (AA) acknowledgement, the result is clean JSON, and the command exits 0.
func TestHL7SendPositiveAck(t *testing.T) {
	host, port := startMLLPServer(t, hl7v2.AckAccept)
	dir := t.TempDir()
	msgFile := filepath.Join(dir, "order.hl7")
	if err := os.WriteFile(msgFile, []byte(syntheticORM), 0o600); err != nil {
		t.Fatalf("write message: %v", err)
	}

	stdout, stderr, code := runRadx(t, "hl7", "send", "--format", "json",
		"--host", host, "--port", strconv.Itoa(port), msgFile)
	if code != exitcode.Success {
		t.Fatalf("hl7 send exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}
	var got hl7AckResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout=%q", err, stdout)
	}
	if got.AckCode != "AA" || !got.Positive {
		t.Errorf("ack = %+v, want positive AA", got)
	}
}

// TestHL7SendNegativeAckExitsNonZero confirms a rejecting acknowledgement (AR) is reported as a
// failure outcome and the command exits non-zero, so a negative ack is never read as success.
func TestHL7SendNegativeAckExitsNonZero(t *testing.T) {
	host, port := startMLLPServer(t, hl7v2.AckReject)
	stdout, _, code := runRadxStdin(t, syntheticORM, "hl7", "send", "--format", "json",
		"--host", host, "--port", strconv.Itoa(port), "-")
	if code == exitcode.Success {
		t.Fatalf("hl7 send with a rejecting ack exited 0; want non-zero\nstdout=%q", stdout)
	}
	var got hl7AckResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v", err)
	}
	if got.Positive || got.AckCode != "AR" {
		t.Errorf("ack = %+v, want a non-positive AR", got)
	}
}

// TestHL7SendStdinPositiveAck confirms the message can be read from stdin when the path is "-".
func TestHL7SendStdinPositiveAck(t *testing.T) {
	host, port := startMLLPServer(t, hl7v2.AckAccept)
	stdout, _, code := runRadxStdin(t, syntheticORM, "hl7", "send", "--format", "json",
		"--host", host, "--port", strconv.Itoa(port), "-")
	if code != exitcode.Success {
		t.Fatalf("hl7 send from stdin exit = %d, want %d\nstdout=%q", code, exitcode.Success, stdout)
	}
	var got hl7AckResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v", err)
	}
	if !got.Positive {
		t.Errorf("ack = %+v, want positive", got)
	}
}

// TestHL7SendRejectAckExits4 confirms an AR (application reject) acknowledgement exits 4 (the
// network/protocol class), not 2 (usage): the message parsed and was sent fine, and the peer
// rejected it at the application level — a peer "no", not a flag mistake. An AE would map the same.
func TestHL7SendRejectAckExits4(t *testing.T) {
	host, port := startMLLPServer(t, hl7v2.AckReject)
	_, _, code := runRadxStdin(t, syntheticORM, "hl7", "send", "--format", "json",
		"--host", host, "--port", strconv.Itoa(port), "-")
	if code != exitcode.NetworkError {
		t.Fatalf("hl7 send with an AR ack exit = %d, want %d (network/protocol error)", code, exitcode.NetworkError)
	}
}

// TestHL7SendAcceptAckExits0 confirms an AA (accept) acknowledgement is success: exit 0.
func TestHL7SendAcceptAckExits0(t *testing.T) {
	host, port := startMLLPServer(t, hl7v2.AckAccept)
	_, _, code := runRadxStdin(t, syntheticORM, "hl7", "send", "--format", "json",
		"--host", host, "--port", strconv.Itoa(port), "-")
	if code != exitcode.Success {
		t.Fatalf("hl7 send with an AA ack exit = %d, want %d (success)", code, exitcode.Success)
	}
}

// TestHL7ListenBadAckIsUsageError confirms an invalid --ack code is a usage error, rejected before
// the listener binds.
func TestHL7ListenBadAckIsUsageError(t *testing.T) {
	_, _, code := runRadx(t, "hl7", "listen", "--ack", "ZZ", "--port", "0")
	if code != exitcode.UsageError {
		t.Fatalf("hl7 listen --ack ZZ exit = %d, want %d (usage error)", code, exitcode.UsageError)
	}
}
