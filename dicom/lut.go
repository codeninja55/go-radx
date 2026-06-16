package dicom

import (
	"encoding/binary"
	"math"
)

// The pixel presentation pipeline turns stored pixel values into the values a viewer
// renders. PS3.3 §C.11 defines two ordered grayscale transformations:
//
//  1. Modality LUT (PS3.3 §C.11.1): stored values -> manufacturer-independent
//     measured units (e.g. Hounsfield units for CT). Either a linear Rescale
//     Slope/Intercept (§C.11.1.1.2) or a Modality LUT Sequence table.
//  2. VOI LUT (PS3.3 §C.11.2): modality-LUT output -> the value-of-interest range a
//     window selects, via Window Center/Width (§C.11.2.1.2 / §C.11.2.1.3) or a VOI
//     LUT Sequence table (§C.11.2.1.1).
//
// These mirror pydicom's apply_modality_lut, apply_voi_lut, apply_windowing, and
// apply_voi. Each function consumes float64 stored/intermediate values so the same
// code serves signed and unsigned pixels without per-call sign branching; callers
// decode a frame to []float64 (signed values negative) before applying the pipeline.

// VOILUTFunction selects the windowing transfer function (PS3.3 §C.11.2.1.2 / C.11.2.1.3).
type VOILUTFunction int

const (
	// VOILUTFunctionLinear is the default windowing function (PS3.3 §C.11.2.1.2.1).
	VOILUTFunctionLinear VOILUTFunction = iota
	// VOILUTFunctionLinearExact is the exact linear windowing function (PS3.3 §C.11.2.1.2.1).
	VOILUTFunctionLinearExact
	// VOILUTFunctionSigmoid is the sigmoid windowing function (PS3.3 §C.11.2.1.3.1).
	VOILUTFunctionSigmoid
)

// parseVOILUTFunction maps the (0028,1056) VOI LUT Function defined term to a
// VOILUTFunction. An absent or unrecognised term defaults to LINEAR, matching
// pydicom (which treats anything other than LINEAR_EXACT / SIGMOID as LINEAR).
func parseVOILUTFunction(term string) VOILUTFunction {
	switch term {
	case "LINEAR_EXACT":
		return VOILUTFunctionLinearExact
	case "SIGMOID":
		return VOILUTFunctionSigmoid
	default:
		return VOILUTFunctionLinear
	}
}

// LUT is a parsed (0028,3002) LUT Descriptor + (0028,3006) LUT Data pair, the table
// form shared by the Modality LUT, VOI LUT, and Presentation LUT modules (PS3.3
// §C.11.1.1.1 / §C.11.2.1.1). The descriptor's three values are: the number of
// entries (0 meaning 2^16), the first stored value the table maps, and the number of
// bits per entry. FirstMapped is read with the descriptor's declared VR so a signed
// LUT Descriptor (US or SS) carries a negative first-mapped value correctly.
type LUT struct {
	// FirstMapped is the input value mapped by Data[0]; inputs below it clamp to
	// Data[0] and inputs at or above FirstMapped+len(Data) clamp to the last entry.
	FirstMapped int
	// Data holds the output values, one per table entry, in input order.
	Data []int
}

// Apply maps one input value through the table, clamping out-of-range inputs to the
// first or last entry per PS3.3 §C.11.1.1.1 (values <= the first mapped value use the
// first entry; values >= the last use the last). An empty table returns x unchanged.
func (l LUT) Apply(x float64) float64 {
	n := len(l.Data)
	if n == 0 {
		return x
	}
	idx := int(math.Floor(x)) - l.FirstMapped
	if idx < 0 {
		idx = 0
	} else if idx >= n {
		idx = n - 1
	}
	return float64(l.Data[idx])
}

