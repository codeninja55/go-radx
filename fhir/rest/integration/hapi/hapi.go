//go:build interop

// Package hapi provides a testcontainers-backed HAPI FHIR server fixture for the FHIR REST client
// interop gate. It starts the upstream HAPI JPA starter image (an independent reference
// implementation, default R4) and blocks until its CapabilityStatement endpoint answers, so the
// gate drives the go-radx client against a server go-radx did not write. It is built only under
// the interop tag so the testcontainers dependency stays out of the default build.
package hapi

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// httpPort is the container-internal port the HAPI starter binds for the FHIR endpoint.
// testcontainers maps it to an ephemeral host port discovered after start.
const httpPort = "8080/tcp"

// fhirPath is the URI path the HAPI starter serves the FHIR service base under, its default.
const fhirPath = "/fhir"

// image pins the upstream HAPI FHIR JPA starter image by immutable digest so an interop run
// resolves the same bytes on every runner. The digest is the v8.10.0-1 multi-arch index; bumping
// it is a deliberate, reviewed change — re-resolve with `docker buildx imagetools inspect
// hapiproject/hapi:<version>` and update the digest here and in tools/versions together.
const image = "hapiproject/hapi:v8.10.0-1@sha256:1be4d7ffe7a35a9fb46151851e5a20b25c5016f16c8ef8b59b0c807ad06a40c1"

// Container wraps a started HAPI FHIR server and the host-side address its mapped HTTP port
// resolves to.
type Container struct {
	container testcontainers.Container
	host      string
	port      string
}

// Start launches a HAPI FHIR server container (default configuration: R4, in-memory H2, no auth)
// and blocks until /fhir/metadata answers. HAPI boots a full Spring/JPA stack, so the wait is
// deliberately generous; first runs also pull the image.
func Start(ctx context.Context) (*Container, error) {
	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{httpPort},
		WaitingFor: wait.ForHTTP(fhirPath + "/metadata").
			WithPort(httpPort).
			WithStartupTimeout(10 * time.Minute),
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("start HAPI FHIR container: %w", err)
	}

	host, err := c.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve HAPI host: %w", err)
	}
	hp, err := c.MappedPort(ctx, httpPort)
	if err != nil {
		return nil, fmt.Errorf("resolve mapped HTTP port: %w", err)
	}

	return &Container{container: c, host: host, port: hp.Port()}, nil
}

// Stop terminates the container. It is safe to call on a nil container.
func (c *Container) Stop(ctx context.Context) error {
	if c == nil || c.container == nil {
		return nil
	}
	return c.container.Terminate(ctx)
}

// BaseURL returns the FHIR service base URL of the started server, e.g. http://host:port/fhir.
func (c *Container) BaseURL() string {
	return fmt.Sprintf("http://%s:%s%s", c.host, c.port, fhirPath)
}
