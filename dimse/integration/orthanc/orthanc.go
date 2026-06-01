//go:build interop

// Package orthanc provides a testcontainers-backed Orthanc PACS fixture for the DIMSE interop
// gate. It is built only under the interop tag so the testcontainers dependency stays out of the
// default build. The fixture starts an Orthanc container, exposes its DICOM (4242) and REST API
// (8042) ports, waits for readiness, and offers a small REST client used to verify that a
// C-STORE actually landed (the regression that proves the prototype's Orthanc abort is fixed).
package orthanc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// AETitle is the Orthanc container's DICOM AE Title. The container is configured to accept any
// Called AE Title (DICOM_CHECK_CALLED_AET=false), but an SCU still names this as the Called AE.
const AETitle = "ORTHANC"

// dicomPort and httpPort are the container-internal ports Orthanc binds: 4242 for DIMSE and 8042
// for the REST API. testcontainers maps each to an ephemeral host port discovered after start.
const (
	dicomPort = "4242/tcp"
	httpPort  = "8042/tcp"
)

// image pins the Orthanc container image. orthancteam/orthanc is the maintained upstream image and
// honours the ORTHANC__* environment overrides used below.
const image = "orthancteam/orthanc:latest"

// Container wraps a started Orthanc testcontainers instance and the host-side addresses its mapped
// ports resolve to.
type Container struct {
	container testcontainers.Container
	dicomHost string
	dicomPort string
	httpHost  string
	httpPort  string
}

// Start launches an Orthanc container and blocks until its REST API answers. The container accepts
// any C-ECHO and C-STORE without authentication so the interop SCU can drive it directly; remote
// access is enabled so the host-side test can reach the mapped REST port.
func Start(ctx context.Context) (*Container, error) {
	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{dicomPort, httpPort},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort(dicomPort),
			wait.ForHTTP("/system").WithPort(httpPort).WithStartupTimeout(120*time.Second),
		),
		Env: map[string]string{
			"ORTHANC__DICOM_AET":                  AETitle,
			"ORTHANC__DICOM_CHECK_CALLED_AET":     "false",
			"ORTHANC__AUTHENTICATION_ENABLED":     "false",
			"ORTHANC__DICOM_ALWAYS_ALLOW_ECHO":    "true",
			"ORTHANC__DICOM_ALWAYS_ALLOW_STORE":   "true",
			"ORTHANC__REMOTE_ACCESS_ALLOWED":      "true",
			"ORTHANC__UNKNOWN_SOP_CLASS_ACCEPTED": "true",
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
	dp, err := c.MappedPort(ctx, dicomPort)
	if err != nil {
		return nil, fmt.Errorf("resolve mapped DICOM port: %w", err)
	}
	hp, err := c.MappedPort(ctx, httpPort)
	if err != nil {
		return nil, fmt.Errorf("resolve mapped HTTP port: %w", err)
	}

	return &Container{
		container: c,
		dicomHost: host,
		dicomPort: dp.Port(),
		httpHost:  host,
		httpPort:  hp.Port(),
	}, nil
}

// Stop terminates the container. It is safe to call on a nil container.
func (c *Container) Stop(ctx context.Context) error {
	if c == nil || c.container == nil {
		return nil
	}
	return c.container.Terminate(ctx)
}

// DICOMAddr returns the host:port the SCU dials for DIMSE.
func (c *Container) DICOMAddr() string {
	return fmt.Sprintf("%s:%s", c.dicomHost, c.dicomPort)
}

// HTTPBaseURL returns the base URL of the Orthanc REST API.
func (c *Container) HTTPBaseURL() string {
	return fmt.Sprintf("http://%s:%s", c.httpHost, c.httpPort)
}

// orthancInstance is the subset of an Orthanc /instances/{id} response the interop gate inspects:
// the main DICOM tags carry the SOP Instance UID a C-STORE landed under.
type orthancInstance struct {
	MainDicomTags struct {
		SOPInstanceUID string `json:"SOPInstanceUID"`
	} `json:"MainDicomTags"`
}

// InstanceIDs lists every Orthanc-internal instance identifier currently stored.
func (c *Container) InstanceIDs(ctx context.Context) ([]string, error) {
	var ids []string
	if err := c.getJSON(ctx, "/instances", &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// HasInstanceWithSOPUID reports whether Orthanc holds an instance whose SOP Instance UID equals
// sopInstanceUID. It walks the stored instances and matches on the main DICOM tags, so the interop
// C-STORE leg can prove the exact instance it sent was actually persisted, not merely that some
// instance arrived.
func (c *Container) HasInstanceWithSOPUID(ctx context.Context, sopInstanceUID string) (bool, error) {
	ids, err := c.InstanceIDs(ctx)
	if err != nil {
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

// ConfigureModality registers a remote DICOM modality (AE Title, host, port) in Orthanc via its
// REST API, so Orthanc can be driven to C-STORE to an external SCP (e.g. a go-radx Server).
func (c *Container) ConfigureModality(ctx context.Context, aet, host string, port int) error {
	body, err := json.Marshal(map[string]any{"AET": aet, "Host": host, "Port": port})
	if err != nil {
		return fmt.Errorf("marshal modality config: %w", err)
	}
	return c.put(ctx, "/modalities/"+aet, body)
}

// put issues a PUT with a JSON body against the Orthanc REST API.
func (c *Container) put(ctx context.Context, path string, body []byte) error {
	url := c.HTTPBaseURL() + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request %s: %w", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("PUT %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("PUT %s: unexpected status %d: %s", path, resp.StatusCode, string(respBody))
	}
	return nil
}
