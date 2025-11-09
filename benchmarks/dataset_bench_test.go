package benchmarks

import (
	"testing"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicom/element"
	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/value"
	"github.com/codeninja55/go-radx/dicom/vr"
)

// BenchmarkDataSetCreation measures dataset creation performance
func BenchmarkDataSetCreation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = dicom.NewDataSet()
	}
}

// BenchmarkElementCreation measures element creation performance
func BenchmarkElementCreation(b *testing.B) {
	b.ReportAllocs()
	val, _ := value.NewStringValue(vr.PersonName, []string{"Test^Patient"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = element.NewElement(tag.PatientName, vr.PersonName, val)
	}
}

// BenchmarkDataSetAdd measures performance of adding elements
func BenchmarkDataSetAdd(b *testing.B) {
	ds := dicom.NewDataSet()
	val, _ := value.NewStringValue(vr.PersonName, []string{"Test^Patient"})
	elem, _ := element.NewElement(tag.PatientName, vr.PersonName, val)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ds.Add(elem)
	}
}

// BenchmarkDataSetGet measures tag lookup performance
func BenchmarkDataSetGet(b *testing.B) {
	ds := setupLargeDataSet(b, 100)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ds.Get(tag.PatientName)
	}
}

// BenchmarkDataSetContains measures tag existence checking
func BenchmarkDataSetContains(b *testing.B) {
	ds := setupLargeDataSet(b, 100)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ds.Contains(tag.PatientName)
	}
}

// BenchmarkDataSetWalk measures iteration performance
func BenchmarkDataSetWalk(b *testing.B) {
	sizes := []int{10, 50, 100, 500, 1000}

	for _, size := range sizes {
		b.Run(string(rune(size))+"_elements", func(b *testing.B) {
			ds := setupLargeDataSet(b, size)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = ds.Walk(func(elem *element.Element) error {
					return nil
				})
			}
		})
	}
}

// BenchmarkDataSetWalkModify measures modification during iteration
func BenchmarkDataSetWalkModify(b *testing.B) {
	sizes := []int{10, 50, 100, 500}

	for _, size := range sizes {
		b.Run(string(rune(size))+"_elements", func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				ds := setupLargeDataSet(b, size)
				b.StartTimer()
			_ = ds.WalkModify(func(elem *element.Element) (bool, error) {
				if elem.VR() == vr.PersonName {
					newVal, _ := value.NewStringValue(vr.PersonName, []string{"ANONYMOUS"})
					_ = elem.SetValue(newVal)
					return true, nil
				}
				return false, nil
			})
				b.StopTimer()
			}
		})
	}
}

// BenchmarkDataSetModifyTags measures tag modification helpers
func BenchmarkDataSetModifyTags(b *testing.B) {
	b.Run("SetPatientName", func(b *testing.B) {
		ds := dicom.NewDataSet()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ds.SetPatientName("Doe^John")
		}
	})

	b.Run("SetPatientID", func(b *testing.B) {
		ds := dicom.NewDataSet()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ds.SetPatientID("123456")
		}
	})

	b.Run("SetStudyInstanceUID", func(b *testing.B) {
		ds := dicom.NewDataSet()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ds.SetStudyInstanceUID("")
		}
	})

	b.Run("GenerateNewUIDs", func(b *testing.B) {
		ds := dicom.NewDataSet()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ds.GenerateNewUIDs()
		}
	})
}

// BenchmarkDataSetRemovePrivateTags measures private tag removal
func BenchmarkDataSetRemovePrivateTags(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ds := setupDataSetWithPrivateTags(b, 100, 50)
		b.StartTimer()
		_ = ds.RemovePrivateTags()
		b.StopTimer()
	}
}

// BenchmarkDataSetRemoveGroupTags measures group tag removal
func BenchmarkDataSetRemoveGroupTags(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ds := setupLargeDataSet(b, 100)
		b.StartTimer()
		_ = ds.RemoveGroupTags(0x0010) // Remove patient group
		b.StopTimer()
	}
}

// Helper functions

func setupLargeDataSet(b *testing.B, numElements int) *dicom.DataSet {
	b.Helper()
	ds := dicom.NewDataSet()

	// Add common tags
	tags := []struct {
		tag tag.Tag
		vr  vr.VR
		val string
	}{
		{tag.PatientName, vr.PersonName, "Doe^John^A"},
		{tag.PatientID, vr.LongString, "123456"},
		{tag.PatientBirthDate, vr.Date, "19800515"},
		{tag.StudyDescription, vr.LongString, "CT Chest"},
		{tag.SeriesDescription, vr.LongString, "Axial"},
		{tag.InstitutionName, vr.LongString, "Test Hospital"},
		{tag.Modality, vr.CodeString, "CT"},
		{tag.Manufacturer, vr.LongString, "Test Manufacturer"},
	}

	for i := 0; i < numElements && i < len(tags); i++ {
		t := tags[i%len(tags)]
		val, _ := value.NewStringValue(t.vr, []string{t.val})
		elem, _ := element.NewElement(t.tag, t.vr, val)
		_ = ds.Add(elem)
	}

	// Add more elements if needed
	for i := len(tags); i < numElements; i++ {
		// Create synthetic tags
		syntheticTag := tag.New(0x0010+uint16(i/256), uint16(i%256))
		val, _ := value.NewStringValue(vr.LongString, []string{"Test Value"})
		elem, _ := element.NewElement(syntheticTag, vr.LongString, val)
		_ = ds.Add(elem)
	}

	return ds
}

func setupDataSetWithPrivateTags(b *testing.B, numPublic, numPrivate int) *dicom.DataSet {
	b.Helper()
	ds := setupLargeDataSet(b, numPublic)

	// Add private tags (odd group numbers)
	for i := 0; i < numPrivate; i++ {
		privateTag := tag.New(0x0009, uint16(i)) // Group 0x0009 is private
		val, _ := value.NewStringValue(vr.LongString, []string{"Private Data"})
		elem, _ := element.NewElement(privateTag, vr.LongString, val)
		_ = ds.Add(elem)
	}

	return ds
}
