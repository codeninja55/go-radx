package fhir

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// ErrNilResource is returned by MarshalSummary when the resource is nil (a nil
// interface or a typed-nil pointer) rather than dereferencing it and panicking
// (Codex FHIR-012). It is a sentinel a caller matches with errors.Is.
var ErrNilResource = newSentinel("resource is nil")

// SummaryMode selects the _summary serialization view. The five modes follow the
// FHIR SummaryEnum value set so MarshalSummary mirrors the server-side _summary
// search parameter: a constrained consumer (a worklist over a slow link) requests a
// reduced view by mode rather than transferring the full record.
type SummaryMode string

const (
	// SummaryFull emits the full resource with no filtering. It is the identity view:
	// the bytes match the resource's own MarshalJSON.
	SummaryFull SummaryMode = "false"

	// SummaryTrue emits only the elements flagged isSummary in the StructureDefinition,
	// plus the mandatory (min >= 1) and modifier elements FHIR always keeps in a summary,
	// plus the always-retained infrastructure elements (id, meta). When any element is
	// dropped it sets the SUBSETTED tag on meta so a consumer knows the view is partial.
	SummaryTrue SummaryMode = "true"

	// SummaryText emits the narrative (text), id, meta, and the mandatory elements only.
	// It is the smallest human-readable view: the rendered narrative plus the minimum
	// structure required for the resource to be valid. It sets the SUBSETTED tag.
	SummaryText SummaryMode = "text"

	// SummaryData emits everything except the narrative (text). It is the inverse of
	// SummaryText: the structured data without the rendered prose. It sets the SUBSETTED
	// tag because the narrative is dropped.
	SummaryData SummaryMode = "data"

	// SummaryCount is intended for a Bundle: it emits the count (a Bundle's total) and the
	// mandatory elements, dropping the entries, so a caller learns how many results match
	// without transferring them while the reduced view stays structurally valid (a Bundle
	// keeps its mandatory type). On a non-Bundle resource it keeps the mandatory elements
	// plus the always-retained infrastructure ones.
	SummaryCount SummaryMode = "count"
)

// summaryAlwaysKeep are the infrastructure element wire keys every summary view retains
// regardless of their isSummary flag: the discriminator, the local id, and the metadata
// the SUBSETTED tag rides on. They are kept for SummaryTrue, SummaryText, and
// SummaryCount, which otherwise filter aggressively, so a summarised resource is still a
// valid, identifiable resource.
var summaryAlwaysKeep = map[string]struct{}{
	"resourceType": {},
	"id":           {},
	"meta":         {},
}

// MarshalSummary serializes a resource under the given summary mode. SummaryFull is the
// resource's own MarshalJSON; every other mode marshals the resource and then drops the
// top-level elements the mode excludes, preserving the canonical element order the
// resource's MarshalJSON produced (the filter walks the encoded object key-by-key and
// re-emits the kept keys in place, never re-sorting through a map). When a mode drops
// any element it tags meta with the FHIR SUBSETTED marker so a consumer can tell the
// payload is a partial view.
//
// Filtering is data-driven by a per-resource summary descriptor the generator emits and
// each release package registers at init time, so MarshalSummary takes no metadata
// reflection on the serialization path: the descriptor already carries which top-level
// wire keys are isSummary, which are mandatory, and which are modifier. A resource whose
// type has no registered summary descriptor cannot be filtered safely, so MarshalSummary
// returns the full encoding for that type rather than guessing which elements to drop.
//
// It returns ErrNilResource for a nil resource (a nil interface or a typed-nil pointer)
// rather than panicking (Codex FHIR-012). It never leaks PHI: the only metadata it adds
// is the SUBSETTED coding, and an error names a mode or a type, never a patient value.
func MarshalSummary(r Resource, mode SummaryMode) ([]byte, error) {
	if r == nil || isNilResource(r) {
		return nil, ErrNilResource
	}

	encoded, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("fhir: marshal summary: %w", err)
	}
	if mode == SummaryFull {
		return encoded, nil
	}

	descriptor, ok := lookupSummaryDescriptor(r.ResourceType())
	if !ok {
		// No descriptor means the element-level summary flags are unknown for this type,
		// so filtering would have to guess which elements to drop. Returning the full
		// encoding keeps the output valid (it is simply not reduced) rather than emitting
		// a payload with arbitrary elements removed.
		return encoded, nil
	}

	keep := descriptor.keepSet(mode)
	filtered, dropped, err := filterTopLevel(encoded, keep)
	if err != nil {
		return nil, err
	}
	if !dropped {
		return filtered, nil
	}
	return tagSubsetted(filtered)
}

