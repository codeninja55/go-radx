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

// BenchmarkMemoryDataSetGrowth measures memory growth patterns
func BenchmarkMemoryDataSetGrowth(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("%d_elements", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				ds := dicom.NewDataSet()
				for j := 0; j < size; j++ {
					t := tag.New(0x0010+uint16(j/256), uint16(j%256))
					val, _ := value.NewStringValue(vr.LongString, []string{"Test Value"})
					elem, _ := element.NewElement(t, vr.LongString, val)
					_ = ds.Add(elem)
				}
			}
		})
	}
}

// BenchmarkMemoryElementCreation measures element allocation overhead
func BenchmarkMemoryElementCreation(b *testing.B) {
	valueTypes := []struct {
		name string
		vr   vr.VR
		data []string
	}{
		{"ShortString", vr.ShortString, []string{"Test"}},
		{"LongString", vr.LongString, []string{"This is a longer test string"}},
		{"PersonName", vr.PersonName, []string{"Doe^John^Robert^^Dr."}},
		{"UniqueIdentifier", vr.UniqueIdentifier, []string{"1.2.840.113619.2.55.3.604688119"}},
	}

	for _, vt := range valueTypes {
		b.Run(vt.name, func(b *testing.B) {
			val, _ := value.NewStringValue(vt.vr, vt.data)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = element.NewElement(tag.PatientName, vt.vr, val)
			}
		})
	}
}

// BenchmarkMemoryValueAllocation measures value allocation patterns
func BenchmarkMemoryValueAllocation(b *testing.B) {
	b.Run("StringValue_Small", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = value.NewStringValue(vr.ShortString, []string{"Test"})
		}
	})

	b.Run("StringValue_Large", func(b *testing.B) {
		largeString := make([]byte, 1024)
		for i := range largeString {
			largeString[i] = 'X'
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = value.NewStringValue(vr.LongText, []string{string(largeString)})
		}
	})

	b.Run("StringValue_Multiple", func(b *testing.B) {
		values := []string{"Value1", "Value2", "Value3", "Value4", "Value5"}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = value.NewStringValue(vr.LongString, values)
		}
	})
}

// BenchmarkMemoryDataSetCopy measures memory overhead of copying
func BenchmarkMemoryDataSetCopy(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("%d_elements", size), func(b *testing.B) {
			template := setupLargeDataSet(b, size)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = template.Copy()
			}
		})
	}
}

// BenchmarkMemoryAnonymization measures anonymization memory usage
func BenchmarkMemoryAnonymization(b *testing.B) {
	sizes := []int{100, 500, 1000, 5000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Basic_%d_elements", size), func(b *testing.B) {
			anonymizer := anonymize.NewAnonymizer(anonymize.ProfileBasic)
			template := setupAnonymizableDataSet(b, size)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ds := template.Copy()
				_, _ = anonymizer.Anonymize(ds)
			}
		})

		b.Run(fmt.Sprintf("Clean_%d_elements", size), func(b *testing.B) {
			anonymizer := anonymize.NewAnonymizer(anonymize.ProfileClean)
			template := setupAnonymizableDataSet(b, size)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ds := template.Copy()
				_, _ = anonymizer.Anonymize(ds)
			}
		})
	}
}

// BenchmarkMemoryWalkOperations measures walk iteration memory overhead
func BenchmarkMemoryWalkOperations(b *testing.B) {
	sizes := []int{100, 500, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Walk_%d_elements", size), func(b *testing.B) {
			ds := setupLargeDataSet(b, size)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = ds.Walk(func(elem *element.Element) error {
					// Simulate some work
					_ = elem.Tag()
					_ = elem.VR()
					return nil
				})
			}
		})

		b.Run(fmt.Sprintf("WalkModify_%d_elements", size), func(b *testing.B) {
			template := setupLargeDataSet(b, size)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ds := template.Copy()
				_ = ds.WalkModify(func(elem *element.Element) (bool, error) {
					if elem.VR() == vr.PersonName {
						newVal, _ := value.NewStringValue(vr.PersonName, []string{"ANONYMOUS"})
						_ = elem.SetValue(newVal)
						return true, nil
					}
					return false, nil
				})
			}
		})
	}
}

