package dicomweb

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"math"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
)

// tagPatientName, tagSOPInstanceUID, tagRows are the small set of standard tags the
// round-trip tests exercise: a PN, a UI, and a US element (the named regression).
var (
	tagPatientName    = dicom.NewTag(0x0010, 0x0010) // PN
	tagSOPInstanceUID = dicom.NewTag(0x0008, 0x0018) // UI
	tagRows           = dicom.NewTag(0x0028, 0x0010) // US
)

// sampleDataSet builds the canonical mixed dataset: a PN, a UI, and a US element.
func sampleDataSet(t *testing.T) *dicom.DataSet {
	t.Helper()
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: tagPatientName, VR: dicom.VRPN, Value: dicom.NewStrings(dicom.VRPN, "Doe^Jane")})
	ds.Set(dicom.Element{Tag: tagSOPInstanceUID, VR: dicom.VRUI, Value: dicom.NewStrings(dicom.VRUI, "1.2.840.113619.2.55.3.123")})
	ds.Set(dicom.Element{Tag: tagRows, VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, 512)})
	return ds
}

func TestMarshalJSONShape(t *testing.T) {
	ds := sampleDataSet(t)
	b, err := MarshalJSON(ds)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	pn, ok := doc["00100010"]
	if !ok {
		t.Fatalf("missing PatientName key 00100010 in %s", b)
	}
	var pnElem struct {
		VR    string            `json:"vr"`
		Value []json.RawMessage `json:"Value"`
	}
	if err := json.Unmarshal(pn, &pnElem); err != nil {
		t.Fatalf("decode PN element: %v", err)
	}
	if pnElem.VR != "PN" {
		t.Errorf("PN vr = %q, want PN", pnElem.VR)
	}
	if len(pnElem.Value) != 1 {
		t.Fatalf("PN Value length = %d, want 1", len(pnElem.Value))
	}
	var pnVal struct {
		Alphabetic  string `json:"Alphabetic"`
		Ideographic string `json:"Ideographic"`
		Phonetic    string `json:"Phonetic"`
	}
	if err := json.Unmarshal(pnElem.Value[0], &pnVal); err != nil {
		t.Fatalf("decode PN component group: %v", err)
	}
	if pnVal.Alphabetic != "Doe^Jane" {
		t.Errorf("PN Alphabetic = %q, want Doe^Jane", pnVal.Alphabetic)
	}
	if pnVal.Ideographic != "" || pnVal.Phonetic != "" {
		t.Errorf("empty PN component groups should be omitted, got %s", pnElem.Value[0])
	}

	// US must serialise as a JSON number, not a string (Annex F.2.3).
	if !bytes.Contains(doc["00280010"], []byte(`"Value":[512]`)) {
		t.Errorf("US Value should be the number 512, got %s", doc["00280010"])
	}

	// UI must serialise as a JSON string.
	if !bytes.Contains(doc["00080018"], []byte(`"1.2.840.113619.2.55.3.123"`)) {
		t.Errorf("UI Value should be a quoted string, got %s", doc["00080018"])
	}
}

func TestRoundTripByteStable(t *testing.T) {
	ds := sampleDataSet(t)

	first, err := MarshalJSON(ds)
	if err != nil {
		t.Fatalf("first MarshalJSON: %v", err)
	}
	decoded, err := UnmarshalJSON(first)
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	second, err := MarshalJSON(decoded)
	if err != nil {
		t.Fatalf("second MarshalJSON: %v", err)
	}

	if !bytes.Equal(first, second) {
		t.Errorf("Marshal->Unmarshal->Marshal not byte-stable:\nfirst:  %s\nsecond: %s", first, second)
	}
}

func TestRoundTripPreservesValues(t *testing.T) {
	ds := sampleDataSet(t)
	b, err := MarshalJSON(ds)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	got, err := UnmarshalJSON(b)
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	if pn, ok := got.GetString(tagPatientName); !ok || pn != "Doe^Jane" {
		t.Errorf("PatientName = %q (ok=%v), want Doe^Jane", pn, ok)
	}
	if uid, ok := got.GetUID(tagSOPInstanceUID); !ok || uid != "1.2.840.113619.2.55.3.123" {
		t.Errorf("SOPInstanceUID = %q (ok=%v)", uid, ok)
	}
	if n, ok := got.GetInt(tagRows); !ok || n != 512 {
		t.Errorf("Rows = %d (ok=%v), want 512", n, ok)
	}
	gotElem, ok := got.Get(tagRows)
	if !ok || gotElem.VR != dicom.VRUS {
		t.Errorf("Rows VR = %v (ok=%v), want US", gotElem.VR, ok)
	}
}

func TestMarshalEmptyValue(t *testing.T) {
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: tagPatientName, VR: dicom.VRPN, Value: dicom.NewStrings(dicom.VRPN)})

	b, err := MarshalJSON(ds)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	// An empty element carries the VR and no Value array per Annex F.2.5.
	if !bytes.Contains(b, []byte(`"00100010":{"vr":"PN"}`)) {
		t.Errorf("empty PN should carry vr with no Value, got %s", b)
	}
}

