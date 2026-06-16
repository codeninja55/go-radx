package dicom

import (
	"encoding/binary"
	"io"
	"math"
	"strings"
)

// decodeValue reads one element's value field into a typed Value, bounds-checking
// the declared length against the bytes remaining before any allocation (Codex
// DCM-004) and surfacing a short read as io.ErrUnexpectedEOF (Codex DCM-003). The
// customisable text VRs are decoded through charset (the dataset's resolved
// (0008,0005)); every other VR is unaffected by it (Codex DCM-011).
func decodeValue(br *boundedReader, h elementHeader, enc encoding, charset *SpecificCharacterSet) (Value, error) {
	raw, err := br.readN(h.length)
	if err != nil {
		return nil, err
	}
	return decodeValueBytes(h.vr, raw, enc, charset)
}

// decodeValueBytes decodes an already-read value field into a typed Value. It is
// shared by the in-line path (decodeValue) and the deferred-load path, so a value
// decodes identically whether it was materialised at read time or on demand.
func decodeValueBytes(vr VR, raw []byte, enc encoding, charset *SpecificCharacterSet) (Value, error) {
	switch vr {
	case VRSS, VRUS, VRSL, VRUL, VRSV, VRUV:
		return decodeInts(vr, raw, enc.byteOrder), nil

	case VRFL, VRFD, VROF, VROD:
		return decodeFloats(vr, raw, enc.byteOrder), nil

	case VRAT:
		return decodeTags(raw, enc.byteOrder), nil

	case VRDS, VRIS:
		return decodeDecimals(vr, raw)

	case VROB, VROW, VROL, VROV, VRUN:
		return NewBytes(vr, raw), nil

	case VROBorOW:
		// Under Implicit VR LE the dictionary yields the ambiguous OB/OW placeholder
		// (PS3.6 marks OverlayData, WaveformData, and similar as "OB or OW"). Both
		// candidates carry raw binary, so the value field is the on-wire bytes
		// verbatim; decoding it as text would corrupt any byte equal to the backslash
		// value delimiter. Keep the placeholder VR so the write path re-emits it.
		return NewBytes(vr, raw), nil

	default:
		// All remaining VRs are text: AE AS CS DA DT LO LT PN SH ST TM UC UI UR UT.
		return decodeStrings(vr, raw, charset)
	}
}

