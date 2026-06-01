package pdu

import (
	"bytes"
	"testing"
)

func FuzzDecodePDV(f *testing.F) {
	f.Add([]byte{0x00, 0x00, 0x00, 0x06, 0x01, 0x03, 0xDE, 0xAD, 0xBE, 0xEF})
	f.Add([]byte{0x00, 0x00, 0x00, 0x00}) // underflow item length
	f.Add([]byte{0x00, 0x00, 0x00, 0x01}) // underflow item length
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF}) // huge declared length, no body
	f.Fuzz(func(t *testing.T, data []byte) {
		br := newBoundedReader(bytes.NewReader(data), int64(len(data)))
		// Must never panic; an error is the acceptable outcome for malformed input.
		_, _ = decodePDV(br)
	})
}
