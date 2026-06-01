package pdu

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestReadPDUDispatchesEachType(t *testing.T) {
	rq := &AssociateRQ{
		ProtocolVersion:      1,
		CalledAETitle:        padAETitle("CALLED"),
		CallingAETitle:       padAETitle("CALLING"),
		ApplicationContext:   "1.2.840.10008.3.1.1.1",
		PresentationContexts: []PresentationContextRQ{{ID: 1, AbstractSyntax: "1.2.840.10008.1.1", TransferSyntaxes: []string{"1.2.840.10008.1.2"}}},
		UserInfo:             UserInformation{MaxPDULength: 16382},
	}
	ac := &AssociateAC{
		ProtocolVersion:      1,
		CalledAETitle:        padAETitle("CALLED"),
		CallingAETitle:       padAETitle("CALLING"),
		ApplicationContext:   "1.2.840.10008.3.1.1.1",
		PresentationContexts: []PresentationContextAC{{ID: 1, Result: PresentationContextAcceptance, TransferSyntax: "1.2.840.10008.1.2"}},
		UserInfo:             UserInformation{MaxPDULength: 16382},
	}
	data := &DataTF{Items: []PresentationDataValue{{PresentationContextID: 1, MessageControlHeader: 0x03, Data: []byte{1, 2, 3}}}}

	cases := []struct {
		name string
		pdu  PDU
		want PDUType
	}{
		{"A-ASSOCIATE-RQ", rq, PDUTypeAssociateRQ},
		{"A-ASSOCIATE-AC", ac, PDUTypeAssociateAC},
		{"A-ASSOCIATE-RJ", &AssociateRJ{Result: 1, Source: 1, Reason: 2}, PDUTypeAssociateRJ},
		{"P-DATA-TF", data, PDUTypeData},
		{"A-RELEASE-RQ", &ReleaseRQ{}, PDUTypeReleaseRQ},
		{"A-RELEASE-RP", &ReleaseRP{}, PDUTypeReleaseRP},
		{"A-ABORT", &Abort{Source: 2, Reason: 5}, PDUTypeAbort},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WritePDU(&buf, c.pdu); err != nil {
				t.Fatalf("WritePDU: %v", err)
			}
			got, err := ReadPDU(&buf)
			if err != nil {
				t.Fatalf("ReadPDU: %v", err)
			}
			if got.Type() != c.want {
				t.Errorf("ReadPDU returned type %v, want %v", got.Type(), c.want)
			}
		})
	}
}

// TestReadPDURejectsOversizedFixedBody guards against a stream desync on the fixed-size
// PDUs (A-ASSOCIATE-RJ, A-ABORT, A-RELEASE-RQ, A-RELEASE-RP). Each carries exactly four
// body bytes. The prototype read only the four fixed bytes and returned success even when
// the declared body length was larger, leaving the surplus in the stream so the NEXT PDU
// was parsed at the wrong offset. ReadPDU must reject the oversized body, and a valid PDU
// that follows in the same stream must still parse correctly.
func TestReadPDURejectsOversizedFixedBody(t *testing.T) {
	cases := []struct {
		name string
		pt   PDUType
	}{
		{"A-ASSOCIATE-RJ", PDUTypeAssociateRJ},
		{"A-ABORT", PDUTypeAbort},
		{"A-RELEASE-RQ", PDUTypeReleaseRQ},
		{"A-RELEASE-RP", PDUTypeReleaseRP},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stream bytes.Buffer
			// A fixed-body PDU declaring eight body bytes (four surplus) instead of four.
			if err := writeHeader(&stream, c.pt, 8); err != nil {
				t.Fatal(err)
			}
			stream.Write(make([]byte, 8))

			// A well-formed PDU immediately after it, to expose any desync.
			next := &AssociateRJ{Result: AssociateRJResultPermanent, Source: AssociateRJSourceServiceUser, Reason: 2}
			if err := WritePDU(&stream, next); err != nil {
				t.Fatal(err)
			}

			if _, err := ReadPDU(&stream); err == nil {
				t.Fatalf("ReadPDU should reject a %s with a declared body length of 8 (fixed body is 4)", c.name)
			}

			// The reader must be positioned exactly at the start of the next PDU's header.
			got, err := ReadPDU(&stream)
			if err != nil {
				t.Fatalf("subsequent PDU did not parse after the oversized %s (stream desync): %v", c.name, err)
			}
			if got.Type() != PDUTypeAssociateRJ {
				t.Errorf("subsequent PDU type = %v, want A-ASSOCIATE-RJ (stream desync after %s)", got.Type(), c.name)
			}
		})
	}
}

func TestReadPDUReturnsEOFAtBoundary(t *testing.T) {
	if _, err := ReadPDU(bytes.NewReader(nil)); !errors.Is(err, io.EOF) {
		t.Errorf("ReadPDU on empty stream = %v, want io.EOF", err)
	}
}

func TestReadPDURejectsOversizedLength(t *testing.T) {
	var h [6]byte
	h[0] = byte(PDUTypeData)
	// Declared length above MaxPDULength must be rejected at the header, before allocation.
	h[2], h[3], h[4], h[5] = 0xFF, 0xFF, 0xFF, 0xFF
	if _, err := ReadPDU(bytes.NewReader(h[:])); err == nil {
		t.Error("ReadPDU should reject a declared body length above MaxPDULength")
	}
}

func TestReadPDURejectsUnknownType(t *testing.T) {
	if _, err := ReadPDU(bytes.NewReader([]byte{0x09, 0x00, 0x00, 0x00, 0x00, 0x00})); err == nil {
		t.Error("ReadPDU should reject an unknown PDU type")
	}
}
