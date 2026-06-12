package server

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/codeninja55/go-radx/fhir"
)

// Repository is the storage seam for the FHIR REST server, the FHIR counterpart of ObjectStore +
// Catalogue: resources are stored and searched by type and id. A Repository is bound to one FHIR
// release (chosen via WithFHIRRelease on the role); the fhir.Resource values it exchanges are that
// release's concrete types (for example *r5.Patient), and the search and transaction Bundles it
// returns are that release's Bundle behind the fhir.Resource interface.
//
// The interface is release-neutral at the boundary by design: every r4/r5 resource and Bundle
// satisfies fhir.Resource, and the role's configured release tells the implementation which concrete
// type to materialise. servers.md types Search and Transaction against the configured release's
// concrete *Bundle; because Go forbids type parameters on interface methods (Go 1.26) and a single
// Repository value cannot name two release Bundles, the seam exchanges the release-agnostic
// fhir.Resource (the Bundle is one) and the implementation builds the configured release's Bundle.
// This keeps one Repository interface usable by an R4 or an R5 role without a parallel resource
// model.
//
// Implementations must be safe for concurrent use (PRD §9.4). Every method takes a context.Context
// for cancellation and deadline propagation. No method logs PHI (PRD §9.1).
type Repository interface {
	// Read returns the current version of one resource by type and id, or ErrNotFound when absent.
	// The returned fhir.Resource is a concrete resource of the role's release. The returned resource
	// carries its version metadata (meta.versionId, meta.lastUpdated), so the role can emit the ETag
	// and Last-Modified headers FHIR R5 http.html#read calls for.
	Read(ctx context.Context, resourceType, id string) (fhir.Resource, error)

	// VRead returns one specific version of a resource (the vread interaction, FHIR R5
	// http.html#vread): ErrNotFound when the resource or the named version is absent, ErrGone when
	// the named version exists but records a deletion (the spec's 410 path).
	VRead(ctx context.Context, resourceType, id, versionID string) (fhir.Resource, error)

	// History returns every stored version of one resource, newest first (the order a history
	// Bundle presents per FHIR R5 http.html#history), or ErrNotFound when the resource has never
	// existed. The role renders the versions into the release's history Bundle; the Repository owns
	// only the version record.
	History(ctx context.Context, resourceType, id string) ([]ResourceVersion, error)

	// Create stores a new resource under a server-assigned id and returns the stored resource. The
	// server always mints the id and ignores any client-supplied id, because FHIR create makes a new
	// resource and is not the update path: a create whose client id collided with an existing resource
	// must never overwrite it. The caller (the role) validates with the release validator first; a
	// resource with error-severity issues never reaches Create.
	Create(ctx context.Context, r fhir.Resource) (fhir.Resource, error)

	// Search executes a type-level search and returns a searchset Bundle of the role's release (built
	// with the release's NewSearchSet so total and the bdl-* invariants hold), behind the
	// fhir.Resource interface. params are the raw FHIR search parameters.
	Search(ctx context.Context, resourceType string, params url.Values) (fhir.Resource, error)

	// Transaction processes a transaction Bundle and returns the transaction-response Bundle of the
	// role's release, behind the fhir.Resource interface. A failed entry's handling (atomic rollback)
	// is the implementation's; the response reports each entry's outcome.
	Transaction(ctx context.Context, bundle fhir.Resource) (fhir.Resource, error)
}

// ResourceVersion is one version of a resource as the Repository's history records it: the stored
// resource (already carrying its meta.versionId/meta.lastUpdated), the version id, the instant the
// version was written, and whether the version records a deletion. A deleted version carries a nil
// Resource — the history Bundle entry for it has request/response but no resource body. The record
// is deliberately interaction-shaped so the deferred update/patch/delete interactions (wave 3)
// extend it by appending versions, never by reshaping it.
type ResourceVersion struct {
	Resource    fhir.Resource
	VersionID   string
	LastUpdated time.Time
	Deleted     bool
}