// ApplyModalityLUT applies the Modality LUT transformation in ds to in, returning a
// new slice (in is not modified). It mirrors pydicom apply_modality_lut: a Modality
// LUT Sequence (0028,3000) table takes precedence when present; otherwise Rescale
// Slope (0028,1053) and Rescale Intercept (0028,1052) are applied as
// output = slope*x + intercept (PS3.3 §C.11.1.1.2). With neither present the input is
// returned unchanged.
func ApplyModalityLUT(in []float64, ds *DataSet) ([]float64, error) {
	out := make([]float64, len(in))

	if lut, ok, err := modalityLUT(ds); err != nil {
		return nil, err
	} else if ok {
		for i, x := range in {
			out[i] = lut.Apply(x)
		}
		return out, nil
	}

	slope, hasSlope := decimalFloat(ds, TagRescaleSlope)
	intercept, hasIntercept := decimalFloat(ds, TagRescaleIntercept)
	if !hasSlope && !hasIntercept {
		copy(out, in)
		return out, nil
	}
	if !hasSlope {
		slope = 1
	}
	for i, x := range in {
		out[i] = slope*x + intercept
	}
	return out, nil
}

// modalityLUT reads the Modality LUT Sequence (0028,3000) table if present. ok is
// false when the sequence is absent or empty.
func modalityLUT(ds *DataSet) (LUT, bool, error) {
	seq, ok := ds.GetSequence(TagModalityLUTSequence)
	if !ok || seq.Len() == 0 {
		return LUT{}, false, nil
	}
	for item := range seq.Items() {
		lut, err := lutFromItem(item.DataSet)
		if err != nil {
			return LUT{}, false, err
		}
		return lut, true, nil
	}
	return LUT{}, false, nil
}

// ApplyVOILUT applies the VOI LUT transformation in ds to in, returning a new slice.
// It mirrors pydicom apply_voi_lut: a VOI LUT Sequence (0028,3010) table takes
// precedence over windowing when present; otherwise Window Center/Width (0028,1050 /
// 0028,1051) windowing is applied. index selects which VOI LUT item or which
// center/width pair to use (PS3.3 allows multiple). With neither present the input is
// returned unchanged. yMin/yMax bound the windowing output range (e.g. 0 and 255 for
// 8-bit display); they are ignored on the table path, whose output range is the
// table's own.
func ApplyVOILUT(in []float64, ds *DataSet, index int, yMin, yMax float64) ([]float64, error) {
	if lut, ok, err := voiLUT(ds, index); err != nil {
		return nil, err
	} else if ok {
		out := make([]float64, len(in))
		for i, x := range in {
			out[i] = lut.Apply(x)
		}
		return out, nil
	}
	return ApplyWindowing(in, ds, index, yMin, yMax)
}

// voiLUT reads the index-th VOI LUT Sequence (0028,3010) table if present. ok is
// false when the sequence is absent, empty, or index is out of range.
func voiLUT(ds *DataSet, index int) (LUT, bool, error) {
	seq, ok := ds.GetSequence(TagVOILUTSequence)
	if !ok || seq.Len() == 0 {
		return LUT{}, false, nil
	}
	if index < 0 || index >= seq.Len() {
		return LUT{}, false, &ValueError{
			Tag: TagVOILUTSequence, VR: VRSQ, Msg: "VOI LUT index out of range",
		}
	}
	i := 0
	for item := range seq.Items() {
		if i == index {
			lut, err := lutFromItem(item.DataSet)
			if err != nil {
				return LUT{}, false, err
			}
			return lut, true, nil
		}
		i++
	}
	return LUT{}, false, nil
}

// ApplyWindowing applies Window Center/Width windowing in ds to in, returning a new
// slice. It mirrors pydicom apply_windowing. Window Center (0028,1050) and Window
// Width (0028,1051) are 1-n DS; index selects the pair. The VOI LUT Function
// (0028,1056) selects LINEAR (default), LINEAR_EXACT, or SIGMOID. yMin/yMax bound the
// output range. With no window present the input is returned unchanged.
func ApplyWindowing(in []float64, ds *DataSet, index int, yMin, yMax float64) ([]float64, error) {
	centers, okC := decimalFloats(ds, TagWindowCenter)
	widths, okW := decimalFloats(ds, TagWindowWidth)
	out := make([]float64, len(in))
	if !okC || !okW || len(centers) == 0 || len(widths) == 0 {
		copy(out, in)
		return out, nil
	}
	if index < 0 || index >= len(centers) || index >= len(widths) {
		return nil, &ValueError{
			Tag: TagWindowCenter, VR: VRDS, Msg: "window index out of range",
		}
	}
	c := centers[index]
	w := widths[index]
	fn := parseVOILUTFunction(stringValue(ds, TagVOILUTFunction))

	if (fn == VOILUTFunctionLinear && w < 1) || (fn != VOILUTFunctionLinear && w <= 0) {
		// PS3.3 §C.11.2.1.2.1 requires Window Width >= 1 for LINEAR; the exact and
		// sigmoid forms divide by w directly, so a non-positive width has no defined
		// output. Reject rather than emit NaN/Inf into a displayed image.
		return nil, &ValueError{
			Tag: TagWindowWidth, VR: VRDS, Msg: "window width below the minimum for the selected VOI LUT function",
		}
	}

	for i, x := range in {
		out[i] = windowValue(x, c, w, fn, yMin, yMax)
	}
	return out, nil
}

