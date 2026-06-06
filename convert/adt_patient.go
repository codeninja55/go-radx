package convert

import (
	"fmt"

	"github.com/codeninja55/go-radx/fhir/r5"
	"github.com/codeninja55/go-radx/hl7v2"
)

// ADTToPatientR5 converts an HL7 v2 admission/discharge/transfer message (ADT^Axx)
// to a FHIR R5 Patient. It reads PID (identity, name, gender, birth date, address);
// PD1 (additional demographics) carries no element the v1 Patient mapping reads, so
// its presence does not change the output. The PID-8 administrative sex is mapped
// through the value-set-safe ParseAdministrativeGender, recording a Substitution
// when the source code is outside HL7 Table 0001. A message that is not an ADT is
// rejected with ErrUnsupportedSource.
//
// Patient has no FHIR-required field, so the conversion never fails closed on a
// sparse PID; an absent element is simply left unset.
func ADTToPatientR5(msg *hl7v2.Message, opts ...Option) (*r5.Patient, *Report, error) {
	_ = newConfig(opts...)
	report := &Report{}

	if msg == nil {
		return nil, nil, fmt.Errorf("%w: message is nil", ErrMalformedSource)
	}
	adt, ok := hl7v2.AsADT(msg)
	if !ok {
		return nil, nil, fmt.Errorf("%w: MSH-9.1 is not ADT", ErrUnsupportedSource)
	}

	pid, hasPID := adt.PID()
	if !hasPID {
		return nil, nil, fmt.Errorf("%w: ADT carries no PID for the Patient demographics",
			ErrMalformedSource)
	}

	pat := &r5.Patient{}

	for _, cx := range pid.AllPatientIDs {
		if cx.ID == "" {
			continue
		}
		pat.Identifier = append(pat.Identifier, cxToIdentifierR5(cx))
	}

	if name := patientName(pid.PatientName); name != nil {
		pat.Name = []r5.HumanName{*name}
	}

	if gender, substituted := ParseAdministrativeGender(pid.Sex); pid.Sex != "" || substituted {
		g := gender
		pat.Gender = &g
		if substituted {
			report.substituted("Patient.gender", string(gender),
				"the PID-8 administrative sex code is not in HL7 Table 0001; mapped to the value-set-safe approximation")
		}
	}

	if bd := birthDate(pid.BirthDate, report); bd != "" {
		pat.BirthDate = &bd
	}

	if addr := patientAddress(pid.Address); addr != nil {
		pat.Address = []r5.Address{*addr}
	}

	return pat, report, nil
}

// ParseAdministrativeGender maps an HL7 v2 administrative-sex code (PID-8, Table
// 0001) to a FHIR AdministrativeGender, value-set-safe: M→male, F→female, O→other,
// A (ambiguous) and N (not applicable)→other, U and the empty value→unknown. Any
// code outside Table 0001 maps to "unknown" and the bool is true, signalling the
// caller to record a Substitution. The result is always a member of the required
// AdministrativeGender value set, so the produced Patient validates by construction.
func ParseAdministrativeGender(code string) (r5.AdministrativeGender, bool) {
	switch code {
	case "M":
		return r5.AdministrativeGenderMale, false
	case "F":
		return r5.AdministrativeGenderFemale, false
	case "O", "A", "N":
		return r5.AdministrativeGenderOther, false
	case "U", "":
		return r5.AdministrativeGenderUnknown, false
	default:
		return r5.AdministrativeGenderUnknown, true
	}
}

// patientName maps an HL7 XPN (PID-5) to a FHIR HumanName, or nil when the name
// carries no family or given component. The family becomes HumanName.family, the
// given and middle names HumanName.given, the suffix and prefix the matching lists.
func patientName(xpn hl7v2.XPN) *r5.HumanName {
	if xpn.Family == "" && xpn.Given == "" {
		return nil
	}
	name := &r5.HumanName{}
	if xpn.Family != "" {
		family := xpn.Family
		name.Family = &family
	}
	if xpn.Given != "" {
		name.Given = append(name.Given, xpn.Given)
	}
	if xpn.Middle != "" {
		name.Given = append(name.Given, xpn.Middle)
	}
	if xpn.Prefix != "" {
		name.Prefix = append(name.Prefix, xpn.Prefix)
	}
	if xpn.Suffix != "" {
		name.Suffix = append(name.Suffix, xpn.Suffix)
	}
	return name
}

// patientAddress maps an HL7 XAD (PID-11) to a FHIR Address, or nil when the
// address carries no postal component.
func patientAddress(xad hl7v2.XAD) *r5.Address {
	if xad.Street == "" && xad.City == "" && xad.State == "" && xad.Zip == "" && xad.Country == "" {
		return nil
	}
	addr := &r5.Address{}
	if xad.Street != "" {
		addr.Line = append(addr.Line, xad.Street)
	}
	if xad.OtherDesignation != "" {
		addr.Line = append(addr.Line, xad.OtherDesignation)
	}
	if xad.City != "" {
		city := xad.City
		addr.City = &city
	}
	if xad.State != "" {
		state := xad.State
		addr.State = &state
	}
	if xad.Zip != "" {
		zip := xad.Zip
		addr.PostalCode = &zip
	}
	if xad.Country != "" {
		country := xad.Country
		addr.Country = &country
	}
	return addr
}

// birthDate renders an HL7 DTM (PID-7) as a FHIR date, preserving the source
// precision. A timed birth-date DTM is reduced to its date components, since FHIR
// birthDate is a date, never a dateTime; the dropped time precision is recorded.
// "" when the DTM is absent.
func birthDate(dtm hl7v2.DTM, report *Report) string {
	t, prec, ok := dtm.Time()
	if !ok {
		return ""
	}
	switch prec {
	case hl7v2.PrecisionYear:
		return pad4(t.Year())
	case hl7v2.PrecisionMonth:
		return pad4(t.Year()) + "-" + pad2(int(t.Month()))
	default:
		if prec > hl7v2.PrecisionDay {
			report.dropped("PID-7 DateTimeOfBirth time precision",
				"FHIR Patient.birthDate is a date; the time-of-birth precision was dropped")
		}
		return pad4(t.Year()) + "-" + pad2(int(t.Month())) + "-" + pad2(t.Day())
	}
}
