package dimse

import (
	"errors"
	"strings"
	"testing"
)

func TestParseAETitleAcceptsValid(t *testing.T) {
	cases := []string{
		"A",                // 1 char (lower length bound)
		"ORTHANC",          // typical
		"RADX-SCU",         // hyphen allowed
		"TEST_SCP",         // underscore allowed
		"AET 1",            // embedded space allowed (significant only at the ends)
		"ABCDEFGHIJKLMNOP", // exactly 16 chars (upper length bound)
	}
	for _, s := range cases {
		got, err := ParseAETitle(s)
		if err != nil {
			t.Errorf("ParseAETitle(%q) returned error %v, want nil", s, err)
			continue
		}
		if string(got) != s {
			t.Errorf("ParseAETitle(%q) = %q, want %q", s, got, s)
		}
		if !got.Valid() {
			t.Errorf("ParseAETitle(%q).Valid() = false, want true", s)
		}
	}
}

func TestParseAETitleRejectsEmpty(t *testing.T) {
	if _, err := ParseAETitle(""); err == nil {
		t.Error("ParseAETitle(\"\") = nil error, want rejection of an empty title")
	}
}

func TestParseAETitleRejectsTooLong(t *testing.T) {
	seventeen := strings.Repeat("A", 17)
	if _, err := ParseAETitle(seventeen); err == nil {
		t.Errorf("ParseAETitle(%q) = nil error, want rejection of a 17-character title", seventeen)
	}
}

func TestParseAETitleRejectsBadRepertoire(t *testing.T) {
	cases := map[string]string{
		"backslash":        `AE\TITLE`,    // 0x5C is forbidden (it is the DICOM value delimiter)
		"control NUL":      "AE\x00TITLE", // control characters are forbidden
		"control DEL":      "AE\x7fTITLE",
		"control tab":      "AE\tTITLE",
		"non-ASCII":        "AÉTITLE", // outside the default ASCII repertoire
		"all whitespace":   "    ",    // trailing/leading spaces are insignificant -> empty
		"leading/trailing": "  OK  ",  // trims to "OK", which is valid (this case must PASS)
	}
	for name, s := range cases {
		_, err := ParseAETitle(s)
		switch name {
		case "leading/trailing":
			if err != nil {
				t.Errorf("ParseAETitle(%q) [%s] = %v, want nil (surrounding spaces are insignificant)", s, name, err)
			}
		default:
			if err == nil {
				t.Errorf("ParseAETitle(%q) [%s] = nil error, want rejection", s, name)
			}
		}
	}
}

func TestParseAETitleErrorIsTyped(t *testing.T) {
	_, err := ParseAETitle("")
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("ParseAETitle(\"\") error = %T, want *ValidationError", err)
	}
}

func TestParseAETitleTrimsSurroundingSpaces(t *testing.T) {
	got, err := ParseAETitle("  RADX-SCU  ")
	if err != nil {
		t.Fatalf("ParseAETitle: %v", err)
	}
	if string(got) != "RADX-SCU" {
		t.Errorf("ParseAETitle trimmed = %q, want %q", got, "RADX-SCU")
	}
}

func TestAETitleStringAndValid(t *testing.T) {
	a := AETitle("ORTHANC")
	if a.String() != "ORTHANC" {
		t.Errorf("AETitle.String() = %q, want ORTHANC", a.String())
	}
	if !a.Valid() {
		t.Error("AETitle(\"ORTHANC\").Valid() = false, want true")
	}
	if AETitle("").Valid() {
		t.Error("AETitle(\"\").Valid() = true, want false")
	}
	if AETitle(strings.Repeat("A", 17)).Valid() {
		t.Error("a 17-character AETitle should not be Valid()")
	}
}
