package dicomweb

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
)

// xmlRoundTripDataSet builds the canonical mixed dataset the XML parity tests exercise: a
// PersonName, a sequence with one nested item, and a binary attribute. It is the XML twin of
// the JSON round-trip fixtures, covering the three structurally distinct value forms the
// Native DICOM Model and DICOM JSON each represent differently.
func xmlRoundTripDataSet(t *testing.T) *dicom.DataSet {
	t.Helper()
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: tagPatientName, VR: dicom.VRPN, Value: dicom.NewStrings(dicom.VRPN, "Doe^Jane^Q^Dr^PhD")})
	ds.Set(dicom.Element{Tag: tagSOPInstanceUID, VR: dicom.VRUI, Value: dicom.NewStrings(dicom.VRUI, "1.2.840.113619.2.55.3.123")})
	ds.Set(dicom.Element{Tag: tagRows, VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, 512)})

	inner := dicom.NewDataSet()
	inner.Set(dicom.Element{Tag: dicom.NewTag(0x0008, 0x1150), VR: dicom.VRUI, Value: dicom.NewStrings(dicom.VRUI, "1.2.840.10008.5.1.4.1.1.2")})
	inner.Set(dicom.Element{Tag: dicom.NewTag(0x0008, 0x1155), VR: dicom.VRUI, Value: dicom.NewStrings(dicom.VRUI, "1.2.3.4.5")})
	seq := dicom.NewSequence(inner)
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x0008, 0x1140), VR: dicom.VRSQ, Value: dicom.NewSequenceValue(seq)})
	return ds
}

// TestXMLRoundTripMatchesJSON is the core parity assertion: a dataset round-tripped through
// the Native DICOM Model XML must decode to the same logical content as the same dataset
// round-tripped through DICOM JSON. Because the JSON codec is byte-stable, re-marshalling
// both decoded datasets to JSON and comparing the bytes proves the XML and JSON forms carry
// equal content, across a PersonName, a nested sequence item, and the scalar VRs.
func TestXMLRoundTripMatchesJSON(t *testing.T) {
	ds := xmlRoundTripDataSet(t)

	wantJSON, err := MarshalJSON(ds)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	xmlBytes, err := MarshalXML(ds)
	if err != nil {
		t.Fatalf("MarshalXML: %v", err)
	}
	fromXML, err := UnmarshalXML(xmlBytes)
	if err != nil {
		t.Fatalf("UnmarshalXML: %v", err)
	}
	gotJSON, err := MarshalJSON(fromXML)
	if err != nil {
		t.Fatalf("MarshalJSON(fromXML): %v", err)
	}

	if !bytes.Equal(wantJSON, gotJSON) {
		t.Errorf("XML round-trip does not match JSON round-trip:\nwant: %s\ngot:  %s", wantJSON, gotJSON)
	}
}

// TestXMLRoundTripByteStable verifies the XML codec re-encodes a decoded document to the
// same bytes, so the serialization is deterministic (attributes in ascending tag order).
func TestXMLRoundTripByteStable(t *testing.T) {
	ds := xmlRoundTripDataSet(t)
	first, err := MarshalXML(ds)
	if err != nil {
		t.Fatalf("first MarshalXML: %v", err)
	}
	decoded, err := UnmarshalXML(first)
	if err != nil {
		t.Fatalf("UnmarshalXML: %v", err)
	}
	second, err := MarshalXML(decoded)
	if err != nil {
		t.Fatalf("second MarshalXML: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Errorf("XML round-trip not byte-stable:\nfirst:  %s\nsecond: %s", first, second)
	}
}

// TestMarshalXMLShape checks the Native DICOM Model structure: the root element, namespace,
// a DicomAttribute carrying tag/vr/keyword, and the positional PersonName component form.
func TestMarshalXMLShape(t *testing.T) {
	ds := xmlRoundTripDataSet(t)
	b, err := MarshalXML(ds)
	if err != nil {
		t.Fatalf("MarshalXML: %v", err)
	}
	got := string(b)

	for _, want := range []string{
		"<NativeDicomModel",
		`xmlns="http://dicom.nema.org/PS3.19/models/NativeDICOM"`,
		`<DicomAttribute tag="00100010" vr="PN" keyword="PatientName">`,
		`<PersonName number="1">`,
		"<Alphabetic>",
		"<FamilyName>Doe</FamilyName>",
		"<GivenName>Jane</GivenName>",
		"<NamePrefix>Dr</NamePrefix>",
		"<NameSuffix>PhD</NameSuffix>",
		`<DicomAttribute tag="00280010" vr="US" keyword="Rows">`,
		`<Value number="1">512</Value>`,
		`<DicomAttribute tag="00081140" vr="SQ"`,
		`<Item number="1">`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("XML missing %q\nfull document:\n%s", want, got)
		}
	}
}

