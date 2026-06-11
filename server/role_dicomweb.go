package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicomweb"
	"github.com/codeninja55/go-radx/dimse"
)

const (
	// defaultDICOMwebPort is the DICOMweb HTTP listen port when WithDICOMwebPort is not set. 8042 is
	// the Orthanc DICOMweb default the interop tests exercise.
	defaultDICOMwebPort = 8042
	// defaultDICOMwebBasePath is the URL prefix the DICOMweb services mount under by default.
	defaultDICOMwebBasePath = "/dicom-web"
)

// dicomwebRoleConfig holds the resolved DICOMwebRole options.
type dicomwebRoleConfig struct {
	port            int
	basePath        string
	maxRequestBytes int64
}

// DICOMwebRoleOption configures a DICOMwebRole at construction.
type DICOMwebRoleOption func(*dicomwebRoleConfig)

// WithDICOMwebPort sets the HTTP listen port (default 8042).
func WithDICOMwebPort(port int) DICOMwebRoleOption {
	return func(c *dicomwebRoleConfig) { c.port = port }
}

// WithDICOMwebBasePath sets the URL prefix the DICOMweb services mount under (default "/dicom-web").
func WithDICOMwebBasePath(p string) DICOMwebRoleOption {
	return func(c *dicomwebRoleConfig) {
		if p != "" {
			c.basePath = "/" + strings.Trim(p, "/")
		}
	}
}

// WithMaxRequestBytes caps a STOW-RS request body before allocation (PRD §9.3).
func WithMaxRequestBytes(n int64) DICOMwebRoleOption {
	return func(c *dicomwebRoleConfig) {
		if n > 0 {
			c.maxRequestBytes = n
		}
	}
}

// DICOMwebRole configures the DICOMweb HTTP server over the shared backends. STOW-RS stores through
// the ObjectStore (and indexes through the Catalogue), QIDO-RS queries the Catalogue, and WADO-RS
// retrieves from the ObjectStore — the same store and catalogue the DIMSE role uses, so a study
// stored over either plane is queryable and retrievable over both. The full WADO-RS retrieval
// surface is mounted: instance, study, series, metadata, frames, and bulkdata.
type DICOMwebRole struct {
	cfg   dicomwebRoleConfig
	store ObjectStore
	cat   Catalogue

	srv *http.Server

	mu    sync.Mutex
	bound net.Addr
}

