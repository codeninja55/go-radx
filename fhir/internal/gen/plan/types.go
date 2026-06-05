package plan

import "github.com/codeninja55/go-radx/fhir/internal/gen/model"

// primitiveGoTypes maps a FHIR primitive type code to its Go scalar. The date/time
// family and the string family all become Go string (validated on decode by a later
// stage); the integer family becomes a sized integer; decimal becomes fhir.Decimal
// so lexical precision survives a round trip rather than collapsing to float64. The
// map is the planner's authority for "is this type a primitive?"; a code absent from
// it is treated as a complex/resource type whose Go name is GoTypeName(code).
var primitiveGoTypes = map[string]string{
	"boolean":      "bool",
	"integer":      "int32",
	"positiveInt":  "int32",
	"unsignedInt":  "int32",
	"integer64":    "int64",
	"decimal":      "fhir.Decimal",
	"string":       "string",
	"code":         "string",
	"id":           "string",
	"markdown":     "string",
	"uri":          "string",
	"url":          "string",
	"canonical":    "string",
	"oid":          "string",
	"uuid":         "string",
	"base64Binary": "string",
	"instant":      "string",
	"dateTime":     "string",
	"date":         "string",
	"time":         "string",
	"xhtml":        "string",
}

// Field is one planned struct field: the decided Go identifier, the Go type with its
// pointer/slice decoration already applied, the JSON tag, and the source element. The
// emitter renders a Field verbatim and makes no further type decision.
type Field struct {
	// GoName is the exported Go field identifier.
	GoName string

	// GoType is the fully-decorated Go type as written in the struct
	// ("*string", "[]Coding", "*fhir.Decimal").
	GoType string

	// JSONName is the wire key (the FHIR element name with any "[x]" suffix already
	// removed for a choice group's base; choice rendering is a later stage).
	JSONName string

	// Optional reports whether the field is omitted when empty (the json
	// ",omitempty" option). A repeating field and an optional scalar are omitempty;
	// a required scalar is a pointer and is also omitempty because nil means absent.
	Optional bool

	// Doc is the element's short description, carried for the field's godoc comment.
	Doc string

	// Primitive reports whether the field's value is a FHIR primitive (its Go type
	// is a scalar or fhir.Decimal, not a struct). Only a primitive field carries a
	// "_field" extension sibling (Codex FHIR-005): a complex field such as
	// Patient.name, a contained resource, or an OperationOutcome.issue never does.
	Primitive bool

	// Repeats reports whether the field is a repeating element (a Go slice). It is
	// carried so the emitter can null-align a repeating primitive's "_field" sibling
	// array against its value array.
	Repeats bool

	// SiblingOf names the value field this field is the "_field" extension sibling
	// of, set only on a generated PrimitiveElement sibling field. It is empty on an
	// ordinary value field. The emitter uses it to pair the sibling with its value
	// when generating null-aligned marshalling for a repeating primitive.
	SiblingOf string

	// Element is the source IR node, kept so a later stage can revisit metadata
	// without re-walking the tree.
	Element *model.Element
}

// IsPrimitiveSibling reports whether the field is a generated "_field" extension
// sibling rather than a value field.
func (f Field) IsPrimitiveSibling() bool { return f.SiblingOf != "" }

// resourceInterfaceType is the Go type goBaseType assigns to an element typed as the
// abstract FHIR Resource (Bundle.entry.resource, contained, ...): the root
// fhir.Resource interface, which the standard JSON codec cannot decode a resource
// object into. The emitter routes such a field through fhir.UnmarshalResource instead.
const resourceInterfaceType = "fhir.Resource"

// IsResourceInterface reports whether the field's value is the abstract FHIR Resource
// (Go type fhir.Resource, single or repeating). The standard codec cannot unmarshal a
// resource object into the interface, so the emitter lifts the field's raw JSON out and
// decodes it through fhir.UnmarshalResource (resourceType peek then registry dispatch)
// rather than letting the default struct decode fail.
func (f Field) IsResourceInterface() bool {
	return f.GoType == "*"+resourceInterfaceType || f.GoType == "[]"+resourceInterfaceType
}

// ResourceIsSlice reports whether a resource-interface field repeats (Go type
// []fhir.Resource, such as DomainResource.contained) rather than being a single
// pointer (*fhir.Resource, such as Bundle.entry.resource), so the emitter selects the
// slice decode helper.
func (f Field) ResourceIsSlice() bool { return f.GoType == "[]"+resourceInterfaceType }

// ValueField returns the Go field name of the value this sibling describes, set
// only on a "_field" sibling. The emitter uses it to take len() of the value array
// when null-aligning a repeating primitive's sibling array.
func (f Field) ValueField() string { return f.SiblingOf }

