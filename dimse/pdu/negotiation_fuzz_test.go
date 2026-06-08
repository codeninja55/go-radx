package pdu

import (
	"bytes"
	"testing"
)

// FuzzDecodeAsyncOperations drives the 0x53 sub-item decoder with arbitrary bytes. A body shorter
// than the two required counts must return an error, never panic (PRD §9.3).
func FuzzDecodeAsyncOperations(f *testing.F) {
	f.Add(subItemBodyHelper(func(b *bytes.Buffer) {
		encodeAsyncOperations(b, AsyncOperations{MaxOperationsInvoked: 1, MaxOperationsPerformed: 1})
	}))
	f.Add([]byte{})
	f.Add([]byte{0x00, 0x01})
	f.Add([]byte{0x00, 0x01, 0x00})
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = decodeAsyncOperations(data)
	})
}

// FuzzDecodeExtendedNegotiation drives the 0x56 sub-item decoder with arbitrary bytes. A declared
// UID length past the bytes present must return an error, never over-read (PRD §9.3).
func FuzzDecodeExtendedNegotiation(f *testing.F) {
	f.Add(subItemBodyHelper(func(b *bytes.Buffer) {
		encodeExtendedNegotiation(b, ExtendedNegotiation{SOPClassUID: "1.2.840.10008.5.1.4.1.1.2", ServiceClassAppInfo: []byte{0x01}})
	}))
	f.Add([]byte{})
	f.Add([]byte{0x00, 0x40, 0x31, 0x32}) // huge declared UID length
	f.Add([]byte{0x00, 0x00})             // zero-length UID, no app info
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = decodeExtendedNegotiation(data)
	})
}

// FuzzDecodeCommonExtendedNegotiation drives the 0x57 sub-item decoder with arbitrary bytes. Any
// declared length running past the buffer must return an error, never panic (PRD §9.3).
func FuzzDecodeCommonExtendedNegotiation(f *testing.F) {
	f.Add(subItemBodyHelper(func(b *bytes.Buffer) {
		encodeCommonExtendedNegotiation(b, CommonExtendedNegotiation{
			SOPClassUID:              "1.2.840.10008.5.1.4.1.1.2",
			ServiceClassUID:          "1.2.840.10008.4.2",
			RelatedGeneralSOPClasses: []string{"1.2.840.10008.5.1.4.1.1.4"},
		})
	}))
	f.Add([]byte{})
	f.Add([]byte{0x00, 0x40, 0x31})                               // SOP Class length past body
	f.Add([]byte{0x00, 0x01, 0x31, 0x00, 0x01, 0x32, 0x00, 0x40}) // related length past body
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = decodeCommonExtendedNegotiation(data)
	})
}

// FuzzDecodeUserIdentityRQ drives the 0x58 sub-item decoder with arbitrary bytes. A truncated type,
// flag, or length-prefixed field must return an error, never panic or over-read (PRD §9.3).
func FuzzDecodeUserIdentityRQ(f *testing.F) {
	f.Add(subItemBodyHelper(func(b *bytes.Buffer) {
		encodeUserIdentityRQ(b, UserIdentityRQ{Type: UserIdentityUsernamePasscode, PrimaryField: []byte("alice"), SecondaryField: []byte("pw"), PositiveResponseRequested: true})
	}))
	f.Add([]byte{})
	f.Add([]byte{0x01})                         // type only
	f.Add([]byte{0x01, 0x00, 0x00, 0x05, 0x31}) // primary length past body
	f.Add([]byte{0x02, 0x01, 0x00, 0x01, 0x41}) // primary ok, no secondary length
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = decodeUserIdentityRQ(data)
	})
}

// FuzzDecodeUserIdentityAC drives the 0x59 sub-item decoder with arbitrary bytes. A declared
// server-response length past the bytes present must return an error, never over-read (PRD §9.3).
func FuzzDecodeUserIdentityAC(f *testing.F) {
	f.Add(subItemBodyHelper(func(b *bytes.Buffer) {
		encodeUserIdentityAC(b, UserIdentityAC{ServerResponse: []byte{0x01, 0x02}})
	}))
	f.Add([]byte{})
	f.Add([]byte{0x00})             // partial length
	f.Add([]byte{0x00, 0x05, 0x31}) // length 5, one byte present
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = decodeUserIdentityAC(data)
	})
}

// subItemBodyHelper encodes one sub-item and returns its body (the bytes after the 4-byte sub-item
// header), matching what the per-sub-item decoders receive. The signature avoids *testing.T so it
// can run inside an f.Add seed expression.
func subItemBodyHelper(encode func(*bytes.Buffer)) []byte {
	var buf bytes.Buffer
	encode(&buf)
	return buf.Bytes()[4:]
}
