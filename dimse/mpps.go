package dimse

import (
	"context"

	"github.com/codeninja55/go-radx/dicom"
)

// ModalityPerformedProcedureStepSOPClass is the Modality Performed Procedure Step SOP Class UID
// (PS3.4 F.6), the abstract syntax of an MPPS presentation context. It is exported so a caller may
// name it (for example to inspect the accepted contexts on an established association) or so an SCP
// recognises it; Association.MPPS selects it implicitly.
const ModalityPerformedProcedureStepSOPClass = modalityPerformedStepSOPClass

// ProcedureStepState is the Performed Procedure Step Status (0040,0252), a CS value (PS3.3
// C.4.14) that tracks where a Modality Performed Procedure Step sits in its lifecycle. An MPPS SCU
// opens the step IN PROGRESS with N-CREATE, then advances it to COMPLETED or DISCONTINUED with
// N-SET. It is the typed status the SCU writes into the attribute set; the wire keyword is the CS
// string String() renders.
type ProcedureStepState uint8

const (
	// ProcedureStepInProgress is the state of a step just opened by an N-CREATE: imaging is under
	// way (PS3.3 C.4.14, wire keyword "IN PROGRESS").
	ProcedureStepInProgress ProcedureStepState = iota
	// ProcedureStepCompleted is the terminal state of a step that finished normally, set by an N-SET
	// (wire keyword "COMPLETED").
	ProcedureStepCompleted
	// ProcedureStepDiscontinued is the terminal state of a step that was abandoned, set by an N-SET
	// (wire keyword "DISCONTINUED").
	ProcedureStepDiscontinued
)

// procedureStepKeywords maps each state to the CS keyword written into Performed Procedure Step
// Status (0040,0252). The IN PROGRESS keyword carries a single embedded space (PS3.3 C.4.14).
var procedureStepKeywords = map[ProcedureStepState]string{
	ProcedureStepInProgress:   "IN PROGRESS",
	ProcedureStepCompleted:    "COMPLETED",
	ProcedureStepDiscontinued: "DISCONTINUED",
}

// String renders the CS keyword written into Performed Procedure Step Status (0040,0252).
func (s ProcedureStepState) String() string {
	if kw, ok := procedureStepKeywords[s]; ok {
		return kw
	}
	return "UNKNOWN"
}

// MPPS is the Modality Performed Procedure Step SCU bound to an association (PS3.4 Annex F). It
// drives the two normalised operations a modality uses to report a procedure step: Create issues
// the N-CREATE that opens the step IN PROGRESS, and Set issues the N-SET that advances it to
// COMPLETED or DISCONTINUED. The SCP side is out of v1 scope. Obtain one with Association.MPPS.
//
// An Association is not safe for concurrent operations; run one MPPS exchange at a time on it.
type MPPS struct {
	assoc *Association
}

// MPPS returns the Modality Performed Procedure Step SCU bound to this association. The caller
// negotiates the MPPS presentation context with ModalityPerformedContexts before associating, then
// drives Create followed by Set. It is nil-safe: an MPPS over a nil/unestablished association still
// returns typed errors from Create/Set rather than panicking (Codex DIMSE-017).
func (a *Association) MPPS() *MPPS { return &MPPS{assoc: a} }

// Create issues an N-CREATE that opens a Modality Performed Procedure Step in the IN PROGRESS state
// (PS3.4 F.7.1). It selects the MPPS presentation context, encodes attrs as the command's data set,
// and returns the SOP Instance UID of the created step together with the typed procedure-step
// status. The instance UID is the one the SCU supplied in attrs as SOP Instance UID (0008,0018) when
// present, otherwise the one the SCP assigned and echoed in the N-CREATE-RSP Affected SOP Instance
// UID — the caller passes whichever it gets to Set.
//
// When attrs does not already carry a Performed Procedure Step Status (0040,0252) Create writes
// IN PROGRESS into a COPY of attrs, leaving the caller's data set untouched. A nil/empty attribute
// set, or no negotiated MPPS context, is a fail-closed *ValidationError/*AssociationError returned
// before any wire I/O (PRD §9.2). The returned Status is meaningful only when the returned error is
// nil; a Failure-category status is in-band data the caller inspects, not a Go error.
//
// No patient, procedure-step, or SOP instance value is ever logged or embedded in an error string
// (PRD §9.1).
func (m *MPPS) Create(ctx context.Context, attrs *dicom.DataSet) (dicom.UID, Status, error) {
	a := m.assoc
	if a == nil || a.requestor == nil {
		return "", Status{}, &AssociationError{Kind: AssociationNotEstablished, Detail: "MPPS Create on an unestablished association"}
	}
	a.mu.Lock()
	released := a.released
	a.mu.Unlock()
	if released {
		return "", Status{}, &AssociationError{Kind: AssociationNotEstablished, Detail: "MPPS Create on a released association"}
	}
	if attrs == nil || attrs.Len() == 0 {
		return "", Status{}, &ValidationError{Detail: "MPPS Create requires a non-empty attribute set for the N-CREATE data set"}
	}

	pcID, ts, ok := a.contextForQuery(modalityPerformedStepSOPClass)
	if !ok {
		return "", Status{}, &AssociationError{
			Kind:   AssociationNotEstablished,
			Detail: "no accepted presentation context for the Modality Performed Procedure Step SOP Class",
		}
	}

	// Write IN PROGRESS into a copy so the caller's data set is untouched; the SCU never overwrites a
	// state the caller set explicitly.
	step := attrs.Clone()
	if _, has := step.GetString(dicom.TagPerformedProcedureStepStatus); !has {
		step.SetString(dicom.TagPerformedProcedureStepStatus, ProcedureStepInProgress.String())
	}
	// An SOP Instance UID the caller supplied in the attributes is the instance the SCU proposes for
	// the new object; the SCP may instead assign its own and echo it in the response.
	suppliedUID, _ := step.GetString(dicom.TagSOPInstanceUID)

	conn := a.requestor.Conn()
	machine := a.requestor.Machine()
	opCtx, cancel := a.dimseContext(ctx)
	defer cancel()

	rq := CommandSet{
		CommandField:        CommandNCreateRQ,
		MessageID:           a.nextMessageID(),
		AffectedSOPClassUID: dicom.UID(modalityPerformedStepSOPClass),
		CommandDataSetType:  CommandDataSetPresent,
	}
	if suppliedUID != "" {
		rq.AffectedSOPInstanceUID = dicom.UID(suppliedUID)
	}
	if err := sendMessage(opCtx, conn, machine, pcID, rq, step, ts, a.sendCap()); err != nil {
		return "", Status{}, err
	}

	rsp, _, _, err := receiveMessage(opCtx, conn, machine, newMessageReassembler(ts))
	if err != nil {
		return "", Status{}, err
	}
	if rsp.CommandField != CommandNCreateRSP {
		return "", Status{}, &ProtocolError{
			State:  machine.CurrentState(),
			Detail: "expected an N-CREATE-RSP command field in the response",
		}
	}
	if !rsp.HasStatus {
		return "", Status{}, &ProtocolError{
			State:  machine.CurrentState(),
			Detail: "N-CREATE-RSP missing the mandatory Status element",
		}
	}

	instanceUID := rsp.AffectedSOPInstanceUID
	if instanceUID == "" {
		instanceUID = dicom.UID(suppliedUID)
	}
	return instanceUID, NewStatus(rsp.Status, ServiceClassProcedureStep), nil
}

