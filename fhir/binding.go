package fhir

import (
	"encoding/json"
	"fmt"
)

// DecodeMode selects how the JSON decoder treats an out-of-set code on a
// required-binding enum field. It is the explicit, per-call form of the unknown-code
// policy (PRD §9.2): the safe default is fail-closed, and lenient retention is opt-in.
// Threading the mode as a value (rather than a package-level toggle) keeps the policy
// free of global mutable state, so two concurrent decodes never race on a shared mode.
type DecodeMode int

const (
	// DecodeStrict rejects an out-of-set code with ErrUnknownCode. It is the default a
	// generated enum's UnmarshalJSON applies, so a non-conformant payload fails closed
	// rather than silently populating a required field with an invalid code.
	DecodeStrict DecodeMode = iota

	// DecodeLenient retains an out-of-set code verbatim on the field instead of
	// rejecting it, so a consumer ingesting partially-conformant data from the wild can
	// surface the issue through Validate rather than failing the whole decode. It is
	// opt-in precisely because fail-closed is the safe default.
	DecodeLenient
)

// DecodeCode validates a decoded code token against a required binding's closed set
// under the given mode. It is the single boundary helper every generated enum's
// UnmarshalJSON calls: it decodes the raw JSON into a string, and under DecodeStrict
// returns ErrUnknownCode (wrapped with the binding name and the offending token, never
// a patient value) when the token is not in the set. Under DecodeLenient an out-of-set
// token is returned with a nil error and ok=false, so the caller stores the raw value
// for Validate to flag.
//
// valid reports whether a token is a member of the closed set; the generated enum
// supplies a set-membership closure over its const set. bindingName names the enum for
// the error message (for example "AdministrativeGender"); it is a Go type name, not
// data, so it carries no PHI.
func DecodeCode(data []byte, valid func(string) bool, bindingName string, mode DecodeMode) (value string, ok bool, err error) {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return "", false, fmt.Errorf("fhir: decode %s code: %w", bindingName, err)
	}
	if valid(s) {
		return s, true, nil
	}
	if mode == DecodeLenient {
		return s, false, nil
	}
	return "", false, fmt.Errorf("fhir: %w: %q is not a valid %s", ErrUnknownCode, s, bindingName)
}

// ParseCode validates a code token against a required binding's closed set, always
// applying the strict rule regardless of any decode mode, so application code that
// wants a guaranteed-valid value can validate explicitly. It is the boundary helper the
// generated ParseXxx functions call; an out-of-set token returns ErrUnknownCode wrapped
// with the binding name and the offending token, never a patient value.
func ParseCode(s string, valid func(string) bool, bindingName string) (string, error) {
	if valid(s) {
		return s, nil
	}
	return "", fmt.Errorf("fhir: %w: %q is not a valid %s", ErrUnknownCode, s, bindingName)
}
