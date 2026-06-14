package dicom

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// deferredFixtureDataSet builds a dataset with one small text element and one
// large native OW pixel value of n bytes, plus the SOP identifiers DataSet.WriteFile
// needs. Identifiers are synthetic sentinels, never real PHI.
func deferredFixtureDataSet(n int) *DataSet {
	big := make([]byte, n)
	for i := range big {
		big[i] = byte(i % 251)
	}
	ds := NewDataSet()
	ds.Set(Element{Tag: TagSOPClassUID, VR: VRUI, Value: NewStrings(VRUI, "1.2.840.10008.5.1.4.1.1.7")})
	ds.Set(Element{Tag: TagSOPInstanceUID, VR: VRUI, Value: NewStrings(VRUI, "1.2.3.4.5.6.7.8.9")})
	ds.Set(Element{Tag: NewTag(0x0010, 0x0010), VR: VRPN, Value: NewStrings(VRPN, "ZZZTEST^DEFER^SENTINEL")})
	ds.Set(Element{Tag: NewTag(0x0028, 0x0010), VR: VRUS, Value: NewInts(VRUS, 2)})
	ds.Set(Element{Tag: TagPixelData, VR: VROW, Value: NewBytes(VROW, big)})
	return ds
}

// writeDeferredFixture writes the fixture dataset to a temp Part 10 file in ts and
// returns its path.
func writeDeferredFixture(t *testing.T, ts TransferSyntax, n int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "deferred.dcm")
	if err := deferredFixtureDataSet(n).WriteFile(path, ts); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestWithDeferredValuesConfig(t *testing.T) {
	cfg := newReadConfig()
	if cfg.deferralEnabled() {
		t.Error("deferral should default off")
	}
	cfg = newReadConfig(WithDeferredValues(0))
	if !cfg.deferralEnabled() || cfg.deferThreshold != 0 {
		t.Error("WithDeferredValues(0) should enable deferral with a zero threshold")
	}
	cfg = newReadConfig(WithDeferredValues(-1))
	if cfg.deferralEnabled() {
		t.Error("a negative threshold should disable deferral")
	}
}

func TestReadFileDefersLargeValues(t *testing.T) {
	const n = 64 << 10
	path := writeDeferredFixture(t, ExplicitVRLittleEndian, n)

	f, err := ReadFile(path, WithDeferredValues(1024))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	pn, ok := f.DataSet.Get(NewTag(0x0010, 0x0010))
	if !ok {
		t.Fatal("PatientName element missing")
	}
	if _, isDeferred := pn.Value.(*DeferredValue); isDeferred {
		t.Error("a value under the threshold must be materialised")
	}

	e, ok := f.DataSet.Get(TagPixelData)
	if !ok {
		t.Fatal("PixelData element missing")
	}
	dv, isDeferred := e.Value.(*DeferredValue)
	if !isDeferred {
		t.Fatalf("PixelData value should be deferred, got %T", e.Value)
	}
	if got := dv.EncodedLen(binaryLittleEndian()); got != n {
		t.Errorf("EncodedLen = %d, want %d (and it must not load the value)", got, n)
	}

	loaded, err := dv.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	b, ok := loaded.(*Bytes)
	if !ok {
		t.Fatalf("loaded value is %T, want *Bytes", loaded)
	}
	want := deferredFixtureDataSet(n).elems[TagPixelData].Value.(*Bytes).Bytes()
	if !bytes.Equal(b.Bytes(), want) {
		t.Error("deferred load returned different bytes from the source value")
	}

	again, err := dv.Load()
	if err != nil || again != loaded {
		t.Error("Load must cache and return the same decoded value")
	}
}

func TestDeferredThresholdBoundary(t *testing.T) {
	// PatientName is 22 bytes on the wire; defer strictly-larger-than semantics.
	path := writeDeferredFixture(t, ExplicitVRLittleEndian, 4096)
	f, err := ReadFile(path, WithDeferredValues(22))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	pn, _ := f.DataSet.Get(NewTag(0x0010, 0x0010))
	if _, isDeferred := pn.Value.(*DeferredValue); isDeferred {
		t.Error("a value exactly at the threshold must be materialised")
	}
	px, _ := f.DataSet.Get(TagPixelData)
	if _, isDeferred := px.Value.(*DeferredValue); !isDeferred {
		t.Error("a value over the threshold must be deferred")
	}
}

