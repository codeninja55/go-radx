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

func TestIsMultipartRelated(t *testing.T) {
	if !isMultipartRelated(`multipart/related; boundary=abc; type="application/dicom"`) {
		t.Fatal("multipart/related Content-Type not recognised")
	}
	if isMultipartRelated("application/dicom+json") {
		t.Fatal("application/dicom+json wrongly recognised as multipart/related")
	}
}
