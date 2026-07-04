package dicomweb

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// capabilityByPath indexes a parsed Capabilities by resource path for assertion convenience.
func capabilityByPath(caps *Capabilities) map[string]CapabilityResource {
	out := make(map[string]CapabilityResource, len(caps.Resources))
	for _, r := range caps.Resources {
		out[r.Path] = r
	}
	return out
}

// methodNames lists a resource's method names for assertion convenience.
func methodNames(r CapabilityResource) []string {
	out := make([]string, 0, len(r.Methods))
	for _, m := range r.Methods {
		out = append(out, m.Name)
	}
	return out
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

// TestServerCapabilitiesDescribesMountedServices asserts an OPTIONS on the service root
// answers the Retrieve Capabilities transaction (PS3.18 §8.9): a WADL description whose
// resources reflect exactly the services the server has mounted.
func TestServerCapabilitiesDescribesMountedServices(t *testing.T) {
	store := newMemStore()
	srv, err := NewServer(
		WithStoreBackend(store),
		WithRetrieveBackend(newWADOStore()),
		WithQueryBackend(&memQuery{}),
	)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	req, err := http.NewRequest(http.MethodOptions, hs.URL+"/", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := hs.Client().Do(req)
	if err != nil {
		t.Fatalf("OPTIONS /: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("OPTIONS / status = %d, want 200", resp.StatusCode)
	}
	if ct := mediaTypeOf(resp.Header.Get("Content-Type")); ct != mediaTypeWADL {
		t.Fatalf("Content-Type = %q, want %q", ct, mediaTypeWADL)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	caps, err := parseCapabilities(raw)
	if err != nil {
		t.Fatalf("parseCapabilities: %v", err)
	}

	byPath := capabilityByPath(caps)
	studies, ok := byPath["studies"]
	if !ok {
		t.Fatalf("capabilities missing the studies resource; got %+v", caps.Resources)
	}
	if !contains(methodNames(studies), http.MethodGet) || !contains(methodNames(studies), http.MethodPost) {
		t.Fatalf("studies methods = %v, want GET (search) and POST (store)", methodNames(studies))
	}
	if _, ok := byPath["studies/{study}/series/{series}/instances/{instance}"]; !ok {
		t.Fatalf("capabilities missing the instance retrieve resource; got %+v", caps.Resources)
	}
	// wadoStore implements MetadataRetriever, so the metadata sub-resource is advertised.
	md, ok := byPath["studies/{study}/series/{series}/instances/{instance}/metadata"]
	if !ok {
		t.Fatalf("capabilities missing the instance metadata resource; got %+v", caps.Resources)
	}
	var mdTypes []string
	for _, m := range md.Methods {
		mdTypes = append(mdTypes, m.MediaTypes...)
	}
	if !contains(mdTypes, mediaTypeDICOMJSON) {
		t.Fatalf("metadata media types = %v, want %s advertised", mdTypes, mediaTypeDICOMJSON)
	}
}

// TestServerCapabilitiesOmitsUnmountedServices asserts the description never advertises a
// transaction whose backend is not mounted: a query-only server describes searches, no
// store target and no retrieve resources.
func TestServerCapabilitiesOmitsUnmountedServices(t *testing.T) {
	srv, err := NewServer(WithQueryBackend(&memQuery{}))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	req, _ := http.NewRequest(http.MethodOptions, hs.URL+"/", nil)
	resp, err := hs.Client().Do(req)
	if err != nil {
		t.Fatalf("OPTIONS /: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	caps, err := parseCapabilities(raw)
	if err != nil {
		t.Fatalf("parseCapabilities: %v", err)
	}

	for _, r := range caps.Resources {
		for _, m := range r.Methods {
			if m.Name == http.MethodPost {
				t.Fatalf("query-only server advertised POST on %q", r.Path)
			}
		}
		if strings.Contains(r.Path, "{instance}") {
			t.Fatalf("query-only server advertised the retrieve resource %q", r.Path)
		}
	}
	if _, ok := capabilityByPath(caps)["studies"]; !ok {
		t.Fatalf("query-only server did not advertise the studies search; got %+v", caps.Resources)
	}
}

// TestServerCapabilitiesNegotiatesWADL asserts the capabilities response is negotiated
// fail-closed: an Accept that does not admit the WADL media type answers 406, never a
// substitute representation.
func TestServerCapabilitiesNegotiatesWADL(t *testing.T) {
	srv, err := NewServer(WithQueryBackend(&memQuery{}))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	req, _ := http.NewRequest(http.MethodOptions, hs.URL+"/", nil)
	req.Header.Set("Accept", "application/json")
	resp, err := hs.Client().Do(req)
	if err != nil {
		t.Fatalf("OPTIONS /: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotAcceptable {
		t.Fatalf("OPTIONS / with Accept application/json = %d, want 406", resp.StatusCode)
	}
}

// TestServerOptionsOffRootUnrouted asserts OPTIONS is the capabilities transaction only at
// the service root; elsewhere it stays the typed 501, never a silent success.
func TestServerOptionsOffRootUnrouted(t *testing.T) {
	srv, err := NewServer(WithQueryBackend(&memQuery{}))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	req, _ := http.NewRequest(http.MethodOptions, hs.URL+"/studies", nil)
	resp, err := hs.Client().Do(req)
	if err != nil {
		t.Fatalf("OPTIONS /studies: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("OPTIONS /studies = %d, want 501", resp.StatusCode)
	}
}

// TestClientCapabilitiesRoundTrip fetches the embeddable server's own capabilities through
// the client and asserts the advertised transactions surface in the parsed form.
func TestClientCapabilitiesRoundTrip(t *testing.T) {
	store := newMemStore()
	srv, err := NewServer(WithStoreBackend(store), WithRetrieveBackend(store), WithQueryBackend(&memQuery{}))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	caps, err := c.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities: %v", err)
	}
	studies, ok := capabilityByPath(caps)["studies"]
	if !ok {
		t.Fatalf("Capabilities missing the studies resource; got %+v", caps.Resources)
	}
	if !contains(methodNames(studies), http.MethodGet) || !contains(methodNames(studies), http.MethodPost) {
		t.Fatalf("studies methods = %v, want GET and POST", methodNames(studies))
	}
}

// TestClientCapabilitiesParsesNestedResources asserts the parser flattens the nested
// resource form WADL permits, joining parent and child paths, and collects representation
// media types from both request and response.
func TestClientCapabilitiesParsesNestedResources(t *testing.T) {
	const doc = `<?xml version="1.0" encoding="UTF-8"?>
<application xmlns="http://wadl.dev.java.net/2009/02">
  <resources base="http://pacs.example.org/dicom-web/">
    <resource path="studies">
      <method name="GET" id="SearchForStudies">
        <response><representation mediaType="application/dicom+json"/></response>
      </method>
      <resource path="{study}/series">
        <method name="GET" id="SearchForStudySeries">
          <response><representation mediaType="application/dicom+json"/></response>
        </method>
      </resource>
    </resource>
  </resources>
</application>`
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodOptions {
			t.Errorf("origin saw method %s, want OPTIONS", r.Method)
		}
		w.Header().Set("Content-Type", mediaTypeWADL)
		_, _ = io.WriteString(w, doc)
	}))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	caps, err := c.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities: %v", err)
	}
	byPath := capabilityByPath(caps)
	if _, ok := byPath["studies"]; !ok {
		t.Fatalf("missing the parent studies resource; got %+v", caps.Resources)
	}
	nested, ok := byPath["studies/{study}/series"]
	if !ok {
		t.Fatalf("nested resource path not flattened; got %+v", caps.Resources)
	}
	if len(nested.Methods) != 1 || nested.Methods[0].Name != http.MethodGet {
		t.Fatalf("nested methods = %+v, want one GET", nested.Methods)
	}
	if !contains(nested.Methods[0].MediaTypes, mediaTypeDICOMJSON) {
		t.Fatalf("nested media types = %v, want %s", nested.Methods[0].MediaTypes, mediaTypeDICOMJSON)
	}
}

// TestClientCapabilitiesFailsClosed asserts a non-200 answer and a non-WADL body are typed
// errors, never an empty Capabilities read as "no services".
func TestClientCapabilitiesFailsClosed(t *testing.T) {
	t.Run("http error", func(t *testing.T) {
		hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotImplemented)
		}))
		t.Cleanup(hs.Close)
		c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		_, err = c.Capabilities(context.Background())
		var he *HTTPError
		if !errors.As(err, &he) || he.StatusCode != http.StatusNotImplemented {
			t.Fatalf("Capabilities against a 501 origin = %v, want *HTTPError 501", err)
		}
	})
	t.Run("non-WADL content type", func(t *testing.T) {
		hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = io.WriteString(w, "<html></html>")
		}))
		t.Cleanup(hs.Close)
		c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		if _, err := c.Capabilities(context.Background()); !errors.Is(err, ErrNotAcceptable) {
			t.Fatalf("Capabilities with a text/html body = %v, want ErrNotAcceptable", err)
		}
	})
	t.Run("malformed WADL", func(t *testing.T) {
		hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", mediaTypeWADL)
			_, _ = io.WriteString(w, "<application><resources>")
		}))
		t.Cleanup(hs.Close)
		c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		if _, err := c.Capabilities(context.Background()); err == nil {
			t.Fatal("Capabilities with a malformed WADL body returned nil error")
		}
	})
}

