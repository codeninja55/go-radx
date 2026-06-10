package dicomweb

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/codeninja55/go-radx/dicom"
)

// defaultMaxJSONDepth bounds SQ nesting on decode so a hostile document cannot drive
// unbounded recursion (PRD §9.3). It is checked before each level is descended.
const defaultMaxJSONDepth = 64

// BulkDataURI is a reference to a binary value retrieved separately (WADO-RS bulkdata).
// On encode, a value at or above the configured threshold is emitted as a BulkDataURI
// when a base URL is configured. On decode, it is left as a reference unless a
// BulkDataResolver is supplied.
type BulkDataURI string

// jsonConfig holds the codec options resolved from the JSONOption list. It is internal:
// callers configure it through the With* options.
type jsonConfig struct {
	bulkThreshold int    // values whose length is >= this emit BulkDataURI when a base URL is set; 0 disables
	bulkBaseURL   string // base URL for generated BulkDataURI references on encode
	maxDepth      int    // SQ nesting cap on decode
	bulkResolver  func(ctx context.Context, uri BulkDataURI) ([]byte, error)
	resolverCtx   context.Context
}

func defaultJSONConfig() jsonConfig {
	return jsonConfig{maxDepth: defaultMaxJSONDepth}
}

// JSONOption configures the DICOM-JSON codec.
type JSONOption func(*jsonConfig)

// WithBulkDataThreshold sets the byte size at or above which a binary value is emitted
// as a BulkDataURI on encode (when a base URL is configured) instead of InlineBinary.
// A threshold of zero disables BulkDataURI emission and always inlines.
func WithBulkDataThreshold(bytes int) JSONOption {
	return func(c *jsonConfig) { c.bulkThreshold = bytes }
}

// WithBulkDataBaseURL sets the base URL used to construct BulkDataURI references when a
// value meets the threshold. Without a base URL, over-threshold values fall back to
// InlineBinary.
func WithBulkDataBaseURL(base string) JSONOption {
	return func(c *jsonConfig) { c.bulkBaseURL = base }
}

// WithBulkDataResolver registers a function invoked for each BulkDataURI encountered on
// decode; its bytes replace the reference. A nil resolver (the default) leaves the value
// as a reference. The resolver receives the context passed via WithResolverContext, or
// context.Background otherwise.
func WithBulkDataResolver(fn func(ctx context.Context, uri BulkDataURI) ([]byte, error)) JSONOption {
	return func(c *jsonConfig) { c.bulkResolver = fn }
}

// WithResolverContext sets the context passed to a BulkDataResolver on decode so a slow
// fetch honours cancellation (PRD §9.4).
func WithResolverContext(ctx context.Context) JSONOption {
	return func(c *jsonConfig) { c.resolverCtx = ctx }
}

// WithMaxJSONDepth sets the maximum SQ nesting depth accepted on decode (default 64).
// The limit guards recursion before allocation; exceeding it returns *LimitExceededError.
func WithMaxJSONDepth(n int) JSONOption {
	return func(c *jsonConfig) { c.maxDepth = n }
}

// MarshalJSON encodes ds as DICOM JSON (PS3.18 Annex F). The result is a JSON object
// keyed by eight-hex-digit uppercase GGGGEEEE tag strings; each value is an object
// carrying "vr" and at most one of "Value", "BulkDataURI", or "InlineBinary". A PN
// element renders as the Alphabetic/Ideographic/Phonetic component-group form, SQ nests
// DICOM-JSON DataSet objects, numeric VRs emit JSON numbers, and string VRs emit string
// arrays. A nil dataset marshals to the empty object.
func MarshalJSON(ds *dicom.DataSet, opts ...JSONOption) ([]byte, error) {
	cfg := defaultJSONConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	obj, err := marshalDataSet(ds, &cfg, "")
	if err != nil {
		return nil, err
	}
	return json.Marshal(obj)
}

// marshalDataSet builds the ordered map of tag-key to element object. encoding/json
// sorts string keys, and the eight-hex tag keys sort identically to ascending tag
// order, so the output is deterministic and byte-stable for canonical input. path is
// the DICOM-JSON locator prefix used to build unique BulkDataURI references for nested
// binary attributes; it is empty at the top level.
func marshalDataSet(ds *dicom.DataSet, cfg *jsonConfig, path string) (map[string]jsonElement, error) {
	out := make(map[string]jsonElement)
	if ds == nil {
		return out, nil
	}
	for e := range ds.All() {
		je, err := marshalElement(e, cfg, path)
		if err != nil {
			return nil, err
		}
		out[tagKey(e.Tag)] = je
	}
	return out, nil
}

// jsonElement is the on-wire shape of one DICOM-JSON attribute. Exactly one of Value,
// BulkDataURI, or InlineBinary is populated (or none, for an empty element).
type jsonElement struct {
	VR           string            `json:"vr"`
	Value        []json.RawMessage `json:"Value,omitempty"`
	BulkDataURI  string            `json:"BulkDataURI,omitempty"`
	InlineBinary string            `json:"InlineBinary,omitempty"`
}

