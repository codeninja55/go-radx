package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/codeninja55/go-radx/fhir"
)

// Result carries a resource together with the response metadata a caller needs for optimistic
// concurrency and addressing: the ETag (the resource version, used as an If-Match on a later
// update), the Location (the canonical URL a create returned), and the Last-Modified timestamp. A
// caller that only wants the resource ignores the rest.
type Result struct {
	// Resource is the concrete resource of the client's release (for example *r5.Patient), decoded
	// from the response body.
	Resource fhir.Resource

	// ETag is the resource version from the ETag header (a weak ETag's W/"..." form is preserved
	// verbatim), suitable to pass back as an If-Match on a version-aware update.
	ETag string

	// Location is the resource's canonical URL from the Location or Content-Location header, set on
	// a create or a version-aware update.
	Location string

	// LastModified is the Last-Modified header verbatim, or "" when the server omitted it.
	LastModified string
}

// Read returns the current version of one resource by type and id. A 404 maps to ErrNotFound
// (errors.Is-comparable). The returned resource is a concrete resource of the client's release.
func (c *Client) Read(ctx context.Context, resourceType, id string) (*Result, error) {
	if err := validateTypeID(resourceType, id); err != nil {
		return nil, err
	}
	path := resourceType + "/" + id
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp.status != http.StatusOK {
		return nil, c.errorForResponse(http.MethodGet, path, resp)
	}
	return c.resultFromResponse(resp)
}

// VRead returns a specific historical version of one resource by type, id, and versionId (the
// vread interaction, GET [type]/[id]/_history/[vid]). A 404 maps to ErrNotFound.
func (c *Client) VRead(ctx context.Context, resourceType, id, versionID string) (*Result, error) {
	if err := validateTypeID(resourceType, id); err != nil {
		return nil, err
	}
	if strings.TrimSpace(versionID) == "" {
		return nil, fmt.Errorf("fhir/rest: VRead requires a non-empty versionId")
	}
	path := resourceType + "/" + id + "/_history/" + versionID
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp.status != http.StatusOK {
		return nil, c.errorForResponse(http.MethodGet, path, resp)
	}
	return c.resultFromResponse(resp)
}

// Create stores a new resource (POST [type]). The resource's ResourceType determines the endpoint;
// a nil resource is a usage error. When ifNoneExist is non-empty it is sent as the If-None-Exist
// header, making this a conditional create: the server creates the resource only if no existing
// resource matches the given search query, and answers 200 with the existing match otherwise (the
// conditional-create idempotency rule). A non-2xx status maps to a typed error; a 412 conditional
// failure maps to ErrConflict. The created (or matched) resource is returned with its Location and
// ETag.
func (c *Client) Create(ctx context.Context, r fhir.Resource, ifNoneExist string) (*Result, error) {
	rt, body, err := c.encodeResource(r)
	if err != nil {
		return nil, err
	}
	headers := map[string]string{"Content-Type": mediaTypeFHIRJSON}
	if ifNoneExist != "" {
		headers["If-None-Exist"] = ifNoneExist
	}
	resp, err := c.doRequest(ctx, http.MethodPost, rt, bytes.NewReader(body), headers)
	if err != nil {
		return nil, err
	}
	// A create answers 201 Created; a conditional create that matched an existing resource answers
	// 200 OK. Both are success and both carry the stored resource.
	if resp.status != http.StatusCreated && resp.status != http.StatusOK {
		return nil, c.errorForResponse(http.MethodPost, rt, resp)
	}
	return c.resultFromResponse(resp)
}

// Update stores a new version of a resource at a known id (PUT [type]/[id]). The id is the
// resource's logical id; the resource's ResourceType determines the endpoint. When ifMatch is
// non-empty it is sent as the If-Match header for optimistic concurrency: the server applies the
// update only if the current version matches, and answers 412 Precondition Failed (mapped to
// ErrConflict) otherwise. An update may create the resource when the server allows update-as-create
// (answering 201); both 200 and 201 are success.
func (c *Client) Update(ctx context.Context, id string, r fhir.Resource, ifMatch string) (*Result, error) {
	rt, body, err := c.encodeResource(r)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("fhir/rest: Update requires a non-empty id")
	}
	path := rt + "/" + id
	headers := map[string]string{"Content-Type": mediaTypeFHIRJSON}
	if ifMatch != "" {
		headers["If-Match"] = ifMatch
	}
	resp, err := c.doRequest(ctx, http.MethodPut, path, bytes.NewReader(body), headers)
	if err != nil {
		return nil, err
	}
	if resp.status != http.StatusOK && resp.status != http.StatusCreated {
		return nil, c.errorForResponse(http.MethodPut, path, resp)
	}
	return c.resultFromResponse(resp)
}

