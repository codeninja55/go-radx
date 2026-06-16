package dicom

import (
	"bytes"
	"math"
	"testing"
)

// setDS sets a DS element carrying the given lexical values on ds.
func setDS(t *testing.T, ds *DataSet, tag Tag, vals ...string) {
	t.Helper()
	decs := make([]Decimal, len(vals))
	for i, s := range vals {
		d, err := ParseDecimal(s)
		if err != nil {
			t.Fatalf("ParseDecimal(%q): %v", s, err)
		}
		decs[i] = d
	}
	ds.Set(Element{Tag: tag, VR: VRDS, Value: NewDecimals(VRDS, decs...)})
}

// lutItem builds a sequence item dataset with a LUT Descriptor (US) and US LUT Data.
func lutItem(descriptor []int64, data []int64) *DataSet {
	item := NewDataSet()
	item.Set(Element{Tag: TagLUTDescriptor, VR: VRUS, Value: NewInts(VRUS, descriptor...)})
	item.Set(Element{Tag: TagLUTData, VR: VRUS, Value: NewInts(VRUS, data...)})
	return item
}

const floatTol = 1e-9

func assertClose(t *testing.T, got, want float64, ctx string) {
	t.Helper()
	if math.Abs(got-want) > floatTol {
		t.Errorf("%s: got %.12g, want %.12g", ctx, got, want)
	}
}

func TestApplyModalityLUTRescale(t *testing.T) {
	// PS3.3 C.11.1.1.2: output = slope*x + intercept. CT Hounsfield rescale.
	ds := NewDataSet()
	setDS(t, ds, TagRescaleSlope, "1")
	setDS(t, ds, TagRescaleIntercept, "-1024")

	in := []float64{0, 1024, 2048}
	out, err := ApplyModalityLUT(in, ds)
	if err != nil {
		t.Fatalf("ApplyModalityLUT: %v", err)
	}
	want := []float64{-1024, 0, 1024}
	for i := range want {
		assertClose(t, out[i], want[i], "rescale")
	}
}

func TestApplyModalityLUTRescaleNonUnitSlope(t *testing.T) {
	ds := NewDataSet()
	setDS(t, ds, TagRescaleSlope, "2")
	setDS(t, ds, TagRescaleIntercept, "10")
	out, err := ApplyModalityLUT([]float64{0, 5, 100}, ds)
	if err != nil {
		t.Fatalf("ApplyModalityLUT: %v", err)
	}
	want := []float64{10, 20, 210}
	for i := range want {
		assertClose(t, out[i], want[i], "rescale-slope-2")
	}
}

func TestApplyModalityLUTAbsentIsIdentity(t *testing.T) {
	ds := NewDataSet()
	in := []float64{0, 100, 4095}
	out, err := ApplyModalityLUT(in, ds)
	if err != nil {
		t.Fatalf("ApplyModalityLUT: %v", err)
	}
	for i := range in {
		assertClose(t, out[i], in[i], "identity")
	}
}

func TestApplyModalityLUTTablePrecedence(t *testing.T) {
	// PS3.3 C.11.1.1.1: the Modality LUT Sequence table takes precedence over rescale.
	ds := NewDataSet()
	// Rescale present but must be ignored when the table is present.
	setDS(t, ds, TagRescaleSlope, "1")
	setDS(t, ds, TagRescaleIntercept, "-1024")
	// Descriptor [4 entries, first mapped 0, 16 bits]; data maps 0->10,1->20,2->30,3->40.
	seq := NewSequence(lutItem([]int64{4, 0, 16}, []int64{10, 20, 30, 40}))
	ds.Set(Element{Tag: TagModalityLUTSequence, VR: VRSQ, Value: NewSequenceValue(seq)})

	in := []float64{-2, 0, 1, 3, 5}
	out, err := ApplyModalityLUT(in, ds)
	if err != nil {
		t.Fatalf("ApplyModalityLUT: %v", err)
	}
	// -2 clamps to entry 0 (10); 5 clamps to last entry (40).
	want := []float64{10, 10, 20, 40, 40}
	for i := range want {
		assertClose(t, out[i], want[i], "modality-table")
	}
}

