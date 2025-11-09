// Package uid provides DICOM Unique Identifier (UID) handling and validation.
//
// UIDs are used throughout DICOM to uniquely identify various entities including
// transfer syntaxes, SOP classes, and instances.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_9
package uid

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"
)

// UID represents a DICOM Unique Identifier.
//
// UIDs are character strings composed of numeric components separated by periods (.).
// They follow the ISO 8824 object identifier format and must:
//   - Contain only digits (0-9) and periods (.)
//   - Not exceed 64 characters in length
//   - Not have leading or trailing periods
//   - Not have consecutive periods
//   - Not have leading zeros in components (except "0" by itself)
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_9.1
type UID struct {
	value string
}

// String returns the string representation of the UID.
func (u UID) String() string {
	return u.value
}

// Equals returns true if this UID equals the other UID.
func (u UID) Equals(other UID) bool {
	return u.value == other.value
}

// IsValid checks if a string is a valid DICOM UID.
//
// Validation rules per DICOM Part 5 Section 9.1:
//   - Maximum length of 64 characters
//   - Contains only digits and periods
//   - Does not start or end with a period
//   - Does not contain consecutive periods
//   - Components do not have leading zeros (except "0" by itself)
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_9.1
func IsValid(s string) bool {
	// Empty string is not valid
	if s == "" {
		return false
	}
	// Maximum length is 64 characters
	if len(s) > 64 {
		return false
	}
	// Must not start or end with a period
	if s[0] == '.' || s[len(s)-1] == '.' {
		return false
	}
	// Split into components
	components := strings.Split(s, ".")
	if len(components) < 2 {
		return false
	}

	for _, comp := range components {
		// Empty component (consecutive dots)
		if comp == "" {
			return false
		}
		// Check for leading zeros (except "0" by itself)
		if len(comp) > 1 && comp[0] == '0' {
			return false
		}
		// Check that all characters are digits
		for _, ch := range comp {
			if ch < '0' || ch > '9' {
				return false
			}
		}
	}
	return true
}

// Parse validates and creates a UID from a string.
//
// Returns an error if the string is not a valid DICOM UID.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_9.1
func Parse(s string) (UID, error) {
	if !IsValid(s) {
		return UID{}, fmt.Errorf("invalid UID: %q", s)
	}
	return UID{value: s}, nil
}

// MustParse validates and creates a UID from a string, panicking on error.
// This should only be used for well-known UIDs that are guaranteed to be valid.
func MustParse(s string) UID {
	u, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

var (
	// ErrInvalidUID is returned when a UID string is invalid.
	ErrInvalidUID = errors.New("invalid UID")
)

// Transfer Syntax UIDs and SOP Class UIDs are now exported as constants in:
//   - transfer_syntax_uids.go (auto-generated, 63 Transfer Syntax UIDs)
//   - sop_class_uids.go (auto-generated, 318 SOP Class UIDs)
//
// All UIDs with metadata are available via:
//   - uidMap (all 519 UIDs from DICOM Standard 2024b)
//   - Lookup(uid string) - lookup by UID string
//   - FindByName(name string) - lookup by human-readable name
//   - FindAllByType(Type) - find all UIDs of a specific type
//
// See README.md in this package for usage examples.

// Lookup returns the Info for the given UID string.
// Returns false if the UID is not found in the standard dictionary.
func Lookup(uid string) (Info, bool) {
	info, ok := uidMap[uid]
	return info, ok
}

// Name returns the human-readable name for the given UID.
// Returns empty string if the UID is not found.
func Name(uid string) string {
	if info, ok := uidMap[uid]; ok {
		return info.Name
	}
	return ""
}

// IsRetired returns true if the given UID has been retired from the DICOM standard.
// Returns false if the UID is not found or is not retired.
func IsRetired(uid string) bool {
	if info, ok := uidMap[uid]; ok {
		return info.Retired
	}
	return false
}

// GetType returns the Type category for the given UID.
// Returns empty Type if the UID is not found.
func GetType(uid string) Type {
	if info, ok := uidMap[uid]; ok {
		return info.Type
	}
	return ""
}

// IsTransferSyntax returns true if the given UID represents a Transfer Syntax.
func IsTransferSyntax(uid string) bool {
	if info, ok := uidMap[uid]; ok {
		return info.Type == TypeTransferSyntax
	}
	return false
}

// IsSOPClass returns true if the given UID represents a SOP Class.
func IsSOPClass(uid string) bool {
	if info, ok := uidMap[uid]; ok {
		return info.Type == TypeSOPClass || info.Type == TypeMetaSOPClass
	}
	return false
}

// Find returns the Info for the given UID string, returning an error if not found.
// This is similar to Lookup but returns an error instead of a boolean for error handling.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part06.html#chapter_A
func Find(uid string) (Info, error) {
	info, ok := uidMap[uid]
	if !ok {
		return Info{}, fmt.Errorf("UID %q not found in dictionary", uid)
	}
	return info, nil
}

// FindByName searches for a UID by its human-readable name.
// Returns an error if no UID with the given name is found.
// Note: This performs a linear search through all UIDs, so it's less efficient than Find.
//
// Example: FindByName("Implicit VR Little Endian")
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part06.html#chapter_A
func FindByName(name string) (Info, error) {
	if name == "" {
		return Info{}, fmt.Errorf("UID name cannot be empty")
	}
	for _, info := range uidMap {
		if info.Name == name {
			return info, nil
		}
	}
	return Info{}, fmt.Errorf("UID with name %q not found in dictionary", name)
}

// FindAllByType returns all UIDs of the specified Type.
// Returns an empty slice if no UIDs of the given type are found.
// Note: The returned slice is a copy and can be safely modified.
//
// Example: FindAllByType(TypeTransferSyntax)
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part06.html#chapter_A
func FindAllByType(t Type) []Info {
	var results []Info
	for _, info := range uidMap {
		if info.Type == t {
			results = append(results, info)
		}
	}
	return results
}

// Generate creates a new unique DICOM UID.
//
// This implementation uses a combination of:
//   - Organizational root: "1.2.826.0.1.3680043.10" (PixelMed reserved root)
//   - Unix timestamp in microseconds
//   - Random 32-bit value for uniqueness
//
// The generated UID follows DICOM UID rules and is guaranteed to be unique.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_9.1
//
// Example:
//
//	studyUID := uid.Generate()
//	fmt.Println(studyUID) // e.g., "1.2.826.0.1.3680043.10.1234567890.12345"
func Generate() string {
	// Use PixelMed reserved root for generated UIDs
	// This is commonly used for DICOM implementations
	const orgRoot = "1.2.826.0.1.3680043.10"

	// Get current timestamp in microseconds
	timestamp := time.Now().UnixMicro()

	// Generate random 32-bit value for additional uniqueness
	var randomBytes [4]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		// Fallback to timestamp-only if random fails
		return fmt.Sprintf("%s.%d", orgRoot, timestamp)
	}
	randomValue := binary.BigEndian.Uint32(randomBytes[:])

	// Construct UID: orgRoot.timestamp.random
	return fmt.Sprintf("%s.%d.%d", orgRoot, timestamp, randomValue)
}
