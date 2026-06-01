package dimse

import (
	"bytes"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/pdu"
)

// sampleStoreDataSet builds a tiny dataset with a SOP Class/Instance and one extra element, enough
// to exercise dataset fragmentation and decode without a vendored file.
func sampleStoreDataSet() *dicom.DataSet {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.NewTag(0x0008, 0x0016), "1.2.840.10008.5.1.4.1.1.4") // SOP Class UID
	ds.SetString(dicom.NewTag(0x0008, 0x0018), "1.2.3.4.5.6.7.8")           // SOP Instance UID
	ds.SetString(dicom.NewTag(0x0010, 0x0010), "Doe^Jane")                  // PatientName
	return ds
}

// collectPDVs flattens a slice of P-DATA-TF PDUs into their PDVs in order.
func collectPDVs(pdus []*pdu.DataTF) []pdu.PresentationDataValue {
	var items []pdu.PresentationDataValue
	for _, p := range pdus {
		items = append(items, p.Items...)
	}
	return items
}

// TestCommandLastBitSetIndependentOfDataset is the named DIMSE-001 regression — the concrete
// Orthanc-abort root cause (PRD line 60). Fragmenting a C-STORE-RQ command set followed by a
// dataset must produce a final COMMAND PDV marked 0x03 (command + last) and THEN dataset PDVs; the
// command-last bit must not wait for the dataset. The prototype left the final command fragment
// unmarked when a dataset followed, which Orthanc aborted.
func TestCommandLastBitSetIndependentOfDataset(t *testing.T) {
	cmd := CommandSet{
		CommandField:           CommandCStoreRQ,
		MessageID:              1,
		AffectedSOPClassUID:    dicom.UID("1.2.840.10008.5.1.4.1.1.4"),
		AffectedSOPInstanceUID: dicom.UID("1.2.3.4.5.6.7.8"),
		HasPriority:            true,
		Priority:               PriorityMedium,
		CommandDataSetType:     CommandDataSetPresent,
	}
	ds := sampleStoreDataSet()

	pdus, err := fragmentMessage(cmd, ds, dicom.ExplicitVRLittleEndian, 3, 16384)
	if err != nil {
		t.Fatalf("fragmentMessage: %v", err)
	}
	items := collectPDVs(pdus)
	if len(items) < 2 {
		t.Fatalf("expected at least one command PDV and one dataset PDV, got %d", len(items))
	}

	// Walk the PDVs: the command stream comes first and its last command PDV must be marked 0x03;
	// the dataset stream follows and its last PDV must be 0x02.
	var lastCommandIdx, firstDatasetIdx = -1, -1
	for i, it := range items {
		if it.IsCommand() {
			lastCommandIdx = i
			if firstDatasetIdx != -1 {
				t.Fatalf("PDV %d is a command but a dataset PDV already appeared at %d — command stream must precede the dataset", i, firstDatasetIdx)
			}
		} else if firstDatasetIdx == -1 {
			firstDatasetIdx = i
		}
	}
	if lastCommandIdx == -1 || firstDatasetIdx == -1 {
		t.Fatalf("expected both command and dataset PDVs (lastCmd=%d firstDs=%d)", lastCommandIdx, firstDatasetIdx)
	}

	lastCmd := items[lastCommandIdx]
	if lastCmd.MessageControlHeader != 0x03 {
		t.Errorf("final command PDV header = %#02x, want 0x03 (command + last) independently of the following dataset (DIMSE-001)",
			lastCmd.MessageControlHeader)
	}
	if !lastCmd.IsLastFragment() {
		t.Error("final command PDV IsLastFragment() = false, want true (DIMSE-001)")
	}
	last := items[len(items)-1]
	if last.IsCommand() || !last.IsLastFragment() || last.MessageControlHeader != 0x02 {
		t.Errorf("final dataset PDV header = %#02x, want 0x02 (dataset + last)", last.MessageControlHeader)
	}
}