func TestApplyModalityLUTTableFirstMappedOffset(t *testing.T) {
	// First mapped value 1000: input 1000 -> Data[0].
	ds := NewDataSet()
	seq := NewSequence(lutItem([]int64{3, 1000, 16}, []int64{5, 15, 25}))
	ds.Set(Element{Tag: TagModalityLUTSequence, VR: VRSQ, Value: NewSequenceValue(seq)})
	out, err := ApplyModalityLUT([]float64{999, 1000, 1001, 1002, 1003}, ds)
	if err != nil {
		t.Fatalf("ApplyModalityLUT: %v", err)
	}
	want := []float64{5, 5, 15, 25, 25}
	for i := range want {
		assertClose(t, out[i], want[i], "first-mapped")
	}
}

func TestApplyWindowingLinear(t *testing.T) {
	// PS3.3 C.11.2.1.2.1 LINEAR: c=128, w=256, output range 0..255.
	// At centre: ((128-127.5)/255 + 0.5)*255 = 0.5 + 127.5 = 128.0 exactly.
	ds := NewDataSet()
	setDS(t, ds, TagWindowCenter, "128")
	setDS(t, ds, TagWindowWidth, "256")

	in := []float64{0, 128, 255, 256}
	out, err := ApplyWindowing(in, ds, 0, 0, 255)
	if err != nil {
		t.Fatalf("ApplyWindowing: %v", err)
	}
	want := []float64{0, 128, 255, 255}
	for i := range want {
		assertClose(t, out[i], want[i], "linear")
	}
}

func TestApplyWindowingLinearExact(t *testing.T) {
	// PS3.3 C.11.2.1.2.1 LINEAR_EXACT: c=128, w=256, output range 0..255.
	// At centre: ((128-128)/256 + 0.5)*255 = 127.5 (the one-LSB difference vs LINEAR).
	ds := NewDataSet()
	setDS(t, ds, TagWindowCenter, "128")
	setDS(t, ds, TagWindowWidth, "256")
	ds.Set(Element{Tag: TagVOILUTFunction, VR: VRCS, Value: NewStrings(VRCS, "LINEAR_EXACT")})

	in := []float64{0, 128, 256, 257}
	out, err := ApplyWindowing(in, ds, 0, 0, 255)
	if err != nil {
		t.Fatalf("ApplyWindowing: %v", err)
	}
	want := []float64{0, 127.5, 255, 255}
	for i := range want {
		assertClose(t, out[i], want[i], "linear-exact")
	}
}

func TestApplyWindowingLinearVsExactDiffer(t *testing.T) {
	// The defining off-by-one: at the window centre LINEAR and LINEAR_EXACT differ.
	base := NewDataSet()
	setDS(t, base, TagWindowCenter, "2048")
	setDS(t, base, TagWindowWidth, "4096")

	linearOut, err := ApplyWindowing([]float64{2048}, base, 0, 0, 255)
	if err != nil {
		t.Fatalf("linear: %v", err)
	}
	exact := base.Clone()
	exact.Set(Element{Tag: TagVOILUTFunction, VR: VRCS, Value: NewStrings(VRCS, "LINEAR_EXACT")})
	exactOut, err := ApplyWindowing([]float64{2048}, exact, 0, 0, 255)
	if err != nil {
		t.Fatalf("exact: %v", err)
	}
	// LINEAR: cc=2047.5, ww=4095 -> ((2048-2047.5)/4095 + 0.5)*255.
	wantLinear := ((2048.0-2047.5)/4095.0 + 0.5) * 255.0
	// LINEAR_EXACT: ((2048-2048)/4096 + 0.5)*255 = 127.5.
	wantExact := 127.5
	assertClose(t, linearOut[0], wantLinear, "linear-centre")
	assertClose(t, exactOut[0], wantExact, "exact-centre")
	if math.Abs(linearOut[0]-exactOut[0]) < 1e-6 {
		t.Errorf("LINEAR and LINEAR_EXACT should differ at the centre: %.9g vs %.9g", linearOut[0], exactOut[0])
	}
}

