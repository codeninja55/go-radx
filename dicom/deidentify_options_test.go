package dicom

import "testing"

func TestDeidentifyRetainPatientCharacteristics(t *testing.T) {
	ds := NewDataSet()
	ds.SetString(TagStudyInstanceUID, "1.2.3")
	ds.SetString(TagPatientSex, "F")
	ds.SetString(TagPatientAge, "045Y")

	// Default: characteristics removed/zeroed.
	def, err := NewProfile(testGenerator(t)).Deidentify(ds)
	if err != nil {
		t.Fatalf("Deidentify default: %v", err)
	}
	if v, _ := def.GetString(TagPatientSex); v == "F" {
		t.Error("PatientSex retained without the opt-in")
	}

	// Opt-in: characteristics kept.
	kept, err := NewProfile(testGenerator(t), WithRetainPatientCharacteristics()).Deidentify(ds)
	if err != nil {
		t.Fatalf("Deidentify retain: %v", err)
	}
	if v, _ := kept.GetString(TagPatientSex); v != "F" {
		t.Errorf("PatientSex = %q, want F under WithRetainPatientCharacteristics", v)
	}
	if v, _ := kept.GetString(TagPatientAge); v != "045Y" {
		t.Errorf("PatientAge = %q, want 045Y under WithRetainPatientCharacteristics", v)
	}
}

func TestDeidentifyRetainDeviceIdentity(t *testing.T) {
	ds := NewDataSet()
	ds.SetString(TagStudyInstanceUID, "1.2.3")
	ds.SetString(TagStationName, "CT01")
	ds.SetString(TagDeviceSerialNumber, "SN12345")

	def, err := NewProfile(testGenerator(t)).Deidentify(ds)
	if err != nil {
		t.Fatalf("Deidentify default: %v", err)
	}
	if _, ok := def.Get(TagStationName); ok {
		t.Error("StationName retained without the device-identity opt-in")
	}

	kept, err := NewProfile(testGenerator(t), WithRetainDeviceIdentity()).Deidentify(ds)
	if err != nil {
		t.Fatalf("Deidentify retain: %v", err)
	}
	if v, _ := kept.GetString(TagStationName); v != "CT01" {
		t.Errorf("StationName = %q, want CT01 under WithRetainDeviceIdentity", v)
	}
	if v, _ := kept.GetString(TagDeviceSerialNumber); v != "SN12345" {
		t.Errorf("DeviceSerialNumber = %q, want SN12345 under WithRetainDeviceIdentity", v)
	}
}

func TestDeidentifyWithDummyValues(t *testing.T) {
	ds := NewDataSet()
	ds.SetString(TagStudyInstanceUID, "1.2.3")
	ds.SetString(TagPatientID, "MRN-9988")

	prof := NewProfile(testGenerator(t), WithDummyValues(map[Tag]string{
		TagPatientID: "STUDY-001",
	}))
	clean, err := prof.Deidentify(ds)
	if err != nil {
		t.Fatalf("Deidentify: %v", err)
	}
	if v, _ := clean.GetString(TagPatientID); v != "STUDY-001" {
		t.Errorf("PatientID = %q, want the supplied dummy STUDY-001", v)
	}
}

// Private tags are removed by default (the Basic Profile's safe action).
func TestDeidentifyRemovesPrivateTagsByDefault(t *testing.T) {
	ds := NewDataSet()
	ds.SetString(TagStudyInstanceUID, "1.2.3")
	creator := NewTag(0x0009, 0x0010)     // private creator (0009,0010)
	privateData := NewTag(0x0009, 0x1001) // private element under that creator
	ds.Set(Element{Tag: creator, VR: VRLO, Value: NewStrings(VRLO, "ACME")})
	ds.Set(Element{Tag: privateData, VR: VRLO, Value: NewStrings(VRLO, "secret")})

	clean, err := NewProfile(testGenerator(t)).Deidentify(ds)
	if err != nil {
		t.Fatalf("Deidentify: %v", err)
	}
	if _, ok := clean.Get(creator); ok {
		t.Error("private creator retained without WithRetainSafePrivate")
	}
	if _, ok := clean.Get(privateData); ok {
		t.Error("private data element retained without WithRetainSafePrivate")
	}
}

// WithRetainSafePrivate keeps private attributes whose creator is allow-listed and
// still removes those whose creator is not.
func TestDeidentifyRetainSafePrivate(t *testing.T) {
	ds := NewDataSet()
	ds.SetString(TagStudyInstanceUID, "1.2.3")
	safeCreator := NewTag(0x0009, 0x0010)
	safeData := NewTag(0x0009, 0x1001)
	unsafeCreator := NewTag(0x000B, 0x0010)
	unsafeData := NewTag(0x000B, 0x1002)
	ds.Set(Element{Tag: safeCreator, VR: VRLO, Value: NewStrings(VRLO, "SAFECO")})
	ds.Set(Element{Tag: safeData, VR: VRLO, Value: NewStrings(VRLO, "keepme")})
	ds.Set(Element{Tag: unsafeCreator, VR: VRLO, Value: NewStrings(VRLO, "OTHERCO")})
	ds.Set(Element{Tag: unsafeData, VR: VRLO, Value: NewStrings(VRLO, "dropme")})

	prof := NewProfile(testGenerator(t), WithRetainSafePrivate("SAFECO"))
	clean, err := prof.Deidentify(ds)
	if err != nil {
		t.Fatalf("Deidentify: %v", err)
	}
	if v, ok := clean.GetString(safeData); !ok || v != "keepme" {
		t.Errorf("allow-listed private data = %q,%v, want keepme", v, ok)
	}
	if _, ok := clean.Get(safeCreator); !ok {
		t.Error("allow-listed private creator should be retained")
	}
	if _, ok := clean.Get(unsafeData); ok {
		t.Error("non-allow-listed private data should be removed")
	}
	if _, ok := clean.Get(unsafeCreator); ok {
		t.Error("non-allow-listed private creator should be removed")
	}
}
