package scp

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/dimse"
	"github.com/codeninja55/go-radx/dimse/dul"
	"github.com/codeninja55/go-radx/dimse/pdu"
)

// Server represents a DIMSE SCP server
type Server struct {
	config       Config
	listener     net.Listener
	associations map[*dul.Association]*associationHandler
	mu           sync.RWMutex
	activeConns  int32
	wg           sync.WaitGroup
	shutdownCh   chan struct{}
	shutdownOnce sync.Once
}

// Config holds SCP server configuration
type Config struct {
	AETitle           string
	ListenAddr        string
	MaxPDULength      uint32
	MaxAssociations   int
	SupportedContexts map[string][]string // abstract syntax -> transfer syntaxes

	// Service handlers
	EchoHandler  EchoHandler
	StoreHandler StoreHandler
	FindHandler  FindHandler
	GetHandler   GetHandler
	MoveHandler  MoveHandler
}

// Service handler interfaces

// EchoHandler handles C-ECHO requests
type EchoHandler interface {
	HandleEcho(ctx context.Context, req *EchoRequest) *EchoResponse
}

// EchoRequest represents a C-ECHO request
type EchoRequest struct {
	CallingAE string
	CalledAE  string
}

// EchoResponse represents a C-ECHO response
type EchoResponse struct {
	Status uint16
}

// StoreHandler handles C-STORE requests
type StoreHandler interface {
	HandleStore(ctx context.Context, req *StoreRequest) *StoreResponse
}

// StoreRequest represents a C-STORE request
type StoreRequest struct {
	CallingAE      string
	CalledAE       string
	SOPClassUID    string
	SOPInstanceUID string
	DataSet        *dicom.DataSet
}

// StoreResponse represents a C-STORE response
type StoreResponse struct {
	Status uint16
}

// FindHandler handles C-FIND requests
type FindHandler interface {
	HandleFind(ctx context.Context, req *FindRequest) *FindResponse
}

// FindRequest represents a C-FIND request
type FindRequest struct {
	CallingAE   string
	CalledAE    string
	SOPClassUID string
	Query       *dicom.DataSet
}

// FindResponse represents a C-FIND response
type FindResponse struct {
	Results []*dicom.DataSet
	Status  uint16
}

// GetHandler handles C-GET requests
type GetHandler interface {
	HandleGet(ctx context.Context, req *GetRequest) *GetResponse
}

// GetRequest represents a C-GET request
type GetRequest struct {
	CallingAE   string
	CalledAE    string
	SOPClassUID string
	Query       *dicom.DataSet
}

// GetResponse represents a C-GET response
type GetResponse struct {
	Instances []*dicom.DataSet
	Status    uint16
}

// MoveHandler handles C-MOVE requests
type MoveHandler interface {
	HandleMove(ctx context.Context, req *MoveRequest) *MoveResponse
}

// MoveRequest represents a C-MOVE request
type MoveRequest struct {
	CallingAE   string
	CalledAE    string
	SOPClassUID string
	Destination string
	Query       *dicom.DataSet
}

// MoveResponse represents a C-MOVE response
type MoveResponse struct {
	NumberOfCompletedSubOps uint16
	NumberOfFailedSubOps    uint16
	NumberOfWarningSubOps   uint16
	Status                  uint16
}

// NewServer creates a new SCP server
func NewServer(config Config) (*Server, error) {
	if config.MaxPDULength == 0 {
		config.MaxPDULength = 16384
	}
	if config.MaxAssociations == 0 {
		config.MaxAssociations = 10
	}

	return &Server{
		config:       config,
		associations: make(map[*dul.Association]*associationHandler),
		shutdownCh:   make(chan struct{}),
	}, nil
}

// Listen starts the server listening for connections
func (s *Server) Listen(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.config.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	s.listener = listener

	go s.acceptLoop(ctx)

	return nil
}

// acceptLoop accepts incoming connections
func (s *Server) acceptLoop(ctx context.Context) {
	for {
		select {
		case <-s.shutdownCh:
			return
		case <-ctx.Done():
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.shutdownCh:
				return
			default:
				continue
			}
		}

		// Check max associations
		if atomic.LoadInt32(&s.activeConns) >= int32(s.config.MaxAssociations) {
			//nolint:errcheck // Connection rejection, error not critical
			_ = conn.Close()
			continue
		}

		s.wg.Add(1)
		go s.handleConnection(ctx, conn)
	}
}

