package dimse

import (
	"bytes"
	"context"
	"errors"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/dul"
	"github.com/codeninja55/go-radx/dimse/pdu"
)

// pdvOverhead is the per-PDV item overhead that counts against the negotiated Maximum Length: the
// 4-byte PDV item-length field, the 1-byte presentation-context ID, and the 1-byte message-control
// header (PS3.8 §9.3.5). The 6-byte P-DATA-TF PDU header sits OUTSIDE the negotiated maximum (the
// maximum bounds the PDU data field, not the framing), so it is correctly excluded — but the
// 4-byte PDV item-length field lives INSIDE that data field and must be reserved, otherwise a full
// fragment overshoots the peer's advertised maximum by 4. pynetdicom fragments at max_pdu - 6.
const pdvOverhead = 6

// fragmentMessage encodes a DIMSE message — a command set, optionally followed by a dataset — into
// one or more P-DATA-TF PDUs. The command set is always Implicit VR LE (PS3.7 §6.3.1); the dataset
// is encoded with the negotiated transfer syntax ts (DIMSE-003). The final command fragment is
// always marked last (0x03) independently of whether a dataset follows (DIMSE-001, the concrete
// Orthanc-abort root cause): command-last and dataset-last are separate boundaries. A sendCap of 0
// means unlimited — the whole stream goes in one PDV, never a zero or negative slice bound
// (DIMSE-005).
func fragmentMessage(cmd CommandSet, ds *dicom.DataSet, ts dicom.TransferSyntax, pcID uint8, sendCap MaxPDULength) ([]*pdu.DataTF, error) {
	cmdBytes, err := cmd.Encode()
	if err != nil {
		return nil, err
	}

	var dsBytes []byte
	if ds != nil {
		var buf bytes.Buffer
		if err := dicom.EncodeDataSet(&buf, ds, ts); err != nil {
			return nil, err
		}
		dsBytes = buf.Bytes()
	}

	var pdus []*pdu.DataTF
	// Command stream: marked command, and the LAST command fragment is marked last regardless of a
	// following dataset (DIMSE-001).
	pdus = append(pdus, fragmentStream(cmdBytes, pcID, true, sendCap)...)
	// Dataset stream: marked dataset; its last fragment is dataset-last (0x02). Only emitted when a
	// dataset is actually present.
	if ds != nil {
		pdus = append(pdus, fragmentStream(dsBytes, pcID, false, sendCap)...)
	}
	return pdus, nil
}

// fragmentStream splits one byte stream (the command stream or the dataset stream) into PDVs, each
// carrying at most sendCap-pdvOverhead payload bytes, marking the final fragment last. command
// selects the control bit (command vs dataset); the last-fragment bit is set on the final chunk
// only, so the command stream's last PDV is 0x03 and the dataset stream's last PDV is 0x02.
func fragmentStream(data []byte, pcID uint8, command bool, sendCap MaxPDULength) []*pdu.DataTF {
	chunk := maxPayloadPerPDV(sendCap)

	var pdus []*pdu.DataTF
	offset := 0
	for {
		end := offset + chunk
		if end >= len(data) {
			end = len(data)
		}
		last := end >= len(data)
		pdus = append(pdus, &pdu.DataTF{Items: []pdu.PresentationDataValue{{
			PresentationContextID: pcID,
			MessageControlHeader:  pdu.MakeControlHeader(command, last),
			Data:                  data[offset:end],
		}}})
		offset = end
		if last {
			break
		}
	}
	return pdus
}

// maxPayloadPerPDV resolves the payload bytes a single PDV may carry under sendCap. An unlimited
// cap (0) yields the whole stream in one PDV (DIMSE-005); otherwise the cap minus the PDV item
// overhead, floored at 1 so a pathologically small cap still makes forward progress rather than a
// zero or negative bound.
func maxPayloadPerPDV(sendCap MaxPDULength) int {
	if sendCap.IsUnlimited() {
		return int(^uint(0) >> 1) // max int: one PDV holds the whole stream
	}
	payload := int(sendCap) - pdvOverhead
	if payload < 1 {
		return 1
	}
	return payload
}

