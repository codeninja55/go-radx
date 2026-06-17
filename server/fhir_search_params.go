package server

// This file is the FHIR server role's SearchParameter registry: the minimal, release-neutral map of
// (resourceType, parameterName) to the JSON path and reference-target metadata the search-depth layer
// needs to resolve _include/_revinclude and one-hop chained parameters. FHIR's authoritative
// SearchParameter resources define hundreds of parameters per resource via FHIRPath expressions; this
// registry covers the reference and identifying parameters across the served workflow resource set
// (Patient, Encounter, ServiceRequest, ImagingStudy, DiagnosticReport, Observation) that the role's
// search depth resolves. The registry is the documented boundary: a production Repository with its
// own SearchParameter set resolves any parameter, while this role resolves the workflow references and
// the common identifying parameters the conformance subset exercises.
//
// The registry is deliberately small and JSON-path-based rather than FHIRPath-evaluated: the role
// reads resource fields as JSON (the same release-neutral approach the rest of the role uses for ids
// and references), so a parameter maps to a top-level JSON key, not an arbitrary expression. The R4
// and R5 workflow resources share these field names, so one registry serves both releases.

// searchParam describes one search parameter: the JSON path to its value within the resource, whether
// it is a reference parameter (so _include/_revinclude/chaining can dereference it) and the resource
// types it can point at, whether it is the special _id parameter (the resource's logical id), and
// whether its value is a HumanName (so a name search does a substring match across family/given/text).
type searchParam struct {
	jsonPath    string
	isReference bool
	targets     []string
	isID        bool
	isHumanName bool
}

// searchParamRegistry maps a resource type to its supported search parameters. Only the reference and
// identifying parameters the role's search depth resolves are listed; an unlisted parameter is left to
// the Repository (forwarded as a plain parameter) or ignored by the include/chain resolution, the
// documented boundary.
var searchParamRegistry = map[string]map[string]searchParam{
	"Patient": {
		"_id":  {jsonPath: "id", isID: true},
		"name": {jsonPath: "name", isHumanName: true},
	},
	"Encounter": {
		"_id":     {jsonPath: "id", isID: true},
		"subject": {jsonPath: "subject", isReference: true, targets: []string{"Patient"}},
		"patient": {jsonPath: "subject", isReference: true, targets: []string{"Patient"}},
	},
	"ServiceRequest": {
		"_id":       {jsonPath: "id", isID: true},
		"subject":   {jsonPath: "subject", isReference: true, targets: []string{"Patient"}},
		"patient":   {jsonPath: "subject", isReference: true, targets: []string{"Patient"}},
		"encounter": {jsonPath: "encounter", isReference: true, targets: []string{"Encounter"}},
	},
	"ImagingStudy": {
		"_id":      {jsonPath: "id", isID: true},
		"subject":  {jsonPath: "subject", isReference: true, targets: []string{"Patient"}},
		"patient":  {jsonPath: "subject", isReference: true, targets: []string{"Patient"}},
		"basedon":  {jsonPath: "basedOn", isReference: true, targets: []string{"ServiceRequest"}},
		"basedOn":  {jsonPath: "basedOn", isReference: true, targets: []string{"ServiceRequest"}},
		"endpoint": {jsonPath: "endpoint", isReference: true, targets: []string{"Endpoint"}},
	},
	"DiagnosticReport": {
		"_id":      {jsonPath: "id", isID: true},
		"subject":  {jsonPath: "subject", isReference: true, targets: []string{"Patient"}},
		"patient":  {jsonPath: "subject", isReference: true, targets: []string{"Patient"}},
		"result":   {jsonPath: "result", isReference: true, targets: []string{"Observation"}},
		"based-on": {jsonPath: "basedOn", isReference: true, targets: []string{"ServiceRequest"}},
	},
	"Observation": {
		"_id":          {jsonPath: "id", isID: true},
		"subject":      {jsonPath: "subject", isReference: true, targets: []string{"Patient"}},
		"patient":      {jsonPath: "subject", isReference: true, targets: []string{"Patient"}},
		"encounter":    {jsonPath: "encounter", isReference: true, targets: []string{"Encounter"}},
		"based-on":     {jsonPath: "basedOn", isReference: true, targets: []string{"ServiceRequest"}},
		"has-member":   {jsonPath: "hasMember", isReference: true, targets: []string{"Observation"}},
		"derived-from": {jsonPath: "derivedFrom", isReference: true, targets: []string{"Observation", "ImagingStudy"}},
	},
}

// lookupSearchParam returns the search parameter named name on resourceType and whether it is known to
// the registry. The lookup is case-sensitive on the FHIR parameter name (the spec's names are
// lowercase-with-hyphens), so "subject" resolves and an unknown name is reported absent — the caller
// then leaves the parameter to the Repository or ignores it.
func lookupSearchParam(resourceType, name string) (searchParam, bool) {
	params, ok := searchParamRegistry[resourceType]
	if !ok {
		return searchParam{}, false
	}
	param, ok := params[name]
	return param, ok
}
