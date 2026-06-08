package command

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
)

// TestSCPHandlerStoresObject confirms the SCP handler writes a received object to disk in the
// Study/Series/SOP layout and reports a store-success status, and that a write actually lands (the
// honest-failure rule: success is reported only after the object is persisted).
func TestSCPHandlerStoresObject(t *testing.T) {
	root := t.TempDir()
	h := &scpHandler{root: root, acceptEcho: true, log: zap.NewNop()}

	ds := getInstance("1.2.3.4.400.10")
	status := h.Store(context.Background(), ds, dimse.OpInfo{TransferSyntax: dicom.ExplicitVRLittleEndian})
	if !status.IsSuccess() {
		t.Fatalf("Store status = %s, want success", status)
	}
	if h.received.Load() != 1 {
		t.Errorf("received count = %d, want 1", h.received.Load())
	}
	p := filepath.Join(root, "1.2.3.4.300.1", "1.2.3.4.300.2", "1.2.3.4.400.10.dcm")
	if _, err := os.Stat(p); err != nil {
		t.Errorf("expected stored object at %s: %v", p, err)
	}
}

// TestSCPHandlerRejectsTraversalUID confirms a sender-controlled UID carrying a path separator is
// rejected (a processing-failure status) and writes nothing, so a malformed identifier cannot
// escape the output directory (RADX-016).
func TestSCPHandlerRejectsTraversalUID(t *testing.T) {
	root := t.TempDir()
	h := &scpHandler{root: root, acceptEcho: true, log: zap.NewNop()}

	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.4")
	ds.SetString(dicom.TagSOPInstanceUID, "1.2.3.4.400.20")
	ds.SetString(dicom.TagStudyInstanceUID, "../escape")
	ds.SetString(dicom.TagSeriesInstanceUID, "1.2.3.4.400.2")

	status := h.Store(context.Background(), ds, dimse.OpInfo{TransferSyntax: dicom.ExplicitVRLittleEndian})
	if status.IsSuccess() {
		t.Fatalf("Store of a traversal UID reported success; want a failure status")
	}
	if h.received.Load() != 0 {
		t.Errorf("received count = %d, want 0 (nothing should be written)", h.received.Load())
	}
	// No file should have escaped the root.
	if _, err := os.Stat(filepath.Join(filepath.Dir(root), "escape")); !os.IsNotExist(err) {
		t.Errorf("a traversal file may have been written outside the root")
	}
}

// TestSCPHandlerEchoRefusedWhenDisabled confirms --accept-echo off makes the handler refuse a
// C-ECHO with an explicit SOP-class-not-supported status rather than a silent accept.
func TestSCPHandlerEchoRefusedWhenDisabled(t *testing.T) {
	h := &scpHandler{acceptEcho: false, log: zap.NewNop()}
	if status := h.Echo(context.Background(), dimse.OpInfo{}); status.IsSuccess() {
		t.Errorf("Echo with --accept-echo off = %s, want a non-success refusal", status)
	}
	on := &scpHandler{acceptEcho: true, log: zap.NewNop()}
	if status := on.Echo(context.Background(), dimse.OpInfo{}); !status.IsSuccess() {
		t.Errorf("Echo with --accept-echo on = %s, want success", status)
	}
}

// TestIsLoopbackBind confirms the loopback-bind detection that drives the non-loopback opt-in
// warning: loopback addresses and localhost are loopback; a wildcard or routable address is not.
func TestIsLoopbackBind(t *testing.T) {
	for bind, want := range map[string]bool{
		"127.0.0.1": true,
		"::1":       true,
		"localhost": true,
		"":          true,
		"0.0.0.0":   false,
		"10.0.0.5":  false,
	} {
		if got := isLoopbackBind(bind); got != want {
			t.Errorf("isLoopbackBind(%q) = %v, want %v", bind, got, want)
		}
	}
}
