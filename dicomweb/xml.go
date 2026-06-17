package dicomweb

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/codeninja55/go-radx/dicom"
)

// nativeDicomModelNamespace is the XML namespace of the PS3.19 Native DICOM Model. PS3.18
// §A.1 / §F binds the application/dicom+xml media type to this model; the root element
// carries the namespace so a consumer can validate against the published schema.
const nativeDicomModelNamespace = "http://dicom.nema.org/PS3.19/models/NativeDICOM"

// nativeModel is the root <NativeDicomModel> element (PS3.19 §A.1). Its children are the
// dataset's attributes in ascending tag order, mirroring the DICOM-JSON object whose keys
// sort identically.
type nativeModel struct {
	XMLName    xml.Name       `xml:"NativeDicomModel"`
	Namespace  string         `xml:"xmlns,attr,omitempty"`
	Attributes []xmlAttribute `xml:"DicomAttribute"`
}

// xmlAttribute is one <DicomAttribute> element. Exactly one child form is populated per
// attribute: Values for text/numeric VRs, PersonNames for PN, Items for SQ, InlineBinary
// or BulkData for binary VRs, or none for an empty element (PS3.19 §A.1).
type xmlAttribute struct {
	Tag            string          `xml:"tag,attr"`
	VR             string          `xml:"vr,attr"`
	Keyword        string          `xml:"keyword,attr,omitempty"`
	PrivateCreator string          `xml:"privateCreator,attr,omitempty"`
	Values         []xmlValue      `xml:"Value,omitempty"`
	PersonNames    []xmlPersonName `xml:"PersonName,omitempty"`
	Items          []xmlItem       `xml:"Item,omitempty"`
	InlineBinary   string          `xml:"InlineBinary,omitempty"`
	BulkData       *xmlBulkData    `xml:"BulkData,omitempty"`
}

// xmlValue is a <Value number="N">text</Value> element. number is the 1-based position of
// the value within a multi-valued attribute (PS3.19 §A.1), distinguishing the positional
// XML form from DICOM-JSON's ordered array.
type xmlValue struct {
	Number int    `xml:"number,attr"`
	Text   string `xml:",chardata"`
}

// xmlBulkData is a <BulkData> reference to a binary value retrieved separately. PS3.19
// §A.1 carries the reference in either a uri attribute (a WADO-RS BulkDataURI) or a uuid
// attribute; this codec emits and accepts the uri form, the WADO-RS twin of the JSON
// BulkDataURI.
type xmlBulkData struct {
	URI  string `xml:"uri,attr,omitempty"`
	UUID string `xml:"uuid,attr,omitempty"`
}

// xmlItem is one <Item number="N"> of a sequence: a nested dataset whose attributes are
// direct DicomAttribute children (PS3.19 §A.1 nests the DicomDataSet content directly,
// with no wrapper element). number is the 1-based item index.
type xmlItem struct {
	Number     int            `xml:"number,attr"`
	Attributes []xmlAttribute `xml:"DicomAttribute"`
}

// xmlPersonName is one <PersonName number="N"> value: up to three component groups
// (PS3.19 §A.1). number is the 1-based position within a multi-valued PN attribute.
type xmlPersonName struct {
	Number      int           `xml:"number,attr"`
	Alphabetic  *xmlNameGroup `xml:"Alphabetic,omitempty"`
	Ideographic *xmlNameGroup `xml:"Ideographic,omitempty"`
	Phonetic    *xmlNameGroup `xml:"Phonetic,omitempty"`
}

// xmlNameGroup is one PersonName component group: the five named components (PS3.19 §A.1).
// Unlike DICOM-JSON's caret-delimited single string, the XML model breaks each group into
// FamilyName/GivenName/MiddleName/NamePrefix/NameSuffix child elements.
type xmlNameGroup struct {
	FamilyName string `xml:"FamilyName,omitempty"`
	GivenName  string `xml:"GivenName,omitempty"`
	MiddleName string `xml:"MiddleName,omitempty"`
	NamePrefix string `xml:"NamePrefix,omitempty"`
	NameSuffix string `xml:"NameSuffix,omitempty"`
}

