package dicom_test

import (
	"fmt"
	"sync"
	"testing"

	dicom "github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/vr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test dataset with all required UIDs for collection
func createTestDataSetForCollection(
	sopInstanceUID string,
	seriesInstanceUID string,
	studyInstanceUID string,
	patientID string,
	accessionNumber string,
	sopClassUID string,
	seriesNumber int,
) *dicom.DataSet {
	ds := dicom.NewDataSet()

	// SOPInstanceUID (0008,0018) - Required
	_ = ds.Add(mustNewElement(tag.New(0x0008, 0x0018), vr.UniqueIdentifier,
		mustNewStringValue(vr.UniqueIdentifier, []string{sopInstanceUID})))

	// SeriesInstanceUID (0020,000E) - Required
	_ = ds.Add(mustNewElement(tag.New(0x0020, 0x000E), vr.UniqueIdentifier,
		mustNewStringValue(vr.UniqueIdentifier, []string{seriesInstanceUID})))

	// StudyInstanceUID (0020,000D) - Required
	_ = ds.Add(mustNewElement(tag.New(0x0020, 0x000D), vr.UniqueIdentifier,
		mustNewStringValue(vr.UniqueIdentifier, []string{studyInstanceUID})))

	// PatientID (0010,0020) - Required
	_ = ds.Add(mustNewElement(tag.New(0x0010, 0x0020), vr.LongString,
		mustNewStringValue(vr.LongString, []string{patientID})))

	// AccessionNumber (0008,0050) - Optional but indexed
	if accessionNumber != "" {
		_ = ds.Add(mustNewElement(tag.New(0x0008, 0x0050), vr.ShortString,
			mustNewStringValue(vr.ShortString, []string{accessionNumber})))
	}

	// SOPClassUID (0008,0016) - Required
	_ = ds.Add(mustNewElement(tag.New(0x0008, 0x0016), vr.UniqueIdentifier,
		mustNewStringValue(vr.UniqueIdentifier, []string{sopClassUID})))

	// SeriesNumber (0020,0011) - Optional but indexed for ordering
	if seriesNumber > 0 {
		_ = ds.Add(mustNewElement(tag.New(0x0020, 0x0011), vr.IntegerString,
			mustNewStringValue(vr.IntegerString, []string{fmt.Sprintf("%d", seriesNumber)})))
	}

	// Add some additional common elements
	_ = ds.Add(mustNewElement(tag.New(0x0010, 0x0010), vr.PersonName,
		mustNewStringValue(vr.PersonName, []string{"Test^Patient"})))

	return ds
}

// TestDataSetCollection_NewDataSetCollection tests creating a new empty collection
func TestDataSetCollection_NewDataSetCollection(t *testing.T) {
	t.Run("empty collection", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()
		assert.NotNil(t, coll)
		assert.Equal(t, 0, coll.Len())
	})
}

// TestDataSetCollection_NewDataSetCollectionWithDataSets tests creating collection with initial datasets
func TestDataSetCollection_NewDataSetCollectionWithDataSets(t *testing.T) {
	t.Run("valid datasets", func(t *testing.T) {
		datasets := []*dicom.DataSet{
			createTestDataSetForCollection("1.2.3.1", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1),
			createTestDataSetForCollection("1.2.3.2", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 2),
		}

		coll, err := dicom.NewDataSetCollectionWithDataSets(datasets)
		require.NoError(t, err)
		assert.NotNil(t, coll)
		assert.Equal(t, 2, coll.Len())
	})

	t.Run("nil datasets slice", func(t *testing.T) {
		coll, err := dicom.NewDataSetCollectionWithDataSets(nil)
		require.NoError(t, err)
		assert.NotNil(t, coll)
		assert.Equal(t, 0, coll.Len())
	})

	t.Run("dataset with missing SOPInstanceUID", func(t *testing.T) {
		ds := dicom.NewDataSet()
		// Missing SOPInstanceUID - should error

		coll, err := dicom.NewDataSetCollectionWithDataSets([]*dicom.DataSet{ds})
		assert.Error(t, err)
		assert.Nil(t, coll)
		assert.Contains(t, err.Error(), "SOPInstanceUID")
	})

	t.Run("duplicate SOPInstanceUID", func(t *testing.T) {
		datasets := []*dicom.DataSet{
			createTestDataSetForCollection("1.2.3.1", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1),
			createTestDataSetForCollection("1.2.3.1", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 2),
		}

		coll, err := dicom.NewDataSetCollectionWithDataSets(datasets)
		assert.Error(t, err)
		assert.Nil(t, coll)
		assert.Contains(t, err.Error(), "duplicate")
	})
}

