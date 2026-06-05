// This file is HAND-WRITTEN, not generated. Like the Bundle builders, the reference
// resolution and integrity helpers encode FHIR prose rules (how a #id contained
// reference and an intra-Bundle fullUrl/relative reference resolve) that the
// StructureDefinition does not express, so they are hand-written per release on the
// generated types and live outside the generated file set.

package r5

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/codeninja55/go-radx/fhir"
)

// ErrContained is the sentinel wrapped by DomainResource.Contained when a contained
// slot is malformed (a nil or otherwise unaddressable contained resource). Returning
// a wrapped error rather than a silent (nil, false) is the FHIR-011 fix: a malformed
// contained resource is surfaced naming its index, never quietly treated as "not
// found". The wrapped text names the index and the offending resourceType, never
// patient data.
var ErrContained = errors.New("fhir/r5: malformed contained resource")

// Resolve looks up the resource a reference string points at within this Bundle. It
// resolves two local reference forms: an entry whose fullUrl equals ref (the absolute
// or relative intra-Bundle form), and a "#id" fragment against the contained resources
// of the entries. It returns (resource, true) on a hit and (nil, false) for an
// unresolved or external reference; an absolute URL that matches no entry fullUrl is
// simply not found, because Resolve does not dereference the network.
//
// Resolve performs no reflection-heavy walk; it indexes the bundle's own entry
// fullUrls and contained ids, so it is cheap to call repeatedly (CheckReferenceIntegrity
// builds the indexes once and reuses them rather than calling Resolve per reference).
func (b *Bundle) Resolve(ref string) (fhir.Resource, bool) {
	if b == nil || ref == "" {
		return nil, false
	}
	if strings.HasPrefix(ref, "#") {
		return b.resolveFragment(ref)
	}
	for i := range b.Entry {
		entry := &b.Entry[i]
		if entry.FullUrl != nil && *entry.FullUrl == ref && entry.Resource != nil {
			return *entry.Resource, true
		}
	}
	return nil, false
}

// resolveFragment resolves a "#id" contained reference against the contained resources
// of every entry in the bundle. A bare "#" (the self reference to the containing
// resource) is not resolvable to an entry and returns not-found.
func (b *Bundle) resolveFragment(ref string) (fhir.Resource, bool) {
	id := strings.TrimPrefix(ref, "#")
	if id == "" {
		return nil, false
	}
	for i := range b.Entry {
		entry := &b.Entry[i]
		if entry.Resource == nil {
			continue
		}
		if r, err := findContained(*entry.Resource, id); err == nil && r != nil {
			return r, true
		}
	}
	return nil, false
}

// ResolveContained returns the contained resource of this DomainResource whose id
// matches id (the target of a "#id" reference). It is named ResolveContained rather
// than Contained because the generated DomainResource already carries a Contained field
// (the contained slice itself); the method resolves against that field. It returns an
// aggregate error, not a silent miss, when a contained slot is malformed: a nil or
// typed-nil contained resource cannot be addressed by id, and surfacing that as an
// error naming the index is the FHIR-011 fix for the prototype that skipped a malformed
// contained resource and reported the reference as merely "not found". A clean
// not-found (no contained resource carries id, and none is malformed) returns (nil, nil).
func (d *DomainResource) ResolveContained(id string) (fhir.Resource, error) {
	if d == nil {
		return nil, nil
	}
	var malformed []error
	for i := range d.Contained {
		c := d.Contained[i]
		if _, ok := fhir.As[fhir.Resource](c); !ok {
			malformed = append(malformed, fmt.Errorf("%w: contained[%d] is nil", ErrContained, i))
			continue
		}
		cid, ok := resourceID(c)
		if !ok {
			malformed = append(malformed, fmt.Errorf("%w: contained[%d] (%s) has no addressable id",
				ErrContained, i, c.ResourceType()))
			continue
		}
		if cid == id {
			return c, nil
		}
	}
	if len(malformed) > 0 {
		return nil, errors.Join(malformed...)
	}
	return nil, nil
}

