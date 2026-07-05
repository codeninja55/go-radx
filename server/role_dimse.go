package server

import (
	"context"
	"fmt"
	"iter"
	"net"
	"strings"
	"sync"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicomweb"
	"github.com/codeninja55/go-radx/dimse"
)

// defaultDIMSEPort is the SCP listen port when WithDIMSEPort is not set. 11112 is the IANA-registered
// DICOM port and the pynetdicom default.
const defaultDIMSEPort = 11112

// dimseRoleConfig holds the resolved DIMSERole options.
type dimseRoleConfig struct {
	port     int
	contexts []dimse.PresentationContext
	worklist WorklistSource
	maxAssoc int
	// moveDests is the known-AE table the C-MOVE SCP resolves a Move Destination AE Title against
	// (the dcmqrscp model: a static AE-title -> host:port map). Nil refuses every move as 0xA801.
	moveDests map[dimse.AETitle]string
	// retrieve mounts the C-GET/C-MOVE SCP capability. It is off by default so an existing
	// NewDIMSERole embedder does not silently gain archive-wide retrieve on upgrade; the retrieve
	// services are an explicit opt-in (WithDIMSERetrieve), and a role without it refuses C-GET/C-MOVE
	// with StatusSOPClassNotSupported exactly as before.
	retrieve bool
}

// DIMSERoleOption configures a DIMSERole at construction.
type DIMSERoleOption func(*dimseRoleConfig)

// WithDIMSEPort sets the SCP listen port (default 11112).
func WithDIMSEPort(port int) DIMSERoleOption {
	return func(c *dimseRoleConfig) { c.port = port }
}

// WithDIMSEContexts sets the supported presentation contexts the SCP negotiates as acceptor (default
// Storage + Query/Retrieve + Verification).
func WithDIMSEContexts(c []dimse.PresentationContext) DIMSERoleOption {
	return func(cfg *dimseRoleConfig) {
		if len(c) > 0 {
			cfg.contexts = c
		}
	}
}

// WithWorklistSource mounts the Modality Worklist SCP alongside the storage and query/retrieve
// services, fed by w (workflow step 2, PRD §5.1).
func WithWorklistSource(w WorklistSource) DIMSERoleOption {
	return func(c *dimseRoleConfig) { c.worklist = w }
}

// WithMaxAssociations bounds the concurrently served inbound associations; capacity is acquired
// before a handler goroutine is spawned (DIMSE-013).
func WithMaxAssociations(n int) DIMSERoleOption {
	return func(c *dimseRoleConfig) { c.maxAssoc = n }
}

// WithDIMSEMoveDestinations configures the known-AE table the role's C-MOVE SCP resolves a Move
// Destination AE Title against (dcmqrscp's static AE table): each entry maps a destination AE
// Title to its network address ("host:port"). A destination absent from the table is answered
// with the terminal 0xA801 "Move Destination Unknown" status (PS3.4 C.4.2.1.5); with no table,
// every C-MOVE is refused that way. The underlying dimse.WithMoveDestinations copies the map, so
// a later caller mutation cannot change the running server's resolution.
func WithDIMSEMoveDestinations(dests map[dimse.AETitle]string) DIMSERoleOption {
	return func(c *dimseRoleConfig) { c.moveDests = dests }
}

// WithDIMSERetrieve mounts the C-GET and C-MOVE SCP capability over the ObjectStore and Catalogue.
// It is off by default: the retrieve services stream stored composite instances to a requestor (or
// a Move Destination AE), so enabling them is an explicit, reviewable decision an embedder makes
// rather than a behaviour that appears on upgrade. Without it the role serves only
// C-ECHO/C-STORE/C-FIND and refuses C-GET/C-MOVE with StatusSOPClassNotSupported. C-MOVE also needs
// a destination table (WithDIMSEMoveDestinations) to resolve where matched instances are sent.
func WithDIMSERetrieve() DIMSERoleOption {
	return func(c *dimseRoleConfig) { c.retrieve = true }
}

