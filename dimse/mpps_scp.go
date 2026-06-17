package dimse

import (
	"context"
	"strings"
	"sync"

	"github.com/codeninja55/go-radx/dicom"
)

// MPPS SCP status codes (PS3.4 F.7.2 plus the PS3.7 Annex C general DIMSE-N codes the Modality
// Performed Procedure Step service class reuses, verified against pynetdicom's MPPS SCP example
// docs/examples/mpps.rst). They are constructed against ServiceClassProcedureStep so the category and
// meaning resolve through procedureStepStatusTable, where 0x0110 reads as the procedure-step-specific
// "may no longer be updated" Failure.
var (
	// StatusMPPSInvalidAttributeValue is the N-CREATE/N-SET Failure "Invalid Attribute Value" (0x0106):
	// an attribute the request supplied has a value the SCP rejects — for example a Performed Procedure
	// Step Status that is not "IN PROGRESS" on N-CREATE (pynetdicom MPPS SCP handle_create).
	StatusMPPSInvalidAttributeValue = NewStatus(0x0106, ServiceClassProcedureStep)
	// StatusMPPSDuplicateInstance is the N-CREATE Failure "Duplicate SOP Instance" (0x0111): the
	// Affected SOP Instance UID names a procedure step the SCP already manages (pynetdicom MPPS SCP).
	StatusMPPSDuplicateInstance = NewStatus(0x0111, ServiceClassProcedureStep)
	// StatusMPPSNoSuchInstance is the N-SET Failure "No Such SOP Instance" (0x0112): the Requested SOP
	// Instance UID names no procedure step the SCP manages (pynetdicom MPPS SCP handle_set).
	StatusMPPSNoSuchInstance = NewStatus(0x0112, ServiceClassProcedureStep)
	// StatusMPPSMissingAttribute is the N-CREATE Failure "Missing Attribute" (0x0120): a mandatory
	// attribute — the Performed Procedure Step Status — is absent from the N-CREATE Attribute List
	// (pynetdicom MPPS SCP handle_create).
	StatusMPPSMissingAttribute = NewStatus(0x0120, ServiceClassProcedureStep)
)

// MPPSStore is the application hook an MPPSProvider delegates the lifecycle of Modality Performed
// Procedure Step instances to (PS3.4 Annex F). The provider performs the protocol-level validation —
// the MPPS SOP class, the mandatory attributes, the state-transition rules — and calls the store only
// to persist and look up the managed step, so a consumer plugs in its own persistence without
// reimplementing the N-CREATE/N-SET semantics.
//
// A data set passed to the store may carry patient and procedure-step attributes (PHI); an
// implementation MUST NOT log it (PRD §9.1). The provider serialises calls per association but a
// shared store may see concurrent calls from multiple associations, so an implementation must be
// safe for concurrent use.
type MPPSStore interface {
	// CreateStep persists a newly created Performed Procedure Step under instanceUID with its initial
	// attributes (state IN PROGRESS). It returns a Failure-category status when it cannot persist the
	// step — for example StatusMPPSDuplicateInstance when instanceUID already names a managed step — so
	// a duplicate is never laundered into success (PRD §9.2 fail-closed).
	CreateStep(ctx context.Context, instanceUID dicom.UID, attrs *dicom.DataSet) Status
	// LookupStep returns the current attributes of the managed step named by instanceUID and whether it
	// exists. The provider reads the step's current Performed Procedure Step Status from the returned
	// attributes to enforce the state transition before applying an N-SET.
	LookupStep(ctx context.Context, instanceUID dicom.UID) (*dicom.DataSet, bool)
	// UpdateStep applies the N-SET modification list to the managed step named by instanceUID, advancing
	// its state. It returns a Failure-category status when it cannot apply the update; the provider has
	// already enforced existence and the state transition before calling it.
	UpdateStep(ctx context.Context, instanceUID dicom.UID, mods *dicom.DataSet) Status
}

// MemoryMPPSStore is a minimal in-memory MPPSStore: it keeps each created Performed Procedure Step in
// a map keyed by SOP Instance UID and merges an N-SET modification list into the stored attributes. It
// is the default store for tests and small deployments; it is safe for concurrent use. It is NOT a
// durable store — a process restart loses every step — so a production SCP supplies its own MPPSStore.
type MemoryMPPSStore struct {
	mu    sync.Mutex
	steps map[dicom.UID]*dicom.DataSet
}

