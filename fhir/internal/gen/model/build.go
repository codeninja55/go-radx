package model

import (
	"fmt"
	"strings"

	"github.com/codeninja55/go-radx/fhir/internal/gen/loader"
)

// BuildType turns one StructureDefinition into a classified Type with its
// element-path tree fully built and recursed. It nests the snapshot's flat,
// dotted-path elements into a tree (so a backbone element carries its real
// children) and resolves each contentReference by grafting the referenced
// element's child structure onto the referencing node (so a node reusing another
// element's shape is populated, not left empty). The result is release-agnostic:
// it reflects only what the snapshot describes.
//
// BuildType fails closed: a snapshot whose element references a parent path that
// the snapshot never defines, or a contentReference whose anchor is absent, is a
// hard error rather than a silently dropped element or an empty backbone.
func BuildType(sd *loader.StructureDefinition) (*Type, error) {
	if sd == nil {
		return nil, fmt.Errorf("model: nil StructureDefinition")
	}
	if sd.Snapshot == nil || len(sd.Snapshot.Element) == 0 {
		return nil, fmt.Errorf("model: %s: StructureDefinition has no snapshot", sd.Name)
	}

	t := &Type{
		Name:     sd.Name,
		URL:      sd.URL,
		Kind:     Classify(sd),
		Abstract: sd.Abstract,
		Base:     sd.BaseDefinition,
	}

	root, byPath, err := buildTree(sd)
	if err != nil {
		return nil, err
	}
	if err := resolveContentReferences(sd.Name, root, byPath); err != nil {
		return nil, err
	}
	t.Root = root
	return t, nil
}

// buildTree nests the snapshot elements into a tree keyed by full path. The first
// snapshot element is the root (its path is the type name); every subsequent
// element is attached under the node at its parent path. It returns the root and a
// by-path index used by contentReference resolution. A child whose parent path is
// absent is a hard error: silently dropping it is exactly how the prototype
// produced empty backbones.
func buildTree(sd *loader.StructureDefinition) (*Element, map[string]*Element, error) {
	elements := sd.Snapshot.Element
	root := newElement(&elements[0])
	byPath := map[string]*Element{root.Path: root}

	for i := 1; i < len(elements); i++ {
		ed := &elements[i]
		node := newElement(ed)

		parentPath := parentOf(ed.Path)
		if parentPath == "" {
			return nil, nil, fmt.Errorf("model: %s: element %q has no parent segment", sd.Name, ed.Path)
		}
		parent, ok := byPath[parentPath]
		if !ok {
			return nil, nil, fmt.Errorf(
				"model: %s: element %q references missing parent %q (snapshot out of order or incomplete)",
				sd.Name, ed.Path, parentPath)
		}
		parent.Children = append(parent.Children, node)

		// Index by full path. A choice element is also indexed under its stripped
		// base so a contentReference or a child path that targets the base resolves;
		// FHIR never nests under a "[x]" element, so the two never collide.
		byPath[node.Path] = node
		if node.IsChoice {
			byPath[strings.TrimSuffix(node.Path, choiceSuffix)] = node
		}
	}
	return root, byPath, nil
}

// newElement maps a raw ElementDefinition to a tree Element, carrying cardinality,
// the type set, binding, and the summary/modifier flags, and detecting the
// polymorphic "[x]" shape. It normalises a FHIRPath System type code to its FHIR
// primitive name so the planner sees uniform type codes.
func newElement(ed *loader.ElementDefinition) *Element {
	e := &Element{
		Name:             lastSegment(ed.Path),
		Path:             ed.Path,
		Cardinality:      Cardinality{Min: ed.Min, Max: ed.Max},
		IsSummary:        ed.IsSummary,
		IsModifier:       ed.IsModifier,
		ContentReference: strings.TrimPrefix(ed.ContentReference, "#"),
	}

	for _, rt := range ed.Type {
		code := rt.Code
		if prim, ok := SystemPrimitive(code); ok {
			code = prim
		}
		e.Types = append(e.Types, TypeRef{
			Code:           code,
			TargetProfiles: rt.TargetProfile,
		})
	}

	if ed.Binding != nil {
		e.Binding = &Binding{
			Strength: ed.Binding.Strength,
			ValueSet: stripVersion(ed.Binding.ValueSet),
		}
	}

	if isChoicePath(ed.Path) {
		e.IsChoice = true
		e.ChoiceBase = stripChoiceSuffix(e.Name)
	}
	return e
}

// resolveContentReferences walks the tree and, for every element carrying a
// contentReference, deep-copies the referenced element's children onto it. FHIR
// uses contentReference for recursive or shared backbone shapes (for example
// Observation.component.referenceRange reuses #Observation.referenceRange) and
// does not restate the referenced children inline, so without this graft the
// referencing backbone would be empty. The copy is deep so each occurrence owns
// its subtree and a later stage editing one does not perturb another.
func resolveContentReferences(typeName string, node *Element, byPath map[string]*Element) error {
	if node.ContentReference != "" && len(node.Children) == 0 {
		target, ok := byPath[node.ContentReference]
		if !ok {
			return fmt.Errorf(
				"model: %s: element %q contentReference %q resolves to no element in the snapshot",
				typeName, node.Path, node.ContentReference)
		}
		for _, child := range target.Children {
			node.Children = append(node.Children, child.clone())
		}
	}
	for _, child := range node.Children {
		if err := resolveContentReferences(typeName, child, byPath); err != nil {
			return err
		}
	}
	return nil
}

// clone deep-copies an element subtree so a grafted contentReference target is
// independent of its source. Slices are copied rather than aliased so the two
// occurrences never share backing arrays.
func (e *Element) clone() *Element {
	c := *e
	if e.Types != nil {
		c.Types = make([]TypeRef, len(e.Types))
		for i, t := range e.Types {
			c.Types[i] = t
			if t.TargetProfiles != nil {
				c.Types[i].TargetProfiles = append([]string(nil), t.TargetProfiles...)
			}
		}
	}
	if e.Binding != nil {
		b := *e.Binding
		c.Binding = &b
	}
	if e.Children != nil {
		c.Children = make([]*Element, len(e.Children))
		for i, child := range e.Children {
			c.Children[i] = child.clone()
		}
	}
	return &c
}

// parentOf returns the dotted path one level up from path ("Observation.component"
// from "Observation.component.code"), or empty for a single-segment root path.
func parentOf(path string) string {
	i := strings.LastIndexByte(path, '.')
	if i < 0 {
		return ""
	}
	return path[:i]
}

// lastSegment returns the final dotted segment of a path ("code" from
// "Observation.component.code"); the whole string for a single-segment path.
func lastSegment(path string) string {
	i := strings.LastIndexByte(path, '.')
	if i < 0 {
		return path
	}
	return path[i+1:]
}

// stripVersion removes a "|<version>" canonical-URL version suffix from a value-set
// reference, leaving the bare canonical URL the loader's index is keyed by.
func stripVersion(url string) string {
	if i := strings.IndexByte(url, '|'); i >= 0 {
		return url[:i]
	}
	return url
}
