// Command fixtures marshals a representative, fully-populated instance of each
// FHIR R5 workflow resource go-radx commits to (Patient, Encounter, ServiceRequest,
// ImagingStudy, DiagnosticReport, Observation, Bundle, OperationOutcome,
// CapabilityStatement) and writes the JSON to an output directory, one file per
// resource. The conformance gate (tools/fhir-conformance/validate.sh) then runs the
// official HL7 FHIR validator over that directory, so the gate validates go-radx's
// OWN generated marshalling rather than a borrowed example corpus.
//
// Every instance is authored with entirely synthetic, fictitious data (an obviously
// invented MRN, a TESTPATIENT name, example.org systems), so no fixture content
// encodes real Protected Health Information. The instances are deliberately
// fully-populated — choice types, primitive values, references, and a Bundle that
// references the others — so a validator error reflects a real conformance defect in
// the generated code, not a thin fixture.
//
// Usage:
//
//	go run ./tools/fhir-conformance/fixtures <output-dir>
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: fixtures <output-dir>")
		os.Exit(64)
	}
	outDir := os.Args[1]
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "fixtures: mkdir %s: %v\n", outDir, err)
		os.Exit(1)
	}

	patient := workflowPatient()
	encounter := workflowEncounter()
	serviceRequest := workflowServiceRequest()
	imagingStudy := workflowImagingStudy()
	observation := workflowObservation()
	diagnosticReport := workflowDiagnosticReport()
	operationOutcome := workflowOperationOutcome()
	capabilityStatement := workflowCapabilityStatement()

	bundle, err := workflowBundle(patient, encounter, serviceRequest, imagingStudy,
		observation, diagnosticReport)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fixtures: build bundle: %v\n", err)
		os.Exit(1)
	}

	resources := []struct {
		name string
		res  fhir.Resource
	}{
		{"Patient", patient},
		{"Encounter", encounter},
		{"ServiceRequest", serviceRequest},
		{"ImagingStudy", imagingStudy},
		{"Observation", observation},
		{"DiagnosticReport", diagnosticReport},
		{"OperationOutcome", operationOutcome},
		{"CapabilityStatement", capabilityStatement},
		{"Bundle", bundle},
	}

	for _, r := range resources {
		// Marshal through the resource's own MarshalJSON (the canonical form, with the
		// _field siblings trailing) — exactly the wire output a go-radx consumer emits.
		encoded, err := json.MarshalIndent(r.res, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "fixtures: marshal %s: %v\n", r.name, err)
			os.Exit(1)
		}
		path := filepath.Join(outDir, r.name+".json")
		if err := os.WriteFile(path, append(encoded, '\n'), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "fixtures: write %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s\n", path)
	}
}

// The pointer helpers mirror the project's documented standard set; FHIR primitives
// are predominantly pointer fields so an absent element and a present zero value stay
// distinguishable on the wire.
func strPtr(s string) *string { return &s }
func i32Ptr(i int32) *int32   { return &i }

func decimal(s string) *fhir.Decimal {
	d, err := fhir.ParseDecimal(s)
	if err != nil {
		panic(fmt.Sprintf("fixtures: invalid decimal literal %q: %v", s, err))
	}
	return &d
}

// narrative builds the generated-narrative text element FHIR's dom-6 best-practice
// invariant expects on a domain resource. The div is minimal XHTML; the validator
// checks it is well-formed, not its prose.
func narrative(text string) *r5.Narrative {
	status := r5.NarrativeStatusGenerated
	return &r5.Narrative{
		Status: &status,
		Div:    strPtr(`<div xmlns="http://www.w3.org/1999/xhtml">` + text + `</div>`),
	}
}

func codeableConcept(system, code, display string) *r5.CodeableConcept {
	return &r5.CodeableConcept{
		Coding: []r5.Coding{{
			System:  strPtr(system),
			Code:    strPtr(code),
			Display: strPtr(display),
		}},
		Text: strPtr(display),
	}
}

func reference(ref, display string) *r5.Reference {
	return &r5.Reference{Reference: strPtr(ref), Display: strPtr(display)}
}

const (
	patientRef = "Patient/wf-patient"
	patientID  = "wf-patient"
)

