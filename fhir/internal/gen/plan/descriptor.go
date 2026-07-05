package plan

import (
	"strings"

	"github.com/codeninja55/go-radx/fhir/internal/gen/model"
)

// ValidationDescriptor is the emitter-ready plan for one resource's structural
// validation: the required elements (checked for presence, not truthiness), the
// top-level choice ([x]) groups (checked for at-most-one branch set), the top-level
// required-binding code fields (checked against their generated closed enum), and the
// date/time-family primitive fields (checked against the release's lexical rules). The
// emitter renders it as a generated fhir.ValidationDescriptor registered at init time,
// so the validation engine consumes generated metadata rather than reflecting over the
// resource at call time (Codex FHIR-007 presence, FHIR-001 mutual exclusion, FHIR-013
// binding codes, FHIR-008 primitive lexical forms).
//
// Required-presence and primitive-lexical checks reach inside backbone elements: each
// backbone type with (transitively) required children or date/time-family fields gets a
// generated walk helper (Helpers), and the top-level closures call into them for each
// present backbone instance (RequiredCalls/LexicalCalls), so Bundle.entry.request.method
// is checked whenever entry.request is present. Nested choice and binding checks, and
// the interiors of complex datatypes (Period.start, Timing.event), remain deferred; the
// documented validation contract (docs/conformance/fhir.md) records this boundary. The
// bdl-* Bundle invariants and reference integrity are not expressible from the
// StructureDefinition and are composed by the hand-written Bundle descriptor extra
// checks, not emitted here.
type ValidationDescriptor struct {
	// GoName is the resource's Go type identifier, used to type-assert the
	// fhir.Resource the generated closures receive.
	GoName string

	// FHIRName is the source FHIR resource name, used to build element paths
	// ("Patient.gender") that name where an issue lives without any value.
	FHIRName string

	// Required are the required top-level elements (cardinality min >= 1), each a
	// generated presence check.
	Required []RequiredField

	// Choices are the resource's top-level choice ([x]) groups, each a generated
	// at-most-one-branch check over the suffixed storage fields.
	Choices []ChoiceCheck

	// Bindings are the resource's top-level required-binding code fields, each a
	// generated set-membership check against the field's closed enum.
	Bindings []BindingCheck

	// Primitives are the resource's top-level date/dateTime/time/instant fields —
	// including the boxed date-family branches of top-level choice groups — each a
	// generated lexical check against the release's hand-written validators.
	Primitives []PrimitiveCheck

	// RequiredCalls and LexicalCalls are the top-level backbone fields the Required
	// and Primitives closures walk into, one call per field whose backbone type has
	// (transitively) required children or date-family fields respectively.
	RequiredCalls []BackboneCall
	LexicalCalls  []BackboneCall

	// CheckExtensions reports whether the resource embeds DomainResource and so
	// carries the promoted extension/modifierExtension arrays whose every Extension
	// (recursively) must carry its required url; the Required closure walks them
	// through the hand-written missingExtensionURLs.
	CheckExtensions bool

	// Helpers are the resource's backbone walk helpers, one per distinct backbone
	// type with (transitively) required children or date-family fields, in Go-name
	// order. The emitter renders each as a missingRequired<GoName> and/or
	// lexicalIssues<GoName> function the closures and sibling helpers call.
	Helpers []BackboneHelper

	// HasExtra reports whether the resource carries hand-written extra checks the
	// descriptor composes (the Bundle bdl-* invariants and reference integrity). The
	// emitter wires the resource's <GoName>ValidateExtra hook only when set.
	HasExtra bool

	// Summary are the resource's top-level elements with the per-element flags
	// MarshalSummary filters on (isSummary, mandatory, modifier, the narrative, and the
	// Bundle count element), one entry per wire key. A choice ([x]) group contributes one
	// entry per suffixed branch key, all sharing the group's flags. The emitter renders
	// these as a generated fhir.SummaryDescriptor registered alongside the validation
	// descriptor, so the summary serialiser filters from generated metadata rather than
	// reflecting over the resource (Codex FHIR-012's data-driven companion).
	Summary []SummaryFlag
}

