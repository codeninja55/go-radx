package pdu

import (
	"bytes"
	"testing"
)

// seedAssociateRQ returns a minimal well-formed A-ASSOCIATE-RQ for the fuzz corpus.
func seedAssociateRQ() []byte {
	rq := &AssociateRQ{
		ProtocolVersion:      1,
		CalledAETitle:        padAETitle("CALLED"),
		CallingAETitle:       padAETitle("CALLING"),
		ApplicationContext:   "1.2.840.10008.3.1.1.1",
		PresentationContexts: []PresentationContextRQ{{ID: 1, AbstractSyntax: "1.2.840.10008.1.1", TransferSyntaxes: []string{"1.2.840.10008.1.2"}}},
		UserInfo:             UserInformation{MaxPDULength: 16382},
	}
	var buf bytes.Buffer
	_ = rq.Encode(&buf)
	return buf.Bytes()
}

// FuzzReadPDU drives the whole-PDU dispatch with arbitrary bytes. A malformed PDU must
// return an error, never panic or over-allocate (PRD §9.3).
func FuzzReadPDU(f *testing.F) {
	f.Add(seedAssociateRQ())
	f.Add([]byte{0x01, 0x00, 0xFF, 0xFF, 0xFF, 0xFF}) // A-ASSOCIATE-RQ, huge length, no body
	f.Add([]byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x04}) // A-ASSOCIATE-RJ header, no body
	f.Add([]byte{0x07, 0x00, 0x00, 0x00, 0x00, 0x04}) // A-ABORT header, no body
	f.Add([]byte{0x05, 0x00, 0x00, 0x00, 0x00, 0x00}) // A-RELEASE-RQ, zero-length body
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ReadPDU(bytes.NewReader(data))
	})
}

// FuzzDecodeAssociateAC targets the AC decoder, whose nested presentation-context and
// user-information sub-items have their own bounded length math.
func FuzzDecodeAssociateAC(f *testing.F) {
	ac := &AssociateAC{
		ProtocolVersion:      1,
		CalledAETitle:        padAETitle("CALLED"),
		CallingAETitle:       padAETitle("CALLING"),
		ApplicationContext:   "1.2.840.10008.3.1.1.1",
		PresentationContexts: []PresentationContextAC{{ID: 1, Result: PresentationContextAcceptance, TransferSyntax: "1.2.840.10008.1.2"}},
		UserInfo:             UserInformation{MaxPDULength: 16382},
	}
	var buf bytes.Buffer
	_ = ac.Encode(&buf)
	f.Add(buf.Bytes())
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = DecodeAssociateAC(bytes.NewReader(data))
	})
}
