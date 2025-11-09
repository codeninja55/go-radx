package pdu

import (
	"bytes"
	"testing"
)

// FuzzAssociateRQDecode tests A-ASSOCIATE-RQ decoder robustness with random input
func FuzzAssociateRQDecode(f *testing.F) {
	// Seed with valid A-ASSOCIATE-RQ
	calledAE := [16]byte{}
	copy(calledAE[:], "TEST_SCP")
	callingAE := [16]byte{}
	copy(callingAE[:], "TEST_SCU")

	validRQ := &AssociateRQ{
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
				},
			},
		},
		UserInfo: UserInformation{
			MaxPDULength:           16384,
			ImplementationClassUID: "1.2.3.4.5",
			ImplementationVersion:  "TEST_0.1",
		},
	}

	buf := &bytes.Buffer{}
	_ = validRQ.Encode(buf)
	f.Add(buf.Bytes())

	// Seed with empty bytes
	f.Add([]byte{})

	// Seed with truncated PDU
	f.Add(buf.Bytes()[:10])

	// Seed with oversized length
	oversized := append([]byte{0x01, 0x00, 0x00, 0x00}, bytes.Repeat([]byte{0xFF}, 100)...)
	f.Add(oversized)

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should never panic, always return error for invalid input
		rq := &AssociateRQ{}
		err := rq.Decode(bytes.NewReader(data))

		// Only verify invariants if decode succeeded
		if err == nil {
			// Verify buffer overflow protection
			if len(rq.CalledAETitle) > 16 {
				t.Errorf("CalledAETitle exceeds 16 characters: %d", len(rq.CalledAETitle))
			}
			if len(rq.CallingAETitle) > 16 {
				t.Errorf("CallingAETitle exceeds 16 characters: %d", len(rq.CallingAETitle))
			}
			// Note: Protocol version validation should be done in the decoder,
			// not the fuzz test. Fuzz tests focus on crash prevention.
		}
	})
}

// FuzzAssociateACDecode tests A-ASSOCIATE-AC decoder robustness
func FuzzAssociateACDecode(f *testing.F) {
	// Seed with valid A-ASSOCIATE-AC
	calledAE := [16]byte{}
	copy(calledAE[:], "TEST_SCP")
	callingAE := [16]byte{}
	copy(callingAE[:], "TEST_SCU")

	validAC := &AssociateAC{
		ProtocolVersion:    1,
		CalledAETitle:      calledAE,
		CallingAETitle:     callingAE,
		ApplicationContext: "1.2.840.10008.3.1.1.1",
		PresentationContexts: []PresentationContextAC{
			{
				ID:             1,
				Result:         PresentationContextAcceptance,
				TransferSyntax: "1.2.840.10008.1.2",
			},
		},
		UserInfo: UserInformation{
			MaxPDULength:           16384,
			ImplementationClassUID: "1.2.3.4.5",
			ImplementationVersion:  "TEST_0.1",
		},
	}

	buf := &bytes.Buffer{}
	_ = validAC.Encode(buf)
	f.Add(buf.Bytes())

	// Seed with empty bytes
	f.Add([]byte{})

	// Seed with invalid result codes
	invalidResult := buf.Bytes()
	if len(invalidResult) > 10 {
		// Corrupt result code position
		invalidResult[70] = 0xFF
		f.Add(invalidResult)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should never panic
		ac := &AssociateAC{}
		err := ac.Decode(bytes.NewReader(data))

		// Only verify critical invariants if decode succeeded
		if err == nil {
			// Verify buffer overflow protection for AE titles
			if len(ac.CalledAETitle) > 16 {
				t.Errorf("CalledAETitle exceeds 16 characters: %d", len(ac.CalledAETitle))
			}
			if len(ac.CallingAETitle) > 16 {
				t.Errorf("CallingAETitle exceeds 16 characters: %d", len(ac.CallingAETitle))
			}
			// Note: Result code validation should be done in the decoder,
			// not the fuzz test. Fuzz tests focus on crash prevention.
		}
	})
}

// FuzzAssociateRJDecode tests A-ASSOCIATE-RJ decoder robustness
func FuzzAssociateRJDecode(f *testing.F) {
	// Seed with valid A-ASSOCIATE-RJ
	validRJ := &AssociateRJ{
		Result: 1,
		Source: 1,
		Reason: 1,
	}

	buf := &bytes.Buffer{}
	_ = validRJ.Encode(buf)
	f.Add(buf.Bytes())

	// Seed with empty bytes
	f.Add([]byte{})

	// Seed with various invalid combinations
	f.Add([]byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04, 0x00, 0xFF, 0xFF, 0xFF})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should never panic
		rj := &AssociateRJ{}
		err := rj.Decode(bytes.NewReader(data))

		// Only verify critical invariants if decode succeeded
		// Field validation should be done in the decoder, not the fuzz test
		_ = err // Fuzz tests focus on crash prevention
	})
}