func TestDeferredAccessorsTransparent(t *testing.T) {
	ds := NewDataSet()
	ds.Set(Element{Tag: TagSOPClassUID, VR: VRUI, Value: NewStrings(VRUI, "1.2.840.10008.5.1.4.1.1.7")})
	ds.Set(Element{Tag: TagSOPInstanceUID, VR: VRUI, Value: NewStrings(VRUI, "1.2.3.4.5.6.7.8.9")})
	long := make([]byte, 6000)
	for i := range long {
		long[i] = 'a' + byte(i%26)
	}
	ds.Set(Element{Tag: NewTag(0x0008, 0x4000), VR: VRUT, Value: NewStrings(VRUT, string(long))})
	ints := make([]int64, 2000)
	for i := range ints {
		ints[i] = int64(i % 1000)
	}
	ds.Set(Element{Tag: NewTag(0x0028, 0x3006), VR: VRUS, Value: NewInts(VRUS, ints...)})

	path := filepath.Join(t.TempDir(), "accessors.dcm")
	if err := ds.WriteFile(path, ExplicitVRLittleEndian); err != nil {
		t.Fatalf("write: %v", err)
	}
	f, err := ReadFile(path, WithDeferredValues(512))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	for _, tag := range []Tag{NewTag(0x0008, 0x4000), NewTag(0x0028, 0x3006)} {
		e, _ := f.DataSet.Get(tag)
		if _, isDeferred := e.Value.(*DeferredValue); !isDeferred {
			t.Fatalf("%s should be deferred", tag)
		}
	}
	if got, ok := f.DataSet.GetString(NewTag(0x0008, 0x4000)); !ok || got != string(long) {
		t.Error("GetString must load a deferred text value transparently")
	}
	if got, ok := f.DataSet.GetInt(NewTag(0x0028, 0x3006)); !ok || got != 0 {
		t.Errorf("GetInt must load a deferred integer value transparently, got %d ok=%v", got, ok)
	}
}

func TestDeferredRoundTripByteIdentical(t *testing.T) {
	for _, ts := range []TransferSyntax{
		ImplicitVRLittleEndian, ExplicitVRLittleEndian, ExplicitVRBigEndian,
	} {
		t.Run(ts.Name(), func(t *testing.T) {
			path := writeDeferredFixture(t, ts, 32<<10)
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			f, err := ReadFile(path, WithDeferredValues(1024))
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			var out bytes.Buffer
			if err := Write(&out, f); err != nil {
				t.Fatalf("Write with deferred values: %v", err)
			}
			if !bytes.Equal(out.Bytes(), original) {
				t.Error("deferred round-trip is not byte-identical to the source")
			}
		})
	}
}

func TestReadRejectsDeferralWithoutFile(t *testing.T) {
	path := writeDeferredFixture(t, ExplicitVRLittleEndian, 4096)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Read(bytes.NewReader(raw), WithDeferredValues(0)); !errors.Is(err, errDeferralNeedsPath) {
		t.Errorf("Read must reject WithDeferredValues fail-closed, got %v", err)
	}
	if _, err := DecodeDataSet(bytes.NewReader(nil), ExplicitVRLittleEndian, WithDeferredValues(0)); !errors.Is(err, errDeferralNeedsPath) {
		t.Errorf("DecodeDataSet must reject WithDeferredValues fail-closed, got %v", err)
	}
}

func TestDeferredRejectsDeflated(t *testing.T) {
	path := writeDeferredFixture(t, DeflatedExplicitVRLittleEndian, 4096)
	if _, err := ReadFile(path, WithDeferredValues(0)); !errors.Is(err, errDeferralDeflated) {
		t.Errorf("deflated read must reject WithDeferredValues fail-closed, got %v", err)
	}
	// Without the option the same file still reads.
	if _, err := ReadFile(path); err != nil {
		t.Errorf("default deflated read should still succeed: %v", err)
	}
}

