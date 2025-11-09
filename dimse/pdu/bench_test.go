package pdu

import (
	"bytes"
	"testing"
)

// BenchmarkAssociateRQ_Encode benchmarks A-ASSOCIATE-RQ encoding
func BenchmarkAssociateRQ_Encode(b *testing.B) {
	calledAE := [16]byte{}
	copy(calledAE[:], "TEST_SCP")
	callingAE := [16]byte{}
	copy(callingAE[:], "TEST_SCU")

	rq := &AssociateRQ{
		ProtocolVersion:    1,
		CalledAETitle:      calledAE,
		CallingAETitle:     callingAE,
		ApplicationContext: "1.2.840.10008.3.1.1.1",
		PresentationContexts: []PresentationContextRQ{
			{
				ID:             1,
				AbstractSyntax: "1.2.840.10008.1.1",
				TransferSyntaxes: []string{
					"1.2.840.10008.1.2",
					"1.2.840.10008.1.2.1",
				},
			},
			{
				ID:             3,
				AbstractSyntax: "1.2.840.10008.5.1.4.1.1.2",
				TransferSyntaxes: []string{
					"1.2.840.10008.1.2",
				},
			},
		},
		UserInfo: UserInformation{
			MaxPDULength:           16384,
			ImplementationClassUID: "1.2.840.12345",
			ImplementationVersion:  "TEST_1.0",
		},
	}

	buf := &bytes.Buffer{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = rq.Encode(buf)
	}
}

// BenchmarkAssociateRQ_Decode benchmarks A-ASSOCIATE-RQ decoding
func BenchmarkAssociateRQ_Decode(b *testing.B) {
	calledAE := [16]byte{}
	copy(calledAE[:], "TEST_SCP")
	callingAE := [16]byte{}
	copy(callingAE[:], "TEST_SCU")

	rq := &AssociateRQ{
		ProtocolVersion:    1,
		CalledAETitle:      calledAE,
		CallingAETitle:     callingAE,
		ApplicationContext: "1.2.840.10008.3.1.1.1",
		PresentationContexts: []PresentationContextRQ{
			{
				ID:               1,
				AbstractSyntax:   "1.2.840.10008.1.1",
				TransferSyntaxes: []string{"1.2.840.10008.1.2"},
			},
		},
		UserInfo: UserInformation{
			MaxPDULength:           16384,
			ImplementationClassUID: "1.2.840.12345",
		},
	}

	buf := &bytes.Buffer{}
	_ = rq.Encode(buf)
	encoded := buf.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		decoded := &AssociateRQ{}
		_ = decoded.Decode(bytes.NewReader(encoded))
	}
}

// BenchmarkDataTF_Encode benchmarks P-DATA-TF encoding with various payload sizes
func BenchmarkDataTF_Encode(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"Small_1KB", 1024},
		{"Medium_16KB", 16384},
		{"Large_64KB", 65536},
	}

	for _, sz := range sizes {
		b.Run(sz.name, func(b *testing.B) {
			data := bytes.Repeat([]byte{0x00, 0x01, 0x02, 0x03}, sz.size/4)
			pdu := &DataTF{
				Items: []PresentationDataValue{
					{
						PresentationContextID: 1,
						MessageControlHeader:  MessageControlCommand | MessageControlLastFragment,
						Data:                  data,
					},
				},
			}

			buf := &bytes.Buffer{}

			b.ResetTimer()
			b.SetBytes(int64(sz.size))
			for i := 0; i < b.N; i++ {
				buf.Reset()
				_ = pdu.Encode(buf)
			}
		})
	}
}

// BenchmarkDataTF_Decode benchmarks P-DATA-TF decoding with various payload sizes
func BenchmarkDataTF_Decode(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"Small_1KB", 1024},
		{"Medium_16KB", 16384},
		{"Large_64KB", 65536},
	}

	for _, sz := range sizes {
		b.Run(sz.name, func(b *testing.B) {
			data := bytes.Repeat([]byte{0x00, 0x01, 0x02, 0x03}, sz.size/4)
			pdu := &DataTF{
				Items: []PresentationDataValue{
					{
						PresentationContextID: 1,
						MessageControlHeader:  MessageControlDatasetLast,
						Data:                  data,
					},
				},
			}

			buf := &bytes.Buffer{}
			_ = pdu.Encode(buf)
			encoded := buf.Bytes()

			b.ResetTimer()
			b.SetBytes(int64(sz.size))
			for i := 0; i < b.N; i++ {
				decoded := &DataTF{}
				_ = decoded.Decode(bytes.NewReader(encoded))
			}
		})
	}
}

// BenchmarkPDU_RoundTrip benchmarks full encode/decode round trip
func BenchmarkPDU_RoundTrip(b *testing.B) {
	calledAE := [16]byte{}
	copy(calledAE[:], "TEST_SCP")
	callingAE := [16]byte{}
	copy(callingAE[:], "TEST_SCU")

	rq := &AssociateRQ{
		ProtocolVersion:    1,
		CalledAETitle:      calledAE,
		CallingAETitle:     callingAE,
		ApplicationContext: "1.2.840.10008.3.1.1.1",
		PresentationContexts: []PresentationContextRQ{
			{
				ID:               1,
				AbstractSyntax:   "1.2.840.10008.1.1",
				TransferSyntaxes: []string{"1.2.840.10008.1.2"},
			},
		},
		UserInfo: UserInformation{
			MaxPDULength: 16384,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Encode
		buf := &bytes.Buffer{}
		_ = rq.Encode(buf)

		// Decode
		decoded := &AssociateRQ{}
		_ = decoded.Decode(bytes.NewReader(buf.Bytes()))
	}
}

// BenchmarkPDUType_Identification benchmarks PDU type identification
func BenchmarkPDUType_Identification(b *testing.B) {
	pduTypes := []byte{
		PDUTypeAssociateRQ,
		PDUTypeAssociateAC,
		PDUTypeAssociateRJ,
		PDUTypeData,
		PDUTypeReleaseRQ,
		PDUTypeReleaseRP,
		PDUTypeAbort,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pduType := pduTypes[i%len(pduTypes)]
		_ = pduType >= PDUTypeAssociateRQ && pduType <= PDUTypeAbort
	}
}
