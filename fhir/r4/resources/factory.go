package resources

import (
	"encoding/json"
	"fmt"
)

// resourceTypeField is used to peek at the resourceType field in JSON.
type resourceTypeField struct {
	ResourceType string `json:"resourceType"`
}

// UnmarshalResource unmarshals JSON into the appropriate resource type based on the resourceType field.
// Returns an error if the resourceType is unknown or if unmarshaling fails.
func UnmarshalResource(data []byte) (any, error) {
	// First, peek at the resourceType field
	var typeField resourceTypeField
	if err := json.Unmarshal(data, &typeField); err != nil {
		return nil, fmt.Errorf("failed to read resourceType: %w", err)
	}

	if typeField.ResourceType == "" {
		return nil, fmt.Errorf("missing resourceType field")
	}

	// Unmarshal into the appropriate type
	resource, err := newResourceByType(typeField.ResourceType)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, resource); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s: %w", typeField.ResourceType, err)
	}

	return resource, nil
}

// newResourceByType creates a new resource instance of the specified type.
func newResourceByType(resourceType string) (any, error) {
	switch resourceType {
	case ResourceTypeAccount:
		return &Account{}, nil
	case ResourceTypeActivityDefinition:
		return &ActivityDefinition{}, nil
	case ResourceTypeAdverseEvent:
		return &AdverseEvent{}, nil
	case ResourceTypeAllergyIntolerance:
		return &AllergyIntolerance{}, nil
	case ResourceTypeAppointment:
		return &Appointment{}, nil
	case ResourceTypeAppointmentResponse:
		return &AppointmentResponse{}, nil
	case ResourceTypeAuditEvent:
		return &AuditEvent{}, nil
	case ResourceTypeBasic:
		return &Basic{}, nil
	case ResourceTypeBinary:
		return &Binary{}, nil
	case ResourceTypeBiologicallyDerivedProduct:
		return &BiologicallyDerivedProduct{}, nil
	case ResourceTypeBodyStructure:
		return &BodyStructure{}, nil
	case ResourceTypeBundle:
		return &Bundle{}, nil
	case ResourceTypeCapabilityStatement:
		return &CapabilityStatement{}, nil
	case ResourceTypeCarePlan:
		return &CarePlan{}, nil
	case ResourceTypeCareTeam:
		return &CareTeam{}, nil
	case ResourceTypeCatalogEntry:
		return &CatalogEntry{}, nil
	case ResourceTypeChargeItem:
		return &ChargeItem{}, nil
	case ResourceTypeChargeItemDefinition:
		return &ChargeItemDefinition{}, nil
	case ResourceTypeClaim:
		return &Claim{}, nil
	case ResourceTypeClaimResponse:
		return &ClaimResponse{}, nil
	case ResourceTypeClinicalImpression:
		return &ClinicalImpression{}, nil
	case ResourceTypeCodeSystem:
		return &CodeSystem{}, nil
	case ResourceTypeCommunication:
		return &Communication{}, nil
	case ResourceTypeCommunicationRequest:
		return &CommunicationRequest{}, nil
	case ResourceTypeCompartmentDefinition:
		return &CompartmentDefinition{}, nil
	case ResourceTypeComposition:
		return &Composition{}, nil
	case ResourceTypeConceptMap:
		return &ConceptMap{}, nil
	case ResourceTypeCondition:
		return &Condition{}, nil
	case ResourceTypeConsent:
		return &Consent{}, nil
	case ResourceTypeContract:
		return &Contract{}, nil
	case ResourceTypeCoverage:
		return &Coverage{}, nil
	case ResourceTypeCoverageEligibilityRequest:
		return &CoverageEligibilityRequest{}, nil
	case ResourceTypeCoverageEligibilityResponse:
		return &CoverageEligibilityResponse{}, nil
	case ResourceTypeDetectedIssue:
		return &DetectedIssue{}, nil
	case ResourceTypeDevice:
		return &Device{}, nil
	case ResourceTypeDeviceDefinition:
		return &DeviceDefinition{}, nil
	case ResourceTypeDeviceMetric:
		return &DeviceMetric{}, nil
	case ResourceTypeDeviceRequest:
		return &DeviceRequest{}, nil
	case ResourceTypeDeviceUseStatement:
		return &DeviceUseStatement{}, nil
	case ResourceTypeDiagnosticReport:
		return &DiagnosticReport{}, nil
	case ResourceTypeDocumentManifest:
		return &DocumentManifest{}, nil
	case ResourceTypeDocumentReference:
		return &DocumentReference{}, nil
	case ResourceTypeEffectEvidenceSynthesis:
		return &EffectEvidenceSynthesis{}, nil
	case ResourceTypeEncounter:
		return &Encounter{}, nil
	case ResourceTypeEndpoint:
		return &Endpoint{}, nil
	case ResourceTypeEnrollmentRequest:
		return &EnrollmentRequest{}, nil
	case ResourceTypeEnrollmentResponse:
		return &EnrollmentResponse{}, nil
	case ResourceTypeEpisodeOfCare:
		return &EpisodeOfCare{}, nil
	case ResourceTypeEventDefinition:
		return &EventDefinition{}, nil
	case ResourceTypeEvidence:
		return &Evidence{}, nil
	case ResourceTypeEvidenceVariable:
		return &EvidenceVariable{}, nil
	case ResourceTypeExampleScenario:
		return &ExampleScenario{}, nil
	case ResourceTypeExplanationOfBenefit:
		return &ExplanationOfBenefit{}, nil
	case ResourceTypeFamilyMemberHistory:
		return &FamilyMemberHistory{}, nil
	case ResourceTypeFlag:
		return &Flag{}, nil
	case ResourceTypeGoal:
		return &Goal{}, nil
	case ResourceTypeGraphDefinition:
		return &GraphDefinition{}, nil
	case ResourceTypeGroup:
		return &Group{}, nil
	case ResourceTypeGuidanceResponse:
		return &GuidanceResponse{}, nil
	case ResourceTypeHealthcareService:
		return &HealthcareService{}, nil
	case ResourceTypeImagingStudy:
		return &ImagingStudy{}, nil
	case ResourceTypeImmunization:
		return &Immunization{}, nil
	case ResourceTypeImmunizationEvaluation:
		return &ImmunizationEvaluation{}, nil
	case ResourceTypeImmunizationRecommendation:
		return &ImmunizationRecommendation{}, nil
	case ResourceTypeImplementationGuide:
		return &ImplementationGuide{}, nil
	case ResourceTypeInsurancePlan:
		return &InsurancePlan{}, nil
	case ResourceTypeInvoice:
		return &Invoice{}, nil
	case ResourceTypeLibrary:
		return &Library{}, nil
	case ResourceTypeLinkage:
		return &Linkage{}, nil
	case ResourceTypeList:
		return &List{}, nil
	case ResourceTypeLocation:
		return &Location{}, nil
	case ResourceTypeMeasure:
		return &Measure{}, nil
	case ResourceTypeMeasureReport:
		return &MeasureReport{}, nil
	case ResourceTypeMedia:
		return &Media{}, nil
	case ResourceTypeMedication:
		return &Medication{}, nil
	case ResourceTypeMedicationAdministration:
		return &MedicationAdministration{}, nil
	case ResourceTypeMedicationDispense:
		return &MedicationDispense{}, nil
	case ResourceTypeMedicationKnowledge:
		return &MedicationKnowledge{}, nil
	case ResourceTypeMedicationRequest:
		return &MedicationRequest{}, nil
	case ResourceTypeMedicationStatement:
		return &MedicationStatement{}, nil
	case ResourceTypeMedicinalProduct:
		return &MedicinalProduct{}, nil
	case ResourceTypeMedicinalProductAuthorization:
		return &MedicinalProductAuthorization{}, nil
	case ResourceTypeMedicinalProductContraindication:
		return &MedicinalProductContraindication{}, nil
	case ResourceTypeMedicinalProductIndication:
		return &MedicinalProductIndication{}, nil
	case ResourceTypeMedicinalProductIngredient:
		return &MedicinalProductIngredient{}, nil
	case ResourceTypeMedicinalProductInteraction:
		return &MedicinalProductInteraction{}, nil
	case ResourceTypeMedicinalProductManufactured:
		return &MedicinalProductManufactured{}, nil
	case ResourceTypeMedicinalProductPackaged:
		return &MedicinalProductPackaged{}, nil
	case ResourceTypeMedicinalProductPharmaceutical:
		return &MedicinalProductPharmaceutical{}, nil
	case ResourceTypeMedicinalProductUndesirableEffect:
		return &MedicinalProductUndesirableEffect{}, nil
	case ResourceTypeMessageDefinition:
		return &MessageDefinition{}, nil
	case ResourceTypeMessageHeader:
		return &MessageHeader{}, nil
	case ResourceTypeMolecularSequence:
		return &MolecularSequence{}, nil
	case ResourceTypeNamingSystem:
		return &NamingSystem{}, nil
	case ResourceTypeNutritionOrder:
		return &NutritionOrder{}, nil
	case ResourceTypeObservation:
		return &Observation{}, nil
	case ResourceTypeObservationDefinition:
		return &ObservationDefinition{}, nil
	case ResourceTypeOperationDefinition:
		return &OperationDefinition{}, nil
	case ResourceTypeOperationOutcome:
		return &OperationOutcome{}, nil
	case ResourceTypeOrganization:
		return &Organization{}, nil
	case ResourceTypeOrganizationAffiliation:
		return &OrganizationAffiliation{}, nil
	case ResourceTypeParameters:
		return &Parameters{}, nil
	case ResourceTypePatient:
		return &Patient{}, nil
	case ResourceTypePaymentNotice:
		return &PaymentNotice{}, nil
	case ResourceTypePaymentReconciliation:
		return &PaymentReconciliation{}, nil
	case ResourceTypePerson:
		return &Person{}, nil
	case ResourceTypePlanDefinition:
		return &PlanDefinition{}, nil
	case ResourceTypePractitioner:
		return &Practitioner{}, nil
	case ResourceTypePractitionerRole:
		return &PractitionerRole{}, nil
	case ResourceTypeProcedure:
		return &Procedure{}, nil
	case ResourceTypeProvenance:
		return &Provenance{}, nil
	case ResourceTypeQuestionnaire:
		return &Questionnaire{}, nil
	case ResourceTypeQuestionnaireResponse:
		return &QuestionnaireResponse{}, nil
	case ResourceTypeRelatedPerson:
		return &RelatedPerson{}, nil
	case ResourceTypeRequestGroup:
		return &RequestGroup{}, nil
	case ResourceTypeResearchDefinition:
		return &ResearchDefinition{}, nil
	case ResourceTypeResearchElementDefinition:
		return &ResearchElementDefinition{}, nil
	case ResourceTypeResearchStudy:
		return &ResearchStudy{}, nil
	case ResourceTypeResearchSubject:
		return &ResearchSubject{}, nil
	case ResourceTypeRiskAssessment:
		return &RiskAssessment{}, nil
	case ResourceTypeRiskEvidenceSynthesis:
		return &RiskEvidenceSynthesis{}, nil
	case ResourceTypeSchedule:
		return &Schedule{}, nil
	case ResourceTypeSearchParameter:
		return &SearchParameter{}, nil
	case ResourceTypeServiceRequest:
		return &ServiceRequest{}, nil
	case ResourceTypeSlot:
		return &Slot{}, nil
	case ResourceTypeSpecimen:
		return &Specimen{}, nil
	case ResourceTypeSpecimenDefinition:
		return &SpecimenDefinition{}, nil
	case ResourceTypeStructureDefinition:
		return &StructureDefinition{}, nil
	case ResourceTypeStructureMap:
		return &StructureMap{}, nil
	case ResourceTypeSubscription:
		return &Subscription{}, nil
	case ResourceTypeSubstance:
		return &Substance{}, nil
	case ResourceTypeSubstanceNucleicAcid:
		return &SubstanceNucleicAcid{}, nil
	case ResourceTypeSubstancePolymer:
		return &SubstancePolymer{}, nil
	case ResourceTypeSubstanceProtein:
		return &SubstanceProtein{}, nil
	case ResourceTypeSubstanceReferenceInformation:
		return &SubstanceReferenceInformation{}, nil
	case ResourceTypeSubstanceSourceMaterial:
		return &SubstanceSourceMaterial{}, nil
	case ResourceTypeSubstanceSpecification:
		return &SubstanceSpecification{}, nil
	case ResourceTypeSupplyDelivery:
		return &SupplyDelivery{}, nil
	case ResourceTypeSupplyRequest:
		return &SupplyRequest{}, nil
	case ResourceTypeTask:
		return &Task{}, nil
	case ResourceTypeTerminologyCapabilities:
		return &TerminologyCapabilities{}, nil
	case ResourceTypeTestReport:
		return &TestReport{}, nil
	case ResourceTypeTestScript:
		return &TestScript{}, nil
	case ResourceTypeValueSet:
		return &ValueSet{}, nil
	case ResourceTypeVerificationResult:
		return &VerificationResult{}, nil
	case ResourceTypeVisionPrescription:
		return &VisionPrescription{}, nil
	default:
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
}
