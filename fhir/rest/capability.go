package rest

import (
	"context"
	"fmt"
	"net/http"

	"github.com/codeninja55/go-radx/fhir"
)

// Capability is the parsed, release-neutral summary of a server's CapabilityStatement that the
// client uses to negotiate what it may attempt. It is built from the GET [base]/metadata response
// and exposes the few facts a client acts on: the FHIR version the server reports, the system-level
// interactions (transaction, batch, search-system), and the per-resource interactions
// (read, vread, create, update, delete, search-type, ...). The full CapabilityStatement is kept on
// Statement for a caller that needs more than this summary.
type Capability struct {
	// Statement is the concrete release CapabilityStatement (behind the fhir.Resource interface),
	// for a caller that wants the full document; narrow it with fhir.As.
	Statement fhir.Resource

	// FHIRVersion is the fhirVersion the server reported (for example "5.0.0"), or "" when absent.
	FHIRVersion string

	// systemInteractions is the set of system-level interaction codes the server advertises
	// (transaction, batch, search-system, history-system).
	systemInteractions map[string]struct{}

	// resourceInteractions maps a resource type to the set of type/instance interaction codes the
	// server advertises for it (read, vread, create, update, patch, delete, search-type, ...).
	resourceInteractions map[string]map[string]struct{}
}

// SupportsSystemInteraction reports whether the server advertises a system-level interaction
// (use the FHIR codes "transaction", "batch", "search-system", "history-system").
func (cs *Capability) SupportsSystemInteraction(code string) bool {
	if cs == nil {
		return false
	}
	_, ok := cs.systemInteractions[code]
	return ok
}

// SupportsResourceInteraction reports whether the server advertises an interaction for a resource
// type (use the FHIR codes "read", "vread", "create", "update", "patch", "delete", "search-type",
// "history-instance", "history-type"). A resource type the statement does not list returns false,
// so a client can refuse to attempt an unsupported interaction rather than relying on a runtime
// 405/501.
func (cs *Capability) SupportsResourceInteraction(resourceType, code string) bool {
	if cs == nil {
		return false
	}
	codes, ok := cs.resourceInteractions[resourceType]
	if !ok {
		return false
	}
	_, ok = codes[code]
	return ok
}

// SupportsTransaction reports whether the server advertises the transaction interaction, the
// common pre-flight check before submitting a transaction Bundle.
func (cs *Capability) SupportsTransaction() bool {
	return cs.SupportsSystemInteraction("transaction")
}

// Capabilities fetches and parses the server's CapabilityStatement (GET [base]/metadata), so a
// client can negotiate what it may attempt before issuing an interaction the server would reject.
// The statement is decoded into the client's release; a server that answers metadata with a
// non-CapabilityStatement resource is reported, not silently treated as no capabilities. A non-2xx
// status maps to a typed error.
func (c *Client) Capabilities(ctx context.Context) (*Capability, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "metadata", nil, nil)
	if err != nil {
		return nil, err
	}
	if resp.status != http.StatusOK {
		return nil, c.errorForResponse(http.MethodGet, "metadata", resp)
	}
	r, err := c.decodeResource(resp)
	if err != nil {
		return nil, err
	}
	capability, ok := capabilityFromResource(r)
	if !ok {
		return nil, fmt.Errorf("fhir/rest: metadata response is a %s, not a CapabilityStatement", r.ResourceType())
	}
	return capability, nil
}