func TestSingleEmptyTextOmitsValue(t *testing.T) {
	// A text element whose only value is an empty string is a logically empty element and
	// must encode without a Value array, matching what decoding an empty element yields.
	tagStudyDesc := dicom.NewTag(0x0008, 0x1030) // StudyDescription (LO)
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: tagStudyDesc, VR: dicom.VRLO, Value: dicom.NewStrings(dicom.VRLO, "")})

	b, err := MarshalJSON(ds)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if bytes.Contains(b, []byte(`"Value"`)) {
		t.Errorf("a single empty text value should omit Value, got %s", b)
	}
}

func TestMultiValueAllEmptyPreservesMultiplicity(t *testing.T) {
	// Two empty components is VM=2, not an empty element; it must re-marshal as
	// [null,null] rather than collapse to no Value (which would lose multiplicity).
	got, err := UnmarshalJSON([]byte(`{"00080008":{"vr":"CS","Value":[null,null]}}`))
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	b, err := MarshalJSON(got)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if !bytes.Contains(b, []byte(`"Value":[null,null]`)) {
		t.Errorf("two empty components should re-marshal as [null,null], got %s", b)
	}
}

func TestUnmarshalRejectsBackslashInDelimitedTextValue(t *testing.T) {
	// A single LO JSON value containing the VM delimiter would be re-split into two values
	// by the Part-10 writer; reject it (PRD §9.2).
	_, err := UnmarshalJSON([]byte(`{"00081030":{"vr":"LO","Value":["A\\B"]}}`))
	if err == nil {
		t.Fatal("expected an error for a backslash in an LO value")
	}
	var decErr *DecodeError
	if !errors.As(err, &decErr) {
		t.Errorf("error = %v, want *DecodeError", err)
	}

	// A long-text VR (LT) permits a literal backslash and must still decode.
	if _, err := UnmarshalJSON([]byte(`{"00204000":{"vr":"LT","Value":["path\\to\\file"]}}`)); err != nil {
		t.Errorf("LT with a literal backslash should decode, got %v", err)
	}
}

func TestUnmarshalRejectsBackslashInPNComponentGroup(t *testing.T) {
	// "\" is the value-multiplicity separator; an embedded one in a PN component group
	// would split a single value into multiple on a later Part-10 write (PRD §9.2).
	_, err := UnmarshalJSON([]byte(`{"00100010":{"vr":"PN","Value":[{"Alphabetic":"Doe\\Jane"}]}}`))
	if err == nil {
		t.Fatal("expected an error for '\\' inside a PN component group")
	}
	var decErr *DecodeError
	if !errors.As(err, &decErr) {
		t.Errorf("error = %v, want *DecodeError", err)
	}
}

func TestJSONDepthCapCountsOnlySequenceLevels(t *testing.T) {
	// A flat dataset must decode even with WithMaxJSONDepth(0): the top level is not an
	// SQ nesting level.
	if _, err := UnmarshalJSON([]byte(`{"00080018":{"vr":"UI","Value":["1.2.3"]}}`), WithMaxJSONDepth(0)); err != nil {
		t.Errorf("flat dataset rejected at depth 0: %v", err)
	}

	// One SQ level must decode with WithMaxJSONDepth(1).
	oneLevel := `{"00081140":{"vr":"SQ","Value":[{"00080018":{"vr":"UI","Value":["1.2.3"]}}]}}`
	if _, err := UnmarshalJSON([]byte(oneLevel), WithMaxJSONDepth(1)); err != nil {
		t.Errorf("one SQ level rejected at depth 1: %v", err)
	}

	// Two SQ levels must be rejected at depth 1.
	twoLevels := `{"00081140":{"vr":"SQ","Value":[{"00081140":{"vr":"SQ","Value":[{}]}}]}}`
	if _, err := UnmarshalJSON([]byte(twoLevels), WithMaxJSONDepth(1)); !errors.Is(err, ErrLimitExceeded) {
		t.Errorf("two SQ levels at depth 1: err = %v, want ErrLimitExceeded", err)
	}
}

func TestRoundTripSequence(t *testing.T) {
	inner := dicom.NewDataSet()
	inner.Set(dicom.Element{Tag: tagSOPInstanceUID, VR: dicom.VRUI, Value: dicom.NewStrings(dicom.VRUI, "1.2.3.4")})
	seq := dicom.NewSequence(inner)

	tagRefSeq := dicom.NewTag(0x0008, 0x1140) // ReferencedImageSequence (SQ)
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: tagRefSeq, VR: dicom.VRSQ, Value: dicom.NewSequenceValue(seq)})

	b, err := MarshalJSON(ds)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	got, err := UnmarshalJSON(b)
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	gotSeq, ok := got.GetSequence(tagRefSeq)
	if !ok {
		t.Fatalf("expected a sequence at %s", tagRefSeq)
	}
	if gotSeq.Len() != 1 {
		t.Fatalf("sequence length = %d, want 1", gotSeq.Len())
	}
	for item := range gotSeq.Items() {
		if uid, ok := item.DataSet.GetUID(tagSOPInstanceUID); !ok || uid != "1.2.3.4" {
			t.Errorf("nested UID = %q (ok=%v), want 1.2.3.4", uid, ok)
		}
	}

	// SQ round-trip should be byte-stable too.
	second, err := MarshalJSON(got)
	if err != nil {
		t.Fatalf("second MarshalJSON: %v", err)
	}
	if !bytes.Equal(b, second) {
		t.Errorf("SQ round-trip not byte-stable:\nfirst:  %s\nsecond: %s", b, second)
	}
}