// MarshalXML encodes ds as the PS3.19 Native DICOM Model (application/dicom+xml). It is the
// XML twin of MarshalJSON: it walks the same dataset model and emits the same logical
// content, differing only in serialization. The result is a <NativeDicomModel> document
// whose <DicomAttribute> children appear in ascending tag order; each attribute carries its
// tag, vr, and (when known) keyword, and exactly one value form. A nil dataset marshals to
// an empty model. The same options as the JSON codec govern BulkData emission on encode.
func MarshalXML(ds *dicom.DataSet, opts ...JSONOption) ([]byte, error) {
	cfg := defaultJSONConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	attrs, err := marshalXMLDataSet(ds, &cfg, "")
	if err != nil {
		return nil, err
	}
	model := nativeModel{Namespace: nativeDicomModelNamespace, Attributes: attrs}
	out, err := xml.Marshal(model)
	if err != nil {
		return nil, fmt.Errorf("dicomweb: encode DICOM XML: %w", err)
	}
	return out, nil
}

// marshalXMLDataSet builds the ordered slice of <DicomAttribute> elements for a dataset.
// path is the locator prefix used to mint unique BulkData URIs for nested binary
// attributes, identical to the DICOM-JSON locator so the two codecs reference the same
// bulk data. Attributes are emitted in ascending tag order so the output is deterministic.
func marshalXMLDataSet(ds *dicom.DataSet, cfg *jsonConfig, path string) ([]xmlAttribute, error) {
	if ds == nil {
		return nil, nil
	}
	type entry struct {
		key string
		el  dicom.Element
	}
	entries := make([]entry, 0)
	for e := range ds.All() {
		entries = append(entries, entry{key: tagKey(e.Tag), el: e})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].key < entries[j].key })

	out := make([]xmlAttribute, 0, len(entries))
	for _, en := range entries {
		attr, err := marshalXMLElement(en.el, cfg, path)
		if err != nil {
			return nil, err
		}
		out = append(out, attr)
	}
	return out, nil
}

// marshalXMLElement encodes one element to its <DicomAttribute> form. It mirrors
// marshalElement's type switch so the XML and JSON codecs agree on which value form an
// attribute takes; only the serialization differs. An ambiguous (unresolved) VR is rejected
// fail-closed, as in the JSON codec, because it has no on-wire representation.
func marshalXMLElement(e dicom.Element, cfg *jsonConfig, path string) (xmlAttribute, error) {
	if !isConcreteVR(e.VR) {
		return xmlAttribute{}, &EncodeError{Tag: e.Tag, VR: e.VR, Msg: "ambiguous VR has no DICOM XML representation; resolve it to a concrete VR"}
	}
	attr := xmlAttribute{Tag: tagKey(e.Tag), VR: e.VR.String(), Keyword: keywordOf(e.Tag)}
	elemPath := joinPath(path, tagKey(e.Tag))

	switch v := e.Value.(type) {
	case *dicom.Strings:
		if e.VR == dicom.VRPN {
			pns, err := marshalXMLPersonNames(v.Strings())
			if err != nil {
				return xmlAttribute{}, err
			}
			attr.PersonNames = pns
			return attr, nil
		}
		vals, err := marshalXMLStringValues(e.VR, v.Strings())
		if err != nil {
			return xmlAttribute{}, err
		}
		attr.Values = vals
	case *dicom.Ints:
		vals, err := marshalXMLIntValues(e.VR, v.Ints())
		if err != nil {
			return xmlAttribute{}, err
		}
		attr.Values = vals
	case *dicom.Floats:
		if isOtherFloatVR(e.VR) {
			return marshalXMLBinary(attr, e, otherFloatBytes(e.VR, v.Floats()), cfg, elemPath)
		}
		vals, err := marshalXMLFloatValues(e.VR, v.Floats())
		if err != nil {
			return xmlAttribute{}, err
		}
		attr.Values = vals
	case *dicom.Decimals:
		vals, err := marshalXMLDecimalValues(e.VR, v.Decimals())
		if err != nil {
			return xmlAttribute{}, err
		}
		attr.Values = vals
	case *dicom.Tags:
		attr.Values = marshalXMLTagValues(v.Tags())
	case *dicom.Bytes:
		return marshalXMLBinary(attr, e, v.Bytes(), cfg, elemPath)
	case *bulkRef:
		attr.BulkData = &xmlBulkData{URI: string(v.URI())}
		return attr, nil
	default:
		if seq, ok := sequenceFromValue(e); ok {
			items, err := marshalXMLSequence(seq, cfg, elemPath)
			if err != nil {
				return xmlAttribute{}, err
			}
			attr.Items = items
			return attr, nil
		}
		return xmlAttribute{}, &EncodeError{Tag: e.Tag, VR: e.VR, Msg: "unsupported value type"}
	}
	return attr, nil
}

