package dicom

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"testing"
)

// TestOverlayArraySyntheticBitLayout verifies the LSB-first, row-major bit unpacking
// (PS3.5 §8.1.2) against a hand-built 4x4 overlay whose packed bytes are known exactly.
// 16 pixels = 2 bytes. We set a diagonal: pixels 0, 5, 10, 15 (true). With LSB-first
// packing, pixel i is bit i%8 of byte i/8, so byte0 has bits 0 and 5 set (0x21) and
// byte1 has bits 2 and 7 set (0x84).
func TestOverlayArraySyntheticBitLayout(t *testing.T) {
	const group = 0x6000
	ds := NewDataSet()
	ds.Set(Element{Tag: NewTag(group, overlayElemRows), VR: VRUS, Value: NewInts(VRUS, 4)})
	ds.Set(Element{Tag: NewTag(group, overlayElemColumns), VR: VRUS, Value: NewInts(VRUS, 4)})
	ds.Set(Element{Tag: NewTag(group, overlayElemBitsAllocated), VR: VRUS, Value: NewInts(VRUS, 1)})
	ds.Set(Element{Tag: NewTag(group, overlayElemBitPosition), VR: VRUS, Value: NewInts(VRUS, 0)})
	ds.Set(Element{Tag: NewTag(group, overlayElemType), VR: VRCS, Value: NewStrings(VRCS, "G")})
	ds.Set(Element{Tag: NewTag(group, overlayElemOrigin), VR: VRSS, Value: NewInts(VRSS, 1, 1)})
	// byte0 = bits {0,5} = 0b00100001 = 0x21; byte1 = bits {2,7} = 0b10000100 = 0x84.
	ds.Set(Element{Tag: NewTag(group, overlayElemData), VR: VROW, Value: NewBytes(VROW, []byte{0x21, 0x84})})

	o, err := ds.OverlayArray(group)
	if err != nil {
		t.Fatalf("OverlayArray: %v", err)
	}
	if o.Rows != 4 || o.Columns != 4 {
		t.Fatalf("dims = %dx%d, want 4x4", o.Rows, o.Columns)
	}
	if o.Type != "G" {
		t.Errorf("Type = %q, want G", o.Type)
	}
	if o.OriginRow != 1 || o.OriginColumn != 1 {
		t.Errorf("origin = (%d,%d), want (1,1)", o.OriginRow, o.OriginColumn)
	}

	want := map[int]bool{0: true, 5: true, 10: true, 15: true}
	for i := range 16 {
		if got := o.Bits[i]; got != want[i] {
			t.Errorf("Bits[%d] = %v, want %v", i, got, want[i])
		}
	}
	// At(row,column) maps the diagonal: (0,0),(1,1),(2,2),(3,3) set.
	for d := range 4 {
		if !o.At(d, d) {
			t.Errorf("At(%d,%d) = false, want true (diagonal)", d, d)
		}
	}
	if o.At(0, 1) {
		t.Errorf("At(0,1) = true, want false")
	}
}

