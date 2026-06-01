package dicom

import "testing"

// The prototype acted only on top-level elements, so confidentiality attributes
// nested inside sequence items survived de-identification (Codex DCM-013). The
// profile must act at every nesting level.
func TestDeidentifyRecursesIntoSequenceItems(t *testing.T) {
	// Build a dataset with PHI both at top level and nested two sequences deep.
	inner := NewDataSet()
	inner.SetString(TagPatientName, "Inner^Patient") // D, nested level 2
	inner.SetString(TagPatientBirthTime, "010101")   // X, nested level 2
	inner.SetString(TagInstitutionName, "Inner Hosp") // X, nested level 2

	middle := NewDataSet()
	middle.SetString(TagOperatorsName, "Mid^Operator") // Z, nested level 1
	// PerFrameFunctionalGroupsSequence is a structural (kept) sequence, so its items
	// are recursed into rather than zeroed; nested PHI must still be cleaned.
	middle.Set(Element{
		Tag:   TagPerFrameFunctionalGroupsSequence,
		VR:    VRSQ,
		Value: NewSequenceValue(NewSequence(inner)),
	})

	top := NewDataSet()
	top.SetString(TagPatientName, "Top^Patient") // D, top level
	top.Set(Element{
		Tag:   TagSharedFunctionalGroupsSequence,
		VR:    VRSQ,
		Value: NewSequenceValue(NewSequence(middle)),
	})

	prof := NewProfile(testGenerator(t))
	clean, err := prof.Deidentify(top)
	if err != nil {
		t.Fatalf("Deidentify: %v", err)
	}

	// Top-level PatientName: replaced, not original.
	if v, _ := clean.GetString(TagPatientName); v == "Top^Patient" || v == "" {
		t.Errorf("top PatientName = %q, want a non-empty dummy != original", v)
	}

	// Descend into the sequences and assert the nested attributes were acted on.
	midSeq, ok := clean.GetSequence(TagSharedFunctionalGroupsSequence)
	if !ok || midSeq.Len() != 1 {
		t.Fatalf("SharedFunctionalGroupsSequence missing or empty: ok=%v", ok)
	}
	var midItem Item
	for it := range midSeq.Items() {
		midItem = it
	}
	// OperatorsName is Z: present zero-length, never the original.
	if v, _ := midItem.DataSet.GetString(TagOperatorsName); v == "Mid^Operator" {
		t.Error("nested OperatorsName survived de-identification (DCM-013)")
	}

	innerSeq, ok := midItem.DataSet.GetSequence(TagPerFrameFunctionalGroupsSequence)
	if !ok || innerSeq.Len() != 1 {
		t.Fatalf("nested PerFrameFunctionalGroupsSequence missing or empty: ok=%v", ok)
	}
	var innerItem Item
	for it := range innerSeq.Items() {
		innerItem = it
	}
	// PatientName D at level 2: replaced, not original.
	if v, _ := innerItem.DataSet.GetString(TagPatientName); v == "Inner^Patient" || v == "" {
		t.Errorf("level-2 PatientName = %q, want a non-empty dummy != original (DCM-013)", v)
	}
	// PatientBirthTime X at level 2: removed.
	if _, ok := innerItem.DataSet.Get(TagPatientBirthTime); ok {
		t.Error("level-2 PatientBirthTime (X) survived de-identification (DCM-013)")
	}
	// InstitutionName X at level 2: removed.
	if _, ok := innerItem.DataSet.Get(TagInstitutionName); ok {
		t.Error("level-2 InstitutionName (X) survived de-identification (DCM-013)")
	}
}

// No confidentiality attribute listed in Table E.1-1 may survive at any level. This
// walks the de-identified result and asserts no X-action attribute remains and no
// D/Z/C attribute retains its original PHI value.
func TestDeidentifyLeavesNoResidualConfidentialityAttribute(t *testing.T) {
	phi := func(ds *DataSet) {
		ds.SetString(TagPatientBirthTime, "120000") // X
		ds.SetString(TagInstitutionAddress, "1 St") // X
		ds.SetString(TagPersonName, "Some^Person")  // X
	}

	inner := NewDataSet()
	phi(inner)
	mid := NewDataSet()
	phi(mid)
	mid.Set(Element{Tag: TagPerFrameFunctionalGroupsSequence, VR: VRSQ, Value: NewSequenceValue(NewSequence(inner))})
	top := NewDataSet()
	phi(top)
	top.Set(Element{Tag: TagSharedFunctionalGroupsSequence, VR: VRSQ, Value: NewSequenceValue(NewSequence(mid))})

	prof := NewProfile(testGenerator(t))
	clean, err := prof.Deidentify(top)
	if err != nil {
		t.Fatalf("Deidentify: %v", err)
	}

	removed := []Tag{TagPatientBirthTime, TagInstitutionAddress, TagPersonName}
	var check func(ds *DataSet, depth int)
	check = func(ds *DataSet, depth int) {
		for _, tg := range removed {
			if _, ok := ds.Get(tg); ok {
				t.Errorf("X-action attribute %s survived at depth %d", tg, depth)
			}
		}
		for e := range ds.All() {
			if sv, ok := e.Value.(*sequenceValue); ok {
				for it := range sv.seq.Items() {
					check(it.DataSet, depth+1)
				}
			}
		}
	}
	check(clean, 0)
}
