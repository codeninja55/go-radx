package pdu

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// User-identity negotiation identity-type values (PS3.7 D.3.3.7, Table D.3-15). They select how
// the primary (and, for type 2, secondary) field is interpreted.
const (
	UserIdentityUsername         uint8 = 1 // primary = username (no passcode)
	UserIdentityUsernamePasscode uint8 = 2 // primary = username, secondary = passcode
	UserIdentityKerberos         uint8 = 3 // primary = Kerberos service ticket
	UserIdentitySAML             uint8 = 4 // primary = SAML assertion
	UserIdentityJWT              uint8 = 5 // primary = JSON Web Token
)

// AsyncOperations is the Asynchronous Operations Window sub-item (item type 0x53, PS3.7 D.3.3.3):
// the maximum number of operations the AE may have invoked, and the maximum it may have performed,
// outstanding at once. A field value of 0 means unlimited (PS3.7 D.3.3.3.1).
type AsyncOperations struct {
	MaxOperationsInvoked   uint16
	MaxOperationsPerformed uint16
}

// ExtendedNegotiation is one SOP Class Extended Negotiation sub-item (item type 0x56, PS3.7
// D.3.3.5): a SOP Class UID and an opaque service-class application-information blob whose layout is
// defined by the SOP Class's service class (PS3.4). The codec treats ServiceClassAppInfo as opaque
// bytes; interpreting it is the service class's concern, not the PDU layer's.
type ExtendedNegotiation struct {
	SOPClassUID         string
	ServiceClassAppInfo []byte
}

// CommonExtendedNegotiation is one SOP Class Common Extended Negotiation sub-item (item type 0x57,
// PS3.7 D.3.3.6): a SOP Class UID, the Service Class UID it belongs to, and the Related General SOP
// Class UIDs. The final field is the reserved Service Class Application Information, preserved as
// opaque bytes so an unknown trailing field round-trips rather than being dropped.
type CommonExtendedNegotiation struct {
	SOPClassUID              string
	ServiceClassUID          string
	RelatedGeneralSOPClasses []string
}

// UserIdentityRQ is the User Identity Negotiation request sub-item (item type 0x58, PS3.7 D.3.3.7):
// the identity type, whether a positive response is requested, and the primary and (for type 2)
// secondary fields. The fields are opaque to the codec: a username, passcode, Kerberos ticket, SAML
// assertion, or JWT, per the identity type. Secrets carried here are never logged (PRD §9.8).
type UserIdentityRQ struct {
	Type                      uint8
	PositiveResponseRequested bool
	PrimaryField              []byte
	SecondaryField            []byte
}

// UserIdentityAC is the User Identity Negotiation response sub-item (item type 0x59, PS3.7 D.3.3.7):
// the server response the acceptor returns when the requestor asked for a positive response. The
// ServerResponse is opaque (a Kerberos server ticket or a SAML response, per the request's identity
// type); for a username/passcode request it is empty.
type UserIdentityAC struct {
	ServerResponse []byte
}

// encodeAsyncOperations writes the Asynchronous Operations Window sub-item body: the 2-byte
// maximum-operations-invoked and 2-byte maximum-operations-performed counts (PS3.7 D.3.3.3).
func encodeAsyncOperations(out *bytes.Buffer, ao AsyncOperations) {
	var body [4]byte
	binary.BigEndian.PutUint16(body[0:2], ao.MaxOperationsInvoked)
	binary.BigEndian.PutUint16(body[2:4], ao.MaxOperationsPerformed)
	encodeItem(out, ItemTypeAsyncOperations, body[:])
}

// decodeAsyncOperations parses the Asynchronous Operations Window sub-item body, requiring the two
// 2-byte counts (PS3.7 D.3.3.3). A body shorter than four bytes is a *PDUError, never a panic.
func decodeAsyncOperations(data []byte) (AsyncOperations, error) {
	if len(data) < 4 {
		return AsyncOperations{}, &PDUError{
			Detail: fmt.Sprintf("async-operations sub-item is %d bytes, need 4 for the two operation counts", len(data)),
		}
	}
	return AsyncOperations{
		MaxOperationsInvoked:   binary.BigEndian.Uint16(data[0:2]),
		MaxOperationsPerformed: binary.BigEndian.Uint16(data[2:4]),
	}, nil
}

