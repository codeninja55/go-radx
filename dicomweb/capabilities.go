package dicomweb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Capabilities discovery (PS3.18 §8.9). The standard says every RESTful DICOM service shall
// answer a Retrieve Capabilities request — an OPTIONS on the service base URI — describing the
// transactions and resources it supports. PS3.18 frames the response as a machine-readable
// description and names the OpenAPI/RESTful Services Description Document as the modern form (the
// older Web Application Description Language is also permitted). go-radx serves a pragmatic JSON
// capabilities document rather than the full WADL/OpenAPI surface: it enumerates the services and
// resources this origin actually wires (driven by the registered backends) so a client can probe
// what a server supports without trial requests. The document is deliberately compact and is not
// a conformant OpenAPI description; a consumer needing the full PS3.18 capabilities schema must
// treat this as an informational summary. The media type is application/json (not a DICOM media
// type), because the document describes the service rather than carrying DICOM content.

// CapabilitiesMediaType is the Content-Type the capabilities document is served and parsed as.
// PS3.18 §8.9 permits an implementation-chosen description format; go-radx uses application/json
// for a compact, self-describing summary rather than the heavier WADL or OpenAPI forms.
const CapabilitiesMediaType = "application/json"

// Capabilities is the go-radx capabilities document a DICOMweb origin returns for a Retrieve
// Capabilities request (PS3.18 §8.9, OPTIONS on the service base). It is a pragmatic summary of
// the transactions and resources the origin wires, not a conformant OpenAPI/WADL description.
// Version names the document schema so a client can detect a future, incompatible shape.
type Capabilities struct {
	// Version is the capabilities-document schema version. It is "1" for this shape.
	Version string `json:"version"`
	// Library names the implementation serving the document, for diagnostics.
	Library string `json:"library"`
	// Services lists the DICOMweb services this origin supports (a subset of "WADO-RS",
	// "QIDO-RS", "STOW-RS", "WADO-URI"), in a stable order.
	Services []string `json:"services"`
	// Transactions describes each supported transaction: the methods and the resource paths it
	// answers, so a client can discover the wired surface without probing.
	Transactions []CapabilityTransaction `json:"transactions"`
}

// CapabilityTransaction describes one DICOMweb transaction the origin supports: the service it
// belongs to, the HTTP methods it answers, and the resource path templates it serves. The path
// templates use {study}/{series}/{instance} placeholders rather than concrete UIDs, so the
// document carries no PHI (PRD §9.1).
type CapabilityTransaction struct {
	// Service is the owning service: "WADO-RS", "QIDO-RS", "STOW-RS", or "WADO-URI".
	Service string `json:"service"`
	// Name is the human-readable transaction name, e.g. "Retrieve Instance" or "Search Studies".
	Name string `json:"name"`
	// Methods are the HTTP methods this transaction answers, e.g. ["GET"] or ["POST"].
	Methods []string `json:"methods"`
	// Paths are the resource path templates this transaction serves, e.g.
	// "/studies/{study}/series/{series}/instances/{instance}". Templates carry placeholders, never
	// concrete UIDs.
	Paths []string `json:"paths"`
}

// HasService reports whether the capabilities document advertises the named service (a
// case-insensitive match on the service name, e.g. "WADO-RS"). It is the convenience a client
// uses to branch on what an origin supports after RetrieveCapabilities.
func (c *Capabilities) HasService(service string) bool {
	for _, s := range c.Services {
		if equalFoldASCII(s, service) {
			return true
		}
	}
	return false
}

// RetrieveCapabilities fetches and parses the origin's capabilities document with a Retrieve
// Capabilities request (PS3.18 §8.9): an OPTIONS on the configured service base URL. The origin
// returns a description of the services and transactions it supports. go-radx servers return the
// pragmatic JSON document this package defines; a foreign origin that returns a different
// description format (OpenAPI, WADL) is not parsed here and yields a typed decode error — use a
// caller-supplied parser for those. A non-2xx status is a typed *HTTPError.
func (c *Client) RetrieveCapabilities(ctx context.Context) (*Capabilities, error) {
	req, err := c.newRequest(ctx, http.MethodOptions, "", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", CapabilitiesMediaType)

	resp, err := c.do(req, http.MethodOptions, "/") // #nosec G704 -- URL joined from the caller-configured base URL
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, c.httpError(http.MethodOptions, "/", resp)
	}

	raw, err := io.ReadAll(c.boundedBody(resp))
	if err != nil {
		return nil, c.readError(http.MethodOptions, "/", err)
	}
	var caps Capabilities
	if err := json.Unmarshal(raw, &caps); err != nil {
		return nil, fmt.Errorf("dicomweb: parse capabilities document: %w", &DecodeError{Msg: "capabilities response is not a recognised JSON document"})
	}
	return &caps, nil
}