// marshalXMLStringValues encodes a non-PN text VR's values as positional <Value> elements.
// A logically empty element (no values, or one empty value) emits no <Value>, matching the
// empty-element form the decoder produces; otherwise every value, empty or not, keeps its
// 1-based position so multiplicity round-trips (PS3.19 §A.1).
func marshalXMLStringValues(vr dicom.VR, vals []string) ([]xmlValue, error) {
	if len(vals) <= 1 && allEmpty(vals) {
		return nil, nil
	}
	out := make([]xmlValue, 0, len(vals))
	for i, s := range vals {
		out = append(out, xmlValue{Number: i + 1, Text: s})
	}
	return out, nil
}

// marshalXMLIntValues encodes the integer VRs as decimal text <Value> elements. A value out
// of range for its VR is a hard encode error, mirroring the JSON codec and the decode-side
// check, so the encoder never emits non-conformant XML.
func marshalXMLIntValues(vr dicom.VR, vals []int64) ([]xmlValue, error) {
	out := make([]xmlValue, 0, len(vals))
	for i, n := range vals {
		if !intFitsVR(vr, n) {
			return nil, &EncodeError{VR: vr, Msg: "integer value out of range for its VR"}
		}
		out = append(out, xmlValue{Number: i + 1, Text: strconv.FormatInt(n, 10)})
	}
	return out, nil
}

// marshalXMLFloatValues encodes FL/FD as text <Value> elements. A non-finite value has no
// textual conformant representation and is a hard encode error, as in the JSON codec.
func marshalXMLFloatValues(vr dicom.VR, vals []float64) ([]xmlValue, error) {
	out := make([]xmlValue, 0, len(vals))
	for i, f := range vals {
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return nil, &EncodeError{VR: vr, Msg: "non-finite float has no XML representation"}
		}
		out = append(out, xmlValue{Number: i + 1, Text: strconv.FormatFloat(f, 'g', -1, 64)})
	}
	return out, nil
}

// marshalXMLDecimalValues encodes DS/IS preserving the lexical form. An empty (default)
// Decimal or a fractional IS is a hard encode error, mirroring the JSON codec.
func marshalXMLDecimalValues(vr dicom.VR, vals []dicom.Decimal) ([]xmlValue, error) {
	out := make([]xmlValue, 0, len(vals))
	for i, d := range vals {
		if d.String() == "" {
			return nil, &EncodeError{VR: vr, Msg: "decimal value is empty"}
		}
		if vr == dicom.VRIS {
			if _, ok := d.Int64(); !ok {
				return nil, &EncodeError{VR: vr, Msg: "IS value is not an integer"}
			}
		}
		out = append(out, xmlValue{Number: i + 1, Text: d.String()})
	}
	return out, nil
}

// marshalXMLTagValues encodes AT values as eight-hex-digit text <Value> elements.
func marshalXMLTagValues(vals []dicom.Tag) []xmlValue {
	out := make([]xmlValue, 0, len(vals))
	for i, t := range vals {
		out = append(out, xmlValue{Number: i + 1, Text: tagKey(t)})
	}
	return out
}

