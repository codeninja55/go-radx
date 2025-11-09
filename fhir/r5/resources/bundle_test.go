package resources

import (
	"encoding/json"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/internal/testutil"
	"github.com/codeninja55/go-radx/fhir/primitives"
)

// TestBundle_EntryWithJSONRawMessage tests Bundle entries using json.RawMessage
func TestBundle_EntryWithJSONRawMessage(t *testing.T) {
	t.Run("create bundle with Patient entry", func(t *testing.T) {
		// Create a Patient
		patient := &Patient{
			Active: testutil.BoolPtr(true),
			Name: []HumanName{
				{
					Family: testutil.StringPtr("Doe"),
					Given:  []string{"John"},
				},
			},
			Gender: testutil.StringPtr("male"),
		}
		patient.ID = testutil.StringPtr("patient-123")
		patient.ResourceType = "Patient"

		// Marshal Patient to json.RawMessage
		patientJSON, err := json.Marshal(patient)
		if err != nil {
			t.Fatalf("Failed to marshal patient: %v", err)
		}

		// Create Bundle with the Patient entry
		bundle := &Bundle{
			Type: "searchset",
			Entry: []BundleEntry{
				{
					FullUrl:  testutil.StringPtr("Patient/patient-123"),
					Resource: patientJSON,
				},
			},
		}
		bundle.ResourceType = "Bundle"

		// Verify we can unmarshal the resource back
		if len(bundle.Entry) != 1 {
			t.Fatalf("Expected 1 entry, got %d", len(bundle.Entry))
		}

		retrievedPatient, err := fhir.UnmarshalResource[Patient](bundle.Entry[0].Resource)
		if err != nil {
			t.Fatalf("Failed to unmarshal patient from bundle: %v", err)
		}

		if retrievedPatient.ID == nil || *retrievedPatient.ID != "patient-123" {
			t.Errorf("Expected patient ID 'patient-123', got %v", retrievedPatient.ID)
		}

		if len(retrievedPatient.Name) != 1 || *retrievedPatient.Name[0].Family != "Doe" {
			t.Error("Patient name not preserved correctly")
		}
	})

	t.Run("bundle with multiple resource types", func(t *testing.T) {
		// Create Patient
		patient := &Patient{
			Active: testutil.BoolPtr(true),
		}
		patient.ID = testutil.StringPtr("p1")
		patient.ResourceType = "Patient"
		patientJSON, _ := json.Marshal(patient)

		// Create Observation
		obs := &Observation{
			Status: "final",
			Code: CodeableConcept{
				Text: testutil.StringPtr("Heart rate"),
			},
		}
		obs.ID = testutil.StringPtr("obs1")
		obs.ResourceType = "Observation"
		obsJSON, _ := json.Marshal(obs)

		// Create Bundle
		bundle := &Bundle{
			Type: "searchset",
			Entry: []BundleEntry{
				{
					FullUrl:  testutil.StringPtr("Patient/p1"),
					Resource: patientJSON,
				},
				{
					FullUrl:  testutil.StringPtr("Observation/obs1"),
					Resource: obsJSON,
				},
			},
		}
		bundle.ResourceType = "Bundle"

		// Verify we can unmarshal each resource type correctly
		p, err := fhir.UnmarshalResource[Patient](bundle.Entry[0].Resource)
		if err != nil {
			t.Fatalf("Failed to unmarshal patient: %v", err)
		}
		if p.ID == nil || *p.ID != "p1" {
			t.Errorf("Expected patient ID 'p1', got %v", p.ID)
		}

		o, err := fhir.UnmarshalResource[Observation](bundle.Entry[1].Resource)
		if err != nil {
			t.Fatalf("Failed to unmarshal observation: %v", err)
		}
		if o.ID == nil || *o.ID != "obs1" {
			t.Errorf("Expected observation ID 'obs1', got %v", o.ID)
		}
		if o.Status != "final" {
			t.Errorf("Expected status 'final', got %s", o.Status)
		}
	})
}

// TestBundleEntryResponse_OutcomeWithJSONRawMessage tests BundleEntryResponse.Outcome
func TestBundleEntryResponse_OutcomeWithJSONRawMessage(t *testing.T) {
	t.Run("bundle response with OperationOutcome", func(t *testing.T) {
		// Create an OperationOutcome
		outcome := &OperationOutcome{
			Issue: []OperationOutcomeIssue{
				{
					Severity:    "error",
					Code:        "invalid",
					Diagnostics: testutil.StringPtr("Resource validation failed"),
				},
			},
		}
		outcome.ResourceType = "OperationOutcome"
		outcomeJSON, err := json.Marshal(outcome)
		if err != nil {
			t.Fatalf("Failed to marshal OperationOutcome: %v", err)
		}

		// Create BundleEntryResponse with the outcome
		response := &BundleEntryResponse{
			Status:  "400 Bad Request",
			Outcome: outcomeJSON,
		}

		// Verify we can unmarshal the outcome
		retrievedOutcome, err := fhir.UnmarshalResource[OperationOutcome](response.Outcome)
		if err != nil {
			t.Fatalf("Failed to unmarshal outcome: %v", err)
		}

		if len(retrievedOutcome.Issue) != 1 {
			t.Fatalf("Expected 1 issue, got %d", len(retrievedOutcome.Issue))
		}

		if retrievedOutcome.Issue[0].Severity != "error" {
			t.Errorf("Expected severity 'error', got %s", retrievedOutcome.Issue[0].Severity)
		}
	})
}

