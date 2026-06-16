package dicom

import (
	"encoding/binary"
	"math"
	"testing"
)

// TestWaveformArray16BitSignedScaled verifies the full PS3.3 C.10.9 path: 2 channels,
// 3 samples, 16-bit signed (SS) samples interleaved per sample position, with each
// channel carrying ChannelSensitivity, ChannelSensitivityCorrectionFactor, and
// ChannelBaseline. The corrected value is raw*sensitivity*correction + baseline.
//
// Channel 0: sensitivity 0.5, correction 2.0, baseline 10  => factor 1.0, +10
// Channel 1: sensitivity 1.0, correction 1.0, baseline -5  => factor 1.0, -5
// Raw samples (sample-major, channel-interleaved): s0[c0=100, c1=-200],
// s1[c0=300, c1=-400], s2[c0=-32768, c1=32767].
func TestWaveformArray16BitSignedScaled(t *testing.T) {
	rawSamples := []int16{100, -200, 300, -400, -32768, 32767} // s0c0,s0c1,s1c0,s1c1,s2c0,s2c1
	data := make([]byte, len(rawSamples)*2)
	for i, v := range rawSamples {
		binary.LittleEndian.PutUint16(data[i*2:], uint16(v)) // #nosec G115 -- test fixture
	}

	ch0 := NewDataSet()
	ch0.Set(Element{Tag: TagChannelSensitivity, VR: VRDS, Value: NewDecimals(VRDS, mustDecimal(t, "0.5"))})
	ch0.Set(Element{Tag: TagChannelSensitivityCorrectionFactor, VR: VRDS, Value: NewDecimals(VRDS, mustDecimal(t, "2.0"))})
	ch0.Set(Element{Tag: TagChannelBaseline, VR: VRDS, Value: NewDecimals(VRDS, mustDecimal(t, "10"))})

	ch1 := NewDataSet()
	ch1.Set(Element{Tag: TagChannelSensitivity, VR: VRDS, Value: NewDecimals(VRDS, mustDecimal(t, "1.0"))})
	ch1.Set(Element{Tag: TagChannelSensitivityCorrectionFactor, VR: VRDS, Value: NewDecimals(VRDS, mustDecimal(t, "1.0"))})
	ch1.Set(Element{Tag: TagChannelBaseline, VR: VRDS, Value: NewDecimals(VRDS, mustDecimal(t, "-5"))})

	mplx := NewDataSet()
	mplx.Set(Element{Tag: TagNumberOfWaveformChannels, VR: VRUS, Value: NewInts(VRUS, 2)})
	mplx.Set(Element{Tag: TagNumberOfWaveformSamples, VR: VRUL, Value: NewInts(VRUL, 3)})
	mplx.Set(Element{Tag: TagWaveformBitsAllocated, VR: VRUS, Value: NewInts(VRUS, 16)})
	mplx.Set(Element{Tag: TagWaveformSampleInterpretation, VR: VRCS, Value: NewStrings(VRCS, "SS")})
	mplx.Set(Element{Tag: TagChannelDefinitionSequence, VR: VRSQ, Value: NewSequenceValue(NewSequence(ch0, ch1))})
	mplx.Set(Element{Tag: TagWaveformData, VR: VROW, Value: NewBytes(VROW, data)})

	ds := NewDataSet()
	ds.Set(Element{Tag: TagWaveformSequence, VR: VRSQ, Value: NewSequenceValue(NewSequence(mplx))})

	if n := ds.WaveformGroups(); n != 1 {
		t.Fatalf("WaveformGroups = %d, want 1", n)
	}

	w, err := ds.WaveformArray(0, binary.LittleEndian)
	if err != nil {
		t.Fatalf("WaveformArray: %v", err)
	}
	if w.Channels != 2 || w.Samples != 3 {
		t.Fatalf("shape = %dx%d, want 2x3", w.Channels, w.Samples)
	}
	if w.BitsAllocated != 16 || w.SampleInterpretation != "SS" {
		t.Fatalf("bits=%d interp=%q, want 16 SS", w.BitsAllocated, w.SampleInterpretation)
	}

	// Channel 0 factor = 0.5*2.0 = 1.0, baseline +10.
	wantC0 := []float64{100 + 10, 300 + 10, -32768 + 10}
	// Channel 1 factor = 1.0*1.0 = 1.0, baseline -5.
	wantC1 := []float64{-200 - 5, -400 - 5, 32767 - 5}
	assertChannel(t, "ch0", w.Data[0], wantC0)
	assertChannel(t, "ch1", w.Data[1], wantC1)
}

