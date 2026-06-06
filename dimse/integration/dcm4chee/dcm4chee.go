//go:build interop

// Package dcm4chee provides a testcontainers-backed dcm4chee-arc PACS fixture for the DIMSE interop
// gate. It is built only under the interop tag so the testcontainers dependency stays out of the
// default build. dcm4chee-arc is a three-container stack: an OpenLDAP server holding the archive
// configuration, a PostgreSQL database, and the WildFly-hosted archive itself. The fixture starts
// all three on a shared network (the archive resolves its dependencies by the network aliases "ldap"
// and "db"), exposes the archive's DICOM (11112) and HTTP/REST (8080) ports, waits for WildFly to
// boot, and offers a small REST client used to verify a C-STORE landed (QIDO-RS) and to drive the
// archive to C-STORE out to an external SCP (device config plus a synchronous export).
package dcm4chee

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

// AETitle is the dcm4chee-arc archive's default DICOM AE Title (the "DCM4CHEE" AE listed by
// GET /dcm4chee-arc/aets), which an SCU names as the Called AE and which QIDO-RS and export
// requests are addressed to.
const AETitle = "DCM4CHEE"

// dicomPort and httpPort are the container-internal ports the archive binds: 11112 for DIMSE and
// 8080 for the HTTP/REST/QIDO API. testcontainers maps each to an ephemeral host port discovered
// after start. The archive's configured dicomHostname is 127.0.0.1 but its listener binds
// 0.0.0.0:11112 inside the container, so the mapped DICOM port is reachable from the host.
const (
	dicomPort = "11112/tcp"
	httpPort  = "8080/tcp"
)

// ldapImage, dbImage, and arcImage pin the three dcm4chee-arc container images by immutable digest so
// an interop run resolves the same bytes on every runner. The archive image is the PostgreSQL-backed
// WildFly application; the other two are its LDAP configuration store and its database, resolved by
// the ldapAlias and dbAlias network aliases below. The tag before each digest is the human-readable
// version; bumping any of them is a deliberate, reviewed change — re-resolve with `docker buildx
// imagetools inspect <image>:<version>` and update the digest and the tag together.
const (
	ldapImage = "dcm4che/slapd-dcm4chee:2.6.10-34.2@sha256:5c04ced61e943af2175c69ec7955c28bbaf676c50b5876c02e1437261cfaeefc"
	dbImage   = "dcm4che/postgres-dcm4chee:17.4-34@sha256:728c3055b894127c661a1645ffd0ccc2ad6461b86e957e245a9cc9bbc9c499e2"
	arcImage  = "dcm4che/dcm4chee-arc-psql:5.34.2@sha256:316eba283d3c8538e4c3b954edcdd59572f76aa7271ca6336ddf771703082f8c"
)

// ldapAlias and dbAlias are the network aliases the archive resolves its dependencies by. The
// archive's WILDFLY_WAIT_FOR and its baked-in configuration reference these exact hostnames, so the
// LDAP and database containers must be attached to the shared network under these aliases.
const (
	ldapAlias = "ldap"
	dbAlias   = "db"
)

// HostAccessHost is the hostname a container uses to reach a service bound on the Docker host. The
// host-gateway ExtraHost wired into the archive in Start makes it resolvable on plain Linux/CI as
// well as on Docker Desktop and OrbStack, so the archive can C-STORE out to a go-radx Server bound
// on the host.
const HostAccessHost = "host.docker.internal"

// arcStartupTimeout bounds the wait for WildFly to boot and answer GET /dcm4chee-arc/aets. The cold
// archive image is slow to start, so the timeout is generous.
const arcStartupTimeout = 4 * time.Minute

// Container wraps a started dcm4chee-arc stack: the three testcontainers instances, the network they
// share, and the host-side addresses the archive's mapped ports resolve to.
type Container struct {
	net       *testcontainers.DockerNetwork
	ldap      testcontainers.Container
	db        testcontainers.Container
	arc       testcontainers.Container
	dicomHost string
	dicomPort string
	httpHost  string
	httpPort  string
}

