// Command fixtures-r4 marshals a representative, fully-populated instance of each
// FHIR R4 (4.0.1) workflow resource go-radx commits to (Patient, Encounter,
// ServiceRequest, ImagingStudy, DiagnosticReport, Observation, Bundle,
// OperationOutcome, CapabilityStatement) and writes the JSON to an output directory,
// one file per resource. The conformance gate (tools/fhir-conformance/validate.sh)
// then runs the official HL7 FHIR validator over that directory with -version 4.0.1,
// so the gate validates go-radx's OWN generated R4 marshalling rather than a borrowed
// example corpus.
//
// It is the R4 sibling of tools/fhir-conformance/fixtures (R5). The two are separate
// because the release type spaces never mix and several resources differ on the wire:
// R4 has no CodeableReference (ServiceRequest.code is a CodeableConcept), R4
// ImagingStudy.modality is a Coding, R4 Encounter.class is a single Coding, and R4
// uses DiagnosticReport.imagingStudy rather than .study.
//
// Every instance is authored with entirely synthetic, fictitious data (an obviously
// invented MRN, a TESTPATIENT name, example.org systems), so no fixture content
// encodes real Protected Health Information.
//
// Usage:
//
//	go run ./tools/fhir-conformance/fixtures-r4 <output-dir>
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r4"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: fixtures-r4 <output-dir>")
		os.Exit(64)
	}
	outDir := os.Args[1]
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "fixtures-r4: mkdir %s: %v\n", outDir, err)
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
		fmt.Fprintf(os.Stderr, "fixtures-r4: build bundle: %v\n", err)
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
		encoded, err := json.MarshalIndent(r.res, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "fixtures-r4: marshal %s: %v\n", r.name, err)
			os.Exit(1)
		}
		path := filepath.Join(outDir, r.name+".json")
		if err := os.WriteFile(path, append(encoded, '\n'), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "fixtures-r4: write %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s\n", path)
	}
}

func strPtr(s string) *string { return &s }
func i32Ptr(i int32) *int32   { return &i }

func decimal(s string) *fhir.Decimal {
	d, err := fhir.ParseDecimal(s)
	if err != nil {
		panic(fmt.Sprintf("fixtures-r4: invalid decimal literal %q: %v", s, err))
	}
	return &d
}

// narrative builds the generated-narrative text element FHIR's dom-6 best-practice
// invariant expects on a domain resource. The div is minimal XHTML; the validator
// checks it is well-formed, not its prose.
func narrative(text string) *r4.Narrative {
	status := r4.NarrativeStatusGenerated
	return &r4.Narrative{
		Status: &status,
		Div:    strPtr(`<div xmlns="http://www.w3.org/1999/xhtml">` + text + `</div>`),
	}
}

func codeableConcept(system, code, display string) *r4.CodeableConcept {
	return &r4.CodeableConcept{
		Coding: []r4.Coding{{
			System:  strPtr(system),
			Code:    strPtr(code),
			Display: strPtr(display),
		}},
		Text: strPtr(display),
	}
}

// coding builds a Coding, omitting the display element when display is empty: FHIR's
// ele-1 invariant rejects a present-but-empty primitive ("display": ""), so an unset
// display must be absent on the wire rather than an empty string.
func coding(system, code, display string) *r4.Coding {
	c := &r4.Coding{System: strPtr(system), Code: strPtr(code)}
	if display != "" {
		c.Display = strPtr(display)
	}
	return c
}

func reference(ref, display string) *r4.Reference {
	return &r4.Reference{Reference: strPtr(ref), Display: strPtr(display)}
}

const (
	patientRef = "Patient/wf-patient"
	patientID  = "wf-patient"
)

func workflowPatient() *r4.Patient {
	p := &r4.Patient{}
	p.ID = strPtr(patientID)
	p.Text = narrative("Synthetic test patient TESTPATIENT, Workflow")
	gender := r4.AdministrativeGenderFemale
	use := r4.NameUseOfficial
	idUse := r4.IdentifierUseUsual
	p.Identifier = []r4.Identifier{{
		Use:    &idUse,
		System: strPtr("urn:oid:1.2.36.146.595.217.0.1"),
		Value:  strPtr("MRN0001234"),
	}}
	active := true
	p.Active = &active
	p.Name = []r4.HumanName{{
		Use:    &use,
		Family: strPtr("TESTPATIENT"),
		Given:  []string{"Workflow", "Synthetic"},
	}}
	p.Gender = &gender
	p.BirthDate = strPtr("1985-07-12")
	return p
}

