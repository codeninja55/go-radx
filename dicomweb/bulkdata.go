package dicomweb

import (
	"encoding/binary"

	"github.com/codeninja55/go-radx/dicom"
)

// bulkRef is the dicom.Value for a binary attribute decoded from DICOM JSON as a
// BulkDataURI when no resolver was supplied. It carries the reference so the value
// round-trips: a re-marshal re-emits the same BulkDataURI rather than dropping it or
// inventing an empty InlineBinary (PS3.18 Annex F.2.6; the decode contract leaves an
// unresolved BulkDataURI as a reference). It holds no inline bytes.
//
// A bulkRef is a DICOM-JSON-only placeholder: it carries no value field, so a dataset
// still holding one must not be serialised to Part-10 binary (the byte payload is absent
// by definition). Resolve the reference first (supply a WithBulkDataResolver on decode,
// or fetch via WADO-RS bulkdata) before writing such a dataset to the wire.
type bulkRef struct {
	vr  dicom.VR
	uri BulkDataURI
}

// newBulkRef wraps uri under vr as a referenced binary value.
func newBulkRef(vr dicom.VR, uri BulkDataURI) *bulkRef {
	return &bulkRef{vr: vr, uri: uri}
}

func (b *bulkRef) VR() dicom.VR { return b.vr }

// EncodedLen reports a non-zero length so an unresolved reference cannot be written to
// Part-10 binary as a silent, valid zero-length element. The dicom writer encodes a
// foreign value type as zero bytes and then asserts the written length equals
// EncodedLen, so a non-zero report makes that write fail loudly rather than dropping the
// referenced payload (PRD §9.2). Resolve the reference before serialising to the wire.
func (b *bulkRef) EncodedLen(binary.ByteOrder) uint32 { return bulkRefSentinelLen }

// bulkRefSentinelLen is an arbitrary even, non-zero length that guarantees the dicom
// writer's written-length assertion trips for an unresolved reference.
const bulkRefSentinelLen uint32 = 2

// URI returns the referenced BulkDataURI.
func (b *bulkRef) URI() BulkDataURI { return b.uri }

// BulkDataURIs walks ds (and any nested sequence items) and returns the unresolved
// BulkDataURI references it carries, in dataset order. A metadata response decoded without a
// resolver leaves each over-threshold binary value as such a reference; the caller resolves
// each with Client.ResolveBulkDataURI. A dataset with no unresolved references returns nil.
func BulkDataURIs(ds *dicom.DataSet) []BulkDataURI {
	if ds == nil {
		return nil
	}
	var uris []BulkDataURI
	for e := range ds.All() {
		if ref, ok := e.Value.(*bulkRef); ok {
			uris = append(uris, ref.URI())
			continue
		}
		if e.VR == dicom.VRSQ {
			if seq, ok := ds.GetSequence(e.Tag); ok {
				for item := range seq.Items() {
					uris = append(uris, BulkDataURIs(item.DataSet)...)
				}
			}
		}
	}
	return uris
}