// Patch applies a JSON Patch (RFC 6902) document to a resource (PATCH [type]/[id] with
// Content-Type application/json-patch+json). patch is the raw JSON Patch document. When ifMatch is
// non-empty it is sent as the If-Match header for optimistic concurrency. The patched resource is
// returned. A server that does not support patch answers 405/501, mapped to ErrUnsupported.
func (c *Client) Patch(ctx context.Context, resourceType, id string, patch []byte, ifMatch string) (*Result, error) {
	if err := validateTypeID(resourceType, id); err != nil {
		return nil, err
	}
	if len(patch) == 0 {
		return nil, fmt.Errorf("fhir/rest: Patch requires a non-empty patch document")
	}
	if !json.Valid(patch) {
		return nil, fmt.Errorf("fhir/rest: Patch document is not valid JSON")
	}
	path := resourceType + "/" + id
	headers := map[string]string{"Content-Type": mediaTypeJSONPatch}
	if ifMatch != "" {
		headers["If-Match"] = ifMatch
	}
	resp, err := c.doRequest(ctx, http.MethodPatch, path, bytes.NewReader(patch), headers)
	if err != nil {
		return nil, err
	}
	if resp.status != http.StatusOK {
		return nil, c.errorForResponse(http.MethodPatch, path, resp)
	}
	return c.resultFromResponse(resp)
}

// Delete removes a resource (DELETE [type]/[id]). A successful delete answers 200 or 204; both are
// reported as success with a nil error. A 404 maps to ErrNotFound so a caller can distinguish a
// no-op delete from a successful one, and a 409 (the server refusing to delete a referenced
// resource) maps to ErrConflict.
func (c *Client) Delete(ctx context.Context, resourceType, id string) error {
	if err := validateTypeID(resourceType, id); err != nil {
		return err
	}
	path := resourceType + "/" + id
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}
	if resp.status != http.StatusOK && resp.status != http.StatusNoContent {
		return c.errorForResponse(http.MethodDelete, path, resp)
	}
	return nil
}

// History returns the version history of one resource as a history Bundle (GET
// [type]/[id]/_history). The Bundle is returned as a concrete release Bundle behind the
// fhir.Resource interface; a caller narrows it with fhir.As. Like a search, a multi-page history
// can be paged with FollowNext over the returned Bundle's links — use SearchAll-style paging by
// calling FollowNext if the caller wants every page.
func (c *Client) History(ctx context.Context, resourceType, id string) (fhir.Resource, error) {
	if err := validateTypeID(resourceType, id); err != nil {
		return nil, err
	}
	path := resourceType + "/" + id + "/_history"
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp.status != http.StatusOK {
		return nil, c.errorForResponse(http.MethodGet, path, resp)
	}
	return c.decodeResource(resp)
}

// resultFromResponse decodes the body into a release resource and lifts the addressing headers into
// a Result, so the CRUD methods share one response-unpacking path.
func (c *Client) resultFromResponse(resp *response) (*Result, error) {
	r, err := c.decodeResource(resp)
	if err != nil {
		return nil, err
	}
	location := resp.header.Get("Location")
	if location == "" {
		location = resp.header.Get("Content-Location")
	}
	return &Result{
		Resource:     r,
		ETag:         resp.header.Get("ETag"),
		Location:     location,
		LastModified: resp.header.Get("Last-Modified"),
	}, nil
}

// encodeResource marshals a resource to FHIR JSON and returns its resourceType (the endpoint path
// segment) and the encoded bytes. A nil resource (a nil interface or a typed-nil pointer) is a
// usage error rather than a marshalled "null", so a misuse fails fast and never POSTs an empty body
// the server would reject obscurely.
func (c *Client) encodeResource(r fhir.Resource) (string, []byte, error) {
	if _, ok := fhir.As[fhir.Resource](r); !ok {
		return "", nil, fmt.Errorf("fhir/rest: resource is nil")
	}
	rt := r.ResourceType()
	body, err := json.Marshal(r)
	if err != nil {
		return "", nil, fmt.Errorf("fhir/rest: encode %s: %w", rt, err)
	}
	return rt, body, nil
}

// validateTypeID rejects an empty resource type or id before a request is built, so a malformed
// call is a clear usage error rather than a request to a path the server answers with an obscure
// 404. The resource type and id are not PHI, so naming them in the error is safe.
func validateTypeID(resourceType, id string) error {
	if strings.TrimSpace(resourceType) == "" {
		return fmt.Errorf("fhir/rest: resourceType is required")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("fhir/rest: id is required")
	}
	return nil
}
