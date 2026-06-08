//go:build interop

package rest_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r4"
	"github.com/codeninja55/go-radx/fhir/rest"
)

// TestInteropHAPIServer drives the go-radx FHIR REST client against an EXTERNAL reference FHIR server
// (a HAPI FHIR server) and asserts a CRUD round-trip and a capability negotiation succeed against an
// independent implementation. It is the cross-implementation conformance check: it proves go-radx's
// request shape is accepted, and a foreign CapabilityStatement and resource are parsed, by a server
// go-radx did not write.
//
// There is no HAPI FHIR container in this harness, so the test SKIPs unless RADX_FHIR_HAPI_BASE names
// a reachable FHIR service base URL of such a server (for example
// http://localhost:8080/fhir for the public HAPI test server). This is a deliberate skip, not a
// silent pass: the client<->httptest round-trip in client_test.go and the client<->go-radx-role
// round-trip in the server package remain the hard correctness gates (they need no external server),
// and this test only adds confidence against a foreign implementation when one is supplied. The skip
// avoids depending on a fragile container or the public test server's availability in CI.
//
// The reference HAPI test server runs R4, so this test uses an R4 client. Set RADX_FHIR_HAPI_BASE to
// an R5 server's base to exercise R5.
func TestInteropHAPIServer(t *testing.T) {
	base := os.Getenv("RADX_FHIR_HAPI_BASE")
	if base == "" {
		t.Skip("RADX_FHIR_HAPI_BASE not set: no external HAPI FHIR server is provisioned in this " +
			"harness; the client<->httptest and client<->go-radx-role round-trips are the hard interop gates")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c, err := rest.NewClient(fhir.R4, base)
	if err != nil {
		t.Fatalf("NewClient(%q): %v", base, err)
	}

	// Capability negotiation: the server advertises a CapabilityStatement the client parses.
	caps, err := c.Capabilities(ctx)
	if err != nil {
		t.Fatalf("Capabilities: %v", err)
	}
	if caps.FHIRVersion == "" {
		t.Error("Capabilities: server reported no fhirVersion")
	}

	// CRUD round-trip: create a Patient, read it back, search for it by _id.
	gender := r4.AdministrativeGender("female")
	created, err := c.Create(ctx, &r4.Patient{Gender: &gender}, "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	id := resourceID(t, created.Resource)
	if id == "" {
		t.Fatal("Create: HAPI returned no id")
	}

	read, err := c.Read(ctx, "Patient", id)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if read.Resource.ResourceType() != "Patient" {
		t.Errorf("Read: resourceType = %q, want Patient", read.Resource.ResourceType())
	}

	page, err := c.Search(ctx, "Patient", rest.NewSearchParams().Set("_id", id))
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(page.Resources) == 0 {
		t.Error("Search by _id returned no matches")
	}
}
