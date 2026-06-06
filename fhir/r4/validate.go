// This file is HAND-WRITTEN, not generated. The structural validation engine and the
// generated per-resource descriptors (validation_descriptors.go) cover the checks the
// StructureDefinition expresses — required-element presence, choice-group mutual
// exclusion, and required-binding codes. The Bundle bdl-* invariants and intra-Bundle
// reference integrity are FHIR prose rules the StructureDefinition does not encode, the
// same reason the Bundle builders and reference helpers are hand-written per release. The
// Bundle descriptor wires BundleValidateExtra as its extra-check hook so fhir.Validate
// applies these rules to a decoded Bundle (the builders enforce the constructive subset
// up front; Validate is the gate for a Bundle that arrived over the wire).

package r4

import (
	"fmt"
	"reflect"

	"github.com/codeninja55/go-radx/fhir"
)

// BundleValidateExtra runs the Bundle-specific structural checks fhir.Validate cannot
// derive from the StructureDefinition: the bdl-* per-type invariants over a decoded
// Bundle and the intra-Bundle/contained reference-integrity walk. It is registered as
// the Bundle descriptor's Extra hook in validation_descriptors.go and appends each issue
// it finds to the outcome the engine is building. Every issue names a path, a bundle
// type, or a reference string — never a patient value — so the outcome stays PHI-clean.
//
// r is the resource being validated; a non-Bundle (which the descriptor key makes
// impossible in practice) is ignored rather than panicking, honouring the never-panic
// contract.
func BundleValidateExtra(r fhir.Resource, outcome *fhir.OperationOutcome) {
	b, ok := r.(*Bundle)
	if !ok || b == nil {
		return
	}

	checkBundleInvariants(b, outcome)
	composeReferenceIntegrity(b, outcome)
}

// checkBundleInvariants applies the bdl-* per-type invariants to a decoded Bundle: total
// is meaningful only on searchset and history (bdl-1, FHIR-010), search metadata appears
// only on a searchset entry (bdl-2), a document's first entry is a Composition and a
// message's first entry is a MessageHeader (bdl-3), a transaction/batch entry carries a
// request while a response bundle's entry carries a response (bdl-3a/3b request-presence),
// and fullUrl values are unique except across versions or in a history bundle (bdl-7/bdl-8).
// Each violation is an issue naming the entry index and the rule, never a value. A Bundle
// with no type is left to the descriptor's required
// check, which reports the missing Bundle.type once; the type-keyed invariants below
// simply have nothing to check without a type.
func checkBundleInvariants(b *Bundle, outcome *fhir.OperationOutcome) {
	if b.Type == nil {
		return
	}
	bundleType := *b.Type

	if b.Total != nil && bundleType != BundleTypeSearchset && bundleType != BundleTypeHistory {
		fhir.AddIssue(outcome, fhir.SeverityError, fhir.IssueTypeInvalid,
			"Bundle.total",
			fmt.Sprintf("Bundle.total is only allowed on a searchset or history bundle, not %s (bdl-1)", bundleType))
	}

	checkFirstEntryType(b, bundleType, outcome)

	for i := range b.Entry {
		entry := &b.Entry[i]
		path := fmt.Sprintf("Bundle.entry[%d]", i)

		if entry.Search != nil && bundleType != BundleTypeSearchset {
			fhir.AddIssue(outcome, fhir.SeverityError, fhir.IssueTypeInvalid,
				path+".search",
				fmt.Sprintf("entry.search is only allowed in a searchset bundle, not %s (bdl-2)", bundleType))
		}
		checkEntryRequestResponse(entry, path, bundleType, outcome)
	}

	checkUniqueFullURLsOutcome(b, bundleType, outcome)
}

// checkFirstEntryType enforces the document/message first-entry rule (bdl-3): a document
// bundle's first entry must be a Composition and a message bundle's first entry must be a
// MessageHeader. A bundle of the type with no entries, or whose first entry holds a nil or
// typed-nil resource, is reported (the leading resource is mandatory); a first entry of the
// wrong type names the type it should have been. The first resource is recovered through
// fhir.As, which fails closed on a nil interface or a typed-nil pointer, so a malformed
// entry yields an issue rather than a nil-dereference panic (PRD §9.3).
func checkFirstEntryType(b *Bundle, bundleType BundleType, outcome *fhir.OperationOutcome) {
	var want string
	switch bundleType {
	case BundleTypeDocument:
		want = CompositionResourceType
	case BundleTypeMessage:
		want = MessageHeaderResourceType
	default:
		return
	}

	var first fhir.Resource
	if len(b.Entry) > 0 && b.Entry[0].Resource != nil {
		first = *b.Entry[0].Resource
	}
	got, ok := fhir.As[fhir.Resource](first)
	if !ok {
		fhir.AddIssue(outcome, fhir.SeverityError, fhir.IssueTypeInvalid,
			"Bundle.entry[0]",
			fmt.Sprintf("a %s bundle's first entry must be a %s (bdl-3)", bundleType, want))
		return
	}
	if got.ResourceType() != want {
		fhir.AddIssue(outcome, fhir.SeverityError, fhir.IssueTypeInvalid,
			"Bundle.entry[0].resource",
			fmt.Sprintf("a %s bundle's first entry must be a %s, got %s (bdl-3)", bundleType, want, got.ResourceType()))
	}
}

