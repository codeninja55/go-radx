# Cloud-provider authentication for DICOMweb

The DICOMweb client authenticates through a pluggable transport seam: every scheme is either an
`oauth2.TokenSource` wired with `dicomweb.WithTokenSource` or an `http.RoundTripper` wired with
`dicomweb.WithRoundTripper`. The two managed cloud archives reachable over DICOMweb — Google Cloud Healthcare and
AWS HealthImaging — compose through that same seam, so the cloud SDKs stay isolated in the `dicomweb/auth/gcp` and
`dicomweb/auth/aws` subpackages and never enter the core client's import graph. A caller who never touches a cloud
adapter never pays the SDK's transitive-dependency cost.

This guide shows how to wire each provider to a `dicomweb.Client`.

## Prerequisites

- A Go module that depends on `github.com/codeninja55/go-radx`.
- For Google Cloud Healthcare: a project with the Cloud Healthcare API enabled, a DICOM store, and Application
  Default Credentials available in the environment (a service-account key referenced by
  `GOOGLE_APPLICATION_CREDENTIALS`, `gcloud auth application-default login`, or a GCE/GKE workload identity).
- For AWS HealthImaging: an AWS account with a HealthImaging datastore, and either AWS credentials resolvable by the
  default config chain (environment variables, shared config, or an instance/role profile) for SigV4 mode, or an
  OIDC identity provider for OIDC mode.

## Google Cloud Healthcare (Application Default Credentials)

The Cloud Healthcare DICOMweb endpoint authenticates with a Google OAuth2 bearer token scoped to `cloud-platform`.
The `dicomweb/auth/gcp` adapter resolves that token from the ambient ADC chain and exposes it as an
`oauth2.TokenSource`, which you pass to `dicomweb.WithTokenSource`. The token source refreshes on expiry, so a
long-lived client re-authenticates mid-session without your involvement.

```go
package main

import (
	"context"
	"fmt"

	"github.com/codeninja55/go-radx/dicomweb"
	"github.com/codeninja55/go-radx/dicomweb/auth/gcp"
)

func newGoogleHealthcareClient(ctx context.Context, project, location, dataset, dicomStore string) (*dicomweb.Client, error) {
	ts, err := gcp.TokenSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve Google ADC: %w", err)
	}

	endpoint := fmt.Sprintf(
		"https://healthcare.googleapis.com/v1/projects/%s/locations/%s/datasets/%s/dicomStores/%s/dicomWeb",
		project, location, dataset, dicomStore,
	)
	return dicomweb.NewClient(endpoint, dicomweb.WithTokenSource(ts))
}
```

The endpoint path follows the Cloud Healthcare DICOMweb layout (`.../dicomStores/{store}/dicomWeb`); the client then
issues standard WADO-RS, QIDO-RS, and STOW-RS requests beneath it.

## AWS HealthImaging (SigV4)

AWS HealthImaging in its traditional access mode signs every request to the `medical-imaging` service with the
caller's AWS credentials using Signature Version 4 — a per-request signature, not a static header. The
`dicomweb/auth/aws` adapter returns an `http.RoundTripper` that derives a fresh signature for each request from the
credentials in an `aws.Config`, which you pass to `dicomweb.WithRoundTripper`.

```go
package main

import (
	"context"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/codeninja55/go-radx/dicomweb"
	awsauth "github.com/codeninja55/go-radx/dicomweb/auth/aws"
)

func newHealthImagingSigV4Client(ctx context.Context, region, endpoint string) (*dicomweb.Client, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	rt, err := awsauth.SigV4RoundTripper(cfg, region, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build SigV4 transport: %w", err)
	}
	return dicomweb.NewClient(endpoint, dicomweb.WithRoundTripper(rt))
}
```

The `nil` base transport uses `http.DefaultTransport`; pass your own base if you need custom TLS or proxy settings.
The signer reads credentials from `cfg` on every request, so a rotating or assumed-role credential is always current.
Signing is scoped to the `endpoint` origin: the transport signs only requests whose scheme, host, and port match
`endpoint` and forwards any other request unsigned, so a cross-origin `BulkDataURI` a metadata response names can
never make the client attach AWS credentials to a host you did not target.

## AWS HealthImaging (OIDC)

AWS HealthImaging's OIDC access mode authenticates with a standard OAuth2 bearer token, so it needs no dedicated
adapter: any `oauth2.TokenSource` that yields the OIDC access token wires straight into the core
`dicomweb.WithTokenSource` path.

```go
package main

import (
	"github.com/codeninja55/go-radx/dicomweb"
	"golang.org/x/oauth2"
)

func newHealthImagingOIDCClient(endpoint string, ts oauth2.TokenSource) (*dicomweb.Client, error) {
	return dicomweb.NewClient(endpoint, dicomweb.WithTokenSource(ts))
}
```

Construct the token source from your OIDC provider's client-credentials or refresh-token flow (for example with
`golang.org/x/oauth2/clientcredentials`), then hand it to the client. The token refreshes on expiry through the same
mechanism the other token-source schemes use.

## See also

- [DICOMweb conformance statement — client authentication](../../conformance/dicomweb.md)