func TestInlineBinaryRoundTrip(t *testing.T) {
	tagPixelData := dicom.NewTag(0x7FE0, 0x0010)
	raw := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: tagPixelData, VR: dicom.VROB, Value: dicom.NewBytes(dicom.VROB, raw)})

	b, err := MarshalJSON(ds)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if !bytes.Contains(b, []byte(`"InlineBinary"`)) {
		t.Fatalf("OB should emit InlineBinary, got %s", b)
	}
	if !bytes.Contains(b, []byte(base64.StdEncoding.EncodeToString(raw))) {
		t.Errorf("InlineBinary should be the base64 of the bytes, got %s", b)
	}

	got, err := UnmarshalJSON(b)
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	gotElem, ok := got.Get(tagPixelData)
	if !ok {
		t.Fatalf("missing pixel data after round-trip")
	}
	bv, ok := gotElem.Value.(*dicom.Bytes)
	if !ok {
		t.Fatalf("pixel data value is %T, want *dicom.Bytes", gotElem.Value)
	}
	if !bytes.Equal(bv.Bytes(), raw) {
		t.Errorf("decoded bytes = %v, want %v", bv.Bytes(), raw)
	}
}

func TestBulkDataThresholdEmitsBulkDataURI(t *testing.T) {
	tagPixelData := dicom.NewTag(0x7FE0, 0x0010)
	raw := bytes.Repeat([]byte{0xAB}, 32)
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: tagPixelData, VR: dicom.VROB, Value: dicom.NewBytes(dicom.VROB, raw)})

	// With a base URL configured and a threshold below the value size, the encoder
	// emits BulkDataURI instead of inlining the bytes.
	b, err := MarshalJSON(ds, WithBulkDataThreshold(16), WithBulkDataBaseURL("https://pacs.example.org/bulk/"))
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if !bytes.Contains(b, []byte(`"BulkDataURI"`)) {
		t.Fatalf("value above threshold with base URL should emit BulkDataURI, got %s", b)
	}
	if bytes.Contains(b, []byte(`"InlineBinary"`)) {
		t.Errorf("BulkDataURI and InlineBinary are mutually exclusive, got %s", b)
	}
}

func TestBulkDataThresholdWithoutBaseURLInlines(t *testing.T) {
	tagPixelData := dicom.NewTag(0x7FE0, 0x0010)
	raw := bytes.Repeat([]byte{0xCD}, 32)
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: tagPixelData, VR: dicom.VROB, Value: dicom.NewBytes(dicom.VROB, raw)})

	// No base URL: the value falls back to InlineBinary even above the threshold.
	b, err := MarshalJSON(ds, WithBulkDataThreshold(16))
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if !bytes.Contains(b, []byte(`"InlineBinary"`)) {
		t.Errorf("without a base URL the value should inline, got %s", b)
	}
}

func TestBulkDataResolverOnDecode(t *testing.T) {
	doc := `{"7FE00010":{"vr":"OB","BulkDataURI":"https://pacs.example.org/bulk/abc"}}`
	resolved := []byte{0x09, 0x08, 0x07}

	var sawURI BulkDataURI
	got, err := UnmarshalJSON([]byte(doc), WithBulkDataResolver(func(_ context.Context, uri BulkDataURI) ([]byte, error) {
		sawURI = uri
		return resolved, nil
	}))
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if sawURI != "https://pacs.example.org/bulk/abc" {
		t.Errorf("resolver saw uri = %q", sawURI)
	}
	elem, ok := got.Get(dicom.NewTag(0x7FE0, 0x0010))
	if !ok {
		t.Fatalf("missing element after resolve")
	}
	bv, ok := elem.Value.(*dicom.Bytes)
	if !ok || !bytes.Equal(bv.Bytes(), resolved) {
		t.Errorf("resolved value = %v (%T), want %v", elem.Value, elem.Value, resolved)
	}
}

func TestBulkDataURIPreservedWithoutResolver(t *testing.T) {
	// Without a resolver the reference must survive decode and re-encode unchanged
	// (the decode contract: leave a BulkDataURI as a reference), never collapsing to
	// an empty InlineBinary.
	doc := `{"7FE00010":{"vr":"OB","BulkDataURI":"https://pacs.example.org/bulk/xyz"}}`

	got, err := UnmarshalJSON([]byte(doc))
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	round, err := MarshalJSON(got)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if !bytes.Contains(round, []byte(`"BulkDataURI":"https://pacs.example.org/bulk/xyz"`)) {
		t.Errorf("BulkDataURI not preserved on re-marshal, got %s", round)
	}
	if bytes.Contains(round, []byte(`"InlineBinary"`)) {
		t.Errorf("unresolved BulkDataURI must not become InlineBinary, got %s", round)
	}
}