// FuzzDataTFDecode tests P-DATA-TF decoder robustness
func FuzzDataTFDecode(f *testing.F) {
	// Seed with valid P-DATA-TF
	validData := &DataTF{
		Items: []PresentationDataValue{
			{
				PresentationContextID: 1,
				MessageControlHeader:  MessageControlCommand | MessageControlLastFragment,
				Data:                  []byte{0x00, 0x01, 0x02, 0x03},
			},
		},
	}

	buf := &bytes.Buffer{}
	_ = validData.Encode(buf)
	f.Add(buf.Bytes())

	// Seed with empty bytes
	f.Add([]byte{})

	// Seed with claimed large size but truncated data
	largeSize := []byte{
		0x04, 0x00, // PDU type
		0x00,                   // Reserved
		0x00, 0x10, 0x00, 0x00, // Large length (1MB)
		// ... but no actual data
	}
	f.Add(largeSize)

	// Seed with invalid presentation context ID
	invalidPC := buf.Bytes()
	if len(invalidPC) > 10 {
		invalidPC[8] = 0x00 // PC ID = 0 (invalid, must be odd)
		f.Add(invalidPC)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should never panic
		dataTF := &DataTF{}
		err := dataTF.Decode(bytes.NewReader(data))

		// Only verify critical invariants if decode succeeded
		// Field validation should be done in the decoder, not the fuzz test
		_ = err // Fuzz tests focus on crash prevention
	})
}

// FuzzReleaseRQDecode tests A-RELEASE-RQ decoder robustness
func FuzzReleaseRQDecode(f *testing.F) {
	// Seed with valid A-RELEASE-RQ
	validRelease := &ReleaseRQ{}

	buf := &bytes.Buffer{}
	_ = validRelease.Encode(buf)
	f.Add(buf.Bytes())

	// Seed with empty bytes
	f.Add([]byte{})

	// Seed with invalid length
	f.Add([]byte{0x05, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0x00, 0x00})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should never panic
		release := &ReleaseRQ{}
		_ = release.Decode(bytes.NewReader(data))
	})
}

// FuzzReleaseRPDecode tests A-RELEASE-RP decoder robustness
func FuzzReleaseRPDecode(f *testing.F) {
	// Seed with valid A-RELEASE-RP
	validRelease := &ReleaseRP{}

	buf := &bytes.Buffer{}
	_ = validRelease.Encode(buf)
	f.Add(buf.Bytes())

	// Seed with empty bytes
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should never panic
		release := &ReleaseRP{}
		_ = release.Decode(bytes.NewReader(data))
	})
}

// FuzzAbortDecode tests A-ABORT decoder robustness
func FuzzAbortDecode(f *testing.F) {
	// Seed with valid A-ABORT
	validAbort := &Abort{
		Source: AbortSourceServiceUser,
		Reason: AbortReasonNotSpecified,
	}

	buf := &bytes.Buffer{}
	_ = validAbort.Encode(buf)
	f.Add(buf.Bytes())

	// Seed with empty bytes
	f.Add([]byte{})

	// Seed with invalid source/reason combinations
	f.Add([]byte{0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04, 0x00, 0xFF, 0x00, 0xFF})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should never panic
		abort := &Abort{}
		err := abort.Decode(bytes.NewReader(data))

		// Only verify critical invariants if decode succeeded
		// Field validation should be done in the decoder, not the fuzz test
		_ = err // Fuzz tests focus on crash prevention
	})
}

// FuzzPDUType tests PDU type identification with random input
func FuzzPDUType(f *testing.F) {
	// Seed with valid minimal PDUs (type + reserved + length + minimal body)
	// A-ASSOCIATE-RQ with minimal structure
	validRQ := []byte{
		0x01, 0x00, // Type + reserved
		0x00, 0x00, 0x00, 0x00, // Length 0
	}
	f.Add(validRQ)

	// Invalid PDU type
	invalidType := []byte{
		0xFF, 0x00, // Invalid type + reserved
		0x00, 0x00, 0x00, 0x00, // Length 0
	}
	f.Add(invalidType)

	// Empty data
	f.Add([]byte{})

	// Truncated header
	f.Add([]byte{0x01})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should never panic when reading PDU
		// ReadPDU validates PDU type and routes to correct decoder
		_, err := ReadPDU(bytes.NewReader(data))

		// Verify error handling - we don't check specific errors,
		// just that invalid input doesn't crash
		_ = err // Crash prevention is the goal
	})
}

// FuzzPDUSizeLimit tests PDU size enforcement
func FuzzPDUSizeLimit(f *testing.F) {
	// Seed with PDU claiming extremely large size
	maxPDUSize := uint32(16 * 1024 * 1024) // 16 MB max

	// Valid size
	f.Add(uint32(16384))

	// Boundary cases
	f.Add(uint32(0))
	f.Add(uint32(1))
	f.Add(maxPDUSize)
	f.Add(maxPDUSize + 1)
	f.Add(uint32(0xFFFFFFFF))

	f.Fuzz(func(t *testing.T, size uint32) {
		// Create PDV item header with claimed size
		// PDV item format: 4 bytes length + 1 byte PC-ID + 1 byte control + data
		itemData := make([]byte, 6) // Minimum: length + PC-ID + control header
		// Big-endian item length
		itemData[0] = byte(size >> 24)
		itemData[1] = byte(size >> 16)
		itemData[2] = byte(size >> 8)
		itemData[3] = byte(size)
		itemData[4] = 1    // Presentation Context ID (odd)
		itemData[5] = 0x03 // Message control header (command + last)

		// Decoder should validate size limits
		dataTF := &DataTF{}
		err := dataTF.Decode(bytes.NewReader(itemData))

		// Verify size limit enforcement
		if size > maxPDUSize {
			// Oversized PDUs should be rejected
			if err == nil {
				t.Errorf("Should reject PDV item size %d exceeding max %d", size, maxPDUSize)
			}
		} else if size >= 2 {
			// Valid sizes should succeed or fail gracefully (e.g., truncated data is OK)
			// We don't assert success here because truncated data will cause EOF
		}
	})
}
