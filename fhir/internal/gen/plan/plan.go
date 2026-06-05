package plan

import (
	"sort"
	"strings"

	"github.com/codeninja55/go-radx/fhir/internal/gen/model"
)

// PlannedType is the emitter-ready plan for one FHIR type: a Go struct with decided
// fields, plus the distinct nested backbone structs the type owns. The emitter walks
// a PlannedType and renders it mechanically — every Go-shape question is already
// answered here. A PlannedType is release-agnostic in the same sense the IR is: it
// reflects only what the model describes.
type PlannedType struct {
	// GoName is the Go type identifier (the FHIR type name lifted to an exported
	// identifier).
	GoName string

	// FHIRName is the source FHIR type name, kept for documentation and traceability.
	FHIRName string

	// Kind is the model kind (complex-type, resource, primitive-type), carried so the
	// emitter selects the right template and the resource stage adds a resourceType
	// discriminator.
	Kind model.Kind

	// Doc is the type's godoc summary.
	Doc string

	// Fields are the planned top-level fields in canonical (snapshot) order.
	Fields []Field

	// Backbones are the distinct nested backbone structs this type owns, deduplicated
	// by shape and sorted by Go name so the emitter output is stable.
	Backbones []PlannedBackbone
}

// IsResource reports whether the planned type is a FHIR resource, so the emitter
// renders the resourceType discriminator, the ResourceType method, and the
// always-emit-resourceType MarshalJSON for it (and not for a plain datatype). The
// kind decision is the planner's; the emitter only reads this flag.
func (t PlannedType) IsResource() bool { return t.Kind == model.KindResource }

// KindNoun returns the English noun the godoc summary uses for the type's kind
// ("resource" or "datatype"), so the generated comment reads naturally for both.
func (t PlannedType) KindNoun() string {
	if t.Kind == model.KindResource {
		return "resource"
	}
	return "datatype"
}

// HasPrimitiveSibling reports whether the type owns any primitive "_field"
// sibling, the condition under which the emitter generates custom JSON methods.
// Both shapes go through the generated methods so the "_field" key is governed by
// the sibling's emptiness, not by Go's omitempty (which cannot drop a non-nil but
// empty *PrimitiveElement): a scalar sibling is omitted when it carries no id or
// extension, and a repeating sibling is null-aligned with its value array.
func (t PlannedType) HasPrimitiveSibling() bool { return hasPrimitiveSibling(t.Fields) }

// ScalarPrimitives returns the type's scalar primitive sibling fields, which the
// generated MarshalJSON folds in only when non-empty and the generated
// UnmarshalJSON lifts out before decoding the value struct.
func (t PlannedType) ScalarPrimitives() []Field { return primitiveSiblings(t.Fields, false) }

// RepeatingPrimitives returns the type's repeating primitive sibling fields, which
// the generated MarshalJSON null-aligns and the generated UnmarshalJSON lifts out
// before decoding the value struct.
func (t PlannedType) RepeatingPrimitives() []Field { return primitiveSiblings(t.Fields, true) }

// PlannedBackbone is one distinct nested backbone struct: a Go name and its fields.
// Multiple occurrence paths that share the same shape collapse to one PlannedBackbone
// (deduplicated by shape), so a resource with the same anonymous structure at several
// paths emits the type once.
type PlannedBackbone struct {
	GoName string
	Fields []Field
}

// HasPrimitiveSibling reports whether the backbone owns a primitive "_field"
// sibling needing custom marshalling.
func (b PlannedBackbone) HasPrimitiveSibling() bool { return hasPrimitiveSibling(b.Fields) }

// ScalarPrimitives returns the backbone's scalar primitive sibling fields.
func (b PlannedBackbone) ScalarPrimitives() []Field { return primitiveSiblings(b.Fields, false) }

// RepeatingPrimitives returns the backbone's repeating primitive sibling fields.
func (b PlannedBackbone) RepeatingPrimitives() []Field { return primitiveSiblings(b.Fields, true) }

// hasPrimitiveSibling reports whether a field set contains any "_field" sibling.
func hasPrimitiveSibling(fields []Field) bool {
	for _, f := range fields {
		if f.IsPrimitiveSibling() {
			return true
		}
	}
	return false
}

// primitiveSiblings returns the "_field" sibling fields in a field set whose
// repeating-ness matches repeats, in declaration order, so the emitter renders the
// sibling-handling calls deterministically.
func primitiveSiblings(fields []Field, repeats bool) []Field {
	var out []Field
	for _, f := range fields {
		if f.IsPrimitiveSibling() && f.Repeats == repeats {
			out = append(out, f)
		}
	}
	return out
}