// CheckReferenceIntegrity walks every Reference reachable from the resources in this
// Bundle (and their contained resources) and reports each local reference that does
// not resolve within the bundle. A "#id" reference must resolve against a contained
// resource of the bundle, and a relative reference (for example "Patient/p1") or an
// absolute reference that names an entry fullUrl must match an entry; an external
// absolute URL (one that matches no entry fullUrl) is left alone, because the bundle
// makes no claim to resolve the network. Every malformed contained resource discovered
// during the walk is also reported, so a dangling reference and a malformed contained
// slot both become issues rather than silent skips (FHIR-011). The returned
// OperationOutcome aggregates all issues; it is non-nil and HasErrors() is false when
// the bundle is clean.
func (b *Bundle) CheckReferenceIntegrity() *OperationOutcome {
	outcome := &OperationOutcome{}
	if b == nil {
		return outcome
	}

	fullURLs := make(map[string]struct{}, len(b.Entry))
	for i := range b.Entry {
		if b.Entry[i].FullUrl != nil {
			fullURLs[*b.Entry[i].FullUrl] = struct{}{}
		}
	}

	for i := range b.Entry {
		entry := &b.Entry[i]
		if entry.Resource == nil {
			continue
		}
		path := fmt.Sprintf("Bundle.entry[%d].resource", i)
		checkResourceReferences(*entry.Resource, path, fullURLs, outcome)
	}
	return outcome
}

// checkResourceReferences resolves every reference found in one entry resource and
// records an issue for each dangling local reference. Contained references ("#id") are
// resolved against the resource's own contained slice (reporting a malformed contained
// resource), and intra-bundle references against the bundle's fullUrl set.
func checkResourceReferences(r fhir.Resource, path string, fullURLs map[string]struct{}, outcome *OperationOutcome) {
	containedIDs, malformed := containedIndex(r)
	for i := range malformed {
		outcome.Issue = append(outcome.Issue, OperationOutcomeIssue{
			Severity:    severityPtr(IssueSeverityError),
			Code:        issueTypePtr(IssueTypeStructure),
			Diagnostics: strPtr(fmt.Sprintf("%s: %s", path, malformed[i].Error())),
		})
	}

	for _, found := range collectReferences(r, path) {
		if isExternalAbsolute(found.ref) {
			continue
		}
		if strings.HasPrefix(found.ref, "#") {
			id := strings.TrimPrefix(found.ref, "#")
			if _, ok := containedIDs[id]; ok {
				continue
			}
			outcome.Issue = append(outcome.Issue, danglingIssue(found))
			continue
		}
		if _, ok := fullURLs[found.ref]; ok {
			continue
		}
		outcome.Issue = append(outcome.Issue, danglingIssue(found))
	}
}

// foundReference is a reference value and the element path at which it was found, so a
// reported issue names where the dangling reference lives without carrying any value
// beyond the reference string itself (a reference URL is not PHI).
type foundReference struct {
	ref  string
	path string
}

// danglingIssue builds the OperationOutcomeIssue for an unresolved local reference.
func danglingIssue(found foundReference) OperationOutcomeIssue {
	return OperationOutcomeIssue{
		Severity:    severityPtr(IssueSeverityError),
		Code:        issueTypePtr(IssueTypeNotFound),
		Diagnostics: strPtr(fmt.Sprintf("unresolved local reference %q", found.ref)),
		Expression:  []string{found.path},
	}
}