// NewDICOMwebRole builds the DICOMweb HTTP role over the shared backends. A nil store or catalogue is
// a configuration error.
func NewDICOMwebRole(store ObjectStore, cat Catalogue, opts ...DICOMwebRoleOption) (*DICOMwebRole, error) {
	if store == nil {
		return nil, errors.New("server: dicomweb role requires a non-nil ObjectStore")
	}
	if cat == nil {
		return nil, errors.New("server: dicomweb role requires a non-nil Catalogue")
	}
	cfg := dicomwebRoleConfig{
		port:     defaultDICOMwebPort,
		basePath: defaultDICOMwebBasePath,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &DICOMwebRole{cfg: cfg, store: store, cat: cat}, nil
}

func (r *DICOMwebRole) name() string { return "dicomweb" }

// start binds the HTTP listener and serves the DICOMweb handler under the configured base path,
// behind the daemon's authentication middleware and TLS. The listener is bound synchronously so the
// daemon's fail-closed startup aborts on a bind failure before declaring success.
func (r *DICOMwebRole) start(ctx context.Context, host string, env roleEnv) error {
	webOpts := []dicomweb.ServerOption{
		dicomweb.WithStoreBackend(&dicomwebStore{store: r.store, cat: r.cat, logger: env.logger}),
		dicomweb.WithRetrieveBackend(&dicomwebRetrieve{store: r.store, cat: r.cat}),
		dicomweb.WithQueryBackend(&dicomwebQuery{cat: r.cat, store: r.store}),
	}
	if r.cfg.maxRequestBytes > 0 {
		webOpts = append(webOpts, dicomweb.WithMaxRequestBytes(r.cfg.maxRequestBytes))
	}
	web, err := dicomweb.NewServer(webOpts...)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle(r.cfg.basePath+"/", http.StripPrefix(r.cfg.basePath, web.Handler()))
	handler := authMiddleware(env.auth, env.logger, nil, mux)

	addr := joinHostPort(host, r.cfg.port)
	ln, err := listen(addr, env.tlsConfig)
	if err != nil {
		return err
	}
	r.setBound(ln.Addr())
	r.srv = &http.Server{Handler: handler, ReadHeaderTimeout: readHeaderTimeout}
	go func() { _ = r.srv.Serve(ln) }()
	return nil
}

// setBound records the bound listen address under the role lock so addr() (called concurrently by
// Daemon.Addrs) reads a consistent value.
func (r *DICOMwebRole) setBound(a net.Addr) {
	r.mu.Lock()
	r.bound = a
	r.mu.Unlock()
}

func (r *DICOMwebRole) addr() net.Addr {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.bound
}

// shutdown stops accepting and drains in-flight requests bounded by ctx; a request still running at
// the deadline is force-closed and Shutdown returns the deadline error the daemon names the role in.
func (r *DICOMwebRole) shutdown(ctx context.Context) error {
	if r.srv == nil {
		return nil
	}
	return r.srv.Shutdown(ctx)
}

// listen binds a TCP listener at addr, wrapping it in a TLS listener when cfg is non-nil so an HTTP
// role terminates TLS before serving (PRD §9.7). The bind is loopback-resolved by the caller, so addr
// already names a concrete host. The TLS 1.2 floor WithTLS documents is enforced here for the HTTP
// roles (the DIMSE role inherits the dimse AE clamp, the MLLP role the hl7v2 clamp): the caller's
// config is cloned and any MinVersion below 1.2 is raised, a higher pinned floor preserved.
func listen(addr string, cfg *tls.Config) (net.Listener, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	if cfg != nil {
		cfg = cfg.Clone()
		if cfg.MinVersion < tls.VersionTLS12 {
			cfg.MinVersion = tls.VersionTLS12
		}
		ln = tls.NewListener(ln, cfg)
	}
	return ln, nil
}

// dicomwebStore adapts the shared ObjectStore + Catalogue to the DICOMweb StoreBackend: a STOW-RS
// instance is persisted and indexed atomically per object, returning the store error so the server
// records the per-instance Failure Reason rather than reporting a clean success (PRD §9.2).
type dicomwebStore struct {
	store  ObjectStore
	cat    Catalogue
	logger *zap.Logger
}

func (b *dicomwebStore) Store(ctx context.Context, ds *dicom.DataSet) error {
	if err := b.store.Put(ctx, ds); err != nil {
		return err
	}
	return b.cat.Index(ctx, ds)
}

// dicomwebRetrieve adapts the shared ObjectStore + Catalogue to the DICOMweb RetrieveBackend and
// the optional study/series/metadata/frames/bulkdata retriever interfaces: an instance retrieve
// fetches the stored object by SOP Instance UID, while the study- and series-level retrievals
// enumerate the scope's instances through the Catalogue (the index that knows which instances live
// under a study/series) and fetch each full object from the ObjectStore. The store yields decoded
// datasets re-encodable in the default uncompressed transfer syntax, so RetrievedInstance carries
// no stored-syntax override.
type dicomwebRetrieve struct {
	store ObjectStore
	cat   Catalogue
}

// RetrieveInstance fetches the stored object by SOP Instance UID and verifies it lives under the
// study and series the request path names. The object store keys only on SOP Instance UID, so a
// fetch alone would return the instance regardless of the path's parent UIDs; a request for a valid
// SOP UID under the WRONG study/series must answer not-found, not return the instance under a path it
// does not belong to. A parent-UID mismatch and a store miss are therefore mapped to the dicomweb
// not-found sentinel (the server answers 404), preserving the resource hierarchy WADO-RS addresses
// by; any other store error is a backend fault the server answers 500 (the dicomweb retriever error
// contract).
func (b *dicomwebRetrieve) RetrieveInstance(ctx context.Context, p dicomweb.ResourcePath) (*dicom.DataSet, error) {
	ds, err := b.store.Get(ctx, dicom.SOPInstanceUID(p.Instance))
	if errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("%w: instance not stored", dicomweb.ErrNotFound)
	}
	if err != nil {
		return nil, err
	}
	if !datasetUnderPath(ds, p) {
		return nil, fmt.Errorf("%w: instance not under the requested study/series", dicomweb.ErrNotFound)
	}
	return ds, nil
}

