package pdu

import (
	"bytes"
	"io"
	"testing"
)

// benchDataTF builds a P-DATA-TF carrying a single 16 KiB dataset fragment — a typical maximum-PDU
// payload. A C-STORE streams many such PDUs, so the P-DATA-TF encode/decode is the DIMSE wire hot
// path and the right first benchmark target (PRD §9.3 minimise allocations in hot paths).
func benchDataTF() *DataTF {
	payload := bytes.Repeat([]byte{0xAB}, 16*1024)
	return &DataTF{Items: []PresentationDataValue{{
		PresentationContextID: 1,
		MessageControlHeader:  MakeControlHeader(false, true), // final dataset fragment
		Data:                  payload,
	}}}
}

// BenchmarkDataTFEncode measures encoding one P-DATA-TF PDU (the per-fragment C-STORE wire hot path).
func BenchmarkDataTFEncode(b *testing.B) {
	p := benchDataTF()
	var sized bytes.Buffer
	_ = p.Encode(&sized)
	b.SetBytes(int64(sized.Len()))
	b.ReportAllocs()
	for b.Loop() {
		if err := p.Encode(io.Discard); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDataTFDecode measures decoding one P-DATA-TF PDU body off the wire.
func BenchmarkDataTFDecode(b *testing.B) {
	var buf bytes.Buffer
	if err := benchDataTF().Encode(&buf); err != nil {
		b.Fatalf("pre-encode: %v", err)
	}
	encoded := buf.Bytes()
	const headerLen = 6 // P-DATA-TF fixed header: type(1) + reserved(1) + length(4)
	body := encoded[headerLen:]
	b.SetBytes(int64(len(encoded)))
	b.ReportAllocs()
	for b.Loop() {
		var d DataTF
		if err := d.Decode(newBoundedReader(bytes.NewReader(body), int64(len(body)))); err != nil {
			b.Fatal(err)
		}
	}
}
