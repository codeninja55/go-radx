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
	// The returned fhir.Resource is a concrete resource of the role's release.
	Read(ctx context.Context, resourceType, id string) (fhir.Resource, error)

	// Create stores a new resource, assigning a server id when the resource has none, and returns the
	// stored resource. The caller (the role) validates with the release validator first; a resource
	// with error-severity issues never reaches Create.
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
	byKey   map[string]fhir.Resource // "ResourceType/id" -> resource
	counter atomic.Uint64
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
	}, nil
}

// Read returns the stored resource for (resourceType, id), or ErrNotFound.
func (m *MemoryRepository) Read(_ context.Context, resourceType, id string) (fhir.Resource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.byKey[storeKey(resourceType, id)]
	if !ok {
		return nil, fmt.Errorf("%w: %s/%s", ErrNotFound, resourceType, id)
	}
	return r, nil
}

// Create stores r, assigning a server id when it has none, and returns the stored resource. The id
// the repository assigns is set on the resource through the release adapter so the stored resource
// (and the read-back) carries it.
func (m *MemoryRepository) Create(_ context.Context, r fhir.Resource) (fhir.Resource, error) {
	id := m.adapter.resourceID(r)
	if id == "" {
		id = m.nextID()
		updated, err := m.adapter.withResourceID(r, id)
		if err != nil {
			return nil, err
		}
		r = updated
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byKey[storeKey(r.ResourceType(), id)] = r
	return r, nil
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

	return m.adapter.newSearchSet(int32(len(matches)), matches)
}

// Transaction processes a transaction Bundle by applying each entry through the repository and
// builds a transaction-response Bundle of the release. The development repository supports POST
// (create) and GET (read) entries, the two verbs the workflow exercise needs; an unsupported verb in
// an entry fails the transaction, never silently dropping it (PRD §9.2).
//
// The transaction is atomic, the all-or-nothing semantics the role advertises in its
// CapabilityStatement: a snapshot of the store is taken first, the entries are applied in order, and
// any failure restores the snapshot so a partially applied transaction never leaves committed
// resources behind. This matters for clinical data — a failed transaction must leave no orphaned
// resource a later read could surface. The id counter is not rolled back, so a retried transaction
// reuses no id; only the stored resources are restored.
func (m *MemoryRepository) Transaction(ctx context.Context, bundle fhir.Resource) (fhir.Resource, error) {
	snapshot := m.snapshot()
	resp, err := m.adapter.processTransaction(ctx, bundle, m)
	if err != nil {
		m.restore(snapshot)
		return nil, err
	}
	return resp, nil
}

// snapshot copies the store's keys under the read lock so a failed transaction can be rolled back to
// the pre-transaction state. The fhir.Resource values are immutable after Create (the adapter
// re-decodes on id assignment rather than mutating in place), so copying the map's key set without
// deep-copying each resource restores the exact pre-transaction state.
func (m *MemoryRepository) snapshot() map[string]fhir.Resource {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]fhir.Resource, len(m.byKey))
	for k, v := range m.byKey {
		out[k] = v
	}
	return out
}

// restore replaces the live store with the snapshot under the write lock, undoing every create a
// failed transaction applied so no partial commit survives.
func (m *MemoryRepository) restore(snapshot map[string]fhir.Resource) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byKey = snapshot
}

// nextID returns a fresh, monotonic server id. It is a counter rather than a UUID so the development
// repository's ids are short and predictable for tests; a production Repository assigns its own.
func (m *MemoryRepository) nextID() string {
	return strconv.FormatUint(m.counter.Add(1), 10)
}

// storeKey composes the map key for a (resourceType, id) pair.
func storeKey(resourceType, id string) string { return resourceType + "/" + id }

// errUnsupportedTxnVerb is returned by the development repository's transaction processing for a verb
// it does not handle, so an unsupported entry fails the transaction rather than being silently
// skipped.
var errUnsupportedTxnVerb = errors.New("server: transaction entry verb not supported by the in-memory repository")