// marshalXMLPersonNames encodes a PN attribute's values as positional <PersonName>
// elements, each broken into its component groups. A logically empty PN emits no
// <PersonName>, matching the empty-element decode. An invalid PN value is a hard encode
// error, as in the JSON codec.
func marshalXMLPersonNames(vals []string) ([]xmlPersonName, error) {
	if len(vals) <= 1 && allEmpty(vals) {
		return nil, nil
	}
	out := make([]xmlPersonName, 0, len(vals))
	for i, s := range vals {
		pn := xmlPersonName{Number: i + 1}
		if s != "" {
			parsed, err := dicom.ParsePersonName(s)
			if err != nil {
				return nil, &EncodeError{VR: dicom.VRPN, Msg: "invalid PN value"}
			}
			pn.Alphabetic = nameGroup(parsed.Alphabetic)
			pn.Ideographic = nameGroup(parsed.Ideographic)
			pn.Phonetic = nameGroup(parsed.Phonetic)
		}
		out = append(out, pn)
	}
	return out, nil
}

// nameGroup renders one component group to its <FamilyName>...<NameSuffix> form, returning
// nil for an empty group so the group element is omitted entirely.
func nameGroup(c dicom.NameComponents) *xmlNameGroup {
	if c == (dicom.NameComponents{}) {
		return nil
	}
	return &xmlNameGroup{
		FamilyName: c.FamilyName,
		GivenName:  c.GivenName,
		MiddleName: c.MiddleName,
		NamePrefix: c.Prefix,
		NameSuffix: c.Suffix,
	}
}

// marshalXMLBinary encodes OB/OW/OL/OV/UN (and OF/OD as bytes) as InlineBinary or a
// BulkData reference, applying the same threshold policy as the JSON codec. A fixed-width
// VR whose byte length is not a multiple of its word size is a hard encode error.
func marshalXMLBinary(attr xmlAttribute, e dicom.Element, b []byte, cfg *jsonConfig, elemPath string) (xmlAttribute, error) {
	if w := fixedBinaryWidth(e.VR); w > 1 && len(b)%w != 0 {
		return xmlAttribute{}, &EncodeError{Tag: e.Tag, VR: e.VR, Msg: "binary payload length is not a multiple of the VR word size"}
	}
	if cfg.bulkThreshold > 0 && len(b) >= cfg.bulkThreshold && cfg.bulkBaseURL != "" {
		attr.BulkData = &xmlBulkData{URI: cfg.bulkBaseURL + elemPath}
		return attr, nil
	}
	attr.InlineBinary = base64.StdEncoding.EncodeToString(b)
	return attr, nil
}

// marshalXMLSequence encodes an SQ value as positional <Item> elements, each holding the
// nested dataset's <DicomAttribute> children. The item path extends the SQ locator with the
// 1-based index, matching the DICOM-JSON locator so nested BulkData references agree.
func marshalXMLSequence(seq *dicom.Sequence, cfg *jsonConfig, path string) ([]xmlItem, error) {
	out := make([]xmlItem, 0, seq.Len())
	i := 0
	for item := range seq.Items() {
		attrs, err := marshalXMLDataSet(item.DataSet, cfg, joinPath(path, strconv.Itoa(i)))
		if err != nil {
			return nil, err
		}
		out = append(out, xmlItem{Number: i + 1, Attributes: attrs})
		i++
	}
	return out, nil
}

// UnmarshalXML decodes a PS3.19 Native DICOM Model document into a DataSet. It is the XML
// twin of UnmarshalJSON: it accepts the same logical content and reconstructs the same
// dataset, so a document encoded by MarshalXML round-trips to a value-equal dataset, and the
// XML and JSON forms of one dataset decode to equal datasets. SQ nesting is bounded by the
// configured depth (default 64); exceeding it returns *LimitExceededError, never a panic. A
// BulkData reference is left unresolved unless a resolver was supplied via
// WithBulkDataResolver, mirroring the JSON codec's decode contract.
func UnmarshalXML(data []byte, opts ...JSONOption) (*dicom.DataSet, error) {
	cfg := defaultJSONConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	var model nativeModel
	if err := xml.Unmarshal(data, &model); err != nil {
		return nil, &DecodeError{Msg: "document is not a valid Native DICOM Model: " + xmlErrorDetail(err)}
	}
	if model.XMLName.Local != "NativeDicomModel" {
		return nil, &DecodeError{Msg: "root element is not NativeDicomModel"}
	}
	return unmarshalXMLAttributes(model.Attributes, &cfg, 0)
}

