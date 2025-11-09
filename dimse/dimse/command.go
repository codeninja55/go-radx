package dimse

import (
	"fmt"
	"strings"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicom/element"
	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/value"
	"github.com/codeninja55/go-radx/dicom/vr"
)

// Command field values (DICOM Part 7)
const (
	CommandCStoreRQ  uint16 = 0x0001
	CommandCStoreRSP uint16 = 0x8001
	CommandCEchoRQ   uint16 = 0x0030
	CommandCEchoRSP  uint16 = 0x8030
	CommandCFindRQ   uint16 = 0x0020
	CommandCFindRSP  uint16 = 0x8020
	CommandCGetRQ    uint16 = 0x0010
	CommandCGetRSP   uint16 = 0x8010
	CommandCMoveRQ   uint16 = 0x0021
	CommandCMoveRSP  uint16 = 0x8021
	CommandCCancelRQ uint16 = 0x0FFF
)

// Status codes
const (
	StatusSuccess                     uint16 = 0x0000
	StatusPending                     uint16 = 0xFF00
	StatusPendingWarning              uint16 = 0xFF01
	StatusCancel                      uint16 = 0xFE00
	StatusAttributeListError          uint16 = 0x0107
	StatusAttributeValueOutOfRange    uint16 = 0x0116
	StatusSOPClassNotSupported        uint16 = 0x0122
	StatusClassInstanceConflict       uint16 = 0x0119
	StatusDuplicateSOPInstance        uint16 = 0x0111
	StatusResourceLimitation          uint16 = 0xA700
	StatusOutOfResources              uint16 = 0xA900
	StatusDataSetDoesNotMatchSOPClass uint16 = 0xA900
	StatusProcessingFailure           uint16 = 0xC000
	StatusMoveDestinationUnknown      uint16 = 0xA801
)

// Command data set type values
const (
	DataSetPresent    uint16 = 0x0000
	DataSetNotPresent uint16 = 0x0101
)

// Priority values
const (
	PriorityLow    uint16 = 0x0002
	PriorityMedium uint16 = 0x0000
	PriorityHigh   uint16 = 0x0001
)

// CommandSet represents a DIMSE command
type CommandSet struct {
	CommandField              uint16
	MessageID                 uint16
	MessageIDBeingRespondedTo uint16
	AffectedSOPClassUID       string
	AffectedSOPInstanceUID    string
	RequestedSOPClassUID      string
	RequestedSOPInstanceUID   string
	Priority                  uint16
	CommandDataSetType        uint16
	Status                    uint16
	NumberOfRemainingSubOps   uint16
	NumberOfCompletedSubOps   uint16
	NumberOfFailedSubOps      uint16
	NumberOfWarningSubOps     uint16
	MoveDestination           string
	MoveOriginatorAETitle     string
	MoveOriginatorMessageID   uint16
}

