package dicom

import (
	"bytes"
	"io"
	"testing"
)

// benchDataSet loads the vendored MR fixture once and returns its main dataset and transfer syntax.
// MR2_UNCI.dcm is a ~2 MB uncompressed Explicit VR Little Endian instance — a representative dataset
// for exercising the Part 10 codec hot path (PRD §9.3 minimise allocations in hot paths).
func benchDataSet(b *testing.B) (*DataSet, TransferSyntax) {
	b.Helper()
	f, err := ReadFile("../testdata/dicom/MR2_UNCI.dcm")
	if err != nil {
		b.Fatalf("read fixture: %v", err)
	}
	return f.DataSet, f.Meta.TransferSyntaxUID
}

// BenchmarkEncodeDataSet measures the bare-dataset encode hot path: serialising a real dataset in
// its transfer syntax. SetBytes reports throughput; ReportAllocs surfaces per-op allocations.
func BenchmarkEncodeDataSet(b *testing.B) {
	ds, ts := benchDataSet(b)
	var sized bytes.Buffer
	if err := EncodeDataSet(&sized, ds, ts); err != nil {
		b.Fatalf("size encode: %v", err)
	}
	b.SetBytes(int64(sized.Len()))
	b.ReportAllocs()
	for b.Loop() {
		if err := EncodeDataSet(io.Discard, ds, ts); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecodeDataSet measures the bare-dataset decode hot path: parsing a real dataset off the
// wire in its transfer syntax.
func BenchmarkDecodeDataSet(b *testing.B) {
	ds, ts := benchDataSet(b)
	var buf bytes.Buffer
	if err := EncodeDataSet(&buf, ds, ts); err != nil {
		b.Fatalf("pre-encode: %v", err)
	}
	encoded := buf.Bytes()
	b.SetBytes(int64(len(encoded)))
	b.ReportAllocs()
	for b.Loop() {
		if _, err := DecodeDataSet(bytes.NewReader(encoded), ts); err != nil {
			b.Fatal(err)
		}
	}
}
