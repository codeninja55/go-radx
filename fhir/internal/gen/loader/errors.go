package loader

// LoadError is the typed error the loader returns for any fail-closed condition:
// a checksum mismatch, a missing required file, a malformed bundle, or an
// undecodable entry. It names the offending file and a non-sensitive detail; it
// never embeds file contents (a corrupted bundle is arbitrary bytes, and a
// definition bundle could in principle carry surprising data), so diagnostics
// stay safe to log.
type LoadError struct {
	// File is the bundle file the failure relates to, relative to the bundle
	// directory (for example "profiles-resources.json" or "SHA256SUMS").
	File string

	// Detail is a short, content-free description of the failure.
	Detail string

	// Err is an optional wrapped cause (for example an os or json error), exposed
	// through Unwrap for errors.Is/errors.As chaining.
	Err error
}

// Error implements the error interface. The "fhir/gen:" prefix matches the
// package's other diagnostics and identifies the build tool in CI logs.
func (e *LoadError) Error() string {
	msg := "fhir/gen: " + e.File + ": " + e.Detail
	if e.Err != nil {
		msg += ": " + e.Err.Error()
	}
	return msg
}

// Unwrap exposes the wrapped cause so callers can match it with errors.Is and
// errors.As.
func (e *LoadError) Unwrap() error { return e.Err }
