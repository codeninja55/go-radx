package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeTestScript is the FHIR resource type name for TestScript.
const ResourceTypeTestScript = "TestScript"

// TestScriptOrigin represents a FHIR BackboneElement for TestScript.origin.
type TestScriptOrigin struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The index of the abstract origin server starting at 1
	Index int `json:"index"`
	// FHIR-Client | FHIR-SDC-FormFiller
	Profile Coding `json:"profile"`
	// The url path of the origin server
	URL *string `json:"url,omitempty"`
}

// TestScriptDestination represents a FHIR BackboneElement for TestScript.destination.
type TestScriptDestination struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The index of the abstract destination server starting at 1
	Index int `json:"index"`
	// FHIR-Server | FHIR-SDC-FormManager | FHIR-SDC-FormReceiver | FHIR-SDC-FormProcessor
	Profile Coding `json:"profile"`
	// The url path of the destination server
	URL *string `json:"url,omitempty"`
}

// TestScriptMetadataLink represents a FHIR BackboneElement for TestScript.metadata.link.
type TestScriptMetadataLink struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// URL to the specification
	URL string `json:"url"`
	// Short description
	Description *string `json:"description,omitempty"`
}

// TestScriptMetadataCapability represents a FHIR BackboneElement for TestScript.metadata.capability.
type TestScriptMetadataCapability struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Are the capabilities required?
	Required bool `json:"required"`
	// Are the capabilities validated?
	Validated bool `json:"validated"`
	// The expected capabilities of the server
	Description *string `json:"description,omitempty"`
	// Which origin server these requirements apply to
	Origin []int `json:"origin,omitempty"`
	// Which server these requirements apply to
	Destination *int `json:"destination,omitempty"`
	// Links to the FHIR specification
	Link []string `json:"link,omitempty"`
	// Required Capability Statement
	Capabilities string `json:"capabilities"`
}

// TestScriptMetadata represents a FHIR BackboneElement for TestScript.metadata.
type TestScriptMetadata struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Links to the FHIR specification
	Link []TestScriptMetadataLink `json:"link,omitempty"`
	// Capabilities  that are assumed to function correctly on the FHIR server being tested
	Capability []TestScriptMetadataCapability `json:"capability,omitempty"`
}

// TestScriptScope represents a FHIR BackboneElement for TestScript.scope.
type TestScriptScope struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The specific conformance artifact being tested
	Artifact string `json:"artifact"`
	// required | optional | strict
	Conformance *CodeableConcept `json:"conformance,omitempty"`
	// unit | integration | production
	Phase *CodeableConcept `json:"phase,omitempty"`
}

// TestScriptFixture represents a FHIR BackboneElement for TestScript.fixture.
type TestScriptFixture struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Whether or not to implicitly create the fixture during setup
	Autocreate bool `json:"autocreate"`
	// Whether or not to implicitly delete the fixture during teardown
	Autodelete bool `json:"autodelete"`
	// Reference of the resource
	Resource *Reference `json:"resource,omitempty"`
}

// TestScriptVariable represents a FHIR BackboneElement for TestScript.variable.
type TestScriptVariable struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Descriptive name for this variable
	Name string `json:"name"`
	// Default, hard-coded, or user-defined value for this variable
	DefaultValue *string `json:"defaultValue,omitempty"`
	// Natural language description of the variable
	Description *string `json:"description,omitempty"`
	// The FHIRPath expression against the fixture body
	Expression *string `json:"expression,omitempty"`
	// HTTP header field name for source
	HeaderField *string `json:"headerField,omitempty"`
	// Hint help text for default value to enter
	Hint *string `json:"hint,omitempty"`
	// XPath or JSONPath against the fixture body
	Path *string `json:"path,omitempty"`
	// Fixture Id of source expression or headerField within this variable
	SourceId *string `json:"sourceId,omitempty"`
}

// TestScriptSetupActionOperationRequestHeader represents a FHIR BackboneElement for TestScript.setup.action.operation.requestHeader.
type TestScriptSetupActionOperationRequestHeader struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// HTTP header field name
	Field string `json:"field"`
	// HTTP headerfield value
	Value string `json:"value"`
}

