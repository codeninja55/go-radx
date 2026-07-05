package dicomweb

import (
	"encoding/xml"
	"net/http"
	"strings"
)

// mediaTypeWADL is the media type of the Capabilities Description the Retrieve
// Capabilities transaction returns (PS3.18 §8.9): a WADL document.
const mediaTypeWADL = "application/vnd.sun.wadl+xml"

// wadlNamespace is the WADL 2009-02 namespace the Capabilities Description is written in.
const wadlNamespace = "http://wadl.dev.java.net/2009/02"

// wadlApplication is the WADL document root, shared by the server's emitter and the
// client's parser. Only the subset the Capabilities Description uses is modelled:
// resources, methods, and their request/response representations.
type wadlApplication struct {
	XMLName   xml.Name      `xml:"application"`
	Namespace string        `xml:"xmlns,attr,omitempty"`
	Resources wadlResources `xml:"resources"`
}

type wadlResources struct {
	Base     string         `xml:"base,attr,omitempty"`
	Resource []wadlResource `xml:"resource"`
}

type wadlResource struct {
	Path     string         `xml:"path,attr"`
	Methods  []wadlMethod   `xml:"method"`
	Resource []wadlResource `xml:"resource,omitempty"`
}

type wadlMethod struct {
	Name     string       `xml:"name,attr"`
	ID       string       `xml:"id,attr,omitempty"`
	Request  *wadlPayload `xml:"request,omitempty"`
	Response *wadlPayload `xml:"response,omitempty"`
}

type wadlPayload struct {
	Representation []wadlRepresentation `xml:"representation"`
}

type wadlRepresentation struct {
	MediaType string `xml:"mediaType,attr"`
}

// handleCapabilities answers the Retrieve Capabilities transaction (PS3.18 §8.9): an
// OPTIONS on the service root returns a WADL Capabilities Description enumerating the
// transactions this server instance actually mounts. The description is honest by
// construction — it is derived from the registered backends (and their optional
// interfaces), so a transaction that would answer 501 is never advertised. It is a
// pragmatic WADL subset: resources, methods, and representation media types; query
// parameters and the WADO-URI query-string service are not described.
func (s *Server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	if !negotiateWADL(r.Header.Get("Accept")) {
		s.writeProblem(w, r, http.StatusNotAcceptable, ErrNotAcceptable,
			"the capabilities description is served as application/vnd.sun.wadl+xml only")
		return
	}
	// The resources base honours the configured public root exactly as the STOW-RS
	// Retrieve URLs do, so a proxied or path-prefixed deployment advertises URLs a
	// client can actually resolve.
	doc := s.capabilitiesDescription(s.storeRetrieveURLBase(r))
	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		s.writeProblem(w, r, http.StatusInternalServerError, err, "cannot encode the capabilities description")
		return
	}
	w.Header().Set("Content-Type", mediaTypeWADL)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(xml.Header))
	_, _ = w.Write(out) // #nosec G705 -- Content-Type is application/vnd.sun.wadl+xml (set above), not an HTML sink
}

// negotiateWADL reports whether an Accept header admits the WADL capabilities media type.
// An empty Accept (no preference) is accepted; the generic XML ranges (application/xml
// and the legacy text/xml) are admitted since WADL is XML, mirroring the media types the
// client itself accepts on the response side (isWADLContentType).
func negotiateWADL(accept string) bool {
	return negotiate(accept, func(mt string, _ map[string]string) bool {
		switch mt {
		case mediaTypeWADL, "application/xml", "text/xml", "application/*", "*/*":
			return true
		default:
			return false
		}
	})
}

// capabilitiesDescription builds the WADL document for this server's mounted services.
// Resources are keyed by path so a path serving several transactions (GET search and POST
// store on /studies) is described once with all its methods.
func (s *Server) capabilitiesDescription(originBaseURL string) wadlApplication {
	b := &capabilityBuilder{}
	s.describeQuery(b)
	s.describeStore(b)
	s.describeRetrieve(b)
	s.describeCapabilities(b)

	base := ""
	if originBaseURL != "" {
		base = strings.TrimRight(originBaseURL, "/") + "/"
	}
	return wadlApplication{
		Namespace: wadlNamespace,
		Resources: wadlResources{Base: base, Resource: b.resources},
	}
}

// capabilityBuilder accumulates WADL resources in insertion order, merging methods onto
// an already-described path.
type capabilityBuilder struct {
	resources []wadlResource
}

func (b *capabilityBuilder) add(path string, m wadlMethod) {
	for i := range b.resources {
		if b.resources[i].Path == path {
			b.resources[i].Methods = append(b.resources[i].Methods, m)
			return
		}
	}
	b.resources = append(b.resources, wadlResource{Path: path, Methods: []wadlMethod{m}})
}

