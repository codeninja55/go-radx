package fhir

import (
	"fmt"
	"strings"
)

// IssueSeverity is the severity of a single OperationOutcome issue. The set is
// identical across R4 and R5, so it lives in the root package and both release
// OperationOutcome types reuse these values rather than each defining their own.
// R5 adds the informational "success" severity (used by a successful operation
// response); R4 never emits it, but the constant is release-agnostic because the
// closed set is a superset and a stricter release simply never produces "success".
type IssueSeverity string

const (
	// SeverityFatal marks an issue that caused the operation to fail; the action
	// the consumer attempted did not and cannot complete.
	SeverityFatal IssueSeverity = "fatal"

	// SeverityError marks an issue that caused the operation to fail but for which
	// retry with corrected input may succeed.
	SeverityError IssueSeverity = "error"

	// SeverityWarning marks an issue that did not stop the operation but that the
	// consumer should be aware of.
	SeverityWarning IssueSeverity = "warning"

	// SeverityInformation marks an issue that is purely informational.
	SeverityInformation IssueSeverity = "information"

	// SeveritySuccess marks a successful operation result. It is the R5-only
	// severity; R4 never emits it.
	SeveritySuccess IssueSeverity = "success"
)

// IsError reports whether the severity denotes a failed operation (fatal or
// error). HasErrors on a release OperationOutcome is built from this predicate so
// the error/non-error split is defined in one place and stays release-agnostic.
func (s IssueSeverity) IsError() bool {
	return s == SeverityFatal || s == SeverityError
}

// IssueType classifies the kind of an OperationOutcome issue. The constants here
// are the subset the release-agnostic Validate engine emits; they are the same
// codes the generated release IssueType binding carries, kept in the root package
// so the engine can build issues without naming a release type. A release outcome's
// IssueType is a superset; these are the ones structural validation produces.
type IssueType string

const (
	// IssueTypeRequired marks a required element that is absent (a cardinality
	// min >= 1 element with no value present).
	IssueTypeRequired IssueType = "required"

	// IssueTypeStructure marks a structural error: a malformed shape, a choice group
	// with more than one branch set, or a contained resource that cannot be addressed.
	IssueTypeStructure IssueType = "structure"

	// IssueTypeValue marks a value that is invalid in itself: a code outside a
	// required binding's value set, or a primitive whose lexical form is malformed.
	IssueTypeValue IssueType = "value"

	// IssueTypeInvalid marks a content rule violation that is not a missing required
	// element or a bad value, such as a Bundle bdl-* invariant.
	IssueTypeInvalid IssueType = "invalid"

	// IssueTypeNotFound marks an unresolved local reference within a Bundle.
	IssueTypeNotFound IssueType = "not-found"
)

// OutcomeIssue is one issue of a release-agnostic OperationOutcome built by Validate.
// It names a severity, a coded type, a human-readable diagnostic, and the element
// path (expression) at which the issue was found. Every field is an element name, a
// path, a resource type, or a code — never a patient value — so an OutcomeIssue is
// safe to log and surface (PRD §9.1).
type OutcomeIssue struct {
	// Severity is the issue's severity. Validate emits error severity for a structural
	// failure and never embeds a value in any other field.
	Severity IssueSeverity

	// Code is the coded classification of the issue.
	Code IssueType

	// Diagnostics is the human-readable description. It names the element, path, type,
	// or code at fault, never the data value, so it carries no PHI.
	Diagnostics string

	// Expression is the element path the issue applies to (for example
	// "Patient.gender" or "Bundle.entry[1].resource.status"), or empty for a
	// resource-level issue.
	Expression string
}

// OperationOutcome is the release-agnostic result of Validate: an ordered set of
// issues, each naming a severity, a code, a diagnostic, and an element path. It
// mirrors the FHIR OperationOutcome resource but stays in the root package so a
// single engine can validate any release's resources without importing a release
// package. A release-specific OperationOutcome resource (for example
// r5.OperationOutcome) is the on-the-wire form; this is the in-process structural
// result.
type OperationOutcome struct {
	// Issue holds the reported issues in the order Validate found them. Validate
	// reports every issue it finds rather than stopping at the first, so a caller
	// sees the full set in one pass.
	Issue []OutcomeIssue
}

// add appends an issue to the outcome. It is the single mutation point so every
// emitted issue goes through one path; callers build the issue and hand it over.
func (o *OperationOutcome) add(issue OutcomeIssue) {
	o.Issue = append(o.Issue, issue)
}

// HasErrors reports whether the outcome carries at least one issue of error or fatal
// severity. It is nil-safe: a nil *OperationOutcome and an all-information outcome
// both report false, so a caller can write `if oo.HasErrors()` without a nil guard.
func (o *OperationOutcome) HasErrors() bool {
	if o == nil {
		return false
	}
	for i := range o.Issue {
		if o.Issue[i].Severity.IsError() {
			return true
		}
	}
	return false
}

// Error reports the outcome as a Go error, or nil when it carries no error-severity
// issue, so a caller can fold validation into the standard `if err != nil` flow. The
// message names the count and the diagnostics of the error-severity issues; it never
// includes a patient value, because issue diagnostics are built from element paths,
// types, and codes, not data. A nil *OperationOutcome and an outcome with no error
// issues both return nil.
func (o *OperationOutcome) Error() error {
	if !o.HasErrors() {
		return nil
	}
	var (
		msgs  []string
		count int
	)
	for i := range o.Issue {
		issue := &o.Issue[i]
		if !issue.Severity.IsError() {
			continue
		}
		count++
		msg := string(issue.Severity)
		if issue.Expression != "" {
			msg = msg + " at " + issue.Expression
		}
		if issue.Diagnostics != "" {
			msg = msg + ": " + issue.Diagnostics
		}
		msgs = append(msgs, msg)
	}
	return fmt.Errorf("fhir: validation reported %d error(s): %s", count, strings.Join(msgs, "; "))
}
