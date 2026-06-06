package dicomweb

import (
	"fmt"
	"strings"

	"github.com/codeninja55/go-radx/dicom"
)

// StoreResponse models the STOW-RS store response document (PS3.18 §10.5), parsed from
// the application/dicom+json body the origin returns. It distinguishes per-instance
// success from failure so a caller never has to guess whether a partial store occurred.
type StoreResponse struct {
	// RetrieveURL is the response dataset's Retrieve URL (0008,1190), if present: a URL
	// for the study the instances were stored under.
	RetrieveURL string
	// Referenced lists the instances the origin accepted (Referenced SOP Sequence,
	// 0008,1199).
	Referenced []StoredInstance
	// Failed lists the instances the origin rejected, each with its Failure Reason
	// (Failed SOP Sequence, 0008,1198).
	Failed []dicom.FailedSOPInstance
	// OtherFailure is the response's top-level Failure Reason (0008,1197), set when the
	// origin reported a failure that belongs to no single instance (PS3.18 §10.5.3.2 "Other
	// failures"). Zero when the response named no top-level failure.
	OtherFailure uint16
}

// StoredInstance is an accepted STOW-RS instance: the canonical referenced SOP Class /
// SOP Instance pair plus the per-instance Retrieve URL (0008,1190) the origin assigns and an
// optional Warning Reason (0008,1196). A non-zero WarningReason means the instance was stored
// with a caveat (for example a coerced data element or a duplicate that already existed), so a
// caller is never misled into thinking the stored object is byte-identical to what it sent.
// Embedding the shared dicom type keeps the SOP UID vocabulary identical to dimse and dicom
// without re-declaring it.
type StoredInstance struct {
	dicom.ReferencedSOPInstance
	RetrieveURL   string
	WarningReason uint16
}

// IsComplete reports whether every posted instance was accepted: true only when no instance
// failed and the response named no top-level Other failure.
func (r *StoreResponse) IsComplete() bool {
	return r != nil && len(r.Failed) == 0 && r.OtherFailure == 0
}

// parseStoreResponse decodes a STOW-RS store-response dataset (already unmarshalled from
// application/dicom+json) into a StoreResponse. The store-response document is small and
// flat (a top-level Retrieve URL plus two sequences), so it is read with the dicom
// dataset accessors rather than a bespoke JSON shape.
func parseStoreResponse(ds *dicom.DataSet) *StoreResponse {
	resp := &StoreResponse{}
	if ds == nil {
		return resp
	}
	if url, ok := ds.GetString(dicom.TagRetrieveURL); ok {
		resp.RetrieveURL = url
	}
	if reason, ok := ds.GetInt(dicom.TagFailureReason); ok {
		resp.OtherFailure = uint16(reason)
	}
	if seq, ok := ds.GetSequence(dicom.TagReferencedSOPSequence); ok {
		for item := range seq.Items() {
			class, _ := item.DataSet.GetString(dicom.TagReferencedSOPClassUID)
			instance, _ := item.DataSet.GetString(dicom.TagReferencedSOPInstanceUID)
			url, _ := item.DataSet.GetString(dicom.TagRetrieveURL)
			warn, _ := item.DataSet.GetInt(dicom.TagWarningReason)
			resp.Referenced = append(resp.Referenced, StoredInstance{
				ReferencedSOPInstance: dicom.ReferencedSOPInstance{
					SOPClassUID:    dicom.SOPClassUID(class),
					SOPInstanceUID: dicom.SOPInstanceUID(instance),
				},
				RetrieveURL:   url,
				WarningReason: uint16(warn),
			})
		}
	}
	if seq, ok := ds.GetSequence(dicom.TagFailedSOPSequence); ok {
		for item := range seq.Items() {
			class, _ := item.DataSet.GetString(dicom.TagReferencedSOPClassUID)
			instance, _ := item.DataSet.GetString(dicom.TagReferencedSOPInstanceUID)
			reason, _ := item.DataSet.GetInt(dicom.TagFailureReason)
			resp.Failed = append(resp.Failed, dicom.FailedSOPInstance{
				ReferencedSOPInstance: dicom.ReferencedSOPInstance{
					SOPClassUID:    dicom.SOPClassUID(class),
					SOPInstanceUID: dicom.SOPInstanceUID(instance),
				},
				FailureReason: uint16(reason),
			})
		}
	}
	return resp
}

// StoreError reports a STOW-RS transfer in which one or more instances were rejected. It
// is the fail-closed signal (PRD §9.2): Store returns it alongside the parsed
// StoreResponse so neither the partial success nor the failure is silently dropped. It
// names each failed instance's Failure Reason by its registered name, never any patient
// value (PRD §9.1).
type StoreError struct {
	// Failed is the list of rejected instances, copied from the response. It may be empty
	// when the origin signalled failure through the HTTP status alone (a sparse store
	// response); Status then records that status.
	Failed []dicom.FailedSOPInstance
	// Accepted is the count of instances the origin did accept.
	Accepted int
	// Status is the HTTP status the origin returned (202 partial, 409 none accepted).
	Status int
}

func (e *StoreError) Error() string {
	if len(e.Failed) == 0 {
		// The origin reported failure via the status with no per-instance detail.
		return fmt.Sprintf("dicomweb: STOW-RS store reported failure (HTTP %d, accepted %d) with no Failed SOP Sequence",
			e.Status, e.Accepted)
	}
	reasons := make([]string, 0, len(e.Failed))
	for _, f := range e.Failed {
		reasons = append(reasons, failureReasonName(f.FailureReason))
	}
	return fmt.Sprintf("dicomweb: STOW-RS store failed for %d of %d instances (accepted %d): %s",
		len(e.Failed), len(e.Failed)+e.Accepted, e.Accepted, strings.Join(reasons, ", "))
}

// failureReasonName renders a STOW-RS / C-STORE Failure Reason code (0008,1197) by its
// registered name (PS3.18 §10.5.1.2, drawing on the PS3.4 C-STORE refused statuses), or
// a hex form for an unregistered code. It carries no patient value.
func failureReasonName(code uint16) string {
	if name, ok := failureReasonNames[code]; ok {
		return fmt.Sprintf("%s (0x%04X)", name, code)
	}
	return fmt.Sprintf("unknown failure reason (0x%04X)", code)
}

// failureReasonNames maps the STOW-RS Failure Reason codes (PS3.18 §10.5.1.2-3) to their
// registered meanings. The values reuse the C-STORE refused status range (PS3.4 GG.4.2)
// plus the STOW-specific 0xA7xx/0xC3xx codes.
var failureReasonNames = map[uint16]string{
	0x0110: "processing failure",
	0x0122: "referenced SOP Class not supported",
	0x0119: "class-instance conflict",
	0x0242: "SOP Instance access denied",
	0xA700: "out of resources",
	0xA730: "intended recipient not supported",
	0xA800: "data set does not match SOP Class",
	0xA900: "data set does not match SOP Class",
	0xC000: "cannot understand",
	0xC120: "referenced SOP Instance not in study",
	0xC122: "transfer syntax not supported",
}
