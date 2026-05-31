package dicom

import (
	"path/filepath"
	"testing"
)

func TestDataSetWriteFileDerivesMetaUIDs(t *testing.T) {
	ds := NewDataSet()
	ds.Set(Element{Tag: TagSOPClassUID, VR: VRUI, Value: NewStrings(VRUI, "1.2.840.10008.5.1.4.1.1.7")})
	ds.Set(Element{Tag: TagSOPInstanceUID, VR: VRUI, Value: NewStrings(VRUI, "1.2.3.4.5.6.7.8.9")})
	ds.Set(Element{Tag: NewTag(0x0010, 0x0010), VR: VRPN, Value: NewStrings(VRPN, "Doe^Jane")})

	dir := t.TempDir()
	path := filepath.Join(dir, "out.dcm")
	if err := ds.WriteFile(path, ExplicitVRLittleEndian); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if got.Meta.MediaStorageSOPClassUID != "1.2.840.10008.5.1.4.1.1.7" {
		t.Errorf("MediaStorageSOPClassUID = %q, derived from (0008,0016)", got.Meta.MediaStorageSOPClassUID)
	}
	if got.Meta.MediaStorageSOPInstanceUID != "1.2.3.4.5.6.7.8.9" {
		t.Errorf("MediaStorageSOPInstanceUID = %q, derived from (0008,0018)", got.Meta.MediaStorageSOPInstanceUID)
	}
	if got.Meta.TransferSyntaxUID != ExplicitVRLittleEndian {
		t.Errorf("TransferSyntaxUID = %q, want Explicit VR LE", got.Meta.TransferSyntaxUID)
	}
	if v, ok := got.DataSet.GetString(NewTag(0x0010, 0x0010)); !ok || v != "Doe^Jane" {
		t.Errorf("PatientName = %q,%v", v, ok)
	}
}

func TestDataSetWriteFileMissingSOPClassFails(t *testing.T) {
	ds := NewDataSet()
	ds.Set(Element{Tag: TagSOPInstanceUID, VR: VRUI, Value: NewStrings(VRUI, "1.2.3.4.5.6.7.8.9")})
	dir := t.TempDir()
	if err := ds.WriteFile(filepath.Join(dir, "x.dcm"), ExplicitVRLittleEndian); err == nil {
		t.Error("WriteFile should fail when (0008,0016) SOP Class UID is absent")
	}
}

func TestDataSetWriteFileMissingSOPInstanceFails(t *testing.T) {
	ds := NewDataSet()
	ds.Set(Element{Tag: TagSOPClassUID, VR: VRUI, Value: NewStrings(VRUI, "1.2.840.10008.5.1.4.1.1.7")})
	dir := t.TempDir()
	if err := ds.WriteFile(filepath.Join(dir, "x.dcm"), ExplicitVRLittleEndian); err == nil {
		t.Error("WriteFile should fail when (0008,0018) SOP Instance UID is absent")
	}
}

func TestDataSetWriteFileUnsupportedSyntaxFails(t *testing.T) {
	ds := NewDataSet()
	ds.Set(Element{Tag: TagSOPClassUID, VR: VRUI, Value: NewStrings(VRUI, "1.2.840.10008.5.1.4.1.1.7")})
	ds.Set(Element{Tag: TagSOPInstanceUID, VR: VRUI, Value: NewStrings(VRUI, "1.2.3.4.5.6.7.8.9")})
	dir := t.TempDir()
	if err := ds.WriteFile(filepath.Join(dir, "x.dcm"), JPEGBaseline8Bit); err == nil {
		t.Error("WriteFile should reject an unsupported transfer syntax")
	}
}