// handleConnection handles a single connection
func (s *Server) handleConnection(ctx context.Context, netConn net.Conn) {
	defer s.wg.Done()
	//nolint:errcheck // Connection close in defer
	defer func() { _ = netConn.Close() }()

	atomic.AddInt32(&s.activeConns, 1)
	defer atomic.AddInt32(&s.activeConns, -1)

	// Create DUL connection
	conn := dul.NewConnection(netConn)
	conn.SetMaxPDULength(s.config.MaxPDULength)

	// Trigger transport connection indication event (AE-2)
	// This transitions state machine from Sta1 (Idle) to Sta2 (Transport Open)
	if err := conn.TriggerTransportIndication(ctx); err != nil {
		return
	}

	// Wait for A-ASSOCIATE-RQ
	pduMsg, err := conn.ReadPDU(ctx)
	if err != nil {
		return
	}

	assocRQ, ok := pduMsg.(*pdu.AssociateRQ)
	if !ok {
		return
	}

	// Create association
	callingAE := pdu.TrimAETitle(assocRQ.CallingAETitle)
	assoc := dul.NewAssociation(conn, s.config.AETitle, callingAE)

	// Accept association
	if err := assoc.AcceptAssociation(ctx, assocRQ, s.config.SupportedContexts); err != nil {
		return
	}

	// Create association handler
	handler := &associationHandler{
		server:      s,
		assoc:       assoc,
		conn:        conn,
		reassembler: dimse.NewMessageReassembler(),
	}

	s.mu.Lock()
	s.associations[assoc] = handler
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.associations, assoc)
		s.mu.Unlock()
	}()

	// Handle messages
	handler.handleMessages(ctx)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	var err error
	s.shutdownOnce.Do(func() {
		close(s.shutdownCh)

		if s.listener != nil {
			//nolint:errcheck // Listener close during shutdown, error not critical
			_ = s.listener.Close()
		}

		// Wait for active connections with timeout
		done := make(chan struct{})
		go func() {
			s.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			err = ctx.Err()
		}
	})

	return err
}

// associationHandler handles messages for a single association
type associationHandler struct {
	server      *Server
	assoc       *dul.Association
	conn        *dul.Connection
	reassembler *dimse.MessageReassembler
}

// handleMessages handles incoming DIMSE messages
func (h *associationHandler) handleMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Read PDU
		pduMsg, err := h.conn.ReadPDU(ctx)
		if err != nil {
			return
		}

		// Handle different PDU types
		switch p := pduMsg.(type) {
		case *pdu.DataTF:
			if err := h.handleDataPDU(ctx, p); err != nil {
				return
			}

		case *pdu.ReleaseRQ:
			//nolint:errcheck // State machine event during release
			// Trigger A-RELEASE indication (AE-12)
			_, _ = h.conn.StateMachine().ProcessEvent(dul.AE12)

			//nolint:errcheck // State machine event during release
			// Send A-RELEASE-RP (trigger AE-14)
			_, _ = h.conn.StateMachine().ProcessEvent(dul.AE14)
			//nolint:errcheck // PDU send during release
			_ = h.conn.SendPDU(ctx, &pdu.ReleaseRP{})
			return

		case *pdu.Abort:
			// Association aborted
			return
		}
	}
}

// handleDataPDU handles P-DATA-TF PDUs
func (h *associationHandler) handleDataPDU(ctx context.Context, dataPDU *pdu.DataTF) error {
	// Reassemble message
	msg, err := h.reassembler.AddPDU(dataPDU)
	if err != nil {
		return err
	}

	// Message not yet complete
	if msg == nil {
		return nil
	}

	// Dispatch based on command field
	switch msg.CommandSet.CommandField {
	case dimse.CommandCEchoRQ:
		return h.handleCEcho(ctx, msg)
	case dimse.CommandCStoreRQ:
		return h.handleCStore(ctx, msg)
	case dimse.CommandCFindRQ:
		return h.handleCFind(ctx, msg)
	case dimse.CommandCGetRQ:
		return h.handleCGet(ctx, msg)
	case dimse.CommandCMoveRQ:
		return h.handleCMove(ctx, msg)
	default:
		return fmt.Errorf("unsupported command: 0x%04X", msg.CommandSet.CommandField)
	}
}

