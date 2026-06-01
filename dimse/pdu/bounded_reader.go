package pdu

import "io"

// boundedReader caps reads at a declared number of remaining bytes so a length
// read from the wire can be validated against bytes actually present before any
// allocation (PRD §9.3). It fails closed: reads past the bound return io.EOF.
type boundedReader struct {
	r         io.Reader
	remaining int64
}

func newBoundedReader(r io.Reader, n int64) *boundedReader {
	return &boundedReader{r: r, remaining: n}
}

func (b *boundedReader) Read(p []byte) (int, error) {
	if b.remaining <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > b.remaining {
		p = p[:b.remaining]
	}
	n, err := b.r.Read(p)
	b.remaining -= int64(n)
	return n, err
}

// Remaining reports the bytes still readable within the bound.
func (b *boundedReader) Remaining() int64 { return b.remaining }

// CanRead reports whether n bytes may be read without exceeding the bound. The
// PDV decoder calls this before make([]byte, n) (Codex DIMSE-004).
func (b *boundedReader) CanRead(n int64) bool { return n >= 0 && n <= b.remaining }
