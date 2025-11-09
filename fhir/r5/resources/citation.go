package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeCitation is the FHIR resource type name for Citation.
const ResourceTypeCitation = "Citation"

// CitationSummary represents a FHIR BackboneElement for Citation.summary.
type CitationSummary struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Format for display of the citation summary
	Style *CodeableConcept `json:"style,omitempty"`
	// The human-readable display of the citation summary
	Text string `json:"text"`
}

// CitationClassification represents a FHIR BackboneElement for Citation.classification.
type CitationClassification struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The kind of classifier (e.g. publication type, keyword)
	Type *CodeableConcept `json:"type,omitempty"`
	// The specific classification value
	Classifier []CodeableConcept `json:"classifier,omitempty"`
}

// CitationStatusDate represents a FHIR BackboneElement for Citation.statusDate.
type CitationStatusDate struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Classification of the status
	Activity CodeableConcept `json:"activity"`
	// Either occurred or expected
	Actual *bool `json:"actual,omitempty"`
	// When the status started and/or ended
	Period Period `json:"period"`
}

// CitationCitedArtifactVersion represents a FHIR BackboneElement for Citation.citedArtifact.version.
type CitationCitedArtifactVersion struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The version number or other version identifier
	Value string `json:"value"`
	// Citation for the main version of the cited artifact
	BaseCitation *Reference `json:"baseCitation,omitempty"`
}

// CitationCitedArtifactStatusDate represents a FHIR BackboneElement for Citation.citedArtifact.statusDate.
type CitationCitedArtifactStatusDate struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Classification of the status
	Activity CodeableConcept `json:"activity"`
	// Either occurred or expected
	Actual *bool `json:"actual,omitempty"`
	// When the status started and/or ended
	Period Period `json:"period"`
}

// CitationCitedArtifactTitle represents a FHIR BackboneElement for Citation.citedArtifact.title.
type CitationCitedArtifactTitle struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The kind of title
	Type []CodeableConcept `json:"type,omitempty"`
	// Used to express the specific language
	Language *CodeableConcept `json:"language,omitempty"`
	// The title of the article or artifact
	Text string `json:"text"`
}

// CitationCitedArtifactAbstract represents a FHIR BackboneElement for Citation.citedArtifact.abstract.
type CitationCitedArtifactAbstract struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The kind of abstract
	Type *CodeableConcept `json:"type,omitempty"`
	// Used to express the specific language
	Language *CodeableConcept `json:"language,omitempty"`
	// Abstract content
	Text string `json:"text"`
	// Copyright notice for the abstract
	Copyright *string `json:"copyright,omitempty"`
}

// CitationCitedArtifactPart represents a FHIR BackboneElement for Citation.citedArtifact.part.
type CitationCitedArtifactPart struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The kind of component
	Type *CodeableConcept `json:"type,omitempty"`
	// The specification of the component
	Value *string `json:"value,omitempty"`
	// The citation for the full article or artifact
	BaseCitation *Reference `json:"baseCitation,omitempty"`
}

// CitationCitedArtifactRelatesTo represents a FHIR BackboneElement for Citation.citedArtifact.relatesTo.
type CitationCitedArtifactRelatesTo struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// documentation | justification | citation | predecessor | successor | derived-from | depends-on | composed-of | part-of | amends | amended-with | appends | appended-with | cites | cited-by | comments-on | comment-in | contains | contained-in | corrects | correction-in | replaces | replaced-with | retracts | retracted-by | signs | similar-to | supports | supported-with | transforms | transformed-into | transformed-with | documents | specification-of | created-with | cite-as | reprint | reprint-of
	Type string `json:"type"`
	// Additional classifiers
	Classifier []CodeableConcept `json:"classifier,omitempty"`
	// Short label
	Label *string `json:"label,omitempty"`
	// Brief description of the related artifact
	Display *string `json:"display,omitempty"`
	// Bibliographic citation for the artifact
	Citation *string `json:"citation,omitempty"`
	// What document is being referenced
	Document *Attachment `json:"document,omitempty"`
	// What artifact is being referenced
	Resource *string `json:"resource,omitempty"`
	// What artifact, if not a conformance resource
	ResourceReference *Reference `json:"resourceReference,omitempty"`
}

// CitationCitedArtifactPublicationFormPublishedIn represents a FHIR BackboneElement for Citation.citedArtifact.publicationForm.publishedIn.
type CitationCitedArtifactPublicationFormPublishedIn struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Kind of container (e.g. Periodical, database, or book)
	Type *CodeableConcept `json:"type,omitempty"`
	// Journal identifiers include ISSN, ISO Abbreviation and NLMuniqueID; Book identifiers include ISBN
	Identifier []Identifier `json:"identifier,omitempty"`
	// Name of the database or title of the book or journal
	Title *string `json:"title,omitempty"`
	// Name of or resource describing the publisher
	Publisher *Reference `json:"publisher,omitempty"`
	// Geographic location of the publisher
	PublisherLocation *string `json:"publisherLocation,omitempty"`
}

