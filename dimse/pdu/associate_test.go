package pdu

import (
	"bytes"
	"testing"
)

func TestAssociateRQRoundTrip(t *testing.T) {
	rq := &AssociateRQ{
		ProtocolVersion:    1,
		CalledAETitle:      padAETitle("ORTHANC"),
		CallingAETitle:     padAETitle("GORADX"),
		ApplicationContext: "1.2.840.10008.3.1.1.1",
		PresentationContexts: []PresentationContextRQ{
			{
				ID:             1,
				AbstractSyntax: "1.2.840.10008.1.1", // Verification SOP Class
				TransferSyntaxes: []string{
					"1.2.840.10008.1.2.1", // Explicit VR LE
					"1.2.840.10008.1.2",   // Implicit VR LE
				},
			},
			{
				ID:               3,
				AbstractSyntax:   "1.2.840.10008.5.1.4.1.1.2", // CT Image Storage
				TransferSyntaxes: []string{"1.2.840.10008.1.2.1"},
			},
		},
		UserInfo: UserInformation{
			MaxPDULength:           16382,
			ImplementationClassUID: "1.2.826.0.1.3680043.10.999",
			ImplementationVersion:  "GO-RADX",
		},
	}

	var buf bytes.Buffer
	if err := rq.Encode(&buf); err != nil {
		t.Fatalf("AssociateRQ.Encode: %v", err)
	}
	if buf.Bytes()[0] != byte(PDUTypeAssociateRQ) {
		t.Fatalf("first byte = %#02x, want PDU type A-ASSOCIATE-RQ", buf.Bytes()[0])
	}

	got, err := DecodeAssociateRQ(&buf)
	if err != nil {
		t.Fatalf("DecodeAssociateRQ: %v", err)
	}
	if got.ProtocolVersion != rq.ProtocolVersion {
		t.Errorf("ProtocolVersion = %d, want %d", got.ProtocolVersion, rq.ProtocolVersion)
	}
	if got.CalledAETitle != rq.CalledAETitle || got.CallingAETitle != rq.CallingAETitle {
		t.Errorf("AE titles did not round-trip")
	}
	if got.ApplicationContext != rq.ApplicationContext {
		t.Errorf("ApplicationContext = %q, want %q", got.ApplicationContext, rq.ApplicationContext)
	}
	if len(got.PresentationContexts) != len(rq.PresentationContexts) {
		t.Fatalf("presentation contexts = %d, want %d", len(got.PresentationContexts), len(rq.PresentationContexts))
	}
	for i, want := range rq.PresentationContexts {
		gc := got.PresentationContexts[i]
		if gc.ID != want.ID || gc.AbstractSyntax != want.AbstractSyntax {
			t.Errorf("context %d = (id=%d, abs=%q), want (id=%d, abs=%q)",
				i, gc.ID, gc.AbstractSyntax, want.ID, want.AbstractSyntax)
		}
		if len(gc.TransferSyntaxes) != len(want.TransferSyntaxes) {
			t.Errorf("context %d transfer syntaxes = %d, want %d", i, len(gc.TransferSyntaxes), len(want.TransferSyntaxes))
		}
	}
	if got.UserInfo.MaxPDULength != rq.UserInfo.MaxPDULength {
		t.Errorf("MaxPDULength = %d, want %d", got.UserInfo.MaxPDULength, rq.UserInfo.MaxPDULength)
	}
}

func TestAssociateACRoundTrip(t *testing.T) {
	ac := &AssociateAC{
		ProtocolVersion:    1,
		CalledAETitle:      padAETitle("ORTHANC"),
		CallingAETitle:     padAETitle("GORADX"),
		ApplicationContext: "1.2.840.10008.3.1.1.1",
		PresentationContexts: []PresentationContextAC{
			{ID: 1, Result: PresentationContextAcceptance, TransferSyntax: "1.2.840.10008.1.2.1"},
			{ID: 3, Result: PresentationContextAbstractSyntaxNotSupported, TransferSyntax: "1.2.840.10008.1.2"},
		},
		UserInfo: UserInformation{MaxPDULength: 16382},
	}

	var buf bytes.Buffer
	if err := ac.Encode(&buf); err != nil {
		t.Fatalf("AssociateAC.Encode: %v", err)
	}
	got, err := DecodeAssociateAC(&buf)
	if err != nil {
		t.Fatalf("DecodeAssociateAC: %v", err)
	}
	if len(got.PresentationContexts) != 2 {
		t.Fatalf("presentation contexts = %d, want 2", len(got.PresentationContexts))
	}
	for i, want := range ac.PresentationContexts {
		gc := got.PresentationContexts[i]
		if gc.ID != want.ID || gc.Result != want.Result {
			t.Errorf("AC context %d = (id=%d, result=%d), want (id=%d, result=%d)",
				i, gc.ID, gc.Result, want.ID, want.Result)
		}
	}
}

