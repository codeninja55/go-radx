package dicom

import (
	"encoding/binary"
	"fmt"
)

// Waveform is one decoded multiplex group from the Waveform Sequence (5400,0100):
// a [channels][samples] matrix of real-unit values, plus the per-channel scaling that
// produced it. Samples[c][s] is the value of channel c at sample s. This is the
// channel-major transpose of pydicom's (samples, channels) array; the per-channel
// scaling is applied identically (PS3.3 C.10.9).
type Waveform struct {
	Channels int // Number of Waveform Channels (003A,0005)
	Samples  int // Number of Waveform Samples (003A,0010)
	// BitsAllocated is Waveform Bits Allocated (5400,1004): 8 or 16.
	BitsAllocated int
	// SampleInterpretation is Waveform Sample Interpretation (5400,1006): SS, UB, MB,
	// AB, or Uof (US is the legacy spelling, treated as unsigned).
	SampleInterpretation string
	// Samples is [channels][samples] of real-unit values after applying channel
	// sensitivity, sensitivity correction factor, and baseline (PS3.3 C.10.9). When a
	// channel carries no sensitivity, its values are the raw sample magnitudes.
	Data [][]float64
	// ChannelScale records, per channel, the sensitivity * correction multiplier and
	// the baseline offset applied to that channel's raw samples.
	ChannelScale []ChannelScaling
}

// ChannelScaling is the C.10.9 scaling applied to one waveform channel: the corrected
// value is raw*Factor + Baseline. Factor is ChannelSensitivity *
// ChannelSensitivityCorrectionFactor; for a channel with no sensitivity, Factor is 1
// and Baseline is 0 (the raw magnitude is returned unchanged).
type ChannelScaling struct {
	Factor   float64
	Baseline float64
}

// WaveformGroups returns the number of multiplex groups in the Waveform Sequence
// (5400,0100), i.e. the count of valid indices for WaveformArray. It is 0 when the
// dataset carries no waveform sequence.
func (ds *DataSet) WaveformGroups() int {
	seq, ok := ds.GetSequence(TagWaveformSequence)
	if !ok {
		return 0
	}
	return seq.Len()
}

// WaveformArray decodes the multiplexGroupIndex-th item of the Waveform Sequence
// (5400,0100) into a channel-major real-unit matrix. It is the go-radx analogue of
// pydicom's Dataset.waveform_array(index), differing only in axis order (this returns
// [channel][sample]; pydicom returns [sample][channel]).
//
// Sample framing (PS3.5 §8.3, PS3.3 C.10.9): Waveform Data (5400,1010) holds
// interleaved samples — for each sample position, one value per channel in channel
// order. Each value is Waveform Bits Allocated wide (8 or 16 bits). Waveform Sample
// Interpretation (5400,1006) sets the integer type: SS signed 16-bit, US/UB unsigned,
// SB signed 8-bit, MB/AB byte. 16-bit words are read in the dataset's transfer-syntax
// byte order, which the caller supplies as bo (a *Bytes value carries the raw on-wire
// bytes, not host-decoded words).
//
// Scaling (PS3.3 C.10.9 "Waveform sample value representation"): the stored integer
// sample is converted to the channel's real units by
//
//	real = (raw * ChannelSensitivity * ChannelSensitivityCorrectionFactor) + ChannelBaseline
//
// where ChannelSensitivity (003A,0210) is the units-per-LSB for the channel, the
// correction factor (003A,0212) is a calibration multiplier, and ChannelBaseline
// (003A,0213) is the offset in the channel's units that corresponds to a stored sample
// value of zero. A channel with no ChannelSensitivity is returned as raw magnitudes
// (Factor 1, Baseline 0), matching pydicom, which only applies the correction when
// sensitivity is present.
func (ds *DataSet) WaveformArray(multiplexGroupIndex int, bo binary.ByteOrder) (*Waveform, error) {
	seq, ok := ds.GetSequence(TagWaveformSequence)
	if !ok {
		return nil, fmt.Errorf("dicom: dataset has no Waveform Sequence %s", TagWaveformSequence)
	}
	if multiplexGroupIndex < 0 || multiplexGroupIndex >= seq.Len() {
		return nil, fmt.Errorf("dicom: multiplex group index %d out of range [0,%d)", multiplexGroupIndex, seq.Len())
	}

	var item *DataSet
	idx := 0
	for it := range seq.Items() {
		if idx == multiplexGroupIndex {
			item = it.DataSet
			break
		}
		idx++
	}
	if item == nil {
		return nil, fmt.Errorf("dicom: multiplex group %d has no dataset", multiplexGroupIndex)
	}

	channels, ok := item.GetInt(TagNumberOfWaveformChannels)
	if !ok || channels <= 0 {
		return nil, fmt.Errorf("dicom: multiplex group %d is missing a positive Number of Waveform Channels %s", multiplexGroupIndex, TagNumberOfWaveformChannels)
	}
	samples, ok := item.GetInt(TagNumberOfWaveformSamples)
	if !ok || samples <= 0 {
		return nil, fmt.Errorf("dicom: multiplex group %d is missing a positive Number of Waveform Samples %s", multiplexGroupIndex, TagNumberOfWaveformSamples)
	}
	bitsAlloc, ok := item.GetInt(TagWaveformBitsAllocated)
	if !ok || (bitsAlloc != 8 && bitsAlloc != 16) {
		return nil, fmt.Errorf("dicom: multiplex group %d has Waveform Bits Allocated %d, must be 8 or 16", multiplexGroupIndex, bitsAlloc)
	}
	interp, _ := item.GetString(TagWaveformSampleInterpretation)

	dataElem, ok := item.Get(TagWaveformData)
	if !ok {
		return nil, fmt.Errorf("dicom: multiplex group %d is missing Waveform Data %s", multiplexGroupIndex, TagWaveformData)
	}
	raw, ok := binaryValueBytes(dataElem.Value) // OB/OW raw value field
	if !ok {
		return nil, fmt.Errorf("dicom: multiplex group %d Waveform Data is not a binary (OB/OW) value", multiplexGroupIndex)
	}

	sampleBytes := int(bitsAlloc) / 8
	total := int(channels) * int(samples)
	if len(raw) < total*sampleBytes {
		return nil, fmt.Errorf("dicom: multiplex group %d Waveform Data is %d bytes, need %d for %d channels x %d samples x %d-byte samples",
			multiplexGroupIndex, len(raw), total*sampleBytes, channels, samples, sampleBytes)
	}

	scales, err := channelScalings(item, int(channels))
	if err != nil {
		return nil, fmt.Errorf("dicom: multiplex group %d: %w", multiplexGroupIndex, err)
	}

	signed := waveformSigned(interp)
	data := make([][]float64, channels)
	for c := range data {
		data[c] = make([]float64, samples)
	}

	// Samples are channel-interleaved: position p holds one value per channel in order
	// (PS3.5 §8.3). For sample s and channel c, the value sits at flat index
	// s*channels + c.
	for s := range int(samples) {
		for c := range int(channels) {
			flat := s*int(channels) + c
			rawSample := readSample(raw, flat, sampleBytes, signed, bo)
			data[c][s] = float64(rawSample)*scales[c].Factor + scales[c].Baseline
		}
	}

	return &Waveform{
		Channels:             int(channels),
		Samples:              int(samples),
		BitsAllocated:        int(bitsAlloc),
		SampleInterpretation: interp,
		Data:                 data,
		ChannelScale:         scales,
	}, nil
}