// TestXMLPersonNameRoundTrip verifies a multi-group PersonName survives the XML round-trip
// with every component preserved, exercising the Alphabetic/Ideographic structure that the
// Native DICOM Model breaks into named child elements.
func TestXMLPersonNameRoundTrip(t *testing.T) {
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: tagPatientName, VR: dicom.VRPN, Value: dicom.NewStrings(dicom.VRPN, "Yamada^Tarou=山田^太郎")})

	b, err := MarshalXML(ds)
	if err != nil {
		t.Fatalf("MarshalXML: %v", err)
	}
	if !strings.Contains(string(b), "<Ideographic>") {
		t.Errorf("expected an Ideographic group, got %s", b)
	}
	got, err := UnmarshalXML(b)
	if err != nil {
		t.Fatalf("UnmarshalXML: %v", err)
	}
	if pn, ok := got.GetString(tagPatientName); !ok || pn != "Yamada^Tarou=山田^太郎" {
		t.Errorf("PatientName = %q (ok=%v), want Yamada^Tarou=山田^太郎", pn, ok)
	}
}

// TestXMLInlineBinaryRoundTrip verifies a binary attribute encodes as base64 InlineBinary
// and decodes back to the same bytes.
func TestXMLInlineBinaryRoundTrip(t *testing.T) {
	tagPixelData := dicom.NewTag(0x7FE0, 0x0010)
	raw := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: tagPixelData, VR: dicom.VROB, Value: dicom.NewBytes(dicom.VROB, raw)})

	b, err := MarshalXML(ds)
	if err != nil {
		t.Fatalf("MarshalXML: %v", err)
	}
	if !strings.Contains(string(b), "<InlineBinary>"+base64.StdEncoding.EncodeToString(raw)) {
		t.Errorf("OB should emit base64 InlineBinary, got %s", b)
	}
	got, err := UnmarshalXML(b)
	if err != nil {
		t.Fatalf("UnmarshalXML: %v", err)
	}
	gotElem, ok := got.Get(tagPixelData)
	if !ok {
		t.Fatalf("missing pixel data after round-trip")
	}
	bv, ok := gotElem.Value.(*dicom.Bytes)
	if !ok || !bytes.Equal(bv.Bytes(), raw) {
		t.Errorf("pixel data = %v (ok=%v), want %v", gotElem.Value, ok, raw)
	}
}

// TestXMLBulkDataReference verifies a value above the threshold encodes as a <BulkData
// uri=...> reference and decodes back to the same reference when no resolver is supplied,
// the XML twin of the DICOM-JSON BulkDataURI decode contract.
func TestXMLBulkDataReference(t *testing.T) {
	tagPixelData := dicom.NewTag(0x7FE0, 0x0010)
	raw := bytes.Repeat([]byte{0xAB}, 64)
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: tagPixelData, VR: dicom.VROB, Value: dicom.NewBytes(dicom.VROB, raw)})

	b, err := MarshalXML(ds, WithBulkDataThreshold(16), WithBulkDataBaseURL("https://pacs.example.org/bulk/"))
	if err != nil {
		t.Fatalf("MarshalXML: %v", err)
	}
	if !strings.Contains(string(b), "<BulkData uri=") {
		t.Fatalf("value above threshold with base URL should emit BulkData uri, got %s", b)
	}
	if strings.Contains(string(b), "<InlineBinary>") {
		t.Errorf("BulkData and InlineBinary are mutually exclusive, got %s", b)
	}

	got, err := UnmarshalXML(b)
	if err != nil {
		t.Fatalf("UnmarshalXML: %v", err)
	}
	uris := BulkDataURIs(got)
	if len(uris) != 1 || !strings.HasPrefix(string(uris[0]), "https://pacs.example.org/bulk/") {
		t.Errorf("expected one BulkData reference, got %v", uris)
	}

	// With a resolver, the reference resolves to the original bytes.
	resolved, err := UnmarshalXML(b, WithBulkDataResolver(func(_ context.Context, _ BulkDataURI) ([]byte, error) {
		return raw, nil
	}))
	if err != nil {
		t.Fatalf("UnmarshalXML with resolver: %v", err)
	}
	gotElem, ok := resolved.Get(tagPixelData)
	if !ok {
		t.Fatalf("missing pixel data after resolved round-trip")
	}
	bv, ok := gotElem.Value.(*dicom.Bytes)
	if !ok || !bytes.Equal(bv.Bytes(), raw) {
		t.Errorf("resolved pixel data mismatch")
	}
}

