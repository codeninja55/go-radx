package dimse

import (
	"context"
	"errors"

	"github.com/codeninja55/go-radx/dimse/dul"
	"github.com/codeninja55/go-radx/dimse/pdu"
)

// sendCommand transmits a command set as a single P-DATA-TF PDU over conn, advancing m by the
// local P-DATA request event (Evt9 -> DT1, stays Sta6) before the write. A C-ECHO command set
// carries no data set and fits in one PDU, so it is sent as one PDV marked command + last
// (0x03) — the final command fragment is always marked last independently of any data set
// (Codex DIMSE-001). The presentation context ID is the one negotiated for the operation.
//
// Fragmentation against the negotiated max PDU length lands with the data-set-bearing C-STORE
// in Increment 5; a command set this small needs none.
func sendCommand(ctx context.Context, conn *dul.Conn, m *dul.StateMachine, pcID uint8, cmd CommandSet) error {
	body, err := cmd.Encode()
	if err != nil {
		return err
	}
	// Evt9 (P-DATA request) -> DT1 (send P-DATA-TF) -> Sta6. A non-Sta6 machine is a fault the
	// state machine surfaces as a *StateError, which we wrap as a *ProtocolError.
	if _, _, serr := m.Apply(dul.Evt9); serr != nil {
		return wrapStateError(m, serr)
	}
	data := &pdu.DataTF{
		Items: []pdu.PresentationDataValue{{
			PresentationContextID: pcID,
			MessageControlHeader:  pdu.MakeControlHeader(true, true), // command, last
			Data:                  body,
		}},
	}
	return conn.WritePDU(ctx, data)
}

// receiveCommand reads P-DATA-TF PDUs through dul.DriveInbound until the last command fragment
// arrives, reassembles the command set from the command PDVs, and returns it with the
// presentation context ID it arrived on. All inbound reads go through DriveInbound against the
// association's own machine so the provider A-ABORT on an unexpected/invalid PDU and the
// clean-close distinction are never reimplemented (the architect's DUL-ownership decision).
//
// A C-ECHO command set carries no data set; this increment reassembles only the command. The
// data-set reassembler (gating on CommandDataSetType and the negotiated transfer syntax) lands
// with C-STORE in Increment 5.
func receiveCommand(ctx context.Context, conn *dul.Conn, m *dul.StateMachine) (CommandSet, uint8, error) {
	var commandBytes []byte
	var pcID uint8
	for {
		p, _, err := dul.DriveInbound(ctx, conn, m)
		if err != nil {
			return CommandSet{}, 0, translateInboundError(m, err)
		}
		data, ok := p.(*pdu.DataTF)
		if !ok {
			return CommandSet{}, 0, &ProtocolError{
				State:  m.CurrentState(),
				Detail: "expected a P-DATA-TF carrying a command set, got " + p.Type().String(),
			}
		}
		lastSeen := false
		for _, item := range data.Items {
			if pcID == 0 {
				pcID = item.PresentationContextID
			}
			if item.IsCommand() {
				commandBytes = append(commandBytes, item.Data...)
				if item.IsLastFragment() {
					lastSeen = true
				}
			}
		}
		if lastSeen {
			break
		}
	}
	cmd, err := DecodeCommandSet(commandBytes)
	if err != nil {
		return CommandSet{}, 0, err
	}
	return cmd, pcID, nil
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