// handleCEcho handles C-ECHO-RQ
func (h *associationHandler) handleCEcho(ctx context.Context, msg *dimse.Message) error {
	// Call handler if configured
	status := dimse.StatusSuccess
	if h.server.config.EchoHandler != nil {
		req := &EchoRequest{
			CallingAE: h.assoc.CallingAETitle(),
			CalledAE:  h.assoc.CalledAETitle(),
		}
		resp := h.server.config.EchoHandler.HandleEcho(ctx, req)
		status = resp.Status
	}

	// Create response
	rsp := &dimse.CommandSet{
		CommandField:              dimse.CommandCEchoRSP,
		MessageIDBeingRespondedTo: msg.CommandSet.MessageID,
		CommandDataSetType:        dimse.DataSetNotPresent,
		Status:                    status,
		AffectedSOPClassUID:       msg.CommandSet.AffectedSOPClassUID,
	}

	return h.sendResponse(ctx, rsp, nil, msg.PresentationContextID)
}

// handleCStore handles C-STORE-RQ
func (h *associationHandler) handleCStore(ctx context.Context, msg *dimse.Message) error {
	// Call handler if configured
	status := dimse.StatusSuccess
	if h.server.config.StoreHandler != nil {
		req := &StoreRequest{
			CallingAE:      h.assoc.CallingAETitle(),
			CalledAE:       h.assoc.CalledAETitle(),
			SOPClassUID:    msg.CommandSet.AffectedSOPClassUID,
			SOPInstanceUID: msg.CommandSet.AffectedSOPInstanceUID,
			DataSet:        msg.DataSet,
		}
		resp := h.server.config.StoreHandler.HandleStore(ctx, req)
		status = resp.Status
	}

	// Create response
	rsp := &dimse.CommandSet{
		CommandField:              dimse.CommandCStoreRSP,
		MessageIDBeingRespondedTo: msg.CommandSet.MessageID,
		CommandDataSetType:        dimse.DataSetNotPresent,
		Status:                    status,
		AffectedSOPClassUID:       msg.CommandSet.AffectedSOPClassUID,
		AffectedSOPInstanceUID:    msg.CommandSet.AffectedSOPInstanceUID,
	}

	return h.sendResponse(ctx, rsp, nil, msg.PresentationContextID)
}

// handleCFind handles C-FIND-RQ
func (h *associationHandler) handleCFind(ctx context.Context, msg *dimse.Message) error {
	// Call handler if configured
	var results []*dicom.DataSet
	status := dimse.StatusSuccess

	if h.server.config.FindHandler != nil {
		req := &FindRequest{
			CallingAE:   h.assoc.CallingAETitle(),
			CalledAE:    h.assoc.CalledAETitle(),
			SOPClassUID: msg.CommandSet.AffectedSOPClassUID,
			Query:       msg.DataSet,
		}
		resp := h.server.config.FindHandler.HandleFind(ctx, req)
		results = resp.Results
		status = resp.Status
	}

	// Send pending responses with results
	for _, result := range results {
		pendingRsp := &dimse.CommandSet{
			CommandField:              dimse.CommandCFindRSP,
			MessageIDBeingRespondedTo: msg.CommandSet.MessageID,
			CommandDataSetType:        dimse.DataSetPresent,
			Status:                    dimse.StatusPending,
			AffectedSOPClassUID:       msg.CommandSet.AffectedSOPClassUID,
		}

		if err := h.sendResponse(ctx, pendingRsp, result, msg.PresentationContextID); err != nil {
			return err
		}
	}

	// Send final response
	finalRsp := &dimse.CommandSet{
		CommandField:              dimse.CommandCFindRSP,
		MessageIDBeingRespondedTo: msg.CommandSet.MessageID,
		CommandDataSetType:        dimse.DataSetNotPresent,
		Status:                    status,
		AffectedSOPClassUID:       msg.CommandSet.AffectedSOPClassUID,
	}

	return h.sendResponse(ctx, finalRsp, nil, msg.PresentationContextID)
}