func TestMaxJSONDepthExceeded(t *testing.T) {
	// Build a JSON document nesting SQ deeper than the configured limit. It must
	// return a typed *LimitExceededError, never a stack overflow.
	var sb strings.Builder
	const depth = 8
	tag := "00081140" // an SQ tag
	for i := 0; i < depth; i++ {
		sb.WriteString(`{"` + tag + `":{"vr":"SQ","Value":[`)
	}
	sb.WriteString(`{}`)
	for i := 0; i < depth; i++ {
		sb.WriteString(`]}}`)
	}

	_, err := UnmarshalJSON([]byte(sb.String()), WithMaxJSONDepth(4))
	if err == nil {
		t.Fatal("expected an error for over-deep nesting")
	}
	if !errors.Is(err, ErrLimitExceeded) {
		t.Errorf("error = %v, want ErrLimitExceeded", err)
	}
	var lerr *LimitExceededError
	if !errors.As(err, &lerr) {
		t.Errorf("error = %v, want a *LimitExceededError", err)
	}
}

func TestEmptyBinaryRoundTripsAsBytes(t *testing.T) {
	// A zero-length binary VR must stay *dicom.Bytes through a round trip, not become a
	// *dicom.Strings, so the type is stable.
	tagPixelData := dicom.NewTag(0x7FE0, 0x0010)
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: tagPixelData, VR: dicom.VROB, Value: dicom.NewBytes(dicom.VROB, nil)})

	b, err := MarshalJSON(ds)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	got, err := UnmarshalJSON(b)
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	elem, ok := got.Get(tagPixelData)
	if !ok {
		t.Fatal("missing element after round-trip")
	}
	if _, ok := elem.Value.(*dicom.Bytes); !ok {
		t.Errorf("empty OB decoded as %T, want *dicom.Bytes", elem.Value)
	}
}

func TestMarshalNonFiniteFloatErrors(t *testing.T) {
	tagSlope := dicom.NewTag(0x0028, 0x1053) // RescaleSlope is DS, but use FD here for the value type
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: tagSlope, VR: dicom.VRFD, Value: dicom.NewFloats(dicom.VRFD, math.Inf(1))})

	_, err := MarshalJSON(ds)
	if err == nil {
		t.Fatal("expected an error marshalling a non-finite float")
	}
	var encErr *EncodeError
	if !errors.As(err, &encErr) {
		t.Errorf("error = %v, want *EncodeError", err)
	}
}

func TestUnmarshalRejectsFractionalInteger(t *testing.T) {
	// A fractional JSON number for an integer VR must be rejected, not truncated.
	_, err := UnmarshalJSON([]byte(`{"00280010":{"vr":"US","Value":[1.5]}}`))
	if err == nil {
		t.Fatal("expected an error for a fractional US value")
	}
	var decErr *DecodeError
	if !errors.As(err, &decErr) {
		t.Errorf("error = %v, want *DecodeError", err)
	}
}

func TestMarshalRejectsMalformedFixedWidthBinary(t *testing.T) {
	// The encoder mirrors the decode-side fixed-width check: a 3-byte OW is non-conformant
	// and must fail rather than emit a payload the decoder rejects.
	tagPixelData := dicom.NewTag(0x7FE0, 0x0010)
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: tagPixelData, VR: dicom.VROW, Value: dicom.NewBytes(dicom.VROW, []byte{0x01, 0x02, 0x03})})

	if _, err := MarshalJSON(ds); err == nil {
		t.Fatal("expected an error encoding a 3-byte OW value")
	} else {
		var encErr *EncodeError
		if !errors.As(err, &encErr) {
			t.Errorf("error = %v, want *EncodeError", err)
		}
	}
}

func TestUnmarshalRejectsMalformedFixedWidthBinary(t *testing.T) {
	// OW/OL/OV are fixed-width; a payload whose length is not a multiple of the width is
	// malformed and must be rejected. "AQID" is 3 bytes (0x01 0x02 0x03), odd for OW.
	_, err := UnmarshalJSON([]byte(`{"7FE00010":{"vr":"OW","InlineBinary":"AQID"}}`))
	if err == nil {
		t.Fatal("expected an error for a 3-byte OW payload")
	}
	var decErr *DecodeError
	if !errors.As(err, &decErr) {
		t.Errorf("error = %v, want *DecodeError", err)
	}
}

func TestUnmarshalRejectsMultiplePayloadForms(t *testing.T) {
	// An attribute carrying both InlineBinary and BulkDataURI is malformed per Annex F.
	doc := `{"7FE00010":{"vr":"OB","InlineBinary":"AQID","BulkDataURI":"https://x/y"}}`
	_, err := UnmarshalJSON([]byte(doc))
	if err == nil {
		t.Fatal("expected an error for conflicting payload forms")
	}
	var decErr *DecodeError
	if !errors.As(err, &decErr) {
		t.Errorf("error = %v, want *DecodeError", err)
	}
}

