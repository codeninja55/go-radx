package model

import (
	"fmt"
	"strings"
)

// Snapshot renders a Type's element-path tree as a stable, human-readable text
// block for golden-file pinning. The format is one line per element, indented by
// tree depth, carrying the metadata the IR records: cardinality, type set,
// binding, the summary/modifier/choice flags, and any resolved contentReference.
// The rendering is deterministic (children appear in snapshot order, never sorted)
// so a regenerated snapshot diffs cleanly against the committed golden.
func Snapshot(t *Type) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Type %s kind=%s abstract=%t\n", t.Name, t.Kind, t.Abstract)
	if t.Base != "" {
		fmt.Fprintf(&b, "  base %s\n", t.Base)
	}
	writeElement(&b, t.Root, 0)
	return b.String()
}

// writeElement renders one element line and recurses into its children, so the
// snapshot captures the full recursed tree.
func writeElement(b *strings.Builder, e *Element, depth int) {
	b.WriteString(strings.Repeat("  ", depth+1))
	b.WriteString(e.Path)
	fmt.Fprintf(b, " [%d..%s]", e.Cardinality.Min, maxOrStar(e.Cardinality.Max))

	if len(e.Types) > 0 {
		b.WriteString(" types=")
		b.WriteString(typeCodes(e.Types))
	}
	if e.Binding != nil {
		fmt.Fprintf(b, " binding=%s", e.Binding.Strength)
		if e.Binding.ValueSet != "" {
			fmt.Fprintf(b, "(%s)", e.Binding.ValueSet)
		}
	}
	if e.IsChoice {
		fmt.Fprintf(b, " choice=%s", e.ChoiceBase)
	}
	if e.IsSummary {
		b.WriteString(" summary")
	}
	if e.IsModifier {
		b.WriteString(" modifier")
	}
	if e.ContentReference != "" {
		fmt.Fprintf(b, " contentReference=%s", e.ContentReference)
	}
	b.WriteByte('\n')

	for _, child := range e.Children {
		writeElement(b, child, depth+1)
	}
}

// maxOrStar renders an empty max token as "*" so the cardinality column is never
// blank for an unconstrained element written without an explicit max.
func maxOrStar(max string) string {
	if max == "" {
		return "*"
	}
	return max
}

// typeCodes joins an element's type codes, appending the target profiles of a
// reference-bearing type as a parenthesised list so the snapshot pins them.
func typeCodes(types []TypeRef) string {
	parts := make([]string, 0, len(types))
	for _, t := range types {
		part := t.Code
		if len(t.TargetProfiles) > 0 {
			part += "->[" + strings.Join(shortProfiles(t.TargetProfiles), ",") + "]"
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "|")
}

// shortProfiles trims the canonical-URL prefix from reference target profiles so
// the snapshot pins the type name rather than the full URL, keeping it readable
// while staying deterministic.
func shortProfiles(profiles []string) []string {
	out := make([]string, len(profiles))
	for i, p := range profiles {
		if idx := strings.LastIndexByte(p, '/'); idx >= 0 {
			out[i] = p[idx+1:]
		} else {
			out[i] = p
		}
	}
	return out
}