// TestWaveformArray8BitUnsignedNoScaling verifies 8-bit unsigned samples with no
// Channel Definition Sequence: values are returned as raw magnitudes (factor 1,
// baseline 0), matching pydicom's behaviour when sensitivity is absent.
func TestWaveformArray8BitUnsignedNoScaling(t *testing.T) {
	// 1 channel, 4 samples, UB. Raw bytes 0, 127, 200, 255.
	data := []byte{0, 127, 200, 255}
	mplx := NewDataSet()
	mplx.Set(Element{Tag: TagNumberOfWaveformChannels, VR: VRUS, Value: NewInts(VRUS, 1)})
	mplx.Set(Element{Tag: TagNumberOfWaveformSamples, VR: VRUL, Value: NewInts(VRUL, 4)})
	mplx.Set(Element{Tag: TagWaveformBitsAllocated, VR: VRUS, Value: NewInts(VRUS, 8)})
	mplx.Set(Element{Tag: TagWaveformSampleInterpretation, VR: VRCS, Value: NewStrings(VRCS, "UB")})
	mplx.Set(Element{Tag: TagWaveformData, VR: VROB, Value: NewBytes(VROB, data)})

	ds := NewDataSet()
	ds.Set(Element{Tag: TagWaveformSequence, VR: VRSQ, Value: NewSequenceValue(NewSequence(mplx))})

	w, err := ds.WaveformArray(0, binary.LittleEndian)
	if err != nil {
		t.Fatalf("WaveformArray: %v", err)
	}
	assertChannel(t, "ub", w.Data[0], []float64{0, 127, 200, 255})
}

// TestWaveformArray8BitSigned verifies SB (signed byte) interpretation reads negative
// magnitudes correctly.
func TestWaveformArray8BitSigned(t *testing.T) {
	data := []byte{0x80, 0x7F, 0xFF} // -128, 127, -1 as int8
	mplx := NewDataSet()
	mplx.Set(Element{Tag: TagNumberOfWaveformChannels, VR: VRUS, Value: NewInts(VRUS, 1)})
	mplx.Set(Element{Tag: TagNumberOfWaveformSamples, VR: VRUL, Value: NewInts(VRUL, 3)})
	mplx.Set(Element{Tag: TagWaveformBitsAllocated, VR: VRUS, Value: NewInts(VRUS, 8)})
	mplx.Set(Element{Tag: TagWaveformSampleInterpretation, VR: VRCS, Value: NewStrings(VRCS, "SB")})
	mplx.Set(Element{Tag: TagWaveformData, VR: VROB, Value: NewBytes(VROB, data)})

	ds := NewDataSet()
	ds.Set(Element{Tag: TagWaveformSequence, VR: VRSQ, Value: NewSequenceValue(NewSequence(mplx))})

	w, err := ds.WaveformArray(0, binary.LittleEndian)
	if err != nil {
		t.Fatalf("WaveformArray: %v", err)
	}
	assertChannel(t, "sb", w.Data[0], []float64{-128, 127, -1})
}

// TestWaveformArrayBigEndian verifies 16-bit samples honour the supplied byte order.
func TestWaveformArrayBigEndian(t *testing.T) {
	// Two SS samples 0x0102 (258) and 0xFFFE (-2) in big-endian byte order.
	data := []byte{0x01, 0x02, 0xFF, 0xFE}
	mplx := NewDataSet()
	mplx.Set(Element{Tag: TagNumberOfWaveformChannels, VR: VRUS, Value: NewInts(VRUS, 1)})
	mplx.Set(Element{Tag: TagNumberOfWaveformSamples, VR: VRUL, Value: NewInts(VRUL, 2)})
	mplx.Set(Element{Tag: TagWaveformBitsAllocated, VR: VRUS, Value: NewInts(VRUS, 16)})
	mplx.Set(Element{Tag: TagWaveformSampleInterpretation, VR: VRCS, Value: NewStrings(VRCS, "SS")})
	mplx.Set(Element{Tag: TagWaveformData, VR: VROW, Value: NewBytes(VROW, data)})

	ds := NewDataSet()
	ds.Set(Element{Tag: TagWaveformSequence, VR: VRSQ, Value: NewSequenceValue(NewSequence(mplx))})

	w, err := ds.WaveformArray(0, binary.BigEndian)
	if err != nil {
		t.Fatalf("WaveformArray: %v", err)
	}
	assertChannel(t, "be", w.Data[0], []float64{258, -2})
}

func TestWaveformArrayErrors(t *testing.T) {
	empty := NewDataSet()
	if _, err := empty.WaveformArray(0, binary.LittleEndian); err == nil {
		t.Error("WaveformArray on dataset without Waveform Sequence accepted, want error")
	}

	mplx := NewDataSet()
	mplx.Set(Element{Tag: TagNumberOfWaveformChannels, VR: VRUS, Value: NewInts(VRUS, 1)})
	mplx.Set(Element{Tag: TagNumberOfWaveformSamples, VR: VRUL, Value: NewInts(VRUL, 4)})
	mplx.Set(Element{Tag: TagWaveformBitsAllocated, VR: VRUS, Value: NewInts(VRUS, 16)})
	mplx.Set(Element{Tag: TagWaveformData, VR: VROW, Value: NewBytes(VROW, []byte{0, 0})}) // only 1 sample's worth
	ds := NewDataSet()
	ds.Set(Element{Tag: TagWaveformSequence, VR: VRSQ, Value: NewSequenceValue(NewSequence(mplx))})

	if _, err := ds.WaveformArray(0, binary.LittleEndian); err == nil {
		t.Error("WaveformArray accepted truncated Waveform Data, want error")
	}
	if _, err := ds.WaveformArray(1, binary.LittleEndian); err == nil {
		t.Error("WaveformArray accepted out-of-range index, want error")
	}
}

func assertChannel(t *testing.T, name string, got, want []float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: len = %d, want %d", name, len(got), len(want))
	}
	for i := range want {
		if math.Abs(got[i]-want[i]) > 1e-9 {
			t.Errorf("%s[%d] = %v, want %v", name, i, got[i], want[i])
		}
	}
}
