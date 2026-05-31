package dicom

import (
	"strings"
	"testing"
)

func TestNewUIDGeneratorRejectsEmptyRoot(t *testing.T) {
	if _, err := NewUIDGenerator(""); err == nil {
		t.Error("empty root must fail closed (no default registered root, Codex DCM-008)")
	}
}

func TestNewUIDGeneratorRejectsOverlongRoot(t *testing.T) {
	// Root must be <= 54 chars to leave room for a >= 9-digit suffix within 64.
	long := UID("1." + strings.Repeat("2.", 27)) // > 54 chars
	if _, err := NewUIDGenerator(long); err == nil {
		t.Error("over-54-char root must be rejected")
	}
}

func TestGenerateUnderConfiguredRoot(t *testing.T) {
	g, err := NewUIDGenerator("1.2.826.0.1.3680043.2.1143")
	if err != nil {
		t.Fatal(err)
	}
	u := g.Generate()
	if !strings.HasPrefix(string(u), "1.2.826.0.1.3680043.2.1143.") {
		t.Errorf("Generate() = %q, want the configured root prefix", u)
	}
	if !u.IsValid() {
		t.Errorf("Generate() produced an invalid UID: %q", u)
	}
	if len(u) > 64 {
		t.Errorf("Generate() exceeded 64 chars: %d", len(u))
	}
}

func TestGenerateIsUnique(t *testing.T) {
	g, _ := NewUIDGenerator("1.2.826.0.1.3680043.2.1143")
	seen := make(map[UID]bool)
	for i := 0; i < 1000; i++ {
		u := g.Generate()
		if seen[u] {
			t.Fatalf("duplicate UID minted: %q", u)
		}
		seen[u] = true
	}
}

func TestRandomUIDGeneratorUses225Root(t *testing.T) {
	g := NewRandomUIDGenerator()
	u := g.Generate()
	if !strings.HasPrefix(string(u), "2.25.") {
		t.Errorf("Generate() = %q, want 2.25. UUID root", u)
	}
	if !u.IsValid() || len(u) > 64 {
		t.Errorf("invalid 2.25. UID: %q (len %d)", u, len(u))
	}
}
