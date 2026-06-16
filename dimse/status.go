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
//
// The zero value Status{} has code 0x0000 and so reports IsSuccess() == true; it is NOT a "no
// status" sentinel. Operations that return (Status, error) return this zero value on their error
// paths, so a returned Status is only meaningful when the returned error is nil — check the error
// first.
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

	// StatusSOPClassNotSupported is the general DIMSE failure "Refused: SOP Class Not Supported"
	// (0x0122, PS3.7 Annex C). An SCP returns it when the inbound operation's service is one the
	// hosted handler does not provide (e.g. a C-ECHO reaching a store-only handler), refusing the
	// operation gracefully rather than aborting (interface segregation, PRD §8.2).
	StatusSOPClassNotSupported = NewStatus(0x0122, ServiceClassGeneral)

	// StatusStoreSuccess is the Storage (C-STORE) success status (0x0000): the SOP Instance was
	// stored (PS3.4 B.2.3).
	StatusStoreSuccess = NewStatus(0x0000, ServiceClassStorage)
	// StatusStoreOutOfResources is the Storage failure "Refused: Out of Resources" (0xA700 band).
	StatusStoreOutOfResources = NewStatus(0xA700, ServiceClassStorage)
	// StatusStoreDataSetDoesNotMatchSOPClass is the Storage failure "Data Set Does Not Match SOP
	// Class" (0xA900 band).
	StatusStoreDataSetDoesNotMatchSOPClass = NewStatus(0xA900, ServiceClassStorage)
	// StatusStoreCannotUnderstand is the Storage failure "Cannot Understand" (0xC000 band): the
	// SCP could not parse or process the dataset.
	StatusStoreCannotUnderstand = NewStatus(0xC000, ServiceClassStorage)
	// StatusStoreCoercionOfDataElements is the Storage warning "Coercion of Data Elements"
	// (0xB000): the instance was stored, but some elements were coerced.
	StatusStoreCoercionOfDataElements = NewStatus(0xB000, ServiceClassStorage)
	// StatusStoreElementDiscarded is the Storage warning "Element Discarded" (0xB006).
	StatusStoreElementDiscarded = NewStatus(0xB006, ServiceClassStorage)
	// StatusStoreDataSetDoesNotMatchSOPClassWarning is the Storage warning "Data Set Does Not
	// Match SOP Class" (0xB007): stored with a warning, distinct from the 0xA900 failure.
	StatusStoreDataSetDoesNotMatchSOPClassWarning = NewStatus(0xB007, ServiceClassStorage)

	// StatusFindPending is the C-FIND Pending status (0xFF00): a match is supplied and matching
	// continues (PS3.4 C.4.1.1.4). It must never read as success — IsPending(), not IsSuccess().
	StatusFindPending = NewStatus(0xFF00, ServiceClassFind)
	// StatusFindPendingOptionalKeys is the C-FIND Pending-with-warning status (0xFF01): matching
	// continues but one or more optional keys were not supported.
	StatusFindPendingOptionalKeys = NewStatus(0xFF01, ServiceClassFind)
	// StatusFindSuccess is the C-FIND terminal success status (0x0000): matching is complete.
	StatusFindSuccess = NewStatus(0x0000, ServiceClassFind)

	// StatusMovePending is the C-MOVE Pending status (0xFF00): sub-operations are continuing
	// (PS3.4 C.4.2.1.5).
	StatusMovePending = NewStatus(0xFF00, ServiceClassMove)
	// StatusMoveSuccess is the C-MOVE terminal success status (0x0000): all sub-operations
	// completed successfully.
	StatusMoveSuccess = NewStatus(0x0000, ServiceClassMove)
	// StatusMoveSubOpsCompleteWithFailures is the C-MOVE Warning "Sub-operations complete — one or
	// more failures" (0xB000): the retrieve finished but at least one sub-operation C-STORE failed.
	// It is a Warning, never laundered to success (PRD §9.2 fail-closed).
	StatusMoveSubOpsCompleteWithFailures = NewStatus(0xB000, ServiceClassMove)
	// StatusMoveDestinationUnknown is the C-MOVE Failure "Move Destination Unknown" (0xA801): the
	// requested Move Destination AE Title was not recognised.
	StatusMoveDestinationUnknown = NewStatus(0xA801, ServiceClassMove)
	// StatusMoveCancel is the C-MOVE terminal Cancel status (0xFE00): the SCU sent a C-CANCEL-RQ
	// mid-retrieve and the SCP stopped the sub-operation loop, reporting the accumulated counts
	// (PS3.4 C.4.2.3). It is a clean cancellation, never a protocol fault.
	StatusMoveCancel = NewStatus(0xFE00, ServiceClassMove)

	// StatusGetPending is the C-GET Pending status (0xFF00): sub-operations are continuing
	// (PS3.4 C.4.3.1.4).
	StatusGetPending = NewStatus(0xFF00, ServiceClassGet)
	// StatusGetSuccess is the C-GET terminal success status (0x0000): all sub-operations completed
	// successfully.
	StatusGetSuccess = NewStatus(0x0000, ServiceClassGet)
	// StatusGetSubOpsCompleteWithFailures is the C-GET Warning "Sub-operations complete — one or
	// more failures or warnings" (0xB000): the retrieve finished but at least one sub-operation
	// C-STORE failed. It is a Warning, never laundered to success (PRD §9.2 fail-closed).
	StatusGetSubOpsCompleteWithFailures = NewStatus(0xB000, ServiceClassGet)
	// StatusGetCancel is the C-GET terminal Cancel status (0xFE00): the SCU sent a C-CANCEL-RQ
	// mid-retrieve and the SCP stopped the sub-operation loop, reporting the accumulated counts
	// (PS3.7 §9.3.2.3). It is a clean cancellation, never a protocol fault.
	StatusGetCancel = NewStatus(0xFE00, ServiceClassGet)

	// StatusWorklistPending is the Modality Worklist C-FIND Pending status (0xFF00): a worklist
	// item is supplied and matching continues (PS3.4 K.4.1.1.4).
	StatusWorklistPending = NewStatus(0xFF00, ServiceClassWorklist)
	// StatusWorklistSuccess is the Modality Worklist C-FIND terminal success status (0x0000).
	StatusWorklistSuccess = NewStatus(0x0000, ServiceClassWorklist)

	// StatusMPPSSuccess is the MPPS (Procedure Step) success status (0x0000): the N-CREATE or N-SET
	// was accepted (PS3.4 F.7.2).
	StatusMPPSSuccess = NewStatus(0x0000, ServiceClassProcedureStep)
	// StatusMPPSMayNoLongerBeUpdated is the MPPS Failure (0x0110): the Performed Procedure Step
	// object may no longer be updated (it has already reached a final state).
	StatusMPPSMayNoLongerBeUpdated = NewStatus(0x0110, ServiceClassProcedureStep)

	// StatusStorageCommitmentSuccess is the Storage Commitment success status (0x0000): the
	// N-ACTION request was accepted (the commitment result follows asynchronously via
	// N-EVENT-REPORT). PS3.4 J.3.
	StatusStorageCommitmentSuccess = NewStatus(0x0000, ServiceClassStorageCommitment)
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
	case ServiceClassStorage:
		return storageStatusTable
	case ServiceClassVerification:
		return generalStatusTable
	case ServiceClassFind:
		return findStatusTable
	case ServiceClassMove:
		return moveStatusTable
	case ServiceClassGet:
		return getStatusTable
	case ServiceClassWorklist:
		return worklistStatusTable
	case ServiceClassProcedureStep:
		return procedureStepStatusTable
	case ServiceClassStorageCommitment:
		// Storage Commitment defines no service-specific codes; it is GENERAL_STATUS verbatim,
		// mirroring pynetdicom's STORAGE_COMMITMENT_SERVICE_CLASS_STATUS = GENERAL_STATUS.
		return generalStatusTable
	default:
		return generalStatusTable
	}
}