// readSample reads the n-th fixed-width integer sample from raw, applying signedness
// and (for 16-bit) the transfer-syntax byte order.
func readSample(raw []byte, n, sampleBytes int, signed bool, bo binary.ByteOrder) int64 {
	off := n * sampleBytes
	if sampleBytes == 1 {
		if signed {
			return int64(int8(raw[off])) // #nosec G115 -- same-width signed reinterpretation
		}
		return int64(raw[off])
	}
	u := bo.Uint16(raw[off : off+2])
	if signed {
		return int64(int16(u)) // #nosec G115 -- same-width signed reinterpretation per PS3.5 Table 6.2-1
	}
	return int64(u)
}

// waveformSigned reports whether the Waveform Sample Interpretation denotes a signed
// integer. PS3.3 C.10.9.1.4 defines SS (signed 16-bit) and SB (signed 8-bit) as
// signed; US/UB (unsigned) and MB/AB (byte-addressed, treated as unsigned magnitudes)
// are unsigned. An absent or unknown term defaults to signed, matching the SS that
// almost all waveform IODs (ECG, hemodynamic) mandate.
func waveformSigned(interp string) bool {
	switch interp {
	case "US", "UB", "MB", "AB":
		return false
	default:
		return true
	}
}

// channelScalings builds the per-channel C.10.9 scaling from the Channel Definition
// Sequence (003A,0200). A channel item with ChannelSensitivity (003A,0210) present has
// Factor = sensitivity * correction (003A,0212, default 1) and Baseline =
// ChannelBaseline (003A,0213, default 0); a channel without sensitivity returns raw
// magnitudes (Factor 1, Baseline 0), as pydicom does.
func channelScalings(item *DataSet, channels int) ([]ChannelScaling, error) {
	scales := make([]ChannelScaling, channels)
	for c := range scales {
		scales[c] = ChannelScaling{Factor: 1}
	}

	seq, ok := item.GetSequence(TagChannelDefinitionSequence)
	if !ok {
		return scales, nil
	}
	if seq.Len() != channels {
		return nil, fmt.Errorf("channel definition sequence has %d items, expected %d channels", seq.Len(), channels)
	}

	c := 0
	for it := range seq.Items() {
		ch := it.DataSet
		sens, hasSens := dsFloat(ch, TagChannelSensitivity)
		if hasSens {
			factor := sens
			if corr, ok := dsFloat(ch, TagChannelSensitivityCorrectionFactor); ok {
				factor *= corr
			}
			baseline := 0.0
			if b, ok := dsFloat(ch, TagChannelBaseline); ok {
				baseline = b
			}
			scales[c] = ChannelScaling{Factor: factor, Baseline: baseline}
		}
		c++
	}
	return scales, nil
}

// dsFloat reads a DS element as a float64. ok is false when the element is absent or
// its lexical form has no finite float64 value.
func dsFloat(ds *DataSet, t Tag) (float64, bool) {
	d, ok := ds.GetDecimal(t)
	if !ok {
		return 0, false
	}
	return d.Float64()
}