// TestOverlayArrayFixture verifies pixel-exact extraction against a real overlay-bearing
// object (MR-SIEMENS, group 6000, 484x484, BitsAllocated 1). pydicom's documented
// overlay_array(0x6000) example reports the same (484, 484) shape; the set-bit count
// (323) is computed directly from the packed bytes, so this asserts the unpacking
// preserves every set bit.
func TestOverlayArrayFixture(t *testing.T) {
	path := filepath.Join("..", "testdata", "dicom", "MR-SIEMENS-DICOM-WithOverlays.dcm")
	f, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	groups := f.DataSet.OverlayGroups()
	if len(groups) != 1 || groups[0] != 0x6000 {
		t.Fatalf("OverlayGroups = %#v, want [6000]", groups)
	}

	o, err := f.DataSet.OverlayArray(0x6000)
	if err != nil {
		t.Fatalf("OverlayArray(6000): %v", err)
	}
	if o.Rows != 484 || o.Columns != 484 {
		t.Fatalf("dims = %dx%d, want 484x484", o.Rows, o.Columns)
	}
	if len(o.Bits) != 484*484 {
		t.Fatalf("len(Bits) = %d, want %d", len(o.Bits), 484*484)
	}

	// Cross-check the unpacked plane against the packed value field bit-for-bit.
	dataElem, _ := f.DataSet.Get(NewTag(0x6000, overlayElemData))
	packed, _ := binaryValueBytes(dataElem.Value)
	set := 0
	for i := range o.Bits {
		bit := packed[i/8]&(1<<(uint(i)%8)) != 0
		if o.Bits[i] != bit {
			t.Fatalf("Bits[%d] = %v, packed bit = %v", i, o.Bits[i], bit)
		}
		if o.Bits[i] {
			set++
		}
	}
	if set != 323 {
		t.Errorf("set-bit count = %d, want 323", set)
	}
}

// TestOverlayWaveformImplicitVRRoundTrip is the regression guard for the OB/OW data
// being read as the ambiguous VROBorOW placeholder under Implicit VR Little Endian, the
// most common transfer syntax. The dictionary marks Overlay Data (60xx,3000) and
// Waveform Data (5400,1010) as "OB or OW", so an implicit-VR read resolves both to
// VROBorOW. Decoding that placeholder as text yields a *Strings value (and corrupts any
// 0x5C byte by splitting on the value delimiter), so OverlayArray/WaveformArray used to
// reject valid implicit-VR data with "not a binary value". Encoding the dataset as
// Implicit VR LE and decoding it back exercises the real reader path, not a hand-built
// value.
func TestOverlayWaveformImplicitVRRoundTrip(t *testing.T) {
	const group = 0x6000
	// 4x4 overlay, diagonal set: byte0 bits {0,5} = 0x21, byte1 bits {2,7} = 0x84. A
	// 0x84 byte also guards against the text decoder mangling a high byte.
	overlayPacked := []byte{0x21, 0x84}

	// 1 channel, 4 SS samples little-endian: 100, -200, 0x5C5C (a backslash-laden word
	// that a text decoder would split), 1.
	waveSamples := []int16{100, -200, 0x5C5C, 1}
	wavePacked := make([]byte, len(waveSamples)*2)
	for i, v := range waveSamples {
		binary.LittleEndian.PutUint16(wavePacked[i*2:], uint16(v)) // #nosec G115 -- test fixture
	}

	mplx := NewDataSet()
	mplx.Set(Element{Tag: TagNumberOfWaveformChannels, VR: VRUS, Value: NewInts(VRUS, 1)})
	mplx.Set(Element{Tag: TagNumberOfWaveformSamples, VR: VRUL, Value: NewInts(VRUL, 4)})
	mplx.Set(Element{Tag: TagWaveformBitsAllocated, VR: VRUS, Value: NewInts(VRUS, 16)})
	mplx.Set(Element{Tag: TagWaveformSampleInterpretation, VR: VRCS, Value: NewStrings(VRCS, "SS")})
	mplx.Set(Element{Tag: TagWaveformData, VR: VROW, Value: NewBytes(VROW, wavePacked)})

	ds := NewDataSet()
	ds.Set(Element{Tag: NewTag(group, overlayElemRows), VR: VRUS, Value: NewInts(VRUS, 4)})
	ds.Set(Element{Tag: NewTag(group, overlayElemColumns), VR: VRUS, Value: NewInts(VRUS, 4)})
	ds.Set(Element{Tag: NewTag(group, overlayElemBitsAllocated), VR: VRUS, Value: NewInts(VRUS, 1)})
	ds.Set(Element{Tag: NewTag(group, overlayElemBitPosition), VR: VRUS, Value: NewInts(VRUS, 0)})
	ds.Set(Element{Tag: NewTag(group, overlayElemData), VR: VROW, Value: NewBytes(VROW, overlayPacked)})
	ds.Set(Element{Tag: TagWaveformSequence, VR: VRSQ, Value: NewSequenceValue(NewSequence(mplx))})

	var buf bytes.Buffer
	if err := EncodeDataSet(&buf, ds, ImplicitVRLittleEndian); err != nil {
		t.Fatalf("EncodeDataSet implicit: %v", err)
	}
	got, err := DecodeDataSet(bytes.NewReader(buf.Bytes()), ImplicitVRLittleEndian)
	if err != nil {
		t.Fatalf("DecodeDataSet implicit: %v", err)
	}

	// The Overlay Data element must have decoded to a binary value, not text.
	dataElem, ok := got.Get(NewTag(group, overlayElemData))
	if !ok {
		t.Fatal("decoded dataset has no Overlay Data")
	}
	if dataElem.VR != VROBorOW {
		t.Fatalf("Overlay Data VR = %v, want the implicit-VR VROBorOW placeholder", dataElem.VR)
	}

	o, err := got.OverlayArray(group)
	if err != nil {
		t.Fatalf("OverlayArray after implicit-VR round-trip: %v", err)
	}
	if o.Rows != 4 || o.Columns != 4 {
		t.Fatalf("dims = %dx%d, want 4x4", o.Rows, o.Columns)
	}
	for d := range 4 {
		if !o.At(d, d) {
			t.Errorf("At(%d,%d) = false, want true (diagonal)", d, d)
		}
	}

	w, err := got.WaveformArray(0, binary.LittleEndian)
	if err != nil {
		t.Fatalf("WaveformArray after implicit-VR round-trip: %v", err)
	}
	assertChannel(t, "implicit", w.Data[0], []float64{100, -200, 0x5C5C, 1})
}