// encodeExtendedNegotiation writes the SOP Class Extended Negotiation sub-item body (PS3.7 D.3.3.5):
// a 2-byte SOP Class UID length, the SOP Class UID bytes, then the service-class application
// information bytes (the remainder of the sub-item, no length prefix). An over-length SOP Class UID,
// or an assembled body that exceeds the sub-item's 2-byte length prefix, is an *EncodeError.
func encodeExtendedNegotiation(out *bytes.Buffer, en ExtendedNegotiation) error {
	var body bytes.Buffer
	if err := writeUIDField(&body, "Extended Negotiation SOP Class UID", en.SOPClassUID); err != nil {
		return err
	}
	body.Write(en.ServiceClassAppInfo)
	return encodeItemChecked(out, "Extended Negotiation sub-item body", ItemTypeExtendedNegotiation, body.Bytes())
}

// decodeExtendedNegotiation parses one SOP Class Extended Negotiation sub-item body (PS3.7 D.3.3.5),
// validating the declared UID length against the bytes present before slicing. The remainder after
// the UID is the opaque service-class application information. A truncated body is a *PDUError.
func decodeExtendedNegotiation(data []byte) (ExtendedNegotiation, error) {
	var en ExtendedNegotiation
	if len(data) < 2 {
		return en, &PDUError{
			Detail: fmt.Sprintf("extended-negotiation sub-item is %d bytes, need at least 2 for the UID length", len(data)),
		}
	}
	uidLen := int(binary.BigEndian.Uint16(data[:2]))
	if len(data) < 2+uidLen {
		return en, &PDUError{
			Detail: fmt.Sprintf("extended-negotiation sub-item declares a %d-byte UID but carries %d bytes after the length field",
				uidLen, len(data)-2),
		}
	}
	en.SOPClassUID = string(data[2 : 2+uidLen])
	if rest := data[2+uidLen:]; len(rest) > 0 {
		en.ServiceClassAppInfo = append([]byte(nil), rest...)
	}
	return en, nil
}

// encodeCommonExtendedNegotiation writes the SOP Class Common Extended Negotiation sub-item body
// (PS3.7 D.3.3.6): the length-prefixed SOP Class UID, the length-prefixed Service Class UID, a
// 2-byte length for the Related General SOP Class Identification field, then that field as a sequence
// of length-prefixed UIDs. The reserved trailing field is omitted (it carries no value here). An
// over-length UID, a related-classes field whose total length exceeds the 2-byte prefix, or an
// assembled body that exceeds the sub-item's 2-byte length prefix, is an *EncodeError.
func encodeCommonExtendedNegotiation(out *bytes.Buffer, cen CommonExtendedNegotiation) error {
	var body bytes.Buffer
	if err := writeUIDField(&body, "Common Extended Negotiation SOP Class UID", cen.SOPClassUID); err != nil {
		return err
	}
	if err := writeUIDField(&body, "Common Extended Negotiation Service Class UID", cen.ServiceClassUID); err != nil {
		return err
	}

	var related bytes.Buffer
	for _, uid := range cen.RelatedGeneralSOPClasses {
		if err := writeUIDField(&related, "Common Extended Negotiation related SOP Class UID", uid); err != nil {
			return err
		}
	}
	if err := checkUint16Field("Common Extended Negotiation related-classes field", related.Len()); err != nil {
		return err
	}
	var rl [2]byte
	binary.BigEndian.PutUint16(rl[:], uint16(related.Len())) // #nosec G115 -- bounded by the checkUint16Field guard above
	body.Write(rl[:])
	body.Write(related.Bytes())

	return encodeItemChecked(out, "Common Extended Negotiation sub-item body", ItemTypeCommonExtended, body.Bytes())
}

// decodeCommonExtendedNegotiation parses one SOP Class Common Extended Negotiation sub-item body
// (PS3.7 D.3.3.6), validating every declared length against the bytes present before slicing. The
// reserved Service Class Application Information field after the related-classes list is ignored. A
// truncated body is a *PDUError, never a panic or over-read (PRD §9.3).
func decodeCommonExtendedNegotiation(data []byte) (CommonExtendedNegotiation, error) {
	var cen CommonExtendedNegotiation
	r := newFieldReader(data)
	sopClass, err := r.uidField("Common Extended Negotiation SOP Class UID")
	if err != nil {
		return cen, err
	}
	serviceClass, err := r.uidField("Common Extended Negotiation Service Class UID")
	if err != nil {
		return cen, err
	}
	cen.SOPClassUID = sopClass
	cen.ServiceClassUID = serviceClass

	relatedLen, err := r.uint16("Common Extended Negotiation related-classes length")
	if err != nil {
		return cen, err
	}
	relatedBytes, err := r.bytes(int(relatedLen), "Common Extended Negotiation related-classes field")
	if err != nil {
		return cen, err
	}
	related := newFieldReader(relatedBytes)
	for related.remaining() > 0 {
		uid, uerr := related.uidField("Common Extended Negotiation related SOP Class UID")
		if uerr != nil {
			return cen, uerr
		}
		cen.RelatedGeneralSOPClasses = append(cen.RelatedGeneralSOPClasses, uid)
	}
	return cen, nil
}

