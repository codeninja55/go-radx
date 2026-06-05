package fhir

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