// workflowEncounter authors an R4 Encounter. R4 Encounter.class is a single Coding
// (a list in R5), and R4 carries Encounter.period (R5 renamed it to actualPeriod).
func workflowEncounter() *r4.Encounter {
	e := &r4.Encounter{}
	e.ID = strPtr("wf-encounter")
	e.Text = narrative("Synthetic outpatient imaging encounter")
	status := r4.EncounterStatusInProgress
	e.Status = &status
	e.Class = coding("http://terminology.hl7.org/CodeSystem/v3-ActCode", "AMB", "ambulatory")
	e.Subject = reference(patientRef, "TESTPATIENT, Workflow")
	return e
}

// workflowServiceRequest authors an R4 ServiceRequest. R4 ServiceRequest.code is a
// CodeableConcept; R4 has no CodeableReference (R5 wraps code in one).
func workflowServiceRequest() *r4.ServiceRequest {
	sr := &r4.ServiceRequest{}
	sr.ID = strPtr("wf-servicerequest")
	sr.Text = narrative("Synthetic imaging order: CT chest")
	status := r4.RequestStatusActive
	intent := r4.RequestIntentOrder
	sr.Status = &status
	sr.Intent = &intent
	sr.Code = codeableConcept("http://loinc.org", "24627-2", "CT Chest")
	sr.Subject = reference(patientRef, "TESTPATIENT, Workflow")
	sr.Encounter = reference("Encounter/wf-encounter", "Imaging encounter")
	sr.AuthoredOn = strPtr("2026-06-01T09:30:00Z")
	return sr
}

// workflowImagingStudy authors an R4 ImagingStudy. R4 ImagingStudy.modality is a
// list of Coding (a CodeableConcept in R5) and series.modality is a single Coding.
func workflowImagingStudy() *r4.ImagingStudy {
	is := &r4.ImagingStudy{}
	is.ID = strPtr("wf-imagingstudy")
	is.Text = narrative("Synthetic CT chest imaging study, 1 series")
	status := r4.ImagingStudyStatusAvailable
	is.Status = &status
	is.Subject = reference(patientRef, "TESTPATIENT, Workflow")
	is.Encounter = reference("Encounter/wf-encounter", "Imaging encounter")
	is.Started = strPtr("2026-06-01T10:00:00Z")
	is.NumberOfSeries = i32Ptr(1)
	is.NumberOfInstances = i32Ptr(1)
	is.Modality = []r4.Coding{*coding("http://dicom.nema.org/resources/ontology/DCM", "CT", "Computed Tomography")}
	is.Series = []r4.ImagingStudySeries{{
		UID:               strPtr("1.2.840.113619.2.55.3.604688.1"),
		Number:            i32Ptr(1),
		Modality:          coding("http://dicom.nema.org/resources/ontology/DCM", "CT", "Computed Tomography"),
		NumberOfInstances: i32Ptr(1),
		Instance: []r4.ImagingStudySeriesInstance{{
			UID:      strPtr("1.2.840.113619.2.55.3.604688.1.1"),
			SopClass: coding("urn:ietf:rfc:3986", "urn:oid:1.2.840.10008.5.1.4.1.1.2", ""),
			Number:   i32Ptr(1),
		}},
	}}
	return is
}

func workflowObservation() *r4.Observation {
	o := &r4.Observation{}
	o.ID = strPtr("wf-observation")
	o.Text = narrative("Synthetic body weight observation, 72.5 kg")
	status := r4.ObservationStatusFinal
	o.Status = &status
	o.Category = []r4.CodeableConcept{*codeableConcept(
		"http://terminology.hl7.org/CodeSystem/observation-category", "vital-signs", "Vital Signs")}
	o.Code = codeableConcept("http://loinc.org", "29463-7", "Body weight")
	o.Subject = reference(patientRef, "TESTPATIENT, Workflow")
	o.Encounter = reference("Encounter/wf-encounter", "Imaging encounter")
	o.EffectiveDateTime = (*r4.FHIRDateTime)(strPtr("2026-06-01T09:00:00Z"))
	o.ValueQuantity = &r4.Quantity{
		Value:  decimal("72.5"),
		Unit:   strPtr("kg"),
		System: strPtr("http://unitsofmeasure.org"),
		Code:   strPtr("kg"),
	}
	return o
}