func TestApplyWindowingSigmoid(t *testing.T) {
	// PS3.3 C.11.2.1.3.1 SIGMOID: y = (ymax-ymin)/(1+exp(-4(x-c)/w)) + ymin.
	ds := NewDataSet()
	setDS(t, ds, TagWindowCenter, "0")
	setDS(t, ds, TagWindowWidth, "4")
	ds.Set(Element{Tag: TagVOILUTFunction, VR: VRCS, Value: NewStrings(VRCS, "SIGMOID")})

	in := []float64{0, 4, -4}
	out, err := ApplyWindowing(in, ds, 0, 0, 255)
	if err != nil {
		t.Fatalf("ApplyWindowing: %v", err)
	}
	want := []float64{
		255.0 / (1 + math.Exp(0)),  // x=0 -> 127.5
		255.0 / (1 + math.Exp(-4)), // x=4
		255.0 / (1 + math.Exp(4)),  // x=-4
	}
	for i := range want {
		assertClose(t, out[i], want[i], "sigmoid")
	}
	assertClose(t, out[0], 127.5, "sigmoid-centre")
}

func TestApplyWindowingMultipleIndexedPairs(t *testing.T) {
	// PS3.3 allows 1-n Window Center/Width; index selects the pair.
	ds := NewDataSet()
	setDS(t, ds, TagWindowCenter, "40", "400")
	setDS(t, ds, TagWindowWidth, "80", "2000")

	// Index 0 (soft tissue: c=40,w=80) vs index 1 (lung: c=400,w=2000) for x=40.
	out0, err := ApplyWindowing([]float64{40}, ds, 0, 0, 255)
	if err != nil {
		t.Fatalf("index 0: %v", err)
	}
	out1, err := ApplyWindowing([]float64{40}, ds, 1, 0, 255)
	if err != nil {
		t.Fatalf("index 1: %v", err)
	}
	// Index 0 LINEAR: cc=39.5, ww=79 -> ((40-39.5)/79+0.5)*255.
	assertClose(t, out0[0], ((40.0-39.5)/79.0+0.5)*255.0, "index0")
	// Index 1 LINEAR: cc=399.5, ww=1999 -> ((40-399.5)/1999+0.5)*255.
	assertClose(t, out1[0], ((40.0-399.5)/1999.0+0.5)*255.0, "index1")
}

func TestApplyWindowingIndexOutOfRange(t *testing.T) {
	ds := NewDataSet()
	setDS(t, ds, TagWindowCenter, "128")
	setDS(t, ds, TagWindowWidth, "256")
	if _, err := ApplyWindowing([]float64{0}, ds, 1, 0, 255); err == nil {
		t.Fatal("expected error for out-of-range window index")
	}
}

func TestApplyWindowingRejectsBadWidth(t *testing.T) {
	// LINEAR requires w >= 1 (PS3.3 C.11.2.1.2.1); width 0 has no defined output.
	ds := NewDataSet()
	setDS(t, ds, TagWindowCenter, "128")
	setDS(t, ds, TagWindowWidth, "0")
	if _, err := ApplyWindowing([]float64{0}, ds, 0, 0, 255); err == nil {
		t.Fatal("expected error for window width 0")
	}
}

func TestApplyWindowingAbsentIsIdentity(t *testing.T) {
	ds := NewDataSet()
	in := []float64{0, 100, 4095}
	out, err := ApplyWindowing(in, ds, 0, 0, 255)
	if err != nil {
		t.Fatalf("ApplyWindowing: %v", err)
	}
	for i := range in {
		assertClose(t, out[i], in[i], "windowing-identity")
	}
}

