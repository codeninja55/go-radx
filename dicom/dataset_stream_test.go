package dicom

import (
	"bytes"
	"testing"
)

// TestEncodeDecodeDataSetRoundTripExplicitLE confirms the public bare-dataset codec round-trips a
// dataset in Explicit VR Little Endian without a Part 10 preamble or file-meta group. This is the
// entry point the dimse and dicomweb layers use to carry a dataset over the wire (PS3.7 §6.3.1):
// a bare element stream in the negotiated transfer syntax, never a File.
func TestEncodeDecodeDataSetRoundTripExplicitLE(t *testing.T) {
	ds := NewDataSet()
	ds.SetString(NewTag(0x0008, 0x0016), "1.2.840.10008.5.1.4.1.1.4") // SOP Class UID
	ds.SetString(NewTag(0x0008, 0x0018), "1.2.3.4.5")                 // SOP Instance UID
	ds.SetString(NewTag(0x0010, 0x0010), "Doe^Jane")                  // PatientName

	var buf bytes.Buffer
	if err := EncodeDataSet(&buf, ds, ExplicitVRLittleEndian); err != nil {
		t.Fatalf("EncodeDataSet: %v", err)
	}
	// A bare dataset starts at the first element tag, not the 128-byte preamble + "DICM".
	if buf.Len() >= 132 && bytes.Equal(buf.Bytes()[128:132], []byte("DICM")) {
		t.Fatal("EncodeDataSet wrote a Part 10 preamble; the DIMSE/DICOMweb stream must be bare")
	}

	got, err := DecodeDataSet(bytes.NewReader(buf.Bytes()), ExplicitVRLittleEndian)
	if err != nil {
		t.Fatalf("DecodeDataSet: %v", err)
	}
	if got.Len() != ds.Len() {
		t.Fatalf("decoded element count = %d, want %d", got.Len(), ds.Len())
	}
	if v, _ := got.GetString(NewTag(0x0010, 0x0010)); v != "Doe^Jane" {
		t.Errorf("PatientName round-trip = %q, want Doe^Jane", v)
	}
	if v, _ := got.GetString(NewTag(0x0008, 0x0016)); v != "1.2.840.10008.5.1.4.1.1.4" {
		t.Errorf("SOP Class UID round-trip = %q", v)
	}
}

// TestEncodeDecodeDataSetImplicitVR confirms the codec honours Implicit VR Little Endian, the
// encoding the DIMSE command set always uses, distinct from the negotiated dataset syntax.
func TestEncodeDecodeDataSetImplicitVR(t *testing.T) {
	ds := NewDataSet()
	ds.SetString(NewTag(0x0008, 0x0018), "1.2.3.4.5")

	var buf bytes.Buffer
	if err := EncodeDataSet(&buf, ds, ImplicitVRLittleEndian); err != nil {
		t.Fatalf("EncodeDataSet implicit: %v", err)
	}
	got, err := DecodeDataSet(bytes.NewReader(buf.Bytes()), ImplicitVRLittleEndian)
	if err != nil {
		t.Fatalf("DecodeDataSet implicit: %v", err)
	}
	if v, _ := got.GetString(NewTag(0x0008, 0x0018)); v != "1.2.3.4.5" {
		t.Errorf("implicit VR round-trip = %q, want 1.2.3.4.5", v)
	}
}

// TestDecodeDataSetRejectsEncapsulatedSyntax confirms the codec rejects an encapsulated syntax: v1
// reads only the four uncompressed syntaxes as a bare element stream (the pixel pipeline handles
// encapsulated pixel data separately).
func TestDecodeDataSetRejectsEncapsulatedSyntax(t *testing.T) {
	if _, err := DecodeDataSet(bytes.NewReader(nil), TransferSyntax("1.2.840.10008.1.2.4.90")); err == nil {
		t.Error("DecodeDataSet should reject an encapsulated transfer syntax")
	}
}
