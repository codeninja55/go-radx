package r5_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// benchBundleEntries is the entry count of the large searchset Bundle the marshal/unmarshal
// benchmarks walk. It is sized so the bundle is comfortably larger than any single workflow
// resource — a realistic worklist page — without making the benchmark dominated by setup.
const benchBundleEntries = 200

// loadCorpusResource decodes one committed workflow instance through the production decode
// path (fhir.UnmarshalResource), so the benchmarks measure the real decoder over the same
// synthetic, PHI-free corpus the conformance gate validates. It fails the benchmark rather
// than returning an error so a corpus or decode regression surfaces as a benchmark failure.
func loadCorpusResource(b *testing.B, name string) fhir.Resource {
	b.Helper()
	raw, err := os.ReadFile(filepath.Join(corpusSeedDir, name+".json"))
	if err != nil {
		b.Fatalf("read corpus %s: %v", name, err)
	}
	resource, err := fhir.UnmarshalResource(raw)
	if err != nil {
		b.Fatalf("decode corpus %s: %v", name, err)
	}
	return resource
}

// largeSearchSet builds a searchset Bundle of benchBundleEntries entries, each a decoded
// corpus Patient under a unique fullUrl. It is the large-payload subject for the
// marshal/unmarshal benchmarks: a searchset is the bundle type a worklist or query returns,
// and the per-entry polymorphic resource decode is the path the FHIR decoder must handle at
// scale. The Patient carries only synthetic data.
func largeSearchSet(b *testing.B) *r5.Bundle {
	b.Helper()
	patient := loadCorpusResource(b, "Patient")
	entries := make([]r5.SearchEntry, 0, benchBundleEntries)
	for i := range benchBundleEntries {
		entries = append(entries, r5.SearchEntry{
			FullURL:  fmt.Sprintf("http://example.org/go-radx/Patient/wf-%d", i),
			Resource: patient,
		})
	}
	bundle, err := r5.NewSearchSet(int32(benchBundleEntries), entries...)
	if err != nil {
		b.Fatalf("build searchset bundle: %v", err)
	}
	return bundle
}

// BenchmarkMarshalSearchSetBundle measures encoding a large searchset Bundle to FHIR JSON,
// the per-entry polymorphic resource marshal at scale. The bundle is built once outside the
// timed loop; only json.Marshal (through the generated MarshalJSON) is timed, with allocs
// reported so the serialization allocation profile (PRD §9.3) is tracked against the
// committed baseline.
func BenchmarkMarshalSearchSetBundle(b *testing.B) {
	bundle := largeSearchSet(b)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := fhir.MarshalSummary(bundle, fhir.SummaryFull); err != nil {
			b.Fatalf("marshal: %v", err)
		}
	}
}

// BenchmarkUnmarshalSearchSetBundle measures decoding a large searchset Bundle from FHIR
// JSON, the per-entry polymorphic resource decode (resourceType peek then registry dispatch)
// at scale. The encoded bytes are produced once outside the timed loop; SetBytes reports the
// throughput and ReportAllocs the decode allocation profile.
func BenchmarkUnmarshalSearchSetBundle(b *testing.B) {
	bundle := largeSearchSet(b)
	encoded, err := fhir.MarshalSummary(bundle, fhir.SummaryFull)
	if err != nil {
		b.Fatalf("encode bundle: %v", err)
	}
	b.SetBytes(int64(len(encoded)))
	b.ReportAllocs()
	for b.Loop() {
		if _, err := fhir.UnmarshalResource(encoded); err != nil {
			b.Fatalf("unmarshal: %v", err)
		}
	}
}

// BenchmarkValidateWorkflowSet measures the in-process structural gate over the full
// radiology + clinical workflow set: every committed corpus instance decoded once, then
// Validate run over each per iteration. It is the validation hot path the conformance gate
// backstops; ReportAllocs tracks that the data-driven descriptors keep validation
// allocation-light (PRD §9 — no per-call StructureDefinition lookup, no reflection).
func BenchmarkValidateWorkflowSet(b *testing.B) {
	resources := make([]fhir.Resource, 0, len(corpusWorkflowSet))
	for _, name := range corpusWorkflowSet {
		resources = append(resources, loadCorpusResource(b, name))
	}
	b.ReportAllocs()
	for b.Loop() {
		for _, r := range resources {
			_ = fhir.Validate(r)
		}
	}
}

// BenchmarkMarshalSummary measures the _summary serialization views over a decoded corpus
// Patient: SummaryFull is the identity marshal, while SummaryTrue, SummaryText, SummaryData,
// and SummaryCount each marshal then filter the encoded object key-by-key. Benchmarking the
// modes side by side surfaces the filtering overhead each adds over the plain marshal.
func BenchmarkMarshalSummary(b *testing.B) {
	patient := loadCorpusResource(b, "Patient")
	modes := []struct {
		name string
		mode fhir.SummaryMode
	}{
		{"full", fhir.SummaryFull},
		{"true", fhir.SummaryTrue},
		{"text", fhir.SummaryText},
		{"data", fhir.SummaryData},
		{"count", fhir.SummaryCount},
	}
	for _, m := range modes {
		b.Run(m.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := fhir.MarshalSummary(patient, m.mode); err != nil {
					b.Fatalf("MarshalSummary(%s): %v", m.name, err)
				}
			}
		})
	}
}
