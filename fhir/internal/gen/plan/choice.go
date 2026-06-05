package plan

import (
	"sort"
	"unicode"

	"github.com/codeninja55/go-radx/fhir/internal/gen/model"
)

// primitiveWrapperTypes maps a FHIR primitive type code to the release primitive
// wrapper type a choice branch boxes its value in. A choice branch cannot store a
// built-in Go scalar (string, bool, int32) directly: the sealed value interface is
// closed by an unexported marker method, and a built-in cannot carry one. Each
// primitive code therefore boxes through a distinct named wrapper (FHIRString,
// FHIRBoolean, ...) so the branch satisfies the interface and a Value() type switch
// can recover which suffixed branch was set. The wrapper name is "FHIR" + the
// type code lifted to an exported identifier; two primitive codes that share a Go
// scalar (string/code/uri) still get distinct wrappers because their choice suffix
// (valueString/valueCode/valueUri) and storage field differ.
var primitiveWrapperTypes = map[string]string{
	"boolean":      "FHIRBoolean",
	"integer":      "FHIRInteger",
	"positiveInt":  "FHIRPositiveInt",
	"unsignedInt":  "FHIRUnsignedInt",
	"integer64":    "FHIRInteger64",
	"decimal":      "FHIRDecimal",
	"string":       "FHIRString",
	"code":         "FHIRCode",
	"id":           "FHIRID",
	"markdown":     "FHIRMarkdown",
	"uri":          "FHIRURI",
	"url":          "FHIRURL",
	"canonical":    "FHIRCanonical",
	"oid":          "FHIROID",
	"uuid":         "FHIRUUID",
	"base64Binary": "FHIRBase64Binary",
	"instant":      "FHIRInstant",
	"dateTime":     "FHIRDateTime",
	"date":         "FHIRDate",
	"time":         "FHIRTime",
	"xhtml":        "FHIRXHTML",
}

// PlannedChoice is the emitter-ready plan for one polymorphic "[x]" element (for
// example Observation.value[x] or Patient.deceased[x]). The emitter renders it as a
// sealed value interface, one suffixed pointer storage field per branch, a Value()
// getter, and one SetXxxYyy setter per branch that clears the sibling branches so at
// most one is ever populated (FHIR-001). The two-branches-set state is unrepresentable
// through the generated API, and because every storage field is omitempty exactly one
// suffixed key is ever authored on the wire (FHIR-002).
type PlannedChoice struct {
	// Interface is the Go name of the sealed value interface for the group
	// ("ObservationValue"). It is the owning type's Go name plus the choice base
	// lifted to an exported identifier.
	Interface string

	// Marker is the unexported marker method that seals the interface
	// ("isObservationValue"). Only this package's branch types implement it, so the
	// interface is closed and a built-in scalar can never satisfy it.
	Marker string

	// Getter is the Go name of the value accessor ("Value" for value[x], "Deceased"
	// for deceased[x]). It returns (Interface, bool): the set branch, or (nil, false).
	Getter string

	// Base is the FHIR choice base name with the "[x]" suffix removed ("value"),
	// kept for the godoc summary and traceability.
	Base string

	// Branches are the choice's allowed types, in StructureDefinition order, each a
	// distinct suffixed storage field and setter.
	Branches []ChoiceBranch
}

// ChoiceBranch is one allowed type of a choice group: its suffixed wire key, the Go
// storage field, the setter, and the Go value type the setter accepts. A complex
// branch stores a pointer to the generated datatype struct; a primitive branch stores
// a pointer to the release primitive wrapper that carries the interface marker.
type ChoiceBranch struct {
	// Field is the suffixed Go storage field name ("ValueQuantity", "ValueString").
	Field string

	// JSONName is the suffixed wire key ("valueQuantity", "valueString"), the one
	// key authored when this branch is set.
	JSONName string

	// Setter is the Go method name that sets this branch and clears the siblings
	// ("SetValueQuantity").
	Setter string

	// GoType is the branch's underlying Go type, a generated datatype struct for a
	// complex branch ("Quantity") or a release primitive wrapper for a primitive
	// branch ("FHIRString"). The storage field is a pointer to this type; the setter
	// accepts it by value and stores its address.
	GoType string

	// IsPrimitive reports whether the branch boxes a primitive through a wrapper
	// type, kept for the godoc summary that explains the boxing.
	IsPrimitive bool
}