func TestUnmarshalRejectsNullFloat(t *testing.T) {
	_, err := UnmarshalJSON([]byte(`{"00281053":{"vr":"FD","Value":[null]}}`))
	if err == nil {
		t.Fatal("expected an error for a null float value")
	}
	var decErr *DecodeError
	if !errors.As(err, &decErr) {
		t.Errorf("error = %v, want *DecodeError", err)
	}
}

func TestOtherFloatRoundTripsAsBinary(t *testing.T) {
	// OF is a binary VR in Annex F.2.6: it must encode as InlineBinary, not a numeric
	// Value array, and decode back to a *dicom.Floats with the same values.
	tagFloatPixels := dicom.NewTag(0x7FE0, 0x0008) // FloatPixelData (OF)
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: tagFloatPixels, VR: dicom.VROF, Value: dicom.NewFloats(dicom.VROF, 1.5, -2.25)})

	b, err := MarshalJSON(ds)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if !bytes.Contains(b, []byte(`"InlineBinary"`)) {
		t.Fatalf("OF should encode as InlineBinary, got %s", b)
	}
	if bytes.Contains(b, []byte(`"Value"`)) {
		t.Errorf("OF must not encode a numeric Value array, got %s", b)
	}

	got, err := UnmarshalJSON(b)
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	elem, ok := got.Get(tagFloatPixels)
	if !ok {
		t.Fatal("missing OF element after round-trip")
	}
	fv, ok := elem.Value.(*dicom.Floats)
	if !ok {
		t.Fatalf("OF decoded as %T, want *dicom.Floats", elem.Value)
	}
	floats := fv.Floats()
	if len(floats) != 2 || floats[0] != 1.5 || floats[1] != -2.25 {
		t.Errorf("OF values = %v, want [1.5 -2.25]", floats)
	}
}

func TestInlineBinaryOnNonBinaryVRRejected(t *testing.T) {
	_, err := UnmarshalJSON([]byte(`{"00100010":{"vr":"PN","InlineBinary":"AQID"}}`))
	if err == nil {
		t.Fatal("expected an error for InlineBinary on a PN element")
	}
	var decErr *DecodeError
	if !errors.As(err, &decErr) {
		t.Errorf("error = %v, want *DecodeError", err)
	}
}

func TestValueArrayOnBinaryVRRejected(t *testing.T) {
	_, err := UnmarshalJSON([]byte(`{"7FE00010":{"vr":"OB","Value":["AQID"]}}`))
	if err == nil {
		t.Fatal("expected an error for a Value array on an OB element")
	}
	var decErr *DecodeError
	if !errors.As(err, &decErr) {
		t.Errorf("error = %v, want *DecodeError", err)
	}
}

func TestNullStringArrayEntryIsEmptyValue(t *testing.T) {
	// A null array entry represents an absent value within a multi-valued attribute
	// (PS3.18 F.2.5), mapping to an empty string rather than an error.
	got, err := UnmarshalJSON([]byte(`{"00080008":{"vr":"CS","Value":["ORIGINAL",null,"PRIMARY"]}}`))
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	vals, ok := got.GetStrings(dicom.NewTag(0x0008, 0x0008))
	if !ok {
		t.Fatal("missing ImageType after decode")
	}
	want := []string{"ORIGINAL", "", "PRIMARY"}
	if len(vals) != len(want) {
		t.Fatalf("values = %v, want %v", vals, want)
	}
	for i := range want {
		if vals[i] != want[i] {
			t.Errorf("value[%d] = %q, want %q", i, vals[i], want[i])
		}
	}

	// Re-marshalling must emit the null placeholder for the empty interior component so
	// the conformant representation round-trips.
	b, err := MarshalJSON(got)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if !bytes.Contains(b, []byte(`["ORIGINAL",null,"PRIMARY"]`)) {
		t.Errorf("empty interior component should re-marshal as null, got %s", b)
	}
}

func TestSixtyFourBitIntEncodedAsString(t *testing.T) {
	// SV/UV exceed IEEE-754 exact range, so Annex F encodes them as JSON strings; a
	// value above 2^53 must round-trip without precision loss.
	tagSelectorUV := dicom.NewTag(0x0072, 0x007E) // SelectorUVValue (UV)
	const big = int64(9007199254740993)           // 2^53 + 1
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: tagSelectorUV, VR: dicom.VRUV, Value: dicom.NewInts(dicom.VRUV, big)})

	b, err := MarshalJSON(ds)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if !bytes.Contains(b, []byte(`"9007199254740993"`)) {
		t.Fatalf("UV should encode as a quoted string, got %s", b)
	}

	got, err := UnmarshalJSON(b)
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if n, ok := got.GetInt(tagSelectorUV); !ok || n != big {
		t.Errorf("UV round-trip = %d (ok=%v), want %d", n, ok, big)
	}
}

