package plan

import (
	"fmt"
	"strings"
)

// Snapshot renders a PlannedType as a stable, human-readable text block for
// golden-file pinning. The format pins exactly the decisions the planner makes — the
// Go type name and kind, each field's Go name, Go type, JSON key and optionality, and
// each distinct backbone struct with its fields — so a golden mismatch points at the
// precise decision that drifted. The rendering is deterministic (fields in canonical
// order, backbones sorted by name) so a regenerated snapshot diffs cleanly.
func Snapshot(pt PlannedType) string {
	var b strings.Builder
	fmt.Fprintf(&b, "PlannedType %s fhir=%s kind=%s\n", pt.GoName, pt.FHIRName, pt.Kind)
	writeFields(&b, pt.Fields, 1)
	for _, bb := range pt.Backbones {
		fmt.Fprintf(&b, "  backbone %s\n", bb.GoName)
		writeFields(&b, bb.Fields, 2)
	}
	return b.String()
}

// writeFields renders a field list indented by depth, one field per line. A
// primitive value field is tagged "primitive" and a generated "_field" sibling
// records the value it pairs with, so the golden pins the primitive-extension
// decisions (which fields are primitives, which carry siblings) the same way it
// pins the pointer/slice choice.
func writeFields(b *strings.Builder, fields []Field, depth int) {
	indent := strings.Repeat("  ", depth)
	for _, f := range fields {
		fmt.Fprintf(b, "%sfield %s %s json=%s optional=%t", indent, f.GoName, f.GoType, f.JSONName, f.Optional)
		switch {
		case f.IsPrimitiveSibling():
			fmt.Fprintf(b, " sibling-of=%s", f.SiblingOf)
		case f.Primitive:
			b.WriteString(" primitive")
		}
		b.WriteByte('\n')
	}
}