func TestDeferredEncapsulatedPixelData(t *testing.T) {
	src := filepath.Join("..", "testdata", "dicom", "liver_rle.dcm")

	plain, err := ReadFile(src)
	if err != nil {
		t.Fatalf("plain ReadFile: %v", err)
	}
	deferred, err := ReadFile(src, WithDeferredValues(1<<20))
	if err != nil {
		t.Fatalf("deferred ReadFile: %v", err)
	}

	e, _ := deferred.DataSet.Get(TagPixelData)
	dv, isDeferred := e.Value.(*DeferredValue)
	if !isDeferred {
		t.Fatalf("encapsulated PixelData should always be deferred under the option, got %T", e.Value)
	}
	if dv.EncodedLen(binaryLittleEndian()) != undefinedLength {
		t.Error("a deferred encapsulated stream must report the undefined-length sentinel")
	}

	loaded, err := dv.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	ev, ok := loaded.(*encapsulatedValue)
	if !ok {
		t.Fatalf("loaded value is %T, want *encapsulatedValue", loaded)
	}
	pe, _ := plain.DataSet.Get(TagPixelData)
	if !bytes.Equal(ev.stream, pe.Value.(*encapsulatedValue).stream) {
		t.Error("deferred load returned a different fragment stream")
	}

	// The pixel pipeline materialises the deferred stream transparently.
	if _, err := NewPixelData(deferred.DataSet, deferred.Meta.TransferSyntaxUID); err != nil {
		t.Errorf("NewPixelData over a deferred stream: %v", err)
	}
	// ReadPixelData threads the path through, so the option works there too.
	if _, err := ReadPixelData(src, WithDeferredValues(1<<20)); err != nil {
		t.Errorf("ReadPixelData with deferral: %v", err)
	}

	var plainOut, deferredOut bytes.Buffer
	if err := Write(&plainOut, plain); err != nil {
		t.Fatalf("plain Write: %v", err)
	}
	if err := Write(&deferredOut, deferred); err != nil {
		t.Fatalf("deferred Write: %v", err)
	}
	if !bytes.Equal(plainOut.Bytes(), deferredOut.Bytes()) {
		t.Error("deferred write differs from the materialised write")
	}
}

func TestDeferredInsideSequenceItem(t *testing.T) {
	big := make([]byte, 8192)
	for i := range big {
		big[i] = byte(i % 13)
	}
	item := NewDataSet()
	item.Set(Element{Tag: NewTag(0x0042, 0x0011), VR: VROB, Value: NewBytes(VROB, big)})
	ds := NewDataSet()
	ds.Set(Element{Tag: TagSOPClassUID, VR: VRUI, Value: NewStrings(VRUI, "1.2.840.10008.5.1.4.1.1.7")})
	ds.Set(Element{Tag: TagSOPInstanceUID, VR: VRUI, Value: NewStrings(VRUI, "1.2.3.4.5.6.7.8.9")})
	ds.Set(Element{Tag: NewTag(0x0040, 0x0100), VR: VRSQ, Value: NewSequenceValue(NewSequence(item))})

	path := filepath.Join(t.TempDir(), "seq.dcm")
	if err := ds.WriteFile(path, ExplicitVRLittleEndian); err != nil {
		t.Fatalf("write: %v", err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	f, err := ReadFile(path, WithDeferredValues(1024))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	seq, ok := f.DataSet.GetSequence(NewTag(0x0040, 0x0100))
	if !ok || len(seq.items) != 1 {
		t.Fatal("sequence not parsed")
	}
	ie, ok := seq.items[0].DataSet.Get(NewTag(0x0042, 0x0011))
	if !ok {
		t.Fatal("item element missing")
	}
	dv, isDeferred := ie.Value.(*DeferredValue)
	if !isDeferred {
		t.Fatalf("item element should be deferred, got %T", ie.Value)
	}
	loaded, err := dv.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !bytes.Equal(loaded.(*Bytes).Bytes(), big) {
		t.Error("deferred item element loaded different bytes")
	}

	var out bytes.Buffer
	if err := Write(&out, f); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !bytes.Equal(out.Bytes(), original) {
		t.Error("deferred sequence round-trip is not byte-identical")
	}
}

func TestDeferredCharsetCaptured(t *testing.T) {
	long := "Grüße-" + string(bytes.Repeat([]byte("ä"), 600))
	ds := NewDataSet()
	ds.Set(Element{Tag: TagSOPClassUID, VR: VRUI, Value: NewStrings(VRUI, "1.2.840.10008.5.1.4.1.1.7")})
	ds.Set(Element{Tag: TagSOPInstanceUID, VR: VRUI, Value: NewStrings(VRUI, "1.2.3.4.5.6.7.8.9")})
	ds.Set(Element{Tag: TagSpecificCharacterSet, VR: VRCS, Value: NewStrings(VRCS, "ISO_IR 192")})
	ds.Set(Element{Tag: NewTag(0x0010, 0x0010), VR: VRPN, Value: NewStrings(VRPN, long)})

	path := filepath.Join(t.TempDir(), "charset.dcm")
	if err := ds.WriteFile(path, ExplicitVRLittleEndian); err != nil {
		t.Fatalf("write: %v", err)
	}
	f, err := ReadFile(path, WithDeferredValues(64))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	cs, ok := f.DataSet.Get(TagSpecificCharacterSet)
	if !ok {
		t.Fatal("(0008,0005) missing")
	}
	if _, isDeferred := cs.Value.(*DeferredValue); isDeferred {
		t.Error("(0008,0005) must never be deferred")
	}
	pn, _ := f.DataSet.Get(NewTag(0x0010, 0x0010))
	if _, isDeferred := pn.Value.(*DeferredValue); !isDeferred {
		t.Fatal("the long PN should be deferred")
	}
	if got, ok := f.DataSet.GetString(NewTag(0x0010, 0x0010)); !ok || got != long {
		t.Error("the deferred PN must decode through the character set captured at its position")
	}
}

func TestDeferredLoadConcurrent(t *testing.T) {
	path := writeDeferredFixture(t, ExplicitVRLittleEndian, 32<<10)
	f, err := ReadFile(path, WithDeferredValues(1024))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	e, _ := f.DataSet.Get(TagPixelData)
	dv := e.Value.(*DeferredValue)

	var wg sync.WaitGroup
	results := make([]Value, 16)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			v, err := dv.Load()
			if err != nil {
				t.Errorf("concurrent Load: %v", err)
				return
			}
			results[i] = v
		}(i)
	}
	wg.Wait()
	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Fatal("concurrent loads must observe one shared decoded value")
		}
	}
}