// windowValue maps one value through the selected windowing function. The formulae
// are PS3.3 §C.11.2.1.2.1 (LINEAR and LINEAR_EXACT) and §C.11.2.1.3.1 (SIGMOID), with
// the same boundaries pydicom applies in pydicom/pixels/processing.py.
func windowValue(x, c, w float64, fn VOILUTFunction, yMin, yMax float64) float64 {
	yRange := yMax - yMin
	switch fn {
	case VOILUTFunctionLinearExact:
		// PS3.3 §C.11.2.1.2.1: y = ((x - c) / w + 0.5) * (ymax - ymin) + ymin,
		// clamped to ymin for x <= c - w/2 and ymax for x > c + w/2.
		if x <= c-w/2 {
			return yMin
		}
		if x > c+w/2 {
			return yMax
		}
		return ((x-c)/w+0.5)*yRange + yMin
	case VOILUTFunctionSigmoid:
		// PS3.3 §C.11.2.1.3.1: y = (ymax - ymin) / (1 + exp(-4 * (x - c) / w)) + ymin.
		return yRange/(1+math.Exp(-4*(x-c)/w)) + yMin
	default:
		// PS3.3 §C.11.2.1.2.1 LINEAR: centre is shifted by 0.5 and the width by 1,
		// y = ((x - (c - 0.5)) / (w - 1) + 0.5) * (ymax - ymin) + ymin, clamped to
		// ymin for x <= c - 0.5 - (w-1)/2 and ymax for x > c - 0.5 + (w-1)/2. This is
		// the one-LSB offset that distinguishes LINEAR from LINEAR_EXACT.
		cc := c - 0.5
		ww := w - 1
		if x <= cc-ww/2 {
			return yMin
		}
		if x > cc+ww/2 {
			return yMax
		}
		return ((x-cc)/ww+0.5)*yRange + yMin
	}
}

// lutFromItem parses a LUT Descriptor (0028,3002) + LUT Data (0028,3006) pair out of
// a sequence item. The descriptor's first value (entry count) is read as an unsigned
// count where 0 means 2^16 (PS3.3 §C.11.1.1.1); the second value (first mapped value)
// is read honouring the descriptor's signedness so a signed Modality LUT carries a
// negative first-mapped value. LUT Data may be carried as US (decoded integers) or OW
// (raw 16-bit words); both are supported.
func lutFromItem(item *DataSet) (LUT, error) {
	desc, ok := lutDescriptor(item)
	if !ok || len(desc) < 3 {
		return LUT{}, &ValueError{Tag: TagLUTDescriptor, VR: VRUSorSS, Msg: "missing or short LUT Descriptor"}
	}
	// PS3.3 §C.11.1.1.1: the first descriptor value (number of entries) is ALWAYS an
	// unsigned 16-bit count, even when the element VR is SS (only the second value,
	// first-mapped, is signed). Mask to 16 bits so a count above 32767 read through an
	// SS descriptor (sign-extended to negative by decodeInts) is restored, and 0 still
	// means 2^16. Without this a 65536-entry LUT (or any count 32768-65535) yields a
	// negative count and the data[:count] slice below panics.
	count := int(uint16(desc[0])) // #nosec G115 -- intentional 16-bit truncation to read the unsigned entry count
	if count == 0 {
		count = 1 << 16
	}
	firstMapped := int(desc[1])

	data, ok := lutData(item)
	if !ok || len(data) == 0 {
		return LUT{}, &ValueError{Tag: TagLUTData, VR: VRUSorOW, Msg: "missing or empty LUT Data"}
	}
	// PS3.3 §C.11.1.1.1: LUT Data has the number of entries the descriptor declares.
	// A shorter table is malformed; a longer one is tolerated by clamping to count, as
	// some encoders pad. Bound the table by the declared count to keep indexing safe.
	if len(data) > count {
		data = data[:count]
	}
	return LUT{FirstMapped: firstMapped, Data: data}, nil
}