// RetrieveStudy returns every stored instance of the study (dicomweb.StudyRetriever).
func (b *dicomwebRetrieve) RetrieveStudy(ctx context.Context, study dicom.UID) ([]dicomweb.RetrievedInstance, error) {
	return b.retrieveScope(ctx, map[dicom.Tag]string{
		dicom.TagStudyInstanceUID: string(study),
	})
}

// RetrieveSeries returns every stored instance of the series (dicomweb.SeriesRetriever).
func (b *dicomwebRetrieve) RetrieveSeries(ctx context.Context, study, series dicom.UID) ([]dicomweb.RetrievedInstance, error) {
	return b.retrieveScope(ctx, map[dicom.Tag]string{
		dicom.TagStudyInstanceUID:  string(study),
		dicom.TagSeriesInstanceUID: string(series),
	})
}

// retrieveScope enumerates the instances matching the exact-UID scope through the Catalogue at
// IMAGE level (each row carries the SOP Instance UID the ObjectStore fetch keys on) and fetches
// each full stored object. An empty scope returns no instances; the DICOMweb server answers 404
// for it, so an unknown study/series never reads as an empty success. A catalogue fault, and a
// CATALOGUED instance the object store cannot produce (a store/catalogue inconsistency, the
// TOCTOU window), are backend faults the server answers 500 — never the dicomweb not-found
// sentinel, which is reserved for a genuinely empty catalogue result (PRD §9.2).
func (b *dicomwebRetrieve) retrieveScope(ctx context.Context, match map[dicom.Tag]string) ([]dicomweb.RetrievedInstance, error) {
	var out []dicomweb.RetrievedInstance
	cq := CatalogueQuery{Level: dimse.QueryLevelImage, Match: match}
	for row, err := range b.cat.Query(ctx, cq) {
		if err != nil {
			return nil, err
		}
		instance, ok := row.GetString(dicom.TagSOPInstanceUID)
		if !ok || instance == "" {
			continue
		}
		ds, err := b.store.Get(ctx, dicom.SOPInstanceUID(instance))
		if errors.Is(err, ErrNotFound) {
			return nil, fmt.Errorf("server: catalogued instance is missing from the object store: %w", err)
		}
		if err != nil {
			return nil, err
		}
		out = append(out, dicomweb.RetrievedInstance{DataSet: ds})
	}
	return out, nil
}

// RetrieveMetadata returns the datasets whose DICOM-JSON metadata the server emits at the
// requested level (dicomweb.MetadataRetriever): the single instance, or every instance of the
// series/study.
func (b *dicomwebRetrieve) RetrieveMetadata(ctx context.Context, p dicomweb.ResourcePath) ([]dicomweb.RetrievedInstance, error) {
	switch p.Level() {
	case dicomweb.LevelInstance:
		ds, err := b.RetrieveInstance(ctx, p)
		if err != nil {
			return nil, err
		}
		return []dicomweb.RetrievedInstance{{DataSet: ds}}, nil
	case dicomweb.LevelSeries:
		return b.RetrieveSeries(ctx, p.Study, p.Series)
	default:
		return b.RetrieveStudy(ctx, p.Study)
	}
}

