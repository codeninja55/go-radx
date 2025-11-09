package benchmarks

import (
	"fmt"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicom/anonymize"
	"github.com/codeninja55/go-radx/dicom/element"
	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/value"
	"github.com/codeninja55/go-radx/dicom/vr"
)

// BenchmarkAnonymizeVsManualClean compares anonymization approaches
func BenchmarkAnonymizeVsManualClean(b *testing.B) {
	sizes := []int{50, 100, 500}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Anonymizer_%d_elements", size), func(b *testing.B) {
			anonymizer := anonymize.NewAnonymizer(anonymize.ProfileBasic)
			template := setupAnonymizableDataSet(b, size)

			avgBytesPerElement := int64(100)
			b.SetBytes(int64(size) * avgBytesPerElement)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ds := template.Copy()
				_, _ = anonymizer.Anonymize(ds)
			}
		})

		b.Run(fmt.Sprintf("ManualWalk_%d_elements", size), func(b *testing.B) {
			template := setupAnonymizableDataSet(b, size)

			avgBytesPerElement := int64(100)
			b.SetBytes(int64(size) * avgBytesPerElement)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ds := template.Copy()
				_ = ds.WalkModify(func(elem *element.Element) (bool, error) {
					switch elem.Tag() {
					case tag.PatientName, tag.PatientID, tag.PatientBirthDate:
						_ = ds.Remove(elem.Tag())
						return true, nil
					}
					return false, nil
				})
			}
		})
	}
}

// BenchmarkDataSetCopyVsNew compares dataset copying strategies
func BenchmarkDataSetCopyVsNew(b *testing.B) {
	sizes := []int{10, 50, 100, 500}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Copy_%d_elements", size), func(b *testing.B) {
			template := setupLargeDataSet(b, size)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = template.Copy()
			}
		})

		b.Run(fmt.Sprintf("DeepCopy_%d_elements", size), func(b *testing.B) {
			template := setupLargeDataSet(b, size)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Deep copy by walking and creating new elements
				newDS := dicom.NewDataSet()
				_ = template.Walk(func(elem *element.Element) error {
					_ = newDS.Add(elem)
					return nil
				})
			}
		})
	}
}

// BenchmarkBatchVsSequentialAdd compares element addition strategies
func BenchmarkBatchVsSequentialAdd(b *testing.B) {
	numElements := []int{10, 50, 100}

	for _, count := range numElements {
		b.Run(fmt.Sprintf("Sequential_%d_elements", count), func(b *testing.B) {
			elements := make([]*element.Element, count)
			for i := 0; i < count; i++ {
				t := tag.New(0x0010, uint16(i))
				val, _ := value.NewStringValue(vr.LongString, []string{"Test"})
				elements[i], _ = element.NewElement(t, vr.LongString, val)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ds := dicom.NewDataSet()
				for _, elem := range elements {
					_ = ds.Add(elem)
				}
			}
		})

		b.Run(fmt.Sprintf("PreallocatedMap_%d_elements", count), func(b *testing.B) {
			elements := make([]*element.Element, count)
			for i := 0; i < count; i++ {
				t := tag.New(0x0010, uint16(i))
				val, _ := value.NewStringValue(vr.LongString, []string{"Test"})
				elements[i], _ = element.NewElement(t, vr.LongString, val)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Note: DataSet doesn't expose capacity hint,
				// but this shows the pattern for comparison
				ds := dicom.NewDataSet()
				for _, elem := range elements {
					_ = ds.Add(elem)
				}
			}
		})
	}
}

// BenchmarkLookupStrategies compares different tag lookup approaches
func BenchmarkLookupStrategies(b *testing.B) {
	sizes := []int{50, 100, 500}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("DirectGet_%d_elements", size), func(b *testing.B) {
			ds := setupLargeDataSet(b, size)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = ds.Get(tag.PatientName)
			}
		})

		b.Run(fmt.Sprintf("ContainsThenGet_%d_elements", size), func(b *testing.B) {
			ds := setupLargeDataSet(b, size)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if ds.Contains(tag.PatientName) {
					_, _ = ds.Get(tag.PatientName)
				}
			}
		})

		b.Run(fmt.Sprintf("WalkToFind_%d_elements", size), func(b *testing.B) {
			ds := setupLargeDataSet(b, size)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = ds.Walk(func(elem *element.Element) error {
					if elem.Tag() == tag.PatientName {
						return nil // Found it
					}
					return nil
				})
			}
		})
	}
}

// BenchmarkRemovalStrategies compares different element removal approaches
func BenchmarkRemovalStrategies(b *testing.B) {
	b.Run("DirectRemove", func(b *testing.B) {
		template := setupLargeDataSet(b, 100)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ds := template.Copy()
			_ = ds.Remove(tag.PatientName)
			_ = ds.Remove(tag.PatientID)
			_ = ds.Remove(tag.PatientBirthDate)
		}
	})

	b.Run("WalkModifyRemove", func(b *testing.B) {
		template := setupLargeDataSet(b, 100)

		tagsToRemove := []tag.Tag{
			tag.PatientName,
			tag.PatientID,
			tag.PatientBirthDate,
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ds := template.Copy()
			_ = ds.WalkModify(func(elem *element.Element) (bool, error) {
				for _, t := range tagsToRemove {
					if elem.Tag() == t {
						_ = ds.Remove(t)
						return true, nil
					}
				}
				return false, nil
			})
		}
	})

	b.Run("RemoveGroupTags", func(b *testing.B) {
		template := setupLargeDataSet(b, 100)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ds := template.Copy()
			_ = ds.RemoveGroupTags(0x0010) // Remove entire patient group
		}
	})
}

// BenchmarkPrivateTagFiltering compares private tag filtering strategies
func BenchmarkPrivateTagFiltering(b *testing.B) {
	b.Run("RemovePrivateTags", func(b *testing.B) {
		template := setupDataSetWithPrivateTags(b, 100, 50)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ds := template.Copy()
			_ = ds.RemovePrivateTags()
		}
	})

	b.Run("WalkModifyPrivate", func(b *testing.B) {
		template := setupDataSetWithPrivateTags(b, 100, 50)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ds := template.Copy()
			_ = ds.WalkModify(func(elem *element.Element) (bool, error) {
				if elem.Tag().Group%2 == 1 { // Odd group = private
					_ = ds.Remove(elem.Tag())
					return true, nil
				}
				return false, nil
			})
		}
	})
}

// BenchmarkMergeStrategies compares dataset merging approaches
func BenchmarkMergeStrategies(b *testing.B) {
	sizes := []int{50, 100, 200}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("MergeMethod_%d_elements", size), func(b *testing.B) {
			ds1 := setupLargeDataSet(b, size)
			ds2 := setupLargeDataSet(b, size/2)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dsCopy := ds1.Copy()
				_ = dsCopy.Merge(ds2)
			}
		})

		b.Run(fmt.Sprintf("ManualWalk_%d_elements", size), func(b *testing.B) {
			ds1 := setupLargeDataSet(b, size)
			ds2 := setupLargeDataSet(b, size/2)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dsCopy := ds1.Copy()
				_ = ds2.Walk(func(elem *element.Element) error {
					return dsCopy.Add(elem)
				})
			}
		})
	}
}