func TestDecodeLargeUnsignedFromString(t *testing.T) {
	// A large UV carried as a string (Annex F) up to the int64-backed model ceiling must
	// decode without being rejected by signed parsing.
	tagSelectorUV := dicom.NewTag(0x0072, 0x007E)
	const big = int64(9223372036854775807) // math.MaxInt64, the model ceiling for UV
	doc := `{"0072007E":{"vr":"UV","Value":["9223372036854775807"]}}`

	got, err := UnmarshalJSON([]byte(doc))
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if n, ok := got.GetInt(tagSelectorUV); !ok || n != big {
		t.Errorf("UV = %d (ok=%v), want %d", n, ok, big)
	}

	// A UV above the int64-backed model range is reported, not silently wrapped.
	if _, err := UnmarshalJSON([]byte(`{"0072007E":{"vr":"UV","Value":["18446744073709551615"]}}`)); err == nil {
		t.Error("expected an error for a UV above the int64 model ceiling")
	}
}

func TestUnmarshalRejectsOutOfRangeUnsigned(t *testing.T) {
	// A negative value for an unsigned VR must be rejected, not silently wrapped on a
	// later binary encode.
	_, err := UnmarshalJSON([]byte(`{"00280010":{"vr":"US","Value":[-1]}}`))
	if err == nil {
		t.Fatal("expected an error for a negative US value")
	}
	var decErr *DecodeError
	if !errors.As(err, &decErr) {
		t.Errorf("error = %v, want *DecodeError", err)
	}

	// A value beyond the VR width is also rejected.
	if _, err := UnmarshalJSON([]byte(`{"00280010":{"vr":"US","Value":[70000]}}`)); err == nil {
		t.Error("expected an error for a US value above 65535")
	}
}

func TestUnmarshalRejectsFractionalIS(t *testing.T) {
	// IS is Integer String: a fractional JSON number must be rejected.
	_, err := UnmarshalJSON([]byte(`{"00200013":{"vr":"IS","Value":[1.5]}}`))
	if err == nil {
		t.Fatal("expected an error for a fractional IS value")
	}
	var decErr *DecodeError
	if !errors.As(err, &decErr) {
		t.Errorf("error = %v, want *DecodeError", err)
	}

	// An integral IS value is still accepted.
	if _, err := UnmarshalJSON([]byte(`{"00200013":{"vr":"IS","Value":[42]}}`)); err != nil {
		t.Errorf("integral IS should decode, got %v", err)
	}
}

func TestUnmarshalRejectsEmptyInlineBinaryOnNonBinaryVR(t *testing.T) {
	// A present but empty InlineBinary on a PN element is malformed and must not slip
	// past the binary-VR guard as if absent.
	_, err := UnmarshalJSON([]byte(`{"00100010":{"vr":"PN","InlineBinary":""}}`))
	if err == nil {
		t.Fatal("expected an error for empty InlineBinary on a PN element")
	}
	var decErr *DecodeError
	if !errors.As(err, &decErr) {
		t.Errorf("error = %v, want *DecodeError", err)
	}
}

func TestTruncatedJSONIsUnexpectedEOF(t *testing.T) {
	// A document cut off mid-object is a truncated transfer, surfaced as a typed error
	// wrapping io.ErrUnexpectedEOF (truncation is failure, PRD §9.2).
	_, err := UnmarshalJSON([]byte(`{"00100010":`))
	if err == nil {
		t.Fatal("expected an error for truncated JSON")
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("error = %v, want errors.Is io.ErrUnexpectedEOF", err)
	}
	var truncErr *TruncatedError
	if !errors.As(err, &truncErr) {
		t.Errorf("error = %v, want *TruncatedError", err)
	}
}

func TestDecodeErrorDoesNotEchoTagKey(t *testing.T) {
	// A malformed tag key is attacker-controlled and could carry PHI; the error must not
	// echo it back (PRD §9.1).
	secret := "PATIENTSECRET123"
	_, err := UnmarshalJSON([]byte(`{"` + secret + `":{"vr":"UI","Value":["1.2.3"]}}`))
	if err == nil {
		t.Fatal("expected an error for a malformed tag key")
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("error message echoes the raw tag key: %q", err.Error())
	}
}

func TestMarshalRejectsAmbiguousVR(t *testing.T) {
	// A dictionary ambiguity placeholder has no valid two-letter DICOM JSON VR; encoding
	// it must fail closed rather than emit a vr the decoder rejects.
	tagPixelData := dicom.NewTag(0x7FE0, 0x0010)
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: tagPixelData, VR: dicom.VROBorOW, Value: dicom.NewBytes(dicom.VROBorOW, nil)})

	if _, err := MarshalJSON(ds); err == nil {
		t.Fatal("expected an error encoding an ambiguous VR")
	} else {
		var encErr *EncodeError
		if !errors.As(err, &encErr) {
			t.Errorf("error = %v, want *EncodeError", err)
		}
	}
}

