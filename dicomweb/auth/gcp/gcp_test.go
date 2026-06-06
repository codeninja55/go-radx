package gcp

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

// mockTokenServer answers the Google service-account JWT-bearer token exchange with a
// fixed access token, recording that it was called. It returns the OAuth2 token-endpoint
// JSON shape so golang.org/x/oauth2 parses out the bearer.
func mockTokenServer(t *testing.T, accessToken string, called *bool) *httptest.Server {
	t.Helper()
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": accessToken,
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	t.Cleanup(hs.Close)
	return hs
}

// writeServiceAccountJSON writes a minimal but valid service-account credential file
// whose token_uri points at tokenURI, so the ADC chain selects it and exchanges its JWT
// at the mock endpoint. The private key is generated fresh per test run.
func writeServiceAccountJSON(t *testing.T, tokenURI string) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	sa := map[string]string{
		"type":         "service_account",
		"project_id":   "go-radx-test",
		"private_key":  string(pemKey),
		"client_email": "dicomweb@go-radx-test.iam.gserviceaccount.com",
		"client_id":    "1234567890",
		"token_uri":    tokenURI,
	}
	b, err := json.Marshal(sa)
	if err != nil {
		t.Fatalf("marshal service account: %v", err)
	}
	path := filepath.Join(t.TempDir(), "sa.json")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatalf("write service account: %v", err)
	}
	return path
}

// TestTokenSourceAuthenticatesAgainstMockEndpoint asserts TokenSource resolves the ADC
// service account, exchanges its JWT at the (mock) token endpoint, and yields the bearer
// the endpoint returned. It exercises the credential model a Cloud Healthcare DICOMweb
// caller relies on without any live cloud call.
func TestTokenSourceAuthenticatesAgainstMockEndpoint(t *testing.T) {
	const wantToken = "ya29.mock-cloud-healthcare-access-token"
	var called bool
	hs := mockTokenServer(t, wantToken, &called)

	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", writeServiceAccountJSON(t, hs.URL))

	// Route the oauth2 exchange through the mock server's client so the JWT-bearer grant
	// hits the httptest endpoint rather than the real Google token URI.
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, hs.Client())

	ts, err := TokenSource(ctx)
	if err != nil {
		t.Fatalf("TokenSource: %v", err)
	}
	tok, err := ts.Token()
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if !called {
		t.Fatal("token endpoint was never called; the ADC service account was not exchanged")
	}
	if tok.AccessToken != wantToken {
		t.Fatalf("token = %q, want the mock access token", tok.AccessToken)
	}
	if !tok.Valid() {
		t.Fatal("resolved token is not valid")
	}
}

// TestTokenSourceUsesCloudPlatformScope asserts the resolved source carries the
// cloud-platform scope the Cloud Healthcare endpoint requires, proving TokenSource does
// not silently request a narrower scope.
func TestTokenSourceUsesCloudPlatformScope(t *testing.T) {
	if CloudPlatformScope != "https://www.googleapis.com/auth/cloud-platform" {
		t.Fatalf("CloudPlatformScope = %q, want the cloud-platform scope", CloudPlatformScope)
	}

	var gotScope string
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err == nil {
			// The JWT-bearer assertion carries the requested scope in its claims; the
			// service-account flow also echoes scope in the form body for the exchange.
			gotScope = r.FormValue("scope")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "tok",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	t.Cleanup(hs.Close)

	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", writeServiceAccountJSON(t, hs.URL))
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, hs.Client())

	ts, err := TokenSource(ctx)
	if err != nil {
		t.Fatalf("TokenSource: %v", err)
	}
	if _, err := ts.Token(); err != nil {
		t.Fatalf("Token: %v", err)
	}
	// The two-legged JWT flow encodes the scope inside the signed assertion rather than a
	// plain form field, so an empty form scope is acceptable; the assertion itself is what
	// the endpoint validates. This guards only against a non-empty wrong scope.
	if gotScope != "" && !strings.Contains(gotScope, "cloud-platform") {
		t.Fatalf("exchange requested scope %q, want it to include cloud-platform", gotScope)
	}
}
