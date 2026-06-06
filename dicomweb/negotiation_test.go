package dicomweb

import (
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
)

func TestAcceptInstancesCarriesTransferSyntax(t *testing.T) {
	if got := acceptInstances(); strings.Contains(got, "transfer-syntax") {
		t.Fatalf("acceptInstances() with no preference = %q, should omit transfer-syntax", got)
	}
	got := acceptInstances(dicom.ExplicitVRLittleEndian, dicom.ImplicitVRLittleEndian)
	// Each transfer syntax must be its own media range (comma-separated ranges), not one
	// comma-joined parameter value an origin would misread.
	wantRanges := []string{
		`multipart/related; type="application/dicom"; transfer-syntax=1.2.840.10008.1.2.1`,
		`multipart/related; type="application/dicom"; transfer-syntax=1.2.840.10008.1.2`,
	}
	if got != strings.Join(wantRanges, ", ") {
		t.Fatalf("acceptInstances() = %q, want two ordered transfer-syntax ranges", got)
	}
	// The server, emitting Explicit VR LE, must still accept this header (the first range
	// names that syntax).
	if !negotiateMultipartDICOM(got, dicom.ExplicitVRLittleEndian) {
		t.Fatalf("an Accept carrying transfer-syntax = %q must still negotiate Explicit VR LE", got)
	}
}

func TestNegotiateMultipartDICOM(t *testing.T) {
	cases := []struct {
		accept string
		want   bool
	}{
		{"", true},
		{`multipart/related; type="application/dicom"`, true},
		{"multipart/related", true},
		{"application/dicom", true},
		{"*/*", true},
		{"application/dicom+json", false},
		{`multipart/related; type="application/octet-stream"`, false},
		{"text/html", false},
	}
	for _, tc := range cases {
		t.Run(tc.accept, func(t *testing.T) {
			if got := negotiateMultipartDICOM(tc.accept, dicom.ExplicitVRLittleEndian); got != tc.want {
				t.Fatalf("negotiateMultipartDICOM(%q) = %v, want %v", tc.accept, got, tc.want)
			}
		})
	}
}

func TestNegotiateTransferSyntaxConstraint(t *testing.T) {
	const emit = dicom.ExplicitVRLittleEndian
	cases := []struct {
		accept string
		want   bool
	}{
		{`multipart/related; type="application/dicom"; transfer-syntax=1.2.840.10008.1.2.1`, true},
		{`multipart/related; type="application/dicom"; transfer-syntax=*`, true},
		{`multipart/related; type="application/dicom"; transfer-syntax=1.2.840.10008.1.2.4.50`, false},
	}
	for _, tc := range cases {
		t.Run(tc.accept, func(t *testing.T) {
			if got := negotiateMultipartDICOM(tc.accept, emit); got != tc.want {
				t.Fatalf("negotiateMultipartDICOM(%q) = %v, want %v", tc.accept, got, tc.want)
			}
		})
	}
}

func TestNegotiateDICOMJSON(t *testing.T) {
	cases := []struct {
		accept string
		want   bool
	}{
		{"", true},
		{"application/dicom+json", true},
		{"application/json", true},
		{"*/*", true},
		{"application/dicom+xml", false}, // XML deferred in v1
		{"multipart/related", false},
		// An explicit q=0 on the specific range vetoes a later */* wildcard (HTTP precedence,
		// RFC 9110 §12.5.1): the refused representation is not served via the wildcard.
		{"application/dicom+json;q=0, */*", false},
		{"application/json;q=0, */*", false},
		// A wildcard refusal cannot veto a more specific acceptance.
		{"*/*;q=0, application/dicom+json", true},
	}
	for _, tc := range cases {
		t.Run(tc.accept, func(t *testing.T) {
			if got := negotiateDICOMJSON(tc.accept); got != tc.want {
				t.Fatalf("negotiateDICOMJSON(%q) = %v, want %v", tc.accept, got, tc.want)
			}
		})
	}
}

func TestNegotiateSkipsMalformedRange(t *testing.T) {
	// A malformed media range must not be read as a match: only the well-formed,
	// acceptable range counts.
	if !negotiateDICOMJSON("garbage;;;, application/dicom+json") {
		t.Fatal("a valid range after a malformed one should still match")
	}
	if negotiateDICOMJSON("garbage;;;") {
		t.Fatal("a single malformed range must not match")
	}
}

func TestNegotiateMultipartOctet(t *testing.T) {
	cases := []struct {
		accept string
		want   bool
	}{
		{"", true},
		{`multipart/related; type="application/octet-stream"`, true},
		{"multipart/related", true},
		{"application/octet-stream", true},
		{"*/*", true},
		{`multipart/related; type="application/dicom"`, false},
		{"application/dicom+json", false},
	}
	for _, tc := range cases {
		t.Run(tc.accept, func(t *testing.T) {
			if got := negotiateMultipartOctet(tc.accept, dicom.ExplicitVRLittleEndian); got != tc.want {
				t.Fatalf("negotiateMultipartOctet(%q) = %v, want %v", tc.accept, got, tc.want)
			}
		})
	}
}

