package r5_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
)

func u32ptr(n uint32) *uint32 { return &n }

func TestImagingStudyResourceType(t *testing.T) {
	is := &r5.ImagingStudy{}
	if got := is.ResourceType(); got != "ImagingStudy" {
		t.Errorf("ResourceType() = %q, want ImagingStudy", got)
	}
	var _ fhir.Resource = is
}

func TestImagingStudyNestedSeriesAndInstance(t *testing.T) {
	is := &r5.ImagingStudy{
		Identifier:        []r5.Identifier{{System: strptr("urn:dicom:uid"), Value: strptr("urn:oid:1.2.840.113619.2.55.3.604688.1")}},
		Status:            "available",
		Subject:           &r5.Reference{Type: strptr("Patient"), Identifier: &r5.Identifier{System: strptr("urn:issuer"), Value: strptr("PID-1")}},
		Started:           strptr("2026-06-01T09:30:00Z"),
		NumberOfSeries:    u32ptr(1),
		NumberOfInstances: u32ptr(1),
		Description:       strptr("CT CHEST W CONTRAST"),
		Modality:          []r5.CodeableConcept{{Coding: []r5.Coding{{System: strptr("http://dicom.nema.org/resources/ontology/DCM"), Code: strptr("CT")}}}},
		Series: []r5.ImagingStudySeries{
			{
				Uid:      "1.2.840.113619.2.55.3.604688.2",
				Number:   u32ptr(1),
				Modality: r5.CodeableConcept{Coding: []r5.Coding{{System: strptr("http://dicom.nema.org/resources/ontology/DCM"), Code: strptr("CT")}}},
				BodySite: &r5.CodeableReference{Concept: &r5.CodeableConcept{Text: strptr("CHEST")}},
				Instance: []r5.ImagingStudySeriesInstance{
					{
						Uid:      "1.2.840.113619.2.55.3.604688.3",
						SopClass: r5.Coding{System: strptr("urn:ietf:rfc:3986"), Code: strptr("urn:oid:1.2.840.10008.5.1.4.1.1.2")},
						Number:   u32ptr(1),
					},
				},
			},
		},
	}
	b, err := json.Marshal(is)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if !strings.HasPrefix(got, `{"resourceType":"ImagingStudy"`) {
		t.Errorf("ImagingStudy JSON does not start with resourceType: %s", got)
	}
	want := `{"resourceType":"ImagingStudy",` +
		`"identifier":[{"system":"urn:dicom:uid","value":"urn:oid:1.2.840.113619.2.55.3.604688.1"}],` +
		`"status":"available",` +
		`"subject":{"type":"Patient","identifier":{"system":"urn:issuer","value":"PID-1"}},` +
		`"started":"2026-06-01T09:30:00Z","numberOfSeries":1,"numberOfInstances":1,` +
		`"modality":[{"coding":[{"system":"http://dicom.nema.org/resources/ontology/DCM","code":"CT"}]}],` +
		`"description":"CT CHEST W CONTRAST",` +
		`"series":[{"uid":"1.2.840.113619.2.55.3.604688.2","number":1,` +
		`"modality":{"coding":[{"system":"http://dicom.nema.org/resources/ontology/DCM","code":"CT"}]},` +
		`"bodySite":{"concept":{"text":"CHEST"}},` +
		`"instance":[{"uid":"1.2.840.113619.2.55.3.604688.3",` +
		`"sopClass":{"system":"urn:ietf:rfc:3986","code":"urn:oid:1.2.840.10008.5.1.4.1.1.2"},` +
		`"number":1}]}]}`
	if got != want {
		t.Errorf("ImagingStudy JSON\n got = %s\nwant = %s", got, want)
	}
}

func TestImagingStudyMinimalOmitsEmpty(t *testing.T) {
	is := &r5.ImagingStudy{Status: "available"}
	b, err := json.Marshal(is)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"resourceType":"ImagingStudy","status":"available"}`
	if string(b) != want {
		t.Errorf("ImagingStudy JSON = %s, want %s", b, want)
	}
}
