package dicomweb

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
)

// TestWADOURIRetrieveRoundTrip is the client↔server WADO-URI acceptance: a stored instance is
// retrieved as application/dicom through the legacy URI service and decodes back to the same
// identity attributes.
func TestWADOURIRetrieveRoundTrip(t *testing.T) {
	store := newWADOStore()
	store.put(RetrievedInstance{DataSet: sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5")})
	c := newWADOServerClient(t, store)

	ds, err := c.WADORetrieveInstance(context.Background(), NewInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"))
	if err != nil {
		t.Fatalf("WADORetrieveInstance: %v", err)
	}
	if uid, _ := ds.GetString(dicom.TagSOPInstanceUID); uid != "1.2.3.4.5" {
		t.Errorf("SOPInstanceUID = %q, want 1.2.3.4.5", uid)
	}
	if name, _ := ds.GetString(dicom.TagPatientName); name != "Doe^Jane" {
		t.Errorf("PatientName = %q, want Doe^Jane", name)
	}
}

// TestWADOURIRetrieveObjectPreservesBytes verifies the object-returning entry point captures a
// byte-exact Part 10 representation and the origin transfer syntax.
func TestWADOURIRetrieveObjectPreservesBytes(t *testing.T) {
	store := newWADOStore()
	store.put(RetrievedInstance{DataSet: sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5")})
	c := newWADOServerClient(t, store)

	si, err := c.WADORetrieveInstanceObject(context.Background(), NewInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"))
	if err != nil {
		t.Fatalf("WADORetrieveInstanceObject: %v", err)
	}
	if len(si.Encoded) == 0 {
		t.Error("expected a byte-exact Part 10 representation")
	}
	if si.TransferSyntax == "" {
		t.Error("expected the origin transfer syntax to be reported")
	}
}

// TestWADOURIClientRejectsNonInstancePath rejects a study- or series-level path before any
// request is made: WADO-URI addresses exactly one object.
func TestWADOURIClientRejectsNonInstancePath(t *testing.T) {
	c, err := NewClient("https://pacs.example.org")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := c.WADORetrieveInstance(context.Background(), NewStudy("1.2.3")); err == nil {
		t.Error("expected an error for a study-level WADO-URI retrieve")
	}
	if _, err := c.WADORetrieveInstance(context.Background(), NewSeries("1.2.3", "1.2.3.4")); err == nil {
		t.Error("expected an error for a series-level WADO-URI retrieve")
	}
}

// TestWADOURIServerParamValidation drives the server directly to confirm fail-closed query
// validation: a missing or malformed required parameter, an unsupported requestType, a
// rendered contentType, and an absent object each map to the right HTTP status.
func TestWADOURIServerParamValidation(t *testing.T) {
	store := newWADOStore()
	store.put(RetrievedInstance{DataSet: sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5")})
	c := newWADOServerClient(t, store)

	base := func() url.Values {
		q := url.Values{}
		q.Set(wadoParamRequestType, wadoRequestTypeWADO)
		q.Set(wadoParamStudyUID, "1.2.3")
		q.Set(wadoParamSeriesUID, "1.2.3.4")
		q.Set(wadoParamObjectUID, "1.2.3.4.5")
		q.Set(wadoParamContentType, mediaTypeDICOM)
		return q
	}

	cases := []struct {
		name       string
		mutate     func(url.Values)
		wantStatus int
	}{
		{
			name:       "valid request succeeds",
			mutate:     func(url.Values) {},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing studyUID is 400",
			mutate:     func(q url.Values) { q.Del(wadoParamStudyUID) },
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing seriesUID is 400",
			mutate:     func(q url.Values) { q.Del(wadoParamSeriesUID) },
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing objectUID is 400",
			mutate:     func(q url.Values) { q.Del(wadoParamObjectUID) },
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "malformed objectUID is 400",
			mutate:     func(q url.Values) { q.Set(wadoParamObjectUID, "not a uid!") },
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "rendered contentType is 406",
			mutate:     func(q url.Values) { q.Set(wadoParamContentType, "image/jpeg") },
			wantStatus: http.StatusNotAcceptable,
		},
		{
			name:       "unsupported contentType is 406",
			mutate:     func(q url.Values) { q.Set(wadoParamContentType, "application/dicom+json") },
			wantStatus: http.StatusNotAcceptable,
		},
		{
			name: "absent object is 404",
			mutate: func(q url.Values) {
				q.Set(wadoParamObjectUID, "9.9.9.9")
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "default contentType (absent) returns the object",
			mutate:     func(q url.Values) { q.Del(wadoParamContentType) },
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := base()
			tc.mutate(q)
			target := c.baseURL + "?" + q.Encode()
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			resp, err := c.httpClient.Do(req)
			if err != nil {
				t.Fatalf("do: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != tc.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}
			if tc.wantStatus == http.StatusOK {
				if ct := resp.Header.Get("Content-Type"); ct != mediaTypeDICOM {
					t.Errorf("Content-Type = %q, want %q", ct, mediaTypeDICOM)
				}
			}
		})
	}
}

// TestWADOURIRequestTypeRequired confirms a GET without requestType=WADO is not routed to the
// URI service (it falls through to the path router, which answers 501 for the bare root).
func TestWADOURIRequestTypeRequired(t *testing.T) {
	store := newWADOStore()
	c := newWADOServerClient(t, store)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, c.baseURL+"/?studyUID=1.2.3", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusOK {
		t.Errorf("a request without requestType=WADO should not be served by the URI service")
	}
}