// checkEntryRequestResponse enforces the request/response-presence rules: every entry of a
// transaction or batch must carry a request, and every entry of a transaction-response or
// batch-response must carry a response (bdl-3a/3b). Other bundle types impose no such
// rule, so they are left alone here.
func checkEntryRequestResponse(entry *BundleEntry, path string, bundleType BundleType, outcome *fhir.OperationOutcome) {
	switch bundleType {
	case BundleTypeTransaction, BundleTypeBatch:
		if entry.Request == nil {
			fhir.AddIssue(outcome, fhir.SeverityError, fhir.IssueTypeRequired,
				path+".request",
				fmt.Sprintf("every entry of a %s bundle must carry a request (bdl-3a)", bundleType))
		}
	case BundleTypeTransactionResponse, BundleTypeBatchResponse:
		if entry.Response == nil {
			fhir.AddIssue(outcome, fhir.SeverityError, fhir.IssueTypeRequired,
				path+".response",
				fmt.Sprintf("every entry of a %s bundle must carry a response (bdl-3b)", bundleType))
		}
	}
}

// checkUniqueFullURLsOutcome reports each duplicate fullUrl across the entries (bdl-7),
// honouring the two FHIR exceptions to flat uniqueness: a history bundle is exempt
// entirely (bdl-8 — a history bundle is a sequence of versions of the same resource, so
// the same fullUrl recurs by design), and in any other bundle two entries may share a
// fullUrl when their resources carry different meta.versionId (bdl-7's versioned clause).
// The uniqueness key is therefore (fullUrl, versionId): a repeat of the same pair is the
// violation. The repeated fullUrl value is deliberately not embedded in the diagnostic — a
// fullUrl can be a server URL that encodes an identifier, so the message names the two
// offending entry indices only, never a value. An entry with no fullUrl is skipped,
// because uniqueness constrains only the values present.
func checkUniqueFullURLsOutcome(b *Bundle, bundleType BundleType, outcome *fhir.OperationOutcome) {
	if bundleType == BundleTypeHistory {
		return
	}
	type urlKey struct{ url, version string }
	seen := make(map[urlKey]int, len(b.Entry))
	for i := range b.Entry {
		entry := &b.Entry[i]
		if entry.FullUrl == nil {
			continue
		}
		key := urlKey{url: *entry.FullUrl}
		if entry.Resource != nil {
			if v, ok := resourceVersionID(*entry.Resource); ok {
				key.version = v
			}
		}
		if first, dup := seen[key]; dup {
			fhir.AddIssue(outcome, fhir.SeverityError, fhir.IssueTypeInvalid,
				fmt.Sprintf("Bundle.entry[%d].fullUrl", i),
				fmt.Sprintf("fullUrl is not unique; it repeats the fullUrl of entry[%d] (bdl-7)", first))
			continue
		}
		seen[key] = i
	}
}

// composeReferenceIntegrity runs the existing CheckReferenceIntegrity walk and folds its
// release-typed OperationOutcome issues into the release-agnostic outcome the engine
// builds, so reference integrity is reported through the same fhir.Validate result as the
// structural checks. The translation copies the severity, code, diagnostics, and the first
// expression path of each issue; CheckReferenceIntegrity already builds those from
// reference strings and element paths, never patient values.
func composeReferenceIntegrity(b *Bundle, outcome *fhir.OperationOutcome) {
	refOutcome := b.CheckReferenceIntegrity()
	for i := range refOutcome.Issue {
		issue := &refOutcome.Issue[i]
		severity := fhir.SeverityError
		if issue.Severity != nil {
			severity = fhir.IssueSeverity(*issue.Severity)
		}
		code := fhir.IssueTypeNotFound
		if issue.Code != nil {
			code = fhir.IssueType(*issue.Code)
		}
		var expression string
		if len(issue.Expression) > 0 {
			expression = issue.Expression[0]
		}
		var diagnostics string
		if issue.Diagnostics != nil {
			diagnostics = *issue.Diagnostics
		}
		fhir.AddIssue(outcome, severity, code, expression, diagnostics)
	}
}

// resourceVersionID reads the meta.versionId of a resource and whether it is set, so the
// fullUrl-uniqueness check can key by (fullUrl, versionId) and honour bdl-7's versioned
// clause. It descends the embedded Resource/DomainResource base to its Meta field and then
// Meta.VersionId, all via reflection so it works across every release resource without a
// per-type accessor (the same reflection confinement reference.go uses for resource id). A
// resource with no meta or no versionId returns ("", false), which the caller treats as the
// unversioned key. It is nil-safe at every hop, so a malformed resource never panics here.
func resourceVersionID(r fhir.Resource) (string, bool) {
	v := reflect.ValueOf(r)
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return "", false
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return "", false
	}
	meta := v.FieldByName("Meta")
	if !meta.IsValid() || meta.Kind() != reflect.Pointer || meta.IsNil() {
		return "", false
	}
	meta = meta.Elem()
	if meta.Kind() != reflect.Struct {
		return "", false
	}
	version := meta.FieldByName("VersionId")
	if !version.IsValid() || version.Kind() != reflect.Pointer || version.IsNil() {
		return "", false
	}
	if s, ok := version.Elem().Interface().(string); ok && s != "" {
		return s, true
	}
	return "", false
}
