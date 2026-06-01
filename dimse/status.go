package dimse

import "fmt"

// ServiceClass selects the per-class categorisation table used to resolve a status code's
// category and meaning (PS3.7 Annex C plus the PS3.4 service-class annexes). The same numeric
// code can read differently per class, so a Status is always interpreted against the class it
// was constructed with. v1 models the classes the walking skeleton exercises; the remaining
// tables are added as their services land.
type ServiceClass uint8

const (
	ServiceClassGeneral ServiceClass = iota
	ServiceClassVerification
	ServiceClassStorage
	ServiceClassFind
	ServiceClassMove
	ServiceClassGet
	ServiceClassWorklist
	ServiceClassProcedureStep
	ServiceClassStorageCommitment
)

// StatusCategory is the Success / Pending / Warning / Cancel / Failure classification of a
// status code (PS3.7 Annex C). A code with no registered categorisation in its service class is
// StatusCategoryUnknown — never silently coerced to success.
type StatusCategory uint8

const (
	StatusCategoryUnknown StatusCategory = iota
	StatusCategorySuccess
	StatusCategoryPending
	StatusCategoryWarning
	StatusCategoryCancel
	StatusCategoryFailure
)

var statusCategoryNames = map[StatusCategory]string{
	StatusCategoryUnknown: "Unknown",
	StatusCategorySuccess: "Success",
	StatusCategoryPending: "Pending",
	StatusCategoryWarning: "Warning",
	StatusCategoryCancel:  "Cancel",
	StatusCategoryFailure: "Failure",
}

// String renders the category name in English (PRD §8.2).
func (c StatusCategory) String() string {
	if name, ok := statusCategoryNames[c]; ok {
		return name
	}
	return "Unknown"
}

// Status is a DIMSE status value (0000,0900). It wraps only the 16-bit wire code; Category() and
// Meaning() are derived against the ServiceClass it was constructed with, so a caller can never
// author a status whose category contradicts its code (PRD §8.1). Construct one from a named
// constant or from NewStatus.
type Status struct {
	Code uint16

	// serviceClass is the table that resolves Code's category and meaning. It is unexported so
	// the category is always derived, never an authorable field.
	serviceClass ServiceClass
}

// NewStatus binds a raw status code to the service class that decides how Category()/Meaning()
// resolve it. Prefer the named constants; reach for NewStatus only when handling a code received
// from the wire (PS3.7 §9).
func NewStatus(code uint16, sc ServiceClass) Status {
	return Status{Code: code, serviceClass: sc}
}

// ServiceClass reports the service class the status was constructed against.
func (s Status) ServiceClass() ServiceClass { return s.serviceClass }

// Category resolves the Success/Pending/Warning/Cancel/Failure class from the code and its
// service class. An entry in the class meaning table fixes the category; otherwise the general
// PS3.7 ranges decide; an unmatched code is StatusCategoryUnknown with the code preserved.
func (s Status) Category() StatusCategory {
	if entry, ok := statusTable(s.serviceClass)[s.Code]; ok {
		return entry.category
	}
	return codeToCategory(s.Code)
}

// Meaning returns the registered meaning, e.g. "Refused: SOP Class Not Supported". It is empty
// for bare Success and Cancel, and for a code with no registered meaning.
func (s Status) Meaning() string {
	if entry, ok := statusTable(s.serviceClass)[s.Code]; ok {
		return entry.meaning
	}
	return ""
}

// IsSuccess reports whether Category() is Success.
func (s Status) IsSuccess() bool { return s.Category() == StatusCategorySuccess }

// IsPending reports whether Category() is Pending (a continuing C-FIND/C-GET/C-MOVE response).
func (s Status) IsPending() bool { return s.Category() == StatusCategoryPending }

// IsWarning reports whether Category() is Warning.
func (s Status) IsWarning() bool { return s.Category() == StatusCategoryWarning }

// IsCancel reports whether Category() is Cancel.
func (s Status) IsCancel() bool { return s.Category() == StatusCategoryCancel }

// IsFailure reports whether Category() is Failure.
func (s Status) IsFailure() bool { return s.Category() == StatusCategoryFailure }

// String renders "0xC000 Failure: Unable to Process" — the code, its class, and the meaning
// when registered — never bare hex (PRD §8.2 legibility rule).
func (s Status) String() string {
	if meaning := s.Meaning(); meaning != "" {
		return fmt.Sprintf("0x%04X %s: %s", s.Code, s.Category(), meaning)
	}
	return fmt.Sprintf("0x%04X %s", s.Code, s.Category())
}

