package pdu

import (
	"bytes"
	"testing"
)

// TestRoleSelectionSubItemRoundTrip verifies the SCP/SCU Role Selection sub-item (item type
// 0x54, PS3.7 D.3.3.4) encodes and decodes byte-for-byte: a 2-byte UID length, the SOP Class
// UID bytes, then the 1-byte SCU-role and 1-byte SCP-role flags.
func TestRoleSelectionSubItemRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		in   RoleSelection
	}{
		{"scu only", RoleSelection{SOPClassUID: "1.2.840.10008.5.1.4.1.1.2", SCURole: true, SCPRole: false}},
		{"scp only", RoleSelection{SOPClassUID: "1.2.840.10008.5.1.4.1.1.2", SCURole: false, SCPRole: true}},
		{"both roles", RoleSelection{SOPClassUID: "1.2.840.10008.1.1", SCURole: true, SCPRole: true}},
		{"neither role", RoleSelection{SOPClassUID: "1.2.840.10008.1.1", SCURole: false, SCPRole: false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			encodeRoleSelection(&buf, tt.in)

			itemType, data, err := readItem(newBoundedReader(bytes.NewReader(buf.Bytes()), int64(buf.Len())))
			if err != nil {
				t.Fatalf("readItem: %v", err)
			}
			if itemType != ItemTypeRoleSelection {
				t.Fatalf("item type = %#02x, want %#02x", itemType, ItemTypeRoleSelection)
			}
			got, err := decodeRoleSelection(data)
			if err != nil {
				t.Fatalf("decodeRoleSelection: %v", err)
			}
			if got != tt.in {
				t.Errorf("round-trip = %+v, want %+v", got, tt.in)
			}
		})
	}
}

// TestRoleSelectionWireLayout pins the exact body byte layout (PS3.7 D.3.3.4): the SCU/SCP
// role flags follow the UID, and the declared UID length matches the UID byte count.
func TestRoleSelectionWireLayout(t *testing.T) {
	const uid = "1.2.840.10008.1.1"
	var buf bytes.Buffer
	encodeRoleSelection(&buf, RoleSelection{SOPClassUID: uid, SCURole: true, SCPRole: false})

	b := buf.Bytes()
	if b[0] != ItemTypeRoleSelection {
		t.Fatalf("item type byte = %#02x, want %#02x", b[0], ItemTypeRoleSelection)
	}
	body := b[4:] // strip the 4-byte sub-item header
	uidLen := int(body[0])<<8 | int(body[1])
	if uidLen != len(uid) {
		t.Fatalf("declared UID length = %d, want %d", uidLen, len(uid))
	}
	if string(body[2:2+uidLen]) != uid {
		t.Errorf("UID bytes = %q, want %q", string(body[2:2+uidLen]), uid)
	}
	if body[2+uidLen] != 1 {
		t.Errorf("SCU-role byte = %d, want 1", body[2+uidLen])
	}
	if body[3+uidLen] != 0 {
		t.Errorf("SCP-role byte = %d, want 0", body[3+uidLen])
	}
}

// TestDecodeRoleSelectionRejectsTruncated guards malformed input: a body whose declared UID
// length runs past the bytes present, or one missing the two role flags, must be an error,
// never a panic or out-of-bounds read (PRD §9.3).
func TestDecodeRoleSelectionRejectsTruncated(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"length only", []byte{0x00, 0x05}},
		{"uid length past body", []byte{0x00, 0x40, 0x31, 0x32}},
		{"missing role flags", []byte{0x00, 0x02, 0x31, 0x32}},
		{"missing scp flag", []byte{0x00, 0x01, 0x31, 0x01}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := decodeRoleSelection(tt.data); err == nil {
				t.Errorf("decodeRoleSelection(%v) = nil error, want a decode error", tt.data)
			}
		})
	}
}

// TestAssociateRQRoundTripWithRoleSelection verifies the role-selection sub-items survive a
// full A-ASSOCIATE-RQ encode/decode through the user-information item, preserving order.
func TestAssociateRQRoundTripWithRoleSelection(t *testing.T) {
	rq := &AssociateRQ{
		ProtocolVersion:    1,
		CalledAETitle:      padAETitle("ACCEPTOR"),
		CallingAETitle:     padAETitle("REQUESTOR"),
		ApplicationContext: "1.2.840.10008.3.1.1.1",
		PresentationContexts: []PresentationContextRQ{
			{ID: 1, AbstractSyntax: "1.2.840.10008.5.1.4.1.1.2", TransferSyntaxes: []string{"1.2.840.10008.1.2.1"}},
		},
		UserInfo: UserInformation{
			MaxPDULength: 16382,
			RoleSelections: []RoleSelection{
				{SOPClassUID: "1.2.840.10008.5.1.4.1.1.2", SCURole: true, SCPRole: true},
			},
		},
	}

	var buf bytes.Buffer
	if err := rq.Encode(&buf); err != nil {
		t.Fatalf("AssociateRQ.Encode: %v", err)
	}
	got, err := DecodeAssociateRQ(&buf)
	if err != nil {
		t.Fatalf("DecodeAssociateRQ: %v", err)
	}
	if len(got.UserInfo.RoleSelections) != 1 {
		t.Fatalf("role selections = %d, want 1", len(got.UserInfo.RoleSelections))
	}
	if got.UserInfo.RoleSelections[0] != rq.UserInfo.RoleSelections[0] {
		t.Errorf("role selection = %+v, want %+v", got.UserInfo.RoleSelections[0], rq.UserInfo.RoleSelections[0])
	}
}

// TestAssociateACRoundTripWithRoleSelection verifies the acceptor's role-selection responses
// survive a full A-ASSOCIATE-AC encode/decode (PS3.7 D.3.3.4: the acceptor echoes one
// role-selection sub-item per SOP Class, carrying the roles it grants).
func TestAssociateACRoundTripWithRoleSelection(t *testing.T) {
	ac := &AssociateAC{
		ProtocolVersion:    1,
		CalledAETitle:      padAETitle("ACCEPTOR"),
		CallingAETitle:     padAETitle("REQUESTOR"),
		ApplicationContext: "1.2.840.10008.3.1.1.1",
		PresentationContexts: []PresentationContextAC{
			{ID: 1, Result: PresentationContextAcceptance, TransferSyntax: "1.2.840.10008.1.2.1"},
		},
		UserInfo: UserInformation{
			MaxPDULength: 16382,
			RoleSelections: []RoleSelection{
				{SOPClassUID: "1.2.840.10008.5.1.4.1.1.2", SCURole: true, SCPRole: true},
			},
		},
	}

	var buf bytes.Buffer
	if err := ac.Encode(&buf); err != nil {
		t.Fatalf("AssociateAC.Encode: %v", err)
	}
	got, err := DecodeAssociateAC(&buf)
	if err != nil {
		t.Fatalf("DecodeAssociateAC: %v", err)
	}
	if len(got.UserInfo.RoleSelections) != 1 {
		t.Fatalf("role selections = %d, want 1", len(got.UserInfo.RoleSelections))
	}
	if got.UserInfo.RoleSelections[0] != ac.UserInfo.RoleSelections[0] {
		t.Errorf("role selection = %+v, want %+v", got.UserInfo.RoleSelections[0], ac.UserInfo.RoleSelections[0])
	}
}
