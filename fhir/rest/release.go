package rest

import (
	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r4"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// This file is the one place fhir/rest imports the release packages. The client is release-fixed,
// but a Bundle, a CapabilityStatement, and an OperationOutcome are release-specific Go types
// (r4.Bundle vs r5.Bundle), so the client needs a release-neutral way to decode and inspect them.
// The bundleView adapter exposes the few Bundle facts the client needs — the navigation links and
// the entry resources — over either release's concrete type, so the rest of the client stays
// release-agnostic and the release split lives here.

// releaseRegistry returns the generated registry for a release and whether the release is one v1
// supports, so the client can dispatch a server response's resourceType to the correct release's
// concrete type. An unsupported release fails closed at construction.
func releaseRegistry(release fhir.Release) (*fhir.Registry, bool) {
	switch release {
	case fhir.R4:
		return r4.Registry, true
	case fhir.R5:
		return r5.Registry, true
	default:
		return nil, false
	}
}

// bundleLink is one navigation link of a searchset or history Bundle: a relation ("self", "next",
// "previous", "first", "last") and the absolute or relative URL to follow. The client follows the
// "next"/"previous" relations to page through a multi-page result.
type bundleLink struct {
	Relation string
	URL      string
}

// bundleView is the release-neutral read view of a Bundle the client paginates and unpacks. It
// names only what paging and resource extraction need: the navigation links and the resources the
// entries carry. A release-specific *r4.Bundle or *r5.Bundle is adapted into this view so the
// paging loop and the search-result unpacking are written once.
type bundleView struct {
	links     []bundleLink
	resources []fhir.Resource
}

// asBundleView adapts a decoded resource into a bundleView when it is the configured release's
// Bundle, returning ok=false for any other resource type (so a server that answered a search with a
// non-Bundle resource is reported, not silently treated as an empty page). The release is fixed on
// the client, so exactly one of the two type assertions can succeed for a given client.
func asBundleView(r fhir.Resource) (bundleView, bool) {
	switch b := r.(type) {
	case *r4.Bundle:
		return r4BundleView(b), true
	case *r5.Bundle:
		return r5BundleView(b), true
	default:
		return bundleView{}, false
	}
}

func r4BundleView(b *r4.Bundle) bundleView {
	v := bundleView{}
	for i := range b.Link {
		link := &b.Link[i]
		if link.Relation == nil || link.URL == nil {
			continue
		}
		v.links = append(v.links, bundleLink{Relation: string(*link.Relation), URL: *link.URL})
	}
	for i := range b.Entry {
		entry := &b.Entry[i]
		if entry.Resource != nil {
			v.resources = append(v.resources, *entry.Resource)
		}
	}
	return v
}

func r5BundleView(b *r5.Bundle) bundleView {
	v := bundleView{}
	for i := range b.Link {
		link := &b.Link[i]
		if link.Relation == nil || link.URL == nil {
			continue
		}
		v.links = append(v.links, bundleLink{Relation: string(*link.Relation), URL: *link.URL})
	}
	for i := range b.Entry {
		entry := &b.Entry[i]
		if entry.Resource != nil {
			v.resources = append(v.resources, *entry.Resource)
		}
	}
	return v
}

// linkURL returns the URL of the first link with the given relation, or "" when the Bundle carries
// no such link. The FHIR paging relations are "next" and "previous"; "self", "first", and "last"
// are also exposed for a caller that wants them.
func (v bundleView) linkURL(relation string) string {
	for _, l := range v.links {
		if l.Relation == relation {
			return l.URL
		}
	}
	return ""
}

// outcomeFromResource reduces a release OperationOutcome resource to the release-agnostic
// fhir.OperationOutcome the typed client error carries, returning nil when r is not an
// OperationOutcome. Only the severity, code, diagnostics, and the first expression of each issue
// are carried — all structural locators, never patient values (PRD §9.1) — so the in-process
// outcome the error exposes is the same shape exitcode.FromOperationOutcome consumes.
func outcomeFromResource(r fhir.Resource) *fhir.OperationOutcome {
	switch oo := r.(type) {
	case *r4.OperationOutcome:
		out := &fhir.OperationOutcome{}
		for i := range oo.Issue {
			out.Issue = append(out.Issue, r4IssueToOutcome(&oo.Issue[i]))
		}
		return out
	case *r5.OperationOutcome:
		out := &fhir.OperationOutcome{}
		for i := range oo.Issue {
			out.Issue = append(out.Issue, r5IssueToOutcome(&oo.Issue[i]))
		}
		return out
	default:
		return nil
	}
}

func r4IssueToOutcome(issue *r4.OperationOutcomeIssue) fhir.OutcomeIssue {
	out := fhir.OutcomeIssue{}
	if issue.Severity != nil {
		out.Severity = fhir.IssueSeverity(*issue.Severity)
	}
	if issue.Code != nil {
		out.Code = fhir.IssueType(*issue.Code)
	}
	if issue.Diagnostics != nil {
		out.Diagnostics = *issue.Diagnostics
	}
	if len(issue.Expression) > 0 {
		out.Expression = issue.Expression[0]
	}
	return out
}

func r5IssueToOutcome(issue *r5.OperationOutcomeIssue) fhir.OutcomeIssue {
	out := fhir.OutcomeIssue{}
	if issue.Severity != nil {
		out.Severity = fhir.IssueSeverity(*issue.Severity)
	}
	if issue.Code != nil {
		out.Code = fhir.IssueType(*issue.Code)
	}
	if issue.Diagnostics != nil {
		out.Diagnostics = *issue.Diagnostics
	}
	if len(issue.Expression) > 0 {
		out.Expression = issue.Expression[0]
	}
	return out
}

// capabilityFromResource adapts a release CapabilityStatement into the release-neutral Capability
// summary the client negotiates against, returning ok=false for any other resource type. It pulls
// the server's fhirVersion and the system- and resource-level interaction codes from the first
// "server"-mode rest component (the mode a FHIR server advertises its interactions under). A server
// with no rest component yields an empty capability rather than an error, so a sparse statement is
// "supports nothing" rather than a parse failure.
func capabilityFromResource(r fhir.Resource) (*Capability, bool) {
	switch cs := r.(type) {
	case *r4.CapabilityStatement:
		return r4Capability(cs), true
	case *r5.CapabilityStatement:
		return r5Capability(cs), true
	default:
		return nil, false
	}
}

func r4Capability(cs *r4.CapabilityStatement) *Capability {
	c := newCapability(cs)
	if cs.FhirVersion != nil {
		c.FHIRVersion = string(*cs.FhirVersion)
	}
	for i := range cs.Rest {
		rest := &cs.Rest[i]
		if rest.Mode == nil || string(*rest.Mode) != restModeServer {
			continue
		}
		for j := range rest.Interaction {
			if code := rest.Interaction[j].Code; code != nil {
				c.systemInteractions[string(*code)] = struct{}{}
			}
		}
		for j := range rest.Resource {
			res := &rest.Resource[j]
			if res.Type == nil {
				continue
			}
			codes := capabilityCodeSet(c, string(*res.Type))
			for k := range res.Interaction {
				if code := res.Interaction[k].Code; code != nil {
					codes[string(*code)] = struct{}{}
				}
			}
		}
	}
	return c
}

func r5Capability(cs *r5.CapabilityStatement) *Capability {
	c := newCapability(cs)
	if cs.FhirVersion != nil {
		c.FHIRVersion = string(*cs.FhirVersion)
	}
	for i := range cs.Rest {
		rest := &cs.Rest[i]
		if rest.Mode == nil || string(*rest.Mode) != restModeServer {
			continue
		}
		for j := range rest.Interaction {
			if code := rest.Interaction[j].Code; code != nil {
				c.systemInteractions[string(*code)] = struct{}{}
			}
		}
		for j := range rest.Resource {
			res := &rest.Resource[j]
			if res.Type == nil {
				continue
			}
			codes := capabilityCodeSet(c, string(*res.Type))
			for k := range res.Interaction {
				if code := res.Interaction[k].Code; code != nil {
					codes[string(*code)] = struct{}{}
				}
			}
		}
	}
	return c
}

// restModeServer is the CapabilityStatement.rest.mode value a FHIR server advertises its
// interactions under. The code is stable across R4 and R5, so it is matched as a literal string.
const restModeServer = "server"

func newCapability(statement fhir.Resource) *Capability {
	return &Capability{
		Statement:            statement,
		systemInteractions:   map[string]struct{}{},
		resourceInteractions: map[string]map[string]struct{}{},
	}
}

// capabilityCodeSet returns the interaction-code set for a resource type, creating it on first use,
// so the two release adapters share the per-resource accumulation.
func capabilityCodeSet(c *Capability, resourceType string) map[string]struct{} {
	codes, ok := c.resourceInteractions[resourceType]
	if !ok {
		codes = map[string]struct{}{}
		c.resourceInteractions[resourceType] = codes
	}
	return codes
}