func TestApplyVOILUTTablePrecedence(t *testing.T) {
	// PS3.3 C.11.2.1.1: a VOI LUT Sequence table takes precedence over windowing.
	ds := NewDataSet()
	setDS(t, ds, TagWindowCenter, "128")
	setDS(t, ds, TagWindowWidth, "256")
	seq := NewSequence(lutItem([]int64{4, 0, 8}, []int64{0, 85, 170, 255}))
	ds.Set(Element{Tag: TagVOILUTSequence, VR: VRSQ, Value: NewSequenceValue(seq)})

	in := []float64{-1, 0, 1, 2, 3, 4}
	out, err := ApplyVOILUT(in, ds, 0, 0, 255)
	if err != nil {
		t.Fatalf("ApplyVOILUT: %v", err)
	}
	want := []float64{0, 0, 85, 170, 255, 255}
	for i := range want {
		assertClose(t, out[i], want[i], "voi-table")
	}
}

func TestApplyVOILUTMultipleItemsIndexed(t *testing.T) {
	ds := NewDataSet()
	seq := NewSequence(
		lutItem([]int64{2, 0, 8}, []int64{10, 20}),
		lutItem([]int64{2, 0, 8}, []int64{100, 200}),
	)
	ds.Set(Element{Tag: TagVOILUTSequence, VR: VRSQ, Value: NewSequenceValue(seq)})

	out0, err := ApplyVOILUT([]float64{1}, ds, 0, 0, 255)
	if err != nil {
		t.Fatalf("index 0: %v", err)
	}
	out1, err := ApplyVOILUT([]float64{1}, ds, 1, 0, 255)
	if err != nil {
		t.Fatalf("index 1: %v", err)
	}
	assertClose(t, out0[0], 20, "voi-item0")
	assertClose(t, out1[0], 200, "voi-item1")
}

func TestApplyVOILUTFallsBackToWindowing(t *testing.T) {
	// No VOI LUT Sequence: ApplyVOILUT must apply windowing.
	ds := NewDataSet()
	setDS(t, ds, TagWindowCenter, "128")
	setDS(t, ds, TagWindowWidth, "256")
	out, err := ApplyVOILUT([]float64{128}, ds, 0, 0, 255)
	if err != nil {
		t.Fatalf("ApplyVOILUT: %v", err)
	}
	assertClose(t, out[0], 128, "voi-fallback")
}

func TestApplyVOILUTOWData(t *testing.T) {
	// LUT Data carried as OW (raw little-endian 16-bit words).
	ds := NewDataSet()
	item := NewDataSet()
	item.Set(Element{Tag: TagLUTDescriptor, VR: VRUS, Value: NewInts(VRUS, 2, 0, 16)})
	// Two 16-bit words: 0x0100 = 256, 0x0200 = 512.
	item.Set(Element{Tag: TagLUTData, VR: VROW, Value: NewBytes(VROW, []byte{0x00, 0x01, 0x00, 0x02})})
	seq := NewSequence(item)
	ds.Set(Element{Tag: TagVOILUTSequence, VR: VRSQ, Value: NewSequenceValue(seq)})

	out, err := ApplyVOILUT([]float64{0, 1}, ds, 0, 0, 255)
	if err != nil {
		t.Fatalf("ApplyVOILUT: %v", err)
	}
	assertClose(t, out[0], 256, "ow-entry0")
	assertClose(t, out[1], 512, "ow-entry1")
}

