package dicom

import (
	"encoding/binary"
	"io"
	"math"
	"strings"
)

// rawSQ holds an SQ element's value field as opaque bytes for Increment 2. The
// reader must never drop a sequence (Codex DCM-005); structured Sequence/Item
// parsing arrives in Increment 3, which replaces this with a real *Sequence. Until
// then the raw bytes are preserved exactly so a sequence-bearing file round-trips
// byte-identically.
//
// undefined marks the encoding of the source element: an undefined-length SQ's raw
// bytes include the Sequence Delimitation Item, and it must be re-emitted with the
// 0xFFFFFFFF length form. A defined-length SQ's raw bytes are exactly its counted
// value field, re-emitted with that byte count.
type rawSQ struct {
	raw       []byte
	undefined bool
}

func (v *rawSQ) VR() VR { return VRSQ }

// EncodedLen is the preserved byte count; SQ is never padded. For an
// undefined-length SQ the on-wire length field is the 0xFFFFFFFF sentinel, written
// by the writer rather than counted here, so EncodedLen still reports the byte
// count for callers that size the value field.
func (v *rawSQ) EncodedLen(binary.ByteOrder) uint32 { return uint32(len(v.raw)) }

// decodeValue reads one element's value field into a typed Value, bounds-checking
// the declared length against the bytes remaining before any allocation (Codex
// DCM-004) and surfacing a short read as io.ErrUnexpectedEOF (Codex DCM-003).
func decodeValue(br *boundedReader, h elementHeader, enc encoding) (Value, error) {
	raw, err := br.readN(h.length)
	if err != nil {
		return nil, err
	}

	switch h.vr {
	case VRSQ:
		return &rawSQ{raw: raw}, nil

	case VRSS, VRUS, VRSL, VRUL, VRSV, VRUV:
		return decodeInts(h.vr, raw, enc.byteOrder), nil

	case VRFL, VRFD, VROF, VROD:
		return decodeFloats(h.vr, raw, enc.byteOrder), nil

	case VRAT:
		return decodeTags(raw, enc.byteOrder), nil

	case VRDS, VRIS:
		return decodeDecimals(h.vr, raw)

	case VROB, VROW, VROL, VROV, VRUN:
		return NewBytes(h.vr, raw), nil

	default:
		// All remaining VRs are text: AE AS CS DA DT LO LT PN SH ST TM UC UI UR UT.
		return decodeStrings(h.vr, raw), nil
	}
}

// decodeStrings trims the single trailing pad byte (SPACE or NULL) and splits the
// backslash-separated values. Charset decoding is Increment 4; for now the raw
// bytes are taken as the lexical string.
func decodeStrings(vr VR, raw []byte) Value {
	s := trimPad(vr, raw)
	if s == "" {
		return NewStrings(vr)
	}
	return NewStrings(vr, strings.Split(s, `\`)...)
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
			vals = append(vals, int64(int16(bo.Uint16(chunk))))
		case VRUS:
			vals = append(vals, int64(bo.Uint16(chunk)))
		case VRSL:
			vals = append(vals, int64(int32(bo.Uint32(chunk))))
		case VRUL:
			vals = append(vals, int64(bo.Uint32(chunk)))
		case VRSV:
			vals = append(vals, int64(bo.Uint64(chunk)))
		case VRUV:
			vals = append(vals, int64(bo.Uint64(chunk)))
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

// encodeValue writes v's value field in enc's byte order and returns the bytes
// written, which equals v.EncodedLen and is always even. Character VRs are padded
// with the correct trailing byte (Codex DCM-007 write half).
func encodeValue(w io.Writer, v Value, enc encoding) (uint32, error) {
	switch t := v.(type) {
	case *rawSQ:
		_, err := w.Write(t.raw)
		return uint32(len(t.raw)), err

	case *Strings:
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
	n, err := io.WriteString(w, s)
	if err != nil {
		return uint32(n), err
	}
	if len(s)%2 == 1 {
		if pad, ok := vr.PadByte(); ok {
			if _, err := w.Write([]byte{pad}); err != nil {
				return uint32(n), err
			}
			return uint32(n + 1), nil
		}
	}
	return uint32(n), nil
}

func encodeInts(w io.Writer, t *Ints, bo binary.ByteOrder) (uint32, error) {
	vals := t.Ints()
	size := intSize(t.VR())
	buf := make([]byte, len(vals)*size)
	for i, n := range vals {
		chunk := buf[i*size : i*size+size]
		switch size {
		case 2:
			bo.PutUint16(chunk, uint16(n))
		case 4:
			bo.PutUint32(chunk, uint32(n))
		case 8:
			bo.PutUint64(chunk, uint64(n))
		}
	}
	written, err := w.Write(buf)
	return uint32(written), err
}

func encodeFloats(w io.Writer, t *Floats, bo binary.ByteOrder) (uint32, error) {
	vals := t.Floats()
	size := 4
	if t.VR() == VRFD || t.VR() == VROD {
		size = 8
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
	return uint32(written), err
}

func encodeTags(w io.Writer, t *Tags, bo binary.ByteOrder) (uint32, error) {
	vals := t.Tags()
	buf := make([]byte, len(vals)*4)
	for i, tag := range vals {
		chunk := buf[i*4 : i*4+4]
		bo.PutUint16(chunk[0:2], tag.Group())
		bo.PutUint16(chunk[2:4], tag.Element())
	}
	written, err := w.Write(buf)
	return uint32(written), err
}

// encodeBytes writes the raw bytes, padding to even with a trailing NULL.
func encodeBytes(w io.Writer, t *Bytes) (uint32, error) {
	b := t.Bytes()
	written, err := w.Write(b)
	if err != nil {
		return uint32(written), err
	}
	if len(b)%2 == 1 {
		if _, err := w.Write([]byte{0x00}); err != nil {
			return uint32(written), err
		}
		return uint32(written + 1), nil
	}
	return uint32(written), nil
}
