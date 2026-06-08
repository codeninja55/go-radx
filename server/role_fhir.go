package server

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/fhir"
)

const (
	// defaultFHIRPort is the FHIR REST HTTP listen port when WithFHIRPort is not set. 8080 is the
	// servers.md default.
	defaultFHIRPort = 8080
	// defaultFHIRBasePath is the URL prefix the FHIR REST API mounts under by default.
	defaultFHIRBasePath = "/fhir"
	// defaultFHIRMaxRequestBytes caps a FHIR request body before allocation when WithFHIRMaxRequestBytes
	// is not set, so a hostile client cannot stream an unbounded body into memory (PRD §9.3).
	defaultFHIRMaxRequestBytes int64 = 8 << 20
)

// mediaTypeFHIRJSON is the content type the FHIR role serves and the canonical content type a write
// declares, matching the fhir package's JSON-only v1 scope.
const mediaTypeFHIRJSON = "application/fhir+json"

// mediaTypeJSON is the generic JSON content type FHIR also permits on a request body, so a write that
// declares application/json (rather than the FHIR-specific application/fhir+json) is accepted.
const mediaTypeJSON = "application/json"

// fhirRoleConfig holds the resolved FHIRRole options.
type fhirRoleConfig struct {
	port            int
	basePath        string
	release         fhir.Release
	maxRequestBytes int64
}

// FHIRRoleOption configures a FHIRRole at construction.
type FHIRRoleOption func(*fhirRoleConfig)

// WithFHIRPort sets the HTTP listen port (default 8080).
func WithFHIRPort(port int) FHIRRoleOption {
	return func(c *fhirRoleConfig) { c.port = port }
}

// WithFHIRBasePath sets the URL prefix the FHIR REST API mounts under (default "/fhir"). To serve
// both releases from one process, mount two FHIRRoles on different base paths (for example
// "/fhir/r4" and "/fhir/r5"); the base path the request hit determines the release, never the
// request itself.
func WithFHIRBasePath(p string) FHIRRoleOption {
	return func(c *fhirRoleConfig) {
		if p != "" {
			c.basePath = "/" + strings.Trim(p, "/")
		}
	}
}

// WithFHIRRelease fixes the FHIR release the role serves (default fhir.R5). One role serves one
// release; the Repository it wraps stores resources of that release. An unsupported release is a
// construction error.
func WithFHIRRelease(r fhir.Release) FHIRRoleOption {
	return func(c *fhirRoleConfig) { c.release = r }
}

// WithFHIRMaxRequestBytes caps a FHIR request body before allocation (PRD §9.3).
func WithFHIRMaxRequestBytes(n int64) FHIRRoleOption {
	return func(c *fhirRoleConfig) {
		if n > 0 {
			c.maxRequestBytes = n
		}
	}
}

// FHIRRole configures the FHIR REST server over a pluggable Repository bound to one FHIR release. It
// serves the conformance subset of the FHIR HTTP API — read, create, search-type, and transaction
// over the workflow resource set — as application/fhir+json, validates inbound resources with the
// release validator, returns a release OperationOutcome for every error, and serves a
// CapabilityStatement at [base]/metadata advertising the supported interactions. It plugs into the
// Daemon exactly like the DIMSE, DICOMweb, and MLLP roles, honouring the loopback-default bind, the
// ErrInsecureBind fail-closed policy, and the HTTP Authenticator middleware.
type FHIRRole struct {
	cfg     fhirRoleConfig
	repo    Repository
	adapter releaseAdapter

	srv *http.Server

	mu    sync.Mutex
	bound net.Addr
}