// Start brings up the dcm4chee-arc stack and blocks until the archive's REST API answers. It starts
// the LDAP and database containers first (waiting for their ports), then the archive, whose
// WILDFLY_WAIT_FOR also guards on the dependencies. On any failure it tears down whatever started so
// a partial stack never leaks. The archive is wired with the host-gateway ExtraHost so it can act as
// a C-STORE SCU against a go-radx Server bound on the host.
func Start(ctx context.Context) (c *Container, err error) {
	net, err := network.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create dcm4chee network: %w", err)
	}
	c = &Container{net: net}
	// Tear down everything started so far on any error path; cleared on success.
	defer func() {
		if err != nil {
			_ = c.Stop(context.Background())
			c = nil
		}
	}()

	c.ldap, err = startContainer(ctx, testcontainers.ContainerRequest{
		Image:          ldapImage,
		Networks:       []string{net.Name},
		NetworkAliases: map[string][]string{net.Name: {ldapAlias}},
		Env:            map[string]string{"STORAGE_DIR": "/storage/fs1"},
		WaitingFor:     wait.ForListeningPort("389/tcp").WithStartupTimeout(2 * time.Minute),
	})
	if err != nil {
		return nil, fmt.Errorf("start LDAP container: %w", err)
	}

	c.db, err = startContainer(ctx, testcontainers.ContainerRequest{
		Image:          dbImage,
		Networks:       []string{net.Name},
		NetworkAliases: map[string][]string{net.Name: {dbAlias}},
		Env: map[string]string{
			"POSTGRES_DB":       "pacsdb",
			"POSTGRES_USER":     "pacs",
			"POSTGRES_PASSWORD": "pacs",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(2 * time.Minute),
	})
	if err != nil {
		return nil, fmt.Errorf("start database container: %w", err)
	}

	c.arc, err = startContainer(ctx, testcontainers.ContainerRequest{
		Image:          arcImage,
		Networks:       []string{net.Name},
		NetworkAliases: map[string][]string{net.Name: {"arc"}},
		ExposedPorts:   []string{dicomPort, httpPort},
		Env: map[string]string{
			"POSTGRES_DB":       "pacsdb",
			"POSTGRES_USER":     "pacs",
			"POSTGRES_PASSWORD": "pacs",
			"WILDFLY_CHOWN":     "/storage",
			"WILDFLY_WAIT_FOR":  ldapAlias + ":389 " + dbAlias + ":5432",
		},
		// The archive entrypoint touches /storage/chown.done before WildFly boots; without a writable
		// /storage it exits 1 and the HTTP port never opens. A tmpfs satisfies the entrypoint without
		// needing a host volume.
		Tmpfs: map[string]string{"/storage": "rw"},
		WaitingFor: wait.ForAll(
			wait.ForHTTP("/dcm4chee-arc/aets").WithPort(httpPort),
			wait.ForListeningPort(dicomPort),
		).WithDeadline(arcStartupTimeout),
		// Make the Docker host reachable from inside the archive as host.docker.internal on every
		// platform, so the archive can C-STORE out to a go-radx Server bound on the host.
		HostConfigModifier: func(hc *dockercontainer.HostConfig) {
			hc.ExtraHosts = append(hc.ExtraHosts, HostAccessHost+":host-gateway")
		},
	})
	if err != nil {
		return nil, fmt.Errorf("start archive container: %w", err)
	}

	// The archive entrypoint can exit 1 (e.g. an unwritable /storage) without the HTTP wait ever
	// firing on plain Linux; assert it is still running before handing the stack back.
	state, err := c.arc.State(ctx)
	if err != nil {
		return nil, fmt.Errorf("inspect archive state: %w", err)
	}
	if !state.Running {
		return nil, fmt.Errorf("archive container is not running (state %q)", state.Status)
	}

	host, err := c.arc.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve archive host: %w", err)
	}
	dp, err := c.arc.MappedPort(ctx, dicomPort)
	if err != nil {
		return nil, fmt.Errorf("resolve mapped DICOM port: %w", err)
	}
	hp, err := c.arc.MappedPort(ctx, httpPort)
	if err != nil {
		return nil, fmt.Errorf("resolve mapped HTTP port: %w", err)
	}
	c.dicomHost = host
	c.dicomPort = dp.Port()
	c.httpHost = host
	c.httpPort = hp.Port()
	return c, nil
}

// startContainer starts a single container from the given request, returning the started instance.
func startContainer(ctx context.Context, req testcontainers.ContainerRequest) (testcontainers.Container, error) {
	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}

// Stop terminates the stack's containers and removes the shared network. It tolerates a partially
// started or nil stack so it is safe to call from an error path or a test cleanup.
func (c *Container) Stop(ctx context.Context) error {
	if c == nil {
		return nil
	}
	var firstErr error
	record := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if c.arc != nil {
		record(c.arc.Terminate(ctx))
	}
	if c.db != nil {
		record(c.db.Terminate(ctx))
	}
	if c.ldap != nil {
		record(c.ldap.Terminate(ctx))
	}
	if c.net != nil {
		record(c.net.Remove(ctx))
	}
	return firstErr
}