// RetrieveFrames returns the requested 1-based frames of the instance's pixel data as raw octets
// (dicomweb.FrameRetriever). The stored object is uncompressed (the store persists the default
// uncompressed syntax), so frames are sliced natively; an instance with no readable pixel data or
// a frame number outside the instance is the dicomweb not-found sentinel, which the server maps
// to 404 (PS3.18 §10.4.3).
func (b *dicomwebRetrieve) RetrieveFrames(ctx context.Context, p dicomweb.ResourcePath, frames []int) ([]dicomweb.BulkDataObject, error) {
	ds, err := b.RetrieveInstance(ctx, p)
	if err != nil {
		return nil, err
	}
	pd, err := dicom.NewPixelData(ds, dicom.ExplicitVRLittleEndian)
	if err != nil {
		return nil, fmt.Errorf("%w: instance has no readable pixel data", dicomweb.ErrNotFound)
	}
	// Collect only the requested frames, stopping once the highest requested frame is sliced so a
	// single-frame request of a large multi-frame instance does not buffer every frame.
	need := make(map[int][]byte, len(frames))
	maxFrame := 0
	for _, n := range frames {
		need[n] = nil
		if n > maxFrame {
			maxFrame = n
		}
	}
	for f, ferr := range pd.Frames() {
		if ferr != nil {
			return nil, ferr
		}
		number := f.Index + 1
		if _, ok := need[number]; ok {
			need[number] = f.Pixels
		}
		if number >= maxFrame {
			break
		}
	}
	out := make([]dicomweb.BulkDataObject, 0, len(frames))
	for _, n := range frames {
		data := need[n]
		if data == nil {
			return nil, fmt.Errorf("%w: frame %d is outside the instance", dicomweb.ErrNotFound, n)
		}
		out = append(out, dicomweb.BulkDataObject{Data: data})
	}
	return out, nil
}

// RetrieveBulkData backs the BARE bulkdata sub-resource: it returns the instance's top-level
// binary (OB/OW/OL/OV/UN) values as ordered octet-stream payloads (dicomweb.BulkDataRetriever).
// A locator-suffixed BulkDataURI (the per-attribute form the metadata response emits) is resolved
// by the dicomweb server itself against RetrieveInstance, so each URI returns exactly its own
// attribute value. An instance carrying none returns an empty set, which the server answers 404.
func (b *dicomwebRetrieve) RetrieveBulkData(ctx context.Context, p dicomweb.ResourcePath) ([]dicomweb.BulkDataObject, error) {
	ds, err := b.RetrieveInstance(ctx, p)
	if err != nil {
		return nil, err
	}
	var out []dicomweb.BulkDataObject
	for e := range ds.All() {
		if v, ok := e.Value.(*dicom.Bytes); ok {
			out = append(out, dicomweb.BulkDataObject{Data: v.Bytes()})
		}
	}
	return out, nil
}

// datasetUnderPath reports whether ds's StudyInstanceUID and SeriesInstanceUID match the parent UIDs
// the request path scopes. An empty path UID does not constrain (it was not supplied), so only a
// supplied, mismatching parent fails the check.
func datasetUnderPath(ds *dicom.DataSet, p dicomweb.ResourcePath) bool {
	if p.Study != "" {
		if v, _ := ds.GetString(dicom.TagStudyInstanceUID); v != string(p.Study) {
			return false
		}
	}
	if p.Series != "" {
		if v, _ := ds.GetString(dicom.TagSeriesInstanceUID); v != string(p.Series) {
			return false
		}
	}
	return true
}

// dicomwebQuery adapts the shared backends to the DICOMweb QueryBackend: a QIDO-RS search narrows on
// the catalogue and the full DICOM match decides which candidates match. The catalogue pushes only a
// conservative SQL narrowing and the Go matcher decides (list/range/wildcard syntax and unindexed keys
// are matched in Go, not as SQL equality); for a key the catalogue does not index, the stored dataset
// is fetched from the ObjectStore so the matcher sees the real attribute value. The candidate set is
// collapsed to the search level before it returns; the DICOMweb server re-applies its matcher,
// includefield projection, and offset/limit paging on top.
type dicomwebQuery struct {
	cat   Catalogue
	store ObjectStore
}

func (b *dicomwebQuery) Query(ctx context.Context, q dicomweb.QueryRequest) ([]*dicom.DataSet, error) {
	level := dimseLevelFromQIDO(q.Level)
	match := matchKeysFromQIDO(q)
	var out []*dicom.DataSet
	// The DICOMweb server re-applies its matcher over every match key and projects the level defaults
	// plus includefield. Ask the catalogue path to carry the includefield attributes onto the collapsed
	// rows so the projection has them; the match-key tags are retained by the catalogue path itself.
	for ds, err := range queryCatalogue(ctx, b.cat, b.store, match, level, q.IncludeFields, q.IncludeAll, q.Fuzzy) {
		if err != nil {
			return nil, err
		}
		out = append(out, ds)
	}
	return out, nil
}
