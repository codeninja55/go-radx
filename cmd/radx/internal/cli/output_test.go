package cli

import (
	"bytes"
	"strings"
	"testing"
)

// TestShowBannerRules asserts the banner-suppression matrix: the banner shows only in human
// format, on an interactive stdout, and when --quiet is not set; a machine format or a
// non-TTY stdout always suppresses it (docs/reference/cli.md "Diagnostics go to stderr").
func TestShowBannerRules(t *testing.T) {
	cases := []struct {
		name   string
		format Format
		quiet  bool
		tty    bool
		want   bool
	}{
		{"human + tty", FormatHuman, false, true, true},
		{"human + tty + quiet", FormatHuman, true, true, false},
		{"human + non-tty", FormatHuman, false, false, false},
		{"json + tty", FormatJSON, false, true, false},
		{"csv + tty", FormatCSV, false, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewOutput(&bytes.Buffer{}, &bytes.Buffer{}, tc.format, tc.quiet, false, tc.tty)
			if got := o.ShowBanner(); got != tc.want {
				t.Errorf("ShowBanner() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestBannerGoesToDiagnosticOnly confirms the banner is written to the diagnostic sink and
// never to the machine sink, so machine stdout stays clean.
func TestBannerGoesToDiagnosticOnly(t *testing.T) {
	var machine, diag bytes.Buffer
	o := NewOutput(&machine, &diag, FormatHuman, false, false, true)
	o.Banner("hello-banner")
	if machine.Len() != 0 {
		t.Errorf("banner reached the machine sink: %q", machine.String())
	}
	if !strings.Contains(diag.String(), "hello-banner") {
		t.Errorf("banner not on the diagnostic sink: %q", diag.String())
	}
}

// TestEmitJSONWritesOnlyMachineSink confirms the JSON emitter touches only the machine sink.
func TestEmitJSONWritesOnlyMachineSink(t *testing.T) {
	var machine, diag bytes.Buffer
	o := NewOutput(&machine, &diag, FormatJSON, false, false, false)
	if err := o.EmitJSON(map[string]string{"status": "success"}); err != nil {
		t.Fatalf("EmitJSON: %v", err)
	}
	if diag.Len() != 0 {
		t.Errorf("EmitJSON wrote to the diagnostic sink: %q", diag.String())
	}
	if !strings.Contains(machine.String(), "\"status\": \"success\"") {
		t.Errorf("EmitJSON machine output = %q", machine.String())
	}
}

// TestIsTTYBufferIsNotTTY confirms a non-*os.File writer (a buffer in tests) is never a TTY,
// so tests see banner-free machine output by construction.
func TestIsTTYBufferIsNotTTY(t *testing.T) {
	if IsTTY(&bytes.Buffer{}) {
		t.Error("a bytes.Buffer must not be reported as a TTY")
	}
}
