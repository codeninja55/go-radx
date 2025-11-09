package benchmarks

import (
	"testing"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicom/anonymize"
	"github.com/codeninja55/go-radx/dicom/element"
	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/value"
	"github.com/codeninja55/go-radx/dicom/vr"
)

// BenchmarkAnonymizerCreation measures anonymizer initialization
func BenchmarkAnonymizerCreation(b *testing.B) {
	profiles := []anonymize.Profile{
		anonymize.ProfileBasic,
		anonymize.ProfileClean,
		anonymize.ProfileRetainUIDs,
		anonymize.ProfileRetainDeviceIdentity,
	}

	for _, profile := range profiles {
		b.Run(string(rune(profile)), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = anonymize.NewAnonymizer(profile)
			}
		})
	}
}

// BenchmarkAnonymizeBasicDataSet measures basic anonymization performance
func BenchmarkAnonymizeBasicDataSet(b *testing.B) {
	anonymizer := anonymize.NewAnonymizer(anonymize.ProfileBasic)

	sizes := []int{10, 50, 100, 500}

	for _, size := range sizes {
		b.Run(string(rune(size))+"_elements", func(b *testing.B) {
			ds := setupAnonymizableDataSet(b, size)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = anonymizer.Anonymize(ds)
			}
		})
	}
}

// BenchmarkAnonymizeWithOptions measures anonymization with different options
func BenchmarkAnonymizeWithOptions(b *testing.B) {
	configs := []struct {
		name   string
		config anonymize.Config
	}{
		{
			name: "BasicProfile",
			config: anonymize.Config{
				Profile:     anonymize.ProfileBasic,
				PatientName: "ANONYMOUS",
				PatientID:   "ANON001",
				Options: anonymize.Options{
					RemovePrivateTags: true,
				},
			},
		},
		{
			name: "RetainUIDs",
			config: anonymize.Config{
				Profile:     anonymize.ProfileBasic,
				PatientName: "ANONYMOUS",
				PatientID:   "ANON001",
				Options: anonymize.Options{
					RemovePrivateTags: true,
					RetainUIDs:        true,
				},
			},
		},
		{
			name: "CleanDescriptors",
			config: anonymize.Config{
				Profile:     anonymize.ProfileClean,
				PatientName: "ANONYMOUS",
				PatientID:   "ANON001",
				Options: anonymize.Options{
					RemovePrivateTags: true,
					CleanDescriptors:  true,
				},
			},
		},
		{
			name: "RemoveOverlays",
			config: anonymize.Config{
				Profile:     anonymize.ProfileBasic,
				PatientName: "ANONYMOUS",
				PatientID:   "ANON001",
				Options: anonymize.Options{
					RemovePrivateTags: true,
					RemoveOverlays:    true,
				},
			},
		},
	}

	ds := setupAnonymizableDataSet(b, 100)

	for _, tc := range configs {
		b.Run(tc.name, func(b *testing.B) {
			anonymizer := anonymize.NewAnonymizerWithConfig(tc.config)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = anonymizer.Anonymize(ds)
			}
		})
	}
}

// BenchmarkDataSetAnonymizeBasic measures dataset helper anonymization
func BenchmarkDataSetAnonymizeBasic(b *testing.B) {
	sizes := []int{10, 50, 100}

	for _, size := range sizes {
		b.Run(string(rune(size))+"_elements", func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				ds := setupAnonymizableDataSet(b, size)
				b.StartTimer()
				_ = ds.AnonymizeBasic()
				b.StopTimer()
			}
		})
	}
}

