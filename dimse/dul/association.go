package dul

import (
	"context"
	"fmt"
	"sync"

	"github.com/codeninja55/go-radx/dimse/pdu"
)

// Association represents a DICOM association
type Association struct {
	conn                   *Connection
	calledAETitle          string
	callingAETitle         string
	applicationContext     string
	presentationContexts   map[uint8]*PresentationContext
	maxPDULength           uint32
	implementationClassUID string
	implementationVersion  string
	mu                     sync.RWMutex
}

// PresentationContext represents a negotiated presentation context
type PresentationContext struct {
	ID             uint8
	AbstractSyntax string
	TransferSyntax string
	Result         uint8
	Accepted       bool
}

// NewAssociation creates a new association
func NewAssociation(conn *Connection, calledAE, callingAE string) *Association {
	return &Association{
		conn:                   conn,
		calledAETitle:          calledAE,
		callingAETitle:         callingAE,
		applicationContext:     "1.2.840.10008.3.1.1.1", // Default DICOM Application Context
		presentationContexts:   make(map[uint8]*PresentationContext),
		maxPDULength:           pdu.DefaultMaxPDULength,
		implementationClassUID: "1.2.840.12345.1.1", // Placeholder UID
		implementationVersion:  "GO-RADX_1.0",
	}
}

// RequestAssociation sends an A-ASSOCIATE-RQ and waits for response
func (a *Association) RequestAssociation(ctx context.Context, pcReqs []PresentationContextRQ) error {
	// Trigger A-ASSOCIATE request event
	action, err := a.conn.sm.ProcessEvent(AE3)
	if err != nil {
		return fmt.Errorf("state machine error: %w", err)
	}

	if action != ActionSendAssociateRQ {
		return fmt.Errorf("unexpected action: %v", action)
	}

	// Build A-ASSOCIATE-RQ PDU
	rq := &pdu.AssociateRQ{
		ProtocolVersion:    0x0001,
		CalledAETitle:      pdu.PadAETitle(a.calledAETitle),
		CallingAETitle:     pdu.PadAETitle(a.callingAETitle),
		ApplicationContext: a.applicationContext,
		UserInfo: pdu.UserInformation{
			MaxPDULength:           a.maxPDULength,
			ImplementationClassUID: a.implementationClassUID,
			ImplementationVersion:  a.implementationVersion,
		},
	}

	// Convert presentation contexts and build a map for later lookup
	pcMap := make(map[uint8]string) // ID -> AbstractSyntax
	for _, pcReq := range pcReqs {
		rq.PresentationContexts = append(rq.PresentationContexts, pdu.PresentationContextRQ{
			ID:               pcReq.ID,
			AbstractSyntax:   pcReq.AbstractSyntax,
			TransferSyntaxes: pcReq.TransferSyntaxes,
		})
		pcMap[pcReq.ID] = pcReq.AbstractSyntax
	}

	// Send A-ASSOCIATE-RQ
	if err := a.conn.SendPDU(ctx, rq); err != nil {
		return fmt.Errorf("send A-ASSOCIATE-RQ: %w", err)
	}

	// Wait for A-ASSOCIATE-AC or A-ASSOCIATE-RJ
	response, err := a.conn.ReadPDU(ctx)
	if err != nil {
		return fmt.Errorf("read association response: %w", err)
	}

	switch p := response.(type) {
	case *pdu.AssociateAC:
		// Process A-ASSOCIATE-AC
		_, err := a.conn.sm.ProcessEvent(AE6)
		if err != nil {
			return fmt.Errorf("state machine error: %w", err)
		}

		// Store negotiated presentation contexts
		a.mu.Lock()
		for _, pc := range p.PresentationContexts {
			// Match back to requested abstract syntax
			abstractSyntax := pcMap[pc.ID]
			a.presentationContexts[pc.ID] = &PresentationContext{
				ID:             pc.ID,
				AbstractSyntax: abstractSyntax,
				TransferSyntax: pc.TransferSyntax,
				Result:         pc.Result,
				Accepted:       pc.Result == pdu.PresentationContextAcceptance,
			}
		}
		a.maxPDULength = p.UserInfo.MaxPDULength
		a.mu.Unlock()

		a.conn.SetMaxPDULength(p.UserInfo.MaxPDULength)

		return nil

	case *pdu.AssociateRJ:
		_, _ = a.conn.sm.ProcessEvent(AE7)
		return fmt.Errorf("association rejected: result=%d source=%d reason=%d",
			p.Result, p.Source, p.Reason)

	default:
		return fmt.Errorf("unexpected PDU type: %T", response)
	}
}