// TestDataSetCollection_Add tests adding datasets to collection
func TestDataSetCollection_Add(t *testing.T) {
	t.Run("add single dataset", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()
		ds := createTestDataSetForCollection("1.2.3.1", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1)

		err := coll.Add(ds)
		assert.NoError(t, err)
		assert.Equal(t, 1, coll.Len())
	})

	t.Run("add multiple datasets", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		ds1 := createTestDataSetForCollection("1.2.3.1", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1)
		ds2 := createTestDataSetForCollection("1.2.3.2", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 2)

		require.NoError(t, coll.Add(ds1))
		require.NoError(t, coll.Add(ds2))
		assert.Equal(t, 2, coll.Len())
	})

	t.Run("add nil dataset", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()
		err := coll.Add(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("add dataset with missing SOPInstanceUID", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()
		ds := dicom.NewDataSet()

		err := coll.Add(ds)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SOPInstanceUID")
	})

	t.Run("add duplicate SOPInstanceUID", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		ds1 := createTestDataSetForCollection("1.2.3.1", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1)
		ds2 := createTestDataSetForCollection("1.2.3.1", "1.2.3.200", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 2)

		require.NoError(t, coll.Add(ds1))

		err := coll.Add(ds2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate")
		assert.Equal(t, 1, coll.Len()) // Should not have added the duplicate
	})
}

// TestDataSetCollection_GetBySOPInstanceUID tests retrieving by SOPInstanceUID
func TestDataSetCollection_GetBySOPInstanceUID(t *testing.T) {
	t.Run("get existing dataset", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()
		ds := createTestDataSetForCollection("1.2.3.1", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1)
		require.NoError(t, coll.Add(ds))

		retrieved, err := coll.GetBySOPInstanceUID("1.2.3.1")
		require.NoError(t, err)
		assert.NotNil(t, retrieved)

		// Verify it's the same dataset
		elem, _ := retrieved.GetByKeyword("SOPInstanceUID")
		assert.Equal(t, "1.2.3.1", elem.Value().String())
	})

	t.Run("get non-existent dataset", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		retrieved, err := coll.GetBySOPInstanceUID("1.2.3.999")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("get from empty collection", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		retrieved, err := coll.GetBySOPInstanceUID("1.2.3.1")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})
}

// TestDataSetCollection_GetBySeriesInstanceUID tests retrieving by SeriesInstanceUID
func TestDataSetCollection_GetBySeriesInstanceUID(t *testing.T) {
	t.Run("get datasets from series", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		// Add 3 datasets in same series
		ds1 := createTestDataSetForCollection("1.2.3.1", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1)
		ds2 := createTestDataSetForCollection("1.2.3.2", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 2)
		ds3 := createTestDataSetForCollection("1.2.3.3", "1.2.3.200", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1)

		require.NoError(t, coll.Add(ds1))
		require.NoError(t, coll.Add(ds2))
		require.NoError(t, coll.Add(ds3))

		datasets := coll.GetBySeriesInstanceUID("1.2.3.100")
		assert.Len(t, datasets, 2) // Should return ds1 and ds2
	})

	t.Run("get from non-existent series", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		datasets := coll.GetBySeriesInstanceUID("1.2.3.999")
		assert.Empty(t, datasets)
	})
}

