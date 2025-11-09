package resources

// ResourceTypeSubstanceSourceMaterial is the FHIR resource type name for SubstanceSourceMaterial.
const ResourceTypeSubstanceSourceMaterial = "SubstanceSourceMaterial"

// SubstanceSourceMaterialFractionDescription represents a FHIR BackboneElement for SubstanceSourceMaterial.fractionDescription.
type SubstanceSourceMaterialFractionDescription struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// This element is capturing information about the fraction of a plant part, or human plasma for fractionation
	Fraction *string `json:"fraction,omitempty"`
	// The specific type of the material constituting the component. For Herbal preparations the particulars of the extracts (liquid/dry) is described in Specified Substance Group 1
	MaterialType *CodeableConcept `json:"materialType,omitempty"`
}

// SubstanceSourceMaterialOrganismAuthor represents a FHIR BackboneElement for SubstanceSourceMaterial.organism.author.
type SubstanceSourceMaterialOrganismAuthor struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The type of author of an organism species shall be specified. The parenthetical author of an organism species refers to the first author who published the plant/animal name (of any rank). The primary author of an organism species refers to the first author(s), who validly published the plant/animal name
	AuthorType *CodeableConcept `json:"authorType,omitempty"`
	// The author of an organism species shall be specified. The author year of an organism shall also be specified when applicable; refers to the year in which the first author(s) published the infraspecific plant/animal name (of any rank)
	AuthorDescription *string `json:"authorDescription,omitempty"`
}

// SubstanceSourceMaterialOrganismHybrid represents a FHIR BackboneElement for SubstanceSourceMaterial.organism.hybrid.
type SubstanceSourceMaterialOrganismHybrid struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The identifier of the maternal species constituting the hybrid organism shall be specified based on a controlled vocabulary. For plants, the parents aren’t always known, and it is unlikely that it will be known which is maternal and which is paternal
	MaternalOrganismId *string `json:"maternalOrganismId,omitempty"`
	// The name of the maternal species constituting the hybrid organism shall be specified. For plants, the parents aren’t always known, and it is unlikely that it will be known which is maternal and which is paternal
	MaternalOrganismName *string `json:"maternalOrganismName,omitempty"`
	// The identifier of the paternal species constituting the hybrid organism shall be specified based on a controlled vocabulary
	PaternalOrganismId *string `json:"paternalOrganismId,omitempty"`
	// The name of the paternal species constituting the hybrid organism shall be specified
	PaternalOrganismName *string `json:"paternalOrganismName,omitempty"`
	// The hybrid type of an organism shall be specified
	HybridType *CodeableConcept `json:"hybridType,omitempty"`
}

// SubstanceSourceMaterialOrganismOrganismGeneral represents a FHIR BackboneElement for SubstanceSourceMaterial.organism.organismGeneral.
type SubstanceSourceMaterialOrganismOrganismGeneral struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The kingdom of an organism shall be specified
	Kingdom *CodeableConcept `json:"kingdom,omitempty"`
	// The phylum of an organism shall be specified
	Phylum *CodeableConcept `json:"phylum,omitempty"`
	// The class of an organism shall be specified
	Class *CodeableConcept `json:"class,omitempty"`
	// The order of an organism shall be specified,
	Order *CodeableConcept `json:"order,omitempty"`
}

// SubstanceSourceMaterialOrganism represents a FHIR BackboneElement for SubstanceSourceMaterial.organism.
type SubstanceSourceMaterialOrganism struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The family of an organism shall be specified
	Family *CodeableConcept `json:"family,omitempty"`
	// The genus of an organism shall be specified; refers to the Latin epithet of the genus element of the plant/animal scientific name; it is present in names for genera, species and infraspecies
	Genus *CodeableConcept `json:"genus,omitempty"`
	// The species of an organism shall be specified; refers to the Latin epithet of the species of the plant/animal; it is present in names for species and infraspecies
	Species *CodeableConcept `json:"species,omitempty"`
	// The Intraspecific type of an organism shall be specified
	IntraspecificType *CodeableConcept `json:"intraspecificType,omitempty"`
	// The intraspecific description of an organism shall be specified based on a controlled vocabulary. For Influenza Vaccine, the intraspecific description shall contain the syntax of the antigen in line with the WHO convention
	IntraspecificDescription *string `json:"intraspecificDescription,omitempty"`
	// 4.9.13.6.1 Author type (Conditional)
	Author []SubstanceSourceMaterialOrganismAuthor `json:"author,omitempty"`
	// 4.9.13.8.1 Hybrid species maternal organism ID (Optional)
	Hybrid *SubstanceSourceMaterialOrganismHybrid `json:"hybrid,omitempty"`
	// 4.9.13.7.1 Kingdom (Conditional)
	OrganismGeneral *SubstanceSourceMaterialOrganismOrganismGeneral `json:"organismGeneral,omitempty"`
}