// planChoice turns a choice "[x]" element into a PlannedChoice. The interface and
// marker names are derived from the owning type's Go name and the choice base; the
// getter is the choice's Go field stem (so a "[x]" disambiguated to "Value2" gets a
// Value2() getter, never colliding with a sibling). One branch is planned per allowed
// type: the suffixed wire key is the choice base plus the type code lifted to an
// exported identifier, and the storage field and setter are formed from the stem plus
// that same suffix. used is the owning scope's name set; the storage field names are
// reserved in it so a branch field never collides with another element's field. The
// getter and setter method names are not reserved here: Go methods and fields share a
// type's namespace, and reserving the suffixed storage fields plus the bare stem (done
// by the caller) already keeps the getter ("Value") and setters ("SetValueQuantity")
// clear of every field.
func planChoice(ownerGoName, choiceStem string, e *model.Element, used map[string]bool) PlannedChoice {
	iface := ownerGoName + GoFieldName(e.ChoiceBase)
	pc := PlannedChoice{
		Interface: iface,
		Marker:    "is" + iface,
		Getter:    choiceStem,
		Base:      e.ChoiceBase,
	}

	for _, t := range e.Types {
		// The Go identifier suffix lifts the type code to an exported Go identifier
		// (applying initialisms, so "id" -> "ID"), which names the storage field and
		// setter. The wire suffix uses FHIR's own casing — the type code with only its
		// first letter upper-cased ("id" -> "Id", "uri" -> "Uri") — because the FHIR
		// choice property key is "<base><Type>" with FHIR casing, never the Go
		// initialism casing. Conflating the two would author a non-conformant key such
		// as "valueID" instead of "valueId".
		goSuffix := GoTypeName(t.Code)
		goType := GoTypeName(t.Code)
		isPrim := false
		if wrapper, ok := primitiveWrapperTypes[t.Code]; ok {
			goType = wrapper
			isPrim = true
		}
		field := resolveCollision(choiceStem+goSuffix, used)
		pc.Branches = append(pc.Branches, ChoiceBranch{
			Field:       field,
			JSONName:    e.ChoiceBase + fhirTypeSuffix(t.Code),
			Setter:      "Set" + choiceStem + goSuffix,
			GoType:      goType,
			IsPrimitive: isPrim,
		})
	}
	return pc
}

// fhirTypeSuffix renders a FHIR type code as the choice-property suffix FHIR uses: the
// code with only its first letter upper-cased, leaving the rest verbatim ("dateTime" ->
// "DateTime", "uri" -> "Uri", "CodeableConcept" -> "CodeableConcept"). This is FHIR's
// own casing for a "[x]" property name and is deliberately not the Go-identifier casing
// (which upper-cases whole initialisms, turning "uri" into "URI"), so the generated
// wire key stays standard-conformant.
func fhirTypeSuffix(code string) string {
	if code == "" {
		return ""
	}
	r := []rune(code)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// PrimitiveWrapper describes one generated release primitive wrapper type: its Go
// name, the underlying Go type it boxes, and how its JSON round-trip is rendered. A
// choice branch stores a pointer to a wrapper so the value carries the sealed-interface
// marker a built-in scalar cannot. The wrapper marshals as the bare FHIR primitive
// value (a JSON string, number, or boolean — never a wrapping object), so a boxed
// branch authors exactly the suffixed key with the plain value.
type PrimitiveWrapper struct {
	// GoName is the wrapper type identifier ("FHIRString", "FHIRDecimal").
	GoName string

	// Underlying is the Go type the wrapper is defined over ("string", "bool",
	// "int32", "int64", "fhir.Decimal"). A scalar underlying marshals natively as the
	// bare FHIR value; the Decimal underlying delegates to fhir.Decimal so the lexical
	// form survives the round trip.
	Underlying string

	// Kind selects the wrapper's marshalling: a plain scalar (native JSON) or the
	// lexical-preserving decimal that delegates to fhir.Decimal.
	Kind WrapperKind
}

// WrapperKind selects how a primitive wrapper renders its JSON round-trip.
type WrapperKind int

const (
	// WrapperScalar is a defined type over a Go scalar (string/bool/int32/int64); Go's
	// encoding/json already marshals it as the bare FHIR value, so no custom method is
	// generated.
	WrapperScalar WrapperKind = iota

	// WrapperDecimal is the decimal wrapper, a defined type over fhir.Decimal whose
	// generated MarshalJSON/UnmarshalJSON delegate to the embedded lexical type so
	// trailing zeros and precision survive (FHIR-009).
	WrapperDecimal
)

// PrimitiveWrappers returns the release primitive wrapper descriptors in stable
// (sorted by Go name) order, one per FHIR primitive code the choice planner boxes
// through. Generating from this single authority keeps the emitted primitives.go
// byte-stable and keeps the wrapper set exactly the set the choice machinery needs.
func PrimitiveWrappers() []PrimitiveWrapper {
	wrappers := make([]PrimitiveWrapper, 0, len(primitiveWrapperTypes))
	for code, goName := range primitiveWrapperTypes {
		w := PrimitiveWrapper{GoName: goName, Underlying: primitiveGoTypes[code], Kind: WrapperScalar}
		if code == "decimal" {
			w.Underlying = "fhir.Decimal"
			w.Kind = WrapperDecimal
		}
		wrappers = append(wrappers, w)
	}
	sort.Slice(wrappers, func(i, j int) bool { return wrappers[i].GoName < wrappers[j].GoName })
	return wrappers
}
