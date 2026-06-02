package dicomweb

import (
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
)

// Default hostile-input caps for multipart/related bodies (PRD §9.3). Exceeding either
// returns a *LimitExceededError before the offending data is read into memory.
const (
	defaultMaxParts     = 10000     // maximum number of parts in one body
	defaultMaxPartBytes = 256 << 20 // 256 MiB maximum per-part size
)

// MultipartWriter assembles a multipart/related body with a single root media type,
// e.g. "application/dicom" for a STOW-RS request. It wraps mime/multipart.Writer and
// fixes the Content-Type of every part to a member of the related set.
type MultipartWriter struct {
	w        *multipart.Writer
	rootType string
}

// NewMultipartWriter returns a writer that emits a multipart/related body to w. rootType
// is the media type of the related content (the "type" parameter of the outer
// Content-Type), e.g. "application/dicom".
func NewMultipartWriter(w io.Writer, rootType string) *MultipartWriter {
	return &MultipartWriter{w: multipart.NewWriter(w), rootType: rootType}
}

// AddPart writes one body part with the given Content-Type, copying body to the output.
// It returns any error from constructing the part header or copying the body.
func (mw *MultipartWriter) AddPart(contentType string, body io.Reader) error {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", contentType)
	part, err := mw.w.CreatePart(header)
	if err != nil {
		return fmt.Errorf("dicomweb: create multipart part: %w", err)
	}
	if _, err := io.Copy(part, body); err != nil {
		return fmt.Errorf("dicomweb: write multipart part body: %w", err)
	}
	return nil
}

// Close finalises the body, writing the closing boundary, and returns the boundary
// string for the caller to embed in the outer Content-Type header.
func (mw *MultipartWriter) Close() (boundary string, err error) {
	boundary = mw.w.Boundary()
	if err := mw.w.Close(); err != nil {
		return "", fmt.Errorf("dicomweb: close multipart writer: %w", err)
	}
	return boundary, nil
}

// ContentType returns the full multipart/related media type for the outer Content-Type
// header, carrying the root media type as the "type" parameter and the writer's boundary,
// e.g. multipart/related; type="application/dicom"; boundary=abc123. Call it after the
// parts are written (the boundary is fixed at construction).
func (mw *MultipartWriter) ContentType() string {
	return fmt.Sprintf("multipart/related; type=%q; boundary=%q", mw.rootType, mw.w.Boundary())
}

// MultipartReader iterates the parts of a multipart/related body from a bounded reader.
// MaxParts caps the number of parts and MaxPartBytes caps each part's size; both default
// to the package caps and may be tightened per instance. Exceeding either returns a
// *LimitExceededError; a body that ends mid-part returns a *TruncatedError wrapping
// io.ErrUnexpectedEOF (PRD §9.2, §9.3).
type MultipartReader struct {
	MaxParts     int
	MaxPartBytes int64

	mr      *multipart.Reader
	count   int
	current *boundedPartReader // the part most recently returned, not yet superseded
}

// NewMultipartReader parses the multipart/related media type for its boundary and
// returns a reader over r. It rejects a non-multipart media type and a multipart media
// type missing its boundary parameter.
func NewMultipartReader(r io.Reader, mediaType string) (*MultipartReader, error) {
	mt, params, err := mime.ParseMediaType(mediaType)
	if err != nil {
		return nil, fmt.Errorf("dicomweb: parse multipart media type: %w", err)
	}
	if mt != "multipart/related" {
		return nil, fmt.Errorf("%w: media type %q is not multipart/related", ErrUnsupported, mt)
	}
	boundary, ok := params["boundary"]
	if !ok || boundary == "" {
		return nil, fmt.Errorf("dicomweb: multipart media type %q has no boundary", mt)
	}
	return &MultipartReader{
		MaxParts:     defaultMaxParts,
		MaxPartBytes: defaultMaxPartBytes,
		mr:           multipart.NewReader(r, boundary),
	}, nil
}

