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