// Set issues an N-SET that updates an existing Modality Performed Procedure Step, typically
// advancing it from IN PROGRESS to COMPLETED or DISCONTINUED (PS3.4 F.7.1). instanceUID is the SOP
// Instance UID returned by Create; it is carried as the Requested SOP Instance UID (0000,1001) — the
// reference pair the N-services use, distinct from the C-service Affected SOP Instance UID. attrs is
// the updated attribute set (carrying at least the new Performed Procedure Step Status).
//
// An empty instance UID, a nil attribute set, or no negotiated MPPS context is a fail-closed
// *ValidationError/*AssociationError returned before any wire I/O (PRD §9.2). The returned Status is
// meaningful only when the returned error is nil; a Failure-category status (for example, the step
// may no longer be updated) is in-band data the caller inspects, never laundered to success.
//
// No patient, procedure-step, or SOP instance value is ever logged or embedded in an error string
// (PRD §9.1).
func (m *MPPS) Set(ctx context.Context, instanceUID dicom.UID, attrs *dicom.DataSet) (Status, error) {
	a := m.assoc
	if a == nil || a.requestor == nil {
		return Status{}, &AssociationError{Kind: AssociationNotEstablished, Detail: "MPPS Set on an unestablished association"}
	}
	a.mu.Lock()
	released := a.released
	a.mu.Unlock()
	if released {
		return Status{}, &AssociationError{Kind: AssociationNotEstablished, Detail: "MPPS Set on a released association"}
	}
	if instanceUID == "" {
		return Status{}, &ValidationError{Detail: "MPPS Set requires the SOP Instance UID of the procedure step to update"}
	}
	if attrs == nil || attrs.Len() == 0 {
		return Status{}, &ValidationError{Detail: "MPPS Set requires a non-empty attribute set for the N-SET data set"}
	}

	pcID, ts, ok := a.contextForQuery(modalityPerformedStepSOPClass)
	if !ok {
		return Status{}, &AssociationError{
			Kind:   AssociationNotEstablished,
			Detail: "no accepted presentation context for the Modality Performed Procedure Step SOP Class",
		}
	}

	conn := a.requestor.Conn()
	machine := a.requestor.Machine()
	opCtx, cancel := a.dimseContext(ctx)
	defer cancel()

	rq := CommandSet{
		CommandField:            CommandNSetRQ,
		MessageID:               a.nextMessageID(),
		RequestedSOPClassUID:    dicom.UID(modalityPerformedStepSOPClass),
		RequestedSOPInstanceUID: instanceUID,
		CommandDataSetType:      CommandDataSetPresent,
	}
	if err := sendMessage(opCtx, conn, machine, pcID, rq, attrs, ts, a.sendCap()); err != nil {
		return Status{}, err
	}

	rsp, _, _, err := receiveMessage(opCtx, conn, machine, newMessageReassembler(ts))
	if err != nil {
		return Status{}, err
	}
	if rsp.CommandField != CommandNSetRSP {
		return Status{}, &ProtocolError{
			State:  machine.CurrentState(),
			Detail: "expected an N-SET-RSP command field in the response",
		}
	}
	if !rsp.HasStatus {
		return Status{}, &ProtocolError{
			State:  machine.CurrentState(),
			Detail: "N-SET-RSP missing the mandatory Status element",
		}
	}
	return NewStatus(rsp.Status, ServiceClassProcedureStep), nil
}