// SummaryFlag is one top-level element's summary metadata: its wire key and the flags the
// five _summary modes test. Presence-on-the-wire is irrelevant here; the flag set is the
// element's static definition, so MarshalSummary keeps or drops a key by definition, not
// by the value a particular resource carries.
type SummaryFlag struct {
	// JSONName is the element's wire key ("gender", "deceasedBoolean"). For a choice
	// branch it is the suffixed key.
	JSONName string

	// IsSummary records the StructureDefinition isSummary flag; SummaryTrue keeps the
	// element when set.
	IsSummary bool

	// IsMandatory records whether the element's minimum cardinality is at least one;
	// SummaryTrue and SummaryText always keep a mandatory element.
	IsMandatory bool

	// IsModifier records the StructureDefinition isModifier flag; SummaryTrue always keeps
	// a modifier element.
	IsModifier bool

	// IsText reports whether the element is the DomainResource narrative ("text"), kept by
	// SummaryText and dropped by SummaryData.
	IsText bool

	// IsCount reports whether the element is the one a count view keeps (a Bundle's
	// "total"); SummaryCount keeps it and drops every other non-infrastructure element.
	IsCount bool
}

// RequiredField is one required top-level element: the Go field that must be present
// and the FHIR element path naming it. Presence is tested by the field being non-nil
// (a single-valued pointer) or a non-empty slice (a repeating element), never by the
// value being non-zero, so a present required false or 0 passes (Codex FHIR-007).
type RequiredField struct {
	// GoName is the Go field to test for presence.
	GoName string

	// Path is the FHIR element path naming the element in an issue ("Observation.status").
	Path string

	// Repeats reports whether the element is a repeating slice (presence is len > 0)
	// rather than a single-valued pointer (presence is != nil).
	Repeats bool
}

// ChoiceCheck is one choice ([x]) group's mutual-exclusion check: the suffixed Go
// storage fields and the FHIR path naming the group. The check counts the non-nil
// storage fields and flags the group when more than one is set, catching a direct
// two-field write the setters would have prevented (Codex FHIR-001).
type ChoiceCheck struct {
	// Fields are the suffixed Go storage field names of the group's branches
	// ("DeceasedBoolean", "DeceasedDateTime"). Each is a pointer, so presence is != nil.
	Fields []string

	// Path is the FHIR choice path naming the group in an issue ("Patient.deceased[x]").
	Path string
}

// BindingCheck is one required-binding code field's validity check: the Go field, its
// validator (the generated valid<Enum> closure name), the enum type name, and the FHIR
// path. The check validates a present code against the closed set and flags an
// out-of-set value (Codex FHIR-013); an unset field is not checked, because absence is a
// cardinality concern, not a binding one.
type BindingCheck struct {
	// GoName is the Go field holding the code value.
	GoName string

	// Validator is the generated set-membership function name for the field's enum
	// ("validAdministrativeGender"), called with the field's underlying string value.
	Validator string

	// EnumName is the enum's Go type name, named in the issue diagnostic
	// ("AdministrativeGender") so the issue identifies the binding, never the value.
	EnumName string

	// Path is the FHIR element path naming the code field ("Patient.gender").
	Path string

	// Repeats reports whether the code field is a repeating slice ([]Enum) rather than
	// a single-valued pointer (*Enum), so the check iterates the slice.
	Repeats bool
}

// PrimitiveCheck is one date/time-family primitive field's lexical check: the Go field,
// the release-local validator the check calls, the FHIR primitive type name for the
// diagnostic, and the element path. A boxed check reads a choice branch's primitive
// wrapper (FHIRDate, ...) through a string conversion; a plain check reads *string or
// []string directly. The diagnostic names the type, never the value — a date can itself
// be PHI.
type PrimitiveCheck struct {
	// GoName is the Go field holding the primitive value.
	GoName string

	// Path is the element path of the field: absolute on a top-level check
	// ("Patient.birthDate"), a "."-prefixed suffix inside a backbone helper
	// (".started"), appended to the helper's runtime path.
	Path string

	// Validator is the release-local lexical validator the check calls
	// ("validDateLexical", "validDateTimeLexical", "validTimeLexical",
	// "validInstantLexical").
	Validator string

	// TypeName is the FHIR primitive type name the diagnostic carries ("dateTime").
	TypeName string

	// Repeats reports whether the field is a repeating []string, so the check
	// iterates and indexes the path.
	Repeats bool

	// Boxed reports whether the field is a choice branch's primitive wrapper, read
	// through a string conversion rather than a plain dereference.
	Boxed bool
}

// BackboneCall is one walk step into a backbone-typed field: the parent's Go field, the
// path segment the walk appends, the backbone type whose helper is called, and whether
// the field repeats (walked per element with an indexed path) or is a single pointer
// (walked when non-nil).
type BackboneCall struct {
	FieldGoName string
	JSONName    string
	TypeGoName  string
	Repeats     bool
}

