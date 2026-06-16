//go:build interop

package rest_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/fhir/rest/integration/hapi"
)

// TestMain provisions the HAPI FHIR server container for the interop gate. When
// RADX_FHIR_HAPI_BASE already names an external server the container is not started, preserving
// the bring-your-own-server escape hatch. A container start failure fails the run (exit 1) rather
// than skipping: under the interop tag this is a gate, and a silent skip would manufacture green.
func TestMain(m *testing.M) {
	os.Exit(runInteropMain(m))
}

func runInteropMain(m *testing.M) int {
	if os.Getenv("RADX_FHIR_HAPI_BASE") != "" {
		return m.Run()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	server, err := hapi.Start(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "interop: start HAPI FHIR container: %v\n", err)
		return 1
	}
	defer func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Minute)
		defer stopCancel()
		if err := server.Stop(stopCtx); err != nil {
			fmt.Fprintf(os.Stderr, "interop: stop HAPI FHIR container: %v\n", err)
		}
	}()

	// TestMain has no *testing.T, so t.Setenv is unavailable; the variable feeds the existing
	// env-gated TestInteropHAPIServer in this same process only.
	if err := os.Setenv("RADX_FHIR_HAPI_BASE", server.BaseURL()); err != nil {
		fmt.Fprintf(os.Stderr, "interop: set RADX_FHIR_HAPI_BASE: %v\n", err)
		return 1
	}
	return m.Run()
}
