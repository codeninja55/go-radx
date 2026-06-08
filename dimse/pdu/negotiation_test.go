package pdu

import (
	"bytes"
	"errors"
	"testing"
)

const (
	negCTImageUID  = "1.2.840.10008.5.1.4.1.1.2"
	negStorageSC   = "1.2.840.10008.4.2"
	negFindSOPUID  = "1.2.840.10008.5.1.4.1.2.2.1"
	negStudyRootSC = "1.2.840.10008.4.2"
)

// roundTripUserInfo encodes a UserInformation into its 0x50 sub-item, then decodes it back so a
// test can assert the nested sub-items survive the full user-information round trip.
func roundTripUserInfo(t *testing.T, ui UserInformation) UserInformation {
	t.Helper()
	var buf bytes.Buffer
	if err := encodeUserInformation(&buf, ui); err != nil {
		t.Fatalf("encodeUserInformation: %v", err)
	}
	itemType, data, err := readItem(newBoundedReader(bytes.NewReader(buf.Bytes()), int64(buf.Len())))
	if err != nil {
		t.Fatalf("readItem: %v", err)
	}
	if itemType != ItemTypeUserInformation {
		t.Fatalf("item type = %#02x, want %#02x", itemType, ItemTypeUserInformation)
	}
	got, err := decodeUserInformation(data)
	if err != nil {
		t.Fatalf("decodeUserInformation: %v", err)
	}
	return got
}

// TestAsyncOperationsRoundTrip verifies the Asynchronous Operations Window sub-item (item type
// 0x53, PS3.7 D.3.3.3) encodes and decodes byte-for-byte, including the unlimited (0) sentinel.
func TestAsyncOperationsRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		in   AsyncOperations
	}{
		{"window 1,1", AsyncOperations{MaxOperationsInvoked: 1, MaxOperationsPerformed: 1}},
		{"asymmetric", AsyncOperations{MaxOperationsInvoked: 3, MaxOperationsPerformed: 7}},
		{"unlimited", AsyncOperations{MaxOperationsInvoked: 0, MaxOperationsPerformed: 0}},
		{"max", AsyncOperations{MaxOperationsInvoked: 0xFFFF, MaxOperationsPerformed: 0xFFFF}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui := roundTripUserInfo(t, UserInformation{MaxPDULength: 16382, AsyncOps: &tt.in})
			if ui.AsyncOps == nil {
				t.Fatal("async-ops sub-item lost in round trip")
			}
			if *ui.AsyncOps != tt.in {
				t.Errorf("round-trip = %+v, want %+v", *ui.AsyncOps, tt.in)
			}
		})
	}
}

// TestAsyncOperationsRejectsTruncated guards the 0x53 decoder against a body shorter than the two
// required counts (PRD §9.3).
func TestAsyncOperationsRejectsTruncated(t *testing.T) {
	for _, data := range [][]byte{nil, {0x00}, {0x00, 0x01}, {0x00, 0x01, 0x00}} {
		if _, err := decodeAsyncOperations(data); err == nil {
			t.Errorf("decodeAsyncOperations(%v) = nil error, want a decode error", data)
		}
	}
}

// TestExtendedNegotiationRoundTrip verifies the SOP Class Extended Negotiation sub-item (item type
// 0x56, PS3.7 D.3.3.5) round-trips its SOP Class UID and opaque application-information blob.
func TestExtendedNegotiationRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		in   ExtendedNegotiation
	}{
		{"with app info", ExtendedNegotiation{SOPClassUID: negFindSOPUID, ServiceClassAppInfo: []byte{0x01, 0x01, 0x01}}},
		{"empty app info", ExtendedNegotiation{SOPClassUID: negCTImageUID}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui := roundTripUserInfo(t, UserInformation{
				MaxPDULength:         16382,
				ExtendedNegotiations: []ExtendedNegotiation{tt.in},
			})
			if len(ui.ExtendedNegotiations) != 1 {
				t.Fatalf("extended negotiations = %d, want 1", len(ui.ExtendedNegotiations))
			}
			got := ui.ExtendedNegotiations[0]
			if got.SOPClassUID != tt.in.SOPClassUID {
				t.Errorf("SOP Class UID = %q, want %q", got.SOPClassUID, tt.in.SOPClassUID)
			}
			if !bytes.Equal(got.ServiceClassAppInfo, tt.in.ServiceClassAppInfo) {
				t.Errorf("app info = %v, want %v", got.ServiceClassAppInfo, tt.in.ServiceClassAppInfo)
			}
		})
	}
}