func TestModalityThenVOIPipeline(t *testing.T) {
	// End-to-end CT-style pipeline: rescale to HU then window (PS3.3 ordering).
	ds := NewDataSet()
	setDS(t, ds, TagRescaleSlope, "1")
	setDS(t, ds, TagRescaleIntercept, "-1024")
	setDS(t, ds, TagWindowCenter, "0")   // soft-tissue centre in HU
	setDS(t, ds, TagWindowWidth, "2048") // wide window

	stored := []float64{1024} // -> 0 HU after rescale
	hu, err := ApplyModalityLUT(stored, ds)
	if err != nil {
		t.Fatalf("modality: %v", err)
	}
	assertClose(t, hu[0], 0, "hu")
	disp, err := ApplyVOILUT(hu, ds, 0, 0, 255)
	if err != nil {
		t.Fatalf("voi: %v", err)
	}
	// LINEAR at x=0, c=0, w=2048: cc=-0.5, ww=2047 -> ((0+0.5)/2047+0.5)*255.
	assertClose(t, disp[0], ((0.0+0.5)/2047.0+0.5)*255.0, "pipeline")
}

func TestLUTDescriptorZeroCountIs65536(t *testing.T) {
	// PS3.3 C.11.1.1.1: a descriptor entry count of 0 means 2^16 entries.
	data := make([]int64, 3)
	data[0], data[1], data[2] = 7, 8, 9
	item := lutItem([]int64{0, 0, 16}, data)
	lut, err := lutFromItem(item)
	if err != nil {
		t.Fatalf("lutFromItem: %v", err)
	}
	if len(lut.Data) != 3 {
		t.Fatalf("expected 3 data entries retained, got %d", len(lut.Data))
	}
	// Index 2 maps to 9; anything beyond clamps to the last present entry.
	assertClose(t, lut.Apply(2), 9, "zero-count")
	assertClose(t, lut.Apply(100), 9, "zero-count-clamp")
}

func TestLUTDescriptorCountUnsignedWithSignedFirstMapped(t *testing.T) {
	// PS3.3 C.11.1.1.1: the first descriptor value (entry count) is ALWAYS unsigned,
	// even under an SS descriptor; only the second value (first-mapped) is signed. An
	// on-disk count word of 40000 (> 32767) is sign-extended to -25536 by the SS reader
	// (decodeInts does int64(int16(...))), which previously made the data[:count] slice
	// panic with a negative bound. The first-mapped value (-2000) legitimately stays
	// negative.
	var unsignedCount uint16 = 40000         // > 32767, the boundary the bug crossed
	countWord := int64(int16(unsignedCount)) // SS reinterpretation (negative on disk)
	if uint16(countWord) != unsignedCount {
		t.Fatalf("test setup: uint16(%d) = %d, want %d", countWord, uint16(countWord), unsignedCount)
	}
	item := NewDataSet()
	item.Set(Element{Tag: TagLUTDescriptor, VR: VRSS, Value: NewInts(VRSS, countWord, -2000, 16)})
	// A 40000-entry table is declared; supply two entries and let clamping cover the
	// rest. The descriptor maths is what is under test, not a full table.
	item.Set(Element{Tag: TagLUTData, VR: VRUS, Value: NewInts(VRUS, 11, 22)})

	lut, err := lutFromItem(item)
	if err != nil {
		t.Fatalf("lutFromItem: %v", err)
	}
	if lut.FirstMapped != -2000 {
		t.Fatalf("FirstMapped = %d, want -2000 (signed first-mapped must be preserved)", lut.FirstMapped)
	}
	if len(lut.Data) != 2 {
		t.Fatalf("len(Data) = %d, want 2 (data shorter than the declared count is retained)", len(lut.Data))
	}
	// FirstMapped -2000 -> Data[0]; -1999 -> Data[1]; below clamps to first, above to last.
	assertClose(t, lut.Apply(-2001), 11, "count-unsigned-clamp-low")
	assertClose(t, lut.Apply(-2000), 11, "count-unsigned-entry0")
	assertClose(t, lut.Apply(-1999), 22, "count-unsigned-entry1")
	assertClose(t, lut.Apply(5000), 22, "count-unsigned-clamp-high")
}