func TestMarshalRejectsOutOfRangeUnsigned(t *testing.T) {
	// The encoder mirrors the decode-side range check: a negative US is non-conformant
	// and must fail rather than emit invalid JSON.
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: tagRows, VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, -1)})

	_, err := MarshalJSON(ds)
	if err == nil {
		t.Fatal("expected an error encoding a negative US value")
	}
	var encErr *EncodeError
	if !errors.As(err, &encErr) {
		t.Errorf("error = %v, want *EncodeError", err)
	}
}

func TestMarshalRejectsEmptyDecimal(t *testing.T) {
	// A default-initialised (zero-value) Decimal has no lexical form and would marshal to
	// null; the encoder must fail closed rather than emit a non-round-trippable Value.
	tagSlope := dicom.NewTag(0x0028, 0x1053) // RescaleSlope (DS)
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: tagSlope, VR: dicom.VRDS, Value: dicom.NewDecimals(dicom.VRDS, dicom.Decimal{})})

	if _, err := MarshalJSON(ds); err == nil {
		t.Fatal("expected an error encoding a zero-value DS decimal")
	} else {
		var encErr *EncodeError
		if !errors.As(err, &encErr) {
			t.Errorf("error = %v, want *EncodeError", err)
		}
	}
}

func TestMarshalRejectsFractionalIS(t *testing.T) {
	// The encoder mirrors the decode-side IS check: a fractional IS must fail rather than
	// emit non-conformant DICOM JSON.
	tagInstanceNumber := dicom.NewTag(0x0020, 0x0013) // InstanceNumber (IS)
	dec, err := dicom.ParseDecimal("1.5")
	if err != nil {
		t.Fatalf("ParseDecimal: %v", err)
	}
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: tagInstanceNumber, VR: dicom.VRIS, Value: dicom.NewDecimals(dicom.VRIS, dec)})

	if _, err := MarshalJSON(ds); err == nil {
		t.Fatal("expected an error encoding a fractional IS value")
	} else {
		var encErr *EncodeError
		if !errors.As(err, &encErr) {
			t.Errorf("error = %v, want *EncodeError", err)
		}
	}
}

func TestUnknownVRErrorDoesNotEchoValue(t *testing.T) {
	// A bogus vr field is attacker-controlled; the error must not echo it (PRD §9.1).
	bogus := "PHIINVR123"
	_, err := UnmarshalJSON([]byte(`{"00100010":{"vr":"` + bogus + `","Value":["x"]}}`))
	if err == nil {
		t.Fatal("expected an error for an unknown VR")
	}
	if strings.Contains(err.Error(), bogus) {
		t.Errorf("error echoes the raw VR text: %q", err.Error())
	}
}

func TestUnmarshalRejectsEqualsInPNComponentGroup(t *testing.T) {
	// A "=" inside a PN component-group field is the group separator and must be
	// rejected, never re-parsed into a different group layout (PRD §9.2).
	_, err := UnmarshalJSON([]byte(`{"00100010":{"vr":"PN","Value":[{"Alphabetic":"A=B","Ideographic":"C"}]}}`))
	if err == nil {
		t.Fatal("expected an error for '=' inside a PN component group")
	}
	var decErr *DecodeError
	if !errors.As(err, &decErr) {
		t.Errorf("error = %v, want *DecodeError", err)
	}
}

func TestUnmarshalRejectsInvalidJSON(t *testing.T) {
	// A structurally invalid (not merely truncated) document still errors.
	_, err := UnmarshalJSON([]byte(`{"00100010": ]`))
	if err == nil {
		t.Fatal("expected an error for malformed JSON")
	}
}

func TestUnmarshalRejectsExplicitNullValue(t *testing.T) {
	// "Value": null is malformed: an empty element omits Value entirely.
	_, err := UnmarshalJSON([]byte(`{"00100010":{"vr":"PN","Value":null}}`))
	if err == nil {
		t.Fatal("expected an error for an explicit null Value")
	}
	var decErr *DecodeError
	if !errors.As(err, &decErr) {
		t.Errorf("error = %v, want *DecodeError", err)
	}
}

func TestBulkDataURIsUniqueAcrossSequenceItems(t *testing.T) {
	// Two sequence items carrying the same binary tag must get distinct BulkDataURIs so
	// a resolver can tell them apart.
	tagPixelData := dicom.NewTag(0x7FE0, 0x0010)
	tagRefSeq := dicom.NewTag(0x0008, 0x1140)

	item0 := dicom.NewDataSet()
	item0.Set(dicom.Element{Tag: tagPixelData, VR: dicom.VROB, Value: dicom.NewBytes(dicom.VROB, bytes.Repeat([]byte{0x01}, 32))})
	item1 := dicom.NewDataSet()
	item1.Set(dicom.Element{Tag: tagPixelData, VR: dicom.VROB, Value: dicom.NewBytes(dicom.VROB, bytes.Repeat([]byte{0x02}, 32))})

	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: tagRefSeq, VR: dicom.VRSQ, Value: dicom.NewSequenceValue(dicom.NewSequence(item0, item1))})

	b, err := MarshalJSON(ds, WithBulkDataThreshold(16), WithBulkDataBaseURL("https://pacs.example.org/bulk/"))
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	var sq struct {
		Value []struct {
			Pixels struct {
				BulkDataURI string `json:"BulkDataURI"`
			} `json:"7FE00010"`
		} `json:"Value"`
	}
	if err := json.Unmarshal(doc["00081140"], &sq); err != nil {
		t.Fatalf("decode SQ: %v", err)
	}
	if len(sq.Value) != 2 {
		t.Fatalf("expected 2 items, got %d", len(sq.Value))
	}
	u0, u1 := sq.Value[0].Pixels.BulkDataURI, sq.Value[1].Pixels.BulkDataURI
	if u0 == "" || u1 == "" {
		t.Fatalf("both items should have a BulkDataURI, got %q and %q", u0, u1)
	}
	if u0 == u1 {
		t.Errorf("same-tag items got identical BulkDataURIs: %q", u0)
	}
}

