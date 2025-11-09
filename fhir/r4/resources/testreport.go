package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeTestReport is the FHIR resource type name for TestReport.
const ResourceTypeTestReport = "TestReport"

// TestReportParticipant represents a FHIR BackboneElement for TestReport.participant.
type TestReportParticipant struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// test-engine | client | server
	Type string `json:"type"`
	// The uri of the participant. An absolute URL is preferred
	URI string `json:"uri"`
	// The display name of the participant
	Display *string `json:"display,omitempty"`
}

// TestReportSetupActionOperation represents a FHIR BackboneElement for TestReport.setup.action.operation.
type TestReportSetupActionOperation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// pass | skip | fail | warning | error
	Result string `json:"result"`
	// A message associated with the result
	Message *string `json:"message,omitempty"`
	// A link to further details on the result
	Detail *string `json:"detail,omitempty"`
}

// TestReportSetupActionAssert represents a FHIR BackboneElement for TestReport.setup.action.assert.
type TestReportSetupActionAssert struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// pass | skip | fail | warning | error
	Result string `json:"result"`
	// A message associated with the result
	Message *string `json:"message,omitempty"`
	// A link to further details on the result
	Detail *string `json:"detail,omitempty"`
}

// TestReportSetupAction represents a FHIR BackboneElement for TestReport.setup.action.
type TestReportSetupAction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The operation to perform
	Operation *TestReportSetupActionOperation `json:"operation,omitempty"`
	// The assertion to perform
	Assert *TestReportSetupActionAssert `json:"assert,omitempty"`
}

// TestReportSetup represents a FHIR BackboneElement for TestReport.setup.
type TestReportSetup struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// A setup operation or assert that was executed
	Action []TestReportSetupAction `json:"action,omitempty"`
}

// TestReportTestActionOperation represents a FHIR BackboneElement for TestReport.test.action.operation.
type TestReportTestActionOperation struct {
}

// TestReportTestActionAssert represents a FHIR BackboneElement for TestReport.test.action.assert.
type TestReportTestActionAssert struct {
}

// TestReportTestAction represents a FHIR BackboneElement for TestReport.test.action.
type TestReportTestAction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The operation performed
	Operation *TestReportTestActionOperation `json:"operation,omitempty"`
	// The assertion performed
	Assert *TestReportTestActionAssert `json:"assert,omitempty"`
}

// TestReportTest represents a FHIR BackboneElement for TestReport.test.
type TestReportTest struct {
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
	// A test operation or assert that was performed
	Action []TestReportTestAction `json:"action,omitempty"`
}

// TestReportTeardownActionOperation represents a FHIR BackboneElement for TestReport.teardown.action.operation.
type TestReportTeardownActionOperation struct {
}

// TestReportTeardownAction represents a FHIR BackboneElement for TestReport.teardown.action.
type TestReportTeardownAction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The teardown operation performed
	Operation TestReportTeardownActionOperation `json:"operation"`
}

// TestReportTeardown represents a FHIR BackboneElement for TestReport.teardown.
type TestReportTeardown struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// One or more teardown operations performed
	Action []TestReportTeardownAction `json:"action,omitempty"`
}

// TestReport represents a FHIR TestReport.
type TestReport struct {
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
	// External identifier
	Identifier *Identifier `json:"identifier,omitempty"`
	// Informal name of the executed TestScript
	Name *string `json:"name,omitempty"`
	// completed | in-progress | waiting | stopped | entered-in-error
	Status string `json:"status"`
	// Reference to the  version-specific TestScript that was executed to produce this TestReport
	TestScript Reference `json:"testScript"`
	// pass | fail | pending
	Result string `json:"result"`
	// The final score (percentage of tests passed) resulting from the execution of the TestScript
	Score *float64 `json:"score,omitempty"`
	// Name of the tester producing this report (Organization or individual)
	Tester *string `json:"tester,omitempty"`
	// When the TestScript was executed and this TestReport was generated
	Issued *primitives.DateTime `json:"issued,omitempty"`
	// A participant in the test execution, either the execution engine, a client, or a server
	Participant []TestReportParticipant `json:"participant,omitempty"`
	// The results of the series of required setup operations before the tests were executed
	Setup *TestReportSetup `json:"setup,omitempty"`
	// A test executed from the test script
	Test []TestReportTest `json:"test,omitempty"`
	// The results of running the series of required clean up steps
	Teardown *TestReportTeardown `json:"teardown,omitempty"`
}
