package convert

import (
	"testing"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/hl7v2"
)

// TestHL7DateTimeToFHIR confirms the HL7 DTM lexical form is rendered to a valid
// FHIR dateTime at the source precision, never the packed HL7 form. A timed DTM
// renders only when it carries a +/-ZZZZ offset (FHIR requires a timezone);
// otherwise it falls back to date-only and records the dropped time.
func TestHL7DateTimeToFHIR(t *testing.T) {
	cases := []struct {
		in          string
		want        string
		wantDropped bool
	}{
		{in: "", want: ""},
		{in: "2026", want: "2026"},
		{in: "202605", want: "2026-05"},
		{in: "20260531", want: "2026-05-31"},
		{in: "202605311230-0500", want: "2026-05-31T12:30:00-05:00"},
		{in: "20260531123045+0000", want: "2026-05-31T12:30:45Z"},
		{in: "20260531123045.123-0500", want: "2026-05-31T12:30:45.123-05:00"}, // fraction preserved
		{in: "202605311230", want: "2026-05-31", wantDropped: true},            // no offset: time dropped
		{in: "202605311230+2460", want: "2026-05-31", wantDropped: true},       // bad offset: time dropped
	}
	for _, c := range cases {
		dtm, err := hl7v2.ParseDTM(c.in)
		if err != nil {
			t.Fatalf("ParseDTM(%q): %v", c.in, err)
		}
		report := &Report{}
		if got := hl7DateTimeToFHIR(dtm, report, "test.path"); got != c.want {
			t.Errorf("hl7DateTimeToFHIR(%q) = %q, want %q", c.in, got, c.want)
		}
		if (len(report.Dropped) > 0) != c.wantDropped {
			t.Errorf("hl7DateTimeToFHIR(%q) dropped = %v, want %v", c.in, report.Dropped, c.wantDropped)
		}
	}
}

// TestCombineDateTimePreservesFraction confirms a DICOM TM fractional second is
// carried into the FHIR dateTime (with the timezone offset) rather than truncated.
func TestCombineDateTimePreservesFraction(t *testing.T) {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagStudyDate, "20040826")
	ds.SetString(dicom.TagStudyTime, "185059.123456")
	ds.SetString(dicom.TagTimezoneOffsetFromUTC, "-0500")

	report := &Report{}
	got := combineDateTime(ds, dicom.TagStudyDate, dicom.TagStudyTime, report, "test.path")
	if got != "2004-08-26T18:50:59.123456-05:00" {
		t.Errorf("combineDateTime = %q, want 2004-08-26T18:50:59.123456-05:00", got)
	}
}

// TestCombineDateTimeNoOffsetDropsTime confirms a timed DICOM value without a
// timezone offset falls back to date-only and records the dropped time, never
// emitting a FHIR-invalid timezone-less time.
func TestCombineDateTimeNoOffsetDropsTime(t *testing.T) {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagStudyDate, "20040826")
	ds.SetString(dicom.TagStudyTime, "185059")

	report := &Report{}
	got := combineDateTime(ds, dicom.TagStudyDate, dicom.TagStudyTime, report, "test.path")
	if got != "2004-08-26" {
		t.Errorf("combineDateTime = %q, want 2004-08-26 (time dropped for lack of offset)", got)
	}
	if len(report.Dropped) == 0 {
		t.Error("combineDateTime did not record the dropped time")
	}
}

// TestCombineDateTimeDateOnly confirms a date with no time yields just the date.
func TestCombineDateTimeDateOnly(t *testing.T) {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagStudyDate, "20040826")

	report := &Report{}
	if got := combineDateTime(ds, dicom.TagStudyDate, dicom.TagStudyTime, report, "test.path"); got != "2004-08-26" {
		t.Errorf("combineDateTime = %q, want 2004-08-26", got)
	}
}

// TestDICOMToImagingStudyR5DropsInstanceWithoutSOPClass is the fail-closed
// regression for a required-field gap: an instance with no SOP Class UID is
// dropped (recorded) rather than emitting an invalid series.instance.sopClass.
func TestDICOMToImagingStudyR5DropsInstanceWithoutSOPClass(t *testing.T) {
	good := dicom.NewDataSet()
	good.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.4")
	good.SetString(dicom.TagSOPInstanceUID, "1.2.3.1.1")
	good.SetString(dicom.TagStudyInstanceUID, "1.2.3")
	good.SetString(dicom.TagSeriesInstanceUID, "1.2.3.1")
	good.SetString(dicom.TagModality, "MR")

	noClass := dicom.NewDataSet()
	noClass.SetString(dicom.TagSOPInstanceUID, "1.2.3.1.2")
	noClass.SetString(dicom.TagStudyInstanceUID, "1.2.3")
	noClass.SetString(dicom.TagSeriesInstanceUID, "1.2.3.1")
	noClass.SetString(dicom.TagModality, "MR")

	study, report, err := DICOMToImagingStudyR5([]*dicom.DataSet{good, noClass})
	if err != nil {
		t.Fatalf("DICOMToImagingStudyR5: %v", err)
	}
	if study.NumberOfInstances == nil || *study.NumberOfInstances != 1 {
		t.Errorf("NumberOfInstances = %v, want 1 (the SOP-class-less instance is dropped)", study.NumberOfInstances)
	}
	if len(study.Series) != 1 || len(study.Series[0].Instance) != 1 {
		t.Fatalf("series/instances = %d/%v, want one series with one instance",
			len(study.Series), study.Series)
	}
	// The kept instance carries a valid required sopClass.
	if study.Series[0].Instance[0].SopClass.Code == nil {
		t.Error("kept instance has an empty required sopClass")
	}
	// The drop is recorded, naming the missing SOP Class UID tag.
	if !hasDropped(report, "DICOM (0008,0016) SOPClassUID") {
		t.Errorf("Report.Dropped does not record the SOP-class-less instance: %+v", report.Dropped)
	}
}

// hasDropped reports whether the report records a drop whose source names src.
func hasDropped(r *Report, src string) bool {
	for _, d := range r.Dropped {
		if d.Source == src {
			return true
		}
	}
	return false
}
