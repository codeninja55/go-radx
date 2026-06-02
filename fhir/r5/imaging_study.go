package r5

import "encoding/json"

// imagingStudyResourceType is the FHIR resourceType discriminator for ImagingStudy.
const imagingStudyResourceType = "ImagingStudy"

// ImagingStudy is the minimal R5 ImagingStudy carrying only the elements the
// DICOM→ImagingStudy converter populates. Modality is a list of CodeableConcept
// at the study level in R5 (it was a single Coding in R4). NumberOfSeries and
// NumberOfInstances are the FHIR unsignedInt primitive, held as *uint32 so an
// absent count is distinguishable from a present zero. Fields are ordered per the
// R5 StructureDefinition so the JSON keys are stable.
type ImagingStudy struct {
	Identifier        []Identifier         `json:"identifier,omitempty"`
	Status            string               `json:"status,omitempty"`
	Subject           *Reference           `json:"subject,omitempty"`
	Started           *string              `json:"started,omitempty"`
	NumberOfSeries    *uint32              `json:"numberOfSeries,omitempty"`
	NumberOfInstances *uint32              `json:"numberOfInstances,omitempty"`
	Modality          []CodeableConcept    `json:"modality,omitempty"`
	Description       *string              `json:"description,omitempty"`
	Series            []ImagingStudySeries `json:"series,omitempty"`
}

// ImagingStudySeries is one series within an ImagingStudy. Uid is the required
// Series Instance UID (FHIR id), so it is a plain string the converter always
// sets. Modality is a required CodeableConcept (R5; Coding in R4). BodySite is a
// CodeableReference in R5 (it was a Coding in R4).
type ImagingStudySeries struct {
	Uid         string                       `json:"uid,omitempty"`
	Number      *uint32                      `json:"number,omitempty"`
	Modality    CodeableConcept              `json:"modality"`
	Description *string                      `json:"description,omitempty"`
	BodySite    *CodeableReference           `json:"bodySite,omitempty"`
	Instance    []ImagingStudySeriesInstance `json:"instance,omitempty"`
}

// ImagingStudySeriesInstance is one SOP instance within a series. Uid is the
// required SOP Instance UID (FHIR id) and SopClass the required SOP Class UID as
// a Coding, so the converter always sets both from (0008,0018) and (0008,0016).
type ImagingStudySeriesInstance struct {
	Uid      string  `json:"uid,omitempty"`
	SopClass Coding  `json:"sopClass"`
	Number   *uint32 `json:"number,omitempty"`
}

// ResourceType returns the FHIR discriminator "ImagingStudy".
func (i *ImagingStudy) ResourceType() string { return imagingStudyResourceType }

// MarshalJSON emits the resource with "resourceType" as the first JSON key.
func (i *ImagingStudy) MarshalJSON() ([]byte, error) {
	type alias ImagingStudy
	return json.Marshal(struct {
		ResourceType string `json:"resourceType"`
		*alias
	}{
		ResourceType: imagingStudyResourceType,
		alias:        (*alias)(i),
	})
}