// BenchmarkMemoryPrivateTagRemoval measures memory during tag removal
func BenchmarkMemoryPrivateTagRemoval(b *testing.B) {
	privateCounts := []int{10, 50, 100, 500}

	for _, count := range privateCounts {
		b.Run(fmt.Sprintf("%d_private_tags", count), func(b *testing.B) {
			template := setupDataSetWithPrivateTags(b, 100, count)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ds := template.Copy()
				_ = ds.RemovePrivateTags()
			}
		})
	}
}

// BenchmarkMemoryMergeOperations measures memory during dataset merging
func BenchmarkMemoryMergeOperations(b *testing.B) {
	configs := []struct {
		name string
		ds1  int
		ds2  int
	}{
		{"Small_Small", 50, 50},
		{"Small_Large", 50, 500},
		{"Large_Large", 500, 500},
	}

	for _, cfg := range configs {
		b.Run(cfg.name, func(b *testing.B) {
			ds1 := setupLargeDataSet(b, cfg.ds1)
			ds2 := setupLargeDataSet(b, cfg.ds2)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dsCopy := ds1.Copy()
				_ = dsCopy.Merge(ds2)
			}
		})
	}
}

// BenchmarkMemoryRepeatedOperations measures memory in repeated operations
func BenchmarkMemoryRepeatedOperations(b *testing.B) {
	b.Run("RepeatedAdd", func(b *testing.B) {
		ds := dicom.NewDataSet()

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			t := tag.New(0x0010, uint16(i%256))
			val, _ := value.NewStringValue(vr.LongString, []string{"Test"})
			elem, _ := element.NewElement(t, vr.LongString, val)
			_ = ds.Add(elem)
		}
	})

	b.Run("RepeatedRemove", func(b *testing.B) {
		// Pre-populate dataset
		template := setupLargeDataSet(b, 1000)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ds := template.Copy()
			tags := ds.Tags()
			for j := 0; j < len(tags) && j < 100; j++ {
				_ = ds.Remove(tags[j])
			}
		}
	})

	b.Run("RepeatedGetSet", func(b *testing.B) {
		template := setupLargeDataSet(b, 100)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ds := template.Copy()
			for j := 0; j < 10; j++ {
				elem, _ := ds.Get(tag.PatientName)
				if elem != nil {
					newVal, _ := value.NewStringValue(vr.PersonName, []string{"Updated"})
					_ = elem.SetValue(newVal)
				}
			}
		}
	})
}

// BenchmarkMemoryLargeDataSet measures memory for very large datasets
func BenchmarkMemoryLargeDataSet(b *testing.B) {
	// Simulate real-world large DICOM files
	sizes := []int{1000, 5000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("CreateAndPopulate_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				ds := dicom.NewDataSet()

				// Add standard tags
				for j := 0; j < size; j++ {
					t := tag.New(0x0010+uint16(j/256), uint16(j%256))
					val, _ := value.NewStringValue(vr.LongString, []string{"Data"})
					elem, _ := element.NewElement(t, vr.LongString, val)
					_ = ds.Add(elem)
				}
			}
		})

		b.Run(fmt.Sprintf("CopyAndModify_%d", size), func(b *testing.B) {
			template := setupLargeDataSet(b, size)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				ds := template.Copy()
				_ = ds.WalkModify(func(elem *element.Element) (bool, error) {
					if elem.Tag().Group == 0x0010 {
						newVal, _ := value.NewStringValue(vr.LongString, []string{"Modified"})
						_ = elem.SetValue(newVal)
						return true, nil
					}
					return false, nil
				})
			}
		})
	}
}
