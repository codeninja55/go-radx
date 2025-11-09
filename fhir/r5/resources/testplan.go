package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeTestPlan is the FHIR resource type name for TestPlan.
const ResourceTypeTestPlan = "TestPlan"

// TestPlanDependency represents a FHIR BackboneElement for TestPlan.dependency.
type TestPlanDependency struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Description of the dependency criterium
	Description *string `json:"description,omitempty"`
	// Link to predecessor test plans
	Predecessor *Reference `json:"predecessor,omitempty"`
}

// TestPlanTestCaseDependency represents a FHIR BackboneElement for TestPlan.testCase.dependency.
type TestPlanTestCaseDependency struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Description of the criteria
	Description *string `json:"description,omitempty"`
	// Link to predecessor test plans
	Predecessor *Reference `json:"predecessor,omitempty"`
}

// TestPlanTestCaseTestRunScript represents a FHIR BackboneElement for TestPlan.testCase.testRun.script.
type TestPlanTestCaseTestRunScript struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The language for the test cases e.g. 'gherkin', 'testscript'
	Language *CodeableConcept `json:"language,omitempty"`
	// The actual content of the cases - references to TestScripts or externally defined content
	Source *any `json:"source,omitempty"`
}

// TestPlanTestCaseTestRun represents a FHIR BackboneElement for TestPlan.testCase.testRun.
type TestPlanTestCaseTestRun struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The narrative description of the tests
	Narrative *string `json:"narrative,omitempty"`
	// The test cases in a structured language e.g. gherkin, Postman, or FHIR TestScript
	Script *TestPlanTestCaseTestRunScript `json:"script,omitempty"`
}

// TestPlanTestCaseTestData represents a FHIR BackboneElement for TestPlan.testCase.testData.
type TestPlanTestCaseTestData struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The type of test data description, e.g. 'synthea'
	Type Coding `json:"type"`
	// The actual test resources when they exist
	Content *Reference `json:"content,omitempty"`
	// Pointer to a definition of test resources - narrative or structured e.g. synthetic data generation, etc
	Source *any `json:"source,omitempty"`
}

// TestPlanTestCaseAssertion represents a FHIR BackboneElement for TestPlan.testCase.assertion.
type TestPlanTestCaseAssertion struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Assertion type - for example 'informative' or 'required'
	Type []CodeableConcept `json:"type,omitempty"`
	// The focus or object of the assertion
	Object []CodeableReference `json:"object,omitempty"`
	// The actual result assertion
	Result []CodeableReference `json:"result,omitempty"`
}

// TestPlanTestCase represents a FHIR BackboneElement for TestPlan.testCase.
type TestPlanTestCase struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Sequence of test case in the test plan
	Sequence *int `json:"sequence,omitempty"`
	// The scope or artifact covered by the case
	Scope []Reference `json:"scope,omitempty"`
	// Required criteria to execute the test case
	Dependency []TestPlanTestCaseDependency `json:"dependency,omitempty"`
	// The actual test to be executed
	TestRun []TestPlanTestCaseTestRun `json:"testRun,omitempty"`
	// The test data used in the test case
	TestData []TestPlanTestCaseTestData `json:"testData,omitempty"`
	// Test assertions or expectations
	Assertion []TestPlanTestCaseAssertion `json:"assertion,omitempty"`
}

// TestPlan represents a FHIR TestPlan.
type TestPlan struct {
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
	// Canonical identifier for this test plan, represented as a URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Business identifier identifier for the test plan
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the test plan
	Version *string `json:"version,omitempty"`
	// How to compare versions
	VersionAlgorithm *any `json:"versionAlgorithm,omitempty"`
	// Name for this test plan (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this test plan (human friendly)
	Title *string `json:"title,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// For testing purposes, not real usage
	Experimental *bool `json:"experimental,omitempty"`
	// Date last changed
	Date *primitives.DateTime `json:"date,omitempty"`
	// Name of the publisher/steward (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Natural language description of the test plan
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction where the test plan applies (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this test plan is defined
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// Copyright holder and year(s)
	CopyrightLabel *string `json:"copyrightLabel,omitempty"`
	// The category of the Test Plan - can be acceptance, unit, performance
	Category []CodeableConcept `json:"category,omitempty"`
	// What is being tested with this Test Plan - a conformance resource, or narrative criteria, or an external reference
	Scope []Reference `json:"scope,omitempty"`
	// A description of test tools to be used in the test plan - narrative for now
	TestTools *string `json:"testTools,omitempty"`
	// The required criteria to execute the test plan - e.g. preconditions, previous tests
	Dependency []TestPlanDependency `json:"dependency,omitempty"`
	// The threshold or criteria for the test plan to be considered successfully executed - narrative
	ExitCriteria *string `json:"exitCriteria,omitempty"`
	// The test cases that constitute this plan
	TestCase []TestPlanTestCase `json:"testCase,omitempty"`
}
