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

	// EmbeddedBase is the Go base type this type embeds (DomainResource or Resource
	// for a resource, Element or BackboneElement for a complex datatype), or empty
	// for a base type itself. The emitter renders it as the struct's first,
	// anonymous field so the base members (id, meta, text, extension, ...) are
	// promoted and the type is faithful without restating them inline.
	EmbeddedBase string

	// IsBaseType marks one of the shared abstract base types (Element,
	// BackboneElement, Resource, DomainResource). A base type is emitted as a plain
	// struct: it carries no resourceType discriminator, no ResourceType method, and
	// no MarshalJSON, even though Resource and DomainResource classify as FHIR
	// resources. That suppression is mandatory, because a value-embedded type whose
	// MarshalJSON is promoted would shadow the embedding resource's own MarshalJSON
	// and drop every non-base field on the wire.
	IsBaseType bool

	// Backbones are the distinct nested backbone structs this type owns, deduplicated
	// by shape and sorted by Go name so the emitter output is stable.
	Backbones []PlannedBackbone

	// Choices are the polymorphic "[x]" groups this type owns at the top level, each
	// rendered as a sealed value interface, suffixed storage fields, a getter, and
	// mutually-exclusive setters. The storage fields appear in Fields (in canonical
	// order) so the struct carries them; Choices carries the accessor machinery the
	// emitter renders alongside the struct.
	Choices []PlannedChoice
}

// IsResource reports whether the planned type is a FHIR resource the emitter renders
// with the resourceType discriminator, the ResourceType method, and the always-emit-
// resourceType MarshalJSON. A base type is never treated as a resource for emission
// even though Resource and DomainResource classify as resources, so a base type
// stays a plain struct with no promoted MarshalJSON.
func (t PlannedType) IsResource() bool { return t.Kind == model.KindResource && !t.IsBaseType }

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

	// EmbeddedBase is the Go base type this backbone embeds: BackboneElement for a
	// resource backbone (which carries id, extension, and modifierExtension) or
	// Element for a datatype backbone (id and extension). The members the base
	// supplies are dropped from Fields and promoted through the embedded base.
	EmbeddedBase string

	// Choices are the polymorphic "[x]" groups this backbone owns, rendered the same
	// way a top-level type's choices are. The suffixed storage fields appear in
	// Fields; Choices carries the accessor machinery.
	Choices []PlannedChoice
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

// Options tunes a planning run.
type Options struct {
	// IsBaseType marks the planned type as one of the shared abstract base types
	// (Element, BackboneElement, Resource, DomainResource). A base type is planned
	// with its own members intact, embeds no base, and suppresses the primitive
	// "_field" siblings: a base type must not define MarshalJSON, because a value-
	// embedded type whose MarshalJSON is promoted would shadow the embedding type's
	// own MarshalJSON and drop every non-base field on the wire. Base-member
	// primitive extensions (Resource.id, language, implicitRules) are therefore not
	// modelled in v1; the extension machinery is a later increment.
	IsBaseType bool
}

// baseStrip describes how a concrete type inherits its base members: the Go base
// type it embeds and the set of element names that base supplies (and which are
// therefore dropped from the concrete type's own field list, since the embedded
// base carries them). A faithful resource keeps meta/text/contained/extension and
// the rest through the embedded DomainResource rather than restating them.
type baseStrip struct {
	embed string          // Go base type name to embed (for example "DomainResource")
	drop  map[string]bool // element names supplied by the embedded base
}

// elementBaseMembers are the members the Element base supplies (id, extension). A
// complex datatype embeds Element and drops these.
var elementBaseMembers = map[string]bool{"id": true, "extension": true}

// backboneBaseMembers are the members the BackboneElement base supplies (Element's
// id and extension plus modifierExtension). A resource backbone, and a complex
// datatype that carries a top-level modifierExtension (a BackboneType such as
// Dosage or Timing), embeds BackboneElement and drops these.
var backboneBaseMembers = map[string]bool{"id": true, "extension": true, "modifierExtension": true}

// resourceBaseMembers are the members the Resource base supplies (id, meta,
// implicitRules, language). A resource that is not a DomainResource (Bundle,
// Binary, Parameters) embeds Resource and drops these.
var resourceBaseMembers = map[string]bool{"id": true, "meta": true, "implicitRules": true, "language": true}

