package dicom

import "testing"

// The action table must cover the PS3.15 Annex E Basic Profile confidentiality
// attributes, not the sparse top-level subset the prototype shipped (Codex DCM-013).
func TestBasicProfileActionsCoverConfidentialityAttributes(t *testing.T) {
	// A representative slice of Table E.1-1 attributes spanning the patient, study,
	// series, equipment, image, and SOP-common modules, with their basic-profile
	// action. Each must be present with the expected action.
	cases := []struct {
		keyword string
		want    deidAction
	}{
		{"PatientName", deidReplaceDummy},
		{"PatientID", deidReplaceDummy},
		{"PatientBirthDate", deidReplaceZero},
		{"PatientBirthTime", deidRemove},
		{"OtherPatientIDs", deidRemove},
		{"OtherPatientNames", deidRemove},
		{"PatientBirthName", deidRemove},
		{"PatientMotherBirthName", deidRemove},
		{"PatientAddress", deidRemove},
		{"PatientTelephoneNumbers", deidRemove},
		{"EthnicGroup", deidRemove},
		{"PatientComments", deidRemove},
		{"AdditionalPatientHistory", deidRemove},
		{"MilitaryRank", deidRemove},
		{"Occupation", deidRemove},
		{"ReferringPhysicianName", deidReplaceZero},
		{"ReferringPhysicianAddress", deidRemove},
		{"ReferringPhysicianTelephoneNumbers", deidRemove},
		{"PhysiciansOfRecord", deidRemove},
		{"NameOfPhysiciansReadingStudy", deidRemove},
		{"PerformingPhysicianName", deidReplaceZero},
		{"OperatorsName", deidReplaceZero},
		{"StudyDate", deidReplaceZero},
		{"StudyTime", deidReplaceZero},
		{"StudyID", deidReplaceZero},
		{"AccessionNumber", deidReplaceZero},
		{"StudyDescription", deidClean},
		{"SeriesDescription", deidClean},
		{"ProtocolName", deidClean},
		{"InstitutionName", deidRemove},
		{"InstitutionAddress", deidRemove},
		{"StationName", deidRemove},
		{"DeviceSerialNumber", deidRemove},
		{"InstanceCreatorUID", deidReplaceUID},
		{"StudyInstanceUID", deidReplaceUID},
		{"SeriesInstanceUID", deidReplaceUID},
		{"SOPInstanceUID", deidReplaceUID},
		{"FrameOfReferenceUID", deidReplaceUID},
		{"ReferencedSOPInstanceUID", deidReplaceUID},
		{"AcquisitionDate", deidReplaceZero},
		{"AcquisitionTime", deidReplaceZero},
		{"AcquisitionDateTime", deidReplaceZero},
		{"ContentDate", deidReplaceZero},
		{"ContentTime", deidReplaceZero},
		{"PersonName", deidRemove},
		{"PersonAddress", deidRemove},
		{"DigitalSignaturesSequence", deidRemove},
		{"ReferencedImageSequence", deidReplaceZero},
		{"SourceImageSequence", deidReplaceZero},
		{"RequestAttributesSequence", deidReplaceZero},
	}

	for _, c := range cases {
		t.Run(c.keyword, func(t *testing.T) {
			tag, ok := LookupKeyword(c.keyword)
			if !ok {
				t.Fatalf("keyword %q absent from the dictionary", c.keyword)
			}
			got, ok := basicProfileAction(tag)
			if !ok {
				t.Fatalf("%s (%s) has no basic-profile action", c.keyword, tag)
			}
			if got != c.want {
				t.Errorf("%s action = %v, want %v", c.keyword, got, c.want)
			}
		})
	}
}

// The table must be large enough to be a real profile, not the handful the prototype
// shipped (Codex DCM-013 called out the sparseness explicitly).
func TestBasicProfileActionTableIsNotSparse(t *testing.T) {
	if n := len(basicProfileActions); n < 100 {
		t.Errorf("basic-profile action table has %d entries, expected a substantial Table E.1-1 set (>= 100)", n)
	}
}

// Every UID-remap attribute must carry the UI VR, otherwise the remap would attempt
// to rewrite a non-UID value.
func TestUIDRemapAttributesAreUIVR(t *testing.T) {
	for tag, action := range basicProfileActions {
		if action != deidReplaceUID {
			continue
		}
		info, ok := Lookup(tag)
		if !ok {
			t.Errorf("UID-remap tag %s is not in the dictionary", tag)
			continue
		}
		if info.VR != VRUI {
			t.Errorf("UID-remap tag %s (%s) has VR %s, want UI", info.Keyword, tag, info.VR)
		}
	}
}
