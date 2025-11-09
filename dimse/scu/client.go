package scu

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/dimse"
	"github.com/codeninja55/go-radx/dimse/dul"
	"github.com/codeninja55/go-radx/dimse/pdu"
)

// Client represents a DIMSE SCU client
type Client struct {
	config      Config
	conn        *dul.Connection
	assoc       *dul.Association
	messageID   uint32
	reassembler *dimse.MessageReassembler
}

// Config holds SCU client configuration
type Config struct {
	CallingAETitle       string
	CalledAETitle        string
	RemoteAddr           string
	MaxPDULength         uint32
	PresentationContexts []dul.PresentationContextRQ
}

// NewClient creates a new SCU client
func NewClient(config Config) *Client {
	if config.MaxPDULength == 0 {
		config.MaxPDULength = 16384
	}
	return &Client{
		config:      config,
		messageID:   0,
		reassembler: dimse.NewMessageReassembler(),
	}
}

// Connect establishes a connection and association
func (c *Client) Connect(ctx context.Context) error {
	// Establish TCP connection
	conn, err := dul.Dial(ctx, "tcp", c.config.RemoteAddr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	c.conn = conn

	// Create association
	c.assoc = dul.NewAssociation(conn, c.config.CalledAETitle, c.config.CallingAETitle)

	// Request association
	if err := c.assoc.RequestAssociation(ctx, c.config.PresentationContexts); err != nil {
		//nolint:errcheck // Connection cleanup in error path
		c.conn.Close()
		return fmt.Errorf("request association: %w", err)
	}

	return nil
}

// Close closes the association and connection
func (c *Client) Close(ctx context.Context) error {
	if c.assoc != nil {
		return c.assoc.Release(ctx)
	}
	return nil
}

// Echo performs C-ECHO
func (c *Client) Echo(ctx context.Context) error {
	// Create C-ECHO-RQ command
	cmd := &dimse.CommandSet{
		CommandField:        dimse.CommandCEchoRQ,
		MessageID:           c.nextMessageID(),
		CommandDataSetType:  dimse.DataSetNotPresent,
		AffectedSOPClassUID: "1.2.840.10008.1.1", // Verification SOP Class
	}

	// Find presentation context
	pc, ok := c.assoc.FindPresentationContext("1.2.840.10008.1.1")
	if !ok {
		return fmt.Errorf("no presentation context for verification")
	}

	// Send message
	msg := &dimse.Message{
		CommandSet:            cmd,
		PresentationContextID: pc.ID,
	}

	if err := c.sendMessage(ctx, msg); err != nil {
		return fmt.Errorf("send C-ECHO-RQ: %w", err)
	}

	// Receive response
	rsp, err := c.receiveMessage(ctx)
	if err != nil {
		return fmt.Errorf("receive C-ECHO-RSP: %w", err)
	}

	if rsp.CommandSet.Status != dimse.StatusSuccess {
		return fmt.Errorf("C-ECHO failed: status=0x%04X", rsp.CommandSet.Status)
	}

	return nil
}

// Store performs C-STORE
func (c *Client) Store(ctx context.Context, ds *dicom.DataSet, sopClassUID, sopInstanceUID string) error {
	// Create C-STORE-RQ command
	cmd := &dimse.CommandSet{
		CommandField:           dimse.CommandCStoreRQ,
		MessageID:              c.nextMessageID(),
		Priority:               dimse.PriorityMedium,
		CommandDataSetType:     dimse.DataSetPresent,
		AffectedSOPClassUID:    sopClassUID,
		AffectedSOPInstanceUID: sopInstanceUID,
	}

	// Find presentation context
	pc, ok := c.assoc.FindPresentationContext(sopClassUID)
	if !ok {
		return fmt.Errorf("no presentation context for %s", sopClassUID)
	}

	// Send message with dataset
	msg := &dimse.Message{
		CommandSet:            cmd,
		DataSet:               ds,
		PresentationContextID: pc.ID,
	}

	if err := c.sendMessage(ctx, msg); err != nil {
		return fmt.Errorf("send C-STORE-RQ: %w", err)
	}

	// Receive response
	rsp, err := c.receiveMessage(ctx)
	if err != nil {
		return fmt.Errorf("receive C-STORE-RSP: %w", err)
	}

	if rsp.CommandSet.Status != dimse.StatusSuccess {
		return fmt.Errorf("C-STORE failed: status=0x%04X", rsp.CommandSet.Status)
	}

	return nil
}

// Find performs C-FIND and returns results via callback
func (c *Client) Find(ctx context.Context, queryLevel, sopClassUID string, query *dicom.DataSet, callback func(*dicom.DataSet) error) error {
	// Create C-FIND-RQ command
	cmd := &dimse.CommandSet{
		CommandField:        dimse.CommandCFindRQ,
		MessageID:           c.nextMessageID(),
		Priority:            dimse.PriorityMedium,
		CommandDataSetType:  dimse.DataSetPresent,
		AffectedSOPClassUID: sopClassUID,
	}

	// Find presentation context
	pc, ok := c.assoc.FindPresentationContext(sopClassUID)
	if !ok {
		return fmt.Errorf("no presentation context for %s", sopClassUID)
	}

	// Send query
	msg := &dimse.Message{
		CommandSet:            cmd,
		DataSet:               query,
		PresentationContextID: pc.ID,
	}

	if err := c.sendMessage(ctx, msg); err != nil {
		return fmt.Errorf("send C-FIND-RQ: %w", err)
	}

	// Receive responses
	for {
		rsp, err := c.receiveMessage(ctx)
		if err != nil {
			return fmt.Errorf("receive C-FIND-RSP: %w", err)
		}

		status := rsp.CommandSet.Status

		// Handle pending responses with results
		if status == dimse.StatusPending {
			if rsp.DataSet != nil && callback != nil {
				if err := callback(rsp.DataSet); err != nil {
					return err
				}
			}
			continue
		}

		// Success - end of results
		if status == dimse.StatusSuccess {
			return nil
		}

		// Error
		return fmt.Errorf("C-FIND failed: status=0x%04X", status)
	}
}

// Get performs C-GET and stores retrieved instances
func (c *Client) Get(ctx context.Context, sopClassUID string, query *dicom.DataSet, storeHandler func(*dicom.DataSet) error) error {
	// Create C-GET-RQ command
	cmd := &dimse.CommandSet{
		CommandField:        dimse.CommandCGetRQ,
		MessageID:           c.nextMessageID(),
		Priority:            dimse.PriorityMedium,
		CommandDataSetType:  dimse.DataSetPresent,
		AffectedSOPClassUID: sopClassUID,
	}

	// Find presentation context
	pc, ok := c.assoc.FindPresentationContext(sopClassUID)
	if !ok {
		return fmt.Errorf("no presentation context for %s", sopClassUID)
	}

	// Send query
	msg := &dimse.Message{
		CommandSet:            cmd,
		DataSet:               query,
		PresentationContextID: pc.ID,
	}

	if err := c.sendMessage(ctx, msg); err != nil {
		return fmt.Errorf("send C-GET-RQ: %w", err)
	}

	// Handle C-GET responses and C-STORE sub-operations
	for {
		rsp, err := c.receiveMessage(ctx)
		if err != nil {
			return fmt.Errorf("receive C-GET-RSP: %w", err)
		}

		// Check if it's a C-STORE request (sub-operation)
		if rsp.CommandSet.CommandField == dimse.CommandCStoreRQ {
			// Handle C-STORE
			if storeHandler != nil && rsp.DataSet != nil {
				if err := storeHandler(rsp.DataSet); err != nil {
					return fmt.Errorf("store handler: %w", err)
				}
			}

			// Send C-STORE-RSP
			storeRsp := &dimse.CommandSet{
				CommandField:              dimse.CommandCStoreRSP,
				MessageIDBeingRespondedTo: rsp.CommandSet.MessageID,
				CommandDataSetType:        dimse.DataSetNotPresent,
				Status:                    dimse.StatusSuccess,
				AffectedSOPClassUID:       rsp.CommandSet.AffectedSOPClassUID,
				AffectedSOPInstanceUID:    rsp.CommandSet.AffectedSOPInstanceUID,
			}

			storeRspMsg := &dimse.Message{
				CommandSet:            storeRsp,
				PresentationContextID: rsp.PresentationContextID,
			}

			if err := c.sendMessage(ctx, storeRspMsg); err != nil {
				return fmt.Errorf("send C-STORE-RSP: %w", err)
			}
			continue
		}

		// C-GET-RSP
		status := rsp.CommandSet.Status
		if status == dimse.StatusPending {
			continue
		}

		if status == dimse.StatusSuccess {
			return nil
		}

		return fmt.Errorf("C-GET failed: status=0x%04X", status)
	}
}

// Move performs C-MOVE
func (c *Client) Move(ctx context.Context, sopClassUID, destination string, query *dicom.DataSet) error {
	// Create C-MOVE-RQ command
	cmd := &dimse.CommandSet{
		CommandField:        dimse.CommandCMoveRQ,
		MessageID:           c.nextMessageID(),
		Priority:            dimse.PriorityMedium,
		CommandDataSetType:  dimse.DataSetPresent,
		AffectedSOPClassUID: sopClassUID,
		MoveDestination:     destination,
	}

	// Find presentation context
	pc, ok := c.assoc.FindPresentationContext(sopClassUID)
	if !ok {
		return fmt.Errorf("no presentation context for %s", sopClassUID)
	}

	// Send query
	msg := &dimse.Message{
		CommandSet:            cmd,
		DataSet:               query,
		PresentationContextID: pc.ID,
	}

	if err := c.sendMessage(ctx, msg); err != nil {
		return fmt.Errorf("send C-MOVE-RQ: %w", err)
	}

	// Receive responses
	for {
		rsp, err := c.receiveMessage(ctx)
		if err != nil {
			return fmt.Errorf("receive C-MOVE-RSP: %w", err)
		}

		status := rsp.CommandSet.Status

		if status == dimse.StatusPending {
			continue
		}

		if status == dimse.StatusSuccess {
			return nil
		}

		return fmt.Errorf("C-MOVE failed: status=0x%04X", status)
	}
}

// Helper methods
func (c *Client) nextMessageID() uint16 {
	id := atomic.AddUint32(&c.messageID, 1)
	if id > 65535 {
		atomic.StoreUint32(&c.messageID, 1)
		return 1
	}
	return uint16(id)
}

func (c *Client) sendMessage(ctx context.Context, msg *dimse.Message) error {
	pdus, err := msg.Encode(c.conn.GetMaxPDULength())
	if err != nil {
		return err
	}

	for _, pdu := range pdus {
		if err := c.assoc.SendData(ctx, pdu); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) receiveMessage(ctx context.Context) (*dimse.Message, error) {
	for {
		pduItem, err := c.conn.ReadPDU(ctx)
		if err != nil {
			return nil, err
		}

		dataPDU, ok := pduItem.(*pdu.DataTF)
		if !ok {
			return nil, fmt.Errorf("expected P-DATA-TF, got %T", pduItem)
		}

		msg, err := c.reassembler.AddPDU(dataPDU)
		if err != nil {
			return nil, err
		}

		if msg != nil {
			return msg, nil
		}
	}
}
