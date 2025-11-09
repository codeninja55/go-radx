package pdu_test

import (
	"bytes"
	"testing"

	"github.com/codeninja55/go-radx/dimse/pdu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAssociateRQ_EncodeDecode tests A-ASSOCIATE-RQ encoding and decoding
func TestAssociateRQ_EncodeDecode(t *testing.T) {
	original := &pdu.AssociateRQ{
		ProtocolVersion:    0x0001,
		CalledAETitle:      pdu.PadAETitle("CALLED_AE"),
		CallingAETitle:     pdu.PadAETitle("CALLING_AE"),
		ApplicationContext: "1.2.840.10008.3.1.1.1",
		PresentationContexts: []pdu.PresentationContextRQ{
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
		UserInfo: pdu.UserInformation{
			MaxPDULength:           16384,
			ImplementationClassUID: "1.2.840.12345.1.1",
			ImplementationVersion:  "GO-RADX_1.0",
		},
	}

	// Encode
	var buf bytes.Buffer
	err := original.Encode(&buf)
	require.NoError(t, err)

	// Verify PDU type
	data := buf.Bytes()
	assert.Equal(t, pdu.PDUTypeAssociateRQ, data[0])

	// Decode
	decoded := &pdu.AssociateRQ{}
	err = decoded.Decode(bytes.NewReader(data[6:])) // Skip header
	require.NoError(t, err)

	// Verify fields
	assert.Equal(t, original.ProtocolVersion, decoded.ProtocolVersion)
	assert.Equal(t, original.CalledAETitle, decoded.CalledAETitle)
	assert.Equal(t, original.CallingAETitle, decoded.CallingAETitle)
	assert.Equal(t, original.ApplicationContext, decoded.ApplicationContext)
	assert.Len(t, decoded.PresentationContexts, len(original.PresentationContexts))
	assert.Equal(t, original.UserInfo.MaxPDULength, decoded.UserInfo.MaxPDULength)
}

// TestAssociateAC_EncodeDecode tests A-ASSOCIATE-AC encoding and decoding
func TestAssociateAC_EncodeDecode(t *testing.T) {
	original := &pdu.AssociateAC{
		ProtocolVersion:    0x0001,
		CalledAETitle:      pdu.PadAETitle("CALLED_AE"),
		CallingAETitle:     pdu.PadAETitle("CALLING_AE"),
		ApplicationContext: "1.2.840.10008.3.1.1.1",
		PresentationContexts: []pdu.PresentationContextAC{
			{
				ID:             1,
				Result:         pdu.PresentationContextAcceptance,
				TransferSyntax: "1.2.840.10008.1.2",
			},
			{
				ID:             3,
				Result:         pdu.PresentationContextAcceptance,
				TransferSyntax: "1.2.840.10008.1.2",
			},
		},
		UserInfo: pdu.UserInformation{
			MaxPDULength:           16384,
			ImplementationClassUID: "1.2.840.12345.1.1",
			ImplementationVersion:  "GO-RADX_1.0",
		},
	}

	// Encode
	var buf bytes.Buffer
	err := original.Encode(&buf)
	require.NoError(t, err)

	// Verify PDU type
	data := buf.Bytes()
	assert.Equal(t, pdu.PDUTypeAssociateAC, data[0])

	// Decode
	decoded := &pdu.AssociateAC{}
	err = decoded.Decode(bytes.NewReader(data[6:]))
	require.NoError(t, err)

	// Verify fields
	assert.Equal(t, original.ProtocolVersion, decoded.ProtocolVersion)
	assert.Len(t, decoded.PresentationContexts, len(original.PresentationContexts))
	for i, pc := range decoded.PresentationContexts {
		assert.Equal(t, original.PresentationContexts[i].ID, pc.ID)
		assert.Equal(t, original.PresentationContexts[i].Result, pc.Result)
		assert.Equal(t, original.PresentationContexts[i].TransferSyntax, pc.TransferSyntax)
	}
}

// TestAssociateRJ_EncodeDecode tests A-ASSOCIATE-RJ encoding and decoding
func TestAssociateRJ_EncodeDecode(t *testing.T) {
	original := &pdu.AssociateRJ{
		Result: 1,
		Source: 1,
		Reason: 2,
	}

	// Encode
	var buf bytes.Buffer
	err := original.Encode(&buf)
	require.NoError(t, err)

	// Verify PDU type
	data := buf.Bytes()
	assert.Equal(t, pdu.PDUTypeAssociateRJ, data[0])

	// Decode
	decoded := &pdu.AssociateRJ{}
	err = decoded.Decode(bytes.NewReader(data[6:]))
	require.NoError(t, err)

	// Verify fields
	assert.Equal(t, original.Result, decoded.Result)
	assert.Equal(t, original.Source, decoded.Source)
	assert.Equal(t, original.Reason, decoded.Reason)
}

// TestDataTF_EncodeDecode tests P-DATA-TF encoding and decoding
func TestDataTF_EncodeDecode(t *testing.T) {
	original := &pdu.DataTF{
		Items: []pdu.PresentationDataValue{
			{
				PresentationContextID: 1,
				MessageControlHeader:  0x01, // Command, not last
				Data:                  []byte{1, 2, 3, 4, 5},
			},
			{
				PresentationContextID: 1,
				MessageControlHeader:  0x03, // Command, last
				Data:                  []byte{6, 7, 8},
			},
		},
	}

	// Encode
	var buf bytes.Buffer
	err := original.Encode(&buf)
	require.NoError(t, err)

	// Verify PDU type
	data := buf.Bytes()
	assert.Equal(t, pdu.PDUTypeData, data[0])

	// Decode
	decoded := &pdu.DataTF{}
	err = decoded.Decode(bytes.NewReader(data[6:]))
	require.NoError(t, err)

	// Verify items
	assert.Len(t, decoded.Items, len(original.Items))
	for i, item := range decoded.Items {
		assert.Equal(t, original.Items[i].PresentationContextID, item.PresentationContextID)
		assert.Equal(t, original.Items[i].MessageControlHeader, item.MessageControlHeader)
		assert.Equal(t, original.Items[i].Data, item.Data)
	}
}

// TestReleaseRQ_EncodeDecode tests A-RELEASE-RQ encoding and decoding
func TestReleaseRQ_EncodeDecode(t *testing.T) {
	original := &pdu.ReleaseRQ{}

	// Encode
	var buf bytes.Buffer
	err := original.Encode(&buf)
	require.NoError(t, err)

	// Verify PDU type
	data := buf.Bytes()
	assert.Equal(t, pdu.PDUTypeReleaseRQ, data[0])

	// Decode
	decoded := &pdu.ReleaseRQ{}
	err = decoded.Decode(bytes.NewReader(data[6:]))
	require.NoError(t, err)
}

// TestReleaseRP_EncodeDecode tests A-RELEASE-RP encoding and decoding
func TestReleaseRP_EncodeDecode(t *testing.T) {
	original := &pdu.ReleaseRP{}

	// Encode
	var buf bytes.Buffer
	err := original.Encode(&buf)
	require.NoError(t, err)

	// Verify PDU type
	data := buf.Bytes()
	assert.Equal(t, pdu.PDUTypeReleaseRP, data[0])

	// Decode
	decoded := &pdu.ReleaseRP{}
	err = decoded.Decode(bytes.NewReader(data[6:]))
	require.NoError(t, err)
}

// TestAbort_EncodeDecode tests A-ABORT encoding and decoding
func TestAbort_EncodeDecode(t *testing.T) {
	original := &pdu.Abort{
		Source: 0,
		Reason: 2,
	}

	// Encode
	var buf bytes.Buffer
	err := original.Encode(&buf)
	require.NoError(t, err)

	// Verify PDU type
	data := buf.Bytes()
	assert.Equal(t, pdu.PDUTypeAbort, data[0])

	// Decode
	decoded := &pdu.Abort{}
	err = decoded.Decode(bytes.NewReader(data[6:]))
	require.NoError(t, err)

	// Verify fields
	assert.Equal(t, original.Source, decoded.Source)
	assert.Equal(t, original.Reason, decoded.Reason)
}

// TestReadPDU tests reading various PDU types
func TestReadPDU(t *testing.T) {
	tests := []struct {
		name     string
		pdu      pdu.PDU
		expected byte
	}{
		{"AssociateRQ", &pdu.AssociateRQ{
			ProtocolVersion:    0x0001,
			CalledAETitle:      pdu.PadAETitle("CALLED"),
			CallingAETitle:     pdu.PadAETitle("CALLING"),
			ApplicationContext: "1.2.840.10008.3.1.1.1",
			UserInfo: pdu.UserInformation{
				MaxPDULength:           16384,
				ImplementationClassUID: "1.2.840.12345.1.1",
			},
		}, pdu.PDUTypeAssociateRQ},
		{"ReleaseRQ", &pdu.ReleaseRQ{}, pdu.PDUTypeReleaseRQ},
		{"ReleaseRP", &pdu.ReleaseRP{}, pdu.PDUTypeReleaseRP},
		{"Abort", &pdu.Abort{Source: 0, Reason: 2}, pdu.PDUTypeAbort},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			var buf bytes.Buffer
			err := tt.pdu.Encode(&buf)
			require.NoError(t, err)

			// Read PDU
			decoded, err := pdu.ReadPDU(&buf)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, decoded.Type())
		})
	}
}

