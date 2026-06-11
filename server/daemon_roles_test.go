package server

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"errors"
	"io"
	"iter"
	"math/big"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/hl7v2"
)

// TestDaemonMLLPRoundTrip starts a daemon with the MLLP role on loopback, sends one HL7 v2 message
// over MLLP, and asserts the configured handler's acknowledgement comes back, then shuts down cleanly.
func TestDaemonMLLPRoundTrip(t *testing.T) {
	t.Parallel()
	handler := hl7v2.HandlerFunc(func(_ context.Context, m *hl7v2.Message) (*hl7v2.Message, error) {
		return m.BuildACK(hl7v2.AckAccept)
	})
	mllpRole, err := NewMLLPRole(handler, WithMLLPPort(0))
	if err != nil {
		t.Fatalf("NewMLLPRole: %v", err)
	}
	d, err := New(WithMLLP(mllpRole))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	runCtx, cancelRun := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()
	waitForAddrs(t, d, "mllp")

	addr := d.Addrs()["mllp"]
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := hl7v2.NewClient(addr.String())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	msg, err := hl7v2.Parse([]byte(
		"MSH|^~\\&|SEND|FAC|RECV|FAC|20260101000000||ADT^A01^ADT_A01|MSG0001|P|2.5.1\r" +
			"EVN|A01|20260101000000\r" +
			"PID|1||MRN001^^^HOSP^MR||DOE^JOHN||19700101|M\r" +
			"PV1|1|I\r"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	ack, err := client.Send(ctx, msg)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if ack == nil {
		t.Fatal("nil acknowledgement from MLLP round-trip")
	}

	cancelRun()
	if err := <-runErr; err != nil {
		t.Fatalf("Run returned %v on clean shutdown, want nil", err)
	}
}

// TestMLLPNonLoopbackRequiresMTLS asserts the MLLP transport-auth fix: a non-loopback MLLP bind
// WITHOUT client-certificate-verifying TLS fails to start with ErrInsecureBind, while the same bind
// WITH mutual TLS starts cleanly (Finding 2). MLLP carries no application-level identity, so a
// generic Authenticator cannot gate it; the bind policy requires mTLS for network exposure.
func TestMLLPNonLoopbackRequiresMTLS(t *testing.T) {
	t.Parallel()
	handler := hl7v2.HandlerFunc(func(_ context.Context, m *hl7v2.Message) (*hl7v2.Message, error) {
		return m.BuildACK(hl7v2.AckAccept)
	})

	// Non-loopback MLLP bind with an Authenticator but NO mTLS: refused with ErrInsecureBind.
	noTLSRole, err := NewMLLPRole(handler, WithMLLPPort(0))
	if err != nil {
		t.Fatalf("NewMLLPRole: %v", err)
	}
	dNoTLS, err := New(WithMLLP(noTLSRole), WithBind("0.0.0.0"), WithAuthenticator(AllowAll()))
	if err != nil {
		t.Fatalf("New (no mTLS): %v", err)
	}
	if err := dNoTLS.Run(context.Background()); !errors.Is(err, ErrInsecureBind) {
		t.Fatalf("Run on a non-loopback MLLP bind without mTLS = %v, want ErrInsecureBind", err)
	}

	// Non-loopback MLLP bind WITH client-certificate-verifying TLS: starts cleanly.
	mtlsRole, err := NewMLLPRole(handler, WithMLLPPort(0))
	if err != nil {
		t.Fatalf("NewMLLPRole (mTLS): %v", err)
	}
	dMTLS, err := New(WithMLLP(mtlsRole), WithBind("0.0.0.0"),
		WithAuthenticator(AllowAll()), WithTLS(mutualTLSConfig(t)))
	if err != nil {
		t.Fatalf("New (mTLS): %v", err)
	}
	runCtx, cancelRun := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- dMTLS.Run(runCtx) }()
	waitForAddrs(t, dMTLS, "mllp")
	cancelRun()
	if err := <-runErr; err != nil {
		t.Fatalf("Run on a non-loopback MLLP bind with mTLS = %v, want a clean start/stop", err)
	}
}

// mutualTLSConfig builds a server TLS config that requires and verifies a client certificate
// (mTLS), so a non-loopback MLLP bind authenticates the peer at the transport. The CA pool trusts
// the same self-signed certificate the server presents, which is sufficient for the bind-policy
// check (the test does not complete a handshake).
func mutualTLSConfig(t *testing.T) *tls.Config {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "go-radx-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(leaf)
	return &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf}},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pool,
		MinVersion:   tls.VersionTLS12,
	}
}

