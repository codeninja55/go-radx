package dimse

import (
	"testing"

	"github.com/codeninja55/go-radx/dicom"
)

// TestNGetNDeleteCommandFieldValues pins the N-GET and N-DELETE command field constants to their
// PS3.7 wire values (verified against pynetdicom dimse_messages.py N_GET_RQ/RSP, N_DELETE_RQ/RSP).
func TestNGetNDeleteCommandFieldValues(t *testing.T) {
	cases := []struct {
		name string
		got  CommandField
		want CommandField
	}{
		{"N-GET-RQ", CommandNGetRQ, 0x0110},
		{"N-GET-RSP", CommandNGetRSP, 0x8110},
		{"N-DELETE-RQ", CommandNDeleteRQ, 0x0150},
		{"N-DELETE-RSP", CommandNDeleteRSP, 0x8150},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s = %#04x, want %#04x", tc.name, uint16(tc.got), uint16(tc.want))
		}
	}
}

// TestNGetRQCommandSetRoundTrip builds an N-GET-RQ command set with an Attribute Identifier List,
// encodes it, decodes it, and asserts every field survives — in particular that the AT-VR Attribute
// Identifier List (0000,1005) round-trips as the same list of tags, and that the request uses the
// Requested (not Affected) SOP Class/Instance reference pair.
func TestNGetRQCommandSetRoundTrip(t *testing.T) {
	const sopClass = "1.2.840.10008.5.1.1.40" // Display System SOP Class
	const sopInstance = dicom.UID("1.2.840.10008.5.1.1.40.1")
	attrs := []dicom.Tag{
		dicom.NewTag(0x0008, 0x0070), // Manufacturer
		dicom.NewTag(0x0008, 0x1090), // Manufacturer's Model Name
		dicom.NewTag(0x0018, 0x1020), // Software Versions
	}
	cs := CommandSet{
		CommandField:            CommandNGetRQ,
		MessageID:               42,
		RequestedSOPClassUID:    dicom.UID(sopClass),
		RequestedSOPInstanceUID: sopInstance,
		AttributeIdentifierList: attrs,
		CommandDataSetType:      CommandDataSetNotPresent,
	}

	encoded, err := cs.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	assertIncreasingTagOrder(t, encoded)

	got, err := DecodeCommandSet(encoded)
	if err != nil {
		t.Fatalf("DecodeCommandSet: %v", err)
	}
	if got.CommandField != CommandNGetRQ {
		t.Errorf("CommandField = %#04x, want N-GET-RQ", uint16(got.CommandField))
	}
	if got.IsResponse() {
		t.Error("N-GET-RQ must not have the response bit set")
	}
	if got.MessageID != 42 {
		t.Errorf("MessageID = %d, want 42", got.MessageID)
	}
	if got.RequestedSOPClassUID != dicom.UID(sopClass) {
		t.Errorf("RequestedSOPClassUID = %q, want %q", got.RequestedSOPClassUID, sopClass)
	}
	if got.RequestedSOPInstanceUID != sopInstance {
		t.Errorf("RequestedSOPInstanceUID = %q, want %q", got.RequestedSOPInstanceUID, sopInstance)
	}
	if got.AffectedSOPClassUID != "" || got.AffectedSOPInstanceUID != "" {
		t.Errorf("N-GET-RQ must not carry an Affected pair, got class=%q instance=%q",
			got.AffectedSOPClassUID, got.AffectedSOPInstanceUID)
	}
	if got.CommandDataSetType != CommandDataSetNotPresent {
		t.Errorf("CommandDataSetType = %#04x, want not-present (0x0101) for an N-GET-RQ", got.CommandDataSetType)
	}
	if len(got.AttributeIdentifierList) != len(attrs) {
		t.Fatalf("AttributeIdentifierList length = %d, want %d", len(got.AttributeIdentifierList), len(attrs))
	}
	for i, want := range attrs {
		if got.AttributeIdentifierList[i] != want {
			t.Errorf("AttributeIdentifierList[%d] = %v, want %v", i, got.AttributeIdentifierList[i], want)
		}
	}
}

// TestNGetRQNoAttributeList confirms an N-GET-RQ with no Attribute Identifier List encodes none (it
// is Type 2 / optional: an absent list requests every attribute) and decodes back to a nil list.
func TestNGetRQNoAttributeList(t *testing.T) {
	cs := CommandSet{
		CommandField:            CommandNGetRQ,
		MessageID:               1,
		RequestedSOPClassUID:    "1.2.3",
		RequestedSOPInstanceUID: "1.2.3.4",
		CommandDataSetType:      CommandDataSetNotPresent,
	}
	encoded, err := cs.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	got, err := DecodeCommandSet(encoded)
	if err != nil {
		t.Fatalf("DecodeCommandSet: %v", err)
	}
	if got.AttributeIdentifierList != nil {
		t.Errorf("AttributeIdentifierList = %v, want nil for an N-GET-RQ that named no attributes", got.AttributeIdentifierList)
	}
}