// mergeGeneral returns a copy of specific with every general status folded in where specific does
// not already define the code, mirroring pynetdicom's `QR_*_SERVICE_CLASS_STATUS = {...}` then
// `.update(GENERAL_STATUS)` pattern (the service-specific codes win over the general defaults).
func mergeGeneral(specific map[uint16]statusEntry) map[uint16]statusEntry {
	t := make(map[uint16]statusEntry, len(specific)+len(generalStatusTable))
	for code, entry := range specific {
		t[code] = entry
	}
	for code, entry := range generalStatusTable {
		if _, taken := t[code]; !taken {
			t[code] = entry
		}
	}
	return t
}

// findStatusTable is the PS3.4 C.4.1.1.4 Query/Retrieve FIND service-class status table, ported
// from pynetdicom's QR_FIND_SERVICE_CLASS_STATUS (the service-specific codes merged over
// GENERAL_STATUS). 0xFF00/0xFF01 are Pending; 0xB001 is the "response limit reached" Warning; the
// 0xC000–0xCFFF Unable-to-Process band resolves via codeToCategory.
var findStatusTable = mergeGeneral(map[uint16]statusEntry{
	0x0000: {StatusCategorySuccess, ""},
	0xA700: {StatusCategoryFailure, "Refused: Out of Resources"},
	0xA710: {StatusCategoryFailure, "Invalid Prior Record Key"},
	0xA900: {StatusCategoryFailure, "Identifier Does Not Match SOP Class"},
	0xB001: {StatusCategoryWarning, "Matching reached response limit, subsequent request may return additional matches"},
	0xC000: {StatusCategoryFailure, "Unable to Process"},
	0xFF00: {StatusCategoryPending, "Matches are continuing, current match supplied"},
	0xFF01: {StatusCategoryPending, "Matches are continuing, optional keys not supported"},
})

