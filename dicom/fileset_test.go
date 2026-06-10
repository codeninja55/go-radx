package dicom

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// fileSetSample builds a complete in-memory instance carrying every required
// directory-record key.
func fileSetSample(patientID, studyUID, seriesUID, sopUID, instanceNumber string) *File {
	ds := NewDataSet()
	ds.SetString(TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.7")
	ds.SetString(TagSOPInstanceUID, sopUID)
	ds.SetString(TagStudyDate, "20240101")
	ds.SetString(TagStudyTime, "120000")
	ds.SetString(TagModality, "OT")
	ds.SetString(TagPatientName, "Doe^Jane")
	ds.SetString(TagPatientID, patientID)
	ds.SetString(TagStudyInstanceUID, studyUID)
	ds.SetString(TagSeriesInstanceUID, seriesUID)
	ds.SetString(TagStudyID, "1")
	ds.SetString(TagSeriesNumber, "1")
	ds.SetString(TagInstanceNumber, instanceNumber)
	return &File{
		Meta: &FileMeta{
			MediaStorageSOPClassUID:    "1.2.840.10008.5.1.4.1.1.7",
			MediaStorageSOPInstanceUID: SOPInstanceUID(sopUID),
			TransferSyntaxUID:          ExplicitVRLittleEndian,
		},
		DataSet: ds,
	}
}

// srSampleFile builds a Basic Text SR instance with the SR DOCUMENT record keys.
func srSampleFile(sopUID string) *File {
	f := fileSetSample("SRPAT", "1.2.3.100", "1.2.3.100.1", sopUID, "1")
	ds := f.DataSet
	ds.SetString(TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.88.11")
	ds.SetString(TagModality, "SR")
	ds.SetString(TagContentDate, "20240101")
	ds.SetString(TagContentTime, "120000")
	ds.SetString(TagCompletionFlag, "PARTIAL")
	ds.SetString(TagVerificationFlag, "UNVERIFIED")
	code := NewDataSet()
	code.SetString(TagCodeValue, "11528-7")
	code.SetString(TagCodingSchemeDesignator, "LN")
	code.SetString(TagCodeMeaning, "Radiology Report")
	ds.Set(Element{Tag: TagConceptNameCodeSequence, VR: VRSQ, Value: NewSequenceValue(NewSequence(code))})
	f.Meta.MediaStorageSOPClassUID = "1.2.840.10008.5.1.4.1.1.88.11"
	return f
}

func fixturePath(name string) string {
	return filepath.Join("..", "testdata", "dicom", name)
}

func TestFileSetBuildWriteOpenRoundTrip(t *testing.T) {
	b := NewFileSetBuilder()
	b.SetID("GO RADX TEST")
	fixtures := []string{"liver.dcm", "MR2_UNCI.dcm", "SC_rgb_expb.dcm"}
	for _, name := range fixtures {
		if err := b.Add(fixturePath(name)); err != nil {
			t.Fatalf("Add(%s): %v", name, err)
		}
	}

	root := t.TempDir()
	fs, err := b.Write(root)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	if fs.ID() != "GO RADX TEST" {
		t.Errorf("ID = %q, want %q", fs.ID(), "GO RADX TEST")
	}
	if fs.UID() == "" {
		t.Error("UID is empty")
	}
	if got := len(fs.Roots()); got != 3 {
		t.Fatalf("Roots = %d records, want 3 patients", got)
	}
	var total int
	for r := range fs.Records() {
		total++
		if r.Type == "" {
			t.Error("record has no Directory Record Type")
		}
	}
	if total != 12 {
		t.Errorf("Records walked %d, want 12 (3 x patient/study/series/image)", total)
	}
	for _, root := range fs.Roots() {
		if root.Type != "PATIENT" {
			t.Errorf("root record type = %q, want PATIENT", root.Type)
		}
		if root.Parent() != nil {
			t.Error("root record has a parent")
		}
		if len(root.Children()) != 1 || root.Children()[0].Type != "STUDY" {
			t.Fatalf("patient children = %v, want one STUDY", root.Children())
		}
		study := root.Children()[0]
		if study.Parent() != root {
			t.Error("study parent is not its patient")
		}
	}
	if got := len(fs.Instances()); got != 3 {
		t.Fatalf("Instances = %d, want 3", got)
	}

	// The DICOMDIR is a normal Part 10 file: plain Read must parse it.
	plain, err := ReadFile(fs.Path())
	if err != nil {
		t.Fatalf("plain ReadFile(DICOMDIR): %v", err)
	}
	if plain.Meta.MediaStorageSOPClassUID != MediaStorageDirectoryStorage {
		t.Errorf("DICOMDIR SOP Class = %q, want Media Storage Directory Storage", plain.Meta.MediaStorageSOPClassUID)
	}
	if plain.Meta.TransferSyntaxUID != ExplicitVRLittleEndian {
		t.Errorf("DICOMDIR transfer syntax = %q, want Explicit VR LE", plain.Meta.TransferSyntaxUID)
	}

	// Members are copied verbatim.
	inst := fs.Find(map[Tag]string{TagPatientID: "99000"})
	if len(inst) != 1 {
		t.Fatalf("Find(PatientID=99000) = %d instances, want 1", len(inst))
	}
	want, err := os.ReadFile(fixturePath("liver.dcm"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(inst[0].Path())
	if err != nil {
		t.Fatalf("member file missing: %v", err)
	}
	if !bytes.Equal(want, got) {
		t.Error("member file is not a verbatim copy of the source")
	}

	// Load resolves and re-reads the member; its SOP Instance UID matches the record.
	loaded, err := inst[0].Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	recUID, _ := inst[0].Record.DataSet.GetString(TagReferencedSOPInstanceUIDInFile)
	fileUID, _ := loaded.DataSet.GetString(TagSOPInstanceUID)
	if recUID == "" || recUID != fileUID {
		t.Errorf("Referenced SOP Instance UID %q != loaded SOP Instance UID %q", recUID, fileUID)
	}

	// Queries.
	pids := fs.FindValues(TagPatientID)
	if len(pids) != 3 {
		t.Errorf("FindValues(PatientID) = %v, want 3 values", pids)
	}
	if got := fs.Find(map[Tag]string{TagModality: "MR"}); len(got) != 1 {
		t.Errorf("Find(Modality=MR) = %d, want 1", len(got))
	}
	if got := fs.Find(map[Tag]string{TagPatientID: "5MR2", TagModality: "SEG"}); len(got) != 0 {
		t.Errorf("Find with mismatched criteria = %d, want 0", len(got))
	}
	if got := fs.Find(map[Tag]string{TagPatientID: "5MR2", TagInstanceNumber: "3"}); len(got) != 1 {
		t.Errorf("Find(PatientID+InstanceNumber) = %d, want 1", len(got))
	}
}

func TestFileSetWriteEmpty(t *testing.T) {
	fs, err := NewFileSetBuilder().Write(t.TempDir())
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if len(fs.Roots()) != 0 || len(fs.Instances()) != 0 {
		t.Errorf("empty file-set has %d roots, %d instances", len(fs.Roots()), len(fs.Instances()))
	}
	if first := recordOffset(fs.File().DataSet, TagOffsetOfTheFirstDirectoryRecordOfTheRootDirectoryEntity); first != 0 {
		t.Errorf("first-record offset = %d, want 0", first)
	}
}

func TestFileSetBuilderAddFileGroupsHierarchy(t *testing.T) {
	b := NewFileSetBuilder()
	for i := range 2 {
		f := fileSetSample("P1", "1.2.3.1", "1.2.3.1.1", fmt.Sprintf("1.2.3.1.1.%d", i+1), fmt.Sprintf("%d", i+1))
		if err := b.AddFile(f); err != nil {
			t.Fatalf("AddFile: %v", err)
		}
	}
	fs, err := b.Write(t.TempDir())
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if len(fs.Roots()) != 1 {
		t.Fatalf("Roots = %d, want 1 patient", len(fs.Roots()))
	}
	series := fs.Roots()[0].Children()[0].Children()[0]
	if series.Type != "SERIES" || len(series.Children()) != 2 {
		t.Fatalf("series children = %d, want 2 instances", len(series.Children()))
	}
	if len(fs.Instances()) != 2 {
		t.Fatalf("Instances = %d, want 2", len(fs.Instances()))
	}
	if got := fs.Instances()[0].FileID(); len(got) != 4 || got[3] != "IM000000" {
		t.Errorf("first instance FileID = %v, want .../IM000000", got)
	}
	if got := fs.Instances()[1].FileID(); len(got) != 4 || got[3] != "IM000001" {
		t.Errorf("second instance FileID = %v, want .../IM000001", got)
	}
	// In-memory members are written as Part 10 files.
	loaded, err := fs.Instances()[1].Load()
	if err != nil {
		t.Fatalf("Load in-memory member: %v", err)
	}
	if n, ok := loaded.DataSet.GetInt(TagInstanceNumber); !ok || n != 2 {
		t.Errorf("loaded InstanceNumber = %d,%v want 2", n, ok)
	}
}

func TestFileSetBuilderSRDocumentRecord(t *testing.T) {
	b := NewFileSetBuilder()
	if err := b.AddFile(srSampleFile("1.2.3.200.1")); err != nil {
		t.Fatalf("AddFile(SR): %v", err)
	}
	fs, err := b.Write(t.TempDir())
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	leaf := fs.Instances()[0].Record
	if leaf.Type != "SR DOCUMENT" {
		t.Errorf("leaf record type = %q, want SR DOCUMENT", leaf.Type)
	}
	if v, ok := leaf.DataSet.GetString(TagCompletionFlag); !ok || v != "PARTIAL" {
		t.Errorf("SR record CompletionFlag = %q,%v", v, ok)
	}
	if _, ok := leaf.DataSet.GetSequence(TagConceptNameCodeSequence); !ok {
		t.Error("SR record is missing Concept Name Code Sequence")
	}
}

func TestFileSetBuilderRequiredKeyMissing(t *testing.T) {
	// basic-text-sr.dcm has a zero-length PatientID: the PATIENT record key is
	// required, so Add must fail fast with a typed error naming the tag.
	err := NewFileSetBuilder().Add(fixturePath("basic-text-sr.dcm"))
	var verr *ValueError
	if !errors.As(err, &verr) {
		t.Fatalf("Add = %v, want *ValueError", err)
	}
	if verr.Tag != TagPatientID {
		t.Errorf("error tag = %s, want PatientID", verr.Tag)
	}

	f := fileSetSample("P1", "1.2.3.1", "1.2.3.1.1", "1.2.3.1.1.1", "1")
	f.DataSet.Delete(TagStudyID)
	err = NewFileSetBuilder().AddFile(f)
	if !errors.As(err, &verr) || verr.Tag != TagStudyID {
		t.Errorf("AddFile without StudyID = %v, want *ValueError at StudyID", err)
	}
}

func TestFileSetBuilderDuplicateInstance(t *testing.T) {
	b := NewFileSetBuilder()
	if err := b.Add(fixturePath("liver.dcm")); err != nil {
		t.Fatal(err)
	}
	err := b.Add(fixturePath("liver.dcm"))
	var verr *ValueError
	if !errors.As(err, &verr) || verr.Tag != TagSOPInstanceUID {
		t.Errorf("duplicate Add = %v, want *ValueError at SOPInstanceUID", err)
	}
}

func TestFileSetBuilderIDValidation(t *testing.T) {
	for _, id := range []string{"lowercase", "SEVENTEEN_CHARS17", "BAD*CHAR"} {
		b := NewFileSetBuilder()
		b.SetID(id)
		if _, err := b.Write(t.TempDir()); err == nil {
			t.Errorf("Write accepted File-set ID %q", id)
		}
	}
}

func TestOpenFileSetRejectsNonDICOMDIR(t *testing.T) {
	_, err := OpenFileSet(fixturePath("liver.dcm"))
	var verr *ValueError
	if !errors.As(err, &verr) {
		t.Fatalf("OpenFileSet(liver.dcm) = %v, want *ValueError", err)
	}
}

func TestOpenFileSetAcceptsDirectoryPath(t *testing.T) {
	b := NewFileSetBuilder()
	if err := b.AddFile(fileSetSample("P1", "1.2.3.1", "1.2.3.1.1", "1.2.3.1.1.1", "1")); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if _, err := b.Write(root); err != nil {
		t.Fatalf("Write: %v", err)
	}
	fs, err := OpenFileSet(root)
	if err != nil {
		t.Fatalf("OpenFileSet(dir): %v", err)
	}
	if len(fs.Instances()) != 1 {
		t.Errorf("Instances = %d, want 1", len(fs.Instances()))
	}
	if fs.RootPath() != root {
		t.Errorf("RootPath = %q, want %q", fs.RootPath(), root)
	}
}
