package convert

import (
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/hl7v2"
)

// seededPHI are patient values planted in the ORU and ADT fixtures below. No
// Report entry (Dropped, Defaulted, Substituted), no Substitution field, and no
// returned error string may contain any of them: the report names concepts and
// loci, never patient data (PRD §9.1).
var seededPHI = []string{
	"555-44-4444",            // MRN (PID-3)
	"EVERYWOMAN",             // family name (PID-5)
	"EVE",                    // given name (PID-5)
	"19620320",               // birth date (PID-7)
	"123 MAIN ST",            // street (PID-11)
	"METROPOLIS",             // city (PID-11)
	"VISIT-9001",             // visit number (PV1-19)
	"242",                    // observed numeric value (OBX-5)
	"Sample mildly lipemic.", // narrative (OBX-5)
}

// phiORU and phiADT plant every seeded value, plus an unmappable OBX value type
// and an out-of-table gender and trigger event so the Substituted and Dropped
// channels are populated and can be audited for leaks.
const phiORU = "MSH|^~\\&|LIS|HOSP|EMR|HOSP|202605311230||ORU^R01|MSGORU1|P|2.4\r" +
	"PID|||555-44-4444^^^HOSP^MR||EVERYWOMAN^EVE^E^^^^L||19620320|F|||123 MAIN ST^^METROPOLIS^NY^12345^USA\r" +
	"OBR|1|PLACER123|FILLER456|24331-1^LIPID PANEL^LN|||202605311231\r" +
	"OBX|1|NM|2093-3^CHOLESTEROL^LN||242|mg/dL^mg/dL^UCUM|0-200|H|||F\r" +
	"OBX|2|ZZ|9999-9^UNKNOWN^LN||Sample mildly lipemic.|||||\r"

const phiADT = "MSH|^~\\&|ADT1|HOSP|EMR|HOSP|202605311230||ADT^A99|MSGADT1|P|2.4\r" +
	"EVN|A99|202605311230\r" +
	"PID|||555-44-4444^^^HOSP^MR||EVERYWOMAN^EVE^E^^^^L||19620320|ZZ|||123 MAIN ST^^METROPOLIS^NY^12345^USA\r" +
	"PV1|1|I|WARD3^301^A||||||||||||||||VISIT-9001\r"

func TestORUNoPHIInReport(t *testing.T) {
	msg, err := hl7v2.Parse([]byte(phiORU))
	if err != nil {
		t.Fatalf("parse ORU: %v", err)
	}
	dr, obs, report, err := ORUToDiagnosticReportR5(msg)
	if err != nil {
		t.Fatalf("ORUToDiagnosticReportR5: %v", err)
	}
	if dr == nil || len(obs) == 0 {
		t.Fatal("conversion produced no resources")
	}
	assertNoPHIInReport(t, report)
}

func TestADTNoPHIInReport(t *testing.T) {
	msg, err := hl7v2.Parse([]byte(phiADT))
	if err != nil {
		t.Fatalf("parse ADT: %v", err)
	}
	pat, patReport, err := ADTToPatientR5(msg)
	if err != nil {
		t.Fatalf("ADTToPatientR5: %v", err)
	}
	if pat == nil {
		t.Fatal("Patient is nil")
	}
	assertNoPHIInReport(t, patReport)

	enc, encReport, err := ADTToEncounterR5(msg)
	if err != nil {
		t.Fatalf("ADTToEncounterR5: %v", err)
	}
	if enc == nil {
		t.Fatal("Encounter is nil")
	}
	assertNoPHIInReport(t, encReport)
}

// assertNoPHIInReport fails the test if any seeded patient value appears in any
// Report channel.
func assertNoPHIInReport(t *testing.T, report *Report) {
	t.Helper()
	if report == nil {
		return
	}
	var blob strings.Builder
	for _, d := range report.Dropped {
		blob.WriteString(d.Source)
		blob.WriteString(d.Reason)
	}
	for _, d := range report.Defaulted {
		blob.WriteString(d.Target)
		blob.WriteString(d.Value)
		blob.WriteString(d.Reason)
	}
	for _, s := range report.Substituted {
		blob.WriteString(s.Concept)
		blob.WriteString(s.Approximation)
		blob.WriteString(s.Reason)
	}
	text := blob.String()
	for _, phi := range seededPHI {
		if strings.Contains(text, phi) {
			t.Errorf("Report leaks seeded patient value %q", phi)
		}
	}
}