// MemoryRepository is a simple in-memory Repository so the FHIR role is runnable out of the box, the
// FHIR counterpart of the filesystem object store and SQLite catalogue. It stores resources keyed by
// (resourceType, id) in a map guarded by a mutex, assigns a monotonic server id on create, and
// implements a deliberately small search: an exact-match on the _id parameter, otherwise a
// type-level "all of this type" search. It is bound to one release through a release adapter so it
// builds that release's Bundle.
//
// It is a development default, not a database: it holds everything in memory, performs no profile or
// terminology validation beyond the role's structural validate, and offers no persistence. A
// production deployment supplies its own Repository over a real store (PRD §3.2 — not a PACS/archive,
// and the FHIR equivalent here).
type MemoryRepository struct {
	release fhir.Release
	adapter releaseAdapter

	mu      sync.RWMutex
	byKey   map[string]fhir.Resource     // "ResourceType/id" -> current version
	byVer   map[string][]ResourceVersion // "ResourceType/id" -> all versions, oldest first
	counter atomic.Uint64

	// now supplies the version timestamps; it is the field (defaulting to time.Now) so tests can
	// pin a deterministic clock. It must return a time with a location (UTC) so the FHIR instant
	// and the Last-Modified header are stable.
	now func() time.Time
}

// NewMemoryRepository returns an empty in-memory repository bound to the given release. An
// unsupported release is a construction error so a role is never wired to a repository that cannot
// build its Bundle.
func NewMemoryRepository(release fhir.Release) (*MemoryRepository, error) {
	adapter, ok := adapterForRelease(release)
	if !ok {
		return nil, fmt.Errorf("server: NewMemoryRepository unsupported FHIR release %q", release)
	}
	return &MemoryRepository{
		release: release,
		adapter: adapter,
		byKey:   map[string]fhir.Resource{},
		byVer:   map[string][]ResourceVersion{},
		now:     func() time.Time { return time.Now().UTC() },
	}, nil
}

// Read returns the stored resource for (resourceType, id), or ErrNotFound. It is a thin read-locked
// wrapper around readLocked; a transaction calls readLocked directly because it already holds the
// write lock for the whole transaction.
func (m *MemoryRepository) Read(_ context.Context, resourceType, id string) (fhir.Resource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.readLocked(resourceType, id)
}

// readLocked returns the stored resource for (resourceType, id), or ErrNotFound. The caller must hold
// m.mu (read or write); it never locks, so it composes inside the transaction's write-locked section
// without deadlocking.
func (m *MemoryRepository) readLocked(resourceType, id string) (fhir.Resource, error) {
	r, ok := m.byKey[storeKey(resourceType, id)]
	if !ok {
		return nil, fmt.Errorf("%w: %s/%s", ErrNotFound, resourceType, id)
	}
	return r, nil
}

// Create stores r under a server-assigned id and returns the stored resource. The repository always
// mints the id (ignoring any client-supplied id) and sets it on the resource through the release
// adapter, so the stored resource (and the read-back) carries the server id and a create never
// overwrites an existing resource. It is a thin write-locked wrapper around createLocked; a
// transaction calls createLocked directly because it already holds the write lock.
func (m *MemoryRepository) Create(_ context.Context, r fhir.Resource) (fhir.Resource, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.createLocked(r)
}

// createLocked stores r under a freshly minted server id and returns the stored resource. The caller
// must hold m.mu for writing; it never locks, so the transaction can apply many creates inside one
// write-locked section. The id is assigned (nextID is itself atomic) and the resource is stored under
// the held lock, so a concurrent Create or transaction is serialised, not interleaved.
//
// The server always assigns the id and ignores any client-supplied id: FHIR create makes a new
// resource, it is not the update path, and this role exposes no update. Honouring a client id would
// let a second create with the same id silently clobber the first (bypassing concurrency control), so
// every create instead gets a fresh unique id and the new id is set on the stored resource through the
// release adapter. Because nextID is monotonic, the minted id never collides with an existing entry.
//
// The create is version 1 of the resource: the stored resource carries meta.versionId "1" and
// meta.lastUpdated, and the version is appended to the resource's history. A future update appends
// version n+1 the same way (the version store is interaction-shaped — see ResourceVersion); only
// create writes versions today because update/patch/delete are deferred.
func (m *MemoryRepository) createLocked(r fhir.Resource) (fhir.Resource, error) {
	id := m.nextID()
	at := m.now()
	r, err := m.adapter.withResourceVersion(r, id, initialVersionID, fhirInstant(at))
	if err != nil {
		return nil, err
	}
	key := storeKey(r.ResourceType(), id)
	m.byKey[key] = r
	m.byVer[key] = append(m.byVer[key], ResourceVersion{
		Resource:    r,
		VersionID:   initialVersionID,
		LastUpdated: at,
	})
	return r, nil
}