// SummaryDescriptor is the generated, per-resource summary metadata MarshalSummary
// consumes. Each release package emits one descriptor per concrete resource and
// registers it at init time, keyed by the resource's resourceType. The descriptor lists
// each top-level element's wire key together with the flags that decide whether a given
// summary mode keeps it, so the filter is a set-membership test over already-known keys
// rather than a reflective field walk or a per-call StructureDefinition lookup.
//
// A choice ([x]) group contributes one entry per suffixed branch wire key
// ("deceasedBoolean", "deceasedDateTime"), all sharing the group's flags, so whichever
// branch a resource set survives or is dropped together with its siblings.
type SummaryDescriptor struct {
	// Elements are the resource's top-level elements in no particular order; the filter
	// only tests membership, and the canonical wire order is preserved by the encoded
	// bytes the filter walks, not by this slice.
	Elements []SummaryElement
}

// SummaryElement is one top-level element's summary metadata: its wire key and the flags
// the modes test. Primitive "_field" siblings are not listed; the filter keeps a "_field"
// key exactly when it keeps the matching value key, so a summarised primitive keeps its
// extensions and an excluded one drops both halves together.
type SummaryElement struct {
	// JSONName is the element's wire key ("gender", "deceasedBoolean"). For a choice
	// branch it is the suffixed key.
	JSONName string

	// IsSummary records the StructureDefinition isSummary flag; SummaryTrue keeps the
	// element when set.
	IsSummary bool

	// IsMandatory records whether the element's minimum cardinality is at least one;
	// SummaryTrue and SummaryText always keep a mandatory element so the reduced view
	// stays structurally valid.
	IsMandatory bool

	// IsModifier records the StructureDefinition isModifier flag; SummaryTrue always
	// keeps a modifier element because dropping one could change how the resource is
	// interpreted, which a summary must never do.
	IsModifier bool

	// IsText reports whether the element is the DomainResource narrative ("text"), the
	// one element SummaryText keeps and SummaryData drops.
	IsText bool

	// IsCount reports whether the element is the one a count view keeps (a Bundle's
	// "total"); SummaryCount keeps it and drops every other non-infrastructure element.
	IsCount bool
}

// keepSet returns the set of top-level wire keys the mode retains, always including the
// infrastructure keys (resourceType, id, meta) so a summarised resource stays a valid,
// identifiable resource. The per-mode rules follow the FHIR _summary parameter: true
// keeps summary, mandatory, and modifier elements; text keeps the narrative and mandatory
// elements; data keeps everything but the narrative; count keeps only the count element.
func (d SummaryDescriptor) keepSet(mode SummaryMode) map[string]struct{} {
	keep := make(map[string]struct{}, len(d.Elements)+len(summaryAlwaysKeep))
	for k := range summaryAlwaysKeep {
		keep[k] = struct{}{}
	}
	for _, e := range d.Elements {
		if summaryKeeps(mode, e) {
			keep[e.JSONName] = struct{}{}
		}
	}
	return keep
}

// summaryKeeps reports whether a single element survives a mode's filter. SummaryFull is
// handled before any descriptor lookup, so it is not a case here.
func summaryKeeps(mode SummaryMode, e SummaryElement) bool {
	switch mode {
	case SummaryTrue:
		return e.IsSummary || e.IsMandatory || e.IsModifier
	case SummaryText:
		return e.IsText || e.IsMandatory
	case SummaryData:
		return !e.IsText
	case SummaryCount:
		return e.IsCount || e.IsMandatory
	default:
		return false
	}
}

// filterTopLevel re-emits the encoded JSON object keeping only the top-level keys in
// keep, preserving their original order, and reports whether any non-infrastructure key
// was dropped. A "_field" primitive-extension sibling is kept exactly when its value key
// ("_gender" follows "gender") is kept, so a primitive's extensions ride with the value
// or are dropped with it. Walking the token stream rather than decoding into a map keeps
// the canonical element ordering the resource's MarshalJSON produced; a map round-trip
// would re-sort every key.
func filterTopLevel(encoded []byte, keep map[string]struct{}) ([]byte, bool, error) {
	dec := json.NewDecoder(bytes.NewReader(encoded))
	open, err := dec.Token()
	if err != nil {
		return nil, false, fmt.Errorf("fhir: summary filter: read object start: %w", err)
	}
	if delim, ok := open.(json.Delim); !ok || delim != '{' {
		return nil, false, fmt.Errorf("fhir: summary filter: expected a JSON object, got %v", open)
	}

	var buf bytes.Buffer
	buf.WriteByte('{')
	first := true
	dropped := false
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, false, fmt.Errorf("fhir: summary filter: read key: %w", err)
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, false, fmt.Errorf("fhir: summary filter: object key was not a string")
		}

		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, false, fmt.Errorf("fhir: summary filter: read value for %q: %w", key, err)
		}

		if !keepKey(key, keep) {
			dropped = true
			continue
		}

		if !first {
			buf.WriteByte(',')
		}
		first = false
		encodedKey, err := json.Marshal(key)
		if err != nil {
			return nil, false, fmt.Errorf("fhir: summary filter: encode key %q: %w", key, err)
		}
		buf.Write(encodedKey)
		buf.WriteByte(':')
		buf.Write(raw)
	}
	buf.WriteByte('}')
	return buf.Bytes(), dropped, nil
}