// BackboneHelper is one backbone type's generated walk helper: its own required fields
// and date-family primitive checks (paths "."-prefixed, relative to the runtime path
// parameter) plus the calls into child backbones. EmitRequired/EmitLexical report which
// helper functions the emitter renders — a backbone with no transitive requireds gets no
// missingRequired helper even if it needs a lexicalIssues one, and vice versa.
type BackboneHelper struct {
	GoName        string
	Required      []RequiredField
	Primitives    []PrimitiveCheck
	RequiredCalls []BackboneCall
	LexicalCalls  []BackboneCall
	EmitRequired  bool
	EmitLexical   bool
}

// EmitsRequired reports whether the descriptor renders a Required closure: it has
// top-level required fields, backbone walks with transitive requireds, or the
// extension-url walk.
func (vd ValidationDescriptor) EmitsRequired() bool {
	return len(vd.Required) > 0 || len(vd.RequiredCalls) > 0 || vd.CheckExtensions
}

// EmitsPrimitives reports whether the descriptor renders a Primitives closure: it has
// top-level date-family fields or backbone walks with transitive date-family fields.
func (vd ValidationDescriptor) EmitsPrimitives() bool {
	return len(vd.Primitives) > 0 || len(vd.LexicalCalls) > 0
}

// primitiveLexicalKinds maps the date/time-family primitive type codes to the
// release-local lexical validator the generated check calls. The four codes are the
// primitives whose lexical rules go beyond what the Go type carries (decimal preserves
// its lexical form in fhir.Decimal and integer64 is parsed at decode time; the string
// family has no lexical grammar to enforce in v1).
var primitiveLexicalKinds = map[string]string{
	"date":     "validDateLexical",
	"dateTime": "validDateTimeLexical",
	"time":     "validTimeLexical",
	"instant":  "validInstantLexical",
}

// primitiveWrapperLexicalCodes maps a choice branch's primitive wrapper type back to
// its date/time-family type code, so a boxed branch (FHIRDateTime) gets the same
// lexical check as a plain field.
var primitiveWrapperLexicalCodes = map[string]string{
	"FHIRDate":     "date",
	"FHIRDateTime": "dateTime",
	"FHIRTime":     "time",
	"FHIRInstant":  "instant",
}

// PlanValidationDescriptor builds the validation descriptor for one planned resource by
// reading the model's top-level elements: a required element (min >= 1) becomes a
// RequiredField, a choice ([x]) element becomes a ChoiceCheck over its planned suffixed
// storage fields, a required-binding code element becomes a BindingCheck against its
// generated enum, and a date/time-family primitive element (or boxed choice branch)
// becomes a PrimitiveCheck against the release's lexical validators. The resource's
// backbone types are analysed transitively so the Required and Primitives closures walk
// into every present backbone instance that can carry a violation. It is a pure function
// of the model and the plan; it reads no I/O and is deterministic, so the emitted
// descriptor file is byte-stable.
//
// PlanValidationDescriptor returns ok=false for a type that is not a resource (a complex
// datatype has no resourceType to register under) and for the abstract base types. The
// caller skips a not-ok type, so only concrete resources get a registered descriptor.
func PlanValidationDescriptor(t *model.Type, pt PlannedType, bindings BindingResolver) (ValidationDescriptor, bool) {
	if !pt.IsResource() {
		return ValidationDescriptor{}, false
	}

	vd := ValidationDescriptor{
		GoName:          pt.GoName,
		FHIRName:        t.Name,
		HasExtra:        t.Name == "Bundle",
		CheckExtensions: pt.EmbeddedBase == "DomainResource",
	}

	choiceByBase := indexChoicesByBase(pt.Choices)

	for _, child := range t.Root.Children {
		vd.Summary = append(vd.Summary, summaryFlags(child, choiceByBase)...)
		if child.IsChoice {
			if check, ok := choiceCheck(t.Name, child, choiceByBase); ok {
				vd.Choices = append(vd.Choices, check)
			}
			continue
		}
		if child.Cardinality.Required() {
			vd.Required = append(vd.Required, RequiredField{
				GoName:  fieldGoName(pt, child.Name),
				Path:    t.Name + "." + child.Name,
				Repeats: child.Cardinality.Repeats(),
			})
		}
		if check, ok := bindingCheck(t.Name, pt, child, bindings); ok {
			vd.Bindings = append(vd.Bindings, check)
		}
	}

	helpers := planBackboneHelpers(pt)
	vd.Primitives = primitiveChecks(t.Name+".", pt.Fields, pt.Choices)
	vd.RequiredCalls = backboneCalls(pt.Fields, helpers, func(h *BackboneHelper) bool { return h.EmitRequired })
	vd.LexicalCalls = backboneCalls(pt.Fields, helpers, func(h *BackboneHelper) bool { return h.EmitLexical })
	for _, bb := range pt.Backbones {
		h := helpers[bb.GoName]
		if h.EmitRequired || h.EmitLexical {
			vd.Helpers = append(vd.Helpers, *h)
		}
	}
	return vd, true
}

