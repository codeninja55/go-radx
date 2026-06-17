package dicomweb

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
)

// TestRetrieveMetadataXMLRoundTrip asserts a client↔server XML metadata round-trip: the
// server produces a multipart/related body of Native DICOM Model parts and the client parses
// them back into datasets carrying the original identity and patient-name attributes.
func TestRetrieveMetadataXMLRoundTrip(t *testing.T) {
	store := newWADOStore()
	store.put(RetrievedInstance{DataSet: sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5")})
	c := newWADOServerClient(t, store)

	metas, err := c.RetrieveMetadataXML(context.Background(), NewInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"))
	if err != nil {
		t.Fatalf("RetrieveMetadataXML: %v", err)
	}
	if len(metas) != 1 {
		t.Fatalf("RetrieveMetadataXML returned %d objects, want 1", len(metas))
	}
	if name, _ := metas[0].GetString(dicom.TagPatientName); name != "Doe^Jane" {
		t.Fatalf("metadata PatientName = %q, want Doe^Jane", name)
	}
}

// TestMetadataContentNegotiation drives the metadata sub-resource directly to confirm the
// Accept header selects between application/dicom+xml and application/dicom+json, and that an
// Accept naming neither answers 406.
func TestMetadataContentNegotiation(t *testing.T) {
	store := newWADOStore()
	store.put(RetrievedInstance{DataSet: sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5")})
	c := newWADOServerClient(t, store)

	metaPath, err := NewInstance("1.2.3", "1.2.3.4", "1.2.3.4.5").Metadata()
	if err != nil {
		t.Fatalf("metadata path: %v", err)
	}

	cases := []struct {
		name        string
		accept      string
		wantStatus  int
		wantCTHas   string
		wantBodyHas string
	}{
		{
			name:        "xml accept selects native dicom model",
			accept:      relatedContentType(mediaTypeDICOMXML),
			wantStatus:  http.StatusOK,
			wantCTHas:   "multipart/related",
			wantBodyHas: "<NativeDicomModel",
		},
		{
			name:        "bare dicom+xml accept selects xml",
			accept:      mediaTypeDICOMXML,
			wantStatus:  http.StatusOK,
			wantCTHas:   "multipart/related",
			wantBodyHas: "<NativeDicomModel",
		},
		{
			name:        "json accept selects dicom json",
			accept:      mediaTypeDICOMJSON,
			wantStatus:  http.StatusOK,
			wantCTHas:   mediaTypeDICOMJSON,
			wantBodyHas: `"vr"`,
		},
		{
			name:        "empty accept defaults to json",
			accept:      "",
			wantStatus:  http.StatusOK,
			wantCTHas:   mediaTypeDICOMJSON,
			wantBodyHas: `"vr"`,
		},
		{
			name:        "wildcard accept defaults to json",
			accept:      "*/*",
			wantStatus:  http.StatusOK,
			wantCTHas:   mediaTypeDICOMJSON,
			wantBodyHas: `"vr"`,
		},
		{
			name:       "unservable accept answers 406",
			accept:     "image/jpeg",
			wantStatus: http.StatusNotAcceptable,
		},
		{
			name:        "json refused falls through to xml",
			accept:      mediaTypeDICOMJSON + ";q=0, " + mediaTypeDICOMXML,
			wantStatus:  http.StatusOK,
			wantCTHas:   "multipart/related",
			wantBodyHas: "<NativeDicomModel",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, c.baseURL+metaPath, nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			if tc.accept != "" {
				req.Header.Set("Accept", tc.accept)
			}
			resp, err := c.httpClient.Do(req)
			if err != nil {
				t.Fatalf("do: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}
			if tc.wantStatus != http.StatusOK {
				return
			}
			if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, tc.wantCTHas) {
				t.Errorf("Content-Type = %q, want it to contain %q", ct, tc.wantCTHas)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if !strings.Contains(string(body), tc.wantBodyHas) {
				t.Errorf("body does not contain %q:\n%s", tc.wantBodyHas, body)
			}
		})
	}
}