// TestExtendedNegotiationRejectsTruncated guards the 0x56 decoder against a declared UID length that
// runs past the bytes present (PRD §9.3).
func TestExtendedNegotiationRejectsTruncated(t *testing.T) {
	for _, data := range [][]byte{nil, {0x00}, {0x00, 0x40, 0x31, 0x32}} {
		if _, err := decodeExtendedNegotiation(data); err == nil {
			t.Errorf("decodeExtendedNegotiation(%v) = nil error, want a decode error", data)
		}
	}
}

// TestCommonExtendedNegotiationRoundTrip verifies the SOP Class Common Extended Negotiation sub-item
// (item type 0x57, PS3.7 D.3.3.6) round-trips its SOP Class UID, Service Class UID, and the
// related-general-SOP-class list.
func TestCommonExtendedNegotiationRoundTrip(t *testing.T) {
	in := CommonExtendedNegotiation{
		SOPClassUID:              negCTImageUID,
		ServiceClassUID:          negStorageSC,
		RelatedGeneralSOPClasses: []string{negFindSOPUID, "1.2.840.10008.5.1.4.1.1.4"},
	}
	ui := roundTripUserInfo(t, UserInformation{
		MaxPDULength:               16382,
		CommonExtendedNegotiations: []CommonExtendedNegotiation{in},
	})
	if len(ui.CommonExtendedNegotiations) != 1 {
		t.Fatalf("common extended negotiations = %d, want 1", len(ui.CommonExtendedNegotiations))
	}
	got := ui.CommonExtendedNegotiations[0]
	if got.SOPClassUID != in.SOPClassUID || got.ServiceClassUID != in.ServiceClassUID {
		t.Errorf("got (%q, %q), want (%q, %q)", got.SOPClassUID, got.ServiceClassUID, in.SOPClassUID, in.ServiceClassUID)
	}
	if len(got.RelatedGeneralSOPClasses) != len(in.RelatedGeneralSOPClasses) {
		t.Fatalf("related classes = %d, want %d", len(got.RelatedGeneralSOPClasses), len(in.RelatedGeneralSOPClasses))
	}
	for i, uid := range in.RelatedGeneralSOPClasses {
		if got.RelatedGeneralSOPClasses[i] != uid {
			t.Errorf("related[%d] = %q, want %q", i, got.RelatedGeneralSOPClasses[i], uid)
		}
	}
}

// TestCommonExtendedNegotiationNoRelated verifies the sub-item round-trips with an empty
// related-classes list.
func TestCommonExtendedNegotiationNoRelated(t *testing.T) {
	in := CommonExtendedNegotiation{SOPClassUID: negCTImageUID, ServiceClassUID: negStudyRootSC}
	ui := roundTripUserInfo(t, UserInformation{
		MaxPDULength:               16382,
		CommonExtendedNegotiations: []CommonExtendedNegotiation{in},
	})
	if len(ui.CommonExtendedNegotiations) != 1 {
		t.Fatalf("common extended negotiations = %d, want 1", len(ui.CommonExtendedNegotiations))
	}
	if got := ui.CommonExtendedNegotiations[0]; len(got.RelatedGeneralSOPClasses) != 0 {
		t.Errorf("related classes = %d, want 0", len(got.RelatedGeneralSOPClasses))
	}
}

// TestCommonExtendedNegotiationRejectsTruncated guards the 0x57 decoder against truncated length
// fields at each position (PRD §9.3).
func TestCommonExtendedNegotiationRejectsTruncated(t *testing.T) {
	tests := [][]byte{
		nil,
		{0x00},                               // partial SOP Class length
		{0x00, 0x40, 0x31},                   // SOP Class length past body
		{0x00, 0x01, 0x31},                   // SOP Class ok, no Service Class
		{0x00, 0x01, 0x31, 0x00, 0x01, 0x32}, // both UIDs ok, no related length
		{0x00, 0x01, 0x31, 0x00, 0x01, 0x32, 0x00, 0x40}, // related length past body
	}
	for _, data := range tests {
		if _, err := decodeCommonExtendedNegotiation(data); err == nil {
			t.Errorf("decodeCommonExtendedNegotiation(%v) = nil error, want a decode error", data)
		}
	}
}