// CitationCitedArtifactPublicationForm represents a FHIR BackboneElement for Citation.citedArtifact.publicationForm.
type CitationCitedArtifactPublicationForm struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The collection the cited article or artifact is published in
	PublishedIn *CitationCitedArtifactPublicationFormPublishedIn `json:"publishedIn,omitempty"`
	// Internet or Print
	CitedMedium *CodeableConcept `json:"citedMedium,omitempty"`
	// Volume number of journal or other collection in which the article is published
	Volume *string `json:"volume,omitempty"`
	// Issue, part or supplement of journal or other collection in which the article is published
	Issue *string `json:"issue,omitempty"`
	// The date the article was added to the database, or the date the article was released
	ArticleDate *primitives.DateTime `json:"articleDate,omitempty"`
	// Text representation of the date on which the issue of the cited artifact was published
	PublicationDateText *string `json:"publicationDateText,omitempty"`
	// Season in which the cited artifact was published
	PublicationDateSeason *string `json:"publicationDateSeason,omitempty"`
	// The date the article was last revised or updated in the database
	LastRevisionDate *primitives.DateTime `json:"lastRevisionDate,omitempty"`
	// Language(s) in which this form of the article is published
	Language []CodeableConcept `json:"language,omitempty"`
	// Entry number or identifier for inclusion in a database
	AccessionNumber *string `json:"accessionNumber,omitempty"`
	// Used for full display of pagination
	PageString *string `json:"pageString,omitempty"`
	// Used for isolated representation of first page
	FirstPage *string `json:"firstPage,omitempty"`
	// Used for isolated representation of last page
	LastPage *string `json:"lastPage,omitempty"`
	// Number of pages or screens
	PageCount *string `json:"pageCount,omitempty"`
	// Copyright notice for the full article or artifact
	Copyright *string `json:"copyright,omitempty"`
}

// CitationCitedArtifactWebLocation represents a FHIR BackboneElement for Citation.citedArtifact.webLocation.
type CitationCitedArtifactWebLocation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Code the reason for different URLs, e.g. abstract and full-text
	Classifier []CodeableConcept `json:"classifier,omitempty"`
	// The specific URL
	URL *string `json:"url,omitempty"`
}

// CitationCitedArtifactClassification represents a FHIR BackboneElement for Citation.citedArtifact.classification.
type CitationCitedArtifactClassification struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The kind of classifier (e.g. publication type, keyword)
	Type *CodeableConcept `json:"type,omitempty"`
	// The specific classification value
	Classifier []CodeableConcept `json:"classifier,omitempty"`
	// Complex or externally created classification
	ArtifactAssessment []Reference `json:"artifactAssessment,omitempty"`
}

// CitationCitedArtifactContributorshipEntryContributionInstance represents a FHIR BackboneElement for Citation.citedArtifact.contributorship.entry.contributionInstance.
type CitationCitedArtifactContributorshipEntryContributionInstance struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The specific contribution
	Type CodeableConcept `json:"type"`
	// The time that the contribution was made
	Time *primitives.DateTime `json:"time,omitempty"`
}

// CitationCitedArtifactContributorshipEntry represents a FHIR BackboneElement for Citation.citedArtifact.contributorship.entry.
type CitationCitedArtifactContributorshipEntry struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The identity of the individual contributor
	Contributor Reference `json:"contributor"`
	// For citation styles that use initials
	ForenameInitials *string `json:"forenameInitials,omitempty"`
	// Organizational affiliation
	Affiliation []Reference `json:"affiliation,omitempty"`
	// The specific contribution
	ContributionType []CodeableConcept `json:"contributionType,omitempty"`
	// The role of the contributor (e.g. author, editor, reviewer, funder)
	Role *CodeableConcept `json:"role,omitempty"`
	// Contributions with accounting for time or number
	ContributionInstance []CitationCitedArtifactContributorshipEntryContributionInstance `json:"contributionInstance,omitempty"`
	// Whether the contributor is the corresponding contributor for the role
	CorrespondingContact *bool `json:"correspondingContact,omitempty"`
	// Ranked order of contribution
	RankingOrder *int `json:"rankingOrder,omitempty"`
}

// CitationCitedArtifactContributorshipSummary represents a FHIR BackboneElement for Citation.citedArtifact.contributorship.summary.
type CitationCitedArtifactContributorshipSummary struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Such as author list, contributorship statement, funding statement, acknowledgements statement, or conflicts of interest statement
	Type *CodeableConcept `json:"type,omitempty"`
	// The format for the display string
	Style *CodeableConcept `json:"style,omitempty"`
	// Used to code the producer or rule for creating the display string
	Source *CodeableConcept `json:"source,omitempty"`
	// The display string for the author list, contributor list, or contributorship statement
	Value string `json:"value"`
}

