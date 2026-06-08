//go:build unix

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/internal/phisweep"
	"github.com/codeninja55/go-radx/logging"
)

// PHI sentinels: synthetic, never real, yet shaped like the patient values they stand in for, so a
// leak through a value-formatting path is caught the same way a real value would be.
const (
	phiPatientName = "SENTINEL^PHI^DONOTLOG"
	phiPatientID   = "ZZZTEST-MRN-PHI-SENTINEL"
	phiAccession   = "ZZZTEST-ACC-PHI-SENTINEL"
)

// TestPHIDefaultSweep drives a runnable daemon (default file store + SQLite catalogue + DIMSE SCP)
// over a sentinel-bearing object at default verbosity through a real C-STORE, then asserts no PHI
// sentinel surfaces in stdout, stderr, returned errors, or the structured log. It mirrors the
// internal/phisweep ethos and reuses its capture harness so the server's no-PHI default is
// test-enforced, not convention-enforced (PRD §11.2).
func TestPHIDefaultSweep(t *testing.T) {
	sentinels := []string{phiPatientName, phiPatientID, phiAccession}

	capture, err := phisweep.Run(func(ctx context.Context) []error {
		return exerciseDaemonStore(ctx)
	})
	if err != nil {
		t.Fatalf("phisweep.Run: %v", err)
	}

	if leaks := phisweep.Scan(capture, sentinels); len(leaks) > 0 {
		for _, l := range leaks {
			t.Errorf("PHI leak: %s", l)
		}
		t.Fatalf("%d PHI sentinel(s) surfaced at default verbosity", len(leaks))
	}
}

// TestFHIRRolePHIDefaultSweep drives the FHIR REST role over a sentinel-bearing Patient at default
// verbosity (a create whose Patient.name carries the PHI sentinel, then a read and a search), and
// asserts no sentinel surfaces in stdout, stderr, returned errors, or the structured log. It is the
// FHIR-role counterpart of the DIMSE sweep, proving the role logs structure and identifiers only —
// resource type, id, and interaction name — never the patient values in the resource body (PRD
// §9.1, §11.2).
func TestFHIRRolePHIDefaultSweep(t *testing.T) {
	sentinels := []string{phiPatientName, phiPatientID}

	capture, err := phisweep.Run(func(ctx context.Context) []error {
		return exerciseFHIRRole(ctx)
	})
	if err != nil {
		t.Fatalf("phisweep.Run: %v", err)
	}

	if leaks := phisweep.Scan(capture, sentinels); len(leaks) > 0 {
		for _, l := range leaks {
			t.Errorf("PHI leak: %s", l)
		}
		t.Fatalf("%d PHI sentinel(s) surfaced at default verbosity", len(leaks))
	}
}

// exerciseFHIRRole runs a daemon with the FHIR role at default verbosity, creates a sentinel-bearing
// Patient over real HTTP, reads it back, and searches for it, returning every error it produced so
// the harness scans their strings for sentinels. The Patient carries the PHI sentinels in its name
// and identifier so a careless logging or error path that formatted the body would surface one.
func exerciseFHIRRole(ctx context.Context) []error {
	logger := logging.FromContext(ctx)
	var errs []error

	repo, err := NewMemoryRepository(fhir.R5)
	if err != nil {
		return []error{err}
	}
	role, err := NewFHIRRole(repo, WithFHIRPort(0))
	if err != nil {
		return []error{err}
	}
	d, err := New(WithLogger(logger), WithFHIR(role))
	if err != nil {
		return []error{err}
	}

	runCtx, cancelRun := context.WithCancel(ctx)
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()

	addr := waitDaemonAddr(d, "fhir")
	if addr == nil {
		cancelRun()
		<-runErr
		return []error{fmt.Errorf("daemon did not bind fhir role")}
	}
	base := "http://" + addr.String() + "/fhir"

	opCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// A Patient whose name and identifier carry the PHI sentinels.
	patient := []byte(fmt.Sprintf(
		`{"resourceType":"Patient","identifier":[{"value":%q}],"name":[{"family":%q}],"gender":"female"}`,
		phiPatientID, phiPatientName))

	id, cerr := fhirCreate(opCtx, base, patient)
	if cerr != nil {
		errs = append(errs, cerr)
	} else {
		if rerr := fhirGet(opCtx, base+"/Patient/"+id); rerr != nil {
			errs = append(errs, rerr)
		}
		if serr := fhirGet(opCtx, base+"/Patient?_id="+id); serr != nil {
			errs = append(errs, serr)
		}
	}

	cancelRun()
	if rerr := <-runErr; rerr != nil {
		errs = append(errs, rerr)
	}
	_ = logger.Sync()
	return errs
}