// isExternalAbsolute reports whether ref is an absolute URL to an external system,
// which CheckReferenceIntegrity leaves alone. A "urn:" reference and an "http(s)://"
// or "ftp://"-style scheme are treated as external; a "#id" fragment and a relative
// "Type/id" reference are local and are checked. The check is intentionally simple:
// any reference carrying a scheme (a "scheme:" prefix before the first "/") is external.
func isExternalAbsolute(ref string) bool {
	if strings.HasPrefix(ref, "#") {
		return false
	}
	if strings.HasPrefix(ref, "urn:") {
		return true
	}
	scheme := ref
	if slash := strings.IndexByte(ref, '/'); slash >= 0 {
		scheme = ref[:slash]
	}
	return strings.HasSuffix(scheme, ":") && len(scheme) > 1
}

// HasErrors reports whether the outcome carries at least one issue of error or fatal
// severity. It is nil-safe: a nil *OperationOutcome and an all-information outcome both
// report false, so a caller can write `if oo.HasErrors()` without a nil guard.
func (o *OperationOutcome) HasErrors() bool {
	if o == nil {
		return false
	}
	for i := range o.Issue {
		if o.Issue[i].Severity != nil && fhir.IssueSeverity(*o.Issue[i].Severity).IsError() {
			return true
		}
	}
	return false
}

// Error reports the outcome as a Go error, or nil when it carries no error-severity
// issue, so a caller can fold validation into the standard `if err != nil` flow. The
// message names the count and the diagnostics of the error-severity issues; it never
// includes a patient value, because issue diagnostics are built from element paths and
// codes, not data. A nil *OperationOutcome and an outcome with no error issues both
// return nil.
func (o *OperationOutcome) Error() error {
	if !o.HasErrors() {
		return nil
	}
	var msgs []string
	for i := range o.Issue {
		issue := &o.Issue[i]
		if issue.Severity == nil || !fhir.IssueSeverity(*issue.Severity).IsError() {
			continue
		}
		msg := string(*issue.Severity)
		if issue.Diagnostics != nil {
			msg = msg + ": " + *issue.Diagnostics
		}
		msgs = append(msgs, msg)
	}
	return fmt.Errorf("fhir/r5: operation outcome reported %d error(s): %s",
		len(msgs), strings.Join(msgs, "; "))
}

// containedIndex returns the set of addressable contained resource ids on r and the
// errors for any malformed contained slot, so the reference walk can both resolve a
// "#id" and report a malformed contained resource. r is reflected for its embedded
// DomainResource.Contained; a resource type with no contained slot yields an empty set.
func containedIndex(r fhir.Resource) (map[string]struct{}, []error) {
	contained := containedResources(r)
	ids := make(map[string]struct{}, len(contained))
	var malformed []error
	for i := range contained {
		c := contained[i]
		if _, ok := fhir.As[fhir.Resource](c); !ok {
			malformed = append(malformed, fmt.Errorf("contained[%d] is nil", i))
			continue
		}
		id, ok := resourceID(c)
		if !ok {
			malformed = append(malformed, fmt.Errorf("contained[%d] (%s) has no addressable id", i, c.ResourceType()))
			continue
		}
		ids[id] = struct{}{}
	}
	return ids, malformed
}

// findContained returns the contained resource of r whose id matches id, or an error
// when a contained slot is malformed (so a "#id" resolution against a malformed slot is
// surfaced, not silently skipped). It is the single-resource form of Contained used by
// Bundle fragment resolution.
func findContained(r fhir.Resource, id string) (fhir.Resource, error) {
	contained := containedResources(r)
	var malformed []error
	for i := range contained {
		c := contained[i]
		if _, ok := fhir.As[fhir.Resource](c); !ok {
			malformed = append(malformed, fmt.Errorf("%w: contained[%d] is nil", ErrContained, i))
			continue
		}
		cid, ok := resourceID(c)
		if !ok {
			malformed = append(malformed, fmt.Errorf("%w: contained[%d] has no addressable id", ErrContained, i))
			continue
		}
		if cid == id {
			return c, nil
		}
	}
	if len(malformed) > 0 {
		return nil, errors.Join(malformed...)
	}
	return nil, nil
}