// domainResourceBaseMembers are the members the DomainResource base supplies (the
// Resource members plus text, contained, extension, modifierExtension). Most
// resources embed DomainResource and drop these.
var domainResourceBaseMembers = map[string]bool{
	"id": true, "meta": true, "implicitRules": true, "language": true,
	"text": true, "contained": true, "extension": true, "modifierExtension": true,
}

// planBase decides which base a concrete type embeds and which of its members the
// base supplies, by reading the type's own top-level members rather than walking
// the base-definition chain (which the planner does not resolve): a resource that
// carries text/contained is a DomainResource, one that does not is a plain
// Resource; a complex datatype that carries a top-level modifierExtension is a
// BackboneType (embeds BackboneElement), otherwise it embeds Element. This keeps
// the decision release-agnostic and dependent only on the IR.
func planBase(t *model.Type) baseStrip {
	top := topLevelNames(t.Root)
	if t.Kind == model.KindResource {
		if top["text"] || top["contained"] {
			return baseStrip{embed: "DomainResource", drop: domainResourceBaseMembers}
		}
		return baseStrip{embed: "Resource", drop: resourceBaseMembers}
	}
	if top["modifierExtension"] {
		return baseStrip{embed: "BackboneElement", drop: backboneBaseMembers}
	}
	return baseStrip{embed: "Element", drop: elementBaseMembers}
}

// topLevelNames is the set of direct-member element names of a type's root.
func topLevelNames(root *model.Element) map[string]bool {
	names := make(map[string]bool, len(root.Children))
	for _, c := range root.Children {
		names[c.Name] = true
	}
	return names
}

// PlanType turns a classified model.Type into an emitter-ready PlannedType. It plans
// each top-level element into a Go field with a deterministic, collision-free name,
// collects the distinct nested backbone structs (deduplicated by shape), and records
// the canonical field order. A concrete type embeds the shared base it inherits from
// (DomainResource/Resource for a resource, Element/BackboneElement for a datatype)
// and drops the members that base supplies, so a generated resource is faithful
// (meta, text, contained, extension, ...) without restating the base inline. A base
// type itself (Options.IsBaseType) embeds nothing and keeps every member. The
// planner makes no I/O and reads no template.
func PlanType(t *model.Type, opts Options) PlannedType {
	pt := PlannedType{
		GoName:     GoTypeName(t.Name),
		FHIRName:   t.Name,
		Kind:       t.Kind,
		IsBaseType: opts.IsBaseType,
	}

	var drop map[string]bool
	if !opts.IsBaseType {
		base := planBase(t)
		pt.EmbeddedBase = base.embed
		drop = base.drop
	}

	backbones := newBackboneSet()
	used := map[string]bool{}
	if pt.EmbeddedBase != "" {
		// Reserve the embedded base's Go name so an own field never collides with it.
		used[pt.EmbeddedBase] = true
	}
	pt.Fields, pt.Choices = planFields(t.Name, pt.GoName, t.Root.Children, drop, opts.IsBaseType, used, backbones)
	pt.Backbones = backbones.sorted()
	return pt
}