func TestLUTDescriptorCountBoundaryAt32768(t *testing.T) {
	// Count 32768 on disk is int16 -32768 after SS decoding; read unsigned it is 32768,
	// not a negative bound. The table here has fewer entries, which the clamp tolerates.
	item := NewDataSet()
	item.Set(Element{Tag: TagLUTDescriptor, VR: VRSS, Value: NewInts(VRSS, -32768, 0, 16)})
	item.Set(Element{Tag: TagLUTData, VR: VRUS, Value: NewInts(VRUS, 1, 2, 3)})
	lut, err := lutFromItem(item)
	if err != nil {
		t.Fatalf("lutFromItem: %v", err)
	}
	if len(lut.Data) != 3 {
		t.Fatalf("len(Data) = %d, want 3", len(lut.Data))
	}
	assertClose(t, lut.Apply(2), 3, "boundary-32768")
}

func TestApplyVOILUTImplicitVRLE(t *testing.T) {
	// Bug regression: a VOI LUT Sequence read from Implicit VR LE materialises the
	// descriptor (VRUSorSS) and data (VRUSorOW) as *Strings, not *Ints/*Bytes. The LUT
	// extraction must handle that implicit-VR form rather than returning a
	// missing-descriptor error.
	src := NewDataSet()
	item := NewDataSet()
	item.Set(Element{Tag: TagLUTDescriptor, VR: VRUS, Value: NewInts(VRUS, 4, 0, 16)})
	item.Set(Element{Tag: TagLUTData, VR: VRUS, Value: NewInts(VRUS, 0, 85, 170, 255)})
	src.Set(Element{Tag: TagVOILUTSequence, VR: VRSQ, Value: NewSequenceValue(NewSequence(item))})

	ds := roundTripImplicitVRLE(t, src)

	in := []float64{-1, 0, 1, 2, 3, 4}
	out, err := ApplyVOILUT(in, ds, 0, 0, 255)
	if err != nil {
		t.Fatalf("ApplyVOILUT on implicit-VR dataset: %v", err)
	}
	want := []float64{0, 0, 85, 170, 255, 255}
	for i := range want {
		assertClose(t, out[i], want[i], "implicit-voi-table")
	}
}

func TestApplyModalityLUTImplicitVRLE(t *testing.T) {
	// Same regression on the Modality LUT path, with a negative first-mapped value to
	// exercise signed-descriptor decoding through the implicit-VR *Strings form.
	src := NewDataSet()
	item := NewDataSet()
	item.Set(Element{Tag: TagLUTDescriptor, VR: VRSS, Value: NewInts(VRSS, 3, -1000, 16)})
	item.Set(Element{Tag: TagLUTData, VR: VRUS, Value: NewInts(VRUS, 5, 15, 25)})
	src.Set(Element{Tag: TagModalityLUTSequence, VR: VRSQ, Value: NewSequenceValue(NewSequence(item))})

	ds := roundTripImplicitVRLE(t, src)

	out, err := ApplyModalityLUT([]float64{-1001, -1000, -999, -998, -997}, ds)
	if err != nil {
		t.Fatalf("ApplyModalityLUT on implicit-VR dataset: %v", err)
	}
	want := []float64{5, 5, 15, 25, 25}
	for i := range want {
		assertClose(t, out[i], want[i], "implicit-modality-table")
	}
}

// roundTripImplicitVRLE encodes ds as Implicit VR LE and reads it back, so the LUT
// elements carry the unresolved placeholder VRs the implicit-VR reader produces.
func roundTripImplicitVRLE(t *testing.T, ds *DataSet) *DataSet {
	t.Helper()
	var buf bytes.Buffer
	if err := EncodeDataSet(&buf, ds, ImplicitVRLittleEndian); err != nil {
		t.Fatalf("EncodeDataSet: %v", err)
	}
	br := newBoundedReader(bytes.NewReader(buf.Bytes()), defaultMaxElementLen)
	got, err := readDataSet(br, ImplicitVRLittleEndian, newReadConfig())
	if err != nil {
		t.Fatalf("readDataSet: %v", err)
	}
	return got
}