// TestPadTrimAETitle tests AE title padding and trimming
func TestPadTrimAETitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Short title", "TEST", "TEST"},
		{"Long title", "VERY_LONG_AE_TIT", "VERY_LONG_AE_TIT"},
		{"Max length", "1234567890123456", "1234567890123456"},
		{"Empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			padded := pdu.PadAETitle(tt.input)
			trimmed := pdu.TrimAETitle(padded)
			assert.Equal(t, tt.expected, trimmed)

			// Verify padding
			if len(tt.input) < 16 {
				// Check that remaining bytes are spaces
				for i := len(tt.input); i < 16; i++ {
					assert.Equal(t, byte(' '), padded[i])
				}
			}
		})
	}
}

// TestDataTF_MessageControlHeader tests message control header helpers
func TestDataTF_MessageControlHeader(t *testing.T) {
	tests := []struct {
		name               string
		header             uint8
		expectCommand      bool
		expectLastFragment bool
	}{
		{"Command first", 0x01, true, false},
		{"Command last", 0x03, true, true},
		{"Dataset first", 0x00, false, false},
		{"Dataset last", 0x02, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdv := pdu.PresentationDataValue{
				PresentationContextID: 1,
				MessageControlHeader:  tt.header,
				Data:                  []byte{1, 2, 3},
			}

			assert.Equal(t, tt.expectCommand, pdv.IsCommand())
			assert.Equal(t, tt.expectLastFragment, pdv.IsLastFragment())
		})
	}
}