// JSONTag is the struct-tag body the emitter writes for the field. Every "_field"
// extension sibling is excluded from the default codec ("-") and handled by the
// generated MarshalJSON/UnmarshalJSON: a scalar sibling so its key is dropped when
// it carries no id or extension (Go's omitempty cannot drop a non-nil but empty
// pointer), and a repeating sibling so its array is null-aligned with the value
// array. Every value field uses its JSON key with ",omitempty".
func (f Field) JSONTag() string {
	if f.IsPrimitiveSibling() {
		return "-"
	}
	if f.Optional {
		return f.JSONName + ",omitempty"
	}
	return f.JSONName
}

// fieldShape captures the Go field shapes the planner chooses between.
type fieldShape int

const (
	// shapePointer is a single value rendered as a pointer, so absence (nil) is
	// distinguishable from a present zero value.
	shapePointer fieldShape = iota

	// shapeSlice is a repeating value rendered as a Go slice.
	shapeSlice
)

// PlanField decides the Go field for one IR element: its Go name, its Go type with
// pointer/slice decoration, and its JSON key. The cardinality rules are:
//
//   - a repeating element (max "*" or > 1) becomes a slice, never a pointer;
//   - a required scalar (min >= 1, max 1) becomes a *pointer*, not a bare value —
//     this is the structural half of the required-presence fix (FHIR-007): a
//     present required false or 0 must be distinguishable from an absent field, and
//     only a pointer carries that distinction at the struct level;
//   - an optional scalar (min 0, max 1) becomes a pointer.
//
// The Go base type is the primitive Go scalar for a primitive element or the Go type
// name for a complex/resource element. A choice element ("[x]") is not planned through
// PlanField in the pipeline: planFields intercepts a choice and expands it into its
// suffixed storage fields and a PlannedChoice. PlanField left applied to a choice
// element still yields a non-primitive single field (its branches box their own
// primitives), which the FHIR-005 guard test relies on, but the bulk pipeline never
// reaches that path for a "[x]" element.
func PlanField(e *model.Element) Field {
	base := goBaseType(e)

	goType := "*" + base
	if chooseShape(e.Cardinality) == shapeSlice {
		goType = "[]" + base
	}

	jsonName := e.Name
	if e.IsChoice {
		jsonName = e.ChoiceBase
	}

	return Field{
		GoName:    GoFieldName(e.Name),
		GoType:    goType,
		JSONName:  jsonName,
		Optional:  true,
		Primitive: elementHasPrimitiveSibling(e),
		Repeats:   chooseShape(e.Cardinality) == shapeSlice,
		Doc:       "",
		Element:   e,
	}
}

// elementHasPrimitiveSibling reports whether an element is a true primitive that
// carries a "_field" extension sibling. The FHIR rule (Codex FHIR-005) is that only
// a genuine primitive value gets a "_field" companion: a complex field, a backbone,
// a choice element (its branches box their own primitives), and a contentReference
// recursion boundary never do. An element with no declared type (a pure backbone)
// is structural, not primitive. The single-type primitive code drives the
// decision; a "[x]" choice is excluded even when one branch is a primitive, since
// the wire key is the branch-suffixed name, not the choice base.
func elementHasPrimitiveSibling(e *model.Element) bool {
	if e.IsChoice || e.IsBackbone() || e.ContentReference != "" {
		return false
	}
	if len(e.Types) != 1 {
		return false
	}
	return IsPrimitiveCode(e.Types[0].Code)
}

// chooseShape applies the cardinality rules to pick a field shape. A repeating
// element is a slice; every single-valued scalar is a pointer so presence is carried
// by non-nil, honouring the required-presence rule for required scalars and the
// natural "absent is nil" rule for optional ones.
func chooseShape(c model.Cardinality) fieldShape {
	if c.Repeats() {
		return shapeSlice
	}
	return shapePointer
}

// goBaseType returns the undecorated Go type for an element: the primitive Go scalar
// for a primitive-typed element, or the Go type name for a complex/resource-typed
// element. An element with no type (a pure backbone, handled elsewhere) falls back to
// its Go field name, which the backbone stage replaces with the backbone type name.
//
// An element typed as the abstract "Resource" (Bundle.entry.resource,
// Parameters.parameter.resource, DomainResource.contained) maps to the root
// fhir.Resource interface, not to the generated Resource base struct, so the field
// holds any concrete resource rather than the bare base. The release packages
// resolve a concrete type from such a field through fhir.As / fhir.UnmarshalResource.
func goBaseType(e *model.Element) string {
	if len(e.Types) == 0 {
		return GoFieldName(e.Name)
	}
	code := e.Types[0].Code
	if goPrim, ok := primitiveGoTypes[code]; ok {
		return goPrim
	}
	if code == "Resource" || code == "DomainResource" {
		return "fhir.Resource"
	}
	return GoTypeName(code)
}

// IsPrimitiveCode reports whether a FHIR type code maps to a Go scalar primitive,
// exposed so the dedup and emitter stages can reason about primitive-ness without
// re-deriving the table.
func IsPrimitiveCode(code string) bool {
	_, ok := primitiveGoTypes[code]
	return ok
}
