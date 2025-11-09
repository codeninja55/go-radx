package resources

import (
	"fmt"

	"github.com/codeninja55/go-radx/fhir/validation"
)

// Validate performs validation on the Patient resource.
// This is a demonstration of the validation pattern that can be generated
// or implemented manually for all FHIR resources.
func (p *Patient) Validate() error {
	errs := &validation.Errors{}

	// Validate ID format (if present)
	if p.ID != nil && *p.ID == "" {
		errs.Add("Patient.id", "id cannot be empty if present")
	}

	// Validate gender (if present, must be one of the allowed values)
	if p.Gender != nil {
		validGenders := map[string]bool{
			"male":    true,
			"female":  true,
			"other":   true,
			"unknown": true,
		}
		if !validGenders[*p.Gender] {
			errs.Addf("Patient.gender", "invalid gender value: %s (must be male|female|other|unknown)", *p.Gender)
		}
	}

	// Validate HumanName entries
	for i, name := range p.Name {
		if err := validateHumanName(name, i); err != nil {
			errs.Add(fmt.Sprintf("Patient.name[%d]", i), fmt.Sprintf("invalid name: %v", err))
		}
	}

	// Validate ContactPoint entries (telecom)
	for i, telecom := range p.Telecom {
		if err := validateContactPoint(telecom, i); err != nil {
			errs.Add(fmt.Sprintf("Patient.telecom[%d]", i), fmt.Sprintf("invalid telecom: %v", err))
		}
	}

	// Validate Address entries
	for i, addr := range p.Address {
		if err := validateAddress(addr, i); err != nil {
			errs.Add(fmt.Sprintf("Patient.address[%d]", i), fmt.Sprintf("invalid address: %v", err))
		}
	}

	// Validate Photo attachments
	for i, photo := range p.Photo {
		if err := validateAttachment(photo, i); err != nil {
			errs.Add(fmt.Sprintf("Patient.photo[%d]", i), fmt.Sprintf("invalid photo: %v", err))
		}
	}

	// Validate Contact (emergency contact) entries
	for i, contact := range p.Contact {
		if err := validatePatientContact(contact, i); err != nil {
			errs.Add(fmt.Sprintf("Patient.contact[%d]", i), fmt.Sprintf("invalid contact: %v", err))
		}
	}

	// Validate Communication entries
	for i, comm := range p.Communication {
		if err := validatePatientCommunication(comm, i); err != nil {
			errs.Add(fmt.Sprintf("Patient.communication[%d]", i), fmt.Sprintf("invalid communication: %v", err))
		}
	}

	// Validate GeneralPractitioner references
	for i, ref := range p.GeneralPractitioner {
		if err := validateReference(ref, "Patient.generalPractitioner", i); err != nil {
			errs.Add(fmt.Sprintf("Patient.generalPractitioner[%d]", i), fmt.Sprintf("%v", err))
		}
	}

	// Validate ManagingOrganization reference
	if p.ManagingOrganization != nil {
		if err := validateReference(*p.ManagingOrganization, "Patient.managingOrganization", -1); err != nil {
			errs.Add("Patient.managingOrganization", err.Error())
		}
	}

	// Validate Link entries
	for i, link := range p.Link {
		if err := validatePatientLink(link, i); err != nil {
			errs.Add(fmt.Sprintf("Patient.link[%d]", i), fmt.Sprintf("invalid link: %v", err))
		}
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

// Helper validation functions

func validateHumanName(name HumanName, index int) error {
	errs := &validation.Errors{}

	// At least one of family or given must be present
	if (name.Family == nil || *name.Family == "") && len(name.Given) == 0 {
		errs.Add("HumanName", "must have either family or given name")
	}

	// Validate use (if present)
	if name.Use != nil {
		validUses := map[string]bool{
			"usual":     true,
			"official":  true,
			"temp":      true,
			"nickname":  true,
			"anonymous": true,
			"old":       true,
			"maiden":    true,
		}
		if !validUses[*name.Use] {
			errs.Addf("HumanName.use", "invalid use value: %s", *name.Use)
		}
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

func validateContactPoint(cp ContactPoint, index int) error {
	errs := &validation.Errors{}

	// Validate system (if present)
	if cp.System != nil {
		validSystems := map[string]bool{
			"phone": true,
			"fax":   true,
			"email": true,
			"pager": true,
			"url":   true,
			"sms":   true,
			"other": true,
		}
		if !validSystems[*cp.System] {
			errs.Addf("ContactPoint.system", "invalid system value: %s", *cp.System)
		}
	}

	// Validate use (if present)
	if cp.Use != nil {
		validUses := map[string]bool{
			"home":   true,
			"work":   true,
			"temp":   true,
			"old":    true,
			"mobile": true,
		}
		if !validUses[*cp.Use] {
			errs.Addf("ContactPoint.use", "invalid use value: %s", *cp.Use)
		}
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

func validateAddress(addr Address, index int) error {
	errs := &validation.Errors{}

	// Validate use (if present)
	if addr.Use != nil {
		validUses := map[string]bool{
			"home": true,
			"work": true,
			"temp": true,
			"old":  true,
		}
		if !validUses[*addr.Use] {
			errs.Addf("Address.use", "invalid use value: %s", *addr.Use)
		}
	}

	// Validate type (if present)
	if addr.Type != nil {
		validTypes := map[string]bool{
			"postal":   true,
			"physical": true,
			"both":     true,
		}
		if !validTypes[*addr.Type] {
			errs.Addf("Address.type", "invalid type value: %s", *addr.Type)
		}
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

func validateAttachment(attach Attachment, index int) error {
	// Basic attachment validation
	// More sophisticated validation could check content type, size, etc.
	return nil
}

func validateReference(ref Reference, fieldPath string, index int) error {
	if ref.Reference == nil {
		return nil // Empty reference is valid
	}

	return validation.ValidateReference(fieldPath, *ref.Reference)
}

func validatePatientContact(contact PatientContact, index int) error {
	// Validate that at least one contact method is provided
	if contact.Name == nil && len(contact.Telecom) == 0 && contact.Address == nil {
		return &validation.Error{
			Field:   "PatientContact",
			Message: "must have at least one of name, telecom, or address",
		}
	}
	return nil
}

func validatePatientCommunication(comm PatientCommunication, index int) error {
	errs := &validation.Errors{}

	// Language is required (cardinality 1..1)
	// Since it's not a pointer in the struct, it can't be nil, but we can check if it's empty
	if len(comm.Language.Coding) == 0 && comm.Language.Text == nil {
		errs.Add("PatientCommunication.language", "language is required")
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

func validatePatientLink(link PatientLink, index int) error {
	errs := &validation.Errors{}

	// Other is required (cardinality 1..1)
	if err := validateReference(link.Other, "PatientLink.other", -1); err != nil {
		errs.Add("PatientLink.other", err.Error())
	}

	// Type is required (cardinality 1..1)
	if link.Type == "" {
		errs.Add("PatientLink.type", "type is required")
	} else {
		// Validate type value
		validTypes := map[string]bool{
			"replaced-by": true,
			"replaces":    true,
			"refer":       true,
			"seealso":     true,
		}
		if !validTypes[link.Type] {
			errs.Addf("PatientLink.type", "invalid type value: %s", link.Type)
		}
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}