// NewMemoryMPPSStore builds an empty in-memory MPPS store.
func NewMemoryMPPSStore() *MemoryMPPSStore {
	return &MemoryMPPSStore{steps: make(map[dicom.UID]*dicom.DataSet)}
}

// CreateStep stores a copy of attrs under instanceUID, refusing a duplicate with
// StatusMPPSDuplicateInstance (the provider has already rejected an empty instance UID and a missing
// or non-IN-PROGRESS state before calling this).
func (s *MemoryMPPSStore) CreateStep(_ context.Context, instanceUID dicom.UID, attrs *dicom.DataSet) Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.steps[instanceUID]; exists {
		return StatusMPPSDuplicateInstance
	}
	s.steps[instanceUID] = attrs.Clone()
	return StatusMPPSSuccess
}

// LookupStep returns a copy of the stored step's attributes so a caller cannot mutate the store's
// state through the returned data set.
func (s *MemoryMPPSStore) LookupStep(_ context.Context, instanceUID dicom.UID) (*dicom.DataSet, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	step, ok := s.steps[instanceUID]
	if !ok {
		return nil, false
	}
	return step.Clone(), true
}

// UpdateStep merges every element of mods into the stored step, overwriting on tag collision. The
// provider has already enforced existence and the state transition, so this only applies the change.
func (s *MemoryMPPSStore) UpdateStep(_ context.Context, instanceUID dicom.UID, mods *dicom.DataSet) Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	step, ok := s.steps[instanceUID]
	if !ok {
		return StatusMPPSNoSuchInstance
	}
	for el := range mods.All() {
		step.Set(el)
	}
	return StatusMPPSSuccess
}

// MPPSProvider is the Modality Performed Procedure Step SCP (PS3.4 Annex F): it answers the N-CREATE
// that opens a procedure step IN PROGRESS and the N-SET that advances it to COMPLETED or
// DISCONTINUED, enforcing the MPPS protocol rules pynetdicom's MPPS SCP enforces (the mandatory
// Affected SOP Instance UID and Performed Procedure Step Status on N-CREATE, the existence of the step
// on N-SET, and the one-way IN PROGRESS to a final state transition). It plugs into the DIMSE-N SCP
// dispatch substrate as an NCreateHandler and NSetHandler; register it as the Server's handler over a
// Modality Performed Procedure Step presentation context (ModalityPerformedContexts).
//
// The provider holds no procedure-step state itself; it delegates persistence and lookup to an
// MPPSStore. It carries no global mutable state and is safe to use from the per-association goroutines
// the Server runs.
type MPPSProvider struct {
	store MPPSStore
}

// NewMPPSProvider builds an MPPS SCP backed by store. Pass NewMemoryMPPSStore for the in-memory
// default, or a custom MPPSStore to persist steps durably. A nil store is a programming error; the
// caller supplies one.
func NewMPPSProvider(store MPPSStore) *MPPSProvider {
	return &MPPSProvider{store: store}
}

// NCreate answers an N-CREATE that opens a Modality Performed Procedure Step (PS3.4 F.7.1). It
// enforces the rules pynetdicom's MPPS SCP enforces before persisting the step:
//
//   - the MPPS SOP class — the dispatch already validated the presentation context against the
//     command's Affected SOP Class UID (validateNContext); this rejects an N-CREATE whose Affected SOP
//     Class UID is not Modality Performed Procedure Step with StatusSOPClassNotSupported;
//   - a present Affected SOP Instance UID — MPPS requires the SCU to assign the step's instance UID
//     (PS3.4 F.7.1); an absent one is StatusMPPSInvalidAttributeValue (0x0106);
//   - a non-empty Attribute List carrying Performed Procedure Step Status (0040,0252) — an absent
//     status is StatusMPPSMissingAttribute (0x0120);
//   - the status value must be IN PROGRESS — a step is always opened IN PROGRESS (PS3.3 C.4.14); any
//     other value is StatusMPPSInvalidAttributeValue (0x0106).
//
// On success it persists the step through the store under the SCU-supplied instance UID and returns
// that UID for the dispatch to echo in the N-CREATE-RSP Affected SOP Instance UID. A store-reported
// failure (for example a duplicate instance) is returned as-is, never laundered into success.
//
// No patient or procedure-step value is logged (PRD §9.1).
func (p *MPPSProvider) NCreate(ctx context.Context, req NRequest) (Status, dicom.UID) {
	if dicom.SOPClassUID(req.AffectedSOPClassUID) != modalityPerformedStepSOPClass {
		return StatusSOPClassNotSupported, ""
	}
	if req.AffectedSOPInstanceUID == "" {
		return StatusMPPSInvalidAttributeValue, ""
	}
	if req.DataSet == nil || req.DataSet.Len() == 0 {
		return StatusMPPSMissingAttribute, ""
	}
	state, ok := req.DataSet.GetString(dicom.TagPerformedProcedureStepStatus)
	if !ok {
		return StatusMPPSMissingAttribute, ""
	}
	if !equalsStepState(state, ProcedureStepInProgress) {
		return StatusMPPSInvalidAttributeValue, ""
	}

	status := p.store.CreateStep(ctx, req.AffectedSOPInstanceUID, req.DataSet)
	if !status.IsSuccess() {
		return status, ""
	}
	return status, req.AffectedSOPInstanceUID
}