// workflowDiagnosticReport authors an R4 DiagnosticReport. R4 names the imaging-study
// link DiagnosticReport.imagingStudy (R5 renamed it to study).
func workflowDiagnosticReport() *r4.DiagnosticReport {
	dr := &r4.DiagnosticReport{}
	dr.ID = strPtr("wf-diagnosticreport")
	dr.Text = narrative("Synthetic CT chest radiology report")
	status := r4.DiagnosticReportStatusFinal
	dr.Status = &status
	dr.Category = []r4.CodeableConcept{*codeableConcept(
		"http://terminology.hl7.org/CodeSystem/v2-0074", "RAD", "Radiology")}
	dr.Code = codeableConcept("http://loinc.org", "24627-2", "CT Chest")
	dr.Subject = reference(patientRef, "TESTPATIENT, Workflow")
	dr.Encounter = reference("Encounter/wf-encounter", "Imaging encounter")
	dr.EffectiveDateTime = (*r4.FHIRDateTime)(strPtr("2026-06-01T10:30:00Z"))
	dr.Issued = strPtr("2026-06-01T11:00:00Z")
	dr.ImagingStudy = []r4.Reference{*reference("ImagingStudy/wf-imagingstudy", "CT chest study")}
	dr.Result = []r4.Reference{*reference("Observation/wf-observation", "Body weight")}
	return dr
}

func workflowOperationOutcome() *r4.OperationOutcome {
	oo := &r4.OperationOutcome{}
	oo.ID = strPtr("wf-operationoutcome")
	oo.Text = narrative("Synthetic informational operation outcome")
	severity := r4.IssueSeverityInformation
	code := r4.IssueTypeInformational
	oo.Issue = []r4.OperationOutcomeIssue{{
		Severity:    &severity,
		Code:        &code,
		Diagnostics: strPtr("Synthetic outcome for the conformance gate"),
	}}
	return oo
}

func workflowCapabilityStatement() *r4.CapabilityStatement {
	cs := &r4.CapabilityStatement{}
	cs.ID = strPtr("wf-capabilitystatement")
	cs.Text = narrative("Synthetic go-radx server capability statement")
	cs.URL = strPtr("http://example.org/go-radx/CapabilityStatement/workflow")
	cs.Name = strPtr("GoRadxWorkflowServer")
	cs.Title = strPtr("go-radx workflow server")
	status := r4.PublicationStatusActive
	cs.Status = &status
	cs.Date = strPtr("2026-06-01")
	cs.Publisher = strPtr("go-radx")
	cs.Description = strPtr("Synthetic capability statement for the go-radx FHIR conformance gate")
	kind := r4.CapabilityStatementKindInstance
	cs.Kind = &kind
	// cpb-14: a kind=instance statement must carry an implementation element.
	cs.Implementation = &r4.CapabilityStatementImplementation{
		Description: strPtr("go-radx workflow server instance"),
		URL:         strPtr("https://go-radx.test/fhir"),
	}
	fhirVersion := r4.FHIRVersionN401
	cs.FhirVersion = &fhirVersion
	cs.Format = []string{"json"}
	mode := r4.RestfulCapabilityModeServer
	cs.Rest = []r4.CapabilityStatementRest{{
		Mode:          &mode,
		Documentation: strPtr("Synthetic RESTful capability"),
	}}
	return cs
}

// workflowBundle assembles a collection Bundle that holds the workflow resources by
// fullUrl, exercising the Bundle builder and intra-bundle references on the wire.
func workflowBundle(resources ...fhir.Resource) (*r4.Bundle, error) {
	entries := make([]r4.CollectionEntry, 0, len(resources))
	for _, r := range resources {
		entries = append(entries, r4.CollectionEntry{
			FullURL:  "http://example.org/go-radx/" + r.ResourceType(),
			Resource: r,
		})
	}
	return r4.NewCollection(entries...)
}