// TestDaemonDICOMwebServes starts a daemon with the DICOMweb role on loopback and asserts the HTTP
// surface answers under the configured base path (a QIDO-RS studies search against the empty
// catalogue returns a non-5xx response), then shuts down cleanly. It exercises the role's HTTP
// listener, base-path mount, and auth middleware end-to-end.
func TestDaemonDICOMwebServes(t *testing.T) {
	t.Parallel()
	store, cat := newTestBackends(t)
	webRole, err := NewDICOMwebRole(store, cat, WithDICOMwebPort(0))
	if err != nil {
		t.Fatalf("NewDICOMwebRole: %v", err)
	}
	d, err := New(WithDICOMweb(webRole))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	runCtx, cancelRun := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()
	waitForAddrs(t, d, "dicomweb")

	addr := d.Addrs()["dicomweb"]
	url := "http://" + addr.String() + "/dicom-web/studies"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode >= 500 {
		t.Errorf("QIDO-RS studies search returned %d, want a non-5xx response", resp.StatusCode)
	}

	cancelRun()
	if err := <-runErr; err != nil {
		t.Fatalf("Run returned %v on clean shutdown, want nil", err)
	}
}

// startDICOMwebDaemon starts a daemon hosting the DICOMweb role over the given backends on loopback
// and returns the role's base URL. The daemon is shut down on test cleanup.
func startDICOMwebDaemon(t *testing.T, store ObjectStore, cat Catalogue) string {
	t.Helper()
	webRole, err := NewDICOMwebRole(store, cat, WithDICOMwebPort(0))
	if err != nil {
		t.Fatalf("NewDICOMwebRole: %v", err)
	}
	d, err := New(WithDICOMweb(webRole))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	runCtx, cancelRun := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()
	waitForAddrs(t, d, "dicomweb")
	t.Cleanup(func() {
		cancelRun()
		if err := <-runErr; err != nil {
			t.Errorf("Run returned %v on clean shutdown, want nil", err)
		}
	})
	return "http://" + d.Addrs()["dicomweb"].String() + "/dicom-web"
}

// dicomwebGET issues a GET with the given Accept header and returns the status and body.
func dicomwebGET(t *testing.T, url, accept string) (int, string, []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("NewRequest %s: %v", url, err)
	}
	req.Header.Set("Accept", accept)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body of %s: %v", url, err)
	}
	return resp.StatusCode, resp.Header.Get("Content-Type"), body
}

// multipartParts splits a multipart/related response body into its raw parts.
func multipartParts(t *testing.T, contentType string, body []byte) [][]byte {
	t.Helper()
	mt, params, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.HasPrefix(mt, "multipart/") {
		t.Fatalf("response Content-Type = %q, want multipart/related", contentType)
	}
	mr := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	var parts [][]byte
	for {
		p, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			return parts
		}
		if err != nil {
			t.Fatalf("NextPart: %v", err)
		}
		data, err := io.ReadAll(p)
		if err != nil {
			t.Fatalf("read part: %v", err)
		}
		parts = append(parts, data)
	}
}