// responseOf builds the single-representation response payload most methods carry.
func responseOf(mediaTypes ...string) *wadlPayload {
	reps := make([]wadlRepresentation, 0, len(mediaTypes))
	for _, mt := range mediaTypes {
		reps = append(reps, wadlRepresentation{MediaType: mt})
	}
	return &wadlPayload{Representation: reps}
}

// describeQuery advertises the six QIDO-RS search resources (PS3.18 §10.6) when a query
// backend is mounted.
func (s *Server) describeQuery(b *capabilityBuilder) {
	if s.query == nil {
		return
	}
	searches := []struct{ path, id string }{
		{"studies", "SearchForStudies"},
		{"series", "SearchForSeries"},
		{"instances", "SearchForInstances"},
		{"studies/{study}/series", "SearchForStudySeries"},
		{"studies/{study}/instances", "SearchForStudyInstances"},
		{"studies/{study}/series/{series}/instances", "SearchForStudySeriesInstances"},
	}
	for _, sr := range searches {
		b.add(sr.path, wadlMethod{
			Name:     http.MethodGet,
			ID:       sr.id,
			Response: responseOf(mediaTypeDICOMJSON),
		})
	}
}

// describeStore advertises the two STOW-RS store targets (PS3.18 §10.5) when a store
// backend is mounted. Both accepted body variants are named as request representations.
func (s *Server) describeStore(b *capabilityBuilder) {
	if s.store == nil {
		return
	}
	request := &wadlPayload{Representation: []wadlRepresentation{
		{MediaType: relatedContentType(mediaTypeDICOM)},
		{MediaType: relatedContentType(mediaTypeDICOMJSON)},
	}}
	b.add("studies", wadlMethod{
		Name: http.MethodPost, ID: "StoreInstances",
		Request: request, Response: responseOf(mediaTypeDICOMJSON),
	})
	b.add("studies/{study}", wadlMethod{
		Name: http.MethodPost, ID: "StoreStudyInstances",
		Request: request, Response: responseOf(mediaTypeDICOMJSON),
	})
}

// describeRetrieve advertises the WADO-RS retrieve resources (PS3.18 §10.4) the mounted
// retrieval backend can actually answer: the base RetrieveBackend carries instance
// retrieval and the locator-suffixed bulkdata reference; each optional retriever interface
// adds its own resource only when the backend implements it.
func (s *Server) describeRetrieve(b *capabilityBuilder) {
	if s.retrieve == nil {
		return
	}
	const instancePath = "studies/{study}/series/{series}/instances/{instance}"
	multipartDICOM := relatedContentType(mediaTypeDICOM)
	multipartOctet := relatedContentType(mediaTypeOctet)

	b.add(instancePath, wadlMethod{
		Name: http.MethodGet, ID: "RetrieveInstance", Response: responseOf(multipartDICOM),
	})
	b.add(instancePath+"/bulkdata/{reference}", wadlMethod{
		Name: http.MethodGet, ID: "RetrieveBulkdataReference", Response: responseOf(multipartOctet),
	})
	if _, ok := s.retrieve.(StudyRetriever); ok {
		b.add("studies/{study}", wadlMethod{
			Name: http.MethodGet, ID: "RetrieveStudy", Response: responseOf(multipartDICOM),
		})
	}
	if _, ok := s.retrieve.(SeriesRetriever); ok {
		b.add("studies/{study}/series/{series}", wadlMethod{
			Name: http.MethodGet, ID: "RetrieveSeries", Response: responseOf(multipartDICOM),
		})
	}
	if _, ok := s.retrieve.(MetadataRetriever); ok {
		metadata := []struct{ path, id string }{
			{"studies/{study}/metadata", "RetrieveStudyMetadata"},
			{"studies/{study}/series/{series}/metadata", "RetrieveSeriesMetadata"},
			{instancePath + "/metadata", "RetrieveInstanceMetadata"},
		}
		for _, md := range metadata {
			b.add(md.path, wadlMethod{
				Name: http.MethodGet, ID: md.id,
				Response: responseOf(mediaTypeDICOMJSON, relatedContentType(mediaTypeDICOMXML)),
			})
		}
	}
	if _, ok := s.retrieve.(FrameRetriever); ok {
		b.add(instancePath+"/frames/{frames}", wadlMethod{
			Name: http.MethodGet, ID: "RetrieveFrames", Response: responseOf(multipartOctet),
		})
	}
	if _, ok := s.retrieve.(BulkDataRetriever); ok {
		b.add(instancePath+"/bulkdata", wadlMethod{
			Name: http.MethodGet, ID: "RetrieveBulkdata", Response: responseOf(multipartOctet),
		})
	}
}

// describeCapabilities advertises the Retrieve Capabilities transaction itself, which is
// always served.
func (s *Server) describeCapabilities(b *capabilityBuilder) {
	b.add("", wadlMethod{
		Name: http.MethodOptions, ID: "RetrieveCapabilities", Response: responseOf(mediaTypeWADL),
	})
}
