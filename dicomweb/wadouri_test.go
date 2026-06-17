package dicomweb

import (
	"bytes"
	"context"
	"io"
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

// getWADOURIObject performs a raw WADO-URI application/dicom GET for one instance and returns
// the response status and body bytes, so a test can assert on the exact bytes served.
func getWADOURIObject(t *testing.T, c *Client, study, series, object string) (int, []byte) {
	t.Helper()
	q := url.Values{}
	q.Set(wadoParamRequestType, wadoRequestTypeWADO)
	q.Set(wadoParamStudyUID, study)
	q.Set(wadoParamSeriesUID, series)
	q.Set(wadoParamObjectUID, object)
	q.Set(wadoParamContentType, mediaTypeDICOM)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, c.baseURL+"?"+q.Encode(), nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, body
}

// TestWADOURIServesStoredBytesVerbatim is the P2-A regression: when the backend supplies the
// stored Part 10 bytes (Encoded), the WADO-URI application/dicom response must be those exact
// bytes, never a re-encode from the DataSet. The sentinel bytes here are deliberately not a
// valid Part 10 object: re-encoding the DataSet could never reproduce them, so a byte-exact
// match proves the stored bytes were passed through verbatim, even for an uncompressed syntax.
func TestWADOURIServesStoredBytesVerbatim(t *testing.T) {
	stored := []byte("STORED-PART10-BYTES-VERBATIM-\x00\x01\x02\xff")
	store := newWADOStore()
	store.put(RetrievedInstance{
		DataSet:        sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"),
		TransferSyntax: dicom.ExplicitVRLittleEndian, // uncompressed: re-encode path would otherwise rewrite bytes
		Encoded:        stored,
	})
	c := newWADOServerClient(t, store)

	status, body := getWADOURIObject(t, c, "1.2.3", "1.2.3.4", "1.2.3.4.5")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if !bytes.Equal(body, stored) {
		t.Errorf("WADO-URI body was re-encoded, not served verbatim:\n got %q\nwant %q", body, stored)
	}
}

// TestWADOURIUnservableTransferSyntaxIs406 is the P2-B regression: a backend reporting a
// compressed transfer syntax with no Encoded bytes cannot be served unchanged (go-radx writes
// no encapsulated syntax), so WADO-URI must answer 406, not 500.
func TestWADOURIUnservableTransferSyntaxIs406(t *testing.T) {
	store := newWADOStore()
	store.put(RetrievedInstance{
		DataSet:        sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"),
		TransferSyntax: dicom.JPEGBaseline8Bit, // encapsulated, with no Encoded bytes to pass through
	})
	c := newWADOServerClient(t, store)

	status, _ := getWADOURIObject(t, c, "1.2.3", "1.2.3.4", "1.2.3.4.5")
	if status != http.StatusNotAcceptable {
		t.Errorf("status = %d, want 406 for an unservable compressed transfer syntax", status)
	}
}

// TestWADOURIEncodeFailureIs500 confirms a genuine internal encode failure (not a content
// negotiation refusal) still maps to 500, so the 406 mapping does not swallow real errors.
// An uncompressed instance with no Encoded bytes and no SOP Class UID fails encodeInstance's
// SOP-identity check with ErrInvalidResource, which is not ErrNotAcceptable.
func TestWADOURIEncodeFailureIs500(t *testing.T) {
	ds := sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5")
	ds.Delete(dicom.TagSOPClassUID) // breaks the Part 10 SOP-identity requirement -> internal encode error
	store := newWADOStore()
	store.put(RetrievedInstance{DataSet: ds, TransferSyntax: dicom.ExplicitVRLittleEndian})
	c := newWADOServerClient(t, store)

	status, _ := getWADOURIObject(t, c, "1.2.3", "1.2.3.4", "1.2.3.4.5")
	if status != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 for a genuine internal encode failure", status)
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