// TestDaemonDICOMwebRetrievalRoutes is the daemon retrieval-wiring regression: the library DICOMweb
// server implements study/series/metadata/frames/bulkdata retrieval through optional retriever
// interfaces, but the daemon role only mounted instance retrieval, so these routes answered 501.
// Each route must now serve from the daemon's shared ObjectStore/Catalogue backends.
func TestDaemonDICOMwebRetrievalRoutes(t *testing.T) {
	t.Parallel()
	store, cat := newTestBackends(t)
	ctx := context.Background()

	const study = "20.1"
	const series1 = "20.1.1"
	const series2 = "20.1.2"
	const inst1 = "20.1.1.1"
	const inst2 = "20.1.2.1"

	// inst1 carries two 4-byte frames of native 8-bit pixel data; inst2 is a second series of the
	// same study with no pixel data.
	pixels := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	withPixels := newTestObject(study, series1, inst1)
	withPixels.Set(dicom.Element{Tag: dicom.TagRows, VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, 2)})
	withPixels.Set(dicom.Element{Tag: dicom.TagColumns, VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, 2)})
	withPixels.Set(dicom.Element{Tag: dicom.TagBitsAllocated, VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, 8)})
	withPixels.SetString(dicom.TagNumberOfFrames, "2")
	withPixels.Set(dicom.Element{Tag: dicom.TagPixelData, VR: dicom.VROB, Value: dicom.NewBytes(dicom.VROB, pixels)})
	other := newTestObject(study, series2, inst2)

	for _, ds := range []*dicom.DataSet{withPixels, other} {
		if err := store.Put(ctx, ds); err != nil {
			t.Fatalf("Put: %v", err)
		}
		if err := cat.Index(ctx, ds); err != nil {
			t.Fatalf("Index: %v", err)
		}
	}

	base := startDICOMwebDaemon(t, store, cat)
	const acceptDICOM = `multipart/related; type="application/dicom"`
	const acceptOctet = `multipart/related; type="application/octet-stream"`

	// WADO-RS study retrieval returns one application/dicom part per instance in the study.
	status, ct, body := dicomwebGET(t, base+"/studies/"+study, acceptDICOM)
	if status != http.StatusOK {
		t.Fatalf("study retrieval status = %d, want 200 (body: %s)", status, body)
	}
	if parts := multipartParts(t, ct, body); len(parts) != 2 {
		t.Errorf("study retrieval returned %d parts, want 2 (both instances)", len(parts))
	}

	// WADO-RS series retrieval scopes to the series.
	status, ct, body = dicomwebGET(t, base+"/studies/"+study+"/series/"+series1, acceptDICOM)
	if status != http.StatusOK {
		t.Fatalf("series retrieval status = %d, want 200 (body: %s)", status, body)
	}
	if parts := multipartParts(t, ct, body); len(parts) != 1 {
		t.Errorf("series retrieval returned %d parts, want 1", len(parts))
	}

	// WADO-RS metadata returns one DICOM-JSON object per instance at the requested level.
	status, _, body = dicomwebGET(t, base+"/studies/"+study+"/metadata", "application/dicom+json")
	if status != http.StatusOK {
		t.Fatalf("metadata retrieval status = %d, want 200 (body: %s)", status, body)
	}
	var metas []map[string]any
	if err := json.Unmarshal(body, &metas); err != nil {
		t.Fatalf("metadata body is not a JSON array: %v", err)
	}
	if len(metas) != 2 {
		t.Errorf("study metadata returned %d objects, want 2", len(metas))
	}

	// WADO-RS frames returns the requested 1-based frame's octets.
	instancePath := base + "/studies/" + study + "/series/" + series1 + "/instances/" + inst1
	status, ct, body = dicomwebGET(t, instancePath+"/frames/2", acceptOctet)
	if status != http.StatusOK {
		t.Fatalf("frame retrieval status = %d, want 200 (body: %s)", status, body)
	}
	if parts := multipartParts(t, ct, body); len(parts) != 1 || !bytes.Equal(parts[0], pixels[4:]) {
		t.Errorf("frame 2 parts = %v, want one part carrying the second frame %v", parts, pixels[4:])
	}

	// A frame outside the instance is 404, never a truncated payload.
	if status, _, _ = dicomwebGET(t, instancePath+"/frames/3", acceptOctet); status != http.StatusNotFound {
		t.Errorf("out-of-range frame status = %d, want 404", status)
	}

	// WADO-RS bulkdata returns the instance's binary values (here the pixel data).
	status, ct, body = dicomwebGET(t, instancePath+"/bulkdata", acceptOctet)
	if status != http.StatusOK {
		t.Fatalf("bulkdata retrieval status = %d, want 200 (body: %s)", status, body)
	}
	if parts := multipartParts(t, ct, body); len(parts) != 1 || !bytes.Equal(parts[0], pixels) {
		t.Errorf("bulkdata parts = %v, want one part carrying the pixel data", parts)
	}
}