// Named status constants. Each pairs a code with the service class that defines its meaning, so
// the category can never be mis-authored: StatusEchoSuccess.IsSuccess() is always true.
var (
	// StatusSuccess is the general success status (0x0000).
	StatusSuccess = NewStatus(0x0000, ServiceClassGeneral)
	// StatusCancel is the general cancel status (0xFE00).
	StatusCancel = NewStatus(0xFE00, ServiceClassGeneral)
	// StatusEchoSuccess is the Verification (C-ECHO) success status (0x0000).
	StatusEchoSuccess = NewStatus(0x0000, ServiceClassVerification)
)

// statusEntry is a categorised, human-readable status meaning in a service-class table.
type statusEntry struct {
	category StatusCategory
	meaning  string
}

// statusTable returns the meaning table for a service class. Verification reuses the general
// table (PS3.4 A.4 defines no additional Verification statuses), mirroring pynetdicom's
// VERIFICATION_SERVICE_CLASS_STATUS = GENERAL_STATUS. Storage and the query/retrieve classes are
// added with their services (Increment 5+).
func statusTable(sc ServiceClass) map[uint16]statusEntry {
	switch sc {
	case ServiceClassVerification:
		return generalStatusTable
	default:
		return generalStatusTable
	}
}

// generalStatusTable is the PS3.7 Annex C general status table, ported from pynetdicom's
// GENERAL_STATUS. Only explicitly-named codes appear; ranged bands (0xA000–0xBFFF,
// 0xC000–0xCFFF) are resolved by codeToCategory.
var generalStatusTable = map[uint16]statusEntry{
	0x0000: {StatusCategorySuccess, ""},
	0x0105: {StatusCategoryFailure, "No Such Attribute"},
	0x0106: {StatusCategoryFailure, "Invalid Attribute Value"},
	0x0107: {StatusCategoryWarning, "Attribute List Error"},
	0x0110: {StatusCategoryFailure, "Processing Failure"},
	0x0111: {StatusCategoryFailure, "Duplicate SOP Instance"},
	0x0112: {StatusCategoryFailure, "No Such SOP Instance"},
	0x0113: {StatusCategoryFailure, "No Such Event Type"},
	0x0114: {StatusCategoryFailure, "No Such Argument"},
	0x0115: {StatusCategoryFailure, "Invalid Argument Value"},
	0x0116: {StatusCategoryWarning, "Attribute Value Out of Range"},
	0x0117: {StatusCategoryFailure, "Invalid Object Instance"},
	0x0118: {StatusCategoryFailure, "No Such SOP Class"},
	0x0119: {StatusCategoryFailure, "Class-Instance Conflict"},
	0x0120: {StatusCategoryFailure, "Missing Attribute"},
	0x0121: {StatusCategoryFailure, "Missing Attribute Value"},
	0x0122: {StatusCategoryFailure, "Refused: SOP Class Not Supported"},
	0x0123: {StatusCategoryFailure, "No Such Action"},
	0x0124: {StatusCategoryFailure, "Refused: Not Authorised"},
	0x0210: {StatusCategoryFailure, "Duplicate Invocation"},
	0x0211: {StatusCategoryFailure, "Unrecognised Operation"},
	0x0212: {StatusCategoryFailure, "Mistyped Argument"},
	0x0213: {StatusCategoryFailure, "Resource Limitation"},
	0xFE00: {StatusCategoryCancel, ""},
}

// codeToCategory resolves a status code's category by the PS3.7 general ranges, ported faithfully
// from pynetdicom's code_to_category. A code matching no specific value or range is Unknown — the
// code is never coerced to success.
func codeToCategory(code uint16) StatusCategory {
	switch {
	case code == 0x0000:
		return StatusCategorySuccess
	case code == 0xFF00 || code == 0xFF01:
		return StatusCategoryPending
	case code == 0xFE00:
		return StatusCategoryCancel
	case code >= 0xA000 && code <= 0xAFFF:
		return StatusCategoryFailure
	case code >= 0xC000 && code <= 0xCFFF:
		return StatusCategoryFailure
	case code == 0x0107 || code == 0x0116:
		return StatusCategoryWarning
	case code >= 0xB000 && code <= 0xBFFF:
		return StatusCategoryWarning
	case code == 0x0001:
		return StatusCategoryWarning
	default:
		return StatusCategoryUnknown
	}
}