// xmlErrorDetail reduces an encoding/xml error to a structural description, never echoing
// document bytes: a malformed document is attacker-controlled and could carry PHI (PRD §9.1).
func xmlErrorDetail(err error) string {
	if se, ok := err.(*xml.SyntaxError); ok {
		return fmt.Sprintf("syntax error at line %d", se.Line)
	}
	return "malformed XML"
}

// unmarshalXMLAttributes reconstructs a DataSet from a slice of decoded <DicomAttribute>
// elements at one nesting level. depth is the current SQ nesting depth, validated against
// the cap before recursing so an over-deep document is rejected before its forbidden level
// is materialised (PRD §9.3). Attributes are applied in ascending tag order so a re-encode
// is byte-stable regardless of document order.
func unmarshalXMLAttributes(attrs []xmlAttribute, cfg *jsonConfig, depth int) (*dicom.DataSet, error) {
	if depth > cfg.maxDepth {
		return nil, &LimitExceededError{
			Limit:  uint64(cfg.maxDepth), // #nosec G115 -- small non-negative configured depth cap
			Actual: uint64(depth),        // #nosec G115 -- small non-negative recursion counter
			Kind:   "xml-sequence-depth",
		}
	}
	sorted := make([]xmlAttribute, len(attrs))
	copy(sorted, attrs)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Tag < sorted[j].Tag })

	ds := dicom.NewDataSet()
	for _, attr := range sorted {
		tag, err := parseTagKey(attr.Tag)
		if err != nil {
			return nil, err
		}
		vr, err := parseVR(attr.VR, tag)
		if err != nil {
			return nil, err
		}
		value, err := decodeXMLValue(tag, vr, attr, cfg, depth)
		if err != nil {
			return nil, err
		}
		ds.Set(dicom.Element{Tag: tag, VR: vr, Value: value})
	}
	return ds, nil
}

// decodeXMLValue reconstructs the dicom.Value for one attribute from its decoded
// <DicomAttribute>. It enforces the same one-of-form rule as the JSON codec: an attribute
// carrying more than one of Value/PersonName/Item/InlineBinary/BulkData is malformed.
func decodeXMLValue(tag dicom.Tag, vr dicom.VR, attr xmlAttribute, cfg *jsonConfig, depth int) (dicom.Value, error) {
	if xmlPayloadFormCount(attr) > 1 {
		return nil, &DecodeError{Tag: tag, VR: vr, Msg: "attribute declares more than one value form"}
	}

	binaryVR := isByteVR(vr) || isOtherFloatVR(vr)

	if attr.InlineBinary != "" {
		if !binaryVR {
			return nil, &DecodeError{Tag: tag, VR: vr, Msg: "InlineBinary on a non-binary VR"}
		}
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(attr.InlineBinary))
		if err != nil {
			return nil, &DecodeError{Tag: tag, VR: vr, Msg: "invalid InlineBinary base64"}
		}
		return decodeBinaryPayload(tag, vr, raw)
	}
	if attr.BulkData != nil {
		if !binaryVR {
			return nil, &DecodeError{Tag: tag, VR: vr, Msg: "BulkData on a non-binary VR"}
		}
		ref := attr.BulkData.URI
		if ref == "" {
			ref = attr.BulkData.UUID
		}
		if ref == "" {
			return nil, &DecodeError{Tag: tag, VR: vr, Msg: "BulkData has neither uri nor uuid"}
		}
		if cfg.bulkResolver == nil {
			return newBulkRef(vr, BulkDataURI(ref)), nil
		}
		ctx := cfg.resolverCtx
		if ctx == nil {
			ctx = context.Background()
		}
		raw, err := cfg.bulkResolver(ctx, BulkDataURI(ref))
		if err != nil {
			return nil, fmt.Errorf("dicomweb: resolve BulkData for %s %s: %w", keywordOf(tag), tag, err)
		}
		return decodeBinaryPayload(tag, vr, raw)
	}

	if vr == dicom.VRSQ {
		return decodeXMLSequence(attr.Items, cfg, depth)
	}
	if vr == dicom.VRPN {
		return decodeXMLPersonNames(attr.PersonNames)
	}
	if binaryVR {
		if len(attr.Values) > 0 {
			return nil, &DecodeError{Tag: tag, VR: vr, Msg: "binary VR carries Value elements instead of InlineBinary or BulkData"}
		}
		return decodeBinaryPayload(tag, vr, nil)
	}
	return decodeXMLNonBinary(tag, vr, attr.Values)
}