// DICOMAddr returns the host:port the SCU dials for DIMSE against the archive.
func (c *Container) DICOMAddr() string {
	return fmt.Sprintf("%s:%s", c.dicomHost, c.dicomPort)
}

// HTTPBaseURL returns the base URL of the archive's HTTP/REST API.
func (c *Container) HTTPBaseURL() string {
	return fmt.Sprintf("http://%s:%s", c.httpHost, c.httpPort)
}

// dicomJSONAttr is one attribute of a DICOM-JSON object as returned by QIDO-RS: a VR plus the value
// array. The interop gate reads the SOP Instance UID attribute's first value.
type dicomJSONAttr struct {
	VR    string `json:"vr"`
	Value []any  `json:"Value"`
}

// sopInstanceUIDTag is the DICOM-JSON key for (0008,0018) SOP Instance UID, the attribute QIDO-RS
// returns the stored instance's UID under.
const sopInstanceUIDTag = "00080018"

// HasInstanceWithSOPUID reports whether the archive holds an instance whose SOP Instance UID equals
// sopInstanceUID, by issuing a QIDO-RS instance query filtered on that UID and confirming the
// returned DICOM-JSON object echoes the same UID in (0008,0018). It lets the C-STORE leg prove the
// exact instance it sent was indexed, not merely that some instance arrived. The archive indexes
// asynchronously, so callers poll this with a deadline.
func (c *Container) HasInstanceWithSOPUID(ctx context.Context, sopInstanceUID string) (bool, error) {
	path := fmt.Sprintf(
		"/dcm4chee-arc/aets/%s/rs/instances?SOPInstanceUID=%s",
		AETitle, sopInstanceUID,
	)
	var results []map[string]dicomJSONAttr
	if err := c.getJSON(ctx, path, &results); err != nil {
		return false, err
	}
	for _, attrs := range results {
		uid := dicomJSONFirstString(attrs[sopInstanceUIDTag])
		if uid == sopInstanceUID {
			return true, nil
		}
	}
	return false, nil
}

// dicomJSONFirstString returns the first value of a DICOM-JSON attribute as a string, or "" when the
// attribute has no string value.
func dicomJSONFirstString(attr dicomJSONAttr) string {
	if len(attr.Value) == 0 {
		return ""
	}
	s, _ := attr.Value[0].(string)
	return s
}

