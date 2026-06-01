package dimse

import (
	"context"

	"github.com/codeninja55/go-radx/dicom"
)

// storeMessageID is the Message ID used for a single C-STORE request. A Store is one
// request/response with no concurrent operations on the association, so a fixed non-zero ID
// suffices (PS3.7 §9.1.1); the allocator for concurrent operations lands with query/retrieve (M3).
const storeMessageID uint16 = 1

// tagSOPClassUID and tagSOPInstanceUID identify the composite SOP Instance the C-STORE carries,
// read from the dataset to populate the command set's Affected SOP Class/Instance UID.
var (
	tagSOPClassUID    = dicom.NewTag(0x0008, 0x0016)
	tagSOPInstanceUID = dicom.NewTag(0x0008, 0x0018)
)

// storeConfig holds the resolved StoreOptions.
type storeConfig struct {
	priority         Priority
	moveOriginatorAE AETitle
	moveOriginatorID uint16
	hasMoveOrigin    bool
}

// StoreOption configures a C-STORE.
type StoreOption func(*storeConfig)

// WithStorePriority sets the C-STORE operation priority (default medium).
func WithStorePriority(p Priority) StoreOption {
	return func(c *storeConfig) { c.priority = p }
}

// WithMoveOriginator propagates the Move Originator AE Title and Message ID for a C-STORE issued as
// a sub-operation of a C-MOVE/C-GET (PS3.7 §9.1.1).
func WithMoveOriginator(aet AETitle, msgID uint16) StoreOption {
	return func(c *storeConfig) {
		c.moveOriginatorAE = aet
		c.moveOriginatorID = msgID
		c.hasMoveOrigin = true
	}
}

// Store transmits one dataset via C-STORE (PS3.4 B.2). The presentation context is selected from
// the dataset's SOP Class UID (0008,0016) and a negotiated transfer syntax; if no accepted context
// matches, Store returns a typed *AssociationError and transmits NOTHING — it never reports
// success on work it did not do (PRD §9.2 fail-closed, the rule the prototype's store violated). A
// Failure-category status is data the caller inspects with status.IsFailure(), not a Go error; the
// returned error reports a transport, association, or protocol fault. It guards against an
// unestablished or released association with a typed error, never a panic (Codex DIMSE-017).
func (a *Association) Store(ctx context.Context, ds *dicom.DataSet, opts ...StoreOption) (Status, error) {
	if a == nil || a.requestor == nil {
		return Status{}, &AssociationError{Kind: AssociationNotEstablished, Detail: "Store on an unestablished association"}
	}
	a.mu.Lock()
	released := a.released
	a.mu.Unlock()
	if released {
		return Status{}, &AssociationError{Kind: AssociationNotEstablished, Detail: "Store on a released association"}
	}

	if ds == nil {
		return Status{}, &ValidationError{Detail: "Store requires a non-nil dataset"}
	}
	sopClass, ok := ds.GetString(tagSOPClassUID)
	if !ok || sopClass == "" {
		return Status{}, &ValidationError{Detail: "dataset has no SOP Class UID (0008,0016) to select a presentation context"}
	}
	sopInstance, _ := ds.GetString(tagSOPInstanceUID)

	cfg := storeConfig{priority: PriorityMedium}
	for _, opt := range opts {
		opt(&cfg)
	}

	// Select the accepted context for this SOP Class. No match is fail-closed: return before any
	// PDU is written (PRD §9.2).
	pcID, ts, ok := a.contextForStorage(dicom.SOPClassUID(sopClass))
	if !ok {
		return Status{}, &AssociationError{
			Kind:   AssociationNotEstablished,
			Detail: "no accepted presentation context for SOP Class " + sopClass,
		}
	}

	conn := a.requestor.Conn()
	m := a.requestor.Machine()

	opCtx, cancel := a.dimseContext(ctx)
	defer cancel()

	rq := CommandSet{
		CommandField:           CommandCStoreRQ,
		MessageID:              storeMessageID,
		AffectedSOPClassUID:    dicom.UID(sopClass),
		AffectedSOPInstanceUID: dicom.UID(sopInstance),
		HasPriority:            true,
		Priority:               cfg.priority,
		CommandDataSetType:     CommandDataSetPresent,
	}
	if cfg.hasMoveOrigin {
		rq.HasMoveOriginator = true
		rq.MoveOriginatorAETitle = cfg.moveOriginatorAE
		rq.MoveOriginatorMessageID = cfg.moveOriginatorID
	}
	if err := sendMessage(opCtx, conn, m, pcID, rq, ds, ts, a.sendCap()); err != nil {
		return Status{}, err
	}

	rsp, _, _, err := receiveMessage(opCtx, conn, m, newMessageReassembler(ts))
	if err != nil {
		return Status{}, err
	}
	if rsp.CommandField != CommandCStoreRSP {
		return Status{}, &ProtocolError{
			State:  m.CurrentState(),
			Detail: "expected a C-STORE-RSP command field in the response",
		}
	}
	// The Status element is mandatory in a C-STORE-RSP (PS3.7 §9.1.1.2); a response without it is
	// malformed, not an implicit success (PRD §9.2).
	if !rsp.HasStatus {
		return Status{}, &ProtocolError{
			State:  m.CurrentState(),
			Detail: "C-STORE-RSP missing the mandatory Status element",
		}
	}
	return NewStatus(rsp.Status, ServiceClassStorage), nil
}

// contextForStorage returns the accepted presentation context ID and its negotiated transfer
// syntax for the given SOP Class, or false when none was accepted (so Store fails closed rather
// than guessing a context). When several contexts were accepted for the same SOP Class the first
// is chosen, matching the proposal order.
func (a *Association) contextForStorage(sopClass dicom.SOPClassUID) (uint8, dicom.TransferSyntax, bool) {
	for _, pc := range a.accepted {
		if pc.Result != ContextAccepted {
			continue
		}
		if pc.AbstractSyntax != sopClass {
			continue
		}
		if len(pc.TransferSyntaxes) == 0 {
			continue
		}
		return pc.ID, pc.TransferSyntaxes[0], true
	}
	return 0, "", false
}
