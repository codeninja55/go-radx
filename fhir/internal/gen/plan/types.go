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

	// Element is the source IR node, kept so a later stage can revisit metadata
	// without re-walking the tree.
	Element *model.Element
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
// name for a complex/resource element. A choice element ("[x]") is handled by a later
// stage; PlanField plans its base name and leaves the decorated type to that stage by
// using the first branch's type, which the choice stage overrides.
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
		GoName:   GoFieldName(e.Name),
		GoType:   goType,
		JSONName: jsonName,
		Optional: true,
		Doc:      "",
		Element:  e,
	}
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
func goBaseType(e *model.Element) string {
	if len(e.Types) == 0 {
		return GoFieldName(e.Name)
	}
	code := e.Types[0].Code
	if goPrim, ok := primitiveGoTypes[code]; ok {
		return goPrim
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