// TestUserIdentityRQRoundTripAllTypes verifies the User Identity Negotiation request sub-item (item
// type 0x58, PS3.7 D.3.3.7) round-trips for every identity type 1..5, preserving the primary and
// secondary fields and the positive-response flag.
func TestUserIdentityRQRoundTripAllTypes(t *testing.T) {
	tests := []struct {
		name string
		in   UserIdentityRQ
	}{
		{"username", UserIdentityRQ{Type: UserIdentityUsername, PrimaryField: []byte("alice")}},
		{"username+passcode", UserIdentityRQ{Type: UserIdentityUsernamePasscode, PrimaryField: []byte("alice"), SecondaryField: []byte("s3cret"), PositiveResponseRequested: true}},
		{"kerberos", UserIdentityRQ{Type: UserIdentityKerberos, PrimaryField: []byte{0xDE, 0xAD, 0xBE, 0xEF}, PositiveResponseRequested: true}},
		{"saml", UserIdentityRQ{Type: UserIdentitySAML, PrimaryField: []byte("<saml:Assertion/>")}},
		{"jwt", UserIdentityRQ{Type: UserIdentityJWT, PrimaryField: []byte("eyJhbGciOiJIUzI1NiJ9.e30.sig"), PositiveResponseRequested: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui := roundTripUserInfo(t, UserInformation{MaxPDULength: 16382, UserIdentityRQ: &tt.in})
			if ui.UserIdentityRQ == nil {
				t.Fatal("user-identity RQ lost in round trip")
			}
			got := *ui.UserIdentityRQ
			if got.Type != tt.in.Type {
				t.Errorf("type = %d, want %d", got.Type, tt.in.Type)
			}
			if got.PositiveResponseRequested != tt.in.PositiveResponseRequested {
				t.Errorf("positive-response flag = %v, want %v", got.PositiveResponseRequested, tt.in.PositiveResponseRequested)
			}
			if !bytes.Equal(got.PrimaryField, tt.in.PrimaryField) {
				t.Errorf("primary field = %v, want %v", got.PrimaryField, tt.in.PrimaryField)
			}
			if !bytes.Equal(got.SecondaryField, tt.in.SecondaryField) {
				t.Errorf("secondary field = %v, want %v", got.SecondaryField, tt.in.SecondaryField)
			}
		})
	}
}

// TestUserIdentityRQRejectsTruncated guards the 0x58 decoder against bodies that underrun the type,
// flag, or length-prefixed fields (PRD §9.3).
func TestUserIdentityRQRejectsTruncated(t *testing.T) {
	tests := [][]byte{
		nil,
		{0x01},                         // type only, no flag
		{0x01, 0x00},                   // no primary length
		{0x01, 0x00, 0x00, 0x05, 0x31}, // primary length 5, one byte present
		{0x02, 0x01, 0x00, 0x01, 0x41}, // primary ok, no secondary length
	}
	for _, data := range tests {
		if _, err := decodeUserIdentityRQ(data); err == nil {
			t.Errorf("decodeUserIdentityRQ(%v) = nil error, want a decode error", data)
		}
	}
}

// TestUserIdentityACRoundTrip verifies the User Identity Negotiation response sub-item (item type
// 0x59, PS3.7 D.3.3.7) round-trips its server-response field, including the empty case.
func TestUserIdentityACRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		in   UserIdentityAC
	}{
		{"with response", UserIdentityAC{ServerResponse: []byte{0x01, 0x02, 0x03}}},
		{"empty response", UserIdentityAC{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui := roundTripUserInfo(t, UserInformation{MaxPDULength: 16382, UserIdentityAC: &tt.in})
			if ui.UserIdentityAC == nil {
				t.Fatal("user-identity AC lost in round trip")
			}
			if !bytes.Equal(ui.UserIdentityAC.ServerResponse, tt.in.ServerResponse) {
				t.Errorf("server response = %v, want %v", ui.UserIdentityAC.ServerResponse, tt.in.ServerResponse)
			}
		})
	}
}

