package dicom

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestDeferredLoadSourceShrank: a source truncated between the read and the load
// must surface a typed *DeferredLoadError, never a panic or a short value.
func TestDeferredLoadSourceShrank(t *testing.T) {
	path := writeDeferredFixture(t, ExplicitVRLittleEndian, 32<<10)
	f, err := ReadFile(path, WithDeferredValues(1024))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	e, _ := f.DataSet.Get(TagPixelData)
	dv := e.Value.(*DeferredValue)

	if err := os.Truncate(path, dv.offset+dv.length/2); err != nil {
		t.Fatal(err)
	}
	_, err = dv.Load()
	var dle *DeferredLoadError
	if !errors.As(err, &dle) {
		t.Fatalf("Load on a shrunk source = %v, want *DeferredLoadError", err)
	}
	if dle.Tag != TagPixelData {
		t.Errorf("error names tag %s, want %s", dle.Tag, TagPixelData)
	}

	// The write path must propagate the typed failure, never emit a wrong length.
	var out bytes.Buffer
	if err := Write(&out, f); !errors.As(err, &dle) {
		t.Errorf("Write over an unloadable deferred value = %v, want *DeferredLoadError", err)
	}
}

// TestDeferredLoadSourceRemoved: a deleted source is a typed load error.
func TestDeferredLoadSourceRemoved(t *testing.T) {
	path := writeDeferredFixture(t, ExplicitVRLittleEndian, 16<<10)
	f, err := ReadFile(path, WithDeferredValues(1024))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	e, _ := f.DataSet.Get(TagPixelData)
	_, err = e.Value.(*DeferredValue).Load()
	var dle *DeferredLoadError
	if !errors.As(err, &dle) {
		t.Fatalf("Load on a removed source = %v, want *DeferredLoadError", err)
	}
	// The accessor path reads an unloadable value as absent rather than panicking.
	if _, ok := f.DataSet.GetStrings(TagPixelData); ok {
		t.Error("GetStrings over an unloadable deferred value must report absent")
	}
}

// TestDeferredEncapsulatedLoadRevalidates: a deferred encapsulated window whose
// item structure was corrupted on disk after the read must fail typed on load —
// the load re-parses the stream through the same validator the read used.
func TestDeferredEncapsulatedLoadRevalidates(t *testing.T) {
	src := filepath.Join("..", "testdata", "dicom", "liver_rle.dcm")
	raw, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "liver_rle.dcm")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := ReadFile(path, WithDeferredValues(1024))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	e, _ := f.DataSet.Get(TagPixelData)
	dv := e.Value.(*DeferredValue)

	// Overwrite the first item header inside the recorded window with a foreign tag.
	corrupted := append([]byte(nil), raw...)
	copy(corrupted[dv.offset:dv.offset+8], []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})
	if err := os.WriteFile(path, corrupted, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = dv.Load()
	var dle *DeferredLoadError
	if !errors.As(err, &dle) {
		t.Fatalf("Load over a corrupted fragment stream = %v, want *DeferredLoadError", err)
	}
}
