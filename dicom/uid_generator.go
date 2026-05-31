package dicom

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	// maxUIDLen is the PS3.5 UID character cap.
	maxUIDLen = 64
	// maxRootLen caps the configured root so a >= 9-digit suffix still fits within
	// the 64-character field (mirrors pydicom generate_uid).
	maxRootLen = 54
	// randomUIDRoot is the ISO/IEC 9834-8 UUID-derived root, which needs no
	// organisational registration.
	randomUIDRoot = "2.25."
)

// UIDGenerator mints conformant UIDs under a configured organisation root. go-radx
// ships no default registered root: a root-based generator must be constructed with
// an explicit, valid root (Codex DCM-008). The zero value is not usable.
type UIDGenerator struct {
	root   UID  // includes a trailing "." for the root-based form; empty for the 2.25. form
	random bool // true for the NewRandomUIDGenerator 2.25. UUID form
}

// NewUIDGenerator returns a generator that mints UIDs under root. root must be a valid
// UID prefix of at most 54 characters (leaving room for a >= 9-digit suffix within the
// 64-character limit). It returns an error if root is empty or invalid.
func NewUIDGenerator(root UID) (*UIDGenerator, error) {
	if root == "" {
		return nil, &ValueError{VR: VRUI, Msg: "UID generator root is empty (no default registered root)"}
	}
	if len(root) > maxRootLen {
		return nil, &ValueError{VR: VRUI, Msg: fmt.Sprintf("UID generator root exceeds %d characters (%d)", maxRootLen, len(root))}
	}
	if err := root.Validate(); err != nil {
		return nil, err
	}
	return &UIDGenerator{root: root + "."}, nil
}

// NewRandomUIDGenerator returns a generator using the ISO/IEC 9834-8 UUID-derived root
// "2.25.", which requires no organisational registration. Suffixes are the integer
// form of a random UUID. This is the recommended default when no organisation root is
// configured.
func NewRandomUIDGenerator() *UIDGenerator {
	return &UIDGenerator{root: randomUIDRoot, random: true}
}

// Generate returns a fresh UID under the generator's root, at most 64 characters, using
// a cryptographically random suffix.
func (g *UIDGenerator) Generate() UID {
	if g.random {
		return UID(string(g.root) + uuidInt())
	}
	// pydicom generate_uid: prefix + randbelow(10**(64-len(prefix))), truncated to 64.
	digits := maxUIDLen - len(g.root)
	maximum := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil)
	n, err := rand.Int(rand.Reader, maximum)
	if err != nil {
		panic(fmt.Sprintf("dicom: UIDGenerator: crypto/rand failed: %v", err))
	}
	s := string(g.root) + n.String()
	if len(s) > maxUIDLen {
		s = s[:maxUIDLen]
	}
	return UID(s)
}

// uuidInt returns the decimal integer form of a random RFC 4122 v4 UUID's 128 bits,
// mirroring Python's uuid.uuid4().int used by pydicom for the 2.25. root.
func uuidInt() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("dicom: UIDGenerator: crypto/rand failed: %v", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return new(big.Int).SetBytes(b[:]).String()
}
