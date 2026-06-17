package dicomweb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/codeninja55/go-radx/dicom"
)

// WADORetrieveInstance retrieves a single instance through the legacy WADO-URI service
// (PS3.18 §9). It issues a GET to the origin with requestType=WADO and the study/series/object
// UID triple, requesting contentType=application/dicom, and decodes the returned Part 10
// object into a dataset. Unlike WADO-RS, WADO-URI returns the object as the raw response body,
// not a multipart/related part.
//
// Only contentType=application/dicom is supported: rendered consumer formats (image/jpeg and
// the other PS3.18 §9.5 media types) are out of scope, tracked as a separate rendering parity
// item. The UID triple is validated as conformant DICOM UIDs before the request is built, so a
// malformed identifier is rejected client-side rather than sent to the origin. Use
// WADORetrieveInstanceObject when the byte-exact Part 10 representation and transfer syntax are
// needed.
func (c *Client) WADORetrieveInstance(ctx context.Context, p ResourcePath) (*dicom.DataSet, error) {
	si, err := c.wadoURIRetrieve(ctx, p)
	if err != nil {
		return nil, err
	}
	return si.DataSet, nil
}

// WADORetrieveInstanceObject retrieves a single instance through WADO-URI (PS3.18 §9),
// preserving the byte-exact Part 10 representation and the origin's transfer syntax in the
// returned RetrievedInstance. Like WADORetrieveInstance it serves contentType=application/dicom
// only; rendered formats are out of scope.
func (c *Client) WADORetrieveInstanceObject(ctx context.Context, p ResourcePath) (RetrievedInstance, error) {
	return c.wadoURIRetrieve(ctx, p)
}

// wadoURIRetrieve performs the WADO-URI GET for one instance and decodes the Part 10 object it
// returns. The object's byte-exact representation and transfer syntax are always captured, so
// either public entry point can return the decoded dataset or the full object. A response that
// is not application/dicom is a typed error rather than a silently mis-decoded body.
func (c *Client) wadoURIRetrieve(ctx context.Context, p ResourcePath) (RetrievedInstance, error) {
	if p.Level() != LevelInstance {
		return RetrievedInstance{}, fmt.Errorf("%w: WADO-URI requires an instance-level path (study, series, and object UID)", ErrInvalidResource)
	}
	// Validate the UID triple before building the request; a malformed UID is rejected
	// client-side and never placed in the query string.
	if _, err := p.Path(); err != nil {
		return RetrievedInstance{}, err
	}

	q := url.Values{}
	q.Set(wadoParamRequestType, wadoRequestTypeWADO)
	q.Set(wadoParamStudyUID, string(p.Study))
	q.Set(wadoParamSeriesUID, string(p.Series))
	q.Set(wadoParamObjectUID, string(p.Instance))
	q.Set(wadoParamContentType, mediaTypeDICOM)

	target := c.baseURL + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil) // #nosec G704 -- the URL is joined from the caller-configured base URL; requesting the configured service is the client's purpose
	if err != nil {
		return RetrievedInstance{}, fmt.Errorf("dicomweb: build WADO-URI request: %w", err)
	}
	req.Header.Set("Accept", mediaTypeDICOM)

	resp, err := c.httpClient.Do(req) // #nosec G704 -- requesting the configured WADO-URI service is the client's purpose
	if err != nil {
		return RetrievedInstance{}, c.transportError(http.MethodGet, "/?requestType=WADO", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return RetrievedInstance{}, c.httpError(http.MethodGet, "/?requestType=WADO", resp)
	}
	if mt := mediaTypeOf(resp.Header.Get("Content-Type")); mt != mediaTypeDICOM {
		return RetrievedInstance{}, fmt.Errorf("%w: WADO-URI response media type %q is not application/dicom", ErrNotAcceptable, mt)
	}

	body, err := io.ReadAll(c.boundedBody(resp))
	if err != nil {
		return RetrievedInstance{}, c.readError(http.MethodGet, "/?requestType=WADO", err)
	}
	return decodeRetrievedInstance(bytes.NewReader(body), true)
}
