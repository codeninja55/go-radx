//go:build unix

// Package credsweep is the library-wide credential-leak sanity harness (PRD §9.8),
// the sibling of internal/phisweep: where phisweep proves no patient value reaches a
// sink at default verbosity, credsweep proves no authentication secret does. It
// drives each credential-bearing surface — DIMSE user-identity negotiation, the
// DICOMweb client's Authorization schemes, the FHIR REST client's bearer token, and
// the server daemon's HTTP auth middleware — with synthetic credential sentinels,
// then scans the same four sinks (stdout, stderr, returned error strings, the
// structured log) for any sentinel. A single appearance is a failure.
package credsweep

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicomweb"
	"github.com/codeninja55/go-radx/dimse"
	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/rest"
	"github.com/codeninja55/go-radx/internal/phisweep"
	"github.com/codeninja55/go-radx/server"
)

// The sentinels below are deliberately synthetic and never real credentials. Each is
// distinctive enough that an incidental substring match is implausible, yet flows
// through the same fields a real secret would, so a leak through an error-formatting
// or logging path is caught exactly as a real credential would be.
const (
	credUsername = "ZZZCRED-USERNAME-SENTINEL"
	credPasscode = "ZZZCRED-PASSCODE-SENTINEL"
	credBearer   = "ZZZCRED-BEARER-TOKEN-SENTINEL"
	credJWT      = "eyJZZZCRED-JWT-SENTINEL-HEADER.ZZZCRED-JWT-SENTINEL-BODY.ZZZSIG"
)

// credSentinels is every token the sweep scans for, including the base64 form a
// Basic Authorization header carries on the wire: a middleware or client that echoes
// the raw header would leak that encoding rather than the cleartext.
func credSentinels() []string {
	basic := base64.StdEncoding.EncodeToString([]byte(credUsername + ":" + credPasscode))
	return []string{credUsername, credPasscode, credBearer, credJWT, basic}
}

// nopEcho is the minimal DIMSE handler the rejecting SCP carries; authentication
// fails before any service runs, so it is never invoked.
type nopEcho struct{}

func (nopEcho) Echo(context.Context, dimse.OpInfo) dimse.Status { return dimse.StatusEchoSuccess }

// exerciseDIMSEUserIdentity presents username/passcode and JWT user-identity items
// (PS3.7 D.3.3.7) carrying credential sentinels to an SCP whose authenticator
// rejects every association. The rejection travels back through the ACSE machinery
// to the SCU as a typed error; that error string, and everything the server logged,
// must carry no identity field.
func exerciseDIMSEUserIdentity(t *testing.T, ctx context.Context) []error {
	var errs []error

	scpAE, err := dimse.NewAE(dimse.AETitle("CRED-SCP"))
	if err != nil {
		t.Errorf("new SCP AE: %v", err)
		return nil
	}
	contexts := dimse.VerificationContexts()
	for i := range contexts {
		contexts[i].ID = uint8(1 + 2*i)
	}
	// The authenticator records every presented identity so the exercise can prove
	// the sentinels actually crossed the wire; an authenticator that never sees them
	// would make the non-leak scan vacuous.
	var (
		presentedMu sync.Mutex
		presented   []dimse.UserIdentity
	)
	srv := dimse.NewServer(scpAE, contexts, nopEcho{},
		dimse.WithAuthenticator(func(id *dimse.UserIdentity, _ net.Addr) ([]byte, error) {
			if id != nil {
				presentedMu.Lock()
				presented = append(presented, *id)
				presentedMu.Unlock()
			}
			return nil, errors.New("credentials rejected")
		}))
	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(ctx, "127.0.0.1:0") }()
	deadline := time.Now().Add(5 * time.Second)
	for srv.Addr() == nil && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if srv.Addr() == nil {
		t.Error("dimse server never bound")
		return nil
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		<-served
	}()

	scu, err := dimse.NewAE(dimse.AETitle("CRED-SCU"))
	if err != nil {
		t.Errorf("new SCU AE: %v", err)
		return nil
	}
	identities := []dimse.UserIdentity{
		{
			Type:                      dimse.UserIdentityUsernamePasscode,
			PrimaryField:              []byte(credUsername),
			SecondaryField:            []byte(credPasscode),
			PositiveResponseRequested: true,
		},
		{
			Type:         dimse.UserIdentityJWT,
			PrimaryField: []byte(credJWT),
		},
	}
	for _, id := range identities {
		assocCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		_, err := scu.Associate(assocCtx, srv.Addr().String(), dimse.AETitle("CRED-SCP"),
			contexts, dimse.WithUserIdentity(id))
		cancel()
		if err == nil {
			t.Error("rejecting SCP accepted the association")
			continue
		}
		errs = append(errs, err)
	}

	// Prove both sentinel identities crossed the wire intact; without this the
	// rejection errors above could be clean simply because nothing was sent.
	presentedMu.Lock()
	var sawUsername, sawJWT bool
	for _, id := range presented {
		if id.Type == dimse.UserIdentityUsernamePasscode &&
			string(id.PrimaryField) == credUsername && string(id.SecondaryField) == credPasscode {
			sawUsername = true
		}
		if id.Type == dimse.UserIdentityJWT && string(id.PrimaryField) == credJWT {
			sawJWT = true
		}
	}
	presentedMu.Unlock()
	if !sawUsername {
		t.Error("username/passcode identity never reached the SCP authenticator")
	}
	if !sawJWT {
		t.Error("JWT identity never reached the SCP authenticator")
	}
	return errs
}