// TestBundle_IssuesWithJSONRawMessage tests Bundle.Issues field
func TestBundle_IssuesWithJSONRawMessage(t *testing.T) {
	t.Run("bundle with issues", func(t *testing.T) {
		// Create an OperationOutcome for bundle-level issues
		issues := &OperationOutcome{
			Issue: []OperationOutcomeIssue{
				{
					Severity:    "warning",
					Code:        "processing",
					Diagnostics: testutil.StringPtr("Some entries were filtered"),
				},
			},
		}
		issues.ResourceType = "OperationOutcome"
		issuesJSON, err := json.Marshal(issues)
		if err != nil {
			t.Fatalf("Failed to marshal issues: %v", err)
		}

		// Create Bundle with issues
		bundle := &Bundle{
			Type:   "searchset",
			Issues: issuesJSON,
		}
		bundle.ResourceType = "Bundle"

		// Verify we can unmarshal the issues
		retrievedIssues, err := fhir.UnmarshalResource[OperationOutcome](bundle.Issues)
		if err != nil {
			t.Fatalf("Failed to unmarshal issues: %v", err)
		}

		if len(retrievedIssues.Issue) != 1 {
			t.Fatalf("Expected 1 issue, got %d", len(retrievedIssues.Issue))
		}

		if retrievedIssues.Issue[0].Code != "processing" {
			t.Errorf("Expected code 'processing', got %s", retrievedIssues.Issue[0].Code)
		}
	})
}

// TestBundle_RoundTrip tests JSON roundtrip for Bundle with json.RawMessage
func TestBundle_RoundTrip(t *testing.T) {
	t.Run("roundtrip bundle with multiple entries", func(t *testing.T) {
		// Create original bundle
		patient := &Patient{
			Active: testutil.BoolPtr(true),
			Gender: testutil.StringPtr("female"),
		}
		patient.ID = testutil.StringPtr("p1")
		patient.ResourceType = "Patient"
		patientJSON, _ := json.Marshal(patient)

		totalCount := uint(1)
		originalBundle := &Bundle{
			Type:  "searchset",
			Total: &totalCount,
			Entry: []BundleEntry{
				{
					FullUrl:  testutil.StringPtr("Patient/p1"),
					Resource: patientJSON,
				},
			},
		}
		originalBundle.ID = testutil.StringPtr("bundle-123")
		originalBundle.ResourceType = "Bundle"

		// Marshal to JSON
		bundleJSON, err := json.Marshal(originalBundle)
		if err != nil {
			t.Fatalf("Failed to marshal bundle: %v", err)
		}

		// Unmarshal back
		var retrievedBundle Bundle
		if err := json.Unmarshal(bundleJSON, &retrievedBundle); err != nil {
			t.Fatalf("Failed to unmarshal bundle: %v", err)
		}

		// Verify bundle properties
		if retrievedBundle.Type != "searchset" {
			t.Errorf("Expected type 'searchset', got %s", retrievedBundle.Type)
		}

		if retrievedBundle.Total == nil || *retrievedBundle.Total != 1 {
			t.Errorf("Expected total 1, got %v", retrievedBundle.Total)
		}

		if len(retrievedBundle.Entry) != 1 {
			t.Fatalf("Expected 1 entry, got %d", len(retrievedBundle.Entry))
		}

		// Verify we can still unmarshal the embedded resource
		p, err := fhir.UnmarshalResource[Patient](retrievedBundle.Entry[0].Resource)
		if err != nil {
			t.Fatalf("Failed to unmarshal patient from roundtrip bundle: %v", err)
		}

		if p.Gender == nil || *p.Gender != "female" {
			t.Errorf("Expected gender 'female', got %v", p.Gender)
		}
	})
}

