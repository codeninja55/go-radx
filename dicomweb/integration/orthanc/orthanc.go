//go:build interop

// Package orthanc provides a testcontainers-backed Orthanc fixture for the DICOMweb
// interop gate. It starts an Orthanc container with the bundled DICOMweb plugin enabled
// (served under /dicom-web/), exposes the mapped REST/DICOMweb port, and offers a small
// REST client used to confirm a STOW-RS store actually landed. It is built only under
// the interop tag so the testcontainers dependency stays out of the default build.
package orthanc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// httpPort is the container-internal port Orthanc binds for the REST API and the
// DICOMweb plugin (the plugin shares the REST listener). testcontainers maps it to an
// ephemeral host port discovered after start.
const httpPort = "8042/tcp"

// dicomWebRoot is the URI path the bundled DICOMweb plugin serves under, the Orthanc
// default. The interop client's base URL is the REST origin plus this root.
const dicomWebRoot = "/dicom-web"

// image pins the maintained upstream Orthanc image by immutable digest so an interop run resolves the
// same bytes on every runner. It bundles the DICOMweb plugin and honours the ORTHANC__* environment
// overrides used below. The digest is the 26.6.0 multi-arch index; bumping it is a deliberate,
// reviewed change — re-resolve with `docker buildx imagetools inspect orthancteam/orthanc:<version>`
// and update the digest and the version in this comment together.
const image = "orthancteam/orthanc:26.6.0@sha256:510ef4ce24699104244b00d2b93350a801fc2f1c6b0bfc6a1f15e546bff2d1f4"

// Container wraps a started Orthanc instance and the host-side address its mapped REST
// port resolves to.
type Container struct {
	container testcontainers.Container
	httpHost  string
	httpPort  string
}

// Start launches an Orthanc container with the DICOMweb plugin enabled and blocks until
// its REST API answers. The container accepts unauthenticated access so the interop
// client can drive it directly.
func Start(ctx context.Context) (*Container, error) {
	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{httpPort},
		WaitingFor: wait.ForHTTP("/system").
			WithPort(httpPort).
			WithStartupTimeout(120 * time.Second),
		Env: map[string]string{
			"ORTHANC__AUTHENTICATION_ENABLED":     "false",
			"ORTHANC__REMOTE_ACCESS_ALLOWED":      "true",
			"ORTHANC__UNKNOWN_SOP_CLASS_ACCEPTED": "true",
			"ORTHANC__DICOM_WEB__ENABLE":          "true",
			"ORTHANC__DICOM_WEB__ROOT":            dicomWebRoot + "/",
		},
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("start Orthanc container: %w", err)
	}

	host, err := c.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve Orthanc host: %w", err)
	}
	hp, err := c.MappedPort(ctx, httpPort)
	if err != nil {
		return nil, fmt.Errorf("resolve mapped HTTP port: %w", err)
	}

	return &Container{container: c, httpHost: host, httpPort: hp.Port()}, nil
}

// Stop terminates the container. It is safe to call on a nil container.
func (c *Container) Stop(ctx context.Context) error {
	if c == nil || c.container == nil {
		return nil
	}
	return c.container.Terminate(ctx)
}

// HTTPBaseURL returns the base URL of the Orthanc REST API.
func (c *Container) HTTPBaseURL() string {
	return fmt.Sprintf("http://%s:%s", c.httpHost, c.httpPort)
}

// DICOMWebBaseURL returns the base URL of the Orthanc DICOMweb endpoint (the path that
// precedes /studies), e.g. http://host:port/dicom-web.
func (c *Container) DICOMWebBaseURL() string {
	return c.HTTPBaseURL() + dicomWebRoot
}

// orthancInstance is the subset of an Orthanc /instances/{id} response the interop gate
// inspects: the SOP Instance UID a store landed under.
type orthancInstance struct {
	MainDicomTags struct {
		SOPInstanceUID string `json:"SOPInstanceUID"`
	} `json:"MainDicomTags"`
}

// HasInstanceWithSOPUID reports whether Orthanc holds an instance whose SOP Instance UID
// equals sopInstanceUID. It walks the stored instances so the STOW leg can prove the
// exact instance it sent was persisted, not merely that some instance arrived.
func (c *Container) HasInstanceWithSOPUID(ctx context.Context, sopInstanceUID string) (bool, error) {
	var ids []string
	if err := c.getJSON(ctx, "/instances", &ids); err != nil {
		return false, err
	}
	for _, id := range ids {
		var inst orthancInstance
		if err := c.getJSON(ctx, "/instances/"+id, &inst); err != nil {
			return false, err
		}
		if inst.MainDicomTags.SOPInstanceUID == sopInstanceUID {
			return true, nil
		}
	}
	return false, nil
}

// getJSON issues a GET against the Orthanc REST API and decodes the JSON body into out.
func (c *Container) getJSON(ctx context.Context, path string, out any) error {
	url := c.HTTPBaseURL() + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("build request %s: %w", path, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GET %s: unexpected status %d: %s", path, resp.StatusCode, string(body))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}