// messageReassembler is the inbound DIMSE message state machine (DIMSE-002). It collects command
// PDVs until command-last, decodes the command set, reads CommandDataSetType to learn whether a
// dataset follows, and only then collects dataset PDVs until dataset-last, decoding the dataset
// with the negotiated transfer syntax (DIMSE-003). It never treats a non-command PDV as a dataset
// before command-last is seen, and never reports a command-only message complete while a declared
// dataset is still arriving.
type messageReassembler struct {
	// resolveTS maps the presentation context ID the message arrived on to the negotiated transfer
	// syntax used to decode the dataset (DIMSE-003). The SCU knows the syntax up front (it chose
	// the context); the SCP resolves it from the accepted contexts by the inbound context ID.
	resolveTS      func(pcID uint8) (dicom.TransferSyntax, error)
	pcID           uint8
	commandBytes   []byte
	datasetBytes   []byte
	commandLast    bool
	expectsDataset bool

	command *CommandSet
	dataset *dicom.DataSet
}

// newMessageReassembler builds a reassembler that decodes the dataset with the fixed transfer
// syntax ts (the negotiated transfer syntax of the operation's presentation context, known to the
// SCU up front).
func newMessageReassembler(ts dicom.TransferSyntax) *messageReassembler {
	return newMessageReassemblerFunc(func(uint8) (dicom.TransferSyntax, error) { return ts, nil })
}

// newMessageReassemblerFunc builds a reassembler that resolves the dataset transfer syntax from
// the inbound presentation context ID (the SCP path, where the syntax depends on which accepted
// context the C-STORE-RQ arrived on).
func newMessageReassemblerFunc(resolveTS func(pcID uint8) (dicom.TransferSyntax, error)) *messageReassembler {
	return &messageReassembler{resolveTS: resolveTS}
}

// add folds one PDV into the message and reports whether the message is complete. A PDV arriving
// after the command-last but before the command says a dataset follows is a protocol fault, as is
// a command PDV after the command was already completed.
func (r *messageReassembler) add(item pdu.PresentationDataValue) (bool, error) {
	if r.pcID == 0 {
		r.pcID = item.PresentationContextID
	}

	if item.IsCommand() {
		if r.commandLast {
			return false, &ProtocolError{State: Sta6, Detail: "command PDV received after the command set was already complete"}
		}
		r.commandBytes = append(r.commandBytes, item.Data...)
		if item.IsLastFragment() {
			if err := r.decodeCommand(); err != nil {
				return false, err
			}
		}
		return r.complete(), nil
	}

	// A dataset PDV before command-last is a violation: the reassembler must not decide a dataset
	// is present until the command says so (DIMSE-002).
	if !r.commandLast {
		return false, &ProtocolError{State: Sta6, Detail: "dataset PDV received before the command set was complete"}
	}
	if !r.expectsDataset {
		return false, &ProtocolError{State: Sta6, Detail: "dataset PDV received but the command declared CommandDataSetType not present"}
	}
	r.datasetBytes = append(r.datasetBytes, item.Data...)
	if item.IsLastFragment() {
		ts, err := r.resolveTS(r.pcID)
		if err != nil {
			return false, err
		}
		ds, err := dicom.DecodeDataSet(bytes.NewReader(r.datasetBytes), ts)
		if err != nil {
			return false, &ProtocolError{State: Sta6, Detail: "decoding C-STORE dataset with the negotiated transfer syntax: " + err.Error()}
		}
		r.dataset = ds
	}
	return r.complete(), nil
}

// decodeCommand decodes the accumulated command bytes and records whether a dataset follows.
func (r *messageReassembler) decodeCommand() error {
	cmd, err := DecodeCommandSet(r.commandBytes)
	if err != nil {
		return err
	}
	r.command = &cmd
	r.commandLast = true
	r.expectsDataset = cmd.HasDataSet()
	return nil
}

// complete reports whether the whole message has arrived: the command is decoded, and either no
// dataset was declared or the dataset has been fully decoded.
func (r *messageReassembler) complete() bool {
	if r.command == nil {
		return false
	}
	if r.expectsDataset {
		return r.dataset != nil
	}
	return true
}

// sendCommand transmits a command-only DIMSE message (no dataset) over conn. The final command
// fragment is always marked last (0x03), independently of any dataset (Codex DIMSE-001) — there is
// none on this path. It is the C-ECHO / response sender; the dataset-bearing path is sendMessage.
// The command set is small, so an unlimited send cap keeps it in a single PDV.
func sendCommand(ctx context.Context, conn *dul.Conn, m *dul.StateMachine, pcID uint8, cmd CommandSet) error {
	return sendMessage(ctx, conn, m, pcID, cmd, nil, dicom.ImplicitVRLittleEndian, 0)
}

