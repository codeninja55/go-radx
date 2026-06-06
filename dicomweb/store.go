package dicomweb

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/codeninja55/go-radx/dicom"
)

// StoreResult is the per-instance outcome a WarnableStoreBackend reports for an accepted
// instance. Warning, when non-zero, is the Warning Reason (0008,1196) recorded against the
// instance in the Referenced SOP Sequence: a store that succeeded with a caveat (for example
// a coerced data element, 0xB000, or a stored instance that already existed, 0xB007) is still
// accepted but carries the reason so a client is never misled into thinking the stored object
// is byte-identical to what it sent (PS3.18 §10.5.1.2). A zero Warning is a clean accept.
type StoreResult struct {
	Warning uint16
}

// WarnableStoreBackend is the optional STOW-RS backend that reports a per-instance Warning
// Reason alongside an accept. A backend that does not implement it stores through the base
// StoreBackend and reports a clean accept (ISP, PRD §8.2): the warning surface is opt-in so a
// store-only deployment that never coerces or de-duplicates implements only StoreBackend.
type WarnableStoreBackend interface {
	StoreWithResult(ctx context.Context, ds *dicom.DataSet) (StoreResult, error)
}

// storeInstance stores ds through the richer WarnableStoreBackend when the backend implements
// it, so an accept can carry a Warning Reason; a base StoreBackend reports a clean accept. The
// returned StoreResult is meaningful only when err is nil.
func (s *Server) storeInstance(ctx context.Context, ds *dicom.DataSet) (StoreResult, error) {
	if wb, ok := s.store.(WarnableStoreBackend); ok {
		return wb.StoreWithResult(ctx, ds)
	}
	return StoreResult{}, s.store.Store(ctx, ds)
}

// storeRetrieveURLBase returns the absolute base URL the STOW-RS store response's Retrieve
// URLs are rooted at. The configured WithStoreRetrieveURLBase wins; absent one, the base is
// derived from the request's scheme and host so the emitted URL points back at the origin the
// client reached. The request's own scheme/host carry no PHI (a UID is appended by the
// per-resource builders, never logged), so deriving them is safe.
func (s *Server) storeRetrieveURLBase(r *http.Request) string {
	if s.retrieveURLBase != "" {
		return s.retrieveURLBase
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded != "" {
		scheme = forwarded
	}
	return scheme + "://" + r.Host
}

// storeResponseBuilder assembles a STOW-RS store-response document (PS3.18 §10.5.3): the
// Referenced SOP Sequence of accepted instances (each with its Retrieve URL and, when the
// store warned, its Warning Reason), the Failed SOP Sequence of rejected instances (each with
// its Failure Reason), the study-level Retrieve URL once any instance was accepted under a
// known study, and a top-level Failure Reason for a failure that belongs to no single instance
// (the "Other failures" path). It carries only SOP identity, registered reason codes, and
// origin-rooted Retrieve URLs, never a patient value (PRD §9.1).
type storeResponseBuilder struct {
	urlBase    string
	referenced []*dicom.DataSet
	failed     []*dicom.DataSet
	studyUID   string
	otherFail  uint16
}

// newStoreResponseBuilder starts a builder rooting its Retrieve URLs at urlBase.
func newStoreResponseBuilder(urlBase string) *storeResponseBuilder {
	return &storeResponseBuilder{urlBase: strings.TrimRight(urlBase, "/")}
}

// accept records an accepted instance, attaching its Retrieve URL and, when warn is non-zero,
// its Warning Reason (0008,1196). The first accepted instance's StudyInstanceUID seeds the
// study-level Retrieve URL the response carries.
func (b *storeResponseBuilder) accept(ds *dicom.DataSet, warn uint16) {
	item := referencedItem(ds)
	if study, ok := ds.GetString(dicom.TagStudyInstanceUID); ok && study != "" {
		if b.studyUID == "" {
			b.studyUID = study
		}
		item.SetString(dicom.TagRetrieveURL, b.instanceRetrieveURL(ds, study))
	}
	if warn != 0 {
		item.Set(dicom.Element{Tag: dicom.TagWarningReason, VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, int64(warn))})
	}
	b.referenced = append(b.referenced, item)
}

// reject records a rejected instance carrying its Failure Reason (0008,1197).
func (b *storeResponseBuilder) reject(ds *dicom.DataSet, reason uint16) {
	b.failed = append(b.failed, failedItem(ds, reason))
}

// otherFailure records a top-level Failure Reason for a fault that belongs to no single
// instance (PS3.18 §10.5.3.2 "Other failures"), for example a body that named a target study
// no part matched. It is emitted as the response dataset's own (0008,1197).
func (b *storeResponseBuilder) otherFailure(reason uint16) {
	b.otherFail = reason
}

