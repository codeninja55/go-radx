package dicom

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestDecodePlaceholderVRUSorSSAsInts proves the Implicit VR LE reader materialises the
// ambiguous US-or-SS placeholder (PS3.6 dictionary "US or SS") as 16-bit integers rather
// than text, so a value field containing the 0x5C backslash byte is not split into
// fragments. The raw 16-bit words are preserved unsigned (signedness is ambiguous without
// PixelRepresentation); the consumer reinterprets the sign from context.
func TestDecodePlaceholderVRUSorSSAsInts(t *testing.T) {
	// Two 16-bit LE words: 0x5C5C (a backslash-laden word that a text decoder would split)
	// and 0x00FF.
	raw := []byte{0x5C, 0x5C, 0xFF, 0x00}
	br := newBoundedReader(bytes.NewReader(raw), defaultMaxElementLen)
	v, err := decodeValue(br, elementHeader{tag: TagSmallestImagePixelValue, vr: VRUSorSS, length: 4}, encodingFor(ImplicitVRLittleEndian), nil)
	if err != nil {
		t.Fatalf("decodeValue: %v", err)
	}
	iv, ok := v.(*Ints)
	if !ok {
		t.Fatalf("value is %T, want *Ints", v)
	}
	got := iv.Ints()
	want := []int64{0x5C5C, 0x00FF}
	if len(got) != len(want) {
		t.Fatalf("got %d values %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("value[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

// TestDecodePlaceholderWordVRsAsBytes proves the Implicit VR LE reader materialises the
// word/byte placeholders (US or OW, US or SS or OW, OB or OW) as raw bytes, lossless and
// verbatim, so a 0x5C byte in the value field survives uncorrupted.
func TestDecodePlaceholderWordVRsAsBytes(t *testing.T) {
	raw := []byte{0x5C, 0x5C, 0x01, 0x02, 0xFF, 0x00}
	for _, vr := range []VR{VRUSorOW, VRUSorSSorOW, VROBorOW} {
		t.Run(vr.String(), func(t *testing.T) {
			br := newBoundedReader(bytes.NewReader(raw), defaultMaxElementLen)
			v, err := decodeValue(br, elementHeader{tag: TagLUTData, vr: vr, length: uint32(len(raw))}, encodingFor(ImplicitVRLittleEndian), nil)
			if err != nil {
				t.Fatalf("decodeValue: %v", err)
			}
			bv, ok := v.(*Bytes)
			if !ok {
				t.Fatalf("value is %T, want *Bytes", v)
			}
			if !bytes.Equal(bv.Bytes(), raw) {
				t.Errorf("bytes = % x, want % x", bv.Bytes(), raw)
			}
		})
	}
}

// TestPlaceholderVRResolvesToConcreteOnExplicitWrite proves an implicit->explicit
// transcode of a placeholder-VR tag emits a spec-valid concrete VR (US for US-or-SS, OW
// for the word-bearing placeholders), never UN. Reading a value under Implicit VR LE and
// writing it back under Explicit VR LE is the real transcode path.
func TestPlaceholderVRResolvesToConcreteOnExplicitWrite(t *testing.T) {
	cases := []struct {
		name    string
		tag     Tag
		vr      VR
		value   Value
		wantVR  string
		wantRaw []byte
	}{
		{
			name:    "US or SS resolves to US",
			tag:     TagSmallestImagePixelValue,
			vr:      VRUSorSS,
			value:   NewInts(VRUSorSS, 0x5C5C, 0x00FF),
			wantVR:  "US",
			wantRaw: []byte{0x5C, 0x5C, 0xFF, 0x00},
		},
		{
			name:    "OB or OW resolves to OW",
			tag:     NewTag(0x6000, 0x3000), // Overlay Data
			vr:      VROBorOW,
			value:   NewBytes(VROBorOW, []byte{0x5C, 0x5C, 0x01, 0x02}),
			wantVR:  "OW",
			wantRaw: []byte{0x5C, 0x5C, 0x01, 0x02},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ds := NewDataSet()
			ds.Set(Element{Tag: tc.tag, VR: tc.vr, Value: tc.value})

			var buf bytes.Buffer
			if err := EncodeDataSet(&buf, ds, ExplicitVRLittleEndian); err != nil {
				t.Fatalf("EncodeDataSet explicit: %v", err)
			}
			out := buf.Bytes()

			// Element layout: 4-byte tag, then the 2-byte VR field.
			if len(out) < 6 {
				t.Fatalf("encoded element too short: % x", out)
			}
			gotVR := string(out[4:6])
			if gotVR != tc.wantVR {
				t.Fatalf("on-wire VR = %q, want %q (must not be UN)", gotVR, tc.wantVR)
			}

			// Decode it back and confirm the value bytes survive verbatim.
			got, err := DecodeDataSet(bytes.NewReader(out), ExplicitVRLittleEndian)
			if err != nil {
				t.Fatalf("DecodeDataSet explicit: %v", err)
			}
			e, ok := got.Get(tc.tag)
			if !ok {
				t.Fatal("decoded dataset missing the element")
			}
			var raw []byte
			switch v := e.Value.(type) {
			case *Ints:
				raw = make([]byte, len(v.Ints())*2)
				for i, n := range v.Ints() {
					binary.LittleEndian.PutUint16(raw[i*2:], uint16(n)) // #nosec G115 -- test re-pack of 16-bit values
				}
			case *Bytes:
				raw = v.Bytes()
			default:
				t.Fatalf("decoded value is %T, want *Ints or *Bytes", e.Value)
			}
			if !bytes.Equal(raw, tc.wantRaw) {
				t.Errorf("round-tripped value bytes = % x, want % x", raw, tc.wantRaw)
			}
		})
	}
}

// TestPlaceholderUSTagReadableViaIntAccessor proves a 16-bit numeric tag whose dictionary
// VR is the US-or-SS placeholder (e.g. SmallestImagePixelValue) is readable through the
// integer accessor after an Implicit VR LE round-trip, not stranded as text.
func TestPlaceholderUSTagReadableViaIntAccessor(t *testing.T) {
	src := NewDataSet()
	src.Set(Element{Tag: TagSmallestImagePixelValue, VR: VRUS, Value: NewInts(VRUS, 1234)})

	var buf bytes.Buffer
	if err := EncodeDataSet(&buf, src, ImplicitVRLittleEndian); err != nil {
		t.Fatalf("EncodeDataSet implicit: %v", err)
	}
	ds, err := DecodeDataSet(bytes.NewReader(buf.Bytes()), ImplicitVRLittleEndian)
	if err != nil {
		t.Fatalf("DecodeDataSet implicit: %v", err)
	}

	e, ok := ds.Get(TagSmallestImagePixelValue)
	if !ok {
		t.Fatal("decoded dataset missing Smallest Image Pixel Value")
	}
	if e.VR != VRUSorSS {
		t.Fatalf("VR = %v, want the implicit-VR VRUSorSS placeholder", e.VR)
	}
	got, ok := ds.GetInt(TagSmallestImagePixelValue)
	if !ok {
		t.Fatal("GetInt returned ok=false; the placeholder VR must materialise as *Ints")
	}
	if got != 1234 {
		t.Errorf("GetInt = %d, want 1234", got)
	}
}