// TestScriptSetupActionOperation represents a FHIR BackboneElement for TestScript.setup.action.operation.
type TestScriptSetupActionOperation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The operation code type that will be executed
	Type *Coding `json:"type,omitempty"`
	// Resource type
	Resource *string `json:"resource,omitempty"`
	// Tracking/logging operation label
	Label *string `json:"label,omitempty"`
	// Tracking/reporting operation description
	Description *string `json:"description,omitempty"`
	// Mime type to accept in the payload of the response, with charset etc
	Accept *string `json:"accept,omitempty"`
	// Mime type of the request payload contents, with charset etc
	ContentType *string `json:"contentType,omitempty"`
	// Server responding to the request
	Destination *int `json:"destination,omitempty"`
	// Whether or not to send the request url in encoded format
	EncodeRequestUrl bool `json:"encodeRequestUrl"`
	// delete | get | options | patch | post | put | head
	Method *string `json:"method,omitempty"`
	// Server initiating the request
	Origin *int `json:"origin,omitempty"`
	// Explicitly defined path parameters
	Params *string `json:"params,omitempty"`
	// Each operation can have one or more header elements
	RequestHeader []TestScriptSetupActionOperationRequestHeader `json:"requestHeader,omitempty"`
	// Fixture Id of mapped request
	RequestId *string `json:"requestId,omitempty"`
	// Fixture Id of mapped response
	ResponseId *string `json:"responseId,omitempty"`
	// Fixture Id of body for PUT and POST requests
	SourceId *string `json:"sourceId,omitempty"`
	// Id of fixture used for extracting the [id],  [type], and [vid] for GET requests
	TargetId *string `json:"targetId,omitempty"`
	// Request URL
	URL *string `json:"url,omitempty"`
}

// TestScriptSetupActionAssertRequirement represents a FHIR BackboneElement for TestScript.setup.action.assert.requirement.
type TestScriptSetupActionAssertRequirement struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Link or reference to the testing requirement
	Link *any `json:"link,omitempty"`
}

// TestScriptSetupActionAssert represents a FHIR BackboneElement for TestScript.setup.action.assert.
type TestScriptSetupActionAssert struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Tracking/logging assertion label
	Label *string `json:"label,omitempty"`
	// Tracking/reporting assertion description
	Description *string `json:"description,omitempty"`
	// response | request
	Direction *string `json:"direction,omitempty"`
	// Id of the source fixture to be evaluated
	CompareToSourceId *string `json:"compareToSourceId,omitempty"`
	// The FHIRPath expression to evaluate against the source fixture
	CompareToSourceExpression *string `json:"compareToSourceExpression,omitempty"`
	// XPath or JSONPath expression to evaluate against the source fixture
	CompareToSourcePath *string `json:"compareToSourcePath,omitempty"`
	// Mime type to compare against the 'Content-Type' header
	ContentType *string `json:"contentType,omitempty"`
	// fail | pass | skip | stop
	DefaultManualCompletion *string `json:"defaultManualCompletion,omitempty"`
	// The FHIRPath expression to be evaluated
	Expression *string `json:"expression,omitempty"`
	// HTTP header field name
	HeaderField *string `json:"headerField,omitempty"`
	// Fixture Id of minimum content resource
	MinimumId *string `json:"minimumId,omitempty"`
	// Perform validation on navigation links?
	NavigationLinks *bool `json:"navigationLinks,omitempty"`
	// equals | notEquals | in | notIn | greaterThan | lessThan | empty | notEmpty | contains | notContains | eval | manualEval
	Operator *string `json:"operator,omitempty"`
	// XPath or JSONPath expression
	Path *string `json:"path,omitempty"`
	// delete | get | options | patch | post | put | head
	RequestMethod *string `json:"requestMethod,omitempty"`
	// Request URL comparison value
	RequestURL *string `json:"requestURL,omitempty"`
	// Resource type
	Resource *string `json:"resource,omitempty"`
	// continue | switchingProtocols | okay | created | accepted | nonAuthoritativeInformation | noContent | resetContent | partialContent | multipleChoices | movedPermanently | found | seeOther | notModified | useProxy | temporaryRedirect | permanentRedirect | badRequest | unauthorized | paymentRequired | forbidden | notFound | methodNotAllowed | notAcceptable | proxyAuthenticationRequired | requestTimeout | conflict | gone | lengthRequired | preconditionFailed | contentTooLarge | uriTooLong | unsupportedMediaType | rangeNotSatisfiable | expectationFailed | misdirectedRequest | unprocessableContent | upgradeRequired | internalServerError | notImplemented | badGateway | serviceUnavailable | gatewayTimeout | httpVersionNotSupported
	Response *string `json:"response,omitempty"`
	// HTTP response code to test
	ResponseCode *string `json:"responseCode,omitempty"`
	// Fixture Id of source expression or headerField
	SourceId *string `json:"sourceId,omitempty"`
	// If this assert fails, will the current test execution stop?
	StopTestOnFail bool `json:"stopTestOnFail"`
	// Profile Id of validation profile reference
	ValidateProfileId *string `json:"validateProfileId,omitempty"`
	// The value to compare to
	Value *string `json:"value,omitempty"`
	// Will this assert produce a warning only on error?
	WarningOnly bool `json:"warningOnly"`
	// Links or references to the testing requirements
	Requirement []TestScriptSetupActionAssertRequirement `json:"requirement,omitempty"`
}