// TestDataSetCollection_GetByStudyInstanceUID tests retrieving by StudyInstanceUID
func TestDataSetCollection_GetByStudyInstanceUID(t *testing.T) {
	t.Run("get datasets from study", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		// Add datasets from 2 series in same study
		ds1 := createTestDataSetForCollection("1.2.3.1", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1)
		ds2 := createTestDataSetForCollection("1.2.3.2", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 2)
		ds3 := createTestDataSetForCollection("1.2.3.3", "1.2.3.200", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1)
		ds4 := createTestDataSetForCollection("1.2.3.4", "1.2.3.300", "1.2.3.2000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1)

		require.NoError(t, coll.Add(ds1))
		require.NoError(t, coll.Add(ds2))
		require.NoError(t, coll.Add(ds3))
		require.NoError(t, coll.Add(ds4))

		datasets := coll.GetByStudyInstanceUID("1.2.3.1000")
		assert.Len(t, datasets, 3) // Should return ds1, ds2, ds3
	})

	t.Run("get from non-existent study", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		datasets := coll.GetByStudyInstanceUID("1.2.3.999")
		assert.Empty(t, datasets)
	})
}

// TestDataSetCollection_GetByPatientID tests retrieving by PatientID
func TestDataSetCollection_GetByPatientID(t *testing.T) {
	t.Run("get datasets for patient", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		// Add datasets for 2 patients
		ds1 := createTestDataSetForCollection("1.2.3.1", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1)
		ds2 := createTestDataSetForCollection("1.2.3.2", "1.2.3.200", "1.2.3.2000", "P001", "A002", "1.2.840.10008.5.1.4.1.1.2", 1)
		ds3 := createTestDataSetForCollection("1.2.3.3", "1.2.3.300", "1.2.3.3000", "P002", "A003", "1.2.840.10008.5.1.4.1.1.2", 1)

		require.NoError(t, coll.Add(ds1))
		require.NoError(t, coll.Add(ds2))
		require.NoError(t, coll.Add(ds3))

		datasets := coll.GetByPatientID("P001")
		assert.Len(t, datasets, 2) // Should return ds1 and ds2
	})

	t.Run("get from non-existent patient", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		datasets := coll.GetByPatientID("P999")
		assert.Empty(t, datasets)
	})
}

// TestDataSetCollection_GetByAccessionNumber tests retrieving by AccessionNumber
func TestDataSetCollection_GetByAccessionNumber(t *testing.T) {
	t.Run("get datasets by accession number", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		ds1 := createTestDataSetForCollection("1.2.3.1", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1)
		ds2 := createTestDataSetForCollection("1.2.3.2", "1.2.3.200", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1)
		ds3 := createTestDataSetForCollection("1.2.3.3", "1.2.3.300", "1.2.3.2000", "P001", "A002", "1.2.840.10008.5.1.4.1.1.2", 1)

		require.NoError(t, coll.Add(ds1))
		require.NoError(t, coll.Add(ds2))
		require.NoError(t, coll.Add(ds3))

		datasets := coll.GetByAccessionNumber("A001")
		assert.Len(t, datasets, 2) // Should return ds1 and ds2
	})

	t.Run("dataset without accession number", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		// No accession number (empty string)
		ds := createTestDataSetForCollection("1.2.3.1", "1.2.3.100", "1.2.3.1000", "P001", "", "1.2.840.10008.5.1.4.1.1.2", 1)
		require.NoError(t, coll.Add(ds))

		datasets := coll.GetByAccessionNumber("")
		assert.Len(t, datasets, 1)
	})
}

// TestDataSetCollection_GetBySOPClassUID tests retrieving by SOPClassUID
func TestDataSetCollection_GetBySOPClassUID(t *testing.T) {
	t.Run("get datasets by SOP class", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		// CT Image Storage
		ds1 := createTestDataSetForCollection("1.2.3.1", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1)
		// MR Image Storage
		ds2 := createTestDataSetForCollection("1.2.3.2", "1.2.3.200", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.4", 1)
		// CT Image Storage
		ds3 := createTestDataSetForCollection("1.2.3.3", "1.2.3.300", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1)

		require.NoError(t, coll.Add(ds1))
		require.NoError(t, coll.Add(ds2))
		require.NoError(t, coll.Add(ds3))

		datasets := coll.GetBySOPClassUID("1.2.840.10008.5.1.4.1.1.2")
		assert.Len(t, datasets, 2) // Should return ds1 and ds3 (both CT)
	})
}