// xmlPayloadFormCount counts how many of the mutually exclusive value forms are present.
func xmlPayloadFormCount(attr xmlAttribute) int {
	n := 0
	if len(attr.Values) > 0 {
		n++
	}
	if len(attr.PersonNames) > 0 {
		n++
	}
	if len(attr.Items) > 0 {
		n++
	}
	if attr.InlineBinary != "" {
		n++
	}
	if attr.BulkData != nil {
		n++
	}
	return n
}

// orderedValues returns the <Value> texts ordered by their 1-based number attribute,
// filling any gap with the empty string so a sparse multi-valued attribute keeps its
// multiplicity. A number below 1 or beyond a sane bound is rejected fail-closed.
func orderedValues(vals []xmlValue) ([]string, error) {
	if len(vals) == 0 {
		return nil, nil
	}
	maxN := 0
	for _, v := range vals {
		if v.Number < 1 {
			return nil, fmt.Errorf("value number %d is below 1", v.Number)
		}
		if v.Number > len(vals) {
			// number must address a position within the attribute's own values; a number
			// beyond the count would imply an unbounded sparse array, rejected fail-closed.
			return nil, fmt.Errorf("value number %d exceeds the value count %d", v.Number, len(vals))
		}
		if v.Number > maxN {
			maxN = v.Number
		}
	}
	out := make([]string, maxN)
	for _, v := range vals {
		out[v.Number-1] = v.Text
	}
	return out, nil
}

// decodeXMLNonBinary reconstructs a string, numeric, decimal, or AT value from positional
// <Value> elements, reusing the JSON codec's per-value parsers so the two codecs agree on
// range checks and lexical handling.
func decodeXMLNonBinary(tag dicom.Tag, vr dicom.VR, vals []xmlValue) (dicom.Value, error) {
	texts, err := orderedValues(vals)
	if err != nil {
		return nil, &DecodeError{Tag: tag, VR: vr, Msg: err.Error()}
	}
	switch {
	case isStringVR(vr):
		for _, s := range texts {
			if usesBackslashDelimiter(vr) && strings.ContainsRune(s, '\\') {
				return nil, &DecodeError{Tag: tag, VR: vr, Msg: "value contains the value-multiplicity separator '\\'"}
			}
		}
		return dicom.NewStrings(vr, texts...), nil
	case isIntVR(vr):
		ns := make([]int64, 0, len(texts))
		for _, s := range texts {
			n, err := parseXMLInt(vr, s)
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
		fs := make([]float64, 0, len(texts))
		for _, s := range texts {
			f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
			if err != nil {
				return nil, &DecodeError{Tag: tag, VR: vr, Msg: "invalid float value"}
			}
			fs = append(fs, f)
		}
		return dicom.NewFloats(vr, fs...), nil
	case vr == dicom.VRDS || vr == dicom.VRIS:
		decs := make([]dicom.Decimal, 0, len(texts))
		for _, s := range texts {
			d, err := dicom.ParseDecimal(strings.TrimSpace(s))
			if err != nil {
				return nil, &DecodeError{Tag: tag, VR: vr, Msg: "invalid decimal value"}
			}
			if vr == dicom.VRIS {
				if _, ok := d.Int64(); !ok {
					return nil, &DecodeError{Tag: tag, VR: vr, Msg: "IS value is not an integer"}
				}
			}
			decs = append(decs, d)
		}
		return dicom.NewDecimals(vr, decs...), nil
	case vr == dicom.VRAT:
		tags := make([]dicom.Tag, 0, len(texts))
		for _, s := range texts {
			at, err := parseTagKey(strings.TrimSpace(s))
			if err != nil {
				return nil, &DecodeError{Tag: tag, VR: vr, Msg: "invalid AT value"}
			}
			tags = append(tags, at)
		}
		return dicom.NewTags(tags...), nil
	default:
		return dicom.NewStrings(vr, texts...), nil
	}
}

// parseXMLInt parses one integer <Value> text under its VR, applying the same unsigned and
// width handling as the JSON codec's decodeIntValue.
func parseXMLInt(vr dicom.VR, s string) (int64, error) {
	s = strings.TrimSpace(s)
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

// decodeXMLPersonNames reconstructs a PN value from positional <PersonName> elements,
// assembling each component group back into the canonical "=" / "^" string the dicom model
// stores. The component-group delimiter checks of the JSON codec apply, so an embedded
// separator is rejected fail-closed.
func decodeXMLPersonNames(pns []xmlPersonName) (dicom.Value, error) {
	if len(pns) == 0 {
		return dicom.NewStrings(dicom.VRPN), nil
	}
	ordered := make([]xmlPersonName, len(pns))
	copy(ordered, pns)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Number < ordered[j].Number })

	strs := make([]string, 0, len(ordered))
	for _, pn := range ordered {
		s, err := composeXMLPersonName(pn)
		if err != nil {
			return nil, &DecodeError{VR: dicom.VRPN, Msg: err.Error()}
		}
		strs = append(strs, s)
	}
	return dicom.NewStrings(dicom.VRPN, strs...), nil
}