// decodeStrings trims the single trailing pad byte (SPACE or NULL), decodes the field
// through the character set for the customisable text VRs (leaving the default
// repertoire as a verbatim ASCII pass-through), then splits the backslash-separated
// values. A customisable value decoded under a non-default character set retains its
// raw bytes so the write path is byte-exact.
func decodeStrings(vr VR, raw []byte, charset *SpecificCharacterSet) (Value, error) {
	if !vr.usesSpecificCharacterSet() || charset == nil || charset.IsDefaultRepertoire() {
		s := trimPad(vr, raw)
		if s == "" {
			return NewStrings(vr), nil
		}
		return NewStrings(vr, strings.Split(s, `\`)...), nil
	}

	// Decode the whole (unpadded) field through the character set, then split on the
	// backslash value delimiter. Splitting after decoding is correct because the
	// character set's component handling guarantees a backslash only appears as a real
	// delimiter, never as the low byte of a multi-byte character.
	body := raw
	if pad, ok := vr.PadByte(); ok && len(body) > 0 && body[len(body)-1] == pad {
		body = body[:len(body)-1]
	}
	decoded, err := charset.Decode(body)
	if err != nil {
		return nil, err
	}
	if decoded == "" {
		return newStringsRaw(vr, raw), nil
	}
	return newStringsRaw(vr, raw, strings.Split(decoded, `\`)...), nil
}

// decodeDecimals trims the SPACE pad and splits DS/IS values, preserving each
// lexical form.
func decodeDecimals(vr VR, raw []byte) (Value, error) {
	s := trimPad(vr, raw)
	if s == "" {
		return NewDecimals(vr), nil
	}
	parts := strings.Split(s, `\`)
	ds := make([]Decimal, len(parts))
	for i, p := range parts {
		d, err := ParseDecimal(p)
		if err != nil {
			return nil, err
		}
		ds[i] = d
	}
	return NewDecimals(vr, ds...), nil
}

// trimPad removes a single trailing VR pad byte from a value field. PS3.5 §6.2
// treats the trailing pad as insignificant; trimming exactly one keeps the value
// round-trip-stable because the writer re-applies it.
func trimPad(vr VR, raw []byte) string {
	if pad, ok := vr.PadByte(); ok && len(raw) > 0 && raw[len(raw)-1] == pad {
		return string(raw[:len(raw)-1])
	}
	return string(raw)
}

// decodeInts splits the value field into fixed-width integers, sign-extending the
// signed VRs.
func decodeInts(vr VR, raw []byte, bo binary.ByteOrder) Value {
	size := intSize(vr)
	if size == 0 {
		return NewInts(vr)
	}
	n := len(raw) / size
	vals := make([]int64, 0, n)
	for i := 0; i < n; i++ {
		chunk := raw[i*size : i*size+size]
		switch vr {
		case VRSS:
			vals = append(vals, int64(int16(bo.Uint16(chunk)))) // #nosec G115 -- same-width sign reinterpretation per PS3.5 Table 6.2-1
		case VRUS:
			vals = append(vals, int64(bo.Uint16(chunk)))
		case VRSL:
			vals = append(vals, int64(int32(bo.Uint32(chunk)))) // #nosec G115 -- same-width sign reinterpretation per PS3.5 Table 6.2-1
		case VRUL:
			vals = append(vals, int64(bo.Uint32(chunk)))
		case VRSV:
			vals = append(vals, int64(bo.Uint64(chunk))) // #nosec G115 -- same-width sign reinterpretation per PS3.5 Table 6.2-1
		case VRUV:
			vals = append(vals, int64(bo.Uint64(chunk))) // #nosec G115 -- UV occupies the full 64 bits; the bits round-trip through encodeInts's uint64
		}
	}
	return NewInts(vr, vals...)
}

// decodeFloats splits the value field into 32- or 64-bit floats.
func decodeFloats(vr VR, raw []byte, bo binary.ByteOrder) Value {
	size := 4
	if vr == VRFD || vr == VROD {
		size = 8
	}
	n := len(raw) / size
	vals := make([]float64, 0, n)
	for i := 0; i < n; i++ {
		chunk := raw[i*size : i*size+size]
		if size == 4 {
			vals = append(vals, float64(math.Float32frombits(bo.Uint32(chunk))))
		} else {
			vals = append(vals, math.Float64frombits(bo.Uint64(chunk)))
		}
	}
	return NewFloats(vr, vals...)
}

// decodeTags splits the value field into 4-byte (group, element) AT values.
func decodeTags(raw []byte, bo binary.ByteOrder) Value {
	n := len(raw) / 4
	vals := make([]Tag, 0, n)
	for i := 0; i < n; i++ {
		chunk := raw[i*4 : i*4+4]
		vals = append(vals, NewTag(bo.Uint16(chunk[0:2]), bo.Uint16(chunk[2:4])))
	}
	return NewTags(vals...)
}

// maxValueFieldLen is the largest byte count the 32-bit element length field can
// carry; 0xFFFFFFFF is the undefined-length sentinel (PS3.5 §7.1.1). The encode
// helpers reject a larger value with a typed error so a silently truncated length
// can never reach an output stream.
const maxValueFieldLen = int64(0xFFFFFFFE)

// encodeValue writes v's value field in enc's byte order and returns the bytes
// written, which equals v.EncodedLen and is always even. Character VRs are padded
// with the correct trailing byte (Codex DCM-007 write half).
func encodeValue(w io.Writer, v Value, enc encoding) (uint32, error) {
	switch t := v.(type) {
	case *Strings:
		if t.raw != nil {
			// A value read under a non-default character set re-emits its verbatim,
			// already-padded bytes so the round-trip is byte-exact.
			written, err := w.Write(t.raw)
			return uint32(written), err // #nosec G115 -- raw is only set on the read path from a 32-bit length field
		}
		return encodePadded(w, strings.Join(t.Strings(), `\`), t.VR())

	case *Decimals:
		ds := t.Decimals()
		parts := make([]string, len(ds))
		for i, d := range ds {
			parts[i] = d.String()
		}
		return encodePadded(w, strings.Join(parts, `\`), t.VR())

	case *Ints:
		return encodeInts(w, t, enc.byteOrder)

	case *Floats:
		return encodeFloats(w, t, enc.byteOrder)

	case *Tags:
		return encodeTags(w, t, enc.byteOrder)

	case *Bytes:
		return encodeBytes(w, t)

	default:
		// An unknown value type with a zero length writes nothing.
		return 0, nil
	}
}

// encodePadded writes s and, when its length is odd, the VR pad byte, so the
// emitted character value field is even (Codex DCM-007).
func encodePadded(w io.Writer, s string, vr VR) (uint32, error) {
	if int64(len(s)) > maxValueFieldLen {
		return 0, &ValueError{VR: vr, Msg: "value field exceeds the 32-bit element length"}
	}
	n, err := io.WriteString(w, s)
	if err != nil {
		return uint32(n), err // #nosec G115 -- n <= len(s), bounded by the maxValueFieldLen guard
	}
	if len(s)%2 == 1 {
		if pad, ok := vr.PadByte(); ok {
			if _, err := w.Write([]byte{pad}); err != nil {
				return uint32(n), err // #nosec G115 -- n <= len(s), bounded by the maxValueFieldLen guard
			}
			return uint32(n + 1), nil // #nosec G115 -- n <= len(s), bounded by the maxValueFieldLen guard
		}
	}
	return uint32(n), nil // #nosec G115 -- n <= len(s), bounded by the maxValueFieldLen guard
}

func encodeInts(w io.Writer, t *Ints, bo binary.ByteOrder) (uint32, error) {
	vals := t.Ints()
	size := intSize(t.VR())
	if int64(len(vals))*int64(size) > maxValueFieldLen {
		return 0, &ValueError{VR: t.VR(), Msg: "value field exceeds the 32-bit element length"}
	}
	buf := make([]byte, len(vals)*size)
	for i, n := range vals {
		// A caller-supplied value outside the VR's wire width is a typed rejection,
		// never a silent truncation into the output stream.
		if !intFits(t.VR(), n) {
			return 0, &ValueError{VR: t.VR(), Msg: "integer value does not fit the VR wire width"}
		}
		chunk := buf[i*size : i*size+size]
		switch size {
		case 2:
			bo.PutUint16(chunk, uint16(n)) // #nosec G115 -- intFits bounds n to the SS/US range
		case 4:
			bo.PutUint32(chunk, uint32(n)) // #nosec G115 -- intFits bounds n to the SL/UL range
		case 8:
			bo.PutUint64(chunk, uint64(n)) // #nosec G115 -- SV/UV occupy the full 64 bits; same-width reinterpretation
		}
	}
	written, err := w.Write(buf)
	return uint32(written), err // #nosec G115 -- written <= len(buf), bounded by the maxValueFieldLen guard
}

// intFits reports whether n is representable in vr's wire width (PS3.5 Table
// 6.2-1). SV and UV occupy the full 64 bits carried by int64: a UV value above
// MaxInt64 is held as its reinterpreted bit pattern and restored on encode.
func intFits(vr VR, n int64) bool {
	switch vr {
	case VRSS:
		return n >= math.MinInt16 && n <= math.MaxInt16
	case VRUS:
		return n >= 0 && n <= math.MaxUint16
	case VRSL:
		return n >= math.MinInt32 && n <= math.MaxInt32
	case VRUL:
		return n >= 0 && n <= math.MaxUint32
	default:
		return true
	}
}

func encodeFloats(w io.Writer, t *Floats, bo binary.ByteOrder) (uint32, error) {
	vals := t.Floats()
	size := 4
	if t.VR() == VRFD || t.VR() == VROD {
		size = 8
	}
	if int64(len(vals))*int64(size) > maxValueFieldLen {
		return 0, &ValueError{VR: t.VR(), Msg: "value field exceeds the 32-bit element length"}
	}
	buf := make([]byte, len(vals)*size)
	for i, f := range vals {
		chunk := buf[i*size : i*size+size]
		if size == 4 {
			bo.PutUint32(chunk, math.Float32bits(float32(f)))
		} else {
			bo.PutUint64(chunk, math.Float64bits(f))
		}
	}
	written, err := w.Write(buf)
	return uint32(written), err // #nosec G115 -- written <= len(buf), bounded by the maxValueFieldLen guard
}

func encodeTags(w io.Writer, t *Tags, bo binary.ByteOrder) (uint32, error) {
	vals := t.Tags()
	if int64(len(vals))*4 > maxValueFieldLen {
		return 0, &ValueError{VR: VRAT, Msg: "value field exceeds the 32-bit element length"}
	}
	buf := make([]byte, len(vals)*4)
	for i, tag := range vals {
		chunk := buf[i*4 : i*4+4]
		bo.PutUint16(chunk[0:2], tag.Group())
		bo.PutUint16(chunk[2:4], tag.Element())
	}
	written, err := w.Write(buf)
	return uint32(written), err // #nosec G115 -- written <= len(buf), bounded by the maxValueFieldLen guard
}

// encodeBytes writes the raw bytes, padding to even with a trailing NULL.
func encodeBytes(w io.Writer, t *Bytes) (uint32, error) {
	b := t.Bytes()
	if int64(len(b)) > maxValueFieldLen {
		return 0, &ValueError{VR: t.VR(), Msg: "value field exceeds the 32-bit element length"}
	}
	written, err := w.Write(b)
	if err != nil {
		return uint32(written), err // #nosec G115 -- written <= len(b), bounded by the maxValueFieldLen guard
	}
	if len(b)%2 == 1 {
		if _, err := w.Write([]byte{0x00}); err != nil {
			return uint32(written), err // #nosec G115 -- written <= len(b), bounded by the maxValueFieldLen guard
		}
		return uint32(written + 1), nil // #nosec G115 -- written <= len(b), bounded by the maxValueFieldLen guard
	}
	return uint32(written), nil // #nosec G115 -- written <= len(b), bounded by the maxValueFieldLen guard
}