// VRead returns the stored version of (resourceType, id) named by versionID: ErrNotFound when the
// resource or the version is absent, ErrGone when the version records a deletion (FHIR R5
// http.html#vread's 410 path; the in-memory repository writes no deletions yet, but the check keeps
// the contract honest for the version store's future delete support).
func (m *MemoryRepository) VRead(_ context.Context, resourceType, id, versionID string) (fhir.Resource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	versions, ok := m.byVer[storeKey(resourceType, id)]
	if !ok {
		return nil, fmt.Errorf("%w: %s/%s", ErrNotFound, resourceType, id)
	}
	for i := range versions {
		v := &versions[i]
		if v.VersionID != versionID {
			continue
		}
		if v.Deleted {
			return nil, fmt.Errorf("%w: %s/%s/_history/%s", ErrGone, resourceType, id, versionID)
		}
		return v.Resource, nil
	}
	return nil, fmt.Errorf("%w: %s/%s/_history/%s", ErrNotFound, resourceType, id, versionID)
}

// History returns every stored version of (resourceType, id), newest first, or ErrNotFound when the
// resource has never existed. The slice is a copy, so a caller cannot mutate the store's history.
func (m *MemoryRepository) History(_ context.Context, resourceType, id string) ([]ResourceVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	versions, ok := m.byVer[storeKey(resourceType, id)]
	if !ok {
		return nil, fmt.Errorf("%w: %s/%s", ErrNotFound, resourceType, id)
	}
	out := make([]ResourceVersion, 0, len(versions))
	for i := len(versions) - 1; i >= 0; i-- {
		out = append(out, versions[i])
	}
	return out, nil
}

// Search implements the small search the development repository supports: an exact match on _id
// when present, otherwise every resource of the requested type. It returns a searchset Bundle of the
// repository's release with total set to the match count. Unrecognised parameters are ignored rather
// than rejected, so a client's _count/_sort/_include do not fail the search; the documented limit is
// that this repository does not honour arbitrary search parameters (a production Repository does).
func (m *MemoryRepository) Search(_ context.Context, resourceType string, params url.Values) (fhir.Resource, error) {
	wantID := params.Get("_id")

	m.mu.RLock()
	var matches []fhir.Resource
	prefix := resourceType + "/"
	for key, r := range m.byKey {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if wantID != "" && m.adapter.resourceID(r) != wantID {
			continue
		}
		matches = append(matches, r)
	}
	m.mu.RUnlock()

	return m.adapter.newSearchSet(int32(len(matches)), matches) // #nosec G115 -- an in-memory match count is far below int32
}