// TestBundle_TransactionBundle tests transaction bundle with request/response
func TestBundle_TransactionBundle(t *testing.T) {
	t.Run("transaction bundle with request and response", func(t *testing.T) {
		// Create a patient to add
		patient := &Patient{
			Active: testutil.BoolPtr(true),
		}
		patient.ResourceType = "Patient"
		patientJSON, _ := json.Marshal(patient)

		// Create transaction bundle
		bundle := &Bundle{
			Type: "transaction",
			Entry: []BundleEntry{
				{
					Resource: patientJSON,
					Request: &BundleEntryRequest{
						Method: "POST",
						URL:    "Patient",
					},
				},
			},
		}
		bundle.ResourceType = "Bundle"

		// Verify the request is set correctly
		if len(bundle.Entry) != 1 {
			t.Fatalf("Expected 1 entry, got %d", len(bundle.Entry))
		}

		if bundle.Entry[0].Request == nil {
			t.Fatal("Expected request to be set")
		}

		if bundle.Entry[0].Request.Method != "POST" {
			t.Errorf("Expected method 'POST', got %s", bundle.Entry[0].Request.Method)
		}
	})

	t.Run("transaction response with outcome", func(t *testing.T) {
		// Create response with OperationOutcome
		outcome := &OperationOutcome{
			Issue: []OperationOutcomeIssue{
				{
					Severity: "information",
					Code:     "informational",
				},
			},
		}
		outcome.ResourceType = "OperationOutcome"
		outcomeJSON, _ := json.Marshal(outcome)

		// Create bundle entry response
		response := &BundleEntryResponse{
			Status:   "201 Created",
			Location: testutil.StringPtr("Patient/123/_history/1"),
			Etag:     testutil.StringPtr("W/\"1\""),
			Outcome:  outcomeJSON,
		}

		// Verify outcome can be unmarshaled
		retrieved, err := fhir.UnmarshalResource[OperationOutcome](response.Outcome)
		if err != nil {
			t.Fatalf("Failed to unmarshal outcome: %v", err)
		}

		if len(retrieved.Issue) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(retrieved.Issue))
		}
	})
}

// TestBundle_SearchBundle tests search bundle with search information
func TestBundle_SearchBundle(t *testing.T) {
	t.Run("search bundle with search metadata", func(t *testing.T) {
		patient := &Patient{}
		patient.ID = testutil.StringPtr("p1")
		patient.ResourceType = "Patient"
		patientJSON, _ := json.Marshal(patient)

		totalCount := uint(100)
		bundle := &Bundle{
			Type:  "searchset",
			Total: &totalCount,
			Link: []BundleLink{
				{
					Relation: "self",
					URL:      "http://example.org/Patient?name=Smith",
				},
				{
					Relation: "next",
					URL:      "http://example.org/Patient?name=Smith&page=2",
				},
			},
			Entry: []BundleEntry{
				{
					FullUrl:  testutil.StringPtr("http://example.org/Patient/p1"),
					Resource: patientJSON,
					Search: &BundleEntrySearch{
						Mode:  testutil.StringPtr("match"),
						Score: testutil.Float64Ptr(1.0),
					},
				},
			},
		}
		bundle.ResourceType = "Bundle"

		// Verify search metadata
		if bundle.Total == nil || *bundle.Total != 100 {
			t.Errorf("Expected total 100, got %v", bundle.Total)
		}

		if len(bundle.Link) != 2 {
			t.Errorf("Expected 2 links, got %d", len(bundle.Link))
		}

		if bundle.Entry[0].Search == nil {
			t.Fatal("Expected search metadata")
		}

		if bundle.Entry[0].Search.Score == nil || *bundle.Entry[0].Search.Score != 1.0 {
			t.Errorf("Expected score 1.0, got %v", bundle.Entry[0].Search.Score)
		}
	})
}

// TestBundle_EmptyResource tests handling of empty/nil json.RawMessage
func TestBundle_EmptyResource(t *testing.T) {
	t.Run("bundle entry with nil resource", func(t *testing.T) {
		bundle := &Bundle{
			Type: "searchset",
			Entry: []BundleEntry{
				{
					FullUrl:  testutil.StringPtr("Patient/deleted"),
					Resource: nil, // Deleted resource
				},
			},
		}
		bundle.ResourceType = "Bundle"

		// Verify marshaling works with nil resource
		bundleJSON, err := json.Marshal(bundle)
		if err != nil {
			t.Fatalf("Failed to marshal bundle with nil resource: %v", err)
		}

		// Verify unmarshaling works
		var retrieved Bundle
		if err := json.Unmarshal(bundleJSON, &retrieved); err != nil {
			t.Fatalf("Failed to unmarshal bundle: %v", err)
		}

		if len(retrieved.Entry) != 1 {
			t.Errorf("Expected 1 entry, got %d", len(retrieved.Entry))
		}
	})
}

// TestBundle_WithTimestamp tests Bundle with timestamp
func TestBundle_WithTimestamp(t *testing.T) {
	t.Run("bundle with timestamp", func(t *testing.T) {
		timestamp := primitives.MustInstant("2024-01-15T10:30:00Z")

		bundle := &Bundle{
			Type:      "searchset",
			Timestamp: &timestamp,
		}
		bundle.ResourceType = "Bundle"

		// Marshal and unmarshal
		bundleJSON, err := json.Marshal(bundle)
		if err != nil {
			t.Fatalf("Failed to marshal bundle: %v", err)
		}

		var retrieved Bundle
		if err := json.Unmarshal(bundleJSON, &retrieved); err != nil {
			t.Fatalf("Failed to unmarshal bundle: %v", err)
		}

		if retrieved.Timestamp == nil {
			t.Fatal("Expected timestamp to be preserved")
		}

		if retrieved.Timestamp.String() != timestamp.String() {
			t.Errorf("Timestamp mismatch: got %s, want %s", retrieved.Timestamp.String(), timestamp.String())
		}
	})
}