// planBackboneHelpers builds the walk helper for every backbone type of a planned
// resource and decides transitively which helpers the emitter renders: a backbone emits
// a required helper when it (or any backbone reachable from it) has a required child,
// and a lexical helper when it (or any reachable backbone) has a date-family field. The
// reachability pass iterates to a fixpoint because contentReference recursion makes the
// backbone graph cyclic (a step that contains its own process); the runtime walk stays
// finite because the data is.
func planBackboneHelpers(pt PlannedType) map[string]*BackboneHelper {
	helpers := make(map[string]*BackboneHelper, len(pt.Backbones))
	for _, bb := range pt.Backbones {
		h := &BackboneHelper{
			GoName:     bb.GoName,
			Required:   backboneRequired(bb),
			Primitives: primitiveChecks(".", bb.Fields, bb.Choices),
		}
		h.EmitRequired = len(h.Required) > 0
		h.EmitLexical = len(h.Primitives) > 0
		helpers[bb.GoName] = h
	}

	// Propagate reachability to a fixpoint before pruning the calls, so a helper that
	// only forwards to a deeper violating backbone is still emitted and called.
	for changed := true; changed; {
		changed = false
		for _, bb := range pt.Backbones {
			h := helpers[bb.GoName]
			for _, call := range allBackboneCalls(bb.Fields, helpers) {
				child := helpers[call.TypeGoName]
				if child.EmitRequired && !h.EmitRequired {
					h.EmitRequired = true
					changed = true
				}
				if child.EmitLexical && !h.EmitLexical {
					h.EmitLexical = true
					changed = true
				}
			}
		}
	}

	for _, bb := range pt.Backbones {
		h := helpers[bb.GoName]
		h.RequiredCalls = backboneCalls(bb.Fields, helpers, func(c *BackboneHelper) bool { return c.EmitRequired })
		h.LexicalCalls = backboneCalls(bb.Fields, helpers, func(c *BackboneHelper) bool { return c.EmitLexical })
	}
	return helpers
}

// backboneRequired collects a backbone's own required fields (min >= 1, non-choice),
// with "."-prefixed relative paths the helper appends to its runtime path parameter.
// Presence follows the same pointer/slice rule as the top level (Codex FHIR-007).
func backboneRequired(bb PlannedBackbone) []RequiredField {
	var required []RequiredField
	for _, f := range bb.Fields {
		if f.IsPrimitiveSibling() || f.Element == nil || f.Element.IsChoice {
			continue
		}
		if f.Element.Cardinality.Required() {
			required = append(required, RequiredField{
				GoName:  f.GoName,
				Path:    "." + f.JSONName,
				Repeats: f.Repeats,
			})
		}
	}
	return required
}

// primitiveChecks collects the date/time-family lexical checks of one field scope: the
// plain date-family fields plus the boxed date-family branches of its choice groups.
// pathPrefix is the absolute "Type." prefix at the top level or "." inside a helper.
func primitiveChecks(pathPrefix string, fields []Field, choices []PlannedChoice) []PrimitiveCheck {
	var checks []PrimitiveCheck
	for _, f := range fields {
		if f.IsPrimitiveSibling() || f.Element == nil || f.Element.IsChoice {
			continue
		}
		if len(f.Element.Types) != 1 {
			continue
		}
		code := f.Element.Types[0].Code
		validator, ok := primitiveLexicalKinds[code]
		if !ok {
			continue
		}
		checks = append(checks, PrimitiveCheck{
			GoName:    f.GoName,
			Path:      pathPrefix + f.JSONName,
			Validator: validator,
			TypeName:  code,
			Repeats:   f.Repeats,
		})
	}
	for _, pc := range choices {
		for _, b := range pc.Branches {
			code, ok := primitiveWrapperLexicalCodes[b.GoType]
			if !ok {
				continue
			}
			checks = append(checks, PrimitiveCheck{
				GoName:    b.Field,
				Path:      pathPrefix + b.JSONName,
				Validator: primitiveLexicalKinds[code],
				TypeName:  code,
				Boxed:     true,
			})
		}
	}
	return checks
}