func TestOverlayArrayRejectsEmbedded(t *testing.T) {
	const group = 0x6000
	ds := NewDataSet()
	ds.Set(Element{Tag: NewTag(group, overlayElemRows), VR: VRUS, Value: NewInts(VRUS, 2)})
	ds.Set(Element{Tag: NewTag(group, overlayElemColumns), VR: VRUS, Value: NewInts(VRUS, 2)})
	ds.Set(Element{Tag: NewTag(group, overlayElemBitsAllocated), VR: VRUS, Value: NewInts(VRUS, 1)})
	// Non-zero bit position => retired embedded encoding, must be rejected.
	ds.Set(Element{Tag: NewTag(group, overlayElemBitPosition), VR: VRUS, Value: NewInts(VRUS, 12)})
	ds.Set(Element{Tag: NewTag(group, overlayElemData), VR: VROW, Value: NewBytes(VROW, []byte{0x00, 0x00})})

	if _, err := ds.OverlayArray(group); err == nil {
		t.Fatal("OverlayArray accepted a retired embedded overlay, want error")
	}
}

func TestOverlayArrayRejectsBadGroup(t *testing.T) {
	ds := NewDataSet()
	for _, g := range []uint16{0x5FFE, 0x6001, 0x6100} {
		if _, err := ds.OverlayArray(g); err == nil {
			t.Errorf("OverlayArray(%04X) accepted a non-overlay group, want error", g)
		}
	}
}

func TestOverlayArrayShortData(t *testing.T) {
	const group = 0x6000
	ds := NewDataSet()
	ds.Set(Element{Tag: NewTag(group, overlayElemRows), VR: VRUS, Value: NewInts(VRUS, 4)})
	ds.Set(Element{Tag: NewTag(group, overlayElemColumns), VR: VRUS, Value: NewInts(VRUS, 4)})
	ds.Set(Element{Tag: NewTag(group, overlayElemData), VR: VROW, Value: NewBytes(VROW, []byte{0x01})}) // 8 bits, need 16
	if _, err := ds.OverlayArray(group); err == nil {
		t.Fatal("OverlayArray accepted truncated Overlay Data, want error")
	}
}