// rejecting401 returns an httptest origin that records the last Authorization
// header it saw and answers every request 401, the path that tempts a client into
// echoing its credential in the resulting error.
func rejecting401(lastAuth *atomic.Value) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastAuth.Store(r.Header.Get("Authorization"))
		w.Header().Set("WWW-Authenticate", "Bearer")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
}

// exerciseDICOMwebClient drives the DICOMweb client's Basic and Bearer Authorization
// schemes against a 401-rejecting origin. The credential must reach the wire (the
// exercise is vacuous otherwise) and the returned client errors must not carry it.
func exerciseDICOMwebClient(t *testing.T, ctx context.Context) []error {
	var errs []error
	var lastAuth atomic.Value
	hs := rejecting401(&lastAuth)
	defer hs.Close()

	basic := "Basic " + base64.StdEncoding.EncodeToString([]byte(credUsername+":"+credPasscode))
	schemes := []struct {
		name string
		opt  dicomweb.ClientOption
		want string
	}{
		{"basic", dicomweb.WithBasicAuth(credUsername, credPasscode), basic},
		{"bearer", dicomweb.WithBearerToken(credBearer), "Bearer " + credBearer},
	}
	for _, scheme := range schemes {
		lastAuth.Store("")
		c, err := dicomweb.NewClient(hs.URL, scheme.opt)
		if err != nil {
			t.Errorf("new dicomweb client (%s): %v", scheme.name, err)
			continue
		}
		if _, err := c.RetrieveMetadata(ctx, dicomweb.NewStudy(dicom.UID("1.2.3"))); err != nil {
			errs = append(errs, err)
		} else {
			t.Errorf("401 origin produced no client error (%s)", scheme.name)
		}
		// Exact-match the header at the origin: a transformed or redacted credential
		// would make the non-leak assertion vacuous.
		if got, _ := lastAuth.Load().(string); got != scheme.want {
			t.Errorf("origin saw Authorization %q, want the %s sentinel header", got, scheme.name)
		}
	}
	return errs
}

// exerciseFHIRRESTClient drives the FHIR REST client's bearer token against a
// 401-rejecting origin; the returned error must not carry the token.
func exerciseFHIRRESTClient(t *testing.T, ctx context.Context) []error {
	var errs []error
	var lastAuth atomic.Value
	hs := rejecting401(&lastAuth)
	defer hs.Close()

	c, err := rest.NewClient(fhir.R5, hs.URL, rest.WithBearerToken(credBearer))
	if err != nil {
		t.Errorf("new fhir rest client: %v", err)
		return nil
	}
	if _, err := c.Read(ctx, "Patient", "example"); err != nil {
		errs = append(errs, err)
	} else {
		t.Error("401 origin produced no client error")
	}
	if got, _ := lastAuth.Load().(string); got != "Bearer "+credBearer {
		t.Errorf("origin saw Authorization %q, want the bearer sentinel header", got)
	}
	return errs
}

// rejectAll is an Authenticator that rejects every request, so the daemon's auth
// middleware exercises its failure logging with a sentinel-bearing Authorization
// header in hand. It records the last Authorization header it saw so the exercise
// can prove the sentinel actually reached the authenticator.
type rejectAll struct {
	lastAuth *atomic.Value
}

func (r rejectAll) AuthenticateHTTP(_ context.Context, req *http.Request) (server.Principal, error) {
	if r.lastAuth != nil {
		r.lastAuth.Store(req.Header.Get("Authorization"))
	}
	return server.Principal{}, errors.New("credentials rejected")
}

func (rejectAll) AuthenticateDIMSE(context.Context, dimse.AETitle) (server.Principal, error) {
	return server.Principal{}, errors.New("credentials rejected")
}