// Options tunes a planning run. The skeleton increment plans a single representative
// datatype without the shared Element base machinery (id/extension and the
// primitive-extension siblings arrive in later increments), so SkipBaseMembers omits
// the inherited base members a complex type carries until that machinery exists.
type Options struct {
	// SkipBaseMembers omits the Element/DataType base members (id and extension) from
	// a planned complex type. It is set while the shared base type is not yet
	// generated; once the base machinery lands the members are embedded instead of
	// skipped and this option is retired.
	SkipBaseMembers bool
}

// baseMemberNames are the base members a complex type or resource inherits from
// Element, DataType, Resource, and DomainResource. They are planned away under
// Options.SkipBaseMembers until the shared base types exist (Increment 5/6 embeds
// them and retires the option). The set covers both the Element base (id, extension)
// and the resource bases (meta, implicitRules, language) plus the DomainResource
// bases (text, contained, modifierExtension), so a representative resource planned
// before the base machinery lands carries only its own elements and compiles
// against the already-generated type set.
var baseMemberNames = map[string]bool{
	"id":                true,
	"extension":         true,
	"meta":              true,
	"implicitRules":     true,
	"language":          true,
	"text":              true,
	"contained":         true,
	"modifierExtension": true,
}

// PlanType turns a classified model.Type into an emitter-ready PlannedType. It plans
// each top-level element into a Go field with a deterministic, collision-free name,
// collects the distinct nested backbone structs (deduplicated by shape), and records
// the canonical field order. The planner makes no I/O and reads no template.
func PlanType(t *model.Type, opts Options) PlannedType {
	pt := PlannedType{
		GoName:   GoTypeName(t.Name),
		FHIRName: t.Name,
		Kind:     t.Kind,
	}

	backbones := newBackboneSet()
	used := map[string]bool{}
	pt.Fields = planFields(t.Name, t.Root.Children, opts, used, backbones)
	pt.Backbones = backbones.sorted()
	return pt
}

// planFields plans the direct children of a node into fields, resolving Go-name
// collisions deterministically within the node's scope and recording any nested
// backbone shape it encounters. ownerType is the FHIR type the backbone names are
// rooted under. Each scope (a struct's field set) gets its own used-name map, so a
// field named "Type" in one struct never disambiguates a "Type" in another.
func planFields(ownerType string, children []*model.Element, opts Options, used map[string]bool, backbones *backboneSet) []Field {
	fields := make([]Field, 0, len(children))
	for _, child := range children {
		if opts.SkipBaseMembers && baseMemberNames[child.Name] {
			continue
		}

		f := PlanField(child)
		f.GoName = resolveCollision(f.GoName, used)
		f.Doc = child.Path

		switch {
		case isRecursionBoundary(child):
			// A contentReference boundary (a node that kept its marker because its
			// anchor is already on the expansion chain, so it has no children) is a
			// self- or mutually-recursive backbone. Its Go type is the named backbone
			// type of the anchor it points back to, decorated by cardinality, so a
			// recursive structure becomes a slice-of-self rather than an undefined or
			// expanded-forever type.
			anchorName := GoBackboneTypeName(ownerType, pathSegmentsAfterOwner(ownerType, child.ContentReference))
			f.GoType = decorateBackbone(child.Cardinality, anchorName)
		case child.IsBackbone():
			bb := planBackbone(ownerType, child, opts, backbones)
			f.GoType = decorateBackbone(child.Cardinality, bb.GoName)
		}

		fields = append(fields, f)
		if f.Primitive {
			fields = append(fields, planPrimitiveSibling(f, used))
		}
	}
	return fields
}

// planPrimitiveSibling builds the "_field" extension sibling for a primitive value
// field: the XxxElement companion that carries the value's id and extensions
// (Codex FHIR-005). A scalar primitive's sibling is a single *fhir.PrimitiveElement
// keyed "_field"; a repeating primitive's sibling is a []*fhir.PrimitiveElement
// null-aligned to the value array. The Go name is the value field's name with an
// "Element" suffix, collision-resolved in the same scope so two siblings never
// clash (and never clash with a real element literally named "<x>Element"). The
// JSON wire key is the value's key prefixed with an underscore.
func planPrimitiveSibling(value Field, used map[string]bool) Field {
	goName := resolveCollision(value.GoName+"Element", used)
	goType := "*fhir.PrimitiveElement"
	if value.Repeats {
		goType = "[]*fhir.PrimitiveElement"
	}
	return Field{
		GoName:    goName,
		GoType:    goType,
		JSONName:  "_" + value.JSONName,
		Optional:  true,
		Repeats:   value.Repeats,
		SiblingOf: value.GoName,
	}
}

