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

// Registry holds one FHIR release's resourceType-keyed dispatch and validation
// metadata: the resourceType→factory map UnmarshalResource dispatches through, the
// per-resource validation descriptors Validate consumes, and the summary descriptors
// MarshalSummary consumes. Each release package (fhir/r4, fhir/r5) owns its own
// Registry instance, populated by that release's generated init() functions and read
// through the release-local UnmarshalResource/Validate/MarshalSummary entry points.
//
// A release-scoped Registry is what makes a dual-release build possible. A FHIR JSON
// resource carries a "resourceType" but not its release, so a single registry keyed
// by resourceType alone is inherently ambiguous between R4 and R5: both releases
// declare "Account", "Patient", and the rest as distinct Go types. Giving each
// release its own Registry keeps the keys release-local, so importing both fhir/r4
// and fhir/r5 no longer collides at init — and an R4 Bundle decodes its contained
// resources as R4 types while an R5 Bundle decodes them as R5 types, because each
// release's generated decode path dispatches through its own Registry.
//
// Each map is written only by the owning release's generated init() before main runs
// and is read-only in practice thereafter. The RWMutexes guard them so the contract
// holds even if a caller misuses the exported Register* methods at runtime: a stray
// late registration can never race a concurrent reader, so the "no state a caller can
// race on" guarantee (PRD §9.4) is enforced, not merely asserted by convention.
type Registry struct {
	factoryMu sync.RWMutex
	factory   map[string]Factory

	validationMu sync.RWMutex
	validation   map[string]ValidationDescriptor

	summaryMu sync.RWMutex
	summary   map[string]SummaryDescriptor
}

// NewRegistry returns an empty, ready-to-populate Registry. A release package
// constructs one at package scope and fills it from its generated init() functions;
// the root fhir package keeps one default instance for callers that decode or
// validate through the release-agnostic package-level functions.
func NewRegistry() *Registry {
	return &Registry{
		factory:    map[string]Factory{},
		validation: map[string]ValidationDescriptor{},
		summary:    map[string]SummaryDescriptor{},
	}
}

// RegisterFactory records the constructor for a resourceType. It exists for the
// generated per-release registry init() to call; a consumer never calls it directly,
// because every resourceType a release supports is registered by that release
// package's init at import time.
//
// It panics on an empty resourceType, a nil factory, or a duplicate registration: a
// duplicate within one release means the generator emitted conflicting registrations,
// a build-time defect that must fail loudly rather than let one factory silently
// shadow the other. Two releases no longer collide here because each owns its own
// Registry.
func (reg *Registry) RegisterFactory(resourceType string, f Factory) {
	if resourceType == "" {
		panic("fhir: RegisterFactory: empty resourceType")
	}
	if f == nil {
		panic("fhir: RegisterFactory: nil factory for " + resourceType)
	}
	reg.factoryMu.Lock()
	defer reg.factoryMu.Unlock()
	if _, exists := reg.factory[resourceType]; exists {
		panic("fhir: RegisterFactory: duplicate factory for resourceType " + resourceType)
	}
	reg.factory[resourceType] = f
}

// lookupFactory returns the constructor for a resourceType and whether one is
// registered, under the registry read lock so it never races a registration.
func (reg *Registry) lookupFactory(resourceType string) (Factory, bool) {
	reg.factoryMu.RLock()
	defer reg.factoryMu.RUnlock()
	f, ok := reg.factory[resourceType]
	return f, ok
}

// defaultRegistry backs the release-agnostic package-level RegisterFactory,
// UnmarshalResource, Validate, and MarshalSummary functions. The release packages do
// not register into it; it exists so the root package's own machinery is usable and
// independently testable without importing a release package.
var defaultRegistry = NewRegistry()

// RegisterFactory records the constructor for a resourceType in the root package's
// default registry. It is the release-agnostic counterpart to (*Registry).RegisterFactory;
// a release package registers into its own Registry instead.
func RegisterFactory(resourceType string, f Factory) {
	defaultRegistry.RegisterFactory(resourceType, f)
}

// lookupFactory returns the constructor for a resourceType from the default registry.
func lookupFactory(resourceType string) (Factory, bool) {
	return defaultRegistry.lookupFactory(resourceType)
}
