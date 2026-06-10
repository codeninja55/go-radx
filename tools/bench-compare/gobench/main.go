// Command gobench is the go-radx side of the PRD section 11.3 comparative benchmark
// harness for the areas the committed `go test -bench` suites do not already cover with
// a fixture-matched benchmark: loopback DIMSE C-STORE throughput, HL7 v2 parse
// throughput, and FHIR R5 Bundle unmarshal/marshal/validate over the shared file
// fixture. It emits raw wall-clock samples as JSON on stdout; the Python orchestrator
// (tools/bench-compare/runner.py) normalizes both sides with the same arithmetic.
//
// Loopback only: the DIMSE area binds 127.0.0.1 and never reaches the network.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
	"github.com/codeninja55/go-radx/hl7v2"
)

// result is one benchmark series. samples_s holds wall-clock seconds per sample, each
// sample timing `ops` back-to-back operations; the orchestrator derives ns/op, ops/sec,
// and MB/s from the median sample. The schema is shared verbatim with the Python runners.
type result struct {
	Area        string    `json:"area"`
	Name        string    `json:"name"`
	Fixture     string    `json:"fixture"`
	Side        string    `json:"side"`
	Library     string    `json:"library"`
	Status      string    `json:"status"`
	Note        string    `json:"note,omitempty"`
	Ops         int       `json:"ops"`
	BytesPerOp  float64   `json:"bytes_per_op"`
	Samples     []float64 `json:"samples_s"`
	AllocsPerOp float64   `json:"allocs_per_op,omitempty"`
}

const (
	sideGo    = "go-radx"
	statusOK  = "ok"
	statusErr = "error"
)

func main() {
	var (
		area        = flag.String("area", "", "benchmark area: dimse | hl7 | fhir")
		testdata    = flag.String("testdata", "../../../testdata", "path to the repo testdata directory")
		repeats     = flag.Int("repeats", 5, "measured samples per benchmark (median is published)")
		warmup      = flag.Int("warmup", 1, "unmeasured warmup samples per benchmark")
		iters       = flag.Int("iters", 200, "operations per sample for the hl7 and fhir areas")
		smallCount  = flag.Int("small-count", 200, "small instances per C-STORE sample")
		mediumCount = flag.Int("medium-count", 20, "medium instances per C-STORE sample")
	)
	flag.Parse()

	var (
		results []result
		err     error
	)
	switch *area {
	case "dimse":
		results, err = benchDIMSE(*testdata, *repeats, *warmup, *smallCount, *mediumCount)
	case "hl7":
		results, err = benchHL7(*testdata, *repeats, *warmup, *iters)
	case "fhir":
		results, err = benchFHIR(*testdata, *repeats, *warmup, *iters)
	default:
		err = fmt.Errorf("unknown -area %q (want dimse, hl7, or fhir)", *area)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobench: %v\n", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(results); err != nil {
		fmt.Fprintf(os.Stderr, "gobench: encode results: %v\n", err)
		os.Exit(1)
	}
}

// measure times `ops` calls of fn per sample, `repeats` times after `warmup` unmeasured
// samples, and reports wall-clock seconds per sample plus a mallocs-per-op estimate from
// the runtime.MemStats delta of the last sample (single-goroutine areas only; the DIMSE
// area passes allocTrack=false because the SCP goroutines would pollute the delta).
func measure(fn func() error, ops, repeats, warmup int, allocTrack bool) ([]float64, float64, error) {
	for range warmup {
		for range ops {
			if err := fn(); err != nil {
				return nil, 0, err
			}
		}
	}
	samples := make([]float64, 0, repeats)
	var allocsPerOp float64
	for s := range repeats {
		var before runtime.MemStats
		if allocTrack && s == repeats-1 {
			runtime.ReadMemStats(&before)
		}
		start := time.Now()
		for range ops {
			if err := fn(); err != nil {
				return nil, 0, err
			}
		}
		samples = append(samples, time.Since(start).Seconds())
		if allocTrack && s == repeats-1 {
			var after runtime.MemStats
			runtime.ReadMemStats(&after)
			allocsPerOp = float64(after.Mallocs-before.Mallocs) / float64(ops)
		}
	}
	return samples, allocsPerOp, nil
}

