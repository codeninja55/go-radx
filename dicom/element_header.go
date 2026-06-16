package dicom

import (
	"encoding/binary"
	"fmt"
	"io"
)

// encoding captures the on-wire rules a transfer syntax binds: byte order and
// whether VRs are implicit. It is derived once per dataset, never assumed
// (Codex DCM-002).
type encoding struct {
	byteOrder  binary.ByteOrder
	implicitVR bool
}

// encodingFor derives the element encoding from a transfer syntax. Deflate is a
// stream-level concern handled by the reader/writer, not the element codec, so a
// deflated syntax encodes elements exactly like Explicit VR LE.
func encodingFor(ts TransferSyntax) encoding {
	return encoding{byteOrder: ts.byteOrder(), implicitVR: ts.IsImplicitVR()}
}

// elementHeader is the decoded (tag, VR, length) prefix of one data element. A
// length of undefinedLength (0xFFFFFFFF) marks an SQ or encapsulated value whose
// extent is delimited rather than counted.
type elementHeader struct {
	tag    Tag
	vr     VR
	length uint32
}

// readElementHeader reads one element prefix in ts's encoding. A clean io.EOF
// before any byte is consumed signals the end of the element loop; a partial
// header is io.ErrUnexpectedEOF (Codex DCM-003).
func readElementHeader(br *boundedReader, ts TransferSyntax) (elementHeader, error) {
	enc := encodingFor(ts)

	tag, err := br.readTag(enc.byteOrder)
	if err != nil {
		return elementHeader{}, err // io.EOF at a clean boundary, else io.ErrUnexpectedEOF
	}

	// The tag is consumed: we are committed to an element, so any EOF reading the
	// rest of the header is a truncation, never a clean boundary (Codex DCM-003).
	return readElementHeaderBody(br, tag, enc)
}

// readElementHeaderBody reads the VR (if explicit) and length that follow an
// already-consumed tag, in enc's encoding. The tag is committed, so any EOF reading
// the remainder is a truncation, never a clean boundary (Codex DCM-003).
func readElementHeaderBody(br *boundedReader, tag Tag, enc encoding) (elementHeader, error) {
	if enc.implicitVR {
		lenBytes, err := br.readExact(4)
		if err != nil {
			return elementHeader{}, midElementEOF(err)
		}
		return elementHeader{tag: tag, vr: dictVR(tag), length: enc.byteOrder.Uint32(lenBytes)}, nil
	}

	vrBytes, err := br.readExact(2)
	if err != nil {
		return elementHeader{}, midElementEOF(err)
	}
	vr := vrFromBytes(vrBytes)

	if vr.Is32BitLength() {
		// Long form: 2-byte reserved (must be 0x0000) + 4-byte length.
		rest, err := br.readExact(6)
		if err != nil {
			return elementHeader{}, midElementEOF(err)
		}
		return elementHeader{tag: tag, vr: vr, length: enc.byteOrder.Uint32(rest[2:6])}, nil
	}

	lenBytes, err := br.readExact(2)
	if err != nil {
		return elementHeader{}, midElementEOF(err)
	}
	return elementHeader{tag: tag, vr: vr, length: uint32(enc.byteOrder.Uint16(lenBytes))}, nil
}

// midElementEOF maps a clean io.EOF to io.ErrUnexpectedEOF: once a tag is read,
// the element header must complete, so a stream that ends partway through it is
// truncated, not cleanly terminated.
func midElementEOF(err error) error {
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	return err
}

// writeElementHeader writes one element prefix in ts's encoding. signed is the
// dataset's Pixel Representation (0028,0103) being two's-complement (==1): it disambiguates
// the "US or SS" placeholder on an explicit-VR write so a signed value keeps SS semantics
// (see resolveExplicitVR). It is ignored under Implicit VR (no VR on the wire) and for
// every concrete or word/byte VR.
func writeElementHeader(w io.Writer, h elementHeader, ts TransferSyntax, signed bool) error {
	enc := encodingFor(ts)

	var tagBytes [4]byte
	enc.byteOrder.PutUint16(tagBytes[0:2], h.tag.Group())
	enc.byteOrder.PutUint16(tagBytes[2:4], h.tag.Element())
	if _, err := w.Write(tagBytes[:]); err != nil {
		return err
	}

	if enc.implicitVR {
		// Implicit VR carries no VR field on the wire, so an ambiguous dictionary
		// placeholder (US or SS, OB or OW, ...) never needs resolving here.
		var lenBytes [4]byte
		enc.byteOrder.PutUint32(lenBytes[:], h.length)
		_, err := w.Write(lenBytes[:])
		return err
	}

	// Explicit VR must emit a concrete 2-letter VR. A value read under Implicit VR LE
	// keeps the dictionary's ambiguous placeholder; resolve it to a spec-valid VR so an
	// implicit->explicit transcode emits e.g. OW, not UN. Resolve before the length-form
	// check because the resolved VR (OW) uses the 32-bit length form.
	vr := resolveExplicitVR(h.vr, signed)

	if _, err := w.Write(vrToBytes(vr)); err != nil {
		return err
	}

	if vr.Is32BitLength() {
		var buf [6]byte // 2-byte reserved (zero) + 4-byte length
		enc.byteOrder.PutUint32(buf[2:6], h.length)
		_, err := w.Write(buf[:])
		return err
	}

	if h.length > 0xFFFF {
		return &ValueError{Tag: h.tag, VR: h.vr, Msg: fmt.Sprintf("value length %d overflows the 2-byte length form", h.length)}
	}
	var lenBytes [2]byte
	enc.byteOrder.PutUint16(lenBytes[:], uint16(h.length))
	_, err := w.Write(lenBytes[:])
	return err
}

// vrFromBytes maps a 2-byte explicit VR field to the VR. An unrecognised pair is
// treated as UN so a private or future VR reads as raw bytes rather than failing.
func vrFromBytes(b []byte) VR {
	for vr := VRAE; int(vr) < len(vrNames); vr++ {
		name := vrNames[vr]
		if len(name) == 2 && name[0] == b[0] && name[1] == b[1] {
			return vr
		}
	}
	return VRUN
}

// vrToBytes renders a VR as its 2-byte explicit field.
func vrToBytes(vr VR) []byte {
	name := vr.String()
	if len(name) != 2 {
		name = "UN"
	}
	return []byte{name[0], name[1]}
}
