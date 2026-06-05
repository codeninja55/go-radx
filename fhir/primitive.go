package fhir

import (
	"bytes"
	"encoding/json"
	"errors"
)

// errNotObject is returned by AppendSiblings when the value it must splice "_field"
// siblings onto is not a JSON object, which a generated value struct never produces
// but which a corrupt encoder would.
var errNotObject = errors.New("fhir: cannot append primitive siblings to a non-object JSON value")

// PrimitiveElement carries the id and extensions a FHIR primitive may hold
// alongside its value. In FHIR JSON every primitive element is split into two
// keys: the value itself under "field" and this companion under "_field". A
// boolean "active":true with an extension is serialised as "active":true and
// "_active":{"extension":[...]}; the value and the companion travel together but
// in separate keys, and either may be present without the other.
//
// PrimitiveElement is release-agnostic: the generated release packages reference
// this single type for every primitive's "_field" sibling, so the id/extension
// shape is defined once. Extension is left as raw JSON in v1 because the
// Extension datatype is itself generated per release and a release-neutral root
// type cannot name it; the raw form round-trips losslessly and a later increment
// that needs typed extensions decodes it on demand.
type PrimitiveElement struct {
	// ID is the element's local id (the FHIR Element.id), or nil when absent.
	ID *string `json:"id,omitempty"`

	// Extension holds the element's extensions as raw FHIR JSON, preserved
	// verbatim so a round-trip is byte-faithful without depending on the
	// release-specific Extension type.
	Extension json.RawMessage `json:"extension,omitempty"`
}

// IsZero reports whether the element carries neither an id nor any extension, so
// a caller (and the null-alignment marshaller) can treat it as absent. An
// all-empty PrimitiveElement is indistinguishable from a missing "_field" sibling
// and is never emitted.
func (e *PrimitiveElement) IsZero() bool {
	if e == nil {
		return true
	}
	return e.ID == nil && len(e.Extension) == 0
}

// nullToken is the JSON null literal, the placeholder a repeating primitive's
// value array and "_field" sibling array use to keep positions aligned when one
// side has a gap.
var nullToken = []byte("null")

// MarshalPrimitiveExtension renders the "_field" sibling for a scalar primitive, or
// nil when the element carries no id or extension so the "_field" key is dropped
// entirely. It exists because Go's "omitempty" drops only a nil pointer, not a
// non-nil but empty *PrimitiveElement, which would otherwise serialise as the noise
// key "_field":{}; routing the scalar sibling through this helper keeps the
// "emitted only when it carries an id or extension" rule that IsZero defines.
func MarshalPrimitiveExtension(element *PrimitiveElement) ([]byte, error) {
	if element.IsZero() {
		return nil, nil
	}
	return json.Marshal(element)
}

// MarshalPrimitiveExtensions renders the "_field" sibling array for a repeating
// primitive, null-aligned with the value array. FHIR requires the value array
// ("given":["Jane","Q"]) and the sibling array ("_given":[null,{"id":"x"}]) to
// share an index space: position i of the sibling array describes position i of
// the value array, with a JSON null wherever a position has no id/extension. The
// returned bytes are the JSON array to write under the "_field" key, or nil when
// no element carries an id or extension (in which case the "_field" key is
// omitted entirely). count is the length of the value array, so the sibling array
// is padded to the same length even when the longest extended position is earlier.
func MarshalPrimitiveExtensions(count int, elements []*PrimitiveElement) ([]byte, error) {
	if !anyPrimitiveExtension(elements) {
		return nil, nil
	}

	length := count
	if len(elements) > length {
		length = len(elements)
	}

	var buf bytes.Buffer
	buf.WriteByte('[')
	for i := range length {
		if i > 0 {
			buf.WriteByte(',')
		}
		if i >= len(elements) || elements[i].IsZero() {
			buf.Write(nullToken)
			continue
		}
		encoded, err := json.Marshal(elements[i])
		if err != nil {
			return nil, err
		}
		buf.Write(encoded)
	}
	buf.WriteByte(']')
	return buf.Bytes(), nil
}