func workflowPatient() *r5.Patient {
	p := &r5.Patient{}
	p.ID = strPtr(patientID)
	p.Text = narrative("Synthetic test patient TESTPATIENT, Workflow")
	gender := r5.AdministrativeGenderFemale
	use := r5.NameUseOfficial
	idUse := r5.IdentifierUseUsual
	p.Identifier = []r5.Identifier{{
		Use:    &idUse,
		System: strPtr("urn:oid:1.2.36.146.595.217.0.1"),
		Value:  strPtr("MRN0001234"),
	}}
	active := true
	p.Active = &active
	p.Name = []r5.HumanName{{
		Use:    &use,
		Family: strPtr("TESTPATIENT"),
		Given:  []string{"Workflow", "Synthetic"},
	}}
	p.Gender = &gender
	p.BirthDate = strPtr("1985-07-12")
	return p
}

func workflowEncounter() *r5.Encounter {
	e := &r5.Encounter{}
	e.ID = strPtr("wf-encounter")
	e.Text = narrative("Synthetic outpatient imaging encounter")
	status := r5.EncounterStatusInProgress
	e.Status = &status
	e.Class = []r5.CodeableConcept{*codeableConcept(
		"http://terminology.hl7.org/CodeSystem/v3-ActCode", "AMB", "ambulatory")}
	e.Subject = reference(patientRef, "TESTPATIENT, Workflow")
	return e
}

func workflowServiceRequest() *r5.ServiceRequest {
	sr := &r5.ServiceRequest{}
	sr.ID = strPtr("wf-servicerequest")
	sr.Text = narrative("Synthetic imaging order: CT chest")
	status := r5.RequestStatusActive
	intent := r5.RequestIntentOrder
	sr.Status = &status
	sr.Intent = &intent
	sr.Code = &r5.CodeableReference{
		Concept: codeableConcept("http://loinc.org", "24627-2", "CT Chest"),
	}
	sr.Subject = reference(patientRef, "TESTPATIENT, Workflow")
	sr.Encounter = reference("Encounter/wf-encounter", "Imaging encounter")
	sr.AuthoredOn = strPtr("2026-06-01T09:30:00Z")
	return sr
}

func workflowImagingStudy() *r5.ImagingStudy {
	is := &r5.ImagingStudy{}
	is.ID = strPtr("wf-imagingstudy")
	is.Text = narrative("Synthetic CT chest imaging study, 1 series")
	status := r5.ImagingStudyStatusAvailable
	is.Status = &status
	is.Subject = reference(patientRef, "TESTPATIENT, Workflow")
	is.Encounter = reference("Encounter/wf-encounter", "Imaging encounter")
	is.Started = strPtr("2026-06-01T10:00:00Z")
	is.NumberOfSeries = i32Ptr(1)
	is.NumberOfInstances = i32Ptr(1)
	is.Modality = []r5.CodeableConcept{*codeableConcept(
		"http://dicom.nema.org/resources/ontology/DCM", "CT", "Computed Tomography")}
	is.Series = []r5.ImagingStudySeries{{
		UID:               strPtr("1.2.840.113619.2.55.3.604688.1"),
		Number:            i32Ptr(1),
		Modality:          codeableConcept("http://dicom.nema.org/resources/ontology/DCM", "CT", "Computed Tomography"),
		NumberOfInstances: i32Ptr(1),
		Instance: []r5.ImagingStudySeriesInstance{{
			UID:      strPtr("1.2.840.113619.2.55.3.604688.1.1"),
			SopClass: &r5.Coding{System: strPtr("urn:ietf:rfc:3986"), Code: strPtr("urn:oid:1.2.840.10008.5.1.4.1.1.2")},
			Number:   i32Ptr(1),
		}},
	}}
	return is
}