// exerciseServerAuthMiddleware sends sentinel-bearing Bearer and Basic Authorization
// headers at a daemon FHIR role behind a rejecting Authenticator, then surfaces each
// 401 response body as an error string so the sweep scans it. The daemon logs
// through the supplied zap logger; the caller scans those observed entries.
func exerciseServerAuthMiddleware(t *testing.T, ctx context.Context, logger *zap.Logger) []error {
	var errs []error

	repo, err := server.NewMemoryRepository(fhir.R5)
	if err != nil {
		t.Errorf("new memory repository: %v", err)
		return nil
	}
	role, err := server.NewFHIRRole(repo, server.WithFHIRPort(0), server.WithFHIRRelease(fhir.R5))
	if err != nil {
		t.Errorf("new fhir role: %v", err)
		return nil
	}
	var lastAuth atomic.Value
	d, err := server.New(server.WithFHIR(role), server.WithLogger(logger),
		server.WithAuthenticator(rejectAll{lastAuth: &lastAuth}))
	if err != nil {
		t.Errorf("new daemon: %v", err)
		return nil
	}
	runCtx, cancelRun := context.WithCancel(ctx)
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()
	defer func() {
		cancelRun()
		select {
		case <-runErr:
		case <-time.After(10 * time.Second):
		}
	}()
	var addr string
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, a := range d.Addrs() {
			addr = a.String()
		}
		if addr != "" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if addr == "" {
		t.Error("daemon never reported a bound fhir address")
		return nil
	}

	basic := base64.StdEncoding.EncodeToString([]byte(credUsername + ":" + credPasscode))
	for _, header := range []string{"Bearer " + credBearer, "Basic " + basic} {
		lastAuth.Store("")
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/fhir/Patient/example", nil)
		if err != nil {
			t.Errorf("build request: %v", err)
			continue
		}
		req.Header.Set("Authorization", header)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("send request: %v", err)
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			t.Errorf("read 401 body: %v", readErr)
			continue
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("rejecting authenticator answered %d, want 401", resp.StatusCode)
		}
		// Exact-match at the authenticator: a header the middleware stripped or
		// transformed would make the non-leak assertion vacuous.
		if got, _ := lastAuth.Load().(string); got != header {
			t.Errorf("authenticator saw Authorization %q, want the sentinel header", got)
		}
		// The body is surfaced verbatim: if the middleware echoed the Authorization
		// header, the sentinel would appear here and the error-sink scan would catch it.
		errs = append(errs, fmt.Errorf("middleware 401 body: %s", body))
	}
	return errs
}

// observedLogText renders every observed zap entry (message and fields) to one
// string so the sweep scans the daemon's structured-log sink for sentinels.
func observedLogText(logs *observer.ObservedLogs) string {
	var b strings.Builder
	for _, entry := range logs.All() {
		b.WriteString(entry.Message)
		for k, v := range entry.ContextMap() {
			fmt.Fprintf(&b, " %s=%v", k, v)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// TestCredentialLeakSweep is the automated PRD §9.8 credential-leak sweep: synthetic
// credential sentinels are pushed through every credential-bearing surface under
// default config, and no sentinel may surface in stdout, stderr, a returned error
// string, or the structured log.
//
// It does not call t.Parallel: phisweep.Run redirects the process-global standard
// streams, which must not overlap with another redirecting test.
func TestCredentialLeakSweep(t *testing.T) {
	cases := []struct {
		name     string
		exercise func(t *testing.T, ctx context.Context) []error
	}{
		{"dimse-user-identity", exerciseDIMSEUserIdentity},
		{"dicomweb-client", exerciseDICOMwebClient},
		{"fhir-rest-client", exerciseFHIRRESTClient},
	}

	sentinels := credSentinels()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			capture, err := phisweep.Run(func(ctx context.Context) []error { return tc.exercise(t, ctx) })
			if err != nil {
				t.Fatalf("run sweep: %v", err)
			}
			if leaks := phisweep.Scan(capture, sentinels); len(leaks) > 0 {
				for _, leak := range leaks {
					t.Errorf("credential leak: %s", leak)
				}
			}
		})
	}

	t.Run("server-auth-middleware", func(t *testing.T) {
		core, observed := observer.New(zap.InfoLevel)
		logger := zap.New(core)
		capture, err := phisweep.Run(func(ctx context.Context) []error {
			return exerciseServerAuthMiddleware(t, ctx, logger)
		})
		if err != nil {
			t.Fatalf("run sweep: %v", err)
		}
		// The daemon logs through the observer core, not the harness's context
		// logger, so its entries are appended to the log sink before scanning.
		capture.Logs += observedLogText(observed)
		if leaks := phisweep.Scan(capture, sentinels); len(leaks) > 0 {
			for _, leak := range leaks {
				t.Errorf("credential leak: %s", leak)
			}
		}
	})
}

// TestSweepDetectsPlantedCredentialLeak proves the sweep bites: an exercise rigged
// to echo a credential sentinel through the error sink must be detected. A gate that
// cannot fail is worthless.
func TestSweepDetectsPlantedCredentialLeak(t *testing.T) {
	capture, err := phisweep.Run(func(context.Context) []error {
		return []error{fmt.Errorf("authentication failed for token %s", credBearer)}
	})
	if err != nil {
		t.Fatalf("run sweep: %v", err)
	}
	leaks := phisweep.Scan(capture, credSentinels())
	if len(leaks) == 0 {
		t.Fatal("sweep failed to detect a planted credential leak in the error sink")
	}
}
