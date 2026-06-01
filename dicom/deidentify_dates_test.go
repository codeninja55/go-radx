package dicom

import (
	"testing"
	"time"
)

// By default the Basic Profile removes or zeroes all date/time attributes: no
// temporal data survives unless the caller opts in.
func TestDeidentifyRemovesDatesByDefault(t *testing.T) {
	ds := NewDataSet()
	ds.SetString(TagStudyInstanceUID, "1.2.3")
	ds.SetString(TagStudyDate, "20240115") // Z
	ds.SetString(TagStudyTime, "143000")   // Z
	ds.SetString(TagPatientBirthDate, "19800101")
	ds.SetString(TagAcquisitionDateTime, "20240115143000")

	clean, err := NewProfile(testGenerator(t)).Deidentify(ds)
	if err != nil {
		t.Fatalf("Deidentify: %v", err)
	}
	for _, tg := range []Tag{TagStudyDate, TagStudyTime, TagPatientBirthDate, TagAcquisitionDateTime} {
		e, ok := clean.Get(tg)
		if !ok {
			continue // removed is acceptable
		}
		if e.Value.EncodedLen(nil) != 0 {
			v, _ := clean.GetString(tg)
			t.Errorf("%s retained temporal data %q by default, want removed/zeroed", tg, v)
		}
	}
}

// WithRetainLongitudinalTemporalInformation(DateModeShift) keeps dates but shifts
// them by one consistent per-run offset, so absolute dates change while intervals are
// preserved.
func TestDeidentifyDateModeShiftPreservesIntervals(t *testing.T) {
	ds := NewDataSet()
	ds.SetString(TagStudyInstanceUID, "1.2.3.4.5")
	ds.SetString(TagStudyDate, "20240115")
	ds.SetString(TagAcquisitionDate, "20240120") // 5 days after StudyDate
	ds.SetString(TagAcquisitionDateTime, "20240120080000")

	prof := NewProfile(testGenerator(t), WithRetainLongitudinalTemporalInformation(DateModeShift))
	clean, err := prof.Deidentify(ds)
	if err != nil {
		t.Fatalf("Deidentify: %v", err)
	}

	sd, _ := clean.GetString(TagStudyDate)
	ad, _ := clean.GetString(TagAcquisitionDate)
	adt, _ := clean.GetString(TagAcquisitionDateTime)

	if sd == "20240115" {
		t.Errorf("StudyDate %q was not shifted", sd)
	}
	// Parse the shifted dates and confirm the 5-day interval is preserved.
	shiftedStudy, ok := parseYMD(t, sd)
	if !ok {
		return
	}
	shiftedAcq, ok := parseYMD(t, ad)
	if !ok {
		return
	}
	if got := shiftedAcq.Sub(shiftedStudy); got != 5*24*time.Hour {
		t.Errorf("interval after shift = %v, want 120h (5 days preserved)", got)
	}
	// The DateTime must carry the same calendar date as the shifted AcquisitionDate.
	if len(adt) < 8 || adt[:8] != ad {
		t.Errorf("AcquisitionDateTime date part %q != shifted AcquisitionDate %q", adt, ad)
	}
}

// DateModeKeep retains the date/time verbatim (full-dates retention).
func TestDeidentifyDateModeKeep(t *testing.T) {
	ds := NewDataSet()
	ds.SetString(TagStudyInstanceUID, "1.2.3")
	ds.SetString(TagStudyDate, "20240115")

	prof := NewProfile(testGenerator(t), WithRetainLongitudinalTemporalInformation(DateModeKeep))
	clean, err := prof.Deidentify(ds)
	if err != nil {
		t.Fatalf("Deidentify: %v", err)
	}
	if v, _ := clean.GetString(TagStudyDate); v != "20240115" {
		t.Errorf("StudyDate = %q, want verbatim 20240115 under DateModeKeep", v)
	}
}

// The shift is consistent for one study: two calls on the same StudyInstanceUID shift
// the same date the same way (deterministic per-study key).
func TestDeidentifyDateShiftIsStablePerStudy(t *testing.T) {
	build := func() *DataSet {
		ds := NewDataSet()
		ds.SetString(TagStudyInstanceUID, "1.2.3.4.repeatable")
		ds.SetString(TagStudyDate, "20240115")
		return ds
	}
	prof := NewProfile(testGenerator(t), WithRetainLongitudinalTemporalInformation(DateModeShift))

	a, err := prof.Deidentify(build())
	if err != nil {
		t.Fatalf("Deidentify a: %v", err)
	}
	b, err := prof.Deidentify(build())
	if err != nil {
		t.Fatalf("Deidentify b: %v", err)
	}
	av, _ := a.GetString(TagStudyDate)
	bv, _ := b.GetString(TagStudyDate)
	if av != bv {
		t.Errorf("date shift not stable for the same study: %q vs %q", av, bv)
	}
}

func parseYMD(t *testing.T, s string) (time.Time, bool) {
	t.Helper()
	tm, err := time.Parse("20060102", s)
	if err != nil {
		t.Errorf("shifted date %q is not YYYYMMDD: %v", s, err)
		return time.Time{}, false
	}
	return tm, true
}
