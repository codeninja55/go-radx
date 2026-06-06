package hl7v2

import "testing"

const canonicalADT = "MSH|^~\\&|RADIS|HOSP|PACS|HOSP|202605311230||ADT^A01|MSG00010|P|2.5\r" +
	"EVN|A01|202605311230|||OP123\r" +
	"PID|||555-44-4444^^^HOSP^MR||EVERYWOMAN^EVE^E^^^^L||19620320|F\r" +
	"PV1|1|I|ICU^101^A||||7^DOE^JANE^^^^L||||||||||||V123^^^HOSP^VN\r"

const canonicalORU = "MSH|^~\\&|GHH LAB|ELAB-3|GHH OE|BLDG4|200202150930||ORU^R01|CNTRL-3456|P|2.4\r" +
	"PID|||555-44-4444||EVERYWOMAN^EVE^E^^^^L||19620320|F\r" +
	"OBR|1|845439^GHH OE|1045813^GHH LAB|1554-5^GLUCOSE\r" +
	"OBX|1|SN|1554-5^GLUCOSE^POST 12H CFST||182|mg/dl|70_105|H|||F\r" +
	"OBX|2|NM|1556-0^HEMATOCRIT^LN||42|%|36_48|N|||F\r" +
	"OBR|2|845440^GHH OE|1045814^GHH LAB|2000-8^CALCIUM\r" +
	"OBX|1|NM|2000-8^CALCIUM^LN||9.5|mg/dl|8.5_10.2|N|||F\r"

const canonicalACK = "MSH|^~\\&|PACS|HOSP|RADIS|HOSP|202605311230||ACK^O01^ACK|ACK00001|P|2.4\r" +
	"MSA|AE|MSG00001|OBX-5 failed datatype validation\r" +
	"ERR||OBX^1^5|207^internal^HL70357|E\r"

func TestAsADT(t *testing.T) {
	msg, err := Parse([]byte(canonicalADT))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	adt, ok := AsADT(msg)
	if !ok {
		t.Fatal("AsADT = false, want true")
	}
	if adt.Event() != "A01" {
		t.Errorf("Event = %q, want A01", adt.Event())
	}
	if evn, ok := adt.EVN(); !ok || evn.EventTypeCode != "A01" {
		t.Errorf("EVN = %+v ok=%v, want EventTypeCode A01", evn, ok)
	}
	if pid, ok := adt.PID(); !ok || pid.PatientName.Family != "EVERYWOMAN" {
		t.Errorf("PID = %+v ok=%v, want family EVERYWOMAN", pid, ok)
	}
	if pv1, ok := adt.PV1(); !ok || pv1.PatientClass != "I" || pv1.VisitNumber.ID != "V123" {
		t.Errorf("PV1 = %+v ok=%v, want class I visit V123", pv1, ok)
	}
}

func TestAsADTRejectsNonADT(t *testing.T) {
	msg, _ := Parse([]byte(canonicalORU))
	if _, ok := AsADT(msg); ok {
		t.Error("AsADT on an ORU = true, want false")
	}
}

func TestAsORU(t *testing.T) {
	msg, err := Parse([]byte(canonicalORU))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	oru, ok := AsORU(msg)
	if !ok {
		t.Fatal("AsORU = false, want true")
	}
	if pid, ok := oru.PID(); !ok || pid.PatientName.Family != "EVERYWOMAN" {
		t.Errorf("PID = %+v ok=%v, want family EVERYWOMAN", pid, ok)
	}

	var groups []ResultGroup
	for g := range oru.Results() {
		groups = append(groups, g)
	}
	if len(groups) != 2 {
		t.Fatalf("Results yielded %d groups, want 2", len(groups))
	}
	if groups[0].Order.UniversalServiceID.Code != "1554-5" {
		t.Errorf("group[0] order = %q, want 1554-5", groups[0].Order.UniversalServiceID.Code)
	}
	if len(groups[0].Observations) != 2 {
		t.Errorf("group[0] observations = %d, want 2", len(groups[0].Observations))
	}
	if groups[1].Order.UniversalServiceID.Code != "2000-8" {
		t.Errorf("group[1] order = %q, want 2000-8", groups[1].Order.UniversalServiceID.Code)
	}
	if len(groups[1].Observations) != 1 {
		t.Errorf("group[1] observations = %d, want 1", len(groups[1].Observations))
	}
}

func TestAsORURejectsNonORU(t *testing.T) {
	msg, _ := Parse([]byte(canonicalADT))
	if _, ok := AsORU(msg); ok {
		t.Error("AsORU on an ADT = true, want false")
	}
}

// TestORUResultsEarlyStop confirms the iterator honours a break by the consumer,
// the contract for iter.Seq.
func TestORUResultsEarlyStop(t *testing.T) {
	msg, _ := Parse([]byte(canonicalORU))
	oru, _ := AsORU(msg)
	count := 0
	for range oru.Results() {
		count++
		break
	}
	if count != 1 {
		t.Errorf("early-stopped iteration visited %d groups, want 1", count)
	}
}

func TestAsACK(t *testing.T) {
	msg, err := Parse([]byte(canonicalACK))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	ack, ok := AsACK(msg)
	if !ok {
		t.Fatal("AsACK = false, want true")
	}
	msa, ok := ack.MSA()
	if !ok {
		t.Fatal("MSA = false, want true")
	}
	if msa.AckCode != AckError || msa.ControlID != "MSG00001" {
		t.Errorf("MSA = %+v, want code AE control MSG00001", msa)
	}
	errs := ack.Errors()
	if len(errs) != 1 {
		t.Fatalf("Errors = %d, want 1", len(errs))
	}
	if errs[0].Severity != "E" || errs[0].Code.Code != "207" {
		t.Errorf("ERR = %+v, want severity E code 207", errs[0])
	}
}

func TestAsACKRejectsNonACK(t *testing.T) {
	msg, _ := Parse([]byte(canonicalORU))
	if _, ok := AsACK(msg); ok {
		t.Error("AsACK on an ORU = true, want false")
	}
}

// TestLensRejectsMissingMSH proves a malformed message with no MSH is not viewed
// as any typed message. Parse already rejects such input, so the lenses inherit
// the guard through MSH(); this asserts the bool channel rather than a panic.
func TestLensRejectsMissingMSH(t *testing.T) {
	m := &Message{}
	if _, ok := AsADT(m); ok {
		t.Error("AsADT on an MSH-less message = true, want false")
	}
	if _, ok := AsORU(m); ok {
		t.Error("AsORU on an MSH-less message = true, want false")
	}
	if _, ok := AsACK(m); ok {
		t.Error("AsACK on an MSH-less message = true, want false")
	}
}
