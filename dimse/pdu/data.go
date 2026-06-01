package pdu

// Message-control header bits (PS3.8 §9.3.5.1). Bit 0 is the command/dataset bit;
// bit 1 is the last-fragment bit. They are independent: a final command fragment
// is 0x03 whether or not a dataset follows (Codex DIMSE-001).
const (
	controlCommandBit byte = 0x01 // bit 0: 1 = command, 0 = dataset
	controlLastBit    byte = 0x02 // bit 1: 1 = last fragment of this command/dataset
)

// PresentationDataValue is one PDV inside a P-DATA-TF PDU: a presentation context
// ID, a one-byte message-control header, and the fragment payload.
type PresentationDataValue struct {
	PresentationContextID uint8
	MessageControlHeader  byte
	Data                  []byte
}

// IsCommand reports whether the PDV carries command-set bytes (bit 0 set).
func (p PresentationDataValue) IsCommand() bool { return p.MessageControlHeader&controlCommandBit != 0 }

// IsLastFragment reports whether this is the last fragment of its command or
// dataset (bit 1 set).
func (p PresentationDataValue) IsLastFragment() bool {
	return p.MessageControlHeader&controlLastBit != 0
}

// MakeControlHeader composes a message-control header from the two independent
// bits. The DIMSE message layer (Increment 5) uses this so the final command
// fragment is always 0x03 and the final dataset fragment 0x02.
func MakeControlHeader(command, last bool) byte {
	var h byte
	if command {
		h |= controlCommandBit
	}
	if last {
		h |= controlLastBit
	}
	return h
}