// lutDescriptor reads (0028,3002) LUT Descriptor as three integers. The VR is US or SS
// (VRUSorSS); the second value (first mapped) must honour signedness, so an SS
// descriptor's negative value is preserved. When the dataset was read in Explicit VR the
// value materialises as *Ints already carrying the correct sign (decodeInts sign-extends
// SS). In Implicit VR LE the dictionary VR is the unresolved placeholder VRUSorSS, which
// the value codec materialises as *Ints of unsigned 16-bit words (sign is ambiguous
// without context); the first-mapped value is reinterpreted as signed here so an SS
// descriptor keeps its negative first-mapped value. The entry count (first value) is
// masked unsigned by lutFromItem, so sign-extending it too is harmless.
func lutDescriptor(item *DataSet) ([]int64, bool) {
	e, ok := item.Get(TagLUTDescriptor)
	if !ok {
		return nil, false
	}
	v, ok := materialise(e.Value)
	if !ok {
		return nil, false
	}
	val, ok := v.(*Ints)
	if !ok {
		return nil, false
	}
	out := val.Ints()
	if val.VR() == VRUSorSS {
		for i := range out {
			out[i] = int64(int16(uint16(out[i]))) // #nosec G115 -- 16-bit sign reinterpretation of the unsigned placeholder; the count is masked unsigned by lutFromItem
		}
	}
	return out, true
}

// lutData reads (0028,3006) LUT Data as a slice of int. US data materialises as Ints; OW
// data materialises as raw bytes which are decoded as little-endian 16-bit words (the
// only byte order Part 10 native pixel-related data uses; an explicit-VR-BE file would
// carry US, not OW, for this element in practice). In Implicit VR LE the dictionary VR is
// the unresolved placeholder VRUSorOW, which the value codec materialises as *Bytes
// (lossless raw bytes), decoded here the same way as OW.
func lutData(item *DataSet) ([]int, bool) {
	e, ok := item.Get(TagLUTData)
	if !ok {
		return nil, false
	}
	v, ok := materialise(e.Value)
	if !ok {
		return nil, false
	}
	switch val := v.(type) {
	case *Ints:
		ns := val.Ints()
		out := make([]int, len(ns))
		for i, n := range ns {
			out[i] = int(n)
		}
		return out, true
	case *Bytes:
		return wordsLE(val.Bytes()), true
	default:
		return nil, false
	}
}

// wordsLE decodes a byte field as little-endian unsigned 16-bit words.
func wordsLE(b []byte) []int {
	out := make([]int, len(b)/2)
	for i := range out {
		out[i] = int(binary.LittleEndian.Uint16(b[i*2:]))
	}
	return out
}

// decimalFloat reads the first value of a DS element as a float64. ok is false if the
// element is absent or its first value has no finite float64 form.
func decimalFloat(ds *DataSet, t Tag) (float64, bool) {
	d, ok := ds.GetDecimal(t)
	if !ok {
		return 0, false
	}
	return d.Float64()
}

// decimalFloats reads all values of a 1-n DS element as float64. ok is false if the
// element is absent or carries no decimal values; individual unrepresentable values
// are dropped (pydicom reads these through numpy, which would likewise not round-trip
// a non-finite DS).
func decimalFloats(ds *DataSet, t Tag) ([]float64, bool) {
	e, ok := ds.Get(t)
	if !ok {
		return nil, false
	}
	v, ok := materialise(e.Value)
	if !ok {
		return nil, false
	}
	dv, ok := v.(*Decimals)
	if !ok {
		return nil, false
	}
	decimals := dv.Decimals()
	out := make([]float64, 0, len(decimals))
	for _, d := range decimals {
		if f, ok := d.Float64(); ok {
			out = append(out, f)
		}
	}
	return out, len(out) > 0
}

// stringValue reads the first value of a text element, returning "" when absent.
func stringValue(ds *DataSet, t Tag) string {
	s, _ := ds.GetString(t)
	return s
}