// instanceRetrieveURL builds the WADO-RS Retrieve URL for an accepted instance from its
// study/series/SOP identity. A missing series or SOP UID shortens the URL to the deepest known
// level so the reference still resolves to a retrievable resource.
func (b *storeResponseBuilder) instanceRetrieveURL(ds *dicom.DataSet, study string) string {
	url := b.urlBase + "/studies/" + study
	series, _ := ds.GetString(dicom.TagSeriesInstanceUID)
	sop, _ := ds.GetString(dicom.TagSOPInstanceUID)
	if series != "" {
		url += "/series/" + series
		if sop != "" {
			url += "/instances/" + sop
		}
	}
	return url
}

// build renders the accumulated outcome into a store-response dataset. The study-level
// Retrieve URL is set when an accepted instance seeded a study; the sequences and the
// top-level Failure Reason are set only when non-empty, so a clean store carries just the
// Referenced SOP Sequence and the study Retrieve URL.
func (b *storeResponseBuilder) build() *dicom.DataSet {
	resp := dicom.NewDataSet()
	if b.studyUID != "" {
		resp.SetString(dicom.TagRetrieveURL, b.urlBase+"/studies/"+b.studyUID)
	}
	if len(b.referenced) > 0 {
		resp.Set(dicom.Element{
			Tag: dicom.TagReferencedSOPSequence, VR: dicom.VRSQ,
			Value: dicom.NewSequenceValue(dicom.NewSequence(b.referenced...)),
		})
	}
	if len(b.failed) > 0 {
		resp.Set(dicom.Element{
			Tag: dicom.TagFailedSOPSequence, VR: dicom.VRSQ,
			Value: dicom.NewSequenceValue(dicom.NewSequence(b.failed...)),
		})
	}
	if b.otherFail != 0 {
		resp.Set(dicom.Element{Tag: dicom.TagFailureReason, VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, int64(b.otherFail))})
	}
	return resp
}

// counts reports how many instances were accepted and rejected, for the HTTP status decision.
func (b *storeResponseBuilder) counts() (accepted, failed int) {
	return len(b.referenced), len(b.failed)
}

// hasOtherFailure reports whether a top-level Failure Reason (the "Other failures" path) was
// recorded. It feeds the status decision so a store that built no per-instance Failed item yet
// carries a top-level Failure Reason is never reported as a complete success (PS3.18 §10.5.3).
func (b *storeResponseBuilder) hasOtherFailure() bool {
	return b.otherFail != 0
}

// readMetadataBulkParts drains a metadata+bulkdata STOW body, returning the concatenated
// application/dicom+json metadata bytes and a map of bulkdata payloads keyed by every locator
// a metadata BulkDataURI might name: the part's Content-Location and the cid: form of its
// Content-ID. The bounded multipart reader caps the transfer against a hostile origin
// (PRD §9.3). A read fault is returned verbatim for the caller to map to a status.
func readMetadataBulkParts(mr *MultipartReader) (metadata []byte, bulk map[string][]byte, err error) {
	bulk = make(map[string][]byte)
	for {
		ct, part, perr := mr.NextPart()
		if errors.Is(perr, io.EOF) {
			return metadata, bulk, nil
		}
		if perr != nil {
			return nil, nil, perr
		}
		header := mr.PartHeader()
		raw, rerr := io.ReadAll(part)
		if rerr != nil {
			return nil, nil, rerr
		}
		if mediaTypeOf(ct) == mediaTypeDICOMJSON {
			metadata = append(metadata, raw...)
			continue
		}
		for _, key := range bulkPartKeys(header) {
			bulk[key] = raw
		}
	}
}

// bulkPartKeys returns the locators a bulkdata part can be referenced by: its Content-Location
// value and the cid: form of its Content-ID (with the surrounding angle brackets a MIME
// Content-ID carries stripped). A metadata BulkDataURI is matched against these.
func bulkPartKeys(header textproto.MIMEHeader) []string {
	var keys []string
	if loc := strings.TrimSpace(header.Get("Content-Location")); loc != "" {
		keys = append(keys, loc)
	}
	if id := strings.TrimSpace(header.Get("Content-ID")); id != "" {
		id = strings.TrimSuffix(strings.TrimPrefix(id, "<"), ">")
		keys = append(keys, "cid:"+id, id)
	}
	return keys
}

// splitJSONInstances splits a STOW-RS metadata part into one DICOM-JSON document per instance.
// The metadata is a JSON array of instance objects (PS3.18 §10.5.2); a bare object is accepted
// as a single-instance metadata for leniency. Each element is returned as its own object so it
// decodes through UnmarshalJSON, which decodes one object.
func splitJSONInstances(metadata []byte) ([][]byte, error) {
	trimmed := strings.TrimSpace(string(metadata))
	if trimmed == "" {
		return nil, &DecodeError{Msg: "metadata part is empty"}
	}
	if trimmed[0] == '{' {
		return [][]byte{metadata}, nil
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(metadata, &arr); err != nil {
		if isJSONEndOfInput(err) {
			return nil, &TruncatedError{Detail: "metadata array ended mid-document", err: io.ErrUnexpectedEOF}
		}
		return nil, &DecodeError{Msg: "metadata part is not a DICOM-JSON array"}
	}
	docs := make([][]byte, 0, len(arr))
	for _, el := range arr {
		docs = append(docs, el)
	}
	return docs, nil
}