// TestAssociateACRejectedContextKeepsOneTransferSyntax guards Codex DIMSE-008: a
// rejected presentation context in an A-ASSOCIATE-AC must still encode exactly one
// (insignificant) transfer-syntax sub-item. The prototype omitted it for non-accepted
// results, which produced a malformed AC body.
func TestAssociateACRejectedContextKeepsOneTransferSyntax(t *testing.T) {
	ac := &AssociateAC{
		ProtocolVersion:    1,
		CalledAETitle:      padAETitle("ORTHANC"),
		CallingAETitle:     padAETitle("GORADX"),
		ApplicationContext: "1.2.840.10008.3.1.1.1",
		PresentationContexts: []PresentationContextAC{
			{ID: 5, Result: PresentationContextTransferSyntaxesNotSupported}, // no TS set
		},
		UserInfo: UserInformation{MaxPDULength: 16382},
	}

	var buf bytes.Buffer
	if err := ac.Encode(&buf); err != nil {
		t.Fatalf("AssociateAC.Encode: %v", err)
	}
	// Count transfer-syntax sub-items (item type 0x40) in the encoded body.
	body := buf.Bytes()[6:] // strip the 6-byte PDU header
	tsCount := bytes.Count(body, []byte{ItemTypeTransferSyntax, 0x00})
	if tsCount != 1 {
		t.Fatalf("rejected context encoded %d transfer-syntax sub-items, want exactly 1 (DIMSE-008)", tsCount)
	}

	got, err := DecodeAssociateAC(&buf)
	if err != nil {
		t.Fatalf("DecodeAssociateAC of rejected context: %v", err)
	}
	if len(got.PresentationContexts) != 1 {
		t.Fatalf("decoded contexts = %d, want 1", len(got.PresentationContexts))
	}
	if got.PresentationContexts[0].TransferSyntax == "" {
		t.Error("decoded rejected context has no transfer syntax; the insignificant sub-item was lost (DIMSE-008)")
	}
}

func TestAssociateRJRoundTrip(t *testing.T) {
	rj := &AssociateRJ{
		Result: AssociateRJResultPermanent,
		Source: AssociateRJSourceServiceUser,
		Reason: 2,
	}
	var buf bytes.Buffer
	if err := rj.Encode(&buf); err != nil {
		t.Fatalf("AssociateRJ.Encode: %v", err)
	}
	if buf.Bytes()[0] != byte(PDUTypeAssociateRJ) {
		t.Fatalf("first byte = %#02x, want A-ASSOCIATE-RJ", buf.Bytes()[0])
	}
	got, err := DecodeAssociateRJ(&buf)
	if err != nil {
		t.Fatalf("DecodeAssociateRJ: %v", err)
	}
	if got.Result != rj.Result || got.Source != rj.Source || got.Reason != rj.Reason {
		t.Errorf("RJ round-trip = %+v, want %+v", got, rj)
	}
}

func TestAETitlePadTrim(t *testing.T) {
	for _, s := range []string{"A", "ORTHANC", "1234567890123456"} {
		padded := padAETitle(s)
		if len(padded) != 16 {
			t.Fatalf("padAETitle(%q) length = %d, want 16", s, len(padded))
		}
		if got := trimAETitle(padded); got != s {
			t.Errorf("trimAETitle(padAETitle(%q)) = %q", s, got)
		}
	}
}

// TestDecodeAssociateRQRejectsTruncatedItem guards against a sub-item length exceeding
// the remaining body bytes (a hostile A-ASSOCIATE-RQ).
func TestDecodeAssociateRQRejectsTruncatedItem(t *testing.T) {
	// A valid-looking header (protocol version + reserved + two AE titles + 32 reserved),
	// then an application-context item claiming a huge length with no payload.
	called := padAETitle("CALLED")
	calling := padAETitle("CALLING")
	var body bytes.Buffer
	body.Write([]byte{0x00, 0x01, 0x00, 0x00}) // protocol version + reserved
	body.Write(called[:])
	body.Write(calling[:])
	body.Write(make([]byte, 32))                                     // reserved
	body.Write([]byte{ItemTypeApplicationContext, 0x00, 0xFF, 0xF0}) // length 65520, no payload

	var pduBuf bytes.Buffer
	if err := writeHeader(&pduBuf, PDUTypeAssociateRQ, uint32(body.Len())); err != nil {
		t.Fatal(err)
	}
	pduBuf.Write(body.Bytes())
	if _, err := DecodeAssociateRQ(&pduBuf); err == nil {
		t.Error("DecodeAssociateRQ should reject a sub-item length exceeding the body")
	}
}