// CitationCitedArtifactContributorship represents a FHIR BackboneElement for Citation.citedArtifact.contributorship.
type CitationCitedArtifactContributorship struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Indicates if the list includes all authors and/or contributors
	Complete *bool `json:"complete,omitempty"`
	// An individual entity named as a contributor
	Entry []CitationCitedArtifactContributorshipEntry `json:"entry,omitempty"`
	// Used to record a display of the author/contributor list without separate data element for each list member
	Summary []CitationCitedArtifactContributorshipSummary `json:"summary,omitempty"`
}

// CitationCitedArtifact represents a FHIR BackboneElement for Citation.citedArtifact.
type CitationCitedArtifact struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Unique identifier. May include DOI, PMID, PMCID, etc
	Identifier []Identifier `json:"identifier,omitempty"`
	// Identifier not unique to the cited artifact. May include trial registry identifiers
	RelatedIdentifier []Identifier `json:"relatedIdentifier,omitempty"`
	// When the cited artifact was accessed
	DateAccessed *primitives.DateTime `json:"dateAccessed,omitempty"`
	// The defined version of the cited artifact
	Version *CitationCitedArtifactVersion `json:"version,omitempty"`
	// The status of the cited artifact
	CurrentState []CodeableConcept `json:"currentState,omitempty"`
	// An effective date or period for a status of the cited artifact
	StatusDate []CitationCitedArtifactStatusDate `json:"statusDate,omitempty"`
	// The title details of the article or artifact
	Title []CitationCitedArtifactTitle `json:"title,omitempty"`
	// Summary of the article or artifact
	Abstract []CitationCitedArtifactAbstract `json:"abstract,omitempty"`
	// The component of the article or artifact
	Part *CitationCitedArtifactPart `json:"part,omitempty"`
	// The artifact related to the cited artifact
	RelatesTo []CitationCitedArtifactRelatesTo `json:"relatesTo,omitempty"`
	// If multiple, used to represent alternative forms of the article that are not separate citations
	PublicationForm []CitationCitedArtifactPublicationForm `json:"publicationForm,omitempty"`
	// Used for any URL for the article or artifact cited
	WebLocation []CitationCitedArtifactWebLocation `json:"webLocation,omitempty"`
	// The assignment to an organizing scheme
	Classification []CitationCitedArtifactClassification `json:"classification,omitempty"`
	// Attribution of authors and other contributors
	Contributorship *CitationCitedArtifactContributorship `json:"contributorship,omitempty"`
	// Any additional information or content for the article or artifact
	Note []Annotation `json:"note,omitempty"`
}

// Citation represents a FHIR Citation.
type Citation struct {
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
	// Canonical identifier for this citation record, represented as a globally unique URI
	URL *string `json:"url,omitempty"`
	// Identifier for the citation record itself
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the citation record
	Version *string `json:"version,omitempty"`
	// How to compare versions
	VersionAlgorithm *any `json:"versionAlgorithm,omitempty"`
	// Name for this citation record (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this citation record (human friendly)
	Title *string `json:"title,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// For testing purposes, not real usage
	Experimental *bool `json:"experimental,omitempty"`
	// Date last changed
	Date *primitives.DateTime `json:"date,omitempty"`
	// The publisher of the citation record, not the publisher of the article or artifact being cited
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher of the citation record
	Contact []ContactDetail `json:"contact,omitempty"`
	// Natural language description of the citation
	Description *string `json:"description,omitempty"`
	// The context that the citation record content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for citation record (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this citation is defined
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions for the citation record, not for the cited artifact
	Copyright *string `json:"copyright,omitempty"`
	// Copyright holder and year(s) for the ciation record, not for the cited artifact
	CopyrightLabel *string `json:"copyrightLabel,omitempty"`
	// When the citation record was approved by publisher
	ApprovalDate *primitives.Date `json:"approvalDate,omitempty"`
	// When the citation record was last reviewed by the publisher
	LastReviewDate *primitives.Date `json:"lastReviewDate,omitempty"`
	// When the citation record is expected to be used
	EffectivePeriod *Period `json:"effectivePeriod,omitempty"`
	// Who authored the citation record
	Author []ContactDetail `json:"author,omitempty"`
	// Who edited the citation record
	Editor []ContactDetail `json:"editor,omitempty"`
	// Who reviewed the citation record
	Reviewer []ContactDetail `json:"reviewer,omitempty"`
	// Who endorsed the citation record
	Endorser []ContactDetail `json:"endorser,omitempty"`
	// A human-readable display of key concepts to represent the citation
	Summary []CitationSummary `json:"summary,omitempty"`
	// The assignment to an organizing scheme
	Classification []CitationClassification `json:"classification,omitempty"`
	// Used for general notes and annotations not coded elsewhere
	Note []Annotation `json:"note,omitempty"`
	// The status of the citation record
	CurrentState []CodeableConcept `json:"currentState,omitempty"`
	// An effective date or period for a status of the citation record
	StatusDate []CitationStatusDate `json:"statusDate,omitempty"`
	// Artifact related to the citation record
	RelatedArtifact []RelatedArtifact `json:"relatedArtifact,omitempty"`
	// The article or artifact being described
	CitedArtifact *CitationCitedArtifact `json:"citedArtifact,omitempty"`
}