// UnmarshalJSON decodes an attribute, distinguishing an absent payload from a present
// but empty or null one so malformed input fails closed. DICOM-JSON represents an empty
// element by omitting the payload key entirely; a present "Value": null, a present empty
// "InlineBinary": "", or a present empty "BulkDataURI": "" is malformed (PS3.18 Annex F).
func (je *jsonElement) UnmarshalJSON(data []byte) error {
	var raw struct {
		VR           string          `json:"vr"`
		Value        json.RawMessage `json:"Value"`
		BulkDataURI  json.RawMessage `json:"BulkDataURI"`
		InlineBinary json.RawMessage `json:"InlineBinary"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	je.VR = raw.VR

	if raw.InlineBinary != nil {
		s, err := decodePresentString(raw.InlineBinary, "InlineBinary")
		if err != nil {
			return err
		}
		je.InlineBinary = s
	}
	if raw.BulkDataURI != nil {
		s, err := decodePresentString(raw.BulkDataURI, "BulkDataURI")
		if err != nil {
			return err
		}
		je.BulkDataURI = s
	}
	if raw.Value != nil {
		if isJSONNull(raw.Value) {
			return &DecodeError{Msg: "Value is null, omit it for an empty element"}
		}
		if err := json.Unmarshal(raw.Value, &je.Value); err != nil {
			return err
		}
		// A present but empty Value array is recorded as a non-nil zero-length slice so
		// the "binary VR carries a Value array" guard still distinguishes it from absent.
		if je.Value == nil {
			je.Value = []json.RawMessage{}
		}
	}
	return nil
}

// decodePresentString decodes a present InlineBinary/BulkDataURI payload string, rejecting
// a null or empty value: a present payload key must carry a non-empty value (an empty
// element omits the key entirely).
func decodePresentString(raw json.RawMessage, field string) (string, error) {
	if isJSONNull(raw) {
		return "", &DecodeError{Msg: field + " is null, omit it for an empty element"}
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", &DecodeError{Msg: field + " is not a string"}
	}
	if s == "" {
		return "", &DecodeError{Msg: field + " is empty, omit it for an empty element"}
	}
	return s, nil
}

// tagKey renders a Tag as its eight-hex-digit uppercase DICOM-JSON key.
func tagKey(t dicom.Tag) string {
	return fmt.Sprintf("%04X%04X", t.Group(), t.Element())
}

// marshalElement encodes one element to its DICOM-JSON object form. path is the locator
// prefix of the enclosing dataset, extended here with this element's tag so a generated
// BulkDataURI is unique even for the same tag in different sequence items.
func marshalElement(e dicom.Element, cfg *jsonConfig, path string) (jsonElement, error) {
	// A dictionary ambiguity placeholder (US or SS, OB or OW, ...) is not a valid on-wire
	// VR: a properly read dataset resolves it to a concrete VR. Emitting its String() form
	// would produce a vr the decoder rejects, so fail closed (the dataset must be read or
	// constructed with a resolved VR first).
	if !isConcreteVR(e.VR) {
		return jsonElement{}, &EncodeError{Tag: e.Tag, VR: e.VR, Msg: "ambiguous VR has no DICOM JSON representation; resolve it to a concrete VR"}
	}
	je := jsonElement{VR: e.VR.String()}
	elemPath := joinPath(path, tagKey(e.Tag))

	switch v := e.Value.(type) {
	case *dicom.Strings:
		vals := v.Strings()
		raws, err := marshalStringValues(e.VR, vals)
		if err != nil {
			return jsonElement{}, err
		}
		je.Value = raws
	case *dicom.Ints:
		raws, err := marshalIntValues(e.VR, v.Ints())
		if err != nil {
			return jsonElement{}, err
		}
		je.Value = raws
	case *dicom.Floats:
		if isOtherFloatVR(e.VR) {
			// OF/OD are "Other" binary VRs in Annex F.2.6: serialise to little-endian
			// bytes and emit InlineBinary/BulkDataURI, never a numeric Value array.
			return marshalBinary(e, otherFloatBytes(e.VR, v.Floats()), cfg, elemPath)
		}
		raws, err := marshalFloatValues(e.VR, v.Floats())
		if err != nil {
			return jsonElement{}, err
		}
		je.Value = raws
	case *dicom.Decimals:
		raws, err := marshalDecimalValues(e.VR, v.Decimals())
		if err != nil {
			return jsonElement{}, err
		}
		je.Value = raws
	case *dicom.Tags:
		je.Value = marshalTagValues(v.Tags())
	case *dicom.Bytes:
		return marshalBinary(e, v.Bytes(), cfg, elemPath)
	case *bulkRef:
		je.BulkDataURI = string(v.URI())
		return je, nil
	default:
		// SQ is wrapped in an unexported sequenceValue; the public accessor exposes it.
		if seq, ok := sequenceFromValue(e); ok {
			raws, err := marshalSequence(seq, cfg, elemPath)
			if err != nil {
				return jsonElement{}, err
			}
			je.Value = raws
			return je, nil
		}
		return jsonElement{}, &EncodeError{Tag: e.Tag, VR: e.VR, Msg: "unsupported value type"}
	}
	return je, nil
}

// jsonNull is the literal used for an absent value within a non-empty multi-valued
// attribute (PS3.18 F.2.5).
var jsonNull = json.RawMessage("null")

// marshalStringValues encodes a text VR's values. PN renders as the component-group
// object form; every other string VR renders as a JSON string. A logically empty
// element (no values, or a single empty value, VM<=1) collapses to no Value array so the
// encoder's empty-element form matches what decoding an empty element produces. A
// multi-valued attribute with empty components keeps its multiplicity: each empty
// component is emitted as the null placeholder so a dataset decoded from
// Value:["A",null,"B"] or Value:[null,null] re-marshals without losing VM (PS3.18 F.2.5).
func marshalStringValues(vr dicom.VR, vals []string) ([]json.RawMessage, error) {
	if len(vals) <= 1 && allEmpty(vals) {
		return nil, nil
	}
	out := make([]json.RawMessage, 0, len(vals))
	for _, s := range vals {
		if s == "" {
			out = append(out, jsonNull)
			continue
		}
		if vr == dicom.VRPN {
			raw, err := marshalPersonName(s)
			if err != nil {
				return nil, err
			}
			out = append(out, raw)
			continue
		}
		raw, err := json.Marshal(s)
		if err != nil {
			return nil, err
		}
		out = append(out, raw)
	}
	return out, nil
}

// allEmpty reports whether vals is empty or every entry is the empty string: the
// encoded form of a logically empty text element.
func allEmpty(vals []string) bool {
	for _, s := range vals {
		if s != "" {
			return false
		}
	}
	return true
}

// personNameJSON is the Alphabetic/Ideographic/Phonetic component-group object. Empty
// groups are omitted so an alphabetic-only name renders as {"Alphabetic":"Doe^Jane"}.
type personNameJSON struct {
	Alphabetic  string `json:"Alphabetic,omitempty"`
	Ideographic string `json:"Ideographic,omitempty"`
	Phonetic    string `json:"Phonetic,omitempty"`
}

func marshalPersonName(s string) (json.RawMessage, error) {
	pn, err := dicom.ParsePersonName(s)
	if err != nil {
		return nil, &EncodeError{VR: dicom.VRPN, Msg: "invalid PN value"}
	}
	obj := personNameJSON{
		Alphabetic:  componentGroup(pn.Alphabetic),
		Ideographic: componentGroup(pn.Ideographic),
		Phonetic:    componentGroup(pn.Phonetic),
	}
	return json.Marshal(obj)
}

// componentGroup renders one PN component group to its caret-delimited form, dropping
// trailing empty components (so "Doe^Jane" not "Doe^Jane^^^"). It reuses PersonName's
// canonical String by placing the group alone in the alphabetic slot.
func componentGroup(c dicom.NameComponents) string {
	return dicom.PersonName{Alphabetic: c}.String()
}

// marshalIntValues encodes SS/US/SL/UL/SV/UV. The 64-bit VRs (SV/UV) are emitted as JSON
// strings per Annex F.2.3 because their full range exceeds the IEEE-754 double a JSON
// number guarantees, so a value above 2^53 would lose precision; the narrower VRs are
// emitted as JSON numbers. A value out of range for its VR (for example a negative US)
// is a hard encode error, mirroring the decode-side range check, so the encoder never
// emits non-conformant JSON that a binary write would later wrap.
func marshalIntValues(vr dicom.VR, vals []int64) ([]json.RawMessage, error) {
	out := make([]json.RawMessage, 0, len(vals))
	quote := vr == dicom.VRSV || vr == dicom.VRUV
	for _, n := range vals {
		if !intFitsVR(vr, n) {
			return nil, &EncodeError{VR: vr, Msg: "integer value out of range for its VR"}
		}
		s := strconv.FormatInt(n, 10)
		if quote {
			s = `"` + s + `"`
		}
		out = append(out, json.RawMessage(s))
	}
	return out, nil
}

// otherFloatBytes serialises OF/OD float values to their little-endian byte form (OF as
// 32-bit, OD as 64-bit) for InlineBinary/BulkDataURI emission per Annex F.2.6.
func otherFloatBytes(vr dicom.VR, vals []float64) []byte {
	if vr == dicom.VROD {
		b := make([]byte, len(vals)*8)
		for i, f := range vals {
			binary.LittleEndian.PutUint64(b[i*8:], math.Float64bits(f))
		}
		return b
	}
	b := make([]byte, len(vals)*4)
	for i, f := range vals {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(float32(f)))
	}
	return b
}

// marshalFloatValues encodes the JSON-numeric float VRs FL/FD. JSON has no representation
// for NaN or ±Inf, so a non-finite value is a hard encode error, never a silent null
// (PRD §9.2).
func marshalFloatValues(vr dicom.VR, vals []float64) ([]json.RawMessage, error) {
	out := make([]json.RawMessage, 0, len(vals))
	for _, f := range vals {
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return nil, &EncodeError{VR: vr, Msg: "non-finite float has no JSON representation"}
		}
		raw, err := json.Marshal(f)
		if err != nil {
			return nil, &EncodeError{VR: vr, Msg: "cannot encode float value"}
		}
		out = append(out, raw)
	}
	return out, nil
}

// marshalDecimalValues encodes DS/IS preserving the lexical form. Annex F.2.3 permits
// DS/IS as a JSON Number or String; a lexical form that is not a valid JSON number token
// (for example a leading "+", which DICOM allows but JSON forbids) is emitted as a quoted
// string so the value is preserved exactly and the document stays valid JSON. The decoder
// strips the quotes, so both forms round-trip. A fractional IS value is non-conformant
// (IS is Integer String) and is a hard encode error, mirroring the decode-side check.
func marshalDecimalValues(vr dicom.VR, vals []dicom.Decimal) ([]json.RawMessage, error) {
	out := make([]json.RawMessage, 0, len(vals))
	for _, d := range vals {
		// A zero-value (default-initialised) Decimal has no lexical form and marshals to
		// null; fail closed rather than emit a "Value":[null] the decoder would reject.
		if d.String() == "" {
			return nil, &EncodeError{VR: vr, Msg: "decimal value is empty"}
		}
		if vr == dicom.VRIS {
			if _, ok := d.Int64(); !ok {
				return nil, &EncodeError{VR: vr, Msg: "IS value is not an integer"}
			}
		}
		raw, _ := d.MarshalJSON()
		if !json.Valid(raw) {
			quoted, _ := json.Marshal(d.String())
			raw = quoted
		}
		out = append(out, raw)
	}
	return out, nil
}

// marshalTagValues encodes AT values as eight-hex-digit strings (Annex F.2.3).
func marshalTagValues(vals []dicom.Tag) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(vals))
	for _, t := range vals {
		raw, _ := json.Marshal(tagKey(t))
		out = append(out, raw)
	}
	return out
}

// marshalBinary encodes OB/OW/OL/OV/UN (and OF/OD serialised to bytes). Above the
// configured threshold with a base URL, the value becomes a BulkDataURI keyed by its full
// locator path so two same-tag attributes in different sequence items get distinct URIs;
// otherwise it is base64 InlineBinary (Annex F.2.6). elemPath is this element's locator.
// A fixed-width VR (OW/OL/OV) whose byte length is not a multiple of its word size is a
// hard encode error, mirroring the decode-side check, so MarshalJSON never emits a
// payload its own decoder rejects.
func marshalBinary(e dicom.Element, b []byte, cfg *jsonConfig, elemPath string) (jsonElement, error) {
	if w := fixedBinaryWidth(e.VR); w > 1 && len(b)%w != 0 {
		return jsonElement{}, &EncodeError{Tag: e.Tag, VR: e.VR, Msg: "binary payload length is not a multiple of the VR word size"}
	}
	je := jsonElement{VR: e.VR.String()}
	if cfg.bulkThreshold > 0 && len(b) >= cfg.bulkThreshold && cfg.bulkBaseURL != "" {
		je.BulkDataURI = cfg.bulkBaseURL + elemPath
		return je, nil
	}
	je.InlineBinary = base64.StdEncoding.EncodeToString(b)
	return je, nil
}

// fixedBinaryWidth returns the word size of a fixed-width binary VR (OW=2, OL=4, OV=8) or
// 1 for byte-aligned VRs (OB/UN) that accept any length.
func fixedBinaryWidth(vr dicom.VR) int {
	switch vr {
	case dicom.VROW:
		return 2
	case dicom.VROL:
		return 4
	case dicom.VROV:
		return 8
	default:
		return 1
	}
}

// joinPath extends a DICOM-JSON locator prefix with the next segment, slash-separated.
func joinPath(prefix, segment string) string {
	if prefix == "" {
		return segment
	}
	return prefix + "/" + segment
}

// marshalSequence encodes an SQ value as an array of nested DICOM-JSON DataSet objects.
// path is the SQ element's locator; each item extends it with its index so nested
// binary attributes get unique BulkDataURI references.
func marshalSequence(seq *dicom.Sequence, cfg *jsonConfig, path string) ([]json.RawMessage, error) {
	out := make([]json.RawMessage, 0, seq.Len())
	i := 0
	for item := range seq.Items() {
		obj, err := marshalDataSet(item.DataSet, cfg, joinPath(path, strconv.Itoa(i)))
		if err != nil {
			return nil, err
		}
		raw, err := json.Marshal(obj)
		if err != nil {
			return nil, err
		}
		out = append(out, raw)
		i++
	}
	return out, nil
}

// sequenceFromValue extracts the *dicom.Sequence backing an SQ element. The SQ value is
// wrapped in an unexported type, so the only public route is to round-trip through a
// throwaway DataSet's GetSequence accessor.
func sequenceFromValue(e dicom.Element) (*dicom.Sequence, bool) {
	if e.VR != dicom.VRSQ {
		return nil, false
	}
	probe := dicom.NewDataSet()
	probe.Set(e)
	return probe.GetSequence(e.Tag)
}

// UnmarshalJSON decodes DICOM JSON into a DataSet (PS3.18 Annex F). InlineBinary is
// base64-decoded under the declared VR; a BulkDataURI is left as an empty reference
// unless a resolver was supplied via WithBulkDataResolver. SQ nesting is bounded by the
// configured depth (default 64); exceeding it returns *LimitExceededError, never a panic.
func UnmarshalJSON(data []byte, opts ...JSONOption) (*dicom.DataSet, error) {
	cfg := defaultJSONConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.resolverCtx == nil {
		cfg.resolverCtx = context.Background()
	}

	if isJSONNull(data) {
		return nil, &DecodeError{Msg: "document is null, expected a DICOM JSON object"}
	}
	var doc map[string]jsonElement
	if err := json.Unmarshal(data, &doc); err != nil {
		// A body cut off mid-object is a truncated transfer, not merely malformed JSON:
		// surface it as the typed truncation error so callers can distinguish a short
		// transfer from a syntactically invalid one (PRD §9.2).
		if isJSONEndOfInput(err) {
			return nil, &TruncatedError{Detail: "DICOM JSON document ended mid-object", err: io.ErrUnexpectedEOF}
		}
		return nil, fmt.Errorf("dicomweb: decode DICOM JSON: %w", err)
	}
	return unmarshalObject(doc, &cfg, 0)
}

// isJSONEndOfInput reports whether err is encoding/json's premature end-of-input error.
// The standard library surfaces it as io.ErrUnexpectedEOF or a *json.SyntaxError whose
// message is "unexpected end of JSON input"; this checks both forms.
func isJSONEndOfInput(err error) bool {
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var se *json.SyntaxError
	if errors.As(err, &se) {
		return se.Error() == "unexpected end of JSON input"
	}
	return false
}

// unmarshalObject decodes one DICOM-JSON object into a DataSet. depth is the current SQ
// nesting level: 0 for the top-level dataset (which is not an SQ level), incremented once
// per sequence descent. It is validated against the cap before recursing, so
// WithMaxJSONDepth(n) admits exactly n levels of SQ nesting and a flat dataset always
// decodes.
func unmarshalObject(doc map[string]jsonElement, cfg *jsonConfig, depth int) (*dicom.DataSet, error) {
	if depth > cfg.maxDepth {
		return nil, &LimitExceededError{
			Limit:  uint64(cfg.maxDepth), // #nosec G115 -- small non-negative configured depth cap
			Actual: uint64(depth),        // #nosec G115 -- small non-negative recursion counter
			Kind:   "json-sequence-depth",
		}
	}
	ds := dicom.NewDataSet()
	// Decode in ascending tag-key order so a re-encode is byte-stable regardless of the
	// map iteration order the JSON decoder produced.
	keys := make([]string, 0, len(doc))
	for k := range doc {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		je := doc[key]
		tag, err := parseTagKey(key)
		if err != nil {
			return nil, err
		}
		vr, err := parseVR(je.VR, tag)
		if err != nil {
			return nil, err
		}
		value, err := decodeValue(tag, vr, je, cfg, depth)
		if err != nil {
			return nil, err
		}
		ds.Set(dicom.Element{Tag: tag, VR: vr, Value: value})
	}
	return ds, nil
}

// decodeValue reconstructs the dicom.Value for one attribute from its DICOM-JSON form.
func decodeValue(tag dicom.Tag, vr dicom.VR, je jsonElement, cfg *jsonConfig, depth int) (dicom.Value, error) {
	// Annex F requires exactly one (or, for an empty element, none) of Value,
	// BulkDataURI, or InlineBinary. A document carrying more than one is malformed:
	// rejecting it avoids storing data that contradicts the metadata.
	if payloadFormCount(je) > 1 {
		return nil, &DecodeError{Tag: tag, VR: vr, Msg: "attribute declares more than one of Value, BulkDataURI, InlineBinary"}
	}

	binaryVR := isByteVR(vr) || isOtherFloatVR(vr)

	if je.InlineBinary != "" {
		if !binaryVR {
			return nil, &DecodeError{Tag: tag, VR: vr, Msg: "InlineBinary on a non-binary VR"}
		}
		raw, err := base64.StdEncoding.DecodeString(je.InlineBinary)
		if err != nil {
			return nil, &DecodeError{Tag: tag, VR: vr, Msg: "invalid InlineBinary base64"}
		}
		return decodeBinaryPayload(tag, vr, raw)
	}
	if je.BulkDataURI != "" {
		if !binaryVR {
			return nil, &DecodeError{Tag: tag, VR: vr, Msg: "BulkDataURI on a non-binary VR"}
		}
		if cfg.bulkResolver == nil {
			// Leave the reference intact so it re-marshals unchanged (the decode contract).
			return newBulkRef(vr, BulkDataURI(je.BulkDataURI)), nil
		}
		raw, err := cfg.bulkResolver(cfg.resolverCtx, BulkDataURI(je.BulkDataURI))
		if err != nil {
			return nil, fmt.Errorf("dicomweb: resolve BulkDataURI for %s %s: %w", keywordOf(tag), tag, err)
		}
		return decodeBinaryPayload(tag, vr, raw)
	}

	if vr == dicom.VRSQ {
		return decodeSequence(je.Value, cfg, depth)
	}
	if binaryVR {
		// A binary VR is carried as InlineBinary/BulkDataURI, never a Value array. With
		// neither present it is a zero-length value; a Value key here is malformed.
		if je.Value != nil {
			return nil, &DecodeError{Tag: tag, VR: vr, Msg: "binary VR carries a Value array instead of InlineBinary or BulkDataURI"}
		}
		return decodeBinaryPayload(tag, vr, nil)
	}
	return decodeNonBinary(tag, vr, je.Value)
}

// payloadFormCount counts how many of Value, BulkDataURI, and InlineBinary are present.
func payloadFormCount(je jsonElement) int {
	n := 0
	if je.Value != nil {
		n++
	}
	if je.BulkDataURI != "" {
		n++
	}
	if je.InlineBinary != "" {
		n++
	}
	return n
}

// decodeBinaryPayload reconstructs the native value type from raw bytes: OF/OD become
// *dicom.Floats (the dicom model's float representation), every other binary VR becomes
// *dicom.Bytes. Little-endian is the DICOM-JSON binary convention.
func decodeBinaryPayload(tag dicom.Tag, vr dicom.VR, raw []byte) (dicom.Value, error) {
	switch vr {
	case dicom.VROF:
		if len(raw)%4 != 0 {
			return nil, &DecodeError{Tag: tag, VR: vr, Msg: "OF payload length is not a multiple of 4"}
		}
		fs := make([]float64, 0, len(raw)/4)
		for i := 0; i < len(raw); i += 4 {
			fs = append(fs, float64(math.Float32frombits(binary.LittleEndian.Uint32(raw[i:i+4]))))
		}
		return dicom.NewFloats(vr, fs...), nil
	case dicom.VROD:
		if len(raw)%8 != 0 {
			return nil, &DecodeError{Tag: tag, VR: vr, Msg: "OD payload length is not a multiple of 8"}
		}
		fs := make([]float64, 0, len(raw)/8)
		for i := 0; i < len(raw); i += 8 {
			fs = append(fs, math.Float64frombits(binary.LittleEndian.Uint64(raw[i:i+8])))
		}
		return dicom.NewFloats(vr, fs...), nil
	case dicom.VROW:
		if len(raw)%2 != 0 {
			return nil, &DecodeError{Tag: tag, VR: vr, Msg: "OW payload length is not a multiple of 2"}
		}
		return dicom.NewBytes(vr, raw), nil
	case dicom.VROL:
		if len(raw)%4 != 0 {
			return nil, &DecodeError{Tag: tag, VR: vr, Msg: "OL payload length is not a multiple of 4"}
		}
		return dicom.NewBytes(vr, raw), nil
	case dicom.VROV:
		if len(raw)%8 != 0 {
			return nil, &DecodeError{Tag: tag, VR: vr, Msg: "OV payload length is not a multiple of 8"}
		}
		return dicom.NewBytes(vr, raw), nil
	default:
		return dicom.NewBytes(vr, raw), nil
	}
}

// decodeNonBinary reconstructs a string, numeric, decimal, or AT value from its Value
// array. An empty or absent Value array yields a zero-length value of the declared VR.
func decodeNonBinary(tag dicom.Tag, vr dicom.VR, vals []json.RawMessage) (dicom.Value, error) {
	switch {
	case isStringVR(vr):
		strs := make([]string, 0, len(vals))
		for _, raw := range vals {
			s, err := decodeStringValue(vr, raw)
			if err != nil {
				return nil, &DecodeError{Tag: tag, VR: vr, Msg: err.Error()}
			}
			strs = append(strs, s)
		}
		return dicom.NewStrings(vr, strs...), nil
	case isIntVR(vr):
		ns := make([]int64, 0, len(vals))
		for _, raw := range vals {
			n, err := decodeIntValue(vr, raw)
			if err != nil {
				return nil, &DecodeError{Tag: tag, VR: vr, Msg: err.Error()}
			}
			if !intFitsVR(vr, n) {
				return nil, &DecodeError{Tag: tag, VR: vr, Msg: "integer value out of range for its VR"}
			}
			ns = append(ns, n)
		}
		return dicom.NewInts(vr, ns...), nil
	case isFloatVR(vr):
		fs := make([]float64, 0, len(vals))
		for _, raw := range vals {
			// encoding/json treats a JSON null as a no-op when decoding into float64,
			// leaving a real zero; reject it so malformed input is never read as 0.
			if isJSONNull(raw) {
				return nil, &DecodeError{Tag: tag, VR: vr, Msg: "null is not a valid float value"}
			}
			var f float64
			if err := json.Unmarshal(raw, &f); err != nil {
				return nil, &DecodeError{Tag: tag, VR: vr, Msg: "invalid float value"}
			}
			fs = append(fs, f)
		}
		return dicom.NewFloats(vr, fs...), nil
	case vr == dicom.VRDS || vr == dicom.VRIS:
		decs := make([]dicom.Decimal, 0, len(vals))
		for _, raw := range vals {
			d, err := decodeDecimalValue(raw)
			if err != nil {
				return nil, &DecodeError{Tag: tag, VR: vr, Msg: "invalid decimal value"}
			}
			if vr == dicom.VRIS {
				// IS is Integer String: a fractional value is malformed, not a valid IS.
				if _, ok := d.Int64(); !ok {
					return nil, &DecodeError{Tag: tag, VR: vr, Msg: "IS value is not an integer"}
				}
			}
			decs = append(decs, d)
		}
		return dicom.NewDecimals(vr, decs...), nil
	case vr == dicom.VRAT:
		tags := make([]dicom.Tag, 0, len(vals))
		for _, raw := range vals {
			at, err := decodeATValue(raw)
			if err != nil {
				return nil, &DecodeError{Tag: tag, VR: vr, Msg: "invalid AT value"}
			}
			tags = append(tags, at)
		}
		return dicom.NewTags(tags...), nil
	default:
		// Treat any other VR as opaque text rather than failing the whole document.
		strs := make([]string, 0, len(vals))
		for _, raw := range vals {
			s, err := decodeStringValue(vr, raw)
			if err != nil {
				return nil, &DecodeError{Tag: tag, VR: vr, Msg: err.Error()}
			}
			strs = append(strs, s)
		}
		return dicom.NewStrings(vr, strs...), nil
	}
}

// decodeStringValue decodes one Value entry of a string VR. PN entries are the
// component-group object; every other string VR is a plain JSON string. A null array
// entry represents an absent value within a multi-valued attribute (PS3.18 F.2.5) and
// maps to an empty string; a top-level "Value": null is rejected earlier as malformed.
func decodeStringValue(vr dicom.VR, raw json.RawMessage) (string, error) {
	if isJSONNull(raw) {
		return "", nil
	}
	if vr == dicom.VRPN {
		return decodePersonName(raw)
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", fmt.Errorf("not a string")
	}
	// For VRs that use backslash as the value-multiplicity delimiter, an embedded
	// backslash in a single JSON value would be re-split into multiple values by the
	// Part-10 writer; reject it. LT/ST/UT/UR permit a literal backslash (no VM).
	if usesBackslashDelimiter(vr) && strings.ContainsRune(s, '\\') {
		return "", fmt.Errorf("value contains the value-multiplicity separator '\\'")
	}
	return s, nil
}

// usesBackslashDelimiter reports whether vr treats backslash as the value-multiplicity
// separator. The long-text VRs LT/ST/UT and the URI VR UR are single-valued and permit a
// literal backslash (PS3.5 Table 6.2-1); every other string VR uses it as a delimiter.
func usesBackslashDelimiter(vr dicom.VR) bool {
	switch vr {
	case dicom.VRLT, dicom.VRST, dicom.VRUT, dicom.VRUR:
		return false
	default:
		return isStringVR(vr)
	}
}

func decodePersonName(raw json.RawMessage) (string, error) {
	if isJSONNull(raw) {
		return "", nil // absent value within a multi-valued PN attribute
	}
	var obj personNameJSON
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", fmt.Errorf("not a PN component-group object")
	}
	return composePersonName(obj)
}

// isJSONNull reports whether raw is the JSON null literal.
func isJSONNull(raw json.RawMessage) bool {
	return string(bytes.TrimSpace(raw)) == "null"
}

// composePersonName assembles the canonical "=" / "^" PN string from the three groups,
// dropping trailing empty groups. A component-group field carrying a DICOM delimiter is
// rejected: "=" is the group separator and "\" is the value-multiplicity separator, so an
// embedded one would later be re-parsed as a real boundary and shift the patient-name
// data into the wrong groups or split it into extra values (PRD §9.2).
func composePersonName(obj personNameJSON) (string, error) {
	groups := []string{obj.Alphabetic, obj.Ideographic, obj.Phonetic}
	for _, g := range groups {
		if strings.ContainsRune(g, '=') {
			return "", fmt.Errorf("PN component group contains the group separator '='")
		}
		if strings.ContainsRune(g, '\\') {
			return "", fmt.Errorf("PN component group contains the value separator '\\'")
		}
	}
	end := len(groups)
	for end > 0 && groups[end-1] == "" {
		end--
	}
	joined := ""
	for i := 0; i < end; i++ {
		if i > 0 {
			joined += "="
		}
		joined += groups[i]
	}
	// Validate through the dicom parser so a malformed component group is rejected.
	if _, err := dicom.ParsePersonName(joined); err != nil {
		return "", fmt.Errorf("invalid PN value")
	}
	return joined, nil
}

func decodeIntValue(vr dicom.VR, raw json.RawMessage) (int64, error) {
	// Annex F allows narrow integers as JSON numbers and 64-bit VRs (SV/UV) as strings.
	// A fractional number (e.g. 1.5) is not a valid integer value and is rejected rather
	// than truncated, so malformed input never silently changes the stored value.
	s := numericToken(raw)

	// Unsigned VRs parse as uint64 so a large positive UL/UV is not rejected as out of
	// int64 range before the VR width check. The dicom data model stores integers as
	// int64, so a UV above math.MaxInt64 exceeds what the model can hold and is reported
	// as such rather than silently wrapped.
	if vr == dicom.VRUL || vr == dicom.VRUV {
		u, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("not an unsigned integer")
		}
		if u > math.MaxInt64 {
			return 0, fmt.Errorf("unsigned value exceeds the int64-backed model range")
		}
		return int64(u), nil
	}

	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("not an integer")
	}
	return n, nil
}

func decodeDecimalValue(raw json.RawMessage) (dicom.Decimal, error) {
	return dicom.ParseDecimal(numericToken(raw))
}

// numericToken returns the textual form of a DICOM-JSON numeric value, accepting either
// a JSON number or a JSON string (Annex F.2.3 permits DS/IS and the 64-bit VRs as
// strings). A quoted token is decoded through json.Unmarshal so standard JSON escapes
// (for example "+2.5") are resolved rather than parsed literally.
func numericToken(raw json.RawMessage) string {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) > 0 && trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(trimmed, &s); err == nil {
			return s
		}
	}
	return string(trimmed)
}

func decodeATValue(raw json.RawMessage) (dicom.Tag, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, err
	}
	return parseTagKey(s)
}

// decodeSequence reconstructs an SQ value from its array of nested DICOM-JSON objects,
// descending one nesting level (depth+1). The depth cap is checked before the nested
// item is unmarshalled so an over-deep document is rejected before parsing the forbidden
// level into memory (PRD §9.3; guard recursion before allocation).
func decodeSequence(vals []json.RawMessage, cfg *jsonConfig, depth int) (dicom.Value, error) {
	if len(vals) > 0 && depth+1 > cfg.maxDepth {
		return nil, &LimitExceededError{
			Limit:  uint64(cfg.maxDepth), // #nosec G115 -- small non-negative configured depth cap
			Actual: uint64(depth + 1),    // #nosec G115 -- small non-negative recursion counter
			Kind:   "json-sequence-depth",
		}
	}
	seq := dicom.NewSequence()
	for _, raw := range vals {
		if isJSONNull(raw) {
			return nil, &DecodeError{VR: dicom.VRSQ, Msg: "sequence item is null, expected a dataset object"}
		}
		var obj map[string]jsonElement
		if err := json.Unmarshal(raw, &obj); err != nil {
			return nil, &DecodeError{VR: dicom.VRSQ, Msg: "invalid sequence item object"}
		}
		nested, err := unmarshalObject(obj, cfg, depth+1)
		if err != nil {
			return nil, err
		}
		seq.Append(nested)
	}
	return dicom.NewSequenceValue(seq), nil
}

// parseTagKey parses an eight-hex-digit GGGGEEEE key into a Tag. Error messages name
// only the structural fault and the key length, never the key itself: a malformed key is
// attacker-controlled and could carry PHI or be arbitrarily large (PRD §9.1).
func parseTagKey(key string) (dicom.Tag, error) {
	if len(key) != 8 {
		return 0, &DecodeError{Msg: fmt.Sprintf("tag key is not 8 hex digits (length %d)", len(key))}
	}
	g, err := strconv.ParseUint(key[:4], 16, 16)
	if err != nil {
		return 0, &DecodeError{Msg: "tag key has a non-hex group"}
	}
	e, err := strconv.ParseUint(key[4:], 16, 16)
	if err != nil {
		return 0, &DecodeError{Msg: "tag key has a non-hex element"}
	}
	return dicom.NewTag(uint16(g), uint16(e)), nil
}

// parseVR resolves the two-letter VR string to the dicom.VR. An unknown VR is rejected
// with a typed DecodeError naming the tag, never the supplied VR text: a malformed
// document's vr field is attacker-controlled and could carry PHI (PRD §9.1).
func parseVR(s string, tag dicom.Tag) (dicom.VR, error) {
	if vr, ok := vrByName[s]; ok {
		return vr, nil
	}
	return 0, &DecodeError{Tag: tag, Msg: "unknown VR"}
}

// isConcreteVR reports whether vr is one of the 34 real on-wire VRs (in vrByName) rather
// than a dictionary ambiguity placeholder (US or SS, OB or OW, ...).
func isConcreteVR(vr dicom.VR) bool {
	_, ok := vrByName[vr.String()]
	return ok
}

// vrByName is the reverse index from the two-letter VR string to the dicom.VR constant.
var vrByName = func() map[string]dicom.VR {
	all := []dicom.VR{
		dicom.VRAE, dicom.VRAS, dicom.VRAT, dicom.VRCS, dicom.VRDA, dicom.VRDS,
		dicom.VRDT, dicom.VRFL, dicom.VRFD, dicom.VRIS, dicom.VRLO, dicom.VRLT,
		dicom.VROB, dicom.VROD, dicom.VROF, dicom.VROL, dicom.VROV, dicom.VROW,
		dicom.VRPN, dicom.VRSH, dicom.VRSL, dicom.VRSQ, dicom.VRSS, dicom.VRST,
		dicom.VRSV, dicom.VRTM, dicom.VRUC, dicom.VRUI, dicom.VRUL, dicom.VRUN,
		dicom.VRUR, dicom.VRUS, dicom.VRUT, dicom.VRUV,
	}
	m := make(map[string]dicom.VR, len(all))
	for _, vr := range all {
		m[vr.String()] = vr
	}
	return m
}()

func isStringVR(vr dicom.VR) bool {
	switch vr {
	case dicom.VRAE, dicom.VRAS, dicom.VRCS, dicom.VRDA, dicom.VRDT, dicom.VRLO,
		dicom.VRLT, dicom.VRPN, dicom.VRSH, dicom.VRST, dicom.VRTM, dicom.VRUC,
		dicom.VRUI, dicom.VRUR, dicom.VRUT:
		return true
	default:
		return false
	}
}

func isIntVR(vr dicom.VR) bool {
	switch vr {
	case dicom.VRSS, dicom.VRUS, dicom.VRSL, dicom.VRUL, dicom.VRSV, dicom.VRUV:
		return true
	default:
		return false
	}
}

// intFitsVR reports whether n is in range for its integer VR. An unsigned VR rejects a
// negative value (which would wrap on binary encode), and each VR's width is enforced.
// UV's full range exceeds int64, so the dicom model caps it at the int64 ceiling; a
// non-negative int64 is therefore always in range for UV.
func intFitsVR(vr dicom.VR, n int64) bool {
	switch vr {
	case dicom.VRUS:
		return n >= 0 && n <= math.MaxUint16
	case dicom.VRSS:
		return n >= math.MinInt16 && n <= math.MaxInt16
	case dicom.VRUL:
		return n >= 0 && n <= math.MaxUint32
	case dicom.VRSL:
		return n >= math.MinInt32 && n <= math.MaxInt32
	case dicom.VRUV:
		return n >= 0
	case dicom.VRSV:
		return true // any int64 fits SV
	default:
		return true
	}
}

// isFloatVR reports the float VRs encoded as JSON numbers (Annex F.2.3): FL and FD. The
// "Other" float VRs OF/OD are binary (Annex F.2.6) and handled separately.
func isFloatVR(vr dicom.VR) bool {
	switch vr {
	case dicom.VRFL, dicom.VRFD:
		return true
	default:
		return false
	}
}

// isOtherFloatVR reports OF/OD: float data carried as binary InlineBinary/BulkDataURI
// payloads in DICOM JSON (Annex F.2.6), distinct from the numeric FL/FD.
func isOtherFloatVR(vr dicom.VR) bool {
	switch vr {
	case dicom.VROF, dicom.VROD:
		return true
	default:
		return false
	}
}

// isByteVR reports the VRs carried as opaque raw bytes (InlineBinary/BulkDataURI):
// OB/OW/OL/OV/UN. OF/OD are binary too but decode to float values, not bytes.
func isByteVR(vr dicom.VR) bool {
	switch vr {
	case dicom.VROB, dicom.VROW, dicom.VROL, dicom.VROV, dicom.VRUN:
		return true
	default:
		return false
	}
}