// receiveCommand reads a command-only DIMSE message (no dataset) through dul.DriveInbound and
// returns the decoded command set with the presentation context ID it arrived on. It is the
// C-ECHO / response path; the dataset-bearing path is receiveMessage. All inbound reads go through
// DriveInbound against the association's own machine so the provider A-ABORT on an
// unexpected/invalid PDU and the clean-close distinction are never reimplemented (the architect's
// DUL-ownership decision).
func receiveCommand(ctx context.Context, conn *dul.Conn, m *dul.StateMachine) (CommandSet, uint8, error) {
	cmd, _, pcID, err := receiveMessage(ctx, conn, m, newMessageReassembler(dicom.ImplicitVRLittleEndian))
	if err != nil {
		return CommandSet{}, 0, err
	}
	return cmd, pcID, nil
}

// sendMessage transmits a complete DIMSE message — a command set, optionally followed by a dataset
// — as one or more P-DATA-TF PDUs, fragmented within sendCap with the command-last bit set
// independently of the dataset (DIMSE-001). It advances the state machine by the local P-DATA
// request event (Evt9) before each write, exactly as the C-ECHO send does, so the DUL lifecycle is
// honoured. The dataset (when present) is encoded with ts, the negotiated transfer syntax.
func sendMessage(ctx context.Context, conn *dul.Conn, m *dul.StateMachine, pcID uint8, cmd CommandSet, ds *dicom.DataSet, ts dicom.TransferSyntax, sendCap MaxPDULength) error {
	pdus, err := fragmentMessage(cmd, ds, ts, pcID, sendCap)
	if err != nil {
		return err
	}
	for _, p := range pdus {
		if _, _, serr := m.Apply(dul.Evt9); serr != nil { // P-DATA request -> DT1 -> Sta6
			return wrapStateError(m, serr)
		}
		if err := conn.WritePDU(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

// receiveMessage reads a DIMSE message through dul.DriveInbound, driving the reassembler state
// machine: it collects command PDVs until command-last, decodes the command set, and — when the
// command declares CommandDataSetType present — collects dataset PDVs until dataset-last, decoding
// the dataset with ts (DIMSE-002/003). It returns the command set, the dataset (nil when none was
// declared), and the presentation context ID. All inbound reads go through DriveInbound against
// the association's own machine (the architect's DUL-ownership decision).
func receiveMessage(ctx context.Context, conn *dul.Conn, m *dul.StateMachine, r *messageReassembler) (CommandSet, *dicom.DataSet, uint8, error) {
	for {
		p, _, err := dul.DriveInbound(ctx, conn, m)
		if err != nil {
			return CommandSet{}, nil, 0, translateInboundError(m, err)
		}
		data, ok := p.(*pdu.DataTF)
		if !ok {
			return CommandSet{}, nil, 0, &ProtocolError{
				State:  m.CurrentState(),
				Detail: "expected a P-DATA-TF carrying a DIMSE message, got " + p.Type().String(),
			}
		}
		for _, item := range data.Items {
			done, aerr := r.add(item)
			if aerr != nil {
				return CommandSet{}, nil, 0, aerr
			}
			if done {
				return *r.command, r.dataset, r.pcID, nil
			}
		}
	}
}

// translateInboundError maps a dul.DriveInbound error to the public dimse error model. A context
// cancellation or deadline is surfaced as-is (a timeout, not a protocol fault); a clean io.EOF
// is the conversation breaking before the expected PDU arrived; a *dul.StateError is a protocol
// violation DriveInbound already aborted on the wire, surfaced as a *ProtocolError naming the
// violating state.
func translateInboundError(m *dul.StateMachine, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var se *dul.StateError
	if errors.As(err, &se) {
		return &ProtocolError{State: se.State, Detail: se.Error()}
	}
	return &ProtocolError{State: m.CurrentState(), Detail: "reading DIMSE message: " + err.Error()}
}

// wrapStateError wraps a dul state-machine fault as a typed *ProtocolError naming the state.
func wrapStateError(m *dul.StateMachine, err error) error {
	var se *dul.StateError
	if errors.As(err, &se) {
		return &ProtocolError{State: se.State, Detail: se.Error()}
	}
	return &ProtocolError{State: m.CurrentState(), Detail: err.Error()}
}