// planBackbone plans a nested backbone element into a distinct PlannedBackbone,
// deduplicating by shape: a backbone whose field structure matches one already
// collected reuses that type rather than emitting a second identical struct. The
// canonical name is derived from the backbone's occurrence path the first time a
// given shape is seen, so the same shape at several paths always resolves to the same
// Go type. planBackbone recurses, so a backbone containing another backbone is
// planned all the way down.
func planBackbone(ownerType string, e *model.Element, opts Options, backbones *backboneSet) PlannedBackbone {
	used := map[string]bool{}
	fields := planFields(ownerType, e.Children, opts, used, backbones)
	goName := GoBackboneTypeName(ownerType, pathSegmentsAfterOwner(ownerType, e.Path))
	return backbones.add(goName, fields)
}

// isRecursionBoundary reports whether an element is a contentReference recursion
// boundary: the model keeps the contentReference marker (and leaves the node
// childless) when the anchor is already being expanded above it, so a node carrying
// a marker with no children is the boundary of a self- or mutually-recursive
// backbone. The planner turns it into a named reference back to the anchor's
// backbone type rather than expanding it.
func isRecursionBoundary(e *model.Element) bool {
	return e.ContentReference != "" && len(e.Children) == 0
}

// decorateBackbone applies the cardinality decoration to a backbone field's Go type:
// a repeating backbone is a slice of the backbone type, a single one is a pointer to
// it (so absence is nil, consistent with the scalar rule).
func decorateBackbone(c model.Cardinality, backboneType string) string {
	if c.Repeats() {
		return "[]" + backboneType
	}
	return "*" + backboneType
}

// pathSegmentsAfterOwner returns the path segments below the owner type, so
// "Observation.component.referenceRange" under owner "Observation" yields
// ["component", "referenceRange"], the segments GoBackboneTypeName concatenates.
func pathSegmentsAfterOwner(ownerType, path string) []string {
	segs := strings.Split(path, ".")
	if len(segs) > 0 && segs[0] == ownerType {
		return segs[1:]
	}
	return segs
}

// backboneSet collects distinct backbone shapes for one type, deduplicating by a
// structural fingerprint so two identical anonymous structures emit one Go type. It
// keys the first-seen Go name by fingerprint; a later occurrence with the same
// fingerprint returns the already-recorded backbone.
type backboneSet struct {
	byShape map[string]PlannedBackbone // fingerprint -> backbone
}

func newBackboneSet() *backboneSet {
	return &backboneSet{byShape: map[string]PlannedBackbone{}}
}

// add records a backbone shape under its structural fingerprint and returns the
// canonical backbone for that shape. If the shape was seen before, the previously
// recorded backbone (and its name) is returned, so the same shape never produces two
// Go types — shape-dedup, not path-dedup.
func (s *backboneSet) add(goName string, fields []Field) PlannedBackbone {
	fp := fingerprint(fields)
	if existing, ok := s.byShape[fp]; ok {
		return existing
	}
	bb := PlannedBackbone{GoName: goName, Fields: fields}
	s.byShape[fp] = bb
	return bb
}

// sorted returns the collected backbones ordered by Go name, so the emitter writes
// them in a stable order regardless of the order they were discovered (map iteration
// is unordered).
func (s *backboneSet) sorted() []PlannedBackbone {
	out := make([]PlannedBackbone, 0, len(s.byShape))
	for _, bb := range s.byShape {
		out = append(out, bb)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GoName < out[j].GoName })
	return out
}

// fingerprint renders a backbone's field set as a canonical string used to detect
// two structurally identical backbones. It joins each field's JSON name and Go type
// (which already encodes pointer/slice and the resolved backbone type name), so two
// backbones with the same fields in the same order share a fingerprint regardless of
// the occurrence path they came from.
func fingerprint(fields []Field) string {
	var b strings.Builder
	for _, f := range fields {
		b.WriteString(f.JSONName)
		b.WriteByte(' ')
		b.WriteString(f.GoType)
		b.WriteByte(';')
	}
	return b.String()
}
