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
// stored over either plane is queryable and retrievable over both.
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
		dicomweb.WithRetrieveBackend(&dicomwebRetrieve{store: r.store}),
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
	handler := authMiddleware(env.auth, env.logger, mux)

	addr := joinHostPort(host, r.cfg.port)
	ln, err := listen(addr, env.tlsConfig)
	if err != nil {
		return err
	}
	r.setBound(ln.Addr())
	r.srv = &http.Server{Handler: handler}
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
// already names a concrete host.
func listen(addr string, cfg *tls.Config) (net.Listener, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	if cfg != nil {
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

// dicomwebRetrieve adapts the ObjectStore to the DICOMweb RetrieveBackend: a WADO-RS instance
// retrieve resolves the SOP Instance UID from the resource path and fetches the stored object.
type dicomwebRetrieve struct {
	store ObjectStore
}

// RetrieveInstance fetches the stored object by SOP Instance UID and verifies it lives under the
// study and series the request path names. The object store keys only on SOP Instance UID, so a
// fetch alone would return the instance regardless of the path's parent UIDs; a request for a valid
// SOP UID under the WRONG study/series must answer not-found, not return the instance under a path it
// does not belong to. A parent-UID mismatch is therefore mapped to ErrNotFound (the server answers
// 404), preserving the resource hierarchy WADO-RS addresses by.
func (b *dicomwebRetrieve) RetrieveInstance(ctx context.Context, p dicomweb.ResourcePath) (*dicom.DataSet, error) {
	ds, err := b.store.Get(ctx, dicom.SOPInstanceUID(p.Instance))
	if err != nil {
		return nil, err
	}
	if !datasetUnderPath(ds, p) {
		return nil, fmt.Errorf("%w: instance not under the requested study/series", ErrNotFound)
	}
	return ds, nil
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
