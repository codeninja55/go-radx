package dicom

import (
	"bytes"
	"testing"
)

// TestUSorSSResolvesToSSByPixelRepresentation proves an implicit->explicit transcode of a
// "US or SS" element honours Pixel Representation (0028,0103): a signed value (-1000, held
// on the wire as the unsigned word 0xFC18) carried by a dataset with PixelRepresentation==1
// must emit the concrete SS VR on the explicit-VR write, not US. Resolving it to US would
// preserve the bytes but corrupt the signed semantics, so a later explicit-VR read decodes
// 0xFC18 as 64536 rather than -1000 (PS3.5 §8.1.1, PS3.6 dictionary). The dataset is built,
// encoded as Implicit VR LE, read back (yielding the unsigned placeholder), then transcoded
// to Explicit VR LE and read again; the value must survive as -1000 with VR SS.
func TestUSorSSResolvesToSSByPixelRepresentation(t *testing.T) {
	// 0xFC18 is the two's-complement 16-bit encoding of -1000; its unsigned reading is
	// 64536, the form an Implicit VR LE read materialises (signedness is ambiguous there).
	const wireWord = 0xFC18
	const unsigned = 64536 // uint16(0xFC18)
	const signedWant = -1000

	src := NewDataSet()
	src.Set(Element{Tag: TagPixelRepresentation, VR: VRUS, Value: NewInts(VRUS, 1)})
	src.Set(Element{Tag: TagPixelPaddingValue, VR: VRUSorSS, Value: NewInts(VRUSorSS, wireWord)})

	// Encode Implicit VR LE, then read back: the placeholder VR re-materialises and the
	// value is the unsigned word 64536, exactly as a real implicit read produces.
	var implicit bytes.Buffer
	if err := EncodeDataSet(&implicit, src, ImplicitVRLittleEndian); err != nil {
		t.Fatalf("EncodeDataSet implicit: %v", err)
	}
	mid, err := DecodeDataSet(bytes.NewReader(implicit.Bytes()), ImplicitVRLittleEndian)
	if err != nil {
		t.Fatalf("DecodeDataSet implicit: %v", err)
	}
	if e, ok := mid.Get(TagPixelPaddingValue); !ok || e.VR != VRUSorSS {
		t.Fatalf("intermediate VR = %v ok=%v, want VRUSorSS placeholder", e.VR, ok)
	}
	if got, _ := mid.GetInt(TagPixelPaddingValue); got != unsigned {
		t.Fatalf("intermediate value = %d, want %d (unsigned word)", got, unsigned)
	}

	// Transcode: write the placeholder-bearing dataset as Explicit VR LE.
	var explicit bytes.Buffer
	if err := EncodeDataSet(&explicit, mid, ExplicitVRLittleEndian); err != nil {
		t.Fatalf("EncodeDataSet explicit: %v", err)
	}

	// The on-wire VR for Pixel Padding Value must be SS, not US. Walk the explicit stream
	// to the element and read its VR field.
	gotVR := explicitVRForTag(t, explicit.Bytes(), TagPixelPaddingValue)
	if gotVR != "SS" {
		t.Fatalf("on-wire VR = %q, want %q (PixelRepresentation==1 is signed)", gotVR, "SS")
	}

	// Read the explicit stream back: with a concrete SS VR the value sign-reinterprets to
	// -1000, the value that survived the transcode.
	out, err := DecodeDataSet(bytes.NewReader(explicit.Bytes()), ExplicitVRLittleEndian)
	if err != nil {
		t.Fatalf("DecodeDataSet explicit: %v", err)
	}
	got, ok := out.GetInt(TagPixelPaddingValue)
	if !ok {
		t.Fatal("GetInt(PixelPaddingValue) ok=false after explicit round-trip")
	}
	if got != signedWant {
		t.Errorf("transcoded value = %d, want %d (NOT %d)", got, signedWant, unsigned)
	}
}

// TestUSorSSResolvesToUSWhenUnsigned proves the PixelRepresentation==0 case still resolves
// "US or SS" to US, the unsigned default, so an unsigned dataset is unaffected by the
// signed-aware resolution.
func TestUSorSSResolvesToUSWhenUnsigned(t *testing.T) {
	src := NewDataSet()
	src.Set(Element{Tag: TagPixelRepresentation, VR: VRUS, Value: NewInts(VRUS, 0)})
	src.Set(Element{Tag: TagPixelPaddingValue, VR: VRUSorSS, Value: NewInts(VRUSorSS, 64536)})

	var explicit bytes.Buffer
	if err := EncodeDataSet(&explicit, src, ExplicitVRLittleEndian); err != nil {
		t.Fatalf("EncodeDataSet explicit: %v", err)
	}
	if gotVR := explicitVRForTag(t, explicit.Bytes(), TagPixelPaddingValue); gotVR != "US" {
		t.Fatalf("on-wire VR = %q, want %q (PixelRepresentation==0 is unsigned)", gotVR, "US")
	}

	out, err := DecodeDataSet(bytes.NewReader(explicit.Bytes()), ExplicitVRLittleEndian)
	if err != nil {
		t.Fatalf("DecodeDataSet explicit: %v", err)
	}
	if got, _ := out.GetInt(TagPixelPaddingValue); got != 64536 {
		t.Errorf("unsigned value = %d, want 64536", got)
	}
}

// TestUSorSSAbsentPixelRepresentationDefaultsToUS proves that when Pixel Representation is
// absent the resolution defaults to US (the standard's unsigned default), documenting the
// fallback behaviour.
func TestUSorSSAbsentPixelRepresentationDefaultsToUS(t *testing.T) {
	src := NewDataSet()
	src.Set(Element{Tag: TagPixelPaddingValue, VR: VRUSorSS, Value: NewInts(VRUSorSS, 64536)})

	var explicit bytes.Buffer
	if err := EncodeDataSet(&explicit, src, ExplicitVRLittleEndian); err != nil {
		t.Fatalf("EncodeDataSet explicit: %v", err)
	}
	if gotVR := explicitVRForTag(t, explicit.Bytes(), TagPixelPaddingValue); gotVR != "US" {
		t.Fatalf("on-wire VR = %q, want %q (absent PixelRepresentation defaults to unsigned)", gotVR, "US")
	}
}

// explicitVRForTag scans an Explicit VR LE element stream for tag and returns its 2-letter
// VR field. It assumes short-form (no SQ/long-form elements precede the target in the test
// datasets), which holds for the small datasets here.
func explicitVRForTag(t *testing.T, data []byte, tag Tag) string {
	t.Helper()
	i := 0
	for i+8 <= len(data) {
		group := uint16(data[i]) | uint16(data[i+1])<<8
		elem := uint16(data[i+2]) | uint16(data[i+3])<<8
		vr := string(data[i+4 : i+6])
		length := int(uint16(data[i+6]) | uint16(data[i+7])<<8)
		if NewTag(group, elem) == tag {
			return vr
		}
		i += 8 + length
	}
	t.Fatalf("tag %v not found in explicit stream", tag)
	return ""
}
