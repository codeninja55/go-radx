package server

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/hl7v2"
)

// TestDaemonMLLPRoundTrip starts a daemon with the MLLP role on loopback, sends one HL7 v2 message
// over MLLP, and asserts the configured handler's acknowledgement comes back, then shuts down cleanly.
func TestDaemonMLLPRoundTrip(t *testing.T) {
	t.Parallel()
	handler := hl7v2.HandlerFunc(func(_ context.Context, m *hl7v2.Message) (*hl7v2.Message, error) {
		return m.BuildACK(hl7v2.AckAccept)
	})
	mllpRole, err := NewMLLPRole(handler, WithMLLPPort(0))
	if err != nil {
		t.Fatalf("NewMLLPRole: %v", err)
	}
	d, err := New(WithMLLP(mllpRole))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	runCtx, cancelRun := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()
	waitForAddrs(t, d, "mllp")

	addr := d.Addrs()["mllp"]
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := hl7v2.NewClient(addr.String())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	msg, err := hl7v2.Parse([]byte(
		"MSH|^~\\&|SEND|FAC|RECV|FAC|20260101000000||ADT^A01^ADT_A01|MSG0001|P|2.5.1\r" +
			"EVN|A01|20260101000000\r" +
			"PID|1||MRN001^^^HOSP^MR||DOE^JOHN||19700101|M\r" +
			"PV1|1|I\r"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	ack, err := client.Send(ctx, msg)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if ack == nil {
		t.Fatal("nil acknowledgement from MLLP round-trip")
	}

	cancelRun()
	if err := <-runErr; err != nil {
		t.Fatalf("Run returned %v on clean shutdown, want nil", err)
	}
}

// TestDaemonDICOMwebServes starts a daemon with the DICOMweb role on loopback and asserts the HTTP
// surface answers under the configured base path (a QIDO-RS studies search against the empty
// catalogue returns a non-5xx response), then shuts down cleanly. It exercises the role's HTTP
// listener, base-path mount, and auth middleware end-to-end.
func TestDaemonDICOMwebServes(t *testing.T) {
	t.Parallel()
	store, cat := newTestBackends(t)
	webRole, err := NewDICOMwebRole(store, cat, WithDICOMwebPort(0))
	if err != nil {
		t.Fatalf("NewDICOMwebRole: %v", err)
	}
	d, err := New(WithDICOMweb(webRole))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	runCtx, cancelRun := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()
	waitForAddrs(t, d, "dicomweb")

	addr := d.Addrs()["dicomweb"]
	url := "http://" + addr.String() + "/dicom-web/studies"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode >= 500 {
		t.Errorf("QIDO-RS studies search returned %d, want a non-5xx response", resp.StatusCode)
	}

	cancelRun()
	if err := <-runErr; err != nil {
		t.Fatalf("Run returned %v on clean shutdown, want nil", err)
	}
}
