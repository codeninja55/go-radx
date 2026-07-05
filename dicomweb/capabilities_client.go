package dicomweb

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Capabilities is the parsed Retrieve Capabilities description of an origin server
// (PS3.18 §8.9): the resources it advertises and, per resource, the methods and
// representation media types. It is a minimal view of the WADL document — enough to
// answer "which transactions and media types does this origin offer" — not a full WADL
// model.
type Capabilities struct {
	// Resources lists the advertised resources in document order, each with its path
	// relative to the service root (nested WADL resources are flattened by joining paths).
	Resources []CapabilityResource
}

// CapabilityResource is one advertised resource: its path template relative to the
// service root (for example "studies/{study}/series") and the methods it serves.
type CapabilityResource struct {
	Path    string
	Methods []CapabilityMethod
}

// CapabilityMethod is one advertised method on a resource: the HTTP method name and the
// representation media types its request and response name, deduplicated in document
// order.
type CapabilityMethod struct {
	Name       string
	MediaTypes []string
}

// Capabilities issues the Retrieve Capabilities transaction (PS3.18 §8.9): an OPTIONS on
// the service root, expecting the WADL Capabilities Description, parsed minimally into
// the advertised resources, methods, and media types. A non-200 answer is a typed
// *HTTPError; a response that is not a WADL/XML document is a typed error, never an empty
// Capabilities a caller would read as "no services".
func (c *Client) Capabilities(ctx context.Context) (*Capabilities, error) {
	const path = "/"
	req, err := c.newRequest(ctx, http.MethodOptions, path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", mediaTypeWADL)

	resp, err := c.httpClient.Do(req) // #nosec G704 -- the URL is joined from the caller-configured base URL (newRequest); requesting the configured service is the client's purpose
	if err != nil {
		return nil, c.transportError(http.MethodOptions, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, c.httpError(http.MethodOptions, path, resp)
	}
	if !isWADLContentType(resp.Header.Get("Content-Type")) {
		return nil, fmt.Errorf("%w: capabilities response is not a WADL document", ErrNotAcceptable)
	}
	raw, err := io.ReadAll(c.boundedBody(resp))
	if err != nil {
		return nil, c.readError(http.MethodOptions, path, err)
	}
	return parseCapabilities(raw)
}

// isWADLContentType reports whether a Content-Type names the WADL media type or a generic
// XML type an origin may serve the description under.
func isWADLContentType(contentType string) bool {
	switch mt := mediaTypeOf(contentType); mt {
	case mediaTypeWADL, "application/xml", "text/xml":
		return true
	default:
		return strings.HasSuffix(mt, "+xml")
	}
}

// maxWADLResourceDepth caps the nesting of WADL resource elements the parser follows, so
// a hostile origin cannot drive the flattening recursion arbitrarily deep (PRD §9.3).
const maxWADLResourceDepth = 32

// parseCapabilities decodes a WADL Capabilities Description into the minimal Capabilities
// view. Nested resources are flattened by joining parent and child path templates; a
// resource with no method of its own contributes only its path prefix. A document that is
// not well-formed XML is a typed error.
func parseCapabilities(raw []byte) (*Capabilities, error) {
	var app wadlApplication
	if err := xml.Unmarshal(raw, &app); err != nil {
		return nil, fmt.Errorf("dicomweb: parse capabilities description: %w", err)
	}
	caps := &Capabilities{}
	if err := flattenWADLResources("", app.Resources.Resource, 0, caps); err != nil {
		return nil, err
	}
	return caps, nil
}

// flattenWADLResources walks the (possibly nested) WADL resource tree, appending one
// CapabilityResource per resource that carries methods, with its full joined path.
func flattenWADLResources(prefix string, in []wadlResource, depth int, caps *Capabilities) error {
	if depth > maxWADLResourceDepth {
		return &LimitExceededError{
			Limit:  maxWADLResourceDepth,
			Actual: uint64(depth), // #nosec G115 -- depth is a small non-negative recursion counter
			Kind:   "wadl-resource-depth",
		}
	}
	for _, res := range in {
		path := joinWADLPath(prefix, res.Path)
		if len(res.Methods) > 0 {
			cr := CapabilityResource{Path: path}
			for _, m := range res.Methods {
				cr.Methods = append(cr.Methods, CapabilityMethod{
					Name:       strings.ToUpper(strings.TrimSpace(m.Name)),
					MediaTypes: methodMediaTypes(m),
				})
			}
			caps.Resources = append(caps.Resources, cr)
		}
		if err := flattenWADLResources(path, res.Resource, depth+1, caps); err != nil {
			return err
		}
	}
	return nil
}

// joinWADLPath joins a parent and child resource path template with a single separator,
// tolerating stray slashes on either side.
func joinWADLPath(prefix, path string) string {
	prefix = strings.Trim(prefix, "/")
	path = strings.Trim(path, "/")
	switch {
	case prefix == "":
		return path
	case path == "":
		return prefix
	default:
		return prefix + "/" + path
	}
}

// methodMediaTypes collects a method's request and response representation media types,
// deduplicated in document order.
func methodMediaTypes(m wadlMethod) []string {
	var out []string
	seen := make(map[string]struct{})
	for _, payload := range []*wadlPayload{m.Request, m.Response} {
		if payload == nil {
			continue
		}
		for _, rep := range payload.Representation {
			mt := strings.TrimSpace(rep.MediaType)
			if mt == "" {
				continue
			}
			if _, dup := seen[mt]; dup {
				continue
			}
			seen[mt] = struct{}{}
			out = append(out, mt)
		}
	}
	return out
}