// moveStatusTable is the PS3.4 C.4.2.1.5 Query/Retrieve MOVE service-class status table, ported
// from pynetdicom's QR_MOVE_SERVICE_CLASS_STATUS. 0xB000 is the "sub-operations complete, one or
// more failures" Warning (not success); 0xA801 is "Move Destination Unknown"; 0xFF00 is Pending.
var moveStatusTable = mergeGeneral(map[uint16]statusEntry{
	0x0000: {StatusCategorySuccess, ""},
	0xA701: {StatusCategoryFailure, "Refused: Out of Resources — unable to calculate number of matches"},
	0xA702: {StatusCategoryFailure, "Refused: Out of Resources — unable to perform sub-operations"},
	0xA801: {StatusCategoryFailure, "Move Destination Unknown"},
	0xA900: {StatusCategoryFailure, "Identifier Does Not Match SOP Class"},
	0xB000: {StatusCategoryWarning, "Sub-operations Complete — One or More Failures"},
	0xC000: {StatusCategoryFailure, "Unable to Process"},
	0xFF00: {StatusCategoryPending, "Sub-operations are continuing"},
})

// getStatusTable is the PS3.4 C.4.3.1.4 Query/Retrieve GET service-class status table, ported from
// pynetdicom's QR_GET_SERVICE_CLASS_STATUS. It parallels MOVE: 0xB000 is the "sub-operations
// complete, one or more failures or warnings" Warning; 0xFF00 is Pending.
var getStatusTable = mergeGeneral(map[uint16]statusEntry{
	0x0000: {StatusCategorySuccess, ""},
	0xA701: {StatusCategoryFailure, "Refused: Out of Resources — unable to calculate number of matches"},
	0xA702: {StatusCategoryFailure, "Refused: Out of Resources — unable to perform sub-operations"},
	0xA900: {StatusCategoryFailure, "Identifier Does Not Match SOP Class"},
	0xB000: {StatusCategoryWarning, "Sub-operations Complete — One or More Failures or Warnings"},
	0xC000: {StatusCategoryFailure, "Unable to Process"},
	0xFF00: {StatusCategoryPending, "Sub-operations are continuing"},
})

// worklistStatusTable is the PS3.4 Annex K Modality Worklist service-class status table, ported
// from pynetdicom's MODALITY_WORKLIST_SERVICE_CLASS_STATUS. It is FIND-shaped: 0xFF00/0xFF01 are
// Pending, with the worklist-specific Optional-Keys wording on 0xFF01.
var worklistStatusTable = mergeGeneral(map[uint16]statusEntry{
	0x0000: {StatusCategorySuccess, ""},
	0xA700: {StatusCategoryFailure, "Refused: Out of Resources"},
	0xA900: {StatusCategoryFailure, "Identifier Does Not Match SOP Class"},
	0xC000: {StatusCategoryFailure, "Unable to Process"},
	0xFF00: {StatusCategoryPending, "Matches are continuing, current match supplied, optional keys supported"},
	0xFF01: {StatusCategoryPending, "Matches are continuing, optional keys not supported"},
})

// procedureStepStatusTable is the MPPS (Performed Procedure Step) service-class status table,
// ported from pynetdicom's PROCEDURE_STEP_STATUS (the procedure-step-specific codes merged over
// GENERAL_STATUS). 0x0001 is the "optional attributes not supported" Warning; 0x0110 overrides the
// general Processing Failure with the procedure-step-specific "may no longer be updated" meaning,
// keeping the Failure category.
var procedureStepStatusTable = mergeGeneral(map[uint16]statusEntry{
	0x0001: {StatusCategoryWarning, "Requested Optional Attributes Are Not Supported"},
	0x0110: {StatusCategoryFailure, "Performed Procedure Step Object May No Longer Be Updated"},
})

// storageStatusTable is the PS3.4 B.2.3 Storage service-class status table, ported from
// pynetdicom's STORAGE_SERVICE_CLASS_STATUS (which merges the Storage-specific codes over the
// GENERAL_STATUS table). Only the explicitly-named codes carry a meaning here; the ranged failure
// bands (0xA700–0xA7FF Out of Resources, 0xA900–0xA9FF mismatch, 0xC000–0xCFFF Cannot Understand)
// resolve their category via codeToCategory, and a code's meaning is the band representative when
// looked up exactly. General DIMSE statuses inherited from the general table are folded in so a
// peer's general failure (e.g. 0x0110 Processing Failure) still categorises correctly.
var storageStatusTable = mergeGeneral(map[uint16]statusEntry{
	0x0000: {StatusCategorySuccess, ""},
	0xA700: {StatusCategoryFailure, "Refused: Out of Resources"},
	0xA900: {StatusCategoryFailure, "Data Set Does Not Match SOP Class"},
	0xC000: {StatusCategoryFailure, "Cannot Understand"},
	0xB000: {StatusCategoryWarning, "Coercion of Data Elements"},
	0xB006: {StatusCategoryWarning, "Element Discarded"},
	0xB007: {StatusCategoryWarning, "Data Set Does Not Match SOP Class"},
})

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