// Transaction processes a transaction Bundle by applying each entry through the repository and
// builds a transaction-response Bundle of the release. The development repository supports POST
// (create) and GET (read) entries, the two verbs the workflow exercise needs; an unsupported verb in
// an entry fails the transaction, never silently dropping it (PRD §9.2).
//
// The transaction is atomic and isolated. It holds the repository's write lock for the entire
// snapshot/apply window, so no concurrent Create or transaction can commit while it runs: the
// all-or-nothing outcome is decided against a store no one else is mutating. The entries are applied
// to a staging copy of the store; only on full success does the staging copy become the live store,
// so a failure throws the staging copy away and the live store is left exactly as it was — including
// every write another goroutine committed before this transaction took the lock. The earlier design
// snapshotted without holding the lock and restored an older snapshot on failure, which silently
// discarded concurrent writes; holding the lock across the whole window is what makes "atomic" also
// mean "loses no concurrent write".
//
// Because the adapter's per-entry operations call back through the Repository interface, the
// transaction passes a lockedView that routes those calls to the unlocked createLocked/readLocked
// helpers against the staging copy, so applying many entries never re-acquires the held lock and
// never deadlocks. The id counter is not rolled back, so a retried transaction reuses no id; only the
// stored resources are.
func (m *MemoryRepository) Transaction(ctx context.Context, bundle fhir.Resource) (fhir.Resource, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	live := m.byKey
	staging := make(map[string]fhir.Resource, len(live))
	for k, v := range live {
		staging[k] = v
	}
	// The version store is staged alongside the current-version map for the same atomicity: a failed
	// transaction must leave no orphan history entries behind. The per-key version slices are copied
	// (not just the map) because createLocked appends to them in place.
	liveVer := m.byVer
	stagingVer := make(map[string][]ResourceVersion, len(liveVer))
	for k, vs := range liveVer {
		stagingVer[k] = append([]ResourceVersion(nil), vs...)
	}

	// Apply the entries against the staging copies via a lockedView, so the adapter's repo.Create /
	// repo.Read route to the unlocked helpers and never re-take the held write lock.
	m.byKey = staging
	m.byVer = stagingVer
	resp, err := m.adapter.processTransaction(ctx, bundle, lockedView{repo: m})
	if err != nil {
		// Failure: discard the staging copies and restore the live store, keeping every concurrent
		// write that committed before this transaction took the lock.
		m.byKey = live
		m.byVer = liveVer
		return nil, err
	}
	// Success: the staging copies (already installed) become the live store, committing every entry.
	return resp, nil
}

// lockedView adapts a MemoryRepository whose write lock the caller already holds into a Repository
// whose Create/Read use the unlocked helpers, so a transaction can drive the release adapter's
// per-entry callbacks without the adapter re-acquiring the lock. Search, VRead, History, and
// Transaction are not used inside a transaction and are not supported on the view.
type lockedView struct {
	repo *MemoryRepository
}

func (v lockedView) Read(_ context.Context, resourceType, id string) (fhir.Resource, error) {
	return v.repo.readLocked(resourceType, id)
}

func (v lockedView) Create(_ context.Context, r fhir.Resource) (fhir.Resource, error) {
	return v.repo.createLocked(r)
}

func (v lockedView) Search(context.Context, string, url.Values) (fhir.Resource, error) {
	return nil, fmt.Errorf("server: search is not supported inside a transaction")
}

func (v lockedView) VRead(context.Context, string, string, string) (fhir.Resource, error) {
	return nil, fmt.Errorf("server: vread is not supported inside a transaction")
}

func (v lockedView) History(context.Context, string, string) ([]ResourceVersion, error) {
	return nil, fmt.Errorf("server: history is not supported inside a transaction")
}

func (v lockedView) Transaction(context.Context, fhir.Resource) (fhir.Resource, error) {
	return nil, fmt.Errorf("server: nested transactions are not supported")
}

// nextID returns a fresh, monotonic server id. It is a counter rather than a UUID so the development
// repository's ids are short and predictable for tests; a production Repository assigns its own.
func (m *MemoryRepository) nextID() string {
	return strconv.FormatUint(m.counter.Add(1), 10)
}

// storeKey composes the map key for a (resourceType, id) pair.
func storeKey(resourceType, id string) string { return resourceType + "/" + id }

// initialVersionID is the meta.versionId a create writes. Versions are small monotonic integers per
// resource ("1", "2", ...), the convention FHIR's examples use; a future update mints the next one.
const initialVersionID = "1"

// fhirInstant renders a time as a FHIR instant (RFC 3339 with millisecond precision and an explicit
// offset), the lexical form meta.lastUpdated carries. The time is normalised to UTC so the stored
// instant and the Last-Modified header derived from it agree run to run.
func fhirInstant(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z07:00")
}

// errUnsupportedTxnVerb is returned by the development repository's transaction processing for a verb
// it does not handle, so an unsupported entry fails the transaction rather than being silently
// skipped.
var errUnsupportedTxnVerb = errors.New("server: transaction entry verb not supported by the in-memory repository")