// TestReassemblerGatesOnCommandLast is the named DIMSE-002 regression. Feeding the reassembler
// command PDVs that are NOT yet marked last must leave the message incomplete; it must not treat a
// later non-command PDV as a dataset until command-last is seen and CommandDataSetType says a
// dataset follows.
func TestReassemblerGatesOnCommandLast(t *testing.T) {
	cmd := CommandSet{
		CommandField:           CommandCStoreRQ,
		MessageID:              1,
		AffectedSOPClassUID:    dicom.UID("1.2.840.10008.5.1.4.1.1.4"),
		AffectedSOPInstanceUID: dicom.UID("1.2.3.4.5.6.7.8"),
		HasPriority:            true,
		Priority:               PriorityMedium,
		CommandDataSetType:     CommandDataSetPresent,
	}
	cmdBytes, err := cmd.Encode()
	if err != nil {
		t.Fatalf("encode command: %v", err)
	}
	// Split the command into two fragments; the first is NOT last.
	mid := len(cmdBytes) / 2
	r := newMessageReassembler(dicom.ExplicitVRLittleEndian)

	done, err := r.add(pdu.PresentationDataValue{
		PresentationContextID: 3,
		MessageControlHeader:  pdu.MakeControlHeader(true, false), // command, NOT last
		Data:                  cmdBytes[:mid],
	})
	if err != nil {
		t.Fatalf("add first command fragment: %v", err)
	}
	if done {
		t.Fatal("reassembler reported done after a non-last command fragment (DIMSE-002)")
	}
	if r.command != nil {
		t.Fatal("reassembler decoded the command set before command-last (DIMSE-002)")
	}

	// Second command fragment IS last; the command is now decodable, but a dataset is declared, so
	// the message is still incomplete until a dataset-last PDV arrives.
	done, err = r.add(pdu.PresentationDataValue{
		PresentationContextID: 3,
		MessageControlHeader:  pdu.MakeControlHeader(true, true), // command, last
		Data:                  cmdBytes[mid:],
	})
	if err != nil {
		t.Fatalf("add last command fragment: %v", err)
	}
	if r.command == nil {
		t.Fatal("reassembler did not decode the command set after command-last")
	}
	if done {
		t.Fatal("reassembler reported done with a dataset declared but not yet received (DIMSE-002)")
	}
}

// TestDatasetDecodedWithNegotiatedTransferSyntax is the named DIMSE-003 regression: a dataset sent
// under Explicit VR LE must be decoded as Explicit VR LE, not hard-coded Implicit VR LE. We encode
// the dataset with Explicit VR LE, feed the reassembler that syntax, and confirm the dataset
// decodes (an Implicit VR decode of an Explicit VR stream would mis-parse the VR bytes as length).
func TestDatasetDecodedWithNegotiatedTransferSyntax(t *testing.T) {
	cmd := CommandSet{
		CommandField:           CommandCStoreRQ,
		MessageID:              1,
		AffectedSOPClassUID:    dicom.UID("1.2.840.10008.5.1.4.1.1.4"),
		AffectedSOPInstanceUID: dicom.UID("1.2.3.4.5.6.7.8"),
		HasPriority:            true,
		Priority:               PriorityMedium,
		CommandDataSetType:     CommandDataSetPresent,
	}
	ds := sampleStoreDataSet()

	pdus, err := fragmentMessage(cmd, ds, dicom.ExplicitVRLittleEndian, 3, 16384)
	if err != nil {
		t.Fatalf("fragmentMessage: %v", err)
	}

	r := newMessageReassembler(dicom.ExplicitVRLittleEndian)
	var done bool
	for _, it := range collectPDVs(pdus) {
		done, err = r.add(it)
		if err != nil {
			t.Fatalf("reassembler add: %v", err)
		}
	}
	if !done {
		t.Fatal("reassembler did not complete after all PDVs")
	}
	if r.dataset == nil {
		t.Fatal("reassembler produced no dataset")
	}
	// The Explicit VR LE PatientName must round-trip; a wrong (Implicit) decode would corrupt it.
	if v, _ := r.dataset.GetString(dicom.NewTag(0x0010, 0x0010)); v != "Doe^Jane" {
		t.Errorf("dataset PatientName decoded with negotiated TS = %q, want Doe^Jane (DIMSE-003)", v)
	}
	if v, _ := r.dataset.GetString(dicom.NewTag(0x0008, 0x0018)); v != "1.2.3.4.5.6.7.8" {
		t.Errorf("dataset SOP Instance UID = %q, want round-trip", v)
	}
}

// TestReassemblerNoDatasetCompletesAtCommandLast confirms a command-only message (CommandDataSetType
// 0x0101, e.g. a C-ECHO or a C-STORE-RSP) completes at command-last with no dataset, so the gating
// does not hang waiting for a dataset that never comes.
func TestReassemblerNoDatasetCompletesAtCommandLast(t *testing.T) {
	cmd := CommandSet{
		CommandField:              CommandCStoreRSP,
		MessageIDBeingRespondedTo: 1,
		AffectedSOPClassUID:       dicom.UID("1.2.840.10008.5.1.4.1.1.4"),
		CommandDataSetType:        CommandDataSetNotPresent,
		HasStatus:                 true,
		Status:                    StatusStoreSuccess.Code,
	}
	cmdBytes, err := cmd.Encode()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	r := newMessageReassembler(dicom.ExplicitVRLittleEndian)
	done, err := r.add(pdu.PresentationDataValue{
		PresentationContextID: 3,
		MessageControlHeader:  pdu.MakeControlHeader(true, true),
		Data:                  cmdBytes,
	})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if !done {
		t.Fatal("command-only message did not complete at command-last")
	}
	if r.dataset != nil {
		t.Error("command-only message produced a dataset, want nil")
	}
	if r.command.Status != StatusStoreSuccess.Code {
		t.Errorf("decoded status = %#04x, want store success", r.command.Status)
	}
}