// DIMSERole configures the DIMSE SCP. The title is a validated dimse.AETitle (produce it with
// dimse.ParseAETitle, never a bare string); supported contexts are required; the worklist source is
// optional and, when supplied, mounts the Modality Worklist SCP alongside the
// C-STORE/C-FIND/C-GET/C-MOVE/C-ECHO services. It wraps a dimse.Server, storing received objects via
// the ObjectStore and indexing them via the Catalogue, so a study stored over C-STORE is immediately
// queryable over C-FIND and QIDO-RS.
type DIMSERole struct {
	cfg     dimseRoleConfig
	title   dimse.AETitle
	store   ObjectStore
	cat     Catalogue
	handler *dimseHandler

	srv *dimse.Server

	mu    sync.Mutex
	bound net.Addr
}

// NewDIMSERole builds the DIMSE SCP role over the shared backends. A nil store or catalogue is a
// configuration error (the storage and query planes are both required for a C-STORE that indexes).
func NewDIMSERole(title dimse.AETitle, store ObjectStore, cat Catalogue, opts ...DIMSERoleOption) (*DIMSERole, error) {
	if _, err := dimse.ParseAETitle(string(title)); err != nil {
		return nil, fmt.Errorf("server: dimse role AE title: %w", err)
	}
	if store == nil {
		return nil, fmt.Errorf("server: dimse role requires a non-nil ObjectStore")
	}
	if cat == nil {
		return nil, fmt.Errorf("server: dimse role requires a non-nil Catalogue")
	}
	cfg := dimseRoleConfig{
		port:     defaultDIMSEPort,
		contexts: dimse.QueryRetrieveWithStorageContexts(),
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	if len(cfg.contexts) == 0 {
		cfg.contexts = dimse.QueryRetrieveWithStorageContexts()
	}
	// The Verification context is mounted alongside Storage and Query/Retrieve so a C-ECHO probe a
	// peer sends before storing is answered; the merged context list renumbers IDs uniquely.
	cfg.contexts = mergeContexts(cfg.contexts, dimse.VerificationContexts())
	return &DIMSERole{
		cfg:   cfg,
		title: title,
		store: store,
		cat:   cat,
	}, nil
}

func (r *DIMSERole) name() string { return "dimse" }

// start binds the SCP listener and serves in the background. The AE applies the daemon's shared TLS
// config, and the handler logs through the daemon's logger.
func (r *DIMSERole) start(ctx context.Context, host string, env roleEnv) error {
	var aeOpts []dimse.AEOption
	if env.tlsConfig != nil {
		aeOpts = append(aeOpts, dimse.WithTLS(env.tlsConfig))
	}
	ae, err := dimse.NewAE(r.title, aeOpts...)
	if err != nil {
		return err
	}

	r.handler = &dimseHandler{
		store:    r.store,
		cat:      r.cat,
		worklist: r.cfg.worklist,
		logger:   env.logger,
		audit:    env.audit,
	}
	// The retrieve capability is a distinct handler type carrying Get/Move, mounted only when the
	// role opts in (WithDIMSERetrieve). Passing the bare *dimseHandler leaves C-GET/C-MOVE unmounted,
	// so the dispatcher refuses them with StatusSOPClassNotSupported (interface segregation, PRD §8.2).
	var srvHandler any = r.handler
	if r.cfg.retrieve {
		srvHandler = &dimseRetrieveHandler{dimseHandler: r.handler}
	}

	var srvOpts []dimse.ServerOption
	if r.cfg.maxAssoc > 0 {
		srvOpts = append(srvOpts, dimse.WithMaxAssociations(r.cfg.maxAssoc))
	}
	if r.cfg.retrieve {
		if len(r.cfg.moveDests) > 0 {
			srvOpts = append(srvOpts, dimse.WithMoveDestinations(r.cfg.moveDests))
		}
		// Grant the requestor the Storage SCP role for the role's CONFIGURED Storage classes so a
		// C-GET's same-association sub-operation C-STOREs can be received (PS3.7 D.3.3.4). Deriving
		// the grant from the configured contexts (not the fixed preset) means a custom Storage class
		// added via WithDIMSEContexts is deliverable over C-GET, not silently undeliverable.
		srvOpts = append(srvOpts, dimse.WithGetStorageRoles(r.storageContextClasses()...))
	}
	// Enforce the daemon's Authenticator at the association-accept layer: an unauthorized Calling AE
	// Title is rejected with an A-ASSOCIATE-RJ before any C-ECHO/C-STORE/C-FIND runs, so a
	// non-loopback bind actually runs the authenticator the bind policy required (rather than the
	// handler holding an unused auth reference). On a loopback bind the daemon defaults to AllowAll,
	// whose AuthenticateDIMSE admits every association, so loopback behaviour is unchanged.
	if env.auth != nil {
		auth := env.auth
		srvOpts = append(srvOpts, dimse.WithAssociationAuthorizer(func(callingAE string, _ net.Addr) error {
			_, err := auth.AuthenticateDIMSE(ctx, dimse.AETitle(callingAE))
			return err
		}))
	}
	r.srv = dimse.NewServer(ae, r.cfg.contexts, srvHandler, srvOpts...)

	addr := joinHostPort(host, r.cfg.port)
	served := make(chan error, 1)
	go func() { served <- r.srv.ListenAndServe(ctx, addr) }()

	// Block until the listener is bound (Addr non-nil) or ListenAndServe returned an early bind error,
	// so the daemon's fail-closed startup can abort on a bind failure before declaring success.
	return waitBound(served, func() net.Addr { return r.srv.Addr() }, r.setBound)
}

// setBound records the bound listen address under the role lock so addr() (called concurrently by
// Daemon.Addrs) reads a consistent value.
func (r *DIMSERole) setBound(a net.Addr) {
	r.mu.Lock()
	r.bound = a
	r.mu.Unlock()
}

func (r *DIMSERole) addr() net.Addr {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.bound
}

// shutdown drains the SCP: it closes the listener (stop accepting), cancels in-flight handler
// contexts, and closes active association connections, all bounded by ctx (the DIMSE-014 ordering).
func (r *DIMSERole) shutdown(ctx context.Context) error {
	if r.srv == nil {
		return nil
	}
	return r.srv.Shutdown(ctx)
}

// dimseHandler adapts the shared backends to the dimse.Handler capabilities. It implements
// EchoHandler, StoreHandler, and FindHandler (Patient/Study Root from the Catalogue, plus the
// Modality Worklist model when a WorklistSource is configured). The retrieve capabilities
// (GetHandler/MoveHandler) live on dimseRetrieveHandler, mounted only when the role opts in, so an
// embedder does not gain archive-wide retrieve implicitly. The handler logs structural identifiers
// only (AE titles, SOP class/instance UIDs), never PHI (PRD §9.1).
type dimseHandler struct {
	store    ObjectStore
	cat      Catalogue
	worklist WorklistSource
	logger   *zap.Logger
	audit    AuditFunc
}

// Echo answers a C-ECHO with success unless the SCP is degraded.
func (h *dimseHandler) Echo(_ context.Context, info dimse.OpInfo) dimse.Status {
	h.logger.Info("c-echo received",
		zap.Stringer("calling_ae", info.CallingAETitle),
		zap.Stringer("called_ae", info.CalledAETitle))
	return dimse.StatusEchoSuccess
}

// Store persists one received dataset to the ObjectStore and indexes it in the Catalogue. It returns
// success only after both succeed; a store or index failure is mapped to a Storage failure status,
// never laundered into success (PRD §9.2). The dataset must carry its SOP Instance UID — the dispatch
// layer has already validated that the command and dataset agree on it.
func (h *dimseHandler) Store(ctx context.Context, ds *dicom.DataSet, info dimse.OpInfo) dimse.Status {
	h.logger.Info("c-store received",
		zap.String("sop_class", string(info.SOPClassUID)),
		zap.Stringer("calling_ae", info.CallingAETitle),
		zap.Uint16("message_id", info.MessageID))

	if err := h.store.Put(ctx, ds); err != nil {
		h.logger.Warn("c-store object store failed", zap.String("sop_class", string(info.SOPClassUID)))
		return dimse.StatusStoreCannotUnderstand
	}
	if err := h.cat.Index(ctx, ds); err != nil {
		// The object is durably stored but un-indexed; report the warning rather than a clean success
		// so the peer is not told the catalogue is consistent when it is not (PRD §9.2 fail-closed).
		// The durable store IS the modification, so the audit event still fires, carrying the
		// un-indexed outcome — a stored object must never be an unaudited one.
		h.logger.Warn("c-store catalogue index failed", zap.String("sop_class", string(info.SOPClassUID)))
		if h.audit != nil {
			h.audit(dimseStoreAuditEvent(AuditOutcomeStoredUnindexed, ds, string(info.SOPClassUID)))
		}
		return dimse.StatusStoreElementDiscarded
	}
	if h.audit != nil {
		h.audit(dimseStoreAuditEvent(AuditOutcomeStoredIndexed, ds, string(info.SOPClassUID)))
	}
	return dimse.StatusStoreSuccess
}

// Find answers a C-FIND. With a WorklistSource configured it serves the Modality Worklist information
// model from that source; otherwise it serves the Patient/Study Root models from the Catalogue. Each
// match is yielded as a Pending status carrying the matching identifier; the iterator terminates with
// a Success status when matching completes cleanly and a Failure status on a backend fault (never a
// laundered success, PRD §9.2).
func (h *dimseHandler) Find(ctx context.Context, query *dicom.DataSet, level dimse.QueryLevel, info dimse.OpInfo) iter.Seq2[dimse.Status, *dicom.DataSet] {
	if h.worklist != nil && isWorklistQuery(query) {
		return h.findWorklist(ctx, query)
	}
	return h.findCatalogue(ctx, query, level)
}

// findWorklist adapts the WorklistSource iterator to the C-FIND response contract: a Pending
// (0xFF00) per item, a terminal Worklist Success when the source ends cleanly, and a Worklist failure
// status on a source error.
func (h *dimseHandler) findWorklist(ctx context.Context, query *dicom.DataSet) iter.Seq2[dimse.Status, *dicom.DataSet] {
	return func(yield func(dimse.Status, *dicom.DataSet) bool) {
		for item, err := range h.worklist.Find(ctx, query) {
			if err != nil {
				h.logger.Warn("mwl source failed")
				yield(dimse.NewStatus(0xC000, dimse.ServiceClassWorklist), nil)
				return
			}
			if !yield(dimse.StatusWorklistPending, item) {
				return
			}
		}
		yield(dimse.StatusWorklistSuccess, nil)
	}
}

// findCatalogue adapts the Catalogue iterator to the C-FIND response contract for the
// Patient/Study Root models: a Pending (0xFF00) per match, a terminal Find Success on clean
// completion, and a Find failure status on a backend fault.
func (h *dimseHandler) findCatalogue(ctx context.Context, query *dicom.DataSet, level dimse.QueryLevel) iter.Seq2[dimse.Status, *dicom.DataSet] {
	return func(yield func(dimse.Status, *dicom.DataSet) bool) {
		match := matchKeysFromIdentifier(query)
		for result, err := range queryCatalogue(ctx, h.cat, h.store, match, level, nil, false, false) {
			if err != nil {
				h.logger.Warn("catalogue query failed")
				yield(dimse.NewStatus(0xC000, dimse.ServiceClassFind), nil)
				return
			}
			if !yield(dimse.StatusFindPending, result) {
				return
			}
		}
		yield(dimse.StatusFindSuccess, nil)
	}
}

// dimseRetrieveHandler is the opt-in retrieve capability: it embeds the base *dimseHandler (so it
// carries Echo/Store/Find) and adds GetHandler and MoveHandler over the ObjectStore. It is passed
// to the dimse.Server only when the role opts in (WithDIMSERetrieve), so a role without it does not
// implement the retrieve interfaces and the dispatcher refuses C-GET/C-MOVE.
type dimseRetrieveHandler struct {
	*dimseHandler
}

// Get answers a C-GET: it streams each stored instance the identifier matches as a Pending yield,
// and the dimse runtime C-STOREs each one back to the requestor on the same association.
func (h *dimseRetrieveHandler) Get(ctx context.Context, query *dicom.DataSet, level dimse.QueryLevel, info dimse.OpInfo) iter.Seq2[dimse.Status, *dicom.DataSet] {
	h.logger.Info("c-get received",
		zap.Stringer("calling_ae", info.CallingAETitle),
		zap.Stringer("level", level),
		zap.Uint16("message_id", info.MessageID))
	return h.retrieveInstances(ctx, query, level, dimse.ServiceClassGet)
}

// Move answers a C-MOVE: it streams each stored instance the identifier matches as a Pending
// yield, and the dimse runtime C-STOREs each one to the resolved Move Destination AE over its own
// outbound association. Destination resolution (and the 0xA801 refusal for an unknown AE Title)
// happens in the runtime against the configured known-AE table before this handler runs.
func (h *dimseRetrieveHandler) Move(ctx context.Context, query *dicom.DataSet, level dimse.QueryLevel, dest dimse.AETitle, info dimse.OpInfo) iter.Seq2[dimse.Status, *dicom.DataSet] {
	h.logger.Info("c-move received",
		zap.Stringer("calling_ae", info.CallingAETitle),
		zap.Stringer("destination", dest),
		zap.Stringer("level", level),
		zap.Uint16("message_id", info.MessageID))
	return h.retrieveInstances(ctx, query, level, dimse.ServiceClassMove)
}

// retrieveInstances resolves a retrieve identifier to full stored instances. Before querying, it
// enforces the retrieve unique-key requirement (PS3.4 C.2.2.2/C.4.3): the level's identifying UID
// must be present and valued, so an absent or universal key fails closed with 0xA900 rather than
// streaming the entire archive (the PHI-safety guard). It then queries the Catalogue at instance
// granularity with the identifier's match keys, honouring UID-list values, and — mirroring the
// C-FIND seam — re-applies any identifier key the catalogue cannot index against the fetched
// composite dataset (dicomweb.MatchDataSet) so an unindexed key still constrains the retrieve
// rather than over-disclosing. Each surviving instance is yielded under the service's Pending
// status. A backend fault terminates with the service's 0xC000 failure status, never a laundered
// clean end (PRD §9.2); the diagnostic names the fault class only, no PHI.
func (h *dimseHandler) retrieveInstances(ctx context.Context, query *dicom.DataSet, level dimse.QueryLevel, svc dimse.ServiceClass) iter.Seq2[dimse.Status, *dicom.DataSet] {
	pending := dimse.NewStatus(0xFF00, svc)
	failure := dimse.NewStatus(0xC000, svc)
	identifierMismatch := dimse.NewStatus(0xA900, svc)
	return func(yield func(dimse.Status, *dicom.DataSet) bool) {
		match := matchKeysFromIdentifier(query)
		if !hasValuedUniqueKey(match, level) {
			h.logger.Warn("retrieve refused: missing valued unique key for level",
				zap.Stringer("level", level))
			yield(identifierMismatch, nil)
			return
		}
		unindexed := unindexedKeys(match)
		for candidate, err := range h.cat.Query(ctx, CatalogueQuery{Level: dimse.QueryLevelImage, Match: match}) {
			if err != nil {
				h.logger.Warn("retrieve catalogue query failed")
				yield(failure, nil)
				return
			}
			instance, ok := candidate.GetString(dicom.TagSOPInstanceUID)
			if !ok || instance == "" {
				continue
			}
			full, err := h.store.Get(ctx, dicom.SOPInstanceUID(instance))
			if err != nil {
				h.logger.Warn("retrieve object store fetch failed")
				yield(failure, nil)
				return
			}
			// A key the catalogue could not index was not applied by the SQL query; decide it against
			// the real stored values so an unindexed constraint is honoured, never dropped.
			if !dicomweb.MatchDataSet(full, unindexed, false) {
				continue
			}
			if !yield(pending, full) {
				return
			}
		}
	}
}

// retrieveUniqueKeys lists the identifying UID tags a retrieve at level must carry with a value
// (PS3.4 C.2.2.2/C.4.3): a study is identified by its Study UID, a series additionally by its
// Series UID, an instance by its SOP Instance UID, and a patient by Patient ID. A retrieve missing
// its level's unique key is under-specified and must not run an archive-wide query.
func retrieveUniqueKeys(level dimse.QueryLevel) []dicom.Tag {
	switch level {
	case dimse.QueryLevelPatient:
		return []dicom.Tag{dicom.TagPatientID}
	case dimse.QueryLevelStudy:
		return []dicom.Tag{dicom.TagStudyInstanceUID}
	case dimse.QueryLevelSeries:
		return []dicom.Tag{dicom.TagStudyInstanceUID, dicom.TagSeriesInstanceUID}
	default: // IMAGE / FRAME
		return []dicom.Tag{dicom.TagStudyInstanceUID, dicom.TagSeriesInstanceUID, dicom.TagSOPInstanceUID}
	}
}

// hasValuedUniqueKey reports whether match carries every unique key the level requires with a
// non-empty, non-universal value. A universal ("" or "*") value or an absent key means the
// retrieve is under-specified and must be refused with 0xA900.
func hasValuedUniqueKey(match map[dicom.Tag]string, level dimse.QueryLevel) bool {
	for _, tag := range retrieveUniqueKeys(level) {
		v, ok := match[tag]
		if !ok || v == "" || v == "*" {
			return false
		}
	}
	return true
}

// storageContextClasses returns the Storage SOP Classes among the role's configured presentation
// contexts, the classes the C-GET SCP grants the requestor the Storage SCP role for. It excludes
// the Verification and Query/Retrieve information-model abstract syntaxes (which are contexts but
// not Storage classes), so the grant tracks whatever Storage set WithDIMSEContexts configured
// rather than a fixed preset.
func (r *DIMSERole) storageContextClasses() []dicom.SOPClassUID {
	nonStorage := make(map[dicom.SOPClassUID]struct{})
	for _, preset := range [][]dimse.PresentationContext{
		dimse.VerificationContexts(),
		dimse.QueryRetrieveContexts(),
		dimse.ExtendedQueryRetrieveContexts(),
		dimse.BasicWorklistContexts(),
	} {
		for _, pc := range preset {
			nonStorage[pc.AbstractSyntax] = struct{}{}
		}
	}
	seen := make(map[dicom.SOPClassUID]struct{}, len(r.cfg.contexts))
	out := make([]dicom.SOPClassUID, 0, len(r.cfg.contexts))
	for _, pc := range r.cfg.contexts {
		if _, skip := nonStorage[pc.AbstractSyntax]; skip {
			continue
		}
		if _, dup := seen[pc.AbstractSyntax]; dup {
			continue
		}
		seen[pc.AbstractSyntax] = struct{}{}
		out = append(out, pc.AbstractSyntax)
	}
	return out
}

// isWorklistQuery reports whether the C-FIND identifier carries a Scheduled Procedure Step Sequence
// (0040,0100), the hallmark of a Modality Worklist query. A query without it is a Patient/Study Root
// query the catalogue answers.
func isWorklistQuery(query *dicom.DataSet) bool {
	if query == nil {
		return false
	}
	_, ok := query.Get(dicom.TagScheduledProcedureStepSequence)
	return ok
}

// matchKeysFromIdentifier extracts the string match keys from a C-FIND/retrieve identifier into
// the catalogue's tag->value map. Multi-valued keys are reconstituted to their backslash-delimited
// wire form (GetStrings + join), so a UID-list match (StudyInstanceUID=A\B, legal per PS3.4
// C.2.2.2.4) constrains on every listed value rather than only the first — the downstream matcher
// already decodes the list form. A key present with an empty value is a return key (universal
// match), recorded with an empty string so the catalogue returns it without constraining on it.
// Sequence values are left to the catalogue's own matching and not flattened here.
func matchKeysFromIdentifier(query *dicom.DataSet) map[dicom.Tag]string {
	if query == nil {
		return nil
	}
	match := make(map[dicom.Tag]string)
	for elem := range query.All() {
		if elem.Tag == dicom.TagQueryRetrieveLevel {
			continue
		}
		vals, ok := query.GetStrings(elem.Tag)
		if !ok {
			continue
		}
		match[elem.Tag] = strings.Join(vals, `\`)
	}
	return match
}

// mergeContexts appends extra presentation contexts to base, renumbering every context with unique
// odd IDs (1, 3, 5, …) so the merged set negotiates without ID collisions (PS3.8 §9.3.2.2). A context
// whose abstract syntax is already present in base is not duplicated.
func mergeContexts(base, extra []dimse.PresentationContext) []dimse.PresentationContext {
	seen := make(map[dicom.SOPClassUID]struct{}, len(base))
	for _, pc := range base {
		seen[pc.AbstractSyntax] = struct{}{}
	}
	merged := make([]dimse.PresentationContext, 0, len(base)+len(extra))
	merged = append(merged, base...)
	for _, pc := range extra {
		if _, dup := seen[pc.AbstractSyntax]; dup {
			continue
		}
		merged = append(merged, pc)
		seen[pc.AbstractSyntax] = struct{}{}
	}
	id := uint8(1)
	for i := range merged {
		merged[i].ID = id
		id += 2
	}
	return merged
}