// storeSink is the benchmark C-STORE SCP handler: the dataset is already fully decoded by
// the dispatcher when Store is called, so touching the SOP Instance UID proves receipt
// without adding I/O. The Python side mirrors this by reading event.dataset (which forces
// the pynetdicom decode) and returning success without persisting. Neither side writes to
// disk; the measured path is protocol + (de)serialization only.
type storeSink struct{}

func (storeSink) Store(_ context.Context, ds *dicom.DataSet, _ dimse.OpInfo) dimse.Status {
	if _, ok := ds.GetString(dicom.NewTag(0x0008, 0x0018)); !ok {
		return dimse.StatusStoreSuccess // SOP Instance UID lookup is best-effort, never a refusal
	}
	return dimse.StatusStoreSuccess
}

// dimseFixtures pins the loopback C-STORE payloads: a small instance (Segmentation
// Storage, ~100 KiB uncompressed) and a medium one (MR Image Storage, ~2 MiB).
var dimseFixtures = struct{ small, medium string }{
	small:  "liver.dcm",
	medium: "MR2_UNCI.dcm",
}

func benchDIMSE(testdata string, repeats, warmup, smallCount, mediumCount int) ([]result, error) {
	smallPath := filepath.Join(testdata, "dicom", dimseFixtures.small)
	mediumPath := filepath.Join(testdata, "dicom", dimseFixtures.medium)
	small, err := dicom.ReadFile(smallPath)
	if err != nil {
		return nil, fmt.Errorf("read small fixture: %w", err)
	}
	medium, err := dicom.ReadFile(mediumPath)
	if err != nil {
		return nil, fmt.Errorf("read medium fixture: %w", err)
	}
	smallSize, mediumSize := fileSize(smallPath), fileSize(mediumPath)

	scpAE, err := dimse.NewAE(dimse.AETitle("BENCHSCP"))
	if err != nil {
		return nil, err
	}
	scuAE, err := dimse.NewAE(dimse.AETitle("BENCHSCU"))
	if err != nil {
		return nil, err
	}
	server := dimse.NewServer(scpAE, dimse.StorageContexts(), storeSink{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.ListenAndServe(ctx, "127.0.0.1:0") }()
	addr, err := waitForAddr(server, serveErr)
	if err != nil {
		return nil, err
	}
	defer func() {
		shCtx, shCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shCancel()
		_ = server.Shutdown(shCtx)
	}()

	ops := smallCount + mediumCount
	totalBytes := float64(smallCount)*float64(smallSize) + float64(mediumCount)*float64(mediumSize)

	// One sample = one association carrying smallCount + mediumCount sequential C-STOREs;
	// association setup and release are outside the timed window. The pynetdicom runner
	// does exactly the same.
	sample := func() (float64, error) {
		assoc, aerr := scuAE.Associate(ctx, addr, dimse.AETitle("BENCHSCP"), dimse.StorageContexts())
		if aerr != nil {
			return 0, fmt.Errorf("associate: %w", aerr)
		}
		defer func() {
			rCtx, rCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer rCancel()
			_ = assoc.Release(rCtx)
		}()
		start := time.Now()
		for i := 0; i < smallCount; i++ {
			if serr := storeOne(ctx, assoc, small.DataSet); serr != nil {
				return 0, serr
			}
		}
		for i := 0; i < mediumCount; i++ {
			if serr := storeOne(ctx, assoc, medium.DataSet); serr != nil {
				return 0, serr
			}
		}
		return time.Since(start).Seconds(), nil
	}

	for range warmup {
		if _, err := sample(); err != nil {
			return nil, err
		}
	}
	samples := make([]float64, 0, repeats)
	for range repeats {
		s, err := sample()
		if err != nil {
			return nil, err
		}
		samples = append(samples, s)
	}

	return []result{{
		Area:       "dimse",
		Name:       "dimse-cstore-loopback",
		Fixture:    fmt.Sprintf("%dx %s + %dx %s", smallCount, dimseFixtures.small, mediumCount, dimseFixtures.medium),
		Side:       sideGo,
		Library:    "go-radx dimse",
		Status:     statusOK,
		Note:       "same-stack SCU->SCP over loopback; association setup excluded; no persistence",
		Ops:        ops,
		BytesPerOp: totalBytes / float64(ops),
		Samples:    samples,
	}}, nil
}

func storeOne(ctx context.Context, assoc *dimse.Association, ds *dicom.DataSet) error {
	status, err := assoc.Store(ctx, ds)
	if err != nil {
		return fmt.Errorf("store: %w", err)
	}
	if !status.IsSuccess() {
		return fmt.Errorf("store status 0x%04X", status.Code)
	}
	return nil
}

// waitForAddr polls the server for its bound loopback address, surfacing an early
// ListenAndServe failure instead of spinning forever.
func waitForAddr(server *dimse.Server, serveErr <-chan error) (string, error) {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-serveErr:
			return "", fmt.Errorf("scp exited before bind: %w", err)
		default:
		}
		if addr := server.Addr(); addr != nil {
			return addr.String(), nil
		}
		time.Sleep(5 * time.Millisecond)
	}
	return "", errors.New("scp did not bind within 5s")
}