func TestDeferredCloneSharesValue(t *testing.T) {
	path := writeDeferredFixture(t, ExplicitVRLittleEndian, 16<<10)
	f, err := ReadFile(path, WithDeferredValues(1024))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	clone := f.DataSet.Clone()
	ce, _ := clone.Get(TagPixelData)
	oe, _ := f.DataSet.Get(TagPixelData)
	if ce.Value != oe.Value {
		t.Error("Clone should share the immutable deferred value so one load serves both")
	}
}

func TestDeferredDeidentifyMaterialises(t *testing.T) {
	ds := deferredFixtureDataSet(16 << 10)
	ds.Set(Element{Tag: TagStudyInstanceUID, VR: VRUI, Value: NewStrings(VRUI, "1.2.3.4.5.100")})
	path := filepath.Join(t.TempDir(), "deid.dcm")
	if err := ds.WriteFile(path, ExplicitVRLittleEndian); err != nil {
		t.Fatalf("write: %v", err)
	}
	// Threshold 0 defers every non-empty value, so the U-action UID element is
	// deferred and applyUID must materialise it before remapping.
	f, err := ReadFile(path, WithDeferredValues(0))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	gen, err := NewUIDGenerator("1.2.840.99999.1")
	if err != nil {
		t.Fatal(err)
	}
	out, err := NewProfile(gen).Deidentify(f.DataSet)
	if err != nil {
		t.Fatalf("Deidentify: %v", err)
	}
	got, ok := out.GetString(TagStudyInstanceUID)
	if !ok {
		t.Fatal("StudyInstanceUID missing after de-identification")
	}
	if got == "1.2.3.4.5.100" {
		t.Error("a deferred UID must be loaded and remapped, never silently kept")
	}
}

func TestDeferredDeidentifyFailClosedOnLostSource(t *testing.T) {
	ds := deferredFixtureDataSet(16 << 10)
	ds.Set(Element{Tag: TagStudyInstanceUID, VR: VRUI, Value: NewStrings(VRUI, "1.2.3.4.5.100")})
	path := filepath.Join(t.TempDir(), "deid-lost.dcm")
	if err := ds.WriteFile(path, ExplicitVRLittleEndian); err != nil {
		t.Fatalf("write: %v", err)
	}
	f, err := ReadFile(path, WithDeferredValues(0))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	gen, err := NewUIDGenerator("1.2.840.99999.1")
	if err != nil {
		t.Fatal(err)
	}
	out, err := NewProfile(gen).Deidentify(f.DataSet)
	if err != nil {
		t.Fatalf("Deidentify: %v", err)
	}
	if _, ok := out.Get(TagStudyInstanceUID); ok {
		t.Error("an unloadable deferred UID must be removed, never kept as the original")
	}
}

// TestWriteFileInPlaceWithDeferredValuesPreservesSource is the round-trip guard: a
// file read with WithDeferredValues and written back to the SAME path must survive
// byte-identically. WriteFile must materialise the deferred values from the source
// before os.Create truncates it, or the write would destroy its own input.
func TestWriteFileInPlaceWithDeferredValuesPreservesSource(t *testing.T) {
	path := writeDeferredFixture(t, ExplicitVRLittleEndian, 4096) // PixelData deferred at threshold 64
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	f, err := ReadFile(path, WithDeferredValues(64))
	if err != nil {
		t.Fatalf("ReadFile deferred: %v", err)
	}
	if err := WriteFile(path, f); err != nil {
		t.Fatalf("in-place WriteFile after a deferred read: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("re-read after in-place write: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("in-place deferred round-trip changed the file: %d bytes, want %d", len(got), len(want))
	}
}