// TestDataSetCollection_GetBySeriesNumber tests retrieving by SeriesNumber
func TestDataSetCollection_GetBySeriesNumber(t *testing.T) {
	t.Run("get datasets by series number", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		ds1 := createTestDataSetForCollection("1.2.3.1", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1)
		ds2 := createTestDataSetForCollection("1.2.3.2", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 2)
		ds3 := createTestDataSetForCollection("1.2.3.3", "1.2.3.200", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1)

		require.NoError(t, coll.Add(ds1))
		require.NoError(t, coll.Add(ds2))
		require.NoError(t, coll.Add(ds3))

		datasets := coll.GetBySeriesNumber(1)
		assert.Len(t, datasets, 2) // Should return ds1 and ds3
	})

	t.Run("dataset without series number", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		// No series number (0)
		ds := createTestDataSetForCollection("1.2.3.1", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 0)
		require.NoError(t, coll.Add(ds))

		datasets := coll.GetBySeriesNumber(0)
		assert.Len(t, datasets, 1)
	})
}

// TestDataSetCollection_GetSeriesNumberRange tests range queries on SeriesNumber
func TestDataSetCollection_GetSeriesNumberRange(t *testing.T) {
	t.Run("get datasets in range", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		for i := 1; i <= 10; i++ {
			ds := createTestDataSetForCollection(
				fmt.Sprintf("1.2.3.%d", i),
				"1.2.3.100",
				"1.2.3.1000",
				"P001",
				"A001",
				"1.2.840.10008.5.1.4.1.1.2",
				i,
			)
			require.NoError(t, coll.Add(ds))
		}

		// Get series 3-7 (inclusive)
		datasets := coll.GetSeriesNumberRange(3, 7)
		assert.Len(t, datasets, 5) // Should return series 3, 4, 5, 6, 7

		// Verify ordering by series number
		for i, ds := range datasets {
			elem, _ := ds.GetByKeyword("SeriesNumber")
			expected := int64(i + 3)
			assert.Equal(t, fmt.Sprintf("%d", expected), elem.Value().String())
		}
	})

	t.Run("empty range", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		datasets := coll.GetSeriesNumberRange(5, 10)
		assert.Empty(t, datasets)
	})

	t.Run("invalid range", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		ds := createTestDataSetForCollection("1.2.3.1", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 5)
		require.NoError(t, coll.Add(ds))

		// Start > End should return empty
		datasets := coll.GetSeriesNumberRange(10, 5)
		assert.Empty(t, datasets)
	})
}

// TestDataSetCollection_Remove tests removing datasets from collection
func TestDataSetCollection_Remove(t *testing.T) {
	t.Run("remove existing dataset", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		ds := createTestDataSetForCollection("1.2.3.1", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1)
		require.NoError(t, coll.Add(ds))
		assert.Equal(t, 1, coll.Len())

		err := coll.Remove("1.2.3.1")
		assert.NoError(t, err)
		assert.Equal(t, 0, coll.Len())
	})

	t.Run("remove non-existent dataset", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		err := coll.Remove("1.2.3.999")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("remove updates indexes", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		ds1 := createTestDataSetForCollection("1.2.3.1", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1)
		ds2 := createTestDataSetForCollection("1.2.3.2", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 2)

		require.NoError(t, coll.Add(ds1))
		require.NoError(t, coll.Add(ds2))

		// Remove ds1
		require.NoError(t, coll.Remove("1.2.3.1"))

		// Verify indexes updated
		datasets := coll.GetBySeriesInstanceUID("1.2.3.100")
		assert.Len(t, datasets, 1) // Only ds2 should remain
	})
}

// TestDataSetCollection_Contains tests checking dataset existence
func TestDataSetCollection_Contains(t *testing.T) {
	t.Run("contains existing dataset", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		ds := createTestDataSetForCollection("1.2.3.1", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1)
		require.NoError(t, coll.Add(ds))

		assert.True(t, coll.Contains("1.2.3.1"))
	})

	t.Run("does not contain non-existent dataset", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		assert.False(t, coll.Contains("1.2.3.999"))
	})
}

