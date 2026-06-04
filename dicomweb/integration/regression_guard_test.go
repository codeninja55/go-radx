//go:build interop

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicomweb"
)

// regressionGuardEnv gates the negative-control test below. The guard proves the
// DICOMweb interop net actually bites — that a broken STOW-RS path fails the
// assertions rather than passing silently — without leaving a deliberately-failing
// test in the CI matrix. The interop matrix does not set this variable, so the guard
// is skipped there; run it on demand with RADX_INTEROP_REGRESSION_GUARD=1.
const regressionGuardEnv = "RADX_INTEROP_REGRESSION_GUARD"

// TestInteropGuardBrokenDICOMWebPathFails is the negative control for the DICOMweb
// interop gate. It starts the same real Orthanc origin the positive test uses, but
// points the client at a base URL whose DICOMweb root does not exist on the server,
// then asserts the STOW-RS store fails. A passing store here would mean the gate
// could go green against an origin that never accepted the instance — exactly the
// regression this net exists to catch. The test is normally skipped so CI stays
// green; setting RADX_INTEROP_REGRESSION_GUARD=1 runs it to confirm the gate bites.
func TestInteropGuardBrokenDICOMWebPathFails(t *testing.T) {
	if os.Getenv(regressionGuardEnv) == "" {
		t.Skipf("negative control; set %s=1 to confirm the DICOMweb interop gate bites", regressionGuardEnv)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	orth := startOrthanc(ctx, t)
	ds, _, _, _ := readFixture(t)

	// The Orthanc DICOMweb plugin serves under /dicom-web; this root is wrong, so the
	// STOW-RS POST to /studies lands on a path the origin does not route and the store
	// must fail. DICOMWebBaseURL() (the correct root) is what the positive test uses.
	brokenBase := orth.HTTPBaseURL() + "/dicom-web-broken"
	client, err := dicomweb.NewClient(brokenBase)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.Store(ctx, ds)
	if err == nil && resp.IsComplete() {
		t.Fatalf("STOW-RS to a broken DICOMweb root reported a complete store; the interop gate would not catch this regression")
	}
}