// hl7Fixtures are the corpus messages both sides parse: an admission and a result, the
// two message types the PRD names for the HL7 comparison.
var hl7Fixtures = []string{"adt-a01.hl7", "oru-r01.hl7"}

func benchHL7(testdata string, repeats, warmup, iters int) ([]result, error) {
	results := make([]result, 0, len(hl7Fixtures))
	for _, name := range hl7Fixtures {
		raw, err := os.ReadFile(filepath.Join(testdata, "hl7v2", name))
		if err != nil {
			return nil, err
		}
		samples, allocs, err := measure(func() error {
			_, perr := hl7v2.Parse(raw)
			return perr
		}, iters, repeats, warmup, true)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		results = append(results, result{
			Area:        "hl7v2",
			Name:        "hl7v2-parse",
			Fixture:     name,
			Side:        sideGo,
			Library:     "go-radx hl7v2",
			Status:      statusOK,
			Ops:         iters,
			BytesPerOp:  float64(len(raw)),
			Samples:     samples,
			AllocsPerOp: allocs,
		})
	}
	return results, nil
}

func benchFHIR(testdata string, repeats, warmup, iters int) ([]result, error) {
	const fixture = "Bundle.json"
	raw, err := os.ReadFile(filepath.Join(testdata, "fhir", "r5", fixture))
	if err != nil {
		return nil, err
	}
	resource, err := r5.UnmarshalResource(raw)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", fixture, err)
	}

	mk := func(name, note string, fn func() error) (result, error) {
		samples, allocs, merr := measure(fn, iters, repeats, warmup, true)
		if merr != nil {
			return result{}, fmt.Errorf("%s: %w", name, merr)
		}
		return result{
			Area:        "fhir",
			Name:        name,
			Fixture:     fixture,
			Side:        sideGo,
			Library:     "go-radx fhir/r5",
			Status:      statusOK,
			Note:        note,
			Ops:         iters,
			BytesPerOp:  float64(len(raw)),
			Samples:     samples,
			AllocsPerOp: allocs,
		}, nil
	}

	unmarshal, err := mk("fhir-bundle-unmarshal", "", func() error {
		_, uerr := r5.UnmarshalResource(raw)
		return uerr
	})
	if err != nil {
		return nil, err
	}
	marshal, err := mk("fhir-bundle-marshal", "", func() error {
		_, merr := r5.MarshalSummary(resource, fhir.SummaryFull)
		return merr
	})
	if err != nil {
		return nil, err
	}
	validate, err := mk("fhir-bundle-validate", "structural+binding validation of the decoded Bundle", func() error {
		_ = r5.Validate(resource)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return []result{unmarshal, marshal, validate}, nil
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