// NewFHIRRole builds the FHIR REST role over a Repository bound to one FHIR release (default R5; set
// it with WithFHIRRelease). A nil repository, or a release the package does not support, is a
// configuration error so the role never starts misconfigured.
func NewFHIRRole(repo Repository, opts ...FHIRRoleOption) (*FHIRRole, error) {
	if repo == nil {
		return nil, errors.New("server: fhir role requires a non-nil Repository")
	}
	cfg := fhirRoleConfig{
		port:            defaultFHIRPort,
		basePath:        defaultFHIRBasePath,
		release:         fhir.R5,
		maxRequestBytes: defaultFHIRMaxRequestBytes,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	adapter, ok := adapterForRelease(cfg.release)
	if !ok {
		return nil, errors.New("server: fhir role configured with an unsupported FHIR release " + string(cfg.release))
	}
	return &FHIRRole{cfg: cfg, repo: repo, adapter: adapter}, nil
}

func (r *FHIRRole) name() string { return "fhir" }

// start binds the HTTP listener and serves the FHIR REST handler under the configured base path,
// behind the daemon's authentication middleware and TLS. The listener is bound synchronously so the
// daemon's fail-closed startup aborts on a bind failure before declaring success.
func (r *FHIRRole) start(_ context.Context, host string, env roleEnv) error {
	handler := &fhirHandler{
		repo:            r.repo,
		adapter:         r.adapter,
		basePath:        r.cfg.basePath,
		maxRequestBytes: r.cfg.maxRequestBytes,
		logger:          env.logger,
	}

	mux := http.NewServeMux()
	stripped := http.StripPrefix(r.cfg.basePath, handler)
	// Register both the exact base path and its subtree. The subtree pattern ("/fhir/") drives the
	// typed routes ("/fhir/Patient", "/fhir/metadata"), but net/http's ServeMux answers a request for
	// the exact base with no trailing slash ("/fhir") with a 301 subtree redirect, which turns a POST
	// into a 405 and never reaches the transaction handler. The transaction interaction is the system
	// root POST, and the client's Transaction posts to exactly the base, so the exact pattern must
	// route too — otherwise the client cannot run a transaction against this very server.
	mux.Handle(r.cfg.basePath, stripped)
	mux.Handle(r.cfg.basePath+"/", stripped)
	// The auth middleware rejects before the FHIR handler runs, so its 401 must still be a release
	// OperationOutcome to honour the role's FHIR-native error contract: a real Authenticator rejecting a
	// non-loopback request must not leak net/http's plain-text body. The auth decision is unchanged
	// (same Authenticator); only the rejection response format is the FHIR one.
	wrapped := authMiddleware(env.auth, env.logger, handler.writeUnauthorized, mux)

	addr := joinHostPort(host, r.cfg.port)
	ln, err := listen(addr, env.tlsConfig)
	if err != nil {
		return err
	}
	r.setBound(ln.Addr())
	r.srv = &http.Server{Handler: wrapped}
	go func() { _ = r.srv.Serve(ln) }()
	return nil
}

func (r *FHIRRole) setBound(a net.Addr) {
	r.mu.Lock()
	r.bound = a
	r.mu.Unlock()
}

func (r *FHIRRole) addr() net.Addr {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.bound
}

// shutdown stops accepting and drains in-flight requests bounded by ctx; a request still running at
// the deadline is force-closed and Shutdown returns the deadline error the daemon names the role in.
func (r *FHIRRole) shutdown(ctx context.Context) error {
	if r.srv == nil {
		return nil
	}
	return r.srv.Shutdown(ctx)
}

// fhirHandler serves the FHIR REST interactions under the role's base path. It is the HTTP edge of
// the role: it routes the request, decodes and validates the body, calls the Repository, and writes
// either the resource or a release OperationOutcome. It holds no per-request mutable state, so it is
// safe for the concurrent requests the http.Server dispatches.
type fhirHandler struct {
	repo            Repository
	adapter         releaseAdapter
	basePath        string
	maxRequestBytes int64
	logger          *zap.Logger
}

// readBody reads a request body bounded by the role's cap, so a hostile client cannot stream an
// unbounded body into memory (PRD §9.3). An over-cap body is an error the caller maps to a 413/400
// OperationOutcome rather than an OOM.
func (h *fhirHandler) readBody(r *http.Request) ([]byte, error) {
	limited := http.MaxBytesReader(nil, r.Body, h.maxRequestBytes)
	return io.ReadAll(limited)
}