// fhirCreate POSTs a resource to the FHIR role and returns the created resource's id, so the sweep
// can read it back. A non-201 status is an error whose string is scanned for sentinels.
func fhirCreate(ctx context.Context, base string, body []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/Patient", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/fhir+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("fhir create status %d", resp.StatusCode)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(out, &created); err != nil {
		return "", err
	}
	return created.ID, nil
}

// fhirGet issues a GET against a FHIR role URL, returning a non-2xx as an error whose string is
// scanned for sentinels. The response body is discarded; a leak can only arise from a logged field
// or a returned error, not from the resource the client itself receives.
func fhirGet(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fhir get status %d", resp.StatusCode)
	}
	return nil
}

// exerciseDaemonStore runs a daemon with the context-injected logger at default (info) verbosity,
// drives a C-STORE of a sentinel-bearing object through the DIMSE role, queries it back, and shuts
// down. It returns every error it produced so the harness scans their strings for sentinels. It
// never prints to the process streams; a leak can only arise from a logged field or a returned error.
func exerciseDaemonStore(ctx context.Context) []error {
	logger := logging.FromContext(ctx)
	var errs []error

	tmp, err := os.MkdirTemp("", "radx-phisweep-*")
	if err != nil {
		return []error{err}
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	store, err := FileStore(tmp)
	if err != nil {
		return []error{err}
	}
	cat, err := SQLiteCatalogue(ctx, ":memory:")
	if err != nil {
		return []error{err}
	}

	aet, err := dimse.ParseAETitle("RADX-SCP")
	if err != nil {
		return []error{err}
	}
	dimseRole, err := NewDIMSERole(aet, store, cat, WithDIMSEPort(0))
	if err != nil {
		return []error{err}
	}
	d, err := New(WithLogger(logger), WithDIMSE(dimseRole))
	if err != nil {
		return []error{err}
	}

	runCtx, cancelRun := context.WithCancel(ctx)
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()

	addr := waitDaemonAddr(d, "dimse")
	if addr == nil {
		cancelRun()
		<-runErr
		return []error{fmt.Errorf("daemon did not bind dimse role")}
	}

	opCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	scu, err := dimse.NewAE(dimse.AETitle("RADX-SCU"))
	if err != nil {
		errs = append(errs, err)
	} else if assoc, aerr := scu.Associate(opCtx, addr.String(), aet, echoStorageContexts()); aerr != nil {
		errs = append(errs, aerr)
	} else {
		ds := sentinelObject()
		if _, serr := assoc.Store(opCtx, ds); serr != nil {
			errs = append(errs, serr)
		}
		_ = assoc.Release(opCtx)

		// Query the catalogue back over the sentinel PatientID so a value-leaking query path would
		// surface the sentinel through a logged field or a returned error.
		errs = append(errs, qErrors(cat.Query(opCtx, CatalogueQuery{
			Level: dimse.QueryLevelStudy,
			Match: map[dicom.Tag]string{dicom.TagPatientID: phiPatientID},
		}))...)
	}

	cancelRun()
	if rerr := <-runErr; rerr != nil {
		errs = append(errs, rerr)
	}
	_ = logger.Sync()
	return errs
}

// sentinelObject builds a valid object whose dataset carries the synthetic PHI sentinels in their
// canonical tags, so a careless logging or error path that formats a value would surface a sentinel.
func sentinelObject() *dicom.DataSet {
	ds := newTestObject("1.2.840.113619.2.99.1", "1.2.840.113619.2.99.2", "1.2.840.113619.2.99.3")
	ds.SetString(dicom.TagPatientName, phiPatientName)
	ds.SetString(dicom.TagPatientID, phiPatientID)
	ds.SetString(dicom.TagAccessionNumber, phiAccession)
	return ds
}

// qErrors drains a Catalogue.Query iterator, returning only the errors it yielded so the harness can
// scan their strings for sentinels.
func qErrors(seq iter.Seq2[*dicom.DataSet, error]) []error {
	var out []error
	for _, err := range seq {
		if err != nil {
			out = append(out, err)
		}
	}
	return out
}

// waitDaemonAddr polls the daemon for a bound address for role, returning nil after a bounded wait.
func waitDaemonAddr(d *Daemon, role string) net.Addr {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if addrs := d.Addrs(); addrs != nil {
			if a, ok := addrs[role]; ok {
				return a
			}
		}
		time.Sleep(time.Millisecond)
	}
	return nil
}