// containedResources returns the []fhir.Resource of a resource's embedded
// DomainResource.Contained via reflection, or nil for a resource type that embeds the
// plain Resource base (no contained slot). The reflection is confined here so the rest
// of the integrity logic works in terms of a flat slice.
func containedResources(r fhir.Resource) []fhir.Resource {
	v := reflect.ValueOf(r)
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	field := v.FieldByName("Contained")
	if !field.IsValid() || field.Kind() != reflect.Slice {
		return nil
	}
	if slice, ok := field.Interface().([]fhir.Resource); ok {
		return slice
	}
	return nil
}

// resourceID returns the value of a resource's id field and whether it is set, so a
// "#id" reference can be matched against a contained resource's id. A resource whose id
// is nil or empty has no addressable id and reports false, which the caller treats as a
// malformed contained slot for the purpose of "#id" resolution.
func resourceID(r fhir.Resource) (string, bool) {
	v := reflect.ValueOf(r)
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return "", false
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return "", false
	}
	field := v.FieldByName("ID")
	if !field.IsValid() || field.Kind() != reflect.Pointer || field.IsNil() {
		return "", false
	}
	if id, ok := field.Elem().Interface().(string); ok && id != "" {
		return id, true
	}
	return "", false
}

// collectReferences walks r with reflection and returns every non-empty
// Reference.reference string it finds, each tagged with the element path at which it
// was found. The walk descends through pointers, slices, and structs and recognises the
// generated Reference type by reading its Reference field; it does not interpret any
// other datatype, so it visits the whole resource graph without a per-type schema.
func collectReferences(r fhir.Resource, path string) []foundReference {
	var found []foundReference
	walkReferences(reflect.ValueOf(r), path, &found, map[uintptr]struct{}{})
	return found
}

// referenceType is the reflect.Type of the generated Reference struct, resolved once so
// the walk can recognise a Reference value without a string type-name comparison.
var referenceType = reflect.TypeOf(Reference{})

// walkReferences recurses through v, appending each Reference.reference it finds. The
// visited set guards against a pointer cycle (a contained resource that points back at
// its container) so the walk terminates; only addressable pointer values are tracked,
// which is sufficient because cycles in the generated model are always through pointers.
func walkReferences(v reflect.Value, path string, found *[]foundReference, visited map[uintptr]struct{}) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return
		}
		if v.Kind() == reflect.Pointer {
			ptr := v.Pointer()
			if _, seen := visited[ptr]; seen {
				return
			}
			visited[ptr] = struct{}{}
		}
		walkReferences(v.Elem(), path, found, visited)
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			walkReferences(v.Index(i), fmt.Sprintf("%s[%d]", path, i), found, visited)
		}
	case reflect.Struct:
		if v.Type() == referenceType {
			if ref := referenceString(v); ref != "" {
				*found = append(*found, foundReference{ref: ref, path: path + ".reference"})
			}
			return
		}
		for i := 0; i < v.NumField(); i++ {
			fieldType := v.Type().Field(i)
			if fieldType.PkgPath != "" {
				continue // unexported field
			}
			walkReferences(v.Field(i), path+"."+fieldType.Name, found, visited)
		}
	default:
	}
}

// referenceString reads the Reference field of a Reference struct value, returning the
// dereferenced string or "" when it is unset.
func referenceString(v reflect.Value) string {
	field := v.FieldByName("Reference")
	if !field.IsValid() || field.Kind() != reflect.Pointer || field.IsNil() {
		return ""
	}
	if s, ok := field.Elem().Interface().(string); ok {
		return s
	}
	return ""
}

// severityPtr and issueTypePtr box the generated binding constants for the issue
// fields, which carry pointers so an unset code is omitted on the wire.
func severityPtr(s IssueSeverity) *IssueSeverity { return &s }
func issueTypePtr(t IssueType) *IssueType        { return &t }