// SubstanceSourceMaterialPartDescription represents a FHIR BackboneElement for SubstanceSourceMaterial.partDescription.
type SubstanceSourceMaterialPartDescription struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Entity of anatomical origin of source material within an organism
	Part *CodeableConcept `json:"part,omitempty"`
	// The detailed anatomic location when the part can be extracted from different anatomical locations of the organism. Multiple alternative locations may apply
	PartLocation *CodeableConcept `json:"partLocation,omitempty"`
}

// SubstanceSourceMaterial represents a FHIR SubstanceSourceMaterial.
type SubstanceSourceMaterial struct {
	// Logical id of this artifact
	ID *string `json:"id,omitempty"`
	// Metadata about the resource
	Meta *Meta `json:"meta,omitempty"`
	// A set of rules under which this content was created
	ImplicitRules *string `json:"implicitRules,omitempty"`
	// Language of the resource content
	Language *string `json:"language,omitempty"`
	// Text summary of the resource, for human interpretation
	Text *Narrative `json:"text,omitempty"`
	// Contained, inline Resources
	Contained []any `json:"contained,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// General high level classification of the source material specific to the origin of the material
	SourceMaterialClass *CodeableConcept `json:"sourceMaterialClass,omitempty"`
	// The type of the source material shall be specified based on a controlled vocabulary. For vaccines, this subclause refers to the class of infectious agent
	SourceMaterialType *CodeableConcept `json:"sourceMaterialType,omitempty"`
	// The state of the source material when extracted
	SourceMaterialState *CodeableConcept `json:"sourceMaterialState,omitempty"`
	// The unique identifier associated with the source material parent organism shall be specified
	OrganismId *Identifier `json:"organismId,omitempty"`
	// The organism accepted Scientific name shall be provided based on the organism taxonomy
	OrganismName *string `json:"organismName,omitempty"`
	// The parent of the herbal drug Ginkgo biloba, Leaf is the substance ID of the substance (fresh) of Ginkgo biloba L. or Ginkgo biloba L. (Whole plant)
	ParentSubstanceId []Identifier `json:"parentSubstanceId,omitempty"`
	// The parent substance of the Herbal Drug, or Herbal preparation
	ParentSubstanceName []string `json:"parentSubstanceName,omitempty"`
	// The country where the plant material is harvested or the countries where the plasma is sourced from as laid down in accordance with the Plasma Master File. For “Plasma-derived substances” the attribute country of origin provides information about the countries used for the manufacturing of the Cryopoor plama or Crioprecipitate
	CountryOfOrigin []CodeableConcept `json:"countryOfOrigin,omitempty"`
	// The place/region where the plant is harvested or the places/regions where the animal source material has its habitat
	GeographicalLocation []string `json:"geographicalLocation,omitempty"`
	// Stage of life for animals, plants, insects and microorganisms. This information shall be provided only when the substance is significantly different in these stages (e.g. foetal bovine serum)
	DevelopmentStage *CodeableConcept `json:"developmentStage,omitempty"`
	// Many complex materials are fractions of parts of plants, animals, or minerals. Fraction elements are often necessary to define both Substances and Specified Group 1 Substances. For substances derived from Plants, fraction information will be captured at the Substance information level ( . Oils, Juices and Exudates). Additional information for Extracts, such as extraction solvent composition, will be captured at the Specified Substance Group 1 information level. For plasma-derived products fraction information will be captured at the Substance and the Specified Substance Group 1 levels
	FractionDescription []SubstanceSourceMaterialFractionDescription `json:"fractionDescription,omitempty"`
	// This subclause describes the organism which the substance is derived from. For vaccines, the parent organism shall be specified based on these subclause elements. As an example, full taxonomy will be described for the Substance Name: ., Leaf
	Organism *SubstanceSourceMaterialOrganism `json:"organism,omitempty"`
	// To do
	PartDescription []SubstanceSourceMaterialPartDescription `json:"partDescription,omitempty"`
}