// TestUserIdentityACRejectsTruncated guards the 0x59 decoder against a declared length past the
// bytes present (PRD §9.3).
func TestUserIdentityACRejectsTruncated(t *testing.T) {
	for _, data := range [][]byte{{0x00}, {0x00, 0x05, 0x31}} {
		if _, err := decodeUserIdentityAC(data); err == nil {
			t.Errorf("decodeUserIdentityAC(%v) = nil error, want a decode error", data)
		}
	}
}

// TestEncodeRejectsOversizedNegotiationFields verifies every uint16-length-prefixed negotiation field
// is refused with an *EncodeError when it exceeds the 65535-byte length prefix, rather than silently
// truncating the length and emitting a corrupt PDU. A field exactly at the limit still encodes.
func TestEncodeRejectsOversizedNegotiationFields(t *testing.T) {
	oversized := make([]byte, maxUint16Field+1) // 65536 bytes, one past the uint16 prefix
	atLimit := make([]byte, maxUint16Field)     // 65535 bytes, the largest encodable field
	oversizedUID := string(make([]byte, maxUint16Field+1))

	t.Run("user-identity RQ primary field", func(t *testing.T) {
		var buf bytes.Buffer
		err := encodeUserIdentityRQ(&buf, UserIdentityRQ{Type: UserIdentityKerberos, PrimaryField: oversized})
		assertEncodeError(t, err)
	})
	t.Run("user-identity RQ secondary field", func(t *testing.T) {
		var buf bytes.Buffer
		err := encodeUserIdentityRQ(&buf, UserIdentityRQ{Type: UserIdentityUsernamePasscode, PrimaryField: []byte("alice"), SecondaryField: oversized})
		assertEncodeError(t, err)
	})
	t.Run("user-identity AC server response", func(t *testing.T) {
		var buf bytes.Buffer
		err := encodeUserIdentityAC(&buf, UserIdentityAC{ServerResponse: oversized})
		assertEncodeError(t, err)
	})
	t.Run("extended-negotiation SOP Class UID", func(t *testing.T) {
		var buf bytes.Buffer
		err := encodeExtendedNegotiation(&buf, ExtendedNegotiation{SOPClassUID: oversizedUID})
		assertEncodeError(t, err)
	})
	t.Run("common-extended SOP Class UID", func(t *testing.T) {
		var buf bytes.Buffer
		err := encodeCommonExtendedNegotiation(&buf, CommonExtendedNegotiation{SOPClassUID: oversizedUID, ServiceClassUID: negStorageSC})
		assertEncodeError(t, err)
	})
	t.Run("user-information surfaces the field error", func(t *testing.T) {
		var buf bytes.Buffer
		err := encodeUserInformation(&buf, UserInformation{
			MaxPDULength:   16382,
			UserIdentityAC: &UserIdentityAC{ServerResponse: oversized},
		})
		assertEncodeError(t, err)
	})
	t.Run("field at the uint16 limit still encodes", func(t *testing.T) {
		var buf bytes.Buffer
		if err := encodeUserIdentityAC(&buf, UserIdentityAC{ServerResponse: atLimit}); err != nil {
			t.Fatalf("encodeUserIdentityAC at the limit: %v", err)
		}
		// The body is the 4-byte sub-item header plus a 2-byte length plus the field bytes.
		if want := 4 + 2 + len(atLimit); buf.Len() != want {
			t.Errorf("encoded length = %d, want %d", buf.Len(), want)
		}
	})
}

// TestEncodeOversizedFieldCorruptsAssociatePDU verifies an A-ASSOCIATE-AC carrying an over-length
// user-identity server response fails to encode (rather than emitting a PDU whose nested item lengths
// disagree with the bytes that follow).
func TestEncodeOversizedFieldCorruptsAssociatePDU(t *testing.T) {
	ac := &AssociateAC{
		ProtocolVersion:    1,
		CalledAETitle:      padAETitle("ACCEPTOR"),
		CallingAETitle:     padAETitle("REQUESTOR"),
		ApplicationContext: "1.2.840.10008.3.1.1.1",
		PresentationContexts: []PresentationContextAC{
			{ID: 1, Result: PresentationContextAcceptance, TransferSyntax: "1.2.840.10008.1.2.1"},
		},
		UserInfo: UserInformation{
			MaxPDULength:   16382,
			UserIdentityAC: &UserIdentityAC{ServerResponse: make([]byte, maxUint16Field+1)},
		},
	}
	var buf bytes.Buffer
	err := ac.Encode(&buf)
	assertEncodeError(t, err)
	if buf.Len() != 0 {
		t.Errorf("Encode wrote %d bytes despite the over-length field; want a clean refusal", buf.Len())
	}
}

