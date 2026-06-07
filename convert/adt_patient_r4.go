package convert

import (
	"fmt"

	"github.com/codeninja55/go-radx/fhir/r4"
	"github.com/codeninja55/go-radx/hl7v2"
)

// ADTToPatientR4 converts an HL7 v2 admission/discharge/transfer message (ADT^Axx)
// to a FHIR R4 Patient, the R4 twin of ADTToPatientR5. The PID reading, the
// value-set-safe gender mapping, and the birth-date precision handling are
// identical; the Patient resource model is the same in R4 and R5 for the v1
// mapping's fields, so the twin differs only in the release sub-package its types
// and the AdministrativeGender value set come from.
//
// Patient has no FHIR-required field, so the conversion never fails closed on a
// sparse PID. A lossy reduction (a PID-7 birth date carrying time precision
// Patient.birthDate cannot hold) is recorded on the Report and escalated to a
// *LossError when WithStrictLoss is set.
func ADTToPatientR4(msg *hl7v2.Message, opts ...Option) (*r4.Patient, *Report, error) {
	cfg := newConfig(opts...)
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

	pat := &r4.Patient{}

	for _, cx := range pid.AllPatientIDs {
		if cx.ID == "" {
			continue
		}
		pat.Identifier = append(pat.Identifier, cxToIdentifierR4(cx))
	}

	if name := patientNameR4(pid.PatientName); name != nil {
		pat.Name = []r4.HumanName{*name}
	}

	if gender, substituted := ParseAdministrativeGenderR4(pid.Sex); pid.Sex != "" || substituted {
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

	if addr := patientAddressR4(pid.Address); addr != nil {
		pat.Address = []r4.Address{*addr}
	}

	rep, err := cfg.finalize(report)
	return pat, rep, err
}

// ParseAdministrativeGenderR4 maps an HL7 v2 administrative-sex code (PID-8, Table
// 0001) to an R4 AdministrativeGender, value-set-safe, the R4 twin of
// ParseAdministrativeGender. The mapping is identical; the AdministrativeGender
// value set lives in the R4 sub-package, so the produced Patient validates by
// construction against the R4 binding.
func ParseAdministrativeGenderR4(code string) (r4.AdministrativeGender, bool) {
	switch code {
	case "M":
		return r4.AdministrativeGenderMale, false
	case "F":
		return r4.AdministrativeGenderFemale, false
	case "O", "A", "N":
		return r4.AdministrativeGenderOther, false
	case "U", "":
		return r4.AdministrativeGenderUnknown, false
	default:
		return r4.AdministrativeGenderUnknown, true
	}
}

// patientNameR4 maps an HL7 XPN (PID-5) to an R4 HumanName, the R4 twin of
// patientName, or nil when the name carries no family or given component.
func patientNameR4(xpn hl7v2.XPN) *r4.HumanName {
	if xpn.Family == "" && xpn.Given == "" {
		return nil
	}
	name := &r4.HumanName{}
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

// patientAddressR4 maps an HL7 XAD (PID-11) to an R4 Address, the R4 twin of
// patientAddress, or nil when the address carries no postal component.
func patientAddressR4(xad hl7v2.XAD) *r4.Address {
	if xad.Street == "" && xad.City == "" && xad.State == "" && xad.Zip == "" && xad.Country == "" {
		return nil
	}
	addr := &r4.Address{}
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