// encodeUserIdentityRQ writes the User Identity Negotiation request sub-item body (PS3.7 D.3.3.7):
// the 1-byte identity type, the 1-byte positive-response-requested flag, a 2-byte primary-field
// length and the primary-field bytes, then a 2-byte secondary-field length and the secondary-field
// bytes. The secondary field is meaningful only for the username-and-passcode type but is always
// length-prefixed (a 0 length when absent).
// It returns an *EncodeError when either field exceeds its 2-byte length prefix, or when the assembled
// body (both fields plus their length prefixes and the two leading bytes) exceeds the sub-item's
// 2-byte length prefix; the field names are generic ("primary"/"secondary") so the error carries no
// secret bytes (PRD §9.8).
func encodeUserIdentityRQ(out *bytes.Buffer, id UserIdentityRQ) error {
	var body bytes.Buffer
	body.WriteByte(id.Type)
	body.WriteByte(boolToFlag(id.PositiveResponseRequested))
	if err := writeLengthPrefixed(&body, "user-identity primary field", id.PrimaryField); err != nil {
		return err
	}
	if err := writeLengthPrefixed(&body, "user-identity secondary field", id.SecondaryField); err != nil {
		return err
	}
	return encodeItemChecked(out, "user-identity RQ sub-item body", ItemTypeUserIdentityRQ, body.Bytes())
}

// decodeUserIdentityRQ parses one User Identity Negotiation request sub-item body (PS3.7 D.3.3.7),
// validating the two length-prefixed fields against the bytes present before slicing. A truncated
// body is a *PDUError, never a panic. The decoded fields are opaque bytes the acceptor interprets by
// identity type; they are never logged (PRD §9.8).
func decodeUserIdentityRQ(data []byte) (UserIdentityRQ, error) {
	var id UserIdentityRQ
	r := newFieldReader(data)
	idType, err := r.byte("user-identity type")
	if err != nil {
		return id, err
	}
	flag, err := r.byte("user-identity positive-response flag")
	if err != nil {
		return id, err
	}
	primary, err := r.lengthPrefixed("user-identity primary field")
	if err != nil {
		return id, err
	}
	secondary, err := r.lengthPrefixed("user-identity secondary field")
	if err != nil {
		return id, err
	}
	id.Type = idType
	id.PositiveResponseRequested = flag != 0
	id.PrimaryField = primary
	id.SecondaryField = secondary
	return id, nil
}

// encodeUserIdentityAC writes the User Identity Negotiation response sub-item body (PS3.7 D.3.3.7):
// a 2-byte server-response length and the server-response bytes. An over-length server response, or an
// assembled body (the response plus its 2-byte length prefix) that exceeds the sub-item's 2-byte
// length prefix, is an *EncodeError; the field name carries no secret bytes (PRD §9.8).
func encodeUserIdentityAC(out *bytes.Buffer, ac UserIdentityAC) error {
	var body bytes.Buffer
	if err := writeLengthPrefixed(&body, "user-identity server response", ac.ServerResponse); err != nil {
		return err
	}
	return encodeItemChecked(out, "user-identity AC sub-item body", ItemTypeUserIdentityAC, body.Bytes())
}

// decodeUserIdentityAC parses one User Identity Negotiation response sub-item body (PS3.7 D.3.3.7),
// validating the length-prefixed server-response field against the bytes present. A truncated body is
// a *PDUError.
func decodeUserIdentityAC(data []byte) (UserIdentityAC, error) {
	var ac UserIdentityAC
	r := newFieldReader(data)
	resp, err := r.lengthPrefixed("user-identity server response")
	if err != nil {
		return ac, err
	}
	ac.ServerResponse = resp
	return ac, nil
}