// TestNGetRSPCommandSetRoundTrip builds an N-GET-RSP carrying the Affected reference pair, a status,
// and a data-set-present flag, and asserts the status and the response bit survive the round-trip.
func TestNGetRSPCommandSetRoundTrip(t *testing.T) {
	cs := CommandSet{
		CommandField:              CommandNGetRSP,
		MessageIDBeingRespondedTo: 42,
		AffectedSOPClassUID:       "1.2.840.10008.5.1.1.40",
		AffectedSOPInstanceUID:    "1.2.840.10008.5.1.1.40.1",
		HasStatus:                 true,
		Status:                    StatusNSuccess.Code,
		CommandDataSetType:        CommandDataSetPresent,
	}
	encoded, err := cs.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	got, err := DecodeCommandSet(encoded)
	if err != nil {
		t.Fatalf("DecodeCommandSet: %v", err)
	}
	if !got.IsResponse() {
		t.Error("N-GET-RSP must have the response bit set")
	}
	if got.MessageIDBeingRespondedTo != 42 {
		t.Errorf("MessageIDBeingRespondedTo = %d, want 42", got.MessageIDBeingRespondedTo)
	}
	if !got.HasStatus || got.Status != StatusNSuccess.Code {
		t.Errorf("Status = %#04x (has=%v), want 0x0000", got.Status, got.HasStatus)
	}
	if !got.HasDataSet() {
		t.Error("N-GET-RSP declared a data set follows; HasDataSet should report true")
	}
}

// TestNDeleteCommandSetRoundTrip builds an N-DELETE-RQ and its RSP and asserts the reference pair,
// the response bit, the status, and the no-data-set flag survive the round-trip.
func TestNDeleteCommandSetRoundTrip(t *testing.T) {
	const sopClass = dicom.UID("1.2.840.10008.3.1.2.3.3") // MPPS SOP Class
	const sopInstance = dicom.UID("1.2.826.0.1.3680043.8.498.40000400")

	rq := CommandSet{
		CommandField:            CommandNDeleteRQ,
		MessageID:               9,
		RequestedSOPClassUID:    sopClass,
		RequestedSOPInstanceUID: sopInstance,
		CommandDataSetType:      CommandDataSetNotPresent,
	}
	encoded, err := rq.Encode()
	if err != nil {
		t.Fatalf("Encode RQ: %v", err)
	}
	assertIncreasingTagOrder(t, encoded)
	gotRQ, err := DecodeCommandSet(encoded)
	if err != nil {
		t.Fatalf("DecodeCommandSet RQ: %v", err)
	}
	if gotRQ.CommandField != CommandNDeleteRQ || gotRQ.IsResponse() {
		t.Errorf("RQ CommandField = %#04x (response=%v), want N-DELETE-RQ", uint16(gotRQ.CommandField), gotRQ.IsResponse())
	}
	if gotRQ.RequestedSOPClassUID != sopClass || gotRQ.RequestedSOPInstanceUID != sopInstance {
		t.Errorf("RQ reference pair = (%q,%q), want (%q,%q)",
			gotRQ.RequestedSOPClassUID, gotRQ.RequestedSOPInstanceUID, sopClass, sopInstance)
	}
	if gotRQ.HasDataSet() {
		t.Error("N-DELETE-RQ carries no data set; HasDataSet should report false")
	}

	rsp := CommandSet{
		CommandField:              CommandNDeleteRSP,
		MessageIDBeingRespondedTo: 9,
		AffectedSOPClassUID:       sopClass,
		AffectedSOPInstanceUID:    sopInstance,
		HasStatus:                 true,
		Status:                    StatusNoSuchSOPInstance.Code,
		CommandDataSetType:        CommandDataSetNotPresent,
	}
	encodedRSP, err := rsp.Encode()
	if err != nil {
		t.Fatalf("Encode RSP: %v", err)
	}
	gotRSP, err := DecodeCommandSet(encodedRSP)
	if err != nil {
		t.Fatalf("DecodeCommandSet RSP: %v", err)
	}
	if !gotRSP.IsResponse() {
		t.Error("N-DELETE-RSP must have the response bit set")
	}
	if !gotRSP.HasStatus || gotRSP.Status != StatusNoSuchSOPInstance.Code {
		t.Errorf("RSP Status = %#04x, want No Such SOP Instance (0x0112)", gotRSP.Status)
	}
	if NewStatus(gotRSP.Status, ServiceClassGeneral).IsSuccess() {
		t.Error("a 0x0112 No Such SOP Instance status must never read as Success")
	}
}

// TestDecodeATPartialEntryIgnored confirms a non-conformant Attribute Identifier List whose byte
// length is not a multiple of four drops the trailing partial entry rather than misreading it.
func TestDecodeATPartialEntryIgnored(t *testing.T) {
	// Two full tags (8 bytes) plus 2 trailing bytes: the decoder must yield exactly two tags.
	raw := []byte{0x08, 0x00, 0x70, 0x00, 0x08, 0x00, 0x90, 0x10, 0xAB, 0xCD}
	got := decodeAT(raw)
	if len(got) != 2 {
		t.Fatalf("decodeAT returned %d tags, want 2 (the trailing partial entry dropped)", len(got))
	}
	if got[0] != dicom.NewTag(0x0008, 0x0070) || got[1] != dicom.NewTag(0x0008, 0x1090) {
		t.Errorf("decodeAT tags = %v, want [(0008,0070) (0008,1090)]", got)
	}
}
