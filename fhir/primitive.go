package fhir

import (
	"bytes"
	"encoding/json"
)

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

// SetRawField writes a raw JSON value into a decoded object under key, or removes
// the key when value is nil. The generated MarshalJSON for a type with a repeating
// primitive marshals the struct (whose null-aligned "_field" arrays are excluded),
// then folds the helper-rendered "_field" arrays back in through this function, so
// the value array and its companion array share an index space on the wire. The
// raw value is stored verbatim, so a pre-rendered, null-aligned array survives
// unchanged.
func SetRawField(obj map[string]json.RawMessage, key string, value []byte) {
	if value == nil {
		delete(obj, key)
		return
	}
	obj[key] = value
}

// SplitRawObject decodes a FHIR JSON object into its raw key/value pairs so the
// generated UnmarshalJSON can lift out each repeating "_field" array, decode it
// through UnmarshalPrimitiveExtensions, and decode the rest into the struct. The
// returned map aliases no input memory beyond what encoding/json copies.
func SplitRawObject(data []byte) (map[string]json.RawMessage, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// TakeRawField removes key from obj and returns its raw value and whether it was
// present, so the generated UnmarshalJSON can pull a repeating "_field" array out
// of the object before decoding the remaining keys into the struct (which has no
// field bound to that key). Removing rather than copying keeps the residual object
// free of the consumed sibling key.
func TakeRawField(obj map[string]json.RawMessage, key string) (json.RawMessage, bool) {
	v, ok := obj[key]
	if ok {
		delete(obj, key)
	}
	return v, ok
}

// RemarshalObject re-encodes a raw key/value object after the repeating "_field"
// arrays have been removed, producing the bytes the generated UnmarshalJSON feeds
// to the struct decode. The keys are emitted in encoding/json's sorted order, which
// is irrelevant to the struct decode that consumes them.
func RemarshalObject(obj map[string]json.RawMessage) ([]byte, error) {
	return json.Marshal(obj)
}