// ToDataSet converts a CommandSet to a DICOM dataset
func (cs *CommandSet) ToDataSet() (*dicom.DataSet, error) {
	ds := dicom.NewDataSet()

	// Command Field (0000,0100) - US
	if err := addUInt16Element(ds, tag.New(0x0000, 0x0100), cs.CommandField); err != nil {
		return nil, err
	}

	// Message ID (0000,0110) - US (for requests)
	if cs.MessageID != 0 {
		if err := addUInt16Element(ds, tag.New(0x0000, 0x0110), cs.MessageID); err != nil {
			return nil, err
		}
	}

	// Message ID Being Responded To (0000,0120) - US (for responses)
	if cs.MessageIDBeingRespondedTo != 0 {
		if err := addUInt16Element(ds, tag.New(0x0000, 0x0120), cs.MessageIDBeingRespondedTo); err != nil {
			return nil, err
		}
	}

	// Affected SOP Class UID (0000,0002) - UI
	if cs.AffectedSOPClassUID != "" {
		if err := addStringElement(ds, tag.New(0x0000, 0x0002), cs.AffectedSOPClassUID); err != nil {
			return nil, err
		}
	}

	// Affected SOP Instance UID (0000,1000) - UI
	if cs.AffectedSOPInstanceUID != "" {
		if err := addStringElement(ds, tag.New(0x0000, 0x1000), cs.AffectedSOPInstanceUID); err != nil {
			return nil, err
		}
	}

	// Requested SOP Class UID (0000,0003) - UI
	if cs.RequestedSOPClassUID != "" {
		if err := addStringElement(ds, tag.New(0x0000, 0x0003), cs.RequestedSOPClassUID); err != nil {
			return nil, err
		}
	}

	// Requested SOP Instance UID (0000,1001) - UI
	if cs.RequestedSOPInstanceUID != "" {
		if err := addStringElement(ds, tag.New(0x0000, 0x1001), cs.RequestedSOPInstanceUID); err != nil {
			return nil, err
		}
	}

	// Priority (0000,0700) - US
	if cs.Priority != 0 || cs.CommandField&0x8000 == 0 { // Include for requests
		if err := addUInt16Element(ds, tag.New(0x0000, 0x0700), cs.Priority); err != nil {
			return nil, err
		}
	}

	// Command Data Set Type (0000,0800) - US
	if err := addUInt16Element(ds, tag.New(0x0000, 0x0800), cs.CommandDataSetType); err != nil {
		return nil, err
	}

	// Status (0000,0900) - US (for responses)
	if cs.CommandField&0x8000 != 0 { // Response
		if err := addUInt16Element(ds, tag.New(0x0000, 0x0900), cs.Status); err != nil {
			return nil, err
		}
	}

	// Number of Remaining Sub-operations (0000,1020) - US
	if cs.NumberOfRemainingSubOps != 0 {
		if err := addUInt16Element(ds, tag.New(0x0000, 0x1020), cs.NumberOfRemainingSubOps); err != nil {
			return nil, err
		}
	}

	// Number of Completed Sub-operations (0000,1021) - US
	if cs.NumberOfCompletedSubOps != 0 {
		if err := addUInt16Element(ds, tag.New(0x0000, 0x1021), cs.NumberOfCompletedSubOps); err != nil {
			return nil, err
		}
	}

	// Number of Failed Sub-operations (0000,1022) - US
	if cs.NumberOfFailedSubOps != 0 {
		if err := addUInt16Element(ds, tag.New(0x0000, 0x1022), cs.NumberOfFailedSubOps); err != nil {
			return nil, err
		}
	}

	// Number of Warning Sub-operations (0000,1023) - US
	if cs.NumberOfWarningSubOps != 0 {
		if err := addUInt16Element(ds, tag.New(0x0000, 0x1023), cs.NumberOfWarningSubOps); err != nil {
			return nil, err
		}
	}

	// Move Destination (0000,0600) - AE
	if cs.MoveDestination != "" {
		if err := addStringElement(ds, tag.New(0x0000, 0x0600), cs.MoveDestination); err != nil {
			return nil, err
		}
	}

	// Move Originator AE Title (0000,1030) - AE
	if cs.MoveOriginatorAETitle != "" {
		if err := addStringElement(ds, tag.New(0x0000, 0x1030), cs.MoveOriginatorAETitle); err != nil {
			return nil, err
		}
	}

	// Move Originator Message ID (0000,1031) - US
	if cs.MoveOriginatorMessageID != 0 {
		if err := addUInt16Element(ds, tag.New(0x0000, 0x1031), cs.MoveOriginatorMessageID); err != nil {
			return nil, err
		}
	}

	return ds, nil
}