// backboneCalls collects the walk calls of one field scope into the backbone types
// whose helper satisfies emit (has the relevant transitive content), in field order so
// the emitted walk is byte-stable.
func backboneCalls(fields []Field, helpers map[string]*BackboneHelper, emit func(*BackboneHelper) bool) []BackboneCall {
	var calls []BackboneCall
	for _, f := range fields {
		if f.IsPrimitiveSibling() {
			continue
		}
		typeName := strings.TrimPrefix(strings.TrimPrefix(f.GoType, "[]"), "*")
		h, ok := helpers[typeName]
		if !ok || !emit(h) {
			continue
		}
		calls = append(calls, BackboneCall{
			FieldGoName: f.GoName,
			JSONName:    f.JSONName,
			TypeGoName:  typeName,
			Repeats:     f.Repeats,
		})
	}
	return calls
}

// allBackboneCalls collects every backbone-typed field of a scope regardless of emit
// flags, for the reachability fixpoint.
func allBackboneCalls(fields []Field, helpers map[string]*BackboneHelper) []BackboneCall {
	return backboneCalls(fields, helpers, func(*BackboneHelper) bool { return true })
}

// summaryFlags builds the summary metadata for one top-level element. A non-choice
// element yields one SummaryFlag keyed by its wire name; a choice ([x]) element yields
// one flag per suffixed branch wire key, all sharing the group's flags, so MarshalSummary
// filters whichever branch a resource set by the same rule. The narrative element ("text")
// is marked IsText and the Bundle entry-count element ("total") is marked IsCount, the two
// elements the text/data and count modes pivot on.
func summaryFlags(child *model.Element, byBase map[string]PlannedChoice) []SummaryFlag {
	base := SummaryFlag{
		IsSummary:   child.IsSummary,
		IsMandatory: child.Cardinality.Required(),
		IsModifier:  child.IsModifier,
		IsText:      child.Name == "text",
		IsCount:     child.Name == "total",
	}
	if !child.IsChoice {
		base.JSONName = child.Name
		return []SummaryFlag{base}
	}
	pc, ok := byBase[child.ChoiceBase]
	if !ok {
		return nil
	}
	flags := make([]SummaryFlag, 0, len(pc.Branches))
	for _, b := range pc.Branches {
		flag := base
		flag.JSONName = b.JSONName
		flags = append(flags, flag)
	}
	return flags
}

// indexChoicesByBase maps a planned type's choice groups by their FHIR base name so the
// descriptor planner can pair a choice element from the model with the suffixed storage
// fields the planner already decided.
func indexChoicesByBase(choices []PlannedChoice) map[string]PlannedChoice {
	byBase := make(map[string]PlannedChoice, len(choices))
	for _, c := range choices {
		byBase[c.Base] = c
	}
	return byBase
}

// choiceCheck builds the mutual-exclusion check for one choice element, reading the
// suffixed storage fields from the planner's PlannedChoice for the same base. A choice
// with a single branch cannot violate mutual exclusion, so it yields no check.
func choiceCheck(ownerFHIR string, child *model.Element, byBase map[string]PlannedChoice) (ChoiceCheck, bool) {
	pc, ok := byBase[child.ChoiceBase]
	if !ok || len(pc.Branches) < 2 {
		return ChoiceCheck{}, false
	}
	fields := make([]string, 0, len(pc.Branches))
	for _, b := range pc.Branches {
		fields = append(fields, b.Field)
	}
	return ChoiceCheck{Fields: fields, Path: ownerFHIR + "." + child.ChoiceBase + "[x]"}, true
}

// bindingCheck builds the required-binding validity check for one element when it is a
// code field with an enumerable required binding (the same condition that types the
// field as a closed enum). A field whose binding is not inlineable keeps its plain string
// type and gets no check, matching the generated field type.
func bindingCheck(ownerFHIR string, pt PlannedType, child *model.Element, bindings BindingResolver) (BindingCheck, bool) {
	enumName, ok := bindingEnumName(child, bindings)
	if !ok {
		return BindingCheck{}, false
	}
	return BindingCheck{
		GoName:    fieldGoName(pt, child.Name),
		Validator: "valid" + enumName,
		EnumName:  enumName,
		Path:      ownerFHIR + "." + child.Name,
		Repeats:   child.Cardinality.Repeats(),
	}, true
}

// fieldGoName returns the Go field name the planner assigned to a FHIR element name on a
// type, so the descriptor closures reference the same field the struct declares
// (including any collision-resolved suffix). It matches by the field's JSON name, which
// is the FHIR element name for a non-choice field.
func fieldGoName(pt PlannedType, fhirName string) string {
	for _, f := range pt.Fields {
		if f.IsPrimitiveSibling() {
			continue
		}
		if f.JSONName == fhirName {
			return f.GoName
		}
	}
	// Fall back to the deterministic name mapping; this is reached only if the field
	// set and the model element list ever diverge, which the golden tests would catch.
	return GoFieldName(fhirName)
}