// TestNegotiateRetrieveTransferSyntax exercises the WADO-RS retrieve transfer-syntax policy:
// passthrough when the stored syntax satisfies the Accept (or no constraint / wildcard),
// transcode when a transcodable syntax is named, and not-acceptable (the caller answers 406)
// when nothing servable is named.
func TestNegotiateRetrieveTransferSyntax(t *testing.T) {
	const stored = dicom.ExplicitVRLittleEndian
	const storedTS = "1.2.840.10008.1.2.1"
	const jpeg = dicom.JPEGBaseline8Bit
	const jpegTS = "1.2.840.10008.1.2.4.50"

	cases := []struct {
		name            string
		accept          string
		transcodable    []dicom.TransferSyntax
		wantAcceptable  bool
		wantPassthrough bool
		wantSyntax      dicom.TransferSyntax
	}{
		{
			name:            "no constraint passes through stored",
			accept:          `multipart/related; type="application/dicom"`,
			wantAcceptable:  true,
			wantPassthrough: true,
			wantSyntax:      stored,
		},
		{
			name:            "wildcard passes through stored",
			accept:          `multipart/related; type="application/dicom"; transfer-syntax=*`,
			wantAcceptable:  true,
			wantPassthrough: true,
			wantSyntax:      stored,
		},
		{
			name:            "names stored syntax passes through",
			accept:          `multipart/related; type="application/dicom"; transfer-syntax=` + storedTS,
			wantAcceptable:  true,
			wantPassthrough: true,
			wantSyntax:      stored,
		},
		{
			name:            "names a transcodable syntax transcodes",
			accept:          `multipart/related; type="application/dicom"; transfer-syntax=` + jpegTS,
			transcodable:    []dicom.TransferSyntax{jpeg},
			wantAcceptable:  true,
			wantPassthrough: false,
			wantSyntax:      jpeg,
		},
		{
			name:           "names only an unservable syntax is not acceptable",
			accept:         `multipart/related; type="application/dicom"; transfer-syntax=` + jpegTS,
			wantAcceptable: false,
		},
		{
			name:            "prefers stored over transcode when both named",
			accept:          `multipart/related; transfer-syntax=` + jpegTS + `, multipart/related; transfer-syntax=` + storedTS,
			transcodable:    []dicom.TransferSyntax{jpeg},
			wantAcceptable:  true,
			wantPassthrough: true,
			wantSyntax:      stored,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := negotiateRetrieveTransferSyntax(tc.accept, stored, tc.transcodable...)
			if got.acceptable != tc.wantAcceptable {
				t.Fatalf("acceptable = %v, want %v", got.acceptable, tc.wantAcceptable)
			}
			if !tc.wantAcceptable {
				return
			}
			if got.passthrough != tc.wantPassthrough {
				t.Fatalf("passthrough = %v, want %v", got.passthrough, tc.wantPassthrough)
			}
			if got.syntax != tc.wantSyntax {
				t.Fatalf("syntax = %q, want %q", got.syntax, tc.wantSyntax)
			}
		})
	}
}

// TestNegotiateRetrieveTransferSyntaxBindsRangeToMediaType is the mismatched-range regression:
// a transfer-syntax parameter must only constrain the served representation when it is named on
// a media range that actually matches that representation. An Accept whose DICOM range names a
// syntax the server cannot serve, but whose unrelated range (application/json) names the stored
// syntax, must NOT pass: the json range's transfer-syntax does not bind the application/dicom
// part, so the request is 406, not a passthrough.
func TestNegotiateRetrieveTransferSyntaxBindsRangeToMediaType(t *testing.T) {
	const stored = dicom.ExplicitVRLittleEndian
	const storedTS = "1.2.840.10008.1.2.1"
	const jpegTS = "1.2.840.10008.1.2.4.50"

	// The DICOM range names only JPEG (unservable here); the json range names the stored
	// syntax but qualifies application/json, not the served application/dicom part.
	accept := `multipart/related; type="application/dicom"; transfer-syntax=` + jpegTS +
		`, application/json; transfer-syntax=` + storedTS
	got := negotiateRetrieveTransferSyntax(accept, stored)
	if got.acceptable {
		t.Fatalf("acceptable = true, want 406: the stored syntax was named only on an unrelated (json) range")
	}

	// Control: the same stored-syntax token on the DICOM range itself does bind and passes
	// through, confirming the binding is per media type, not a blanket rejection.
	acceptDICOM := `multipart/related; type="application/dicom"; transfer-syntax=` + storedTS
	if got := negotiateRetrieveTransferSyntax(acceptDICOM, stored); !got.acceptable || !got.passthrough {
		t.Fatalf("stored syntax on the DICOM range: acceptable=%v passthrough=%v, want both true", got.acceptable, got.passthrough)
	}
}

// TestAcceptTransferSyntaxesOnlyMatchingRanges asserts acceptTransferSyntaxes collects a
// transfer-syntax token only from a range whose media type matches, ignoring tokens on
// unrelated ranges.
func TestAcceptTransferSyntaxesOnlyMatchingRanges(t *testing.T) {
	const dicomTS = "1.2.840.10008.1.2.1"
	const jsonOnlyTS = "1.2.840.10008.1.2.4.50"
	accept := `application/dicom; transfer-syntax=` + dicomTS +
		`, application/json; transfer-syntax=` + jsonOnlyTS
	got := acceptTransferSyntaxes(accept, dicomRangeMatchesMediaType)
	if len(got) != 1 || got[0] != dicomTS {
		t.Fatalf("acceptTransferSyntaxes = %v, want only the DICOM range's token %q", got, dicomTS)
	}
}

func TestIsMultipartRelated(t *testing.T) {
	if !isMultipartRelated(`multipart/related; boundary=abc; type="application/dicom"`) {
		t.Fatal("multipart/related Content-Type not recognised")
	}
	if isMultipartRelated("application/dicom+json") {
		t.Fatal("application/dicom+json wrongly recognised as multipart/related")
	}
}
