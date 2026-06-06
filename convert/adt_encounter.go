package convert

import (
	"fmt"

	"github.com/codeninja55/go-radx/fhir/r5"
	"github.com/codeninja55/go-radx/hl7v2"
)

// patientClassSystem is the FHIR code system for the Encounter class. The HL7 v2
// PV1-2 patient-class codes (I, O, E, ...) and the FHIR v3 ActCode encounter-class
// codes (IMP, AMB, EMER, ...) differ, so the class is mapped, not carried verbatim.
const patientClassSystem = "http://terminology.hl7.org/CodeSystem/v3-ActCode"

// ADTToEncounterR5 converts an HL7 v2 admission/discharge/transfer message (ADT^Axx)
// to a FHIR R5 Encounter. It reads the MSH-9.2 / EVN-1 trigger event (mapped to the
// encounter status), PV1-2 patient class (mapped to the encounter class), PV1-19
// Visit Number (the logical identifier), and PID-3 (the logical subject). A message
// that is not an ADT is rejected with ErrUnsupportedSource.
//
// The trigger-event to status mapping is approximate: A01→in-progress, A03→completed,
// A11→cancelled. Each approximation, and any trigger event with no exact FHIR status
// (mapped to "unknown"), records a Substitution naming Encounter.status. status is
// always a member of the required EncounterStatus value set, so the produced Encounter
// validates by construction.
func ADTToEncounterR5(msg *hl7v2.Message, opts ...Option) (*r5.Encounter, *Report, error) {
	cfg := newConfig(opts...)
	report := &Report{}

	if msg == nil {
		return nil, nil, fmt.Errorf("%w: message is nil", ErrMalformedSource)
	}
	adt, ok := hl7v2.AsADT(msg)
	if !ok {
		return nil, nil, fmt.Errorf("%w: MSH-9.1 is not ADT", ErrUnsupportedSource)
	}

	enc := &r5.Encounter{}

	status := encounterStatus(adt.Event(), report)
	enc.Status = &status

	if pv1, hasPV1 := adt.PV1(); hasPV1 {
		if pv1.VisitNumber.ID != "" {
			enc.Identifier = append(enc.Identifier, cxToIdentifierR5(pv1.VisitNumber))
		}
		if class := encounterClass(pv1.PatientClass, report); class != nil {
			enc.Class = []r5.CodeableConcept{*class}
		}
	}

	if subject := patientSubjectR5(cfg, msg, report, "Encounter.subject"); subject != nil {
		enc.Subject = subject
	}

	return enc, report, nil
}

// encounterStatus maps an ADT trigger event (MSH-9.2 / EVN-1) to a FHIR
// Encounter.status. The mapping is intentionally narrow and approximate: A01 (admit)
// to in-progress, A03 (discharge) to completed, A11 (cancel admit) to cancelled.
// Every recognised mapping records a Substitution because the HL7 trigger event and
// the FHIR encounter status are not the same concept; an unrecognised trigger maps to
// the value-set-safe "unknown" and records a Substitution naming the concept.
func encounterStatus(event string, report *Report) r5.EncounterStatus {
	switch event {
	case "A01":
		report.substituted("Encounter.status", string(r5.EncounterStatusInProgress),
			"the ADT trigger event A01 has no exact FHIR encounter status; approximated to in-progress")
		return r5.EncounterStatusInProgress
	case "A03":
		report.substituted("Encounter.status", string(r5.EncounterStatusCompleted),
			"the ADT trigger event A03 has no exact FHIR encounter status; approximated to completed")
		return r5.EncounterStatusCompleted
	case "A11":
		report.substituted("Encounter.status", string(r5.EncounterStatusCancelled),
			"the ADT trigger event A11 has no exact FHIR encounter status; approximated to cancelled")
		return r5.EncounterStatusCancelled
	default:
		report.substituted("Encounter.status", string(r5.EncounterStatusUnknown),
			"the ADT trigger event has no mapped FHIR encounter status; approximated to unknown")
		return r5.EncounterStatusUnknown
	}
}

// encounterClass maps an HL7 v2 PV1-2 patient class to a FHIR Encounter.class
// CodeableConcept under the v3 ActCode system: I→IMP (inpatient), O→AMB (ambulatory),
// E→EMER (emergency), P→PRENC (pre-admission), R→AMB (recurring), B→AMB (obstetrics).
// An unrecognised class is carried verbatim under the v3 ActCode system and recorded
// as a Substitution, since Encounter.class is example-bound and a preserved value is
// preferable to a dropped one. nil when PV1-2 is empty.
func encounterClass(class string, report *Report) *r5.CodeableConcept {
	if class == "" {
		return nil
	}
	code, display, mapped := actEncounterClass(class)
	if !mapped {
		report.substituted("Encounter.class", code,
			"the PV1-2 patient class is not in the mapped set; carried verbatim under the v3 ActCode system")
	}
	system := patientClassSystem
	coding := r5.Coding{System: &system, Code: &code}
	if display != "" {
		d := display
		coding.Display = &d
	}
	return &r5.CodeableConcept{Coding: []r5.Coding{coding}}
}

// actEncounterClass maps a PV1-2 patient class to its v3 ActCode code and display.
// mapped is false for an unrecognised class, which is returned as its own code with
// no display so the value is preserved verbatim.
func actEncounterClass(class string) (code, display string, mapped bool) {
	switch class {
	case "I":
		return "IMP", "inpatient encounter", true
	case "O":
		return "AMB", "ambulatory", true
	case "E":
		return "EMER", "emergency", true
	case "P":
		return "PRENC", "pre-admission", true
	case "R", "B":
		return "AMB", "ambulatory", true
	default:
		return class, "", false
	}
}