// planFields plans the direct children of a node into fields, resolving Go-name
// collisions deterministically within the node's scope and recording any nested
// backbone shape it encounters. ownerType is the FHIR type the backbone names are
// rooted under; ownerGoName is the Go name of the enclosing struct, used to name a
// choice group's sealed value interface. drop names the members the embedded base
// supplies (skipped here); suppressSiblings is set for a base type, which must not
// emit primitive "_field" siblings (and so must not define MarshalJSON). Each scope
// (a struct's field set) gets its own used-name map, so a field named "Type" in one
// struct never disambiguates a "Type" in another.
//
// A choice "[x]" element is expanded in place into one suffixed pointer storage field
// per branch (ValueQuantity, ValueString, ...) and one PlannedChoice carrying the
// accessor machinery the emitter renders alongside the struct. The stub that planned a
// choice as a single first-branch-typed field is gone (FHIR-001/002): the suffixed
// storage fields make a two-branches-set state representable only through the setters,
// which clear the siblings, and each storage field is omitempty so exactly one
// suffixed key is ever authored.
func planFields(ownerType, ownerGoName string, children []*model.Element, drop map[string]bool, suppressSiblings bool, used map[string]bool, backbones *backboneSet) ([]Field, []PlannedChoice) {
	fields := make([]Field, 0, len(children))
	var choices []PlannedChoice
	for _, child := range children {
		if drop[child.Name] {
			continue
		}

		if child.IsChoice {
			cf, pc := planChoiceField(ownerType, ownerGoName, child, used)
			fields = append(fields, cf...)
			choices = append(choices, pc)
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
			// expanded-forever type. The donor backbone and this boundary resolve to
			// the same GoBackboneTypeName, so the recursion collapses to one
			// self-referential named type rather than two distinct names.
			anchorName := GoBackboneTypeName(ownerType, pathSegmentsAfterOwner(ownerType, child.ContentReference))
			f.GoType = decorateBackbone(child.Cardinality, anchorName)
		case child.IsBackbone():
			bb := planBackbone(ownerType, child, backbones)
			f.GoType = decorateBackbone(child.Cardinality, bb.GoName)
		}

		fields = append(fields, f)
		if f.Primitive && !suppressSiblings {
			fields = append(fields, planPrimitiveSibling(f, used))
		}
	}
	return fields, choices
}

// planChoiceField expands a choice "[x]" element into its suffixed pointer storage
// fields and the PlannedChoice that carries the accessor machinery. The choice's Go
// field stem is resolved first (so a sibling that already took "Value" pushes the
// choice to "Value2" and every storage field follows that stem), then one storage
// field per branch is planned. The stem name itself is not added to the struct as a
// field — only the suffixed storage fields are — so there is no bare untyped choice
// field, which is exactly what makes a two-branches-set state unrepresentable outside
// the mutually-exclusive setters.
func planChoiceField(ownerType, ownerGoName string, e *model.Element, used map[string]bool) ([]Field, PlannedChoice) {
	stem := resolveCollision(GoFieldName(e.Name), used)
	pc := planChoice(ownerGoName, stem, e, used)
	fields := make([]Field, 0, len(pc.Branches))
	for _, b := range pc.Branches {
		fields = append(fields, Field{
			GoName:   b.Field,
			GoType:   "*" + b.GoType,
			JSONName: b.JSONName,
			Optional: true,
			Doc:      e.Path,
			Element:  e,
		})
	}
	return fields, pc
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
//
// A backbone embeds its element base just as a top-level type does: a resource
// backbone carries a modifierExtension child (a BackboneElement) and embeds
// BackboneElement, a datatype backbone carries only id/extension (an Element) and
// embeds Element. The base members are dropped from the backbone's own fields.
func planBackbone(ownerType string, e *model.Element, backbones *backboneSet) PlannedBackbone {
	embed, drop := backboneBase(e)
	used := map[string]bool{embed: true}
	goName := GoBackboneTypeName(ownerType, pathSegmentsAfterOwner(ownerType, e.Path))
	fields, choices := planFields(ownerType, goName, e.Children, drop, false, used, backbones)
	return backbones.add(goName, fields, embed, choices)
}

// backboneBase decides which element base a backbone embeds and which members that
// base supplies. A backbone carrying a modifierExtension child is a BackboneElement
// (a resource backbone, or a BackboneType datatype's nested element); one carrying
// only id/extension is a plain Element (a datatype backbone such as Timing.repeat).
func backboneBase(e *model.Element) (string, map[string]bool) {
	for _, c := range e.Children {
		if c.Name == "modifierExtension" {
			return "BackboneElement", backboneBaseMembers
		}
	}
	return "Element", elementBaseMembers
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
// Go types — shape-dedup, not path-dedup. The embedded base is part of the
// fingerprint so two backbones with the same own-fields but a different element base
// stay distinct.
func (s *backboneSet) add(goName string, fields []Field, embed string, choices []PlannedChoice) PlannedBackbone {
	fp := embed + "|" + fingerprint(fields)
	if existing, ok := s.byShape[fp]; ok {
		return existing
	}
	bb := PlannedBackbone{GoName: goName, Fields: fields, EmbeddedBase: embed, Choices: choices}
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
