package loader

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyChecksumsRejectsMismatch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, dir, "a.json", []byte(`{"x":1}`))
	writeFile(t, dir, "b.json", []byte(`{"y":2}`))
	writeSums(t, dir, map[string]string{
		"a.json": sha256Hex([]byte(`{"x":1}`)),
		"b.json": sha256Hex([]byte(`{"y":2}`)),
	})
	if err := verifyChecksums(dir); err != nil {
		t.Fatalf("verifyChecksums on matching bundle: %v", err)
	}

	writeFile(t, dir, "a.json", []byte(`{"x":2}`)) // drift
	err := verifyChecksums(dir)
	if err == nil {
		t.Fatal("verifyChecksums should reject a drifted file")
	}
	var le *LoadError
	if !errors.As(err, &le) {
		t.Fatalf("error = %T, want *LoadError", err)
	}
	if !strings.Contains(le.Error(), "a.json") {
		t.Errorf("error %q should name the offending file", le.Error())
	}
	// The error must not leak the file's contents: neither the drifted bytes nor
	// the expected bytes should appear in the diagnostic.
	if strings.Contains(le.Error(), `{"x":2}`) || strings.Contains(le.Error(), `{"x":1}`) {
		t.Errorf("error %q should not contain file bytes", le.Error())
	}
}

func TestVerifyChecksumsRejectsMissingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, dir, "present.json", []byte(`{"x":1}`))
	writeSums(t, dir, map[string]string{
		"present.json": sha256Hex([]byte(`{"x":1}`)),
		"absent.json":  sha256Hex([]byte(`{"z":9}`)),
	})

	err := verifyChecksums(dir)
	if err == nil {
		t.Fatal("verifyChecksums should reject a missing listed file")
	}
	var le *LoadError
	if !errors.As(err, &le) {
		t.Fatalf("error = %T, want *LoadError", err)
	}
	if !strings.Contains(le.Error(), "absent.json") {
		t.Errorf("error %q should name the missing file", le.Error())
	}
}

func TestVerifyChecksumsRejectsMissingManifest(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, dir, "a.json", []byte(`{"x":1}`))

	err := verifyChecksums(dir)
	if err == nil {
		t.Fatal("verifyChecksums should reject a directory without SHA256SUMS")
	}
	var le *LoadError
	if !errors.As(err, &le) {
		t.Fatalf("error = %T, want *LoadError", err)
	}
	if !strings.Contains(le.Error(), sumsFile) {
		t.Errorf("error %q should name the manifest %q", le.Error(), sumsFile)
	}
}

func writeFile(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func writeSums(t *testing.T, dir string, sums map[string]string) {
	t.Helper()
	var b strings.Builder
	for name, sum := range sums {
		b.WriteString(sum)
		b.WriteString("  ")
		b.WriteString(name)
		b.WriteString("\n")
	}
	writeFile(t, dir, sumsFile, []byte(b.String()))
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
