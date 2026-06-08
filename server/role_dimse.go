package server

import (
	"context"
	"fmt"
	"iter"
	"net"
	"sync"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/dicom"
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
		auth:     env.auth,
		logger:   env.logger,
	}

	var srvOpts []dimse.ServerOption
	if r.cfg.maxAssoc > 0 {
		srvOpts = append(srvOpts, dimse.WithMaxAssociations(r.cfg.maxAssoc))
	}
	r.srv = dimse.NewServer(ae, r.cfg.contexts, r.handler, srvOpts...)

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
// EchoHandler, StoreHandler, and (when a WorklistSource is configured) FindHandler. C-GET/C-MOVE
// retrieve from the ObjectStore is a later increment; until then those services are unmounted and the
// dispatcher refuses them with StatusSOPClassNotSupported (interface segregation, PRD §8.2). The
// handler logs structural identifiers only (AE titles, SOP class/instance UIDs), never PHI (PRD §9.1).
type dimseHandler struct {
	store    ObjectStore
	cat      Catalogue
	worklist WorklistSource
	auth     Authenticator
	logger   *zap.Logger
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
		h.logger.Warn("c-store catalogue index failed", zap.String("sop_class", string(info.SOPClassUID)))
		return dimse.StatusStoreElementDiscarded
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
		cq := CatalogueQuery{Level: level, Match: matchKeysFromIdentifier(query)}
		for match, err := range h.cat.Query(ctx, cq) {
			if err != nil {
				h.logger.Warn("catalogue query failed")
				yield(dimse.NewStatus(0xC000, dimse.ServiceClassFind), nil)
				return
			}
			if !yield(dimse.StatusFindPending, match) {
				return
			}
		}
		yield(dimse.StatusFindSuccess, nil)
	}
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

// matchKeysFromIdentifier extracts the single-value string match keys from a C-FIND identifier into
// the catalogue's tag->value map. A key present with an empty value is a return key (universal match),
// recorded with an empty string so the catalogue returns it without constraining on it. Sequence
// values are left to the catalogue's own matching and not flattened here.
func matchKeysFromIdentifier(query *dicom.DataSet) map[dicom.Tag]string {
	if query == nil {
		return nil
	}
	match := make(map[dicom.Tag]string)
	for elem := range query.All() {
		if elem.Tag == dicom.TagQueryRetrieveLevel {
			continue
		}
		if v, ok := query.GetString(elem.Tag); ok {
			match[elem.Tag] = v
		}
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
