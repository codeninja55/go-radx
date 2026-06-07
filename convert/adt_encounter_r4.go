package convert

import (
	"fmt"

	"github.com/codeninja55/go-radx/fhir/r4"
	"github.com/codeninja55/go-radx/hl7v2"
)

// ADTToEncounterR4 converts an HL7 v2 admission/discharge/transfer message (ADT^Axx)
// to a FHIR R4 Encounter, the R4 twin of ADTToEncounterR5. The trigger-event to
// status mapping, the PV1-2 patient-class mapping, the PV1-19 visit identifier, and
// the PID-3 logical subject are identical. The R4 output differs in one
// load-bearing way the R4 resource model imposes:
//
//   - Encounter.class is a single Coding in R4 (a list of CodeableConcept in R5),
//     so the mapped class is a Coding, not a CodeableConcept list element.
//
// The trigger-event to status mapping is approximate (A01->in-progress,
// A03->finished, A11->cancelled); each approximation, and any trigger with no exact
// FHIR status (mapped to "unknown"), records a Substitution. status is always a
// member of the required R4 EncounterStatus value set, so the produced Encounter
// validates by construction.
func ADTToEncounterR4(msg *hl7v2.Message, opts ...Option) (*r4.Encounter, *Report, error) {
	cfg := newConfig(opts...)
	report := &Report{}

	if msg == nil {
		return nil, nil, fmt.Errorf("%w: message is nil", ErrMalformedSource)
	}
	adt, ok := hl7v2.AsADT(msg)
	if !ok {
		return nil, nil, fmt.Errorf("%w: MSH-9.1 is not ADT", ErrUnsupportedSource)
	}

	enc := &r4.Encounter{}

	status := encounterStatusR4(adt.Event(), report)
	enc.Status = &status

	if pv1, hasPV1 := adt.PV1(); hasPV1 {
		if pv1.VisitNumber.ID != "" {
			enc.Identifier = append(enc.Identifier, cxToIdentifierR4(pv1.VisitNumber))
		}
		if class := encounterClassR4(pv1.PatientClass, report); class != nil {
			enc.Class = class
		}
	}

	if subject := patientSubjectR4(cfg, msg, report, "Encounter.subject"); subject != nil {
		enc.Subject = subject
	}

	return enc, report, nil
}

// encounterStatusR4 maps an ADT trigger event (MSH-9.2 / EVN-1) to an R4
// Encounter.status, the R4 twin of encounterStatus. The mapping is the same
// approximation, but the R4 EncounterStatus value set names the completed state
// "finished" (R5 renames it "completed"), so A03 maps to finished here. Every
// recognised mapping records a Substitution because the HL7 trigger event and the
// FHIR encounter status are not the same concept; an unrecognised trigger maps to
// the value-set-safe "unknown".
func encounterStatusR4(event string, report *Report) r4.EncounterStatus {
	switch event {
	case "A01":
		report.substituted("Encounter.status", string(r4.EncounterStatusInProgress),
			"the ADT trigger event A01 has no exact FHIR encounter status; approximated to in-progress")
		return r4.EncounterStatusInProgress
	case "A03":
		report.substituted("Encounter.status", string(r4.EncounterStatusFinished),
			"the ADT trigger event A03 has no exact FHIR encounter status; approximated to finished")
		return r4.EncounterStatusFinished
	case "A11":
		report.substituted("Encounter.status", string(r4.EncounterStatusCancelled),
			"the ADT trigger event A11 has no exact FHIR encounter status; approximated to cancelled")
		return r4.EncounterStatusCancelled
	default:
		report.substituted("Encounter.status", string(r4.EncounterStatusUnknown),
			"the ADT trigger event has no mapped FHIR encounter status; approximated to unknown")
		return r4.EncounterStatusUnknown
	}
}

// encounterClassR4 maps an HL7 v2 PV1-2 patient class to an R4 Encounter.class
// Coding under the v3 ActCode system, the R4 twin of encounterClass. R4 binds
// Encounter.class to a single Coding (R5 widened it to a CodeableConcept list), so
// the mapped class is a Coding. An unrecognised class is carried verbatim under the
// v3 ActCode system and recorded as a Substitution. nil when PV1-2 is empty.
func encounterClassR4(class string, report *Report) *r4.Coding {
	if class == "" {
		return nil
	}
	code, display, mapped := actEncounterClass(class)
	if !mapped {
		report.substituted("Encounter.class", code,
			"the PV1-2 patient class is not in the mapped set; carried verbatim under the v3 ActCode system")
	}
	system := patientClassSystem
	coding := &r4.Coding{System: &system, Code: &code}
	if display != "" {
		d := display
		coding.Display = &d
	}
	return coding
}