// AcceptAssociation processes an A-ASSOCIATE-RQ and sends response
func (a *Association) AcceptAssociation(ctx context.Context, rq *pdu.AssociateRQ, supportedContexts map[string][]string) error {
	// Trigger A-ASSOCIATE indication event
	_, err := a.conn.sm.ProcessEvent(AE8)
	if err != nil {
		return fmt.Errorf("state machine error: %w", err)
	}

	// Store AE titles
	a.mu.Lock()
	a.calledAETitle = pdu.TrimAETitle(rq.CalledAETitle)
	a.callingAETitle = pdu.TrimAETitle(rq.CallingAETitle)
	a.applicationContext = rq.ApplicationContext
	a.mu.Unlock()

	// Negotiate presentation contexts
	var acContexts []pdu.PresentationContextAC
	for _, pcRQ := range rq.PresentationContexts {
		pc := a.negotiatePresentationContext(pcRQ, supportedContexts)
		acContexts = append(acContexts, pdu.PresentationContextAC{
			ID:             pc.ID,
			Result:         pc.Result,
			TransferSyntax: pc.TransferSyntax,
		})

		// Store accepted contexts
		if pc.Result == pdu.PresentationContextAcceptance {
			a.mu.Lock()
			a.presentationContexts[pc.ID] = pc
			a.mu.Unlock()
		}
	}

	// Build A-ASSOCIATE-AC
	ac := &pdu.AssociateAC{
		ProtocolVersion:      0x0001,
		CalledAETitle:        rq.CalledAETitle,
		CallingAETitle:       rq.CallingAETitle,
		ApplicationContext:   rq.ApplicationContext,
		PresentationContexts: acContexts,
		UserInfo: pdu.UserInformation{
			MaxPDULength:           a.maxPDULength,
			ImplementationClassUID: a.implementationClassUID,
			ImplementationVersion:  a.implementationVersion,
		},
	}

	// Trigger A-ASSOCIATE response (accept)
	action, err := a.conn.sm.ProcessEvent(AE4)
	if err != nil {
		return fmt.Errorf("state machine error: %w", err)
	}

	if action != ActionSendAssociateAC {
		return fmt.Errorf("unexpected action: %v", action)
	}

	// Send A-ASSOCIATE-AC
	if err := a.conn.SendPDU(ctx, ac); err != nil {
		return fmt.Errorf("send A-ASSOCIATE-AC: %w", err)
	}

	return nil
}

// Release performs graceful association release
func (a *Association) Release(ctx context.Context) error {
	// Trigger A-RELEASE request
	action, err := a.conn.sm.ProcessEvent(AE11)
	if err != nil {
		return fmt.Errorf("state machine error: %w", err)
	}

	if action != ActionSendReleaseRQ {
		return fmt.Errorf("unexpected action: %v", action)
	}

	// Send A-RELEASE-RQ
	if err := a.conn.SendPDU(ctx, &pdu.ReleaseRQ{}); err != nil {
		return fmt.Errorf("send A-RELEASE-RQ: %w", err)
	}

	// Wait for A-RELEASE-RP
	response, err := a.conn.ReadPDU(ctx)
	if err != nil {
		return fmt.Errorf("read release response: %w", err)
	}

	if _, ok := response.(*pdu.ReleaseRP); !ok {
		return fmt.Errorf("expected A-RELEASE-RP, got %T", response)
	}

	// Trigger A-RELEASE-RP received
	_, err = a.conn.sm.ProcessEvent(AE13)
	if err != nil {
		return fmt.Errorf("state machine error: %w", err)
	}

	// Close connection
	return a.conn.Close()
}

// Abort sends an A-ABORT and closes the connection
func (a *Association) Abort(ctx context.Context, source, reason uint8) error {
	// Trigger A-ABORT request
	_, err := a.conn.sm.ProcessEvent(AE15)
	if err != nil {
		return fmt.Errorf("state machine error: %w", err)
	}

	// Send A-ABORT
	abort := &pdu.Abort{
		Source: source,
		Reason: reason,
	}

	if err := a.conn.SendPDU(ctx, abort); err != nil {
		return fmt.Errorf("send A-ABORT: %w", err)
	}

	// Close connection
	return a.conn.Close()
}

// SendData sends P-DATA-TF PDU
func (a *Association) SendData(ctx context.Context, data *pdu.DataTF) error {
	// Trigger P-DATA request
	action, err := a.conn.sm.ProcessEvent(AE9)
	if err != nil {
		return fmt.Errorf("state machine error: %w", err)
	}

	if action != ActionSendData {
		return fmt.Errorf("unexpected action: %v", action)
	}

	return a.conn.SendPDU(ctx, data)
}

// GetPresentationContext returns the presentation context for the given ID
func (a *Association) GetPresentationContext(id uint8) (*PresentationContext, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	pc, ok := a.presentationContexts[id]
	return pc, ok
}

// FindPresentationContext finds an accepted presentation context by abstract syntax
func (a *Association) FindPresentationContext(abstractSyntax string) (*PresentationContext, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, pc := range a.presentationContexts {
		if pc.AbstractSyntax == abstractSyntax && pc.Accepted {
			return pc, true
		}
	}
	return nil, false
}

// negotiatePresentationContext negotiates a single presentation context
func (a *Association) negotiatePresentationContext(rq pdu.PresentationContextRQ, supported map[string][]string) *PresentationContext {
	pc := &PresentationContext{
		ID:             rq.ID,
		AbstractSyntax: rq.AbstractSyntax,
	}

	// Check if abstract syntax is supported
	supportedTS, abstractOK := supported[rq.AbstractSyntax]
	if !abstractOK {
		pc.Result = pdu.PresentationContextAbstractSyntaxNotSupported
		return pc
	}

	// Find matching transfer syntax
	for _, requestedTS := range rq.TransferSyntaxes {
		for _, supportTS := range supportedTS {
			if requestedTS == supportTS {
				pc.TransferSyntax = requestedTS
				pc.Result = pdu.PresentationContextAcceptance
				pc.Accepted = true
				return pc
			}
		}
	}

	// No matching transfer syntax
	pc.Result = pdu.PresentationContextTransferSyntaxesNotSupported
	return pc
}

// Connection returns the underlying connection
func (a *Association) Connection() *Connection {
	return a.conn
}

// CalledAETitle returns the called AE title
func (a *Association) CalledAETitle() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.calledAETitle
}

// CallingAETitle returns the calling AE title
func (a *Association) CallingAETitle() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.callingAETitle
}

// PresentationContextRQ represents a requested presentation context
type PresentationContextRQ struct {
	ID               uint8
	AbstractSyntax   string
	TransferSyntaxes []string
}
