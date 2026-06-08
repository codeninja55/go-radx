package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/codeninja55/go-radx/fhir"
)

// Transaction submits a transaction or batch Bundle to the server (POST to the service base) and
// returns the transaction-response (or batch-response) Bundle the server answers with. The bundle
// is a release Bundle built with the release's NewTransaction / NewBatch (so the bdl-* invariants
// hold); it is passed as the fhir.Resource interface, which every release Bundle satisfies, and the
// response is decoded into the client's release. A bundle whose resourceType is not "Bundle" is a
// usage error rather than a request the server rejects obscurely.
//
// Transaction semantics (an all-or-nothing apply, where a failed entry rolls back the whole bundle)
// are the server's; the client transmits the bundle and surfaces the response. A non-2xx status
// maps to a typed *OperationOutcomeError, and a 4xx whose body is an OperationOutcome carries the
// per-entry diagnostics through it. A 200 response whose body reports per-entry failures in the
// transaction-response Bundle is returned as a success to the caller, who inspects each entry's
// response.status: the HTTP status reflects the overall request, while a batch's per-entry outcomes
// live in the response Bundle (the caller distinguishes a transaction's all-or-nothing failure,
// which the server signals with a non-2xx, from a batch's per-entry failures, which it does not).
func (c *Client) Transaction(ctx context.Context, bundle fhir.Resource) (fhir.Resource, error) {
	if _, ok := fhir.As[fhir.Resource](bundle); !ok {
		return nil, fmt.Errorf("fhir/rest: Transaction requires a non-nil Bundle")
	}
	if bundle.ResourceType() != "Bundle" {
		return nil, fmt.Errorf("fhir/rest: Transaction requires a Bundle, got %s", bundle.ResourceType())
	}
	if err := c.checkTransactionRelease(bundle); err != nil {
		return nil, err
	}
	body, err := json.Marshal(bundle)
	if err != nil {
		return nil, fmt.Errorf("fhir/rest: encode transaction bundle: %w", err)
	}
	headers := map[string]string{"Content-Type": mediaTypeFHIRJSON}
	resp, err := c.doRequest(ctx, http.MethodPost, "", bytes.NewReader(body), headers)
	if err != nil {
		return nil, err
	}
	if resp.status != http.StatusOK {
		return nil, c.errorForResponse(http.MethodPost, "/", resp)
	}
	return c.decodeResource(resp)
}

// checkTransactionRelease verifies the bundle and every resource its entries carry belong to the
// client's fixed release before the bundle is marshalled and sent. The client is release-fixed, so a
// bundle of the other release — or one mixing in an out-of-release entry resource — would serialise
// the wrong release's shape to this client's endpoint; rejecting it client-side surfaces the
// mistake without a wrong-shape round trip. The bundle must be the client's release Bundle (so its
// entries can be read through the release-neutral view), and each entry resource must pass the same
// per-resource release check a direct create enforces, so the two write paths cannot diverge. The
// error names the release, never a patient value (PRD §9.1).
func (c *Client) checkTransactionRelease(bundle fhir.Resource) error {
	// Check the Bundle resource itself against the client's release first: a wrong-release Bundle with
	// only non-resource entries (e.g. an R5 Bundle of GET entries passed to an R4 client) would have an
	// empty resource view and otherwise slip through the per-entry loop below.
	if err := c.checkRelease(bundle); err != nil {
		return err
	}
	view, ok := asBundleView(bundle)
	if !ok {
		return fmt.Errorf("fhir/rest: transaction Bundle is not an %s Bundle: %w", c.release, ErrReleaseMismatch)
	}
	for _, r := range view.resources {
		if err := c.checkRelease(r); err != nil {
			return err
		}
	}
	return nil
}
