package fhir

import (
	"errors"
	"sync"
)

// newSentinel builds a package sentinel error with the standard "fhir: " prefix.
// Keeping the construction in one place means every sentinel reads identically and
// callers can match it with errors.Is regardless of the wrapping a call site adds.
func newSentinel(msg string) error { return errors.New("fhir: " + msg) }

// Factory constructs a fresh, zero-valued resource of one concrete type. The
// generated registry pairs each resourceType discriminator with the constructor for
// its Go type, so UnmarshalResource can build the right value before decoding into
// it. A Factory returns a pointer to a generated resource struct (for example
// func() Resource { return &r5.Patient{} }).
type Factory func() Resource

// registry maps a FHIR resourceType discriminator to its concrete-type constructor.
// It is the one piece of package-level state. The generated per-release init()
// functions are its only writers, populating it before main runs; it is read-only in
// practice thereafter. The RWMutex guards it so the contract holds even if a caller
// misuses the exported RegisterFactory at runtime: a stray late registration can
// never race a concurrent reader, so the "no state a caller can race on" guarantee
// (PRD §9.4) is enforced, not merely asserted by convention.
//
// v1 ships only the R5 release, so the map is keyed by resourceType alone. When R4
// generation lands (Increment 15), R4 and R5 declare the same resourceType strings
// in distinct Go type spaces; resolving that collision (a release-scoped registry,
// or a release-qualified key) is a deliberate later-increment decision recorded in
// the FHIR generator plan, not silently papered over here.
var (
	registryMu sync.RWMutex
	registry   = map[string]Factory{}
)

// RegisterFactory records the constructor for a resourceType. It exists for the
// generated per-release registry init() to call; a consumer never calls it directly,
// because every resourceType a release supports is registered by that release
// package's init at import time. It is exported only because the generated release
// package and this root package are distinct packages, so the registration hook must
// cross the package boundary.
//
// RegisterFactory records the constructor for a resourceType under the registry
// write lock. It panics on an empty resourceType, a nil factory, or a duplicate
// registration: a duplicate means the generator emitted conflicting registrations
// (or R4/R5 collided before that collision was resolved), a build-time defect that
// must fail loudly rather than let one factory silently shadow the other.
func RegisterFactory(resourceType string, f Factory) {
	if resourceType == "" {
		panic("fhir: RegisterFactory: empty resourceType")
	}
	if f == nil {
		panic("fhir: RegisterFactory: nil factory for " + resourceType)
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[resourceType]; exists {
		panic("fhir: RegisterFactory: duplicate factory for resourceType " + resourceType)
	}
	registry[resourceType] = f
}

// lookupFactory returns the constructor for a resourceType and whether one is
// registered, under the registry read lock so it never races a registration.
func lookupFactory(resourceType string) (Factory, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	f, ok := registry[resourceType]
	return f, ok
}
