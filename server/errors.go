package server

import "errors"

// Sentinel errors the server package returns. Compare with errors.Is; a role or backend wraps one
// with %w so a caller distinguishes the cause without string matching (PRD §8.2). None carries PHI.
var (
	// ErrNotFound is returned by ObjectStore.Get/Delete, Catalogue.Remove, and Repository.Read when
	// the requested object or resource is absent. It lets a caller distinguish a genuine miss from a
	// backend fault, never reporting a silent success on missing data (PRD §9.2).
	ErrNotFound = errors.New("server: object or resource not found")

	// ErrGone is returned by Repository.VRead when the named version exists in the resource's
	// history but records a deletion: FHIR R5 http.html#vread answers that version with 410 Gone,
	// distinct from the 404 an unknown resource or version gets. The in-memory repository cannot
	// produce a deleted version yet (delete is a deferred interaction), but the contract names the
	// sentinel so a versioned production Repository and the role agree on the 404-vs-410 split.
	ErrGone = errors.New("server: resource version is deleted")

	// ErrInsecureBind is returned by New when a non-loopback WithBind is combined with no explicit
	// Authenticator, so the fail-closed default cannot be bypassed by omission. Pass
	// WithAuthenticator(AllowAll()) to override deliberately (see the bind policy in servers.md).
	ErrInsecureBind = errors.New("server: non-loopback bind requires an explicit Authenticator")

	// ErrRoleNotMounted is returned when a daemon operation names a role that was not mounted.
	ErrRoleNotMounted = errors.New("server: requested role is not mounted on this daemon")

	// ErrShutdownTimeout is returned (wrapped, naming the role) when a role does not drain within the
	// configured shutdown deadline. It is an honest report that shutdown was not clean, never a silent
	// success (PRD §9.2).
	ErrShutdownTimeout = errors.New("server: graceful shutdown exceeded its deadline")
)