// BenchmarkActionApplication measures performance of individual anonymization actions
func BenchmarkActionApplication(b *testing.B) {
	b.Run("ActionDummy", func(b *testing.B) {
		anonymizer := anonymize.NewAnonymizer(anonymize.ProfileBasic)
		ds := setupAnonymizableDataSet(b, 50)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = anonymizer.Anonymize(ds)
		}
	})

	b.Run("ActionRemove", func(b *testing.B) {
		config := anonymize.Config{
			Profile:     anonymize.ProfileCustom,
			PatientName: "ANONYMOUS",
			CustomActions: map[tag.Tag]anonymize.Action{
				tag.PatientName:      anonymize.ActionRemove,
				tag.PatientID:        anonymize.ActionRemove,
				tag.PatientBirthDate: anonymize.ActionRemove,
			},
		}
		anonymizer := anonymize.NewAnonymizerWithConfig(config)
		ds := setupAnonymizableDataSet(b, 50)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = anonymizer.Anonymize(ds)
		}
	})

	b.Run("ActionUID", func(b *testing.B) {
		anonymizer := anonymize.NewAnonymizer(anonymize.ProfileBasic)
		ds := setupAnonymizableDataSet(b, 50)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = anonymizer.Anonymize(ds)
		}
	})
}

// BenchmarkAnonymizeMemory measures memory usage during anonymization
func BenchmarkAnonymizeMemory(b *testing.B) {
	anonymizer := anonymize.NewAnonymizer(anonymize.ProfileBasic)

	sizes := []int{100, 500, 1000}

	for _, size := range sizes {
		b.Run(string(rune(size))+"_elements", func(b *testing.B) {
			ds := setupAnonymizableDataSet(b, size)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result, err := anonymizer.Anonymize(ds)
				if err != nil {
					b.Fatal(err)
				}
				_ = result
			}
		})
	}
}

// Helper functions

func setupAnonymizableDataSet(b *testing.B, numElements int) *dicom.DataSet {
	b.Helper()
	ds := dicom.NewDataSet()

	// Add patient identifying information
	patientTags := []struct {
		tag tag.Tag
		vr  vr.VR
		val string
	}{
		{tag.PatientName, vr.PersonName, "Smith^John^Robert^^Dr."},
		{tag.PatientID, vr.LongString, "PAT123456789"},
		{tag.PatientBirthDate, vr.Date, "19750315"},
		{tag.PatientSex, vr.CodeString, "M"},
		{tag.PatientAge, vr.AgeString, "048Y"},
		{tag.InstitutionName, vr.LongString, "General Hospital"},
		{tag.InstitutionAddress, vr.ShortText, "123 Main St, City, State 12345"},
		{tag.ReferringPhysicianName, vr.PersonName, "Jones^Mary^^^Dr."},
		{tag.StudyDescription, vr.LongString, "CT Chest with contrast for patient 123456"},
		{tag.SeriesDescription, vr.LongString, "Axial images"},
	}

	for _, t := range patientTags {
		val, _ := value.NewStringValue(t.vr, []string{t.val})
		elem, _ := element.NewElement(t.tag, t.vr, val)
		_ = ds.Add(elem)
	}

	// Add UIDs
	uidTags := []struct {
		tag tag.Tag
		val string
	}{
		{tag.StudyInstanceUID, "1.2.840.113619.2.55.3.604688119.123.1234567890.123"},
		{tag.SeriesInstanceUID, "1.2.840.113619.2.55.3.604688119.456.1234567890.456"},
		{tag.SOPInstanceUID, "1.2.840.113619.2.55.3.604688119.789.1234567890.789"},
	}

	for _, t := range uidTags {
		val, _ := value.NewStringValue(vr.UniqueIdentifier, []string{t.val})
		elem, _ := element.NewElement(t.tag, vr.UniqueIdentifier, val)
		_ = ds.Add(elem)
	}

	// Add private tags
	for i := 0; i < 10; i++ {
		privateTag := tag.New(0x0009, uint16(0x0010+i))
		val, _ := value.NewStringValue(vr.LongString, []string{"Private vendor data"})
		elem, _ := element.NewElement(privateTag, vr.LongString, val)
		_ = ds.Add(elem)
	}

	// Fill up to requested size with additional tags
	for i := len(patientTags) + len(uidTags) + 10; i < numElements; i++ {
		syntheticTag := tag.New(0x0010+uint16(i/256), uint16(i%256))
		val, _ := value.NewStringValue(vr.LongString, []string{"Additional data"})
		elem, _ := element.NewElement(syntheticTag, vr.LongString, val)
		_ = ds.Add(elem)
	}

	return ds
}