// composeXMLPersonName assembles one PN value from its component groups, validating each
// component for an embedded DICOM delimiter and the assembled value through the dicom parser.
func composeXMLPersonName(pn xmlPersonName) (string, error) {
	groups := []*xmlNameGroup{pn.Alphabetic, pn.Ideographic, pn.Phonetic}
	rendered := make([]string, len(groups))
	for i, g := range groups {
		if g == nil {
			continue
		}
		comps := []string{g.FamilyName, g.GivenName, g.MiddleName, g.NamePrefix, g.NameSuffix}
		for _, c := range comps {
			if strings.ContainsRune(c, '=') {
				return "", fmt.Errorf("PN component contains the group separator '='")
			}
			if strings.ContainsRune(c, '\\') {
				return "", fmt.Errorf("PN component contains the value separator '\\'")
			}
			if strings.ContainsRune(c, '^') {
				return "", fmt.Errorf("PN component contains the component separator '^'")
			}
		}
		end := len(comps)
		for end > 0 && comps[end-1] == "" {
			end--
		}
		rendered[i] = strings.Join(comps[:end], "^")
	}
	end := len(rendered)
	for end > 0 && rendered[end-1] == "" {
		end--
	}
	joined := strings.Join(rendered[:end], "=")
	if _, err := dicom.ParsePersonName(joined); err != nil {
		return "", fmt.Errorf("invalid PN value")
	}
	return joined, nil
}

// decodeXMLSequence reconstructs an SQ value from positional <Item> elements, descending one
// nesting level. The depth cap is checked before a nested item is materialised so an
// over-deep document is rejected early (PRD §9.3). Items are reordered by their number
// attribute so item order round-trips.
func decodeXMLSequence(items []xmlItem, cfg *jsonConfig, depth int) (dicom.Value, error) {
	if len(items) > 0 && depth+1 > cfg.maxDepth {
		return nil, &LimitExceededError{
			Limit:  uint64(cfg.maxDepth), // #nosec G115 -- small non-negative configured depth cap
			Actual: uint64(depth + 1),    // #nosec G115 -- small non-negative recursion counter
			Kind:   "xml-sequence-depth",
		}
	}
	ordered := make([]xmlItem, len(items))
	copy(ordered, items)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Number < ordered[j].Number })

	seq := dicom.NewSequence()
	for _, item := range ordered {
		nested, err := unmarshalXMLAttributes(item.Attributes, cfg, depth+1)
		if err != nil {
			return nil, err
		}
		seq.Append(nested)
	}
	return dicom.NewSequenceValue(seq), nil
}
