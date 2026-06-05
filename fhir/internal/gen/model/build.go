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

	root, idx, err := buildTree(sd)
	if err != nil {
		return nil, err
	}
	if err := resolveContentReferences(sd.Name, root, idx, map[string]bool{}); err != nil {
		return nil, err
	}
	t.Root = root
	return t, nil
}

// byPath is an index of the direct-child structure of every snapshot element,
// keyed by full path. It records each element's own immediate children only, taken
// before any contentReference graft, so it is a frozen donor view: graft resolution
// reads child lists from here rather than from the result tree's nodes, which the
// graft mutates. This keeps resolution order-independent — a donor's children are
// always its pristine direct children, never a partially-grafted version.
type byPath struct {
	nodes    map[string]*Element // path -> the element node (direct children only)
	children map[string][]string // path -> ordered direct-child paths
}

// buildTree nests the snapshot elements into a tree keyed by full path. The first
// snapshot element is the root (its path is the type name); every subsequent
// element is attached under the node at its parent path. It returns the root and a
// by-path index used by contentReference resolution. A child whose parent path is
// absent is a hard error: silently dropping it is exactly how the prototype
// produced empty backbones.
func buildTree(sd *loader.StructureDefinition) (*Element, *byPath, error) {
	elements := sd.Snapshot.Element
	root := newElement(&elements[0])
	idx := &byPath{
		nodes:    map[string]*Element{root.Path: root},
		children: map[string][]string{},
	}

	for i := 1; i < len(elements); i++ {
		ed := &elements[i]
		node := newElement(ed)

		parentPath := parentOf(ed.Path)
		if parentPath == "" {
			return nil, nil, fmt.Errorf("model: %s: element %q has no parent segment", sd.Name, ed.Path)
		}
		parent, ok := idx.nodes[parentPath]
		if !ok {
			return nil, nil, fmt.Errorf(
				"model: %s: element %q references missing parent %q (snapshot out of order or incomplete)",
				sd.Name, ed.Path, parentPath)
		}
		parent.Children = append(parent.Children, node)
		idx.children[parentPath] = append(idx.children[parentPath], node.Path)

		// Index by full path. A choice element is also indexed under its stripped
		// base so a contentReference or a child path that targets the base resolves;
		// FHIR never nests under a "[x]" element, so the two never collide.
		idx.nodes[node.Path] = node
		if node.IsChoice {
			idx.nodes[strings.TrimSuffix(node.Path, choiceSuffix)] = node
		}
	}
	return root, idx, nil
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
			Profiles:       rt.Profile,
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
// contentReference, grafts a deep copy of the referenced element's child structure
// onto it. FHIR uses contentReference for recursive or shared backbone shapes (for
// example Observation.component.referenceRange reuses #Observation.referenceRange)
// and does not restate the referenced children inline, so without this graft the
// referencing backbone would be empty.
//
// The graft is bounded against self- and mutually-recursive anchors, which FHIR
// uses for genuinely recursive structures (CodeSystem.concept.concept reuses
// #CodeSystem.concept; Composition.section.section reuses #Composition.section).
// A node whose contentReference anchor is the node itself or one of its ancestors
// is a direct self-recursion; it is left carrying its contentReference marker
// unexpanded, so the planner collapses it to the anchor's named backbone type and
// the recursion becomes one self-referential type (CodeSystemConcept with a
// []CodeSystemConcept child) rather than two mutually-referential types. Grafting
// such an anchor inline would also never terminate. The chain guard additionally
// catches a cross-branch cycle that is not a direct ancestor. A non-recursive
// anchor (Observation.component.referenceRange reusing #Observation.referenceRange,
// where the anchor is a sibling subtree, not an ancestor) is expanded fully.
//
// The donor's children are read from the frozen byPath index (its pristine direct
// children), not from the result tree, so resolution order never affects the graft.
// chain holds the contentReference target paths currently being expanded above
// node, so a cycle is detected in O(1).
func resolveContentReferences(typeName string, node *Element, idx *byPath, chain map[string]bool) error {
	if node.ContentReference != "" && len(node.Children) == 0 {
		if _, ok := idx.nodes[node.ContentReference]; !ok {
			return fmt.Errorf(
				"model: %s: element %q contentReference %q resolves to no element in the snapshot",
				typeName, node.Path, node.ContentReference)
		}
		// An anchor that is the node itself or an ancestor of it is a direct
		// self-recursion: leave the node as the boundary so it collapses onto the
		// anchor's named type rather than expanding into a distinct sibling type.
		if isAncestorPath(node.ContentReference, node.Path) {
			return nil
		}
		// A target already on the active chain is a (cross-branch) recursion cycle;
		// leave the node as the boundary rather than expanding forever.
		if !chain[node.ContentReference] {
			chain[node.ContentReference] = true
			// Graft a deep copy of each donor child, rebased so its path reflects this
			// occurrence (Observation.component.referenceRange.low) rather than the
			// donor path it was defined under (Observation.referenceRange.low). The
			// rebase keeps the IR a true occurrence-path tree.
			for _, childPath := range idx.children[node.ContentReference] {
				node.Children = append(node.Children, idx.cloneRebased(childPath, node.ContentReference, node.Path))
			}
			if err := resolveChildren(typeName, node, idx, chain); err != nil {
				return err
			}
			delete(chain, node.ContentReference)
			return nil
		}
	}
	return resolveChildren(typeName, node, idx, chain)
}

// resolveChildren recurses contentReference resolution into a node's children.
func resolveChildren(typeName string, node *Element, idx *byPath, chain map[string]bool) error {
	for _, child := range node.Children {
		if err := resolveContentReferences(typeName, child, idx, chain); err != nil {
			return err
		}
	}
	return nil
}

// cloneRebased deep-copies the donor element at donorPath (and its frozen subtree
// from the byPath index) into an independent Element, rewriting each node's path so
// the donor prefix becomes the occurrence prefix. Reading the subtree from the
// frozen index rather than from the donor node's live Children makes the clone
// independent of any graft already applied to the result tree.
func (idx *byPath) cloneRebased(donorPath, donorPrefix, occurrencePrefix string) *Element {
	src := idx.nodes[donorPath]
	c := *src
	c.Path = rebasePath(src.Path, donorPrefix, occurrencePrefix)
	c.Children = nil
	if src.Types != nil {
		c.Types = make([]TypeRef, len(src.Types))
		for i, t := range src.Types {
			c.Types[i] = t
			if t.Profiles != nil {
				c.Types[i].Profiles = append([]string(nil), t.Profiles...)
			}
			if t.TargetProfiles != nil {
				c.Types[i].TargetProfiles = append([]string(nil), t.TargetProfiles...)
			}
		}
	}
	if src.Binding != nil {
		b := *src.Binding
		c.Binding = &b
	}
	for _, childPath := range idx.children[donorPath] {
		c.Children = append(c.Children, idx.cloneRebased(childPath, donorPrefix, occurrencePrefix))
	}
	return &c
}

// rebasePath replaces the donor prefix of a grafted child's path with the
// occurrence prefix, so "Observation.referenceRange.low" grafted under
// "Observation.component.referenceRange" becomes
// "Observation.component.referenceRange.low". A path that does not start with the
// donor prefix is returned unchanged (defensive; the grafted children always do).
func rebasePath(path, donorPrefix, occurrencePrefix string) string {
	if path == donorPrefix {
		return occurrencePrefix
	}
	if strings.HasPrefix(path, donorPrefix+".") {
		return occurrencePrefix + path[len(donorPrefix):]
	}
	return path
}

// isAncestorPath reports whether anchor is path itself or a dotted-path ancestor of
// path, so a contentReference back to the node's own subtree root is recognised as a
// direct self-recursion. "CodeSystem.concept" is an ancestor of
// "CodeSystem.concept.concept"; "Observation.referenceRange" is not an ancestor of
// "Observation.component.referenceRange" (it is a sibling subtree), so the latter is
// grafted rather than treated as recursive.
func isAncestorPath(anchor, path string) bool {
	return anchor == path || strings.HasPrefix(path, anchor+".")
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