// TestUnmarshalXMLRejectsNonRoot rejects a document whose root is not NativeDicomModel.
func TestUnmarshalXMLRejectsNonRoot(t *testing.T) {
	_, err := UnmarshalXML([]byte(`<Other><DicomAttribute tag="00100010" vr="PN"/></Other>`))
	if err == nil {
		t.Fatal("expected an error for a non-NativeDicomModel root")
	}
	if _, ok := errors.AsType[*DecodeError](err); !ok {
		t.Errorf("error = %v, want *DecodeError", err)
	}
}

// TestUnmarshalXMLRejectsMalformed rejects a syntactically invalid document and never echoes
// its bytes in the error (PRD §9.1).
func TestUnmarshalXMLRejectsMalformed(t *testing.T) {
	_, err := UnmarshalXML([]byte(`<NativeDicomModel><DicomAttribute tag=`))
	if err == nil {
		t.Fatal("expected an error for malformed XML")
	}
	if strings.Contains(err.Error(), "00100010") {
		t.Errorf("error should not echo document bytes: %v", err)
	}
}

// TestUnmarshalXMLRejectsAmbiguousVR rejects an unknown VR with a typed DecodeError naming
// the tag, never the supplied VR text.
func TestUnmarshalXMLRejectsUnknownVR(t *testing.T) {
	_, err := UnmarshalXML([]byte(`<NativeDicomModel><DicomAttribute tag="00100010" vr="ZZ"><Value number="1">x</Value></DicomAttribute></NativeDicomModel>`))
	if err == nil {
		t.Fatal("expected an error for an unknown VR")
	}
	if _, ok := errors.AsType[*DecodeError](err); !ok {
		t.Errorf("error = %v, want *DecodeError", err)
	}
}

// TestXMLSequenceDepthLimit rejects over-deep SQ nesting before materialising the forbidden
// level, returning ErrLimitExceeded (PRD §9.3).
func TestXMLSequenceDepthLimit(t *testing.T) {
	twoLevels := `<NativeDicomModel><DicomAttribute tag="00081140" vr="SQ"><Item number="1">` +
		`<DicomAttribute tag="00081140" vr="SQ"><Item number="1"></Item></DicomAttribute>` +
		`</Item></DicomAttribute></NativeDicomModel>`
	if _, err := UnmarshalXML([]byte(twoLevels), WithMaxJSONDepth(1)); !errors.Is(err, ErrLimitExceeded) {
		t.Errorf("two SQ levels at depth 1: err = %v, want ErrLimitExceeded", err)
	}
}

// TestUnmarshalXMLRejectsMultipleForms rejects an attribute that declares more than one value
// form, fail-closed (PS3.19 §A.1 permits exactly one).
func TestUnmarshalXMLRejectsMultipleForms(t *testing.T) {
	doc := `<NativeDicomModel><DicomAttribute tag="7FE00010" vr="OB">` +
		`<InlineBinary>AQID</InlineBinary><BulkData uri="https://x/y"/>` +
		`</DicomAttribute></NativeDicomModel>`
	_, err := UnmarshalXML([]byte(doc))
	if err == nil {
		t.Fatal("expected an error for multiple value forms")
	}
	if _, ok := errors.AsType[*DecodeError](err); !ok {
		t.Errorf("error = %v, want *DecodeError", err)
	}
}