func assertEncodeError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("got nil error, want an *EncodeError for the over-length field")
	}
	var ee *EncodeError
	if !errors.As(err, &ee) {
		t.Fatalf("error = %T (%v), want *EncodeError", err, err)
	}
}

// TestAssociateRQRoundTripWithNegotiationItems verifies the new RQ-side sub-items (async-ops,
// extended, common-extended, user-identity RQ) survive a full A-ASSOCIATE-RQ encode/decode together.
func TestAssociateRQRoundTripWithNegotiationItems(t *testing.T) {
	rq := &AssociateRQ{
		ProtocolVersion:    1,
		CalledAETitle:      padAETitle("ACCEPTOR"),
		CallingAETitle:     padAETitle("REQUESTOR"),
		ApplicationContext: "1.2.840.10008.3.1.1.1",
		PresentationContexts: []PresentationContextRQ{
			{ID: 1, AbstractSyntax: negCTImageUID, TransferSyntaxes: []string{"1.2.840.10008.1.2.1"}},
		},
		UserInfo: UserInformation{
			MaxPDULength:               16382,
			AsyncOps:                   &AsyncOperations{MaxOperationsInvoked: 1, MaxOperationsPerformed: 1},
			ExtendedNegotiations:       []ExtendedNegotiation{{SOPClassUID: negFindSOPUID, ServiceClassAppInfo: []byte{0x01}}},
			CommonExtendedNegotiations: []CommonExtendedNegotiation{{SOPClassUID: negCTImageUID, ServiceClassUID: negStorageSC}},
			UserIdentityRQ:             &UserIdentityRQ{Type: UserIdentityUsernamePasscode, PrimaryField: []byte("alice"), SecondaryField: []byte("pw"), PositiveResponseRequested: true},
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
	ui := got.UserInfo
	if ui.AsyncOps == nil || *ui.AsyncOps != *rq.UserInfo.AsyncOps {
		t.Errorf("async-ops = %+v, want %+v", ui.AsyncOps, rq.UserInfo.AsyncOps)
	}
	if len(ui.ExtendedNegotiations) != 1 || ui.ExtendedNegotiations[0].SOPClassUID != negFindSOPUID {
		t.Errorf("extended negotiations = %+v", ui.ExtendedNegotiations)
	}
	if len(ui.CommonExtendedNegotiations) != 1 || ui.CommonExtendedNegotiations[0].SOPClassUID != negCTImageUID {
		t.Errorf("common extended negotiations = %+v", ui.CommonExtendedNegotiations)
	}
	if ui.UserIdentityRQ == nil || ui.UserIdentityRQ.Type != UserIdentityUsernamePasscode {
		t.Errorf("user-identity RQ = %+v", ui.UserIdentityRQ)
	}
}

// TestAssociateACRoundTripWithUserIdentityAC verifies the acceptor-side user-identity response
// sub-item survives a full A-ASSOCIATE-AC encode/decode.
func TestAssociateACRoundTripWithUserIdentityAC(t *testing.T) {
	ac := &AssociateAC{
		ProtocolVersion:    1,
		CalledAETitle:      padAETitle("ACCEPTOR"),
		CallingAETitle:     padAETitle("REQUESTOR"),
		ApplicationContext: "1.2.840.10008.3.1.1.1",
		PresentationContexts: []PresentationContextAC{
			{ID: 1, Result: PresentationContextAcceptance, TransferSyntax: "1.2.840.10008.1.2.1"},
		},
		UserInfo: UserInformation{
			MaxPDULength:   16382,
			AsyncOps:       &AsyncOperations{MaxOperationsInvoked: 1, MaxOperationsPerformed: 1},
			UserIdentityAC: &UserIdentityAC{ServerResponse: []byte("ok")},
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
	if got.UserInfo.UserIdentityAC == nil || !bytes.Equal(got.UserInfo.UserIdentityAC.ServerResponse, []byte("ok")) {
		t.Errorf("user-identity AC = %+v", got.UserInfo.UserIdentityAC)
	}
	if got.UserInfo.AsyncOps == nil {
		t.Error("async-ops echo lost in AC round trip")
	}
}
