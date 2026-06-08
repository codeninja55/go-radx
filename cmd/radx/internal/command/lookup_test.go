package command

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
)

// TestLookupByKeyword confirms a keyword resolves to its canonical tag, VR, and name from the
// generated dictionary (not a hand-curated partial list, RADX-019).
func TestLookupByKeyword(t *testing.T) {
	stdout, stderr, code := runRadx(t, "lookup", "--format", "json", "PatientName")
	if code != exitcode.Success {
		t.Fatalf("lookup PatientName exit = %d, want %d\nstderr=%q", code, exitcode.Success, stderr)
	}
	var got lookupRecord
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout=%q", err, stdout)
	}
	if got.Status != "found" {
		t.Fatalf("status = %q, want found", got.Status)
	}
	if got.Keyword != "PatientName" {
		t.Errorf("keyword = %q, want PatientName", got.Keyword)
	}
	if got.VR != "PN" {
		t.Errorf("vr = %q, want PN", got.VR)
	}
	if !strings.Contains(got.Tag, "0010") {
		t.Errorf("tag = %q, want the 0010,0010 group", got.Tag)
	}
}

// TestLookupByTag confirms a parenthesised tag resolves to its keyword.
func TestLookupByTag(t *testing.T) {
	stdout, _, code := runRadx(t, "lookup", "--format", "json", "(0008,0016)")
	if code != exitcode.Success {
		t.Fatalf("lookup (0008,0016) exit = %d, want %d", code, exitcode.Success)
	}
	var got lookupRecord
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v", err)
	}
	if got.Keyword != "SOPClassUID" {
		t.Errorf("keyword = %q, want SOPClassUID", got.Keyword)
	}
}

// TestLookupUnknownExitsNonZero confirms a query that resolves to no dictionary entry is a failure,
// not a silent empty success: the input named something the standard does not define.
func TestLookupUnknownExitsNonZero(t *testing.T) {
	_, _, code := runRadx(t, "lookup", "--format", "json", "NotARealKeyword")
	if code == exitcode.Success {
		t.Fatalf("lookup of an unknown keyword exited 0; want non-zero")
	}
}

// TestLookupCSVGolden confirms the csv form emits a header and one row per query.
func TestLookupCSVGolden(t *testing.T) {
	stdout, _, code := runRadx(t, "lookup", "--format", "csv", "PatientID")
	if code != exitcode.Success {
		t.Fatalf("lookup --format csv exit = %d, want %d", code, exitcode.Success)
	}
	lines := nonEmptyLines(stdout)
	if len(lines) != 2 {
		t.Fatalf("want header + 1 row, got %d:\n%s", len(lines), stdout)
	}
	if lines[0] != "query,status,tag,keyword,name,vr,vm" {
		t.Errorf("header = %q", lines[0])
	}
}