// handleCGet handles C-GET-RQ
func (h *associationHandler) handleCGet(ctx context.Context, msg *dimse.Message) error {
	// Call handler if configured
	var instances []*dicom.DataSet
	status := dimse.StatusSuccess

	if h.server.config.GetHandler != nil {
		req := &GetRequest{
			CallingAE:   h.assoc.CallingAETitle(),
			CalledAE:    h.assoc.CalledAETitle(),
			SOPClassUID: msg.CommandSet.AffectedSOPClassUID,
			Query:       msg.DataSet,
		}
		resp := h.server.config.GetHandler.HandleGet(ctx, req)
		instances = resp.Instances
		status = resp.Status
	}

	// Send C-STORE sub-operations for each instance
	completed := uint16(0)
	failed := uint16(0)

	for _, instance := range instances {
		// Get SOP Class and Instance UID from dataset
		sopClassUID, err := getStringFromDataSet(instance, TagSOPClassUID)
		if err != nil {
			failed++
			continue
		}
		sopInstanceUID, err := getStringFromDataSet(instance, TagSOPInstanceUID)
		if err != nil {
			failed++
			continue
		}

		// Find presentation context for this SOP class
		pc, ok := h.assoc.FindPresentationContext(sopClassUID)
		if !ok {
			failed++
			continue
		}

		// Send C-STORE-RQ
		storeCmd := &dimse.CommandSet{
			CommandField:           dimse.CommandCStoreRQ,
			MessageID:              0, // Sub-operation, use 0
			Priority:               dimse.PriorityMedium,
			CommandDataSetType:     dimse.DataSetPresent,
			AffectedSOPClassUID:    sopClassUID,
			AffectedSOPInstanceUID: sopInstanceUID,
		}

		if err := h.sendResponse(ctx, storeCmd, instance, pc.ID); err != nil {
			failed++
			continue
		}

		// Wait for C-STORE-RSP
		storePDU, err := h.conn.ReadPDU(ctx)
		if err != nil {
			failed++
			continue
		}

		dataPDU, ok := storePDU.(*pdu.DataTF)
		if !ok {
			failed++
			continue
		}

		storeMsg, err := h.reassembler.AddPDU(dataPDU)
		if err != nil || storeMsg == nil {
			failed++
			continue
		}

		if storeMsg.CommandSet.Status == dimse.StatusSuccess {
			completed++
		} else {
			failed++
		}

		// Send pending C-GET-RSP
		pendingRsp := &dimse.CommandSet{
			CommandField:              dimse.CommandCGetRSP,
			MessageIDBeingRespondedTo: msg.CommandSet.MessageID,
			CommandDataSetType:        dimse.DataSetNotPresent,
			Status:                    dimse.StatusPending,
			NumberOfRemainingSubOps:   uint16(len(instances) - int(completed) - int(failed)),
			NumberOfCompletedSubOps:   completed,
			NumberOfFailedSubOps:      failed,
		}

		if err := h.sendResponse(ctx, pendingRsp, nil, msg.PresentationContextID); err != nil {
			return err
		}
	}

	// Send final response
	finalRsp := &dimse.CommandSet{
		CommandField:              dimse.CommandCGetRSP,
		MessageIDBeingRespondedTo: msg.CommandSet.MessageID,
		CommandDataSetType:        dimse.DataSetNotPresent,
		Status:                    status,
		NumberOfCompletedSubOps:   completed,
		NumberOfFailedSubOps:      failed,
	}

	return h.sendResponse(ctx, finalRsp, nil, msg.PresentationContextID)
}

// handleCMove handles C-MOVE-RQ
func (h *associationHandler) handleCMove(ctx context.Context, msg *dimse.Message) error {
	// Call handler if configured
	status := dimse.StatusSuccess
	var completed, failed, warning uint16

	if h.server.config.MoveHandler != nil {
		req := &MoveRequest{
			CallingAE:   h.assoc.CallingAETitle(),
			CalledAE:    h.assoc.CalledAETitle(),
			SOPClassUID: msg.CommandSet.AffectedSOPClassUID,
			Destination: msg.CommandSet.MoveDestination,
			Query:       msg.DataSet,
		}
		resp := h.server.config.MoveHandler.HandleMove(ctx, req)
		status = resp.Status
		completed = resp.NumberOfCompletedSubOps
		failed = resp.NumberOfFailedSubOps
		warning = resp.NumberOfWarningSubOps
	}

	// Send final response
	finalRsp := &dimse.CommandSet{
		CommandField:              dimse.CommandCMoveRSP,
		MessageIDBeingRespondedTo: msg.CommandSet.MessageID,
		CommandDataSetType:        dimse.DataSetNotPresent,
		Status:                    status,
		NumberOfCompletedSubOps:   completed,
		NumberOfFailedSubOps:      failed,
		NumberOfWarningSubOps:     warning,
	}

	return h.sendResponse(ctx, finalRsp, nil, msg.PresentationContextID)
}

// sendResponse sends a DIMSE response message
func (h *associationHandler) sendResponse(ctx context.Context, cmd *dimse.CommandSet, ds *dicom.DataSet, pcID uint8) error {
	msg := &dimse.Message{
		CommandSet:            cmd,
		DataSet:               ds,
		PresentationContextID: pcID,
	}

	pdus, err := msg.Encode(h.conn.GetMaxPDULength())
	if err != nil {
		return err
	}

	for _, pdu := range pdus {
		if err := h.assoc.SendData(ctx, pdu); err != nil {
			return err
		}
	}

	return nil
}