// faultingCatalogue wraps a Catalogue whose Query always terminates with a backend error,
// standing in for a broken index (a failed SQLite read, a dropped connection).
type faultingCatalogue struct {
	Catalogue
}

func (f faultingCatalogue) Query(context.Context, CatalogueQuery) iter.Seq2[*dicom.DataSet, error] {
	return func(yield func(*dicom.DataSet, error) bool) {
		yield(nil, errors.New("catalogue fault"))
	}
}

// TestDaemonDICOMwebBackendFaultsAre500 is the fault-vs-absent regression for the WADO-RS
// retrieval surface: only a genuinely empty catalogue result answers 404. A catalogue fault, and
// a CATALOGUED instance the object store cannot produce (the store/catalogue inconsistency, the
// TOCTOU window), are backend faults answered 500 — never disguised as an absent resource
// (PRD §9.2).
func TestDaemonDICOMwebBackendFaultsAre500(t *testing.T) {
	t.Parallel()
	const acceptDICOM = `multipart/related; type="application/dicom"`

	// A catalogue fault during a study retrieval is 500.
	store, cat := newTestBackends(t)
	base := startDICOMwebDaemon(t, store, faultingCatalogue{Catalogue: cat})
	if status, _, body := dicomwebGET(t, base+"/studies/30.1", acceptDICOM); status != http.StatusInternalServerError {
		t.Errorf("catalogue-fault study retrieval status = %d, want 500 (body: %s)", status, body)
	}

	// A catalogued instance missing from the object store is an inconsistency, not a 404.
	store2, cat2 := newTestBackends(t)
	orphan := newTestObject("30.2", "30.2.1", "30.2.1.1")
	if err := cat2.Index(context.Background(), orphan); err != nil {
		t.Fatalf("Index: %v", err)
	}
	base2 := startDICOMwebDaemon(t, store2, cat2)
	if status, _, body := dicomwebGET(t, base2+"/studies/30.2", acceptDICOM); status != http.StatusInternalServerError {
		t.Errorf("catalogued-but-missing-object study retrieval status = %d, want 500 (body: %s)", status, body)
	}

	// Control: a study the catalogue genuinely knows nothing about stays 404.
	if status, _, body := dicomwebGET(t, base2+"/studies/30.999", acceptDICOM); status != http.StatusNotFound {
		t.Errorf("unknown-study retrieval status = %d, want 404 (body: %s)", status, body)
	}
	// Control: an unknown instance stays 404 (the store miss maps to the not-found sentinel).
	if status, _, body := dicomwebGET(t, base2+"/studies/30.999/series/30.999.1/instances/30.999.1.1", acceptDICOM); status != http.StatusNotFound {
		t.Errorf("unknown-instance retrieval status = %d, want 404 (body: %s)", status, body)
	}
}

// serverTLSConfig mints a self-signed server identity for 127.0.0.1 and returns a TLS config
// presenting it plus the pool that trusts it, for the HTTP-role TLS tests that complete a real
// handshake (unlike mutualTLSConfig, which only satisfies the bind-policy check).
func serverTLSConfig(t *testing.T) (*tls.Config, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		IsCA:         true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(leaf)
	return &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf}},
	}, pool
}