// maxUint16Field is the largest value a 2-byte big-endian length prefix can carry. A
// length-prefixed field whose byte length exceeds it cannot be encoded without truncating the prefix,
// which would emit a corrupt PDU (PS3.7 D.3.3 sub-item length fields are 2 bytes).
const maxUint16Field = 0xFFFF

// checkUint16Field reports an *EncodeError when a length-prefixed field is too long for its 2-byte
// length prefix. The encoders call it before writing the prefix so an over-length field is refused
// rather than emitted with a truncated length and trailing bytes that leak into the enclosing item.
func checkUint16Field(field string, n int) error {
	if n > maxUint16Field {
		return &EncodeError{Detail: fmt.Sprintf("%s is %d bytes, exceeds the %d-byte (uint16) length prefix", field, n, maxUint16Field)}
	}
	return nil
}

// writeUIDField writes a 2-byte big-endian length followed by the UID bytes, after validating the UID
// fits in the 2-byte length prefix.
func writeUIDField(buf *bytes.Buffer, field, uid string) error {
	if err := checkUint16Field(field, len(uid)); err != nil {
		return err
	}
	var lb [2]byte
	binary.BigEndian.PutUint16(lb[:], uint16(len(uid))) // #nosec G115 -- bounded by the checkUint16Field guard above
	buf.Write(lb[:])
	buf.WriteString(uid)
	return nil
}

// writeLengthPrefixed writes a 2-byte big-endian length followed by the data bytes, after validating
// the data fits in the 2-byte length prefix. A nil or empty slice writes a zero length and no data.
func writeLengthPrefixed(buf *bytes.Buffer, field string, data []byte) error {
	if err := checkUint16Field(field, len(data)); err != nil {
		return err
	}
	var lb [2]byte
	binary.BigEndian.PutUint16(lb[:], uint16(len(data))) // #nosec G115 -- bounded by the checkUint16Field guard above
	buf.Write(lb[:])
	buf.Write(data)
	return nil
}

// fieldReader walks a sub-item body field by field, validating every declared length against the
// bytes that remain before slicing so an attacker-controlled length never reads past the buffer
// (PRD §9.3). Every read returns a typed *PDUError naming the field on underflow.
type fieldReader struct {
	data []byte
	pos  int
}

func newFieldReader(data []byte) *fieldReader { return &fieldReader{data: data} }

func (r *fieldReader) remaining() int { return len(r.data) - r.pos }

// byte reads one byte, naming field on underflow.
func (r *fieldReader) byte(field string) (byte, error) {
	if r.remaining() < 1 {
		return 0, &PDUError{Detail: fmt.Sprintf("%s: need 1 byte, %d remaining", field, r.remaining())}
	}
	b := r.data[r.pos]
	r.pos++
	return b, nil
}

// uint16 reads a 2-byte big-endian value, naming field on underflow.
func (r *fieldReader) uint16(field string) (uint16, error) {
	if r.remaining() < 2 {
		return 0, &PDUError{Detail: fmt.Sprintf("%s: need 2 bytes, %d remaining", field, r.remaining())}
	}
	v := binary.BigEndian.Uint16(r.data[r.pos : r.pos+2])
	r.pos += 2
	return v, nil
}

// bytes reads exactly n bytes, naming field on underflow. It returns a copy so the caller never
// aliases the decode buffer.
func (r *fieldReader) bytes(n int, field string) ([]byte, error) {
	if n < 0 || r.remaining() < n {
		return nil, &PDUError{Detail: fmt.Sprintf("%s: need %d bytes, %d remaining", field, n, r.remaining())}
	}
	out := append([]byte(nil), r.data[r.pos:r.pos+n]...)
	r.pos += n
	return out, nil
}

// uidField reads a 2-byte length and that many UID bytes as a string, naming field on underflow.
func (r *fieldReader) uidField(field string) (string, error) {
	n, err := r.uint16(field + " length")
	if err != nil {
		return "", err
	}
	b, err := r.bytes(int(n), field)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// lengthPrefixed reads a 2-byte length and that many data bytes, naming field on underflow. A
// zero-length field returns nil so an absent optional field decodes to nil, not an empty slice.
func (r *fieldReader) lengthPrefixed(field string) ([]byte, error) {
	n, err := r.uint16(field + " length")
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}
	return r.bytes(int(n), field)
}