// buildCapabilities assembles the capabilities document from the registered backends, so the
// document reflects exactly the surface this server wires (a query-only deployment advertises
// QIDO-RS alone). The WADO-URI service is advertised whenever a retrieve backend is present,
// since the URI service reuses that backend.
func (s *Server) buildCapabilities() Capabilities {
	caps := Capabilities{
		Version: "1",
		Library: "github.com/codeninja55/go-radx",
	}
	if s.query != nil {
		caps.Services = append(caps.Services, "QIDO-RS")
		caps.Transactions = append(caps.Transactions, qidoCapabilityTransactions()...)
	}
	if s.retrieve != nil {
		caps.Services = append(caps.Services, "WADO-RS", "WADO-URI")
		caps.Transactions = append(caps.Transactions, wadoCapabilityTransactions()...)
	}
	if s.store != nil {
		caps.Services = append(caps.Services, "STOW-RS")
		caps.Transactions = append(caps.Transactions, stowCapabilityTransactions()...)
	}
	return caps
}

// qidoCapabilityTransactions enumerates the QIDO-RS search transactions the server answers
// (PS3.18 §10.6), with their resource path templates.
func qidoCapabilityTransactions() []CapabilityTransaction {
	return []CapabilityTransaction{
		{Service: "QIDO-RS", Name: "Search for Studies", Methods: []string{"GET"}, Paths: []string{"/studies"}},
		{Service: "QIDO-RS", Name: "Search for Series", Methods: []string{"GET"}, Paths: []string{"/series", "/studies/{study}/series"}},
		{Service: "QIDO-RS", Name: "Search for Instances", Methods: []string{"GET"}, Paths: []string{
			"/instances", "/studies/{study}/instances", "/studies/{study}/series/{series}/instances",
		}},
	}
}

// wadoCapabilityTransactions enumerates the WADO-RS retrieval transactions the server answers
// (PS3.18 §10.4) plus the WADO-URI single-object retrieve (PS3.18 §9), with their path templates.
func wadoCapabilityTransactions() []CapabilityTransaction {
	return []CapabilityTransaction{
		{Service: "WADO-RS", Name: "Retrieve Study", Methods: []string{"GET"}, Paths: []string{"/studies/{study}"}},
		{Service: "WADO-RS", Name: "Retrieve Series", Methods: []string{"GET"}, Paths: []string{"/studies/{study}/series/{series}"}},
		{Service: "WADO-RS", Name: "Retrieve Instance", Methods: []string{"GET"}, Paths: []string{
			"/studies/{study}/series/{series}/instances/{instance}",
		}},
		{Service: "WADO-RS", Name: "Retrieve Metadata", Methods: []string{"GET"}, Paths: []string{
			"/studies/{study}/metadata",
			"/studies/{study}/series/{series}/metadata",
			"/studies/{study}/series/{series}/instances/{instance}/metadata",
		}},
		{Service: "WADO-RS", Name: "Retrieve Frames", Methods: []string{"GET"}, Paths: []string{
			"/studies/{study}/series/{series}/instances/{instance}/frames/{frames}",
		}},
		{Service: "WADO-RS", Name: "Retrieve Bulkdata", Methods: []string{"GET"}, Paths: []string{
			"/studies/{study}/series/{series}/instances/{instance}/bulkdata",
		}},
		{Service: "WADO-URI", Name: "Retrieve (URI)", Methods: []string{"GET"}, Paths: []string{"/?requestType=WADO"}},
	}
}

// stowCapabilityTransactions enumerates the STOW-RS store transactions the server answers
// (PS3.18 §10.5), with their target path templates.
func stowCapabilityTransactions() []CapabilityTransaction {
	return []CapabilityTransaction{
		{Service: "STOW-RS", Name: "Store Instances", Methods: []string{"POST"}, Paths: []string{"/studies", "/studies/{study}"}},
	}
}

// equalFoldASCII reports a case-insensitive ASCII equality, used so HasService matches a service
// name regardless of the caller's casing without pulling in the full Unicode folding of
// strings.EqualFold for a fixed ASCII vocabulary.
func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range len(a) {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
