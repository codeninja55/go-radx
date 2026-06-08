//go:build unix

package server

import (
	"context"
	"fmt"
	"iter"
	"net"
	"os"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
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