// StoreInstance stores a DICOM instance (raw Part 10 bytes) into the archive via STOW-RS, so the
// archive holds a study to query or forward. It wraps the instance in a multipart/related body as
// the STOW-RS transaction requires and returns on the success status.
func (c *Container) StoreInstance(ctx context.Context, instance []byte) error {
	const boundary = "radxinteropboundary"
	var body bytes.Buffer
	fmt.Fprintf(&body, "--%s\r\nContent-Type: application/dicom\r\n\r\n", boundary)
	body.Write(instance)
	fmt.Fprintf(&body, "\r\n--%s--\r\n", boundary)

	url := c.HTTPBaseURL() + fmt.Sprintf("/dcm4chee-arc/aets/%s/rs/studies", AETitle)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return fmt.Errorf("build STOW-RS request: %w", err)
	}
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/related;type=\"application/dicom\";boundary=%s", boundary))
	req.Header.Set("Accept", "application/dicom+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("STOW-RS store: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("STOW-RS store: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// deviceConfig is the dcm4chee-arc device configuration document. The field order is load-bearing:
// the archive resolves an AE's dicomNetworkConnectionReference JSON pointer against
// dicomNetworkConnection as it parses, so the connection list MUST appear before the AE list. A
// map[string]any would marshal keys alphabetically (NetworkAE before NetworkConnection) and the
// archive would throw an out-of-bounds error resolving "/dicomNetworkConnection/0" against a
// not-yet-parsed list; a struct preserves declaration order and avoids that.
type deviceConfig struct {
	DeviceName        string             `json:"dicomDeviceName"`
	Installed         bool               `json:"dicomInstalled"`
	NetworkConnection []deviceConnection `json:"dicomNetworkConnection"`
	NetworkAE         []deviceNetworkAE  `json:"dicomNetworkAE"`
}

type deviceConnection struct {
	CN       string `json:"cn"`
	Hostname string `json:"dicomHostname"`
	Port     int    `json:"dicomPort"`
}

type deviceNetworkAE struct {
	AETitle              string   `json:"dicomAETitle"`
	AssociationInitiator bool     `json:"dicomAssociationInitiator"`
	AssociationAcceptor  bool     `json:"dicomAssociationAcceptor"`
	ConnectionReference  []string `json:"dicomNetworkConnectionReference"`
}

// ConfigureDestinationAE registers a remote DICOM destination (AE Title, host, port) as a configured
// device in the archive and reloads the configuration so a subsequent export can resolve the AE. The
// device carries no transfer capability: the archive proposes the presentation contexts as the
// initiating SCU, and a wildcard SOP class is rejected by the LDAP schema. ExportStudy drives the
// archive to C-STORE to this destination.
func (c *Container) ConfigureDestinationAE(ctx context.Context, aet, host string, port int) error {
	device := deviceConfig{
		DeviceName: "radx-" + aet,
		Installed:  true,
		NetworkConnection: []deviceConnection{
			{CN: "dicom", Hostname: host, Port: port},
		},
		NetworkAE: []deviceNetworkAE{
			{
				AETitle:              aet,
				AssociationInitiator: false,
				AssociationAcceptor:  true,
				ConnectionReference:  []string{"/dicomNetworkConnection/0"},
			},
		},
	}
	body, err := json.Marshal(device)
	if err != nil {
		return fmt.Errorf("marshal device config: %w", err)
	}
	if err := c.postWithRetry(ctx, "/dcm4chee-arc/devices/radx-"+aet, "application/json", body); err != nil {
		return err
	}
	if err := c.post(ctx, "/dcm4chee-arc/ctrl/reload", "application/json", nil, nil); err != nil {
		return fmt.Errorf("reload archive configuration: %w", err)
	}
	return nil
}

// configWriteRetries and configWriteBackoff bound the retry of a config write against a freshly
// booted archive, whose LDAP config-write path can briefly fault with a 5xx after the read path is
// already answering. The window is short, so a handful of attempts with a small backoff suffices.
const (
	configWriteRetries = 10
	configWriteBackoff = 2 * time.Second
)

// postWithRetry issues a POST and retries only on a transient server-side fault (5xx), so a
// just-booted archive's LDAP write race clears without masking a permanent request error (4xx) or a
// context cancellation. It returns the last error if every attempt faults transiently.
func (c *Container) postWithRetry(ctx context.Context, path, contentType string, body []byte) error {
	var lastErr error
	for attempt := 0; attempt < configWriteRetries; attempt++ {
		err := c.post(ctx, path, contentType, body, nil)
		if err == nil {
			return nil
		}
		var statusErr *httpStatusError
		if !errors.As(err, &statusErr) || !statusErr.transient() {
			return err
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(configWriteBackoff):
		}
	}
	return fmt.Errorf("config write %s failed after %d attempts: %w", path, configWriteRetries, lastErr)
}

// ExportStudy drives the archive to C-STORE the study identified by studyInstanceUID to the named
// destination AE (registered via ConfigureDestinationAE), synchronously so the call returns only
// once the transfer has completed. It is the foreign-SCU side of the SCP interop gate: the archive
// is the C-STORE SCU against an external SCP (a go-radx Server). It returns the count of instances
// the archive reports as completed.
func (c *Container) ExportStudy(ctx context.Context, studyInstanceUID, destAET string) (int, error) {
	path := fmt.Sprintf(
		"/dcm4chee-arc/aets/%s/rs/studies/%s/export/dicom:%s",
		AETitle, studyInstanceUID, destAET,
	)
	var reply struct {
		Completed int `json:"completed"`
		Failed    int `json:"failed"`
		Warning   int `json:"warning"`
	}
	if err := c.post(ctx, path, "application/json", nil, &reply); err != nil {
		return 0, err
	}
	if reply.Failed > 0 {
		return reply.Completed, fmt.Errorf("export study to %s: %d instance(s) failed", destAET, reply.Failed)
	}
	return reply.Completed, nil
}

// WorklistItem is the minimal set of Modality Worklist attributes CreateWorklistItem seeds into the
// archive so an MWL C-FIND has a scheduled procedure step to return. The values are synthetic test
// fixtures supplied by the interop test; this struct carries no real patient data.
type WorklistItem struct {
	PatientID                string
	PatientName              string // DICOM PN Alphabetic group, e.g. "DOE^JANE"
	StudyInstanceUID         string
	AccessionNumber          string
	RequestedProcedureID     string
	ScheduledStationAETitle  string
	ScheduledProcedureStepID string
	Modality                 string
	ScheduledStartDate       string // DICOM DA, e.g. "20260606"
	ScheduledStartTime       string // DICOM TM, e.g. "090000"
}

// dicomJSONValue is a writable DICOM-JSON attribute (VR plus a value array), the mirror of the
// dicomJSONAttr read type. A nested sequence carries its items in the same Value array, each a
// DICOM-JSON object keyed by tag.
type dicomJSONValue struct {
	VR    string `json:"vr"`
	Value []any  `json:"Value,omitempty"`
}

// CreateWorklistItem seeds a single Modality Worklist item into the archive's MWL SCP via its REST
// API (POST .../rs/mwlitems, application/dicom+json), so a subsequent MWL C-FIND has a scheduled
// procedure step to match. The item carries a top-level Study Instance UID, Accession Number, and
// Requested Procedure ID plus a Scheduled Procedure Step Sequence (0040,0100) item holding the
// Modality, Scheduled Station AE Title, Scheduled Procedure Step ID, and start date/time — the keys
// a modality matches on. The attribute values are synthetic test fixtures, never real patient data.
func (c *Container) CreateWorklistItem(ctx context.Context, item WorklistItem) error {
	doc := map[string]dicomJSONValue{
		"00100010": {VR: "PN", Value: []any{map[string]string{"Alphabetic": item.PatientName}}},
		"00100020": {VR: "LO", Value: []any{item.PatientID}},
		"00080050": {VR: "SH", Value: []any{item.AccessionNumber}},
		"0020000D": {VR: "UI", Value: []any{item.StudyInstanceUID}},
		"00401001": {VR: "SH", Value: []any{item.RequestedProcedureID}},
		"00321060": {VR: "LO", Value: []any{"Interop MWL Procedure"}},
		"00400100": {VR: "SQ", Value: []any{
			map[string]dicomJSONValue{
				"00080060": {VR: "CS", Value: []any{item.Modality}},
				"00400001": {VR: "AE", Value: []any{item.ScheduledStationAETitle}},
				"00400002": {VR: "DA", Value: []any{item.ScheduledStartDate}},
				"00400003": {VR: "TM", Value: []any{item.ScheduledStartTime}},
				"00400007": {VR: "LO", Value: []any{"Interop Scheduled Step"}},
				"00400009": {VR: "SH", Value: []any{item.ScheduledProcedureStepID}},
			},
		}},
	}
	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal MWL item: %w", err)
	}
	path := fmt.Sprintf("/dcm4chee-arc/aets/%s/rs/mwlitems", AETitle)
	if err := c.post(ctx, path, "application/dicom+json", body, nil); err != nil {
		return fmt.Errorf("create MWL item: %w", err)
	}
	return nil
}

// getJSON issues a GET against the archive REST API and decodes the JSON body into out. A QIDO-RS
// query with zero matches answers 204 No Content per the DICOMweb standard, which is a valid empty
// result, not an error: getJSON leaves out at its zero value and returns nil, so the asynchronous
// indexing poll treats a not-yet-indexed instance as a miss and keeps waiting rather than failing.
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
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GET %s: unexpected status %d: %s", path, resp.StatusCode, string(body))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

// post issues a POST with the given body and content type against the archive REST API, decoding the
// JSON response into out when out is non-nil. A body of nil sends an empty request body, which the
// device-create, reload, and export endpoints accept. The reload and device-create endpoints reply
// 204 No Content; both 200 and 204 are treated as success.
func (c *Container) post(ctx context.Context, path, contentType string, body []byte, out any) error {
	url := c.HTTPBaseURL() + path
	var reader io.Reader = http.NoBody
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, reader)
	if err != nil {
		return fmt.Errorf("build request %s: %w", path, err)
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		statusErr := &httpStatusError{path: path, status: resp.StatusCode, body: string(respBody)}
		return statusErr
	}
	if out != nil && resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode %s: %w", path, err)
		}
	}
	return nil
}

// httpStatusError reports an unexpected HTTP status from an archive REST call. It carries the status
// code so callers can distinguish a transient server-side fault (5xx, retryable) from a permanent
// request fault (4xx, not retryable). It never embeds patient data; the body it quotes is an archive
// error message, not instance content.
type httpStatusError struct {
	path   string
	status int
	body   string
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("POST %s: unexpected status %d: %s", e.path, e.status, e.body)
}

// transient reports whether the status is a server-side fault worth retrying (any 5xx). A freshly
// booted archive answers GET /aets (read path) before its LDAP config-write path is ready, so the
// first device-create can fault with a 5xx that a short retry clears.
func (e *httpStatusError) transient() bool {
	return e.status >= http.StatusInternalServerError
}