// TestHTTPRoleTLSFloorClampsCallerConfig asserts the TLS 1.2 floor WithTLS documents holds on the
// HTTP roles' listeners even when the caller pins a weaker floor: a daemon whose WithTLS config
// pins MinVersion = TLS 1.0 still rejects a TLS 1.1-limited client, because listen() clamps the
// floor up to 1.2. The control proves the same listener accepts a 1.2 client, so the rejection is
// the version, not a broken fixture.
func TestHTTPRoleTLSFloorClampsCallerConfig(t *testing.T) {
	t.Parallel()
	serverTLS, pool := serverTLSConfig(t)
	// Caller PINS a weak floor (TLS 1.0); listen() must clamp it up to 1.2.
	serverTLS.MinVersion = tls.VersionTLS10

	store, cat := newTestBackends(t)
	webRole, err := NewDICOMwebRole(store, cat, WithDICOMwebPort(0))
	if err != nil {
		t.Fatalf("NewDICOMwebRole: %v", err)
	}
	d, err := New(WithDICOMweb(webRole), WithTLS(serverTLS))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	runCtx, cancelRun := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()
	waitForAddrs(t, d, "dicomweb")
	defer func() {
		cancelRun()
		select {
		case <-runErr:
		case <-time.After(10 * time.Second):
			t.Error("Run did not return after cancel")
		}
	}()
	addr := d.Addrs()["dicomweb"].String()

	legacy := &tls.Config{
		RootCAs: pool, ServerName: "127.0.0.1",
		MinVersion: tls.VersionTLS10, MaxVersion: tls.VersionTLS11,
	}
	if conn, err := tls.Dial("tcp", addr, legacy); err == nil {
		_ = conn.Close()
		t.Fatal("a TLS 1.1-limited client completed a handshake with the HTTP role, want the clamped 1.2 floor to reject it")
	}

	// Control: a 1.2 client handshakes against the same listener.
	modern := &tls.Config{RootCAs: pool, ServerName: "127.0.0.1", MinVersion: tls.VersionTLS12}
	conn, err := tls.Dial("tcp", addr, modern)
	if err != nil {
		t.Fatalf("control 1.2 handshake against the clamped listener failed: %v", err)
	}
	_ = conn.Close()
}

// TestHTTPRolesSetReadHeaderTimeout asserts each HTTP role's http.Server carries the shared
// ReadHeaderTimeout, so a slowloris peer trickling header bytes cannot pin connections open
// indefinitely (gosec G112).
func TestHTTPRolesSetReadHeaderTimeout(t *testing.T) {
	t.Parallel()
	store, cat := newTestBackends(t)
	webRole, err := NewDICOMwebRole(store, cat, WithDICOMwebPort(0))
	if err != nil {
		t.Fatalf("NewDICOMwebRole: %v", err)
	}
	repo, err := NewMemoryRepository(fhir.R5)
	if err != nil {
		t.Fatalf("NewMemoryRepository: %v", err)
	}
	fhirRole, err := NewFHIRRole(repo, WithFHIRPort(0), WithFHIRRelease(fhir.R5))
	if err != nil {
		t.Fatalf("NewFHIRRole: %v", err)
	}
	d, err := New(WithDICOMweb(webRole), WithFHIR(fhirRole))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	runCtx, cancelRun := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()
	waitForAddrs(t, d, "dicomweb")
	waitForAddrs(t, d, fhirRole.name())
	cancelRun()
	select {
	case <-runErr:
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after cancel")
	}

	// The srv fields are read only after Run has returned: the roles assign them on the start
	// goroutine, so an in-flight read would race the daemon lifecycle the production code sequences.
	if got := webRole.srv.ReadHeaderTimeout; got != readHeaderTimeout {
		t.Errorf("dicomweb role ReadHeaderTimeout = %v, want %v", got, readHeaderTimeout)
	}
	if got := fhirRole.srv.ReadHeaderTimeout; got != readHeaderTimeout {
		t.Errorf("fhir role ReadHeaderTimeout = %v, want %v", got, readHeaderTimeout)
	}
}