// TestFragmentMessageRespectsSendCap confirms a small send cap splits both the command and the
// dataset into multiple PDVs, each payload within the cap, never a single oversized PDU.
func TestFragmentMessageRespectsSendCap(t *testing.T) {
	cmd := CommandSet{
		CommandField:           CommandCStoreRQ,
		MessageID:              1,
		AffectedSOPClassUID:    dicom.UID("1.2.840.10008.5.1.4.1.1.4"),
		AffectedSOPInstanceUID: dicom.UID("1.2.3.4.5.6.7.8"),
		HasPriority:            true,
		Priority:               PriorityMedium,
		CommandDataSetType:     CommandDataSetPresent,
	}
	ds := sampleStoreDataSet()

	const cap = 64
	pdus, err := fragmentMessage(cmd, ds, dicom.ExplicitVRLittleEndian, 3, cap)
	if err != nil {
		t.Fatalf("fragmentMessage: %v", err)
	}
	items := collectPDVs(pdus)
	if len(items) < 3 {
		t.Fatalf("small send cap should split into several PDVs, got %d", len(items))
	}
	// Each emitted PDV occupies, inside the P-DATA-TF data field the negotiated Maximum Length
	// bounds, a 4-byte PDV item length + 1-byte ctx ID + 1-byte control header + payload. Assert
	// that full 6-byte-overhead footprint against the cap with the literal on-wire constant (NOT the
	// pdvOverhead symbol the producer uses), so an under-count regression — e.g. reserving only the
	// 2-byte item header and overshooting the peer's advertised maximum by 4 — is caught here.
	const onWirePDVOverhead = 6 // 4-byte item length + 1 ctx ID + 1 control header
	for i, it := range items {
		if len(it.Data)+onWirePDVOverhead > cap {
			t.Errorf("PDV %d data-field footprint %d (payload %d + %d overhead) exceeds send cap %d",
				i, len(it.Data)+onWirePDVOverhead, len(it.Data), onWirePDVOverhead, cap)
		}
	}

	// The reassembled message must still be intact.
	r := newMessageReassembler(dicom.ExplicitVRLittleEndian)
	var done bool
	for _, it := range items {
		if done, err = r.add(it); err != nil {
			t.Fatalf("reassemble: %v", err)
		}
	}
	if !done || r.dataset == nil {
		t.Fatalf("fragmented message did not reassemble (done=%v dataset=%v)", done, r.dataset != nil)
	}
	if v, _ := r.dataset.GetString(dicom.NewTag(0x0010, 0x0010)); v != "Doe^Jane" {
		t.Errorf("reassembled PatientName = %q, want Doe^Jane", v)
	}
}

// TestFragmentMessageZeroCapTreatedAsUnlimited guards DIMSE-005: a send cap of 0 must not produce a
// negative or zero slice bound; it is treated as a single unlimited PDU.
func TestFragmentMessageZeroCapTreatedAsUnlimited(t *testing.T) {
	cmd := CommandSet{
		CommandField:        CommandCEchoRQ,
		MessageID:           1,
		AffectedSOPClassUID: dicom.UID(verificationSOPClass),
		CommandDataSetType:  CommandDataSetNotPresent,
	}
	pdus, err := fragmentMessage(cmd, nil, dicom.ImplicitVRLittleEndian, 1, 0)
	if err != nil {
		t.Fatalf("fragmentMessage with zero cap: %v", err)
	}
	items := collectPDVs(pdus)
	if len(items) != 1 {
		t.Fatalf("zero cap (unlimited) should send one command PDV, got %d", len(items))
	}
	if items[0].MessageControlHeader != 0x03 {
		t.Errorf("single command PDV header = %#02x, want 0x03", items[0].MessageControlHeader)
	}
	if !bytes.Equal(items[0].Data[0:2], []byte{0x00, 0x00}) {
		t.Errorf("command stream should begin with the group-length tag (0000,xxxx)")
	}
}
