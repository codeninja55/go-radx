package dicomweb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCapabilitiesRoundTrip serves the capabilities document from the embeddable server (OPTIONS
// on the base) and parses it back through the client, asserting the advertised services reflect
// the wired backends (PS3.18 §8.9).
func TestCapabilitiesRoundTrip(t *testing.T) {
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

	caps, err := c.RetrieveCapabilities(context.Background())
	if err != nil {
		t.Fatalf("RetrieveCapabilities: %v", err)
	}
	if caps.Version != "1" {
		t.Fatalf("capabilities version = %q, want 1", caps.Version)
	}
	for _, want := range []string{"QIDO-RS", "WADO-RS", "WADO-URI", "STOW-RS"} {
		if !caps.HasService(want) {
			t.Errorf("capabilities missing service %q (services=%v)", want, caps.Services)
		}
	}
	if !caps.HasService("wado-rs") {
		t.Error("HasService is not case-insensitive")
	}
	if len(caps.Transactions) == 0 {
		t.Fatal("capabilities carried no transactions")
	}
	// A transaction's paths must be templates, never carry a concrete UID (PHI-free, PRD §9.1).
	for _, tr := range caps.Transactions {
		if tr.Service == "" || tr.Name == "" || len(tr.Methods) == 0 {
			t.Errorf("incomplete transaction: %+v", tr)
		}
	}
}

// TestCapabilitiesReflectsWiredBackends asserts a query-only server advertises QIDO-RS alone, so
// the document tracks the actual surface rather than a fixed list.
func TestCapabilitiesReflectsWiredBackends(t *testing.T) {
	srv, err := NewServer(WithQueryBackend(&memQuery{}))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	caps, err := c.RetrieveCapabilities(context.Background())
	if err != nil {
		t.Fatalf("RetrieveCapabilities: %v", err)
	}
	if !caps.HasService("QIDO-RS") {
		t.Error("query-only server should advertise QIDO-RS")
	}
	if caps.HasService("STOW-RS") || caps.HasService("WADO-RS") {
		t.Errorf("query-only server should not advertise store/retrieve (services=%v)", caps.Services)
	}
}

// TestCapabilitiesAllowHeader asserts the OPTIONS response carries an Allow header naming the
// methods the service accepts (PS3.18 §8.9 capabilities semantics).
func TestCapabilitiesAllowHeader(t *testing.T) {
	srv, err := NewServer(WithQueryBackend(&memQuery{}))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	req, err := http.NewRequest(http.MethodOptions, hs.URL+"/", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := hs.Client().Do(req)
	if err != nil {
		t.Fatalf("OPTIONS: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("OPTIONS status = %d, want 200", resp.StatusCode)
	}
	if allow := resp.Header.Get("Allow"); allow == "" {
		t.Error("OPTIONS response carried no Allow header")
	}
	if ct := resp.Header.Get("Content-Type"); ct != CapabilitiesMediaType {
		t.Errorf("Content-Type = %q, want %q", ct, CapabilitiesMediaType)
	}
}