// TestDataSetCollection_Len tests counting datasets
func TestDataSetCollection_Len(t *testing.T) {
	t.Run("empty collection", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()
		assert.Equal(t, 0, coll.Len())
	})

	t.Run("after adds", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		ds1 := createTestDataSetForCollection("1.2.3.1", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1)
		ds2 := createTestDataSetForCollection("1.2.3.2", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 2)

		require.NoError(t, coll.Add(ds1))
		assert.Equal(t, 1, coll.Len())

		require.NoError(t, coll.Add(ds2))
		assert.Equal(t, 2, coll.Len())
	})

	t.Run("after remove", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		ds := createTestDataSetForCollection("1.2.3.1", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1)
		require.NoError(t, coll.Add(ds))
		assert.Equal(t, 1, coll.Len())

		require.NoError(t, coll.Remove("1.2.3.1"))
		assert.Equal(t, 0, coll.Len())
	})
}

// TestDataSetCollection_DataSets tests getting all datasets
func TestDataSetCollection_DataSets(t *testing.T) {
	t.Run("empty collection", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()
		datasets := coll.DataSets()
		assert.Empty(t, datasets)
	})

	t.Run("returns all datasets", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		ds1 := createTestDataSetForCollection("1.2.3.1", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 1)
		ds2 := createTestDataSetForCollection("1.2.3.2", "1.2.3.100", "1.2.3.1000", "P001", "A001", "1.2.840.10008.5.1.4.1.1.2", 2)

		require.NoError(t, coll.Add(ds1))
		require.NoError(t, coll.Add(ds2))

		datasets := coll.DataSets()
		assert.Len(t, datasets, 2)
	})
}

// TestDataSetCollection_ThreadSafety tests concurrent access
func TestDataSetCollection_ThreadSafety(t *testing.T) {
	t.Run("concurrent adds", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		var wg sync.WaitGroup
		numGoroutines := 10
		datasetsPerGoroutine := 10

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(offset int) {
				defer wg.Done()
				for j := 0; j < datasetsPerGoroutine; j++ {
					sopUID := fmt.Sprintf("1.2.3.%d", offset*datasetsPerGoroutine+j)
					ds := createTestDataSetForCollection(
						sopUID,
						"1.2.3.100",
						"1.2.3.1000",
						"P001",
						"A001",
						"1.2.840.10008.5.1.4.1.1.2",
						j+1,
					)
					_ = coll.Add(ds)
				}
			}(i)
		}

		wg.Wait()
		assert.Equal(t, numGoroutines*datasetsPerGoroutine, coll.Len())
	})

	t.Run("concurrent reads and writes", func(t *testing.T) {
		coll := dicom.NewDataSetCollection()

		// Pre-populate
		for i := 0; i < 50; i++ {
			ds := createTestDataSetForCollection(
				fmt.Sprintf("1.2.3.%d", i),
				"1.2.3.100",
				"1.2.3.1000",
				"P001",
				"A001",
				"1.2.840.10008.5.1.4.1.1.2",
				i%10+1,
			)
			require.NoError(t, coll.Add(ds))
		}

		var wg sync.WaitGroup

		// Readers
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					_ = coll.GetBySeriesInstanceUID("1.2.3.100")
					_ = coll.GetByStudyInstanceUID("1.2.3.1000")
				}
			}()
		}

		// Writers
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(offset int) {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					sopUID := fmt.Sprintf("1.2.3.%d", 100+offset*10+j)
					ds := createTestDataSetForCollection(
						sopUID,
						"1.2.3.100",
						"1.2.3.1000",
						"P001",
						"A001",
						"1.2.840.10008.5.1.4.1.1.2",
						j+1,
					)
					_ = coll.Add(ds)
				}
			}(i)
		}

		wg.Wait()
		assert.GreaterOrEqual(t, coll.Len(), 50) // At least initial datasets
	})
}