// TestParseCapabilitiesDepthCap asserts a hostile, deeply nested resource tree trips the
// depth cap with a typed limit error rather than exhausting the stack (PRD §9.3).
func TestParseCapabilitiesDepthCap(t *testing.T) {
	var b strings.Builder
	b.WriteString(`<application xmlns="http://wadl.dev.java.net/2009/02"><resources>`)
	const depth = 200
	for range depth {
		b.WriteString(`<resource path="x"><method name="GET"/>`)
	}
	for range depth {
		b.WriteString(`</resource>`)
	}
	b.WriteString(`</resources></application>`)

	if _, err := parseCapabilities([]byte(b.String())); !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("parseCapabilities on a %d-deep tree = %v, want ErrLimitExceeded", depth, err)
	}
}

// TestServerCapabilitiesUsesConfiguredPublicBase asserts the WADL resources base honours
// the configured public root the way the STOW Retrieve URLs already do, so a deployment
// behind a path-rewriting reverse proxy advertises resolvable URLs.
func TestServerCapabilitiesUsesConfiguredPublicBase(t *testing.T) {
	srv, err := NewServer(WithQueryBackend(&memQuery{}),
		WithStoreRetrieveURLBase("https://pacs.example.org/dicom-web"))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	req, _ := http.NewRequest(http.MethodOptions, hs.URL+"/", nil)
	resp, err := hs.Client().Do(req)
	if err != nil {
		t.Fatalf("OPTIONS /: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)

	var app wadlApplication
	if err := xml.Unmarshal(raw, &app); err != nil {
		t.Fatalf("unmarshal WADL: %v", err)
	}
	if app.Resources.Base != "https://pacs.example.org/dicom-web/" {
		t.Fatalf("resources base = %q, want the configured public root", app.Resources.Base)
	}
}

// TestServerCapabilitiesAcceptsTextXML asserts the legacy generic XML media type is
// admitted, matching what the client itself accepts on the response side.
func TestServerCapabilitiesAcceptsTextXML(t *testing.T) {
	srv, err := NewServer(WithQueryBackend(&memQuery{}))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	req, _ := http.NewRequest(http.MethodOptions, hs.URL+"/", nil)
	req.Header.Set("Accept", "text/xml")
	resp, err := hs.Client().Do(req)
	if err != nil {
		t.Fatalf("OPTIONS /: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("OPTIONS / with Accept text/xml = %d, want 200", resp.StatusCode)
	}
}