func workflowObservation() *r5.Observation {
	o := &r5.Observation{}
	o.ID = strPtr("wf-observation")
	o.Text = narrative("Synthetic body weight observation, 72.5 kg")
	status := r5.ObservationStatusFinal
	o.Status = &status
	o.Category = []r5.CodeableConcept{*codeableConcept(
		"http://terminology.hl7.org/CodeSystem/observation-category", "vital-signs", "Vital Signs")}
	o.Code = codeableConcept("http://loinc.org", "29463-7", "Body weight")
	o.Subject = reference(patientRef, "TESTPATIENT, Workflow")
	o.Encounter = reference("Encounter/wf-encounter", "Imaging encounter")
	o.EffectiveDateTime = (*r5.FHIRDateTime)(strPtr("2026-06-01T09:00:00Z"))
	o.ValueQuantity = &r5.Quantity{
		Value:  decimal("72.5"),
		Unit:   strPtr("kg"),
		System: strPtr("http://unitsofmeasure.org"),
		Code:   strPtr("kg"),
	}
	return o
}

func workflowDiagnosticReport() *r5.DiagnosticReport {
	dr := &r5.DiagnosticReport{}
	dr.ID = strPtr("wf-diagnosticreport")
	dr.Text = narrative("Synthetic CT chest radiology report")
	status := r5.DiagnosticReportStatusFinal
	dr.Status = &status
	dr.Category = []r5.CodeableConcept{*codeableConcept(
		"http://terminology.hl7.org/CodeSystem/v2-0074", "RAD", "Radiology")}
	dr.Code = codeableConcept("http://loinc.org", "24627-2", "CT Chest")
	dr.Subject = reference(patientRef, "TESTPATIENT, Workflow")
	dr.Encounter = reference("Encounter/wf-encounter", "Imaging encounter")
	dr.EffectiveDateTime = (*r5.FHIRDateTime)(strPtr("2026-06-01T10:30:00Z"))
	dr.Issued = strPtr("2026-06-01T11:00:00Z")
	dr.Study = []r5.Reference{*reference("ImagingStudy/wf-imagingstudy", "CT chest study")}
	dr.Result = []r5.Reference{*reference("Observation/wf-observation", "Body weight")}
	return dr
}

func workflowOperationOutcome() *r5.OperationOutcome {
	oo := &r5.OperationOutcome{}
	oo.ID = strPtr("wf-operationoutcome")
	oo.Text = narrative("Synthetic informational operation outcome")
	severity := r5.IssueSeverityInformation
	code := r5.IssueTypeInformational
	oo.Issue = []r5.OperationOutcomeIssue{{
		Severity:    &severity,
		Code:        &code,
		Diagnostics: strPtr("Synthetic outcome for the conformance gate"),
	}}
	return oo
}

func workflowCapabilityStatement() *r5.CapabilityStatement {
	cs := &r5.CapabilityStatement{}
	cs.ID = strPtr("wf-capabilitystatement")
	cs.Text = narrative("Synthetic go-radx server capability statement")
	cs.URL = strPtr("http://example.org/go-radx/CapabilityStatement/workflow")
	cs.Name = strPtr("GoRadxWorkflowServer")
	cs.Title = strPtr("go-radx workflow server")
	status := r5.PublicationStatusActive
	cs.Status = &status
	cs.Date = strPtr("2026-06-01")
	cs.Publisher = strPtr("go-radx")
	cs.Description = strPtr("Synthetic capability statement for the go-radx FHIR conformance gate")
	kind := r5.CapabilityStatementKindInstance
	cs.Kind = &kind
	// cpb-14: a kind=instance statement must carry an implementation element.
	cs.Implementation = &r5.CapabilityStatementImplementation{
		Description: strPtr("go-radx workflow server instance"),
		URL:         strPtr("https://go-radx.test/fhir"),
	}
	fhirVersion := r5.FHIRVersionN500
	cs.FhirVersion = &fhirVersion
	cs.Format = []string{"json"}
	mode := r5.RestfulCapabilityModeServer
	cs.Rest = []r5.CapabilityStatementRest{{
		Mode:          &mode,
		Documentation: strPtr("Synthetic RESTful capability"),
	}}
	return cs
}

// workflowBundle assembles a collection Bundle that holds the workflow resources by
// fullUrl, exercising the Bundle builder and intra-bundle references on the wire.
func workflowBundle(resources ...fhir.Resource) (*r5.Bundle, error) {
	entries := make([]r5.CollectionEntry, 0, len(resources))
	for _, r := range resources {
		entries = append(entries, r5.CollectionEntry{
			FullURL:  "http://example.org/go-radx/" + r.ResourceType(),
			Resource: r,
		})
	}
	return r5.NewCollection(entries...)
}
