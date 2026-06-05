package pdu

import (
	"bytes"
	"testing"
)

// seedRoleSelection returns a well-formed SCP/SCU Role Selection sub-item body for the fuzz
// corpus (the bytes after the 4-byte sub-item header, as decodeRoleSelection receives them).
func seedRoleSelection() []byte {
	var buf bytes.Buffer
	encodeRoleSelection(&buf, RoleSelection{SOPClassUID: "1.2.840.10008.5.1.4.1.1.2", SCURole: true, SCPRole: true})
	// Strip the 4-byte sub-item header; decodeRoleSelection works on the body alone.
	return buf.Bytes()[4:]
}

// FuzzDecodeRoleSelection drives the 0x54 sub-item body decoder with arbitrary bytes. A
// malformed body whose declared UID length runs past the bytes present, or which is missing
// the trailing role flags, must return an error, never panic or over-read (PRD §9.3).
func FuzzDecodeRoleSelection(f *testing.F) {
	f.Add(seedRoleSelection())
	f.Add([]byte{})                       // empty body
	f.Add([]byte{0x00, 0x05})             // UID length 5, no UID or flags
	f.Add([]byte{0xFF, 0xFF, 0x31, 0x32}) // huge declared UID length
	f.Add([]byte{0x00, 0x01, 0x31})       // UID present, role flags missing
	f.Add([]byte{0x00, 0x00, 0x01, 0x00}) // zero-length UID, both flags present
	f.Fuzz(func(t *testing.T, data []byte) {
		// Must never panic; an error is the acceptable outcome for malformed input.
		_, _ = decodeRoleSelection(data)
	})
}