// TestScriptSetupAction represents a FHIR BackboneElement for TestScript.setup.action.
type TestScriptSetupAction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The setup operation to perform
	Operation *TestScriptSetupActionOperation `json:"operation,omitempty"`
	// The assertion to perform
	Assert *TestScriptSetupActionAssert `json:"assert,omitempty"`
}

// TestScriptSetup represents a FHIR BackboneElement for TestScript.setup.
type TestScriptSetup struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// A setup operation or assert to perform
	Action []TestScriptSetupAction `json:"action,omitempty"`
}

// TestScriptTestActionOperation represents a FHIR BackboneElement for TestScript.test.action.operation.
type TestScriptTestActionOperation struct {
}

// TestScriptTestActionAssert represents a FHIR BackboneElement for TestScript.test.action.assert.
type TestScriptTestActionAssert struct {
}

// TestScriptTestAction represents a FHIR BackboneElement for TestScript.test.action.
type TestScriptTestAction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The setup operation to perform
	Operation *TestScriptTestActionOperation `json:"operation,omitempty"`
	// The setup assertion to perform
	Assert *TestScriptTestActionAssert `json:"assert,omitempty"`
}

// TestScriptTest represents a FHIR BackboneElement for TestScript.test.
type TestScriptTest struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Tracking/logging name of this test
	Name *string `json:"name,omitempty"`
	// Tracking/reporting short description of the test
	Description *string `json:"description,omitempty"`
	// A test operation or assert to perform
	Action []TestScriptTestAction `json:"action,omitempty"`
}

// TestScriptTeardownActionOperation represents a FHIR BackboneElement for TestScript.teardown.action.operation.
type TestScriptTeardownActionOperation struct {
}

// TestScriptTeardownAction represents a FHIR BackboneElement for TestScript.teardown.action.
type TestScriptTeardownAction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The teardown operation to perform
	Operation TestScriptTeardownActionOperation `json:"operation"`
}

// TestScriptTeardown represents a FHIR BackboneElement for TestScript.teardown.
type TestScriptTeardown struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// One or more teardown operations to perform
	Action []TestScriptTeardownAction `json:"action,omitempty"`
}

// TestScript represents a FHIR TestScript.
type TestScript struct {
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
	// Canonical identifier for this test script, represented as a URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Additional identifier for the test script
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the test script
	Version *string `json:"version,omitempty"`
	// How to compare versions
	VersionAlgorithm *any `json:"versionAlgorithm,omitempty"`
	// Name for this test script (computer friendly)
	Name string `json:"name"`
	// Name for this test script (human friendly)
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
	// Natural language description of the test script
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for test script (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this test script is defined
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// Copyright holder and year(s)
	CopyrightLabel *string `json:"copyrightLabel,omitempty"`
	// An abstract server representing a client or sender in a message exchange
	Origin []TestScriptOrigin `json:"origin,omitempty"`
	// An abstract server representing a destination or receiver in a message exchange
	Destination []TestScriptDestination `json:"destination,omitempty"`
	// Required capability that is assumed to function correctly on the FHIR server being tested
	Metadata *TestScriptMetadata `json:"metadata,omitempty"`
	// Indication of the artifact(s) that are tested by this test case
	Scope []TestScriptScope `json:"scope,omitempty"`
	// Fixture in the test script - by reference (uri)
	Fixture []TestScriptFixture `json:"fixture,omitempty"`
	// Reference of the validation profile
	Profile []string `json:"profile,omitempty"`
	// Placeholder for evaluated elements
	Variable []TestScriptVariable `json:"variable,omitempty"`
	// A series of required setup operations before tests are executed
	Setup *TestScriptSetup `json:"setup,omitempty"`
	// A test in this script
	Test []TestScriptTest `json:"test,omitempty"`
	// A series of required clean up steps
	Teardown *TestScriptTeardown `json:"teardown,omitempty"`
}