// NextPart advances to the next body part, returning its Content-Type and a reader over
// its body bounded by MaxPartBytes. It returns io.EOF after the final part. The
// part-count cap trips on the part beyond the cap, so a body with exactly MaxParts parts
// is accepted while the (MaxParts+1)th part fails before its body is read into memory.
//
// Before advancing, any unread remainder of the previously returned part is drained
// through its bounded reader so the per-part size cap holds even when the caller skipped
// the body (mime/multipart would otherwise drain it uncapped on the next read).
//
// Parts are read with NextRawPart so a Content-Transfer-Encoding of quoted-printable is
// not transparently decoded: DICOMweb parts carry raw binary DICOM/frame octets and must
// reach the caller byte-for-byte unmodified.
func (mr *MultipartReader) NextPart() (contentType string, body io.Reader, err error) {
	if mr.current != nil {
		if err := mr.current.drain(); err != nil {
			return "", nil, err
		}
		mr.current = nil
	}

	part, err := mr.mr.NextRawPart()
	if err != nil {
		if err == io.EOF {
			return "", nil, io.EOF
		}
		if err == io.ErrUnexpectedEOF {
			return "", nil, &TruncatedError{Detail: "multipart body ended before a part boundary", err: io.ErrUnexpectedEOF}
		}
		// The mime/multipart parser error can embed the offending raw line via %q, which
		// could carry PHI; return a structural message and do not wrap the raw error
		// (PRD §9.1). The Go standard library exposes no typed kind to distinguish here.
		return "", nil, &MalformedPartError{Detail: "malformed multipart framing"}
	}
	mr.count++
	if mr.count > mr.MaxParts {
		// Reject before exposing the body, so the offending part is never read in.
		return "", nil, &LimitExceededError{
			Limit:  uint64(mr.MaxParts),
			Actual: uint64(mr.count),
			Kind:   "multipart-part-count",
		}
	}

	mr.current = &boundedPartReader{part: part, remaining: mr.MaxPartBytes, limit: mr.MaxPartBytes}
	ct := part.Header.Get("Content-Type")
	return ct, mr.current, nil
}

// boundedPartReader caps a single part's body at limit bytes and converts the
// multipart layer's mid-part EOF into a typed truncation error. The cap is enforced as
// bytes are consumed, before the whole part is read into memory (PRD §9.3).
type boundedPartReader struct {
	part      *multipart.Part
	remaining int64
	limit     int64
}

func (b *boundedPartReader) Read(p []byte) (int, error) {
	if b.remaining <= 0 {
		// Peek one byte: if the part still has content, it exceeds the cap.
		var probe [1]byte
		n, err := b.part.Read(probe[:])
		if n > 0 {
			return 0, &LimitExceededError{
				Limit:  uint64(b.limit),
				Actual: uint64(b.limit) + 1,
				Kind:   "multipart-part-bytes",
			}
		}
		if err == io.EOF {
			return 0, io.EOF
		}
		if err == io.ErrUnexpectedEOF {
			return 0, &TruncatedError{Detail: "multipart part ended before its closing boundary", err: io.ErrUnexpectedEOF}
		}
		return 0, err
	}

	if int64(len(p)) > b.remaining {
		p = p[:b.remaining]
	}
	n, err := b.part.Read(p)
	b.remaining -= int64(n)
	if err == io.ErrUnexpectedEOF {
		return n, &TruncatedError{Detail: "multipart part ended before its closing boundary", err: io.ErrUnexpectedEOF}
	}
	return n, err
}

// drain consumes any unread remainder of the part through the same bounded path so a
// caller that skipped the body still trips the per-part cap on an oversized part. A
// clean EOF means the part fit; a LimitExceededError means it overran the cap.
func (b *boundedPartReader) drain() error {
	var scratch [4096]byte
	for {
		_, err := b.Read(scratch[:])
		if err == nil {
			continue
		}
		if err == io.EOF {
			return nil
		}
		return err
	}
}
