package dimse

import (
	"testing"

	"github.com/codeninja55/go-radx/dicom"
)

// TestDefaultTransferSyntaxesExactOrder pins the four committed uncompressed/deflated
// syntaxes in their proposal-preference order (dimse.md "Default transfer syntaxes"):
// Explicit VR Little Endian leads.
func TestDefaultTransferSyntaxesExactOrder(t *testing.T) {
	want := []dicom.TransferSyntax{
		dicom.ExplicitVRLittleEndian,         // 1.2.840.10008.1.2.1
		dicom.ImplicitVRLittleEndian,         // 1.2.840.10008.1.2
		dicom.DeflatedExplicitVRLittleEndian, // 1.2.840.10008.1.2.1.99
		dicom.ExplicitVRBigEndian,            // 1.2.840.10008.1.2.2
	}
	if len(DefaultTransferSyntaxes) != len(want) {
		t.Fatalf("DefaultTransferSyntaxes has %d entries, want %d", len(DefaultTransferSyntaxes), len(want))
	}
	for i := range want {
		if DefaultTransferSyntaxes[i] != want[i] {
			t.Errorf("DefaultTransferSyntaxes[%d] = %q, want %q", i, DefaultTransferSyntaxes[i], want[i])
		}
	}
}

// TestNewPresentationContextDefaultsTransferSyntaxes verifies a context built without an
// explicit transfer-syntax list proposes the default set.
func TestNewPresentationContextDefaultsTransferSyntaxes(t *testing.T) {
	pc := NewPresentationContext(1, dicom.SOPClassUID("1.2.840.10008.1.1"))
	if pc.ID != 1 {
		t.Errorf("ID = %d, want 1", pc.ID)
	}
	if pc.AbstractSyntax != "1.2.840.10008.1.1" {
		t.Errorf("AbstractSyntax = %q, want the Verification SOP Class", pc.AbstractSyntax)
	}
	if len(pc.TransferSyntaxes) != len(DefaultTransferSyntaxes) {
		t.Fatalf("default context proposes %d transfer syntaxes, want %d",
			len(pc.TransferSyntaxes), len(DefaultTransferSyntaxes))
	}
	for i := range DefaultTransferSyntaxes {
		if pc.TransferSyntaxes[i] != DefaultTransferSyntaxes[i] {
			t.Errorf("TransferSyntaxes[%d] = %q, want %q", i, pc.TransferSyntaxes[i], DefaultTransferSyntaxes[i])
		}
	}
}

func TestNewPresentationContextExplicitTransferSyntaxes(t *testing.T) {
	pc := NewPresentationContext(3, dicom.SOPClassUID("1.2.840.10008.5.1.4.1.1.2"),
		dicom.ExplicitVRLittleEndian)
	if len(pc.TransferSyntaxes) != 1 || pc.TransferSyntaxes[0] != dicom.ExplicitVRLittleEndian {
		t.Errorf("TransferSyntaxes = %v, want [Explicit VR LE]", pc.TransferSyntaxes)
	}
}

// TestNewPresentationContextCopiesTransferSyntaxes guards against the proposal sharing the
// caller's backing array: mutating the input afterwards must not change the context, and a
// default context must not share DefaultTransferSyntaxes' backing array.
func TestNewPresentationContextCopiesTransferSyntaxes(t *testing.T) {
	ts := []dicom.TransferSyntax{dicom.ExplicitVRLittleEndian, dicom.ImplicitVRLittleEndian}
	pc := NewPresentationContext(1, dicom.SOPClassUID("1.2.840.10008.1.1"), ts...)
	ts[0] = dicom.ExplicitVRBigEndian
	if pc.TransferSyntaxes[0] != dicom.ExplicitVRLittleEndian {
		t.Error("NewPresentationContext shares the caller's transfer-syntax slice")
	}

	pcDefault := NewPresentationContext(1, dicom.SOPClassUID("1.2.840.10008.1.1"))
	pcDefault.TransferSyntaxes[0] = dicom.ExplicitVRBigEndian
	if DefaultTransferSyntaxes[0] != dicom.ExplicitVRLittleEndian {
		t.Error("a default context shares the DefaultTransferSyntaxes backing array")
	}
}

func TestContextResultValues(t *testing.T) {
	// The result vocabulary must match the wire vocabulary (PS3.8 9.3.3.2).
	cases := map[ContextResult]uint8{
		ContextAccepted:                     0,
		ContextUserRejected:                 1,
		ContextNoReason:                     2,
		ContextAbstractSyntaxNotSupported:   3,
		ContextTransferSyntaxesNotSupported: 4,
	}
	for got, want := range cases {
		if uint8(got) != want {
			t.Errorf("ContextResult %v = %d, want %d", got, uint8(got), want)
		}
	}
}

// TestMaxPDULengthUnlimited guards Codex DIMSE-005: a MaxPDULength of 0 means "no maximum
// specified" (unlimited), never a literal zero allocation size. SendCap resolves 0 to the
// configured local send cap, never 0 and never a negative bound.
func TestMaxPDULengthUnlimited(t *testing.T) {
	const localCap = 16382

	if !MaxPDULength(0).IsUnlimited() {
		t.Error("MaxPDULength(0).IsUnlimited() = false, want true (DIMSE-005)")
	}
	if MaxPDULength(16382).IsUnlimited() {
		t.Error("MaxPDULength(16382).IsUnlimited() = true, want false")
	}

	// An unlimited peer max must resolve to the local send cap, not 0.
	if got := MaxPDULength(0).SendCap(localCap); got != localCap {
		t.Errorf("MaxPDULength(0).SendCap(%d) = %d, want %d (unlimited -> local cap, DIMSE-005)", localCap, got, localCap)
	}
	// A peer that advertises a smaller max must cap the send size to that smaller value.
	if got := MaxPDULength(8192).SendCap(localCap); got != 8192 {
		t.Errorf("MaxPDULength(8192).SendCap(%d) = %d, want 8192 (honour the smaller peer max)", localCap, got)
	}
	// A peer that advertises a larger max than the local cap must still be capped locally.
	if got := MaxPDULength(65536).SendCap(localCap); got != localCap {
		t.Errorf("MaxPDULength(65536).SendCap(%d) = %d, want %d (local cap wins)", localCap, got, localCap)
	}
}
