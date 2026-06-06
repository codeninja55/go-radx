// Package gcp adapts Google Application Default Credentials (ADC) to the dicomweb
// client's pluggable authentication seam. The Google Cloud Healthcare API exposes a
// DICOMweb endpoint at healthcare.googleapis.com/v1/.../dicomWeb that authenticates
// with a Google OAuth2 bearer token scoped to cloud-platform; this package resolves
// that token from the ambient ADC chain (the GOOGLE_APPLICATION_CREDENTIALS service
// account, gcloud user credentials, or the GCE/GKE metadata server) and exposes it as
// an oauth2.TokenSource.
//
// A caller wires the token source through the core dicomweb.WithTokenSource option, so
// the Google credential machinery stays isolated in this subpackage and never enters
// the core dicomweb import graph:
//
//	ts, err := gcp.TokenSource(ctx)
//	if err != nil {
//		return err
//	}
//	c, err := dicomweb.NewClient(endpoint, dicomweb.WithTokenSource(ts))
package gcp

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// CloudPlatformScope is the OAuth2 scope the Cloud Healthcare DICOMweb endpoint
// requires. It is the broad cloud-platform scope Google recommends for service-to-
// service access to the Healthcare API.
const CloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

// TokenSource resolves a Google Application Default Credentials token source scoped to
// the Cloud Healthcare DICOMweb endpoint. It searches the standard ADC chain: the
// service-account JSON named by GOOGLE_APPLICATION_CREDENTIALS, gcloud user
// credentials, and finally the GCE/GKE metadata server. The returned source caches a
// token until it expires and refreshes on demand, so a long-lived client
// re-authenticates mid-session without caller involvement.
//
// The token never appears in the returned error: a failure to locate credentials
// reports only that the ADC lookup failed, not any credential material.
func TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	ts, err := google.DefaultTokenSource(ctx, CloudPlatformScope)
	if err != nil {
		return nil, fmt.Errorf("gcp: resolve application default credentials: %w", err)
	}
	return ts, nil
}