// FromDataSet creates a CommandSet from a DICOM dataset
func FromDataSet(ds *dicom.DataSet) (*CommandSet, error) {
	cs := &CommandSet{}

	// Command Field (required)
	if val, err := getUInt16(ds, tag.New(0x0000, 0x0100)); err == nil {
		cs.CommandField = val
	} else {
		return nil, fmt.Errorf("missing required Command Field: %w", err)
	}

	// Message ID
	if val, err := getUInt16(ds, tag.New(0x0000, 0x0110)); err == nil {
		cs.MessageID = val
	}

	// Message ID Being Responded To
	if val, err := getUInt16(ds, tag.New(0x0000, 0x0120)); err == nil {
		cs.MessageIDBeingRespondedTo = val
	}

	// Affected SOP Class UID
	if val, err := getString(ds, tag.New(0x0000, 0x0002)); err == nil {
		cs.AffectedSOPClassUID = val
	}

	// Affected SOP Instance UID
	if val, err := getString(ds, tag.New(0x0000, 0x1000)); err == nil {
		cs.AffectedSOPInstanceUID = val
	}

	// Requested SOP Class UID
	if val, err := getString(ds, tag.New(0x0000, 0x0003)); err == nil {
		cs.RequestedSOPClassUID = val
	}

	// Requested SOP Instance UID
	if val, err := getString(ds, tag.New(0x0000, 0x1001)); err == nil {
		cs.RequestedSOPInstanceUID = val
	}

	// Priority
	if val, err := getUInt16(ds, tag.New(0x0000, 0x0700)); err == nil {
		cs.Priority = val
	}

	// Command Data Set Type (required)
	if val, err := getUInt16(ds, tag.New(0x0000, 0x0800)); err == nil {
		cs.CommandDataSetType = val
	}

	// Status
	if val, err := getUInt16(ds, tag.New(0x0000, 0x0900)); err == nil {
		cs.Status = val
	}

	// Sub-operations counts
	if val, err := getUInt16(ds, tag.New(0x0000, 0x1020)); err == nil {
		cs.NumberOfRemainingSubOps = val
	}
	if val, err := getUInt16(ds, tag.New(0x0000, 0x1021)); err == nil {
		cs.NumberOfCompletedSubOps = val
	}
	if val, err := getUInt16(ds, tag.New(0x0000, 0x1022)); err == nil {
		cs.NumberOfFailedSubOps = val
	}
	if val, err := getUInt16(ds, tag.New(0x0000, 0x1023)); err == nil {
		cs.NumberOfWarningSubOps = val
	}

	// Move Destination
	if val, err := getString(ds, tag.New(0x0000, 0x0600)); err == nil {
		cs.MoveDestination = val
	}

	// Move Originator
	if val, err := getString(ds, tag.New(0x0000, 0x1030)); err == nil {
		cs.MoveOriginatorAETitle = val
	}
	if val, err := getUInt16(ds, tag.New(0x0000, 0x1031)); err == nil {
		cs.MoveOriginatorMessageID = val
	}

	return cs, nil
}

// Helper functions
func addUInt16Element(ds *dicom.DataSet, t tag.Tag, val uint16) error {
	// Use IntValue since there's no UInt16Value in the value package
	v, err := value.NewIntValue(vr.UnsignedShort, []int64{int64(val)})
	if err != nil {
		return fmt.Errorf("failed to create uint16 value: %w", err)
	}
	elem, err := element.NewElement(t, vr.UnsignedShort, v)
	if err != nil {
		return err
	}
	return ds.Add(elem)
}

func addStringElement(ds *dicom.DataSet, t tag.Tag, val string) error {
	v, err := value.NewStringValue(vr.UniqueIdentifier, []string{val})
	if err != nil {
		return fmt.Errorf("failed to create string value: %w", err)
	}
	elem, err := element.NewElement(t, vr.UniqueIdentifier, v)
	if err != nil {
		return err
	}
	return ds.Add(elem)
}

func getUInt16(ds *dicom.DataSet, t tag.Tag) (uint16, error) {
	elem, err := ds.Get(t)
	if err != nil {
		return 0, err
	}
	val := elem.Value()

	// Use IntValue since there's no UInt16Value in the value package
	if intVal, ok := val.(*value.IntValue); ok {
		ints := intVal.Ints()
		if len(ints) > 0 {
			return uint16(ints[0]), nil
		}
	}

	// For command tags, ElementParser may not know the VR and returns BytesValue
	// Parse the bytes manually as little-endian uint16
	if bytesVal, ok := val.(*value.BytesValue); ok {
		bytes := bytesVal.Bytes()
		if len(bytes) == 2 {
			return uint16(bytes[0]) | uint16(bytes[1])<<8, nil
		}
	}

	return 0, fmt.Errorf("invalid value type for tag %s", t)
}

func getString(ds *dicom.DataSet, t tag.Tag) (string, error) {
	elem, err := ds.Get(t)
	if err != nil {
		return "", err
	}
	val := elem.Value()

	// Try StringValue first (normal case when VR is known)
	if strVal, ok := val.(*value.StringValue); ok {
		strs := strVal.Strings()
		if len(strs) > 0 {
			return strs[0], nil
		}
		return "", nil
	}

	// For command tags, ElementParser may not know the VR and returns BytesValue
	// Convert the bytes to a string and trim null padding
	if bytesVal, ok := val.(*value.BytesValue); ok {
		bytes := bytesVal.Bytes()
		return strings.TrimRight(string(bytes), "\x00"), nil
	}

	// Fallback: use String() method
	return val.String(), nil
}
