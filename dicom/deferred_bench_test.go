package dicom

import (
	"path/filepath"
	"testing"
)

// BenchmarkReadFileLargeElement shows the deferred read's bounded allocation: with
// WithDeferredValues the large pixel value is skipped (B/op stays near the small
// metadata elements), while the default read materialises all of it.
func BenchmarkReadFileLargeElement(b *testing.B) {
	const n = 8 << 20 // 8 MiB pixel value
	path := filepath.Join(b.TempDir(), "bench.dcm")
	if err := deferredFixtureDataSet(n).WriteFile(path, ExplicitVRLittleEndian); err != nil {
		b.Fatalf("write fixture: %v", err)
	}

	b.Run("materialised", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := ReadFile(path); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("deferred", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := ReadFile(path, WithDeferredValues(1<<20)); err != nil {
				b.Fatal(err)
			}
		}
	})
}
