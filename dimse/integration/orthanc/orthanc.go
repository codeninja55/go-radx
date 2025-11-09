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

// OrthancContainer wraps a testcontainers Orthanc instance
type OrthancContainer struct {
	Container testcontainers.Container
	DICOMHost string
	DICOMPort string
	HTTPHost  string
	HTTPPort  string
}

// StartOrthanc starts an Orthanc PACS container for testing
func StartOrthanc(ctx context.Context) (*OrthancContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "orthancteam/orthanc:latest",
		ExposedPorts: []string{"4242/tcp", "8042/tcp"}, // DICOM and HTTP ports
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("8042/tcp"),
			wait.ForHTTP("/system").WithPort("8042/tcp").WithStartupTimeout(60*time.Second),
		),
		Env: map[string]string{
			"ORTHANC__DICOM_AET":                  "ORTHANC",
			"ORTHANC__DICOM_CHECK_CALLED_AET":     "false",
			"ORTHANC__AUTHENTICATION_ENABLED":     "false",
			"ORTHANC__DICOM_ALWAYS_ALLOW_ECHO":    "true",
			"ORTHANC__DICOM_ALWAYS_ALLOW_STORE":   "true",
			"ORTHANC__REMOTE_ACCESS_ALLOWED":      "true",
			"ORTHANC__UNKNOWN_SOP_CLASS_ACCEPTED": "true",
		},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start Orthanc container: %w", err)
	}

	// Get DICOM port mapping
	dicomHost, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DICOM host: %w", err)
	}

	dicomPort, err := container.MappedPort(ctx, "4242")
	if err != nil {
		return nil, fmt.Errorf("failed to get DICOM port: %w", err)
	}

	// Get HTTP port mapping
	httpHost, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP host: %w", err)
	}

	httpPort, err := container.MappedPort(ctx, "8042")
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP port: %w", err)
	}

	return &OrthancContainer{
		Container: container,
		DICOMHost: dicomHost,
		DICOMPort: dicomPort.Port(),
		HTTPHost:  httpHost,
		HTTPPort:  httpPort.Port(),
	}, nil
}

// Stop terminates the Orthanc container
func (oc *OrthancContainer) Stop(ctx context.Context) error {
	if oc.Container != nil {
		return oc.Container.Terminate(ctx)
	}
	return nil
}

// DICOMAddress returns the full DICOM address (host:port)
func (oc *OrthancContainer) DICOMAddress() string {
	return fmt.Sprintf("%s:%s", oc.DICOMHost, oc.DICOMPort)
}

// HTTPBaseURL returns the HTTP base URL
func (oc *OrthancContainer) HTTPBaseURL() string {
	return fmt.Sprintf("http://%s:%s", oc.HTTPHost, oc.HTTPPort)
}

// GetInstances retrieves all instances from Orthanc via REST API
func (oc *OrthancContainer) GetInstances(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("%s/instances", oc.HTTPBaseURL())

	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get instances: %w", err)
	}
	//nolint:errcheck // HTTP response body close in defer
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // Error reading body for error message is not critical
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var instances []string
	if err := parseJSON(resp.Body, &instances); err != nil {
		return nil, fmt.Errorf("failed to parse instances: %w", err)
	}

	return instances, nil
}

// GetStudies retrieves all studies from Orthanc via REST API
func (oc *OrthancContainer) GetStudies(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("%s/studies", oc.HTTPBaseURL())

	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get studies: %w", err)
	}
	//nolint:errcheck // HTTP response body close in defer
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // Error reading body for error message is not critical
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var studies []string
	if err := parseJSON(resp.Body, &studies); err != nil {
		return nil, fmt.Errorf("failed to parse studies: %w", err)
	}

	return studies, nil
}

// DeleteAllContent deletes all content from Orthanc
func (oc *OrthancContainer) DeleteAllContent(ctx context.Context) error {
	// Delete all instances
	instances, err := oc.GetInstances(ctx)
	if err != nil {
		return err
	}

	for _, instanceID := range instances {
		url := fmt.Sprintf("%s/instances/%s", oc.HTTPBaseURL(), instanceID)
		req, err := http.NewRequestWithContext(ctx, "DELETE", url, http.NoBody)
		if err != nil {
			return fmt.Errorf("failed to create delete request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to delete instance: %w", err)
		}
		//nolint:errcheck // HTTP response body close
		_ = resp.Body.Close()
	}

	return nil
}

// ConfigureModality configures a DICOM modality in Orthanc
func (oc *OrthancContainer) ConfigureModality(ctx context.Context, aet, host string, port int) error {
	url := fmt.Sprintf("%s/modalities/%s", oc.HTTPBaseURL(), aet)

	config := map[string]any{
		"AET":  aet,
		"Host": host,
		"Port": port,
	}

	body, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal modality config: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, io.NopCloser(newReader(body)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to configure modality: %w", err)
	}
	//nolint:errcheck // HTTP response body close in defer
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body) //nolint:errcheck // Error reading body for error message is not critical
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// SendToModality sends an instance to a configured modality
func (oc *OrthancContainer) SendToModality(ctx context.Context, modality, instanceID string) error {
	url := fmt.Sprintf("%s/modalities/%s/store", oc.HTTPBaseURL(), modality)

	body, err := json.Marshal([]string{instanceID})
	if err != nil {
		return fmt.Errorf("failed to marshal instance list: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, io.NopCloser(newReader(body)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send to modality: %w", err)
	}
	//nolint:errcheck // HTTP response body close in defer
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body) //nolint:errcheck // Error reading body for error message is not critical
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// parseJSON is a helper to parse JSON responses
func parseJSON(r io.Reader, v any) error {
	decoder := json.NewDecoder(r)
	return decoder.Decode(v)
}

// Helper to create reader from byte slice
type simpleReader struct {
	data []byte
	pos  int
}

func newReader(data []byte) *simpleReader {
	return &simpleReader{data: data}
}

func (r *simpleReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
