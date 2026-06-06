package gen

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// vendoredR5Dir and vendoredR4Dir are the committed, checksum-pinned definition
// bundles the generator reads, one per release. The paths are relative to this
// package directory.
const (
	vendoredR5Dir = "testdata/definitions/r5"
	vendoredR4Dir = "testdata/definitions/r4"
)

// requiredBundleFiles are the bundle files every release's generation depends on.
// The set is the same across R4 and R5.
var requiredBundleFiles = []string{
	"profiles-types.json",
	"profiles-resources.json",
	"valuesets.json",
}

// vendoredBundles names each release's committed bundle directory, so the presence
// and checksum guards run over both R4 and R5 from one table.
var vendoredBundles = []struct {
	release string
	dir     string
}{
	{release: "r4", dir: vendoredR4Dir},
	{release: "r5", dir: vendoredR5Dir},
}

// TestVendoredBundlesPresent guards that each release's vendored bundle and its
// checksum manifest are committed, before the loader consumes them. It is
// deliberately loader-independent: it re-derives the SHA-256 here rather than
// calling generator code, so the pin is verified by an outside witness.
func TestVendoredBundlesPresent(t *testing.T) {
	t.Parallel()

	for _, b := range vendoredBundles {
		t.Run(b.release, func(t *testing.T) {
			for _, name := range requiredBundleFiles {
				if _, err := os.Stat(filepath.Join(b.dir, name)); err != nil {
					t.Errorf("required bundle file missing: %v", err)
				}
			}
			if _, err := os.Stat(filepath.Join(b.dir, "SHA256SUMS")); err != nil {
				t.Fatalf("SHA256SUMS missing: %v", err)
			}
			if _, err := os.Stat(filepath.Join(b.dir, "SOURCE.md")); err != nil {
				t.Errorf("SOURCE.md provenance missing: %v", err)
			}
		})
	}
}

// TestVendoredBundlesChecksumsMatch re-computes the SHA-256 of each on-disk file and
// asserts it equals the value recorded in SHA256SUMS, so the pin is real before the
// fail-closed loader relies on it. Every required file must appear in the manifest.
func TestVendoredBundlesChecksumsMatch(t *testing.T) {
	t.Parallel()

	for _, b := range vendoredBundles {
		t.Run(b.release, func(t *testing.T) {
			sums := readSHA256SUMS(t, filepath.Join(b.dir, "SHA256SUMS"))

			for _, name := range requiredBundleFiles {
				want, ok := sums[name]
				if !ok {
					t.Errorf("SHA256SUMS has no entry for %s", name)
					continue
				}
				got := sha256HexFile(t, filepath.Join(b.dir, name))
				if got != want {
					t.Errorf("%s checksum mismatch: on-disk %s, recorded %s", name, got, want)
				}
			}
		})
	}
}

// readSHA256SUMS parses a shasum-format manifest ("<hex>  <filename>" per line)
// into a filename→hex map.
func readSHA256SUMS(t *testing.T, path string) map[string]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open SHA256SUMS: %v", err)
	}
	defer func() { _ = f.Close() }()

	sums := make(map[string]string)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			t.Fatalf("malformed SHA256SUMS line: %q", line)
		}
		sums[fields[1]] = fields[0]
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan SHA256SUMS: %v", err)
	}
	return sums
}

// sha256HexFile returns the lowercase hex SHA-256 of a file's bytes.
func sha256HexFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
