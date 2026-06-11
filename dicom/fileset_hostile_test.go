package dicom

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeHostileFixtureSet writes a two-instance file-set and returns its DICOMDIR
// path. The mutation harness re-encodes the parsed DICOMDIR after tampering with
// offset elements: every offset element is a fixed-width UL, so the re-encoded file
// keeps every record at its original byte offset and only the tampered link changes.
func writeHostileFixtureSet(t *testing.T) string {
	t.Helper()
	b := NewFileSetBuilder()
	for i := range 2 {
		f := fileSetSample("P1", "1.2.3.1", "1.2.3.1.1", fmt.Sprintf("1.2.3.1.1.%d", i+1), fmt.Sprintf("%d", i+1))
		if err := b.AddFile(f); err != nil {
			t.Fatal(err)
		}
	}
	fs, err := b.Write(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return fs.Path()
}

// mutateDICOMDIR parses path, applies mutate to its main dataset and parsed record
// items, and writes the result back over path.
func mutateDICOMDIR(t *testing.T, path string, mutate func(main *DataSet, records []Item)) {
	t.Helper()
	f, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	seq, ok := f.DataSet.GetSequence(TagDirectoryRecordSequence)
	if !ok {
		t.Fatal("fixture DICOMDIR has no Directory Record Sequence")
	}
	mutate(f.DataSet, seq.items)
	if err := WriteFile(path, f); err != nil {
		t.Fatal(err)
	}
}

// recordType reads a parsed record item's Directory Record Type.
func recordType(it Item) string {
	v, _ := it.DataSet.GetString(TagDirectoryRecordType)
	return v
}

func TestMutationHarnessIsByteStable(t *testing.T) {
	// The hostile tests rely on re-encode keeping every item at its original byte
	// offset; an identity mutation must keep the file-set readable.
	path := writeHostileFixtureSet(t)
	mutateDICOMDIR(t, path, func(*DataSet, []Item) {})
	fs, err := OpenFileSet(path)
	if err != nil {
		t.Fatalf("identity rewrite broke the file-set: %v", err)
	}
	if len(fs.Instances()) != 2 {
		t.Errorf("Instances = %d, want 2", len(fs.Instances()))
	}
}

func TestOpenFileSetRejectsUnknownFirstOffset(t *testing.T) {
	path := writeHostileFixtureSet(t)
	mutateDICOMDIR(t, path, func(main *DataSet, _ []Item) {
		main.Set(ulElement(TagOffsetOfTheFirstDirectoryRecordOfTheRootDirectoryEntity, 99999999))
	})
	assertFileSetValueError(t, path, "does not match any record")
}

func TestOpenFileSetRejectsMisalignedOffset(t *testing.T) {
	path := writeHostileFixtureSet(t)
	mutateDICOMDIR(t, path, func(main *DataSet, records []Item) {
		// Two bytes past a real record start: inside the item header, not on it.
		main.Set(ulElement(TagOffsetOfTheFirstDirectoryRecordOfTheRootDirectoryEntity, records[0].fileOffset+2))
	})
	assertFileSetValueError(t, path, "does not match any record")
}

func TestOpenFileSetRejectsSelfLoopNextOffset(t *testing.T) {
	path := writeHostileFixtureSet(t)
	mutateDICOMDIR(t, path, func(_ *DataSet, records []Item) {
		// The first leaf's next-sibling offset points at itself.
		for _, it := range records {
			if recordType(it) == "IMAGE" {
				it.DataSet.Set(ulElement(TagOffsetOfTheNextDirectoryRecord, it.fileOffset))
				return
			}
		}
		t.Fatal("no IMAGE record in fixture")
	})
	assertFileSetValueError(t, path, "cycle")
}

func TestOpenFileSetRejectsMutualCycle(t *testing.T) {
	path := writeHostileFixtureSet(t)
	mutateDICOMDIR(t, path, func(_ *DataSet, records []Item) {
		var images []Item
		for _, it := range records {
			if recordType(it) == "IMAGE" {
				images = append(images, it)
			}
		}
		if len(images) != 2 {
			t.Fatalf("fixture has %d IMAGE records, want 2", len(images))
		}
		images[0].DataSet.Set(ulElement(TagOffsetOfTheNextDirectoryRecord, images[1].fileOffset))
		images[1].DataSet.Set(ulElement(TagOffsetOfTheNextDirectoryRecord, images[0].fileOffset))
	})
	assertFileSetValueError(t, path, "cycle")
}

func TestOpenFileSetRejectsAncestorChildLink(t *testing.T) {
	path := writeHostileFixtureSet(t)
	mutateDICOMDIR(t, path, func(_ *DataSet, records []Item) {
		// The SERIES record's lower-level entity points back at the PATIENT record:
		// the walk arrives at the patient twice.
		var patient, series Item
		for _, it := range records {
			switch recordType(it) {
			case "PATIENT":
				patient = it
			case "SERIES":
				series = it
			}
		}
		series.DataSet.Set(ulElement(TagOffsetOfReferencedLowerLevelDirectoryEntity, patient.fileOffset))
	})
	assertFileSetValueError(t, path, "cycle")
}

func TestOpenFileSetRejectsTraversalFileID(t *testing.T) {
	path := writeHostileFixtureSet(t)
	mutateDICOMDIR(t, path, func(_ *DataSet, records []Item) {
		// Same encoded length as the original PT/ST/SE/IM components (35 chars + pad),
		// so every record keeps its byte offset and only the file ID is hostile.
		comps := make([]string, 12)
		for i := range comps {
			comps[i] = ".."
		}
		for _, it := range records {
			if recordType(it) == "IMAGE" {
				it.DataSet.Set(Element{Tag: TagReferencedFileID, VR: VRCS, Value: NewStrings(VRCS, comps...)})
				return
			}
		}
	})
	assertFileSetValueError(t, path, "escape")
}

func TestOpenFileSetRejectsSeparatorFileID(t *testing.T) {
	path := writeHostileFixtureSet(t)
	mutateDICOMDIR(t, path, func(_ *DataSet, records []Item) {
		for _, it := range records {
			if recordType(it) == "IMAGE" {
				// "/etc/passwd........../X" is 23 chars... keep the encoded length at 35:
				// one component with separators padded to the original length.
				comp := "/" + strings.Repeat("A", 33) + "B"
				it.DataSet.Set(Element{Tag: TagReferencedFileID, VR: VRCS, Value: NewStrings(VRCS, comp[:35])})
				return
			}
		}
	})
	assertFileSetValueError(t, path, "escape")
}

func TestOpenFileSetRejectsNonExplicitVRLETransferSyntax(t *testing.T) {
	// PS3.10 §8.6: a DICOMDIR is always encoded in Explicit VR Little Endian. Any
	// other syntax must fail closed before offset resolution — for Deflated Explicit
	// VR LE the captured item offsets are relative to the inflated stream, not the
	// file, so resolving them would be wrong even when parsing succeeds.
	for _, ts := range []TransferSyntax{ImplicitVRLittleEndian, DeflatedExplicitVRLittleEndian} {
		t.Run(ts.Name(), func(t *testing.T) {
			path := writeHostileFixtureSet(t)
			f, err := ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			f.Meta.TransferSyntaxUID = ts
			if err := WriteFile(path, f); err != nil {
				t.Fatal(err)
			}
			_, err = OpenFileSet(path)
			var verr *ValueError
			if !errors.As(err, &verr) {
				t.Fatalf("OpenFileSet(%s DICOMDIR) = %v, want *ValueError", ts.Name(), err)
			}
			if verr.Tag != tagTransferSyntax {
				t.Errorf("error tag = %s, want Transfer Syntax UID", verr.Tag)
			}
			// The re-encode invalidates every captured offset, so an offset-resolution
			// error here would prove the transfer-syntax gate did not fire first.
			if !strings.Contains(verr.Error(), "Explicit VR Little Endian") {
				t.Errorf("error %q does not name the Explicit VR Little Endian requirement", verr.Error())
			}
		})
	}
}

func TestOpenFileSetTruncatedDICOMDIR(t *testing.T) {
	path := writeHostileFixtureSet(t)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	short := filepath.Join(t.TempDir(), "DICOMDIR")
	if err := os.WriteFile(short, raw[:len(raw)-7], 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenFileSet(short); err == nil {
		t.Error("OpenFileSet accepted a truncated DICOMDIR")
	}
}

// assertFileSetValueError opens the tampered file-set and requires a typed
// *ValueError whose message names the failure (never a panic, never success).
func assertFileSetValueError(t *testing.T, path, wantSubstring string) {
	t.Helper()
	_, err := OpenFileSet(path)
	var verr *ValueError
	if !errors.As(err, &verr) {
		t.Fatalf("OpenFileSet = %v, want *ValueError", err)
	}
	if !strings.Contains(verr.Error(), wantSubstring) {
		t.Errorf("error %q does not mention %q", verr.Error(), wantSubstring)
	}
}