// NSet answers an N-SET that advances a Modality Performed Procedure Step (PS3.4 F.7.1). It enforces
// the rules pynetdicom's MPPS SCP enforces before applying the modification list:
//
//   - the MPPS SOP class — the dispatch validated the context against the Requested SOP Class UID;
//     this rejects an N-SET whose Requested SOP Class UID is not Modality Performed Procedure Step;
//   - the step must exist — an unknown Requested SOP Instance UID is StatusMPPSNoSuchInstance (0x0112);
//   - the state transition must be legal — a step already in a final state (COMPLETED or DISCONTINUED)
//     may no longer be updated (PS3.4 F.7.1, F.8.1), so an N-SET against it is
//     StatusMPPSMayNoLongerBeUpdated (0x0110); when the modification list carries a new Performed
//     Procedure Step Status it must be a final state (COMPLETED or DISCONTINUED), not a return to
//     IN PROGRESS — an invalid transition is StatusMPPSInvalidAttributeValue (0x0106).
//
// On a legal transition it applies the modification list through the store. No patient or
// procedure-step value is logged (PRD §9.1).
func (p *MPPSProvider) NSet(ctx context.Context, req NRequest) Status {
	if dicom.SOPClassUID(req.RequestedSOPClassUID) != modalityPerformedStepSOPClass {
		return StatusSOPClassNotSupported
	}
	if req.DataSet == nil || req.DataSet.Len() == 0 {
		return StatusMPPSInvalidAttributeValue
	}

	current, ok := p.store.LookupStep(ctx, req.RequestedSOPInstanceUID)
	if !ok {
		return StatusMPPSNoSuchInstance
	}

	currentState, _ := current.GetString(dicom.TagPerformedProcedureStepStatus)
	if isFinalStepState(currentState) {
		// A COMPLETED or DISCONTINUED step is terminal; no N-SET may reopen or re-finalise it.
		return StatusMPPSMayNoLongerBeUpdated
	}

	if newState, has := req.DataSet.GetString(dicom.TagPerformedProcedureStepStatus); has && !isFinalStepState(newState) {
		// The only legal status transition from IN PROGRESS is to a final state; an N-SET that sets the
		// status to anything else (including back to IN PROGRESS) is an invalid attribute value.
		return StatusMPPSInvalidAttributeValue
	}

	return p.store.UpdateStep(ctx, req.RequestedSOPInstanceUID, req.DataSet)
}

// equalsStepState reports whether the wire CS value equals the given ProcedureStepState's keyword,
// case-insensitively and ignoring surrounding whitespace (a CS value may be padded). The IN PROGRESS
// keyword carries an embedded space, so the comparison is against the full keyword, not a token.
func equalsStepState(value string, state ProcedureStepState) bool {
	return strings.EqualFold(strings.TrimSpace(value), state.String())
}

// isFinalStepState reports whether the wire CS value is a terminal Performed Procedure Step Status —
// COMPLETED or DISCONTINUED (PS3.3 C.4.14). A step in a final state may no longer be updated.
func isFinalStepState(value string) bool {
	return equalsStepState(value, ProcedureStepCompleted) || equalsStepState(value, ProcedureStepDiscontinued)
}