// UnmarshalPrimitiveExtensions decodes a "_field" sibling array into a slice of
// PrimitiveElement pointers, restoring the null placeholders as nil entries so the
// result stays index-aligned with the decoded value array. A JSON null at
// position i (a value with no id/extension) becomes a nil pointer at index i,
// preserving the alignment FHIR encodes positionally.
func UnmarshalPrimitiveExtensions(data []byte) ([]*PrimitiveElement, error) {
	if len(data) == 0 || bytes.Equal(bytes.TrimSpace(data), nullToken) {
		return nil, nil
	}
	var elements []*PrimitiveElement
	if err := json.Unmarshal(data, &elements); err != nil {
		return nil, err
	}
	return elements, nil
}

// anyPrimitiveExtension reports whether any element in the slice carries an id or
// extension, the condition under which a "_field" sibling is emitted at all. An
// all-nil or all-empty slice yields no sibling, matching the rule that a primitive
// with no extensions serialises as a bare value with no "_field" companion.
func anyPrimitiveExtension(elements []*PrimitiveElement) bool {
	for _, e := range elements {
		if !e.IsZero() {
			return true
		}
	}
	return false
}

// RawSibling pairs a "_field" wire key with its already-rendered JSON value, the
// unit the generated MarshalJSON appends to an encoded value object. A nil Value
// means the sibling carries nothing (an empty scalar element, or a repeating array
// with no extensions) and is skipped, so no empty "_field" key is written.
type RawSibling struct {
	Key   string
	Value []byte
}

// AppendSiblings splices the primitive "_field" siblings onto an already-encoded
// JSON object, preserving the value object's existing key order and appending the
// siblings after it. Appending (rather than decoding into a map and re-encoding)
// keeps the canonical element ordering the struct marshal produced — a map round
// trip would re-sort every key alphabetically — while still letting each "_field"
// sibling ride alongside its value. A sibling with a nil Value is skipped. The
// encoded input must be a JSON object ("{...}"); a non-object is returned unchanged
// when there is nothing to append, and is an error otherwise.
func AppendSiblings(encoded []byte, siblings []RawSibling) ([]byte, error) {
	var pending []RawSibling
	for _, s := range siblings {
		if s.Value != nil {
			pending = append(pending, s)
		}
	}
	if len(pending) == 0 {
		return encoded, nil
	}

	trimmed := bytes.TrimSpace(encoded)
	if len(trimmed) < 2 || trimmed[0] != '{' || trimmed[len(trimmed)-1] != '}' {
		return nil, errNotObject
	}

	var buf bytes.Buffer
	buf.Grow(len(trimmed) + 32*len(pending))
	buf.Write(trimmed[:len(trimmed)-1])
	// An empty object ("{}") has no trailing comma before the first sibling; a
	// populated one needs the separator between its last value key and the first
	// appended sibling.
	hasMembers := len(bytes.TrimSpace(trimmed[1:len(trimmed)-1])) > 0
	for i, s := range pending {
		if hasMembers || i > 0 {
			buf.WriteByte(',')
		}
		key, err := json.Marshal(s.Key)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteByte(':')
		buf.Write(s.Value)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// SplitRawObject decodes a FHIR JSON object into its raw key/value pairs so the
// generated UnmarshalJSON can lift out each "_field" sibling, decode it, and decode
// the rest into the struct. Decode order is irrelevant, so a map is used; the
// canonical element ordering is a marshal-side property, not a decode-side one.
func SplitRawObject(data []byte) (map[string]json.RawMessage, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// TakeRawField removes key from obj and returns its raw value and whether it was
// present, so the generated UnmarshalJSON can pull a "_field" sibling out of the
// object before decoding the remaining keys into the struct (which has no field
// bound to that key). Removing rather than copying keeps the residual object free of
// the consumed sibling key.
func TakeRawField(obj map[string]json.RawMessage, key string) (json.RawMessage, bool) {
	v, ok := obj[key]
	if ok {
		delete(obj, key)
	}
	return v, ok
}

// RemarshalObject re-encodes the residual key/value object after the "_field"
// siblings have been removed, producing the bytes the generated UnmarshalJSON feeds
// to the struct decode. The keys are emitted in encoding/json's sorted order, which
// is irrelevant to the struct decode that consumes them.
func RemarshalObject(obj map[string]json.RawMessage) ([]byte, error) {
	return json.Marshal(obj)
}
