package server

import (
	"context"
	"iter"
	"net/http"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
)

// ObjectStore persists DICOM objects as opaque, addressable blobs keyed by SOP Instance UID. It is
// the storage primitive behind the DIMSE C-STORE/C-GET/C-MOVE handlers and the DICOMweb
// STOW-RS/WADO-RS backends, so one implementation serves both protocol planes. Implementations must
// be safe for concurrent use (PRD §9.4); all methods honour ctx cancellation and never log PHI
// (PRD §9.1).
type ObjectStore interface {
	// Put persists ds. It is idempotent on SOP Instance UID: storing the same instance twice is not
	// an error, and the implementation decides whether to overwrite or de-duplicate. Returning
	// success without durably persisting is a defect (PRD §9.2 honest-failure rule).
	Put(ctx context.Context, ds *dicom.DataSet) error

	// Get retrieves one stored object by SOP Instance UID. It returns ErrNotFound
	// (errors.Is-comparable) when absent.
	Get(ctx context.Context, instance dicom.SOPInstanceUID) (*dicom.DataSet, error)

	// Exists reports presence without materialising the object, so a STOW-RS de-duplication check is
	// cheap.
	Exists(ctx context.Context, instance dicom.SOPInstanceUID) (bool, error)

	// Delete removes one object. It returns ErrNotFound when the instance is absent so a caller can
	// distinguish a no-op delete from a successful one (never a silent success on missing data).
	Delete(ctx context.Context, instance dicom.SOPInstanceUID) error
}

// Catalogue indexes stored objects for query. The query vocabulary is DICOM tags, matching DIMSE
// C-FIND and QIDO-RS, so a handler translates neither — it forwards the tag-keyed query. The
// catalogue holds PHI, so it is the component the PHI-safety defaults most directly govern (see the
// PHI posture in servers.md). Implementations must be safe for concurrent use (PRD §9.4).
type Catalogue interface {
	// Index records or updates the queryable attributes of ds. It extracts only the attributes the
	// supported query models need; it does not store pixel data.
	Index(ctx context.Context, ds *dicom.DataSet) error

	// Query answers a hierarchical query at the given level (PATIENT/STUDY/SERIES/IMAGE). It streams
	// results as a Go 1.23+ iterator so a large match set is not buffered (the iterator convention
	// used across go-radx, PRD §8.1). Each yielded DataSet carries the requested return keys; the
	// terminal iteration carries a nil error when clean and a typed error on a backend fault, never a
	// laundered empty success (PRD §9.2).
	Query(ctx context.Context, q CatalogueQuery) iter.Seq2[*dicom.DataSet, error]

	// Remove drops the index entry for one instance (paired with ObjectStore.Delete). It returns
	// ErrNotFound when the instance is absent.
	Remove(ctx context.Context, instance dicom.SOPInstanceUID) error
}

// CatalogueQuery is a normalised query shared by the C-FIND and QIDO-RS handlers, so both planes hit
// one index API. The matching rules (single-value, list-of-UID, wildcard, range, universal per
// DICOM PS3.4 Annex C) are applied by the Catalogue; the contract is that the iterator yields
// exactly the matching objects and never silently over- or under-matches.
type CatalogueQuery struct {
	Level  dimse.QueryLevel     // PATIENT, STUDY, SERIES, or IMAGE
	Match  map[dicom.Tag]string // tag -> match value (single-value, range, wildcard, or UID-list matching)
	Return []dicom.Tag          // attributes to return beyond the level's defaults
	Limit  int                  // 0 means no limit (QIDO-RS limit=); DIMSE C-FIND ignores this
	Offset int                  // QIDO-RS offset=; DIMSE C-FIND ignores this
	Fuzzy  bool                 // QIDO-RS fuzzymatching=true
}

// WorklistSource answers Modality Worklist queries. The query is a DIMSE C-FIND identifier (tag-keyed
// match keys such as ScheduledProcedureStepStartDate, Modality, and the ScheduledProcedureStepSequence),
// and each yielded DataSet is one matching Scheduled Procedure Step worklist item. Because a worklist
// is not the stored-object catalogue (it is scheduled procedure steps, typically fed from an HL7
// ORM/OMG order or a FHIR ServiceRequest), it has its own backend (workflow step 2, PRD §5.1).
type WorklistSource interface {
	// Find yields one worklist item per match. Matching follows the MWL information model (PS3.4
	// Annex K). The iterator terminates with a nil error on clean completion; a backend failure
	// terminates it with a typed error that the handler maps to a DIMSE failure status (never a
	// laundered success, PRD §9.2).
	Find(ctx context.Context, query *dicom.DataSet) iter.Seq2[*dicom.DataSet, error]
}

// Authenticator verifies a request's credentials and returns the authenticated Principal, or an
// error. It is consulted by the HTTP middleware (bearer token, mutual-TLS client certificate) and by
// the DIMSE association handler (DICOM user-identity). Credentials come from the request only;
// secrets are never logged (PRD §9.8).
type Authenticator interface {
	// AuthenticateHTTP inspects an HTTP request's Authorization header and/or verified client
	// certificate.
	AuthenticateHTTP(ctx context.Context, r *http.Request) (Principal, error)

	// AuthenticateDIMSE inspects a DICOM user-identity negotiation item presented during A-ASSOCIATE.
	AuthenticateDIMSE(ctx context.Context, calling dimse.AETitle) (Principal, error)
}

// Principal is the authenticated identity. It carries no secret material — only the identity and any
// coarse scopes a deployment chooses to attach — so it is safe to put in a request context and (its
// ID, not its tokens) in a span.
type Principal struct {
	ID     string   // stable subject identifier (username, certificate subject, token subject)
	Scopes []string // optional, deployment-defined authorisation scopes
}

// allowAll is the explicit no-authentication Authenticator the loopback reference daemons use. It is
// a named type so a deployment cannot enable "no auth" by accident: choosing AllowAll is a visible,
// reviewable decision (see the bind policy in servers.md).
type allowAll struct{}

// AllowAll returns the explicit no-authentication Authenticator used by the loopback reference
// daemons. It authenticates every request as an anonymous Principal. A non-loopback bind that still
// uses AllowAll is a configuration the operator owns, made conspicuous by the bind-default policy.
func AllowAll() Authenticator { return allowAll{} }

// AuthenticateHTTP accepts every HTTP request as the anonymous principal.
func (allowAll) AuthenticateHTTP(_ context.Context, _ *http.Request) (Principal, error) {
	return Principal{ID: "anonymous"}, nil
}

// AuthenticateDIMSE accepts every association, attributing it to its Calling AE Title (a protocol
// identifier, not PHI).
func (allowAll) AuthenticateDIMSE(_ context.Context, calling dimse.AETitle) (Principal, error) {
	return Principal{ID: string(calling)}, nil
}