func TestDecodeJSONEscapedNumericString(t *testing.T) {
	// A conforming producer may JSON-escape a string-form numeric value. Build the JSON
	// with an explicit + (the escape for "+") so the decoder must resolve the escape
	// before parsing; the DS string then decodes to "+2.5".
	tagSlope := dicom.NewTag(0x0028, 0x1053) // RescaleSlope (DS)
	dsDoc := "{\"00281053\":{\"vr\":\"DS\",\"Value\":[\"\\u002b2.5\"]}}"
	got, err := UnmarshalJSON([]byte(dsDoc))
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	dec, ok := got.GetDecimal(tagSlope)
	if !ok || dec.String() != "+2.5" {
		t.Errorf("RescaleSlope = %q (ok=%v), want +2.5", dec.String(), ok)
	}

	// An escaped digit in a UV string token (1 == "1") decodes to the integer value.
	tagSelectorUV := dicom.NewTag(0x0072, 0x007E)
	uvDoc := "{\"0072007E\":{\"vr\":\"UV\",\"Value\":[\"\\u00310\"]}}"
	gotUV, err := UnmarshalJSON([]byte(uvDoc))
	if err != nil {
		t.Fatalf("UnmarshalJSON UV: %v", err)
	}
	if n, ok := gotUV.GetInt(tagSelectorUV); !ok || n != 10 {
		t.Errorf("UV = %d (ok=%v), want 10", n, ok)
	}
}

func TestSignedPositiveDecimalEncodes(t *testing.T) {
	// A DS lexical form with a leading "+" is valid DICOM but not a valid JSON number;
	// it must still encode (as a string) and round-trip without error.
	tagSlope := dicom.NewTag(0x0028, 0x1053) // RescaleSlope (DS)
	dec, err := dicom.ParseDecimal("+2.5")
	if err != nil {
		t.Fatalf("ParseDecimal: %v", err)
	}
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: tagSlope, VR: dicom.VRDS, Value: dicom.NewDecimals(dicom.VRDS, dec)})

	b, err := MarshalJSON(ds)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if !json.Valid(b) {
		t.Fatalf("output is not valid JSON: %s", b)
	}
	got, err := UnmarshalJSON(b)
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	gotDec, ok := got.GetDecimal(tagSlope)
	if !ok || gotDec.String() != "+2.5" {
		t.Errorf("RescaleSlope = %q (ok=%v), want +2.5", gotDec.String(), ok)
	}
}

func TestUnresolvedBulkRefHasNonZeroEncodedLen(t *testing.T) {
	// A bulkRef must report a non-zero encoded length so the dicom Part-10 writer's
	// written-vs-EncodedLen assertion trips rather than silently emitting an empty
	// element when an unresolved reference reaches the wire.
	got, err := UnmarshalJSON([]byte(`{"7FE00010":{"vr":"OB","BulkDataURI":"https://x/y"}}`))
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	elem, ok := got.Get(dicom.NewTag(0x7FE0, 0x0010))
	if !ok {
		t.Fatal("missing element")
	}
	if n := elem.Value.EncodedLen(binary.LittleEndian); n == 0 {
		t.Errorf("unresolved bulk reference EncodedLen = 0, want non-zero so binary writes fail loudly")
	}
}

func TestUnmarshalRejectsNullDocument(t *testing.T) {
	_, err := UnmarshalJSON([]byte(`null`))
	if err == nil {
		t.Fatal("expected an error for a null document")
	}
	var decErr *DecodeError
	if !errors.As(err, &decErr) {
		t.Errorf("error = %v, want *DecodeError", err)
	}
}

func TestUnmarshalRejectsNullSequenceItem(t *testing.T) {
	_, err := UnmarshalJSON([]byte(`{"00081140":{"vr":"SQ","Value":[null]}}`))
	if err == nil {
		t.Fatal("expected an error for a null sequence item")
	}
	var decErr *DecodeError
	if !errors.As(err, &decErr) {
		t.Errorf("error = %v, want *DecodeError", err)
	}
}

func TestMarshalNilDataSet(t *testing.T) {
	b, err := MarshalJSON(nil)
	if err != nil {
		t.Fatalf("MarshalJSON(nil): %v", err)
	}
	if string(b) != "{}" {
		t.Errorf("MarshalJSON(nil) = %s, want {}", b)
	}
}