// keepKey reports whether a top-level wire key survives the filter. A primitive "_field"
// sibling ("_gender") is kept exactly when its value key ("gender") is kept, so the value
// and its extensions are filtered as a unit.
func keepKey(key string, keep map[string]struct{}) bool {
	lookup := key
	if strings.HasPrefix(key, "_") {
		lookup = key[1:]
	}
	_, ok := keep[lookup]
	return ok
}

// tagSubsetted records the FHIR SUBSETTED marker on the resource's meta.tag so a consumer
// can tell the payload is a partial view (FHIR records this tag whenever a server returns a
// filtered resource). It splices the tag in without re-sorting the other keys: it walks the
// filtered object key-by-key, re-emitting each key in place, and either extends the existing
// meta with the SUBSETTED coding or, when meta is absent, inserts a meta key carrying just
// the tag right after the id (or after resourceType when there is no id), the canonical
// position for meta in the base resource element order. A map round-trip would re-sort every
// key and break the canonical element ordering MarshalSummary preserves.
func tagSubsetted(filtered []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(filtered))
	open, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("fhir: summary tag: read object start: %w", err)
	}
	if delim, ok := open.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("fhir: summary tag: expected a JSON object, got %v", open)
	}

	var buf bytes.Buffer
	buf.WriteByte('{')
	first := true
	hasMeta := false
	var lastKey string
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("fhir: summary tag: read key: %w", err)
		}
		key := keyTok.(string)

		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, fmt.Errorf("fhir: summary tag: read value for %q: %w", key, err)
		}

		value := raw
		if key == "meta" {
			hasMeta = true
			if value, err = appendMetaTag(raw); err != nil {
				return nil, err
			}
		}
		writeMember(&buf, &first, key, value)

		// meta sits after id (or directly after resourceType when id is absent) in the
		// base resource element order; remember the prior key so a missing meta can be
		// inserted in that canonical slot.
		if !hasMeta && (key == "id" || (key == "resourceType" && lastKey == "")) {
			lastKey = key
		}
	}

	if !hasMeta {
		// No meta was emitted, so insert one carrying only the SUBSETTED tag. Re-walk and
		// splice it after the canonical anchor key recorded above; if there was no anchor
		// (an object with neither resourceType nor id, which a generated resource never
		// produces) it is appended last.
		return spliceMeta(buf.Bytes(), lastKey)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// appendMetaTag returns the meta object with the SUBSETTED coding added to its tag array,
// preserving meta's own canonical key order. It walks meta key-by-key (as the top-level
// rewriters do, so a decode-into-map round trip never re-sorts meta's keys): an existing
// tag array gains the SUBSETTED coding in place. When meta has no tag, the new tag is
// inserted before meta's first primitive "_field" sibling key (a "_"-prefixed key, which
// Meta.MarshalJSON always trails after the value fields), so tag lands among the value
// fields in the same slot a fresh re-marshal of the decoded Meta would put it — keeping the
// summary byte-stable on round-trip. With no sibling key it is appended last.
func appendMetaTag(rawMeta json.RawMessage) (json.RawMessage, error) {
	dec := json.NewDecoder(bytes.NewReader(rawMeta))
	open, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("fhir: summary tag: read meta start: %w", err)
	}
	if delim, ok := open.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("fhir: summary tag: meta is not a JSON object, got %v", open)
	}

	var buf bytes.Buffer
	buf.WriteByte('{')
	first := true
	hasTag := false
	inserted := false
	newTag := json.RawMessage(`[` + subsettedCoding + `]`)
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("fhir: summary tag: read meta key: %w", err)
		}
		key := keyTok.(string)
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, fmt.Errorf("fhir: summary tag: read meta value for %q: %w", key, err)
		}
		if key == "tag" {
			hasTag = true
			if raw, err = appendCodingToArray(raw); err != nil {
				return nil, err
			}
		}
		// A missing tag is a value field, so it sorts ahead of meta's trailing "_field"
		// siblings; insert it the moment the first sibling key is reached.
		if !hasTag && !inserted && strings.HasPrefix(key, "_") {
			writeMember(&buf, &first, "tag", newTag)
			inserted = true
		}
		writeMember(&buf, &first, key, raw)
	}
	if !hasTag && !inserted {
		writeMember(&buf, &first, "tag", newTag)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// appendCodingToArray appends the SUBSETTED coding to an existing meta.tag array,
// preserving the existing codings and their order.
func appendCodingToArray(rawTag json.RawMessage) (json.RawMessage, error) {
	var existing []json.RawMessage
	if err := json.Unmarshal(rawTag, &existing); err != nil {
		return nil, fmt.Errorf("fhir: summary tag: decode existing tags: %w", err)
	}
	tags := append(existing, json.RawMessage(subsettedCoding))
	encoded, err := json.Marshal(tags)
	if err != nil {
		return nil, fmt.Errorf("fhir: summary tag: encode tags: %w", err)
	}
	return encoded, nil
}

// spliceMeta inserts a meta object carrying only the SUBSETTED tag immediately after the
// anchor key (resourceType or id), re-walking the object so the surrounding keys keep their
// order. An empty anchor appends meta as the last key.
func spliceMeta(object []byte, anchor string) ([]byte, error) {
	metaValue := json.RawMessage(`{"tag":[` + subsettedCoding + `]}`)
	dec := json.NewDecoder(bytes.NewReader(object))
	if _, err := dec.Token(); err != nil {
		return nil, fmt.Errorf("fhir: summary tag: splice read object start: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteByte('{')
	first := true
	inserted := false
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("fhir: summary tag: splice read key: %w", err)
		}
		key := keyTok.(string)
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, fmt.Errorf("fhir: summary tag: splice read value: %w", err)
		}
		writeMember(&buf, &first, key, raw)
		if !inserted && key == anchor {
			writeMember(&buf, &first, "meta", metaValue)
			inserted = true
		}
	}
	if !inserted {
		writeMember(&buf, &first, "meta", metaValue)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// writeMember appends one "key":value member to buf, writing the leading comma for every
// member after the first. It centralises the comma bookkeeping the order-preserving
// rewriters share.
func writeMember(buf *bytes.Buffer, first *bool, key string, value json.RawMessage) {
	if !*first {
		buf.WriteByte(',')
	}
	*first = false
	encodedKey, _ := json.Marshal(key)
	buf.Write(encodedKey)
	buf.WriteByte(':')
	buf.Write(value)
}

// subsettedCoding is the FHIR SUBSETTED tag a summary view carries: a Coding from the
// v3 ObservationValue code system marking the resource as a partial representation. It is
// a constant payload with no patient data.
const subsettedCoding = `{"system":"http://terminology.hl7.org/CodeSystem/v3-ObservationValue","code":"SUBSETTED","display":"subsetted"}`

// summaryRegistry maps a resourceType discriminator to its generated summary descriptor.
// Like the factory and validation registries it is the package's only mutable state for
// summary serialization: the generated per-release init() functions are its only writers,
// populating it before main runs, and it is read-only in practice thereafter. The RWMutex
// guards it so a stray late registration can never race a concurrent MarshalSummary,
// keeping the "no state a caller can race on" guarantee.
var (
	summaryRegistryMu sync.RWMutex
	summaryRegistry   = map[string]SummaryDescriptor{}
)

// RegisterSummaryDescriptor records the summary descriptor for a resourceType. It exists
// for the generated per-release summary init() to call; a consumer never calls it
// directly. It is exported only because the generated release package and this root
// package are distinct packages, so the registration hook must cross the package boundary.
//
// It panics on an empty resourceType or a duplicate registration: a duplicate means the
// generator emitted conflicting descriptors (or two releases collided before that
// collision was resolved), a build-time defect that must fail loudly rather than let one
// descriptor silently shadow the other.
func RegisterSummaryDescriptor(resourceType string, d SummaryDescriptor) {
	if resourceType == "" {
		panic("fhir: RegisterSummaryDescriptor: empty resourceType")
	}
	summaryRegistryMu.Lock()
	defer summaryRegistryMu.Unlock()
	if _, exists := summaryRegistry[resourceType]; exists {
		panic("fhir: RegisterSummaryDescriptor: duplicate descriptor for resourceType " + resourceType)
	}
	summaryRegistry[resourceType] = d
}

// lookupSummaryDescriptor returns the descriptor for a resourceType and whether one is
// registered, under the registry read lock so it never races a registration.
func lookupSummaryDescriptor(resourceType string) (SummaryDescriptor, bool) {
	summaryRegistryMu.RLock()
	defer summaryRegistryMu.RUnlock()
	d, ok := summaryRegistry[resourceType]
	return d, ok
}
