package dicom_test

import (
	"testing"

	dicom "github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicom/element"
	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/value"
	"github.com/codeninja55/go-radx/dicom/vr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper functions for creating test elements
func mustNewStringValue(v vr.VR, values []string) *value.StringValue {
	val, err := value.NewStringValue(v, values)
	if err != nil {
		panic(err)
	}
	return val
}

func mustNewElement(t tag.Tag, v vr.VR, val value.Value) *element.Element {
	elem, err := element.NewElement(t, v, val)
	if err != nil {
		panic(err)
	}
	return elem
}

// TestDataSet_NewDataSet tests creating a new empty dataset
func TestDataSet_NewDataSet(t *testing.T) {
	t.Run("empty dataset", func(t *testing.T) {
		ds := dicom.NewDataSet()
		assert.NotNil(t, ds)
		assert.Equal(t, 0, ds.Len())
	})
}

// TestDataSet_NewDataSetWithElements tests creating a dataset with initial elements
func TestDataSet_NewDataSetWithElements(t *testing.T) {
	t.Run("valid elements", func(t *testing.T) {
		elements := []*element.Element{
			mustNewElement(tag.New(0x0010, 0x0010), vr.PersonName,
				mustNewStringValue(vr.PersonName, []string{"Doe^John"})),
			mustNewElement(tag.New(0x0010, 0x0020), vr.LongString,
				mustNewStringValue(vr.LongString, []string{"12345"})),
		}

		ds, err := dicom.NewDataSetWithElements(elements)
		require.NoError(t, err)
		assert.NotNil(t, ds)
		assert.Equal(t, 2, ds.Len())
	})

	t.Run("nil elements slice", func(t *testing.T) {
		ds, err := dicom.NewDataSetWithElements(nil)
		require.NoError(t, err)
		assert.NotNil(t, ds)
		assert.Equal(t, 0, ds.Len())
	})

	t.Run("duplicate tags should error", func(t *testing.T) {
		elements := []*element.Element{
			mustNewElement(tag.New(0x0010, 0x0010), vr.PersonName,
				mustNewStringValue(vr.PersonName, []string{"Doe^John"})),
			mustNewElement(tag.New(0x0010, 0x0010), vr.PersonName,
				mustNewStringValue(vr.PersonName, []string{"Smith^Jane"})),
		}

		ds, err := dicom.NewDataSetWithElements(elements)
		assert.Error(t, err)
		assert.Nil(t, ds)
		assert.Contains(t, err.Error(), "duplicate")
	})
}

// TestDataSet_Add tests adding elements to a dataset
func TestDataSet_Add(t *testing.T) {
	t.Run("add single element", func(t *testing.T) {
		ds := dicom.NewDataSet()
		elem := mustNewElement(tag.New(0x0010, 0x0010), vr.PersonName,
			mustNewStringValue(vr.PersonName, []string{"Doe^John"}))

		err := ds.Add(elem)
		assert.NoError(t, err)
		assert.Equal(t, 1, ds.Len())
	})

	t.Run("add multiple elements", func(t *testing.T) {
		ds := dicom.NewDataSet()

		elem1 := mustNewElement(tag.New(0x0010, 0x0010), vr.PersonName,
			mustNewStringValue(vr.PersonName, []string{"Doe^John"}))
		elem2 := mustNewElement(tag.New(0x0010, 0x0020), vr.LongString,
			mustNewStringValue(vr.LongString, []string{"12345"}))

		require.NoError(t, ds.Add(elem1))
		require.NoError(t, ds.Add(elem2))
		assert.Equal(t, 2, ds.Len())
	})

	t.Run("add nil element should error", func(t *testing.T) {
		ds := dicom.NewDataSet()
		err := ds.Add(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("add duplicate tag replaces", func(t *testing.T) {
		ds := dicom.NewDataSet()

		elem1 := mustNewElement(tag.New(0x0010, 0x0010), vr.PersonName,
			mustNewStringValue(vr.PersonName, []string{"Doe^John"}))
		elem2 := mustNewElement(tag.New(0x0010, 0x0010), vr.PersonName,
			mustNewStringValue(vr.PersonName, []string{"Smith^Jane"}))

		require.NoError(t, ds.Add(elem1))
		require.NoError(t, ds.Add(elem2))

		assert.Equal(t, 1, ds.Len()) // Should still be 1

		retrieved, err := ds.Get(tag.New(0x0010, 0x0010))
		require.NoError(t, err)
		assert.Equal(t, "Smith^Jane", retrieved.Value().String())
	})
}

// TestDataSet_Get tests retrieving elements by tag
func TestDataSet_Get(t *testing.T) {
	t.Run("get existing element", func(t *testing.T) {
		ds := dicom.NewDataSet()
		elem := mustNewElement(tag.New(0x0010, 0x0010), vr.PersonName,
			mustNewStringValue(vr.PersonName, []string{"Doe^John"}))

		require.NoError(t, ds.Add(elem))

		retrieved, err := ds.Get(tag.New(0x0010, 0x0010))
		require.NoError(t, err)
		assert.Equal(t, "Doe^John", retrieved.Value().String())
	})

	t.Run("get non-existent element", func(t *testing.T) {
		ds := dicom.NewDataSet()

		retrieved, err := ds.Get(tag.New(0x0010, 0x0010))
		assert.Error(t, err)
		assert.Nil(t, retrieved)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("get from empty dataset", func(t *testing.T) {
		ds := dicom.NewDataSet()

		retrieved, err := ds.Get(tag.New(0x0010, 0x0010))
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})
}

// TestDataSet_GetByKeyword tests retrieving elements by DICOM keyword
func TestDataSet_GetByKeyword(t *testing.T) {
	t.Run("get by valid keyword", func(t *testing.T) {
		ds := dicom.NewDataSet()
		elem := mustNewElement(tag.New(0x0010, 0x0010), vr.PersonName,
			mustNewStringValue(vr.PersonName, []string{"Doe^John"}))

		require.NoError(t, ds.Add(elem))

		retrieved, err := ds.GetByKeyword("PatientName")
		require.NoError(t, err)
		assert.Equal(t, "Doe^John", retrieved.Value().String())
	})

	t.Run("get by unknown keyword", func(t *testing.T) {
		ds := dicom.NewDataSet()

		retrieved, err := ds.GetByKeyword("UnknownKeyword")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("get by keyword not in dataset", func(t *testing.T) {
		ds := dicom.NewDataSet()

		retrieved, err := ds.GetByKeyword("PatientName")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})
}

// TestDataSet_Contains tests checking for element existence
func TestDataSet_Contains(t *testing.T) {
	t.Run("contains existing tag", func(t *testing.T) {
		ds := dicom.NewDataSet()
		elem := mustNewElement(tag.New(0x0010, 0x0010), vr.PersonName,
			mustNewStringValue(vr.PersonName, []string{"Doe^John"}))

		require.NoError(t, ds.Add(elem))

		assert.True(t, ds.Contains(tag.New(0x0010, 0x0010)))
	})

	t.Run("does not contain non-existent tag", func(t *testing.T) {
		ds := dicom.NewDataSet()

		assert.False(t, ds.Contains(tag.New(0x0010, 0x0010)))
	})
}

// TestDataSet_Remove tests removing elements from a dataset
func TestDataSet_Remove(t *testing.T) {
	t.Run("remove existing element", func(t *testing.T) {
		ds := dicom.NewDataSet()
		elem := mustNewElement(tag.New(0x0010, 0x0010), vr.PersonName,
			mustNewStringValue(vr.PersonName, []string{"Doe^John"}))

		require.NoError(t, ds.Add(elem))
		assert.Equal(t, 1, ds.Len())

		err := ds.Remove(tag.New(0x0010, 0x0010))
		assert.NoError(t, err)
		assert.Equal(t, 0, ds.Len())
		assert.False(t, ds.Contains(tag.New(0x0010, 0x0010)))
	})

	t.Run("remove non-existent element", func(t *testing.T) {
		ds := dicom.NewDataSet()

		err := ds.Remove(tag.New(0x0010, 0x0010))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("remove from empty dataset", func(t *testing.T) {
		ds := dicom.NewDataSet()

		err := ds.Remove(tag.New(0x0010, 0x0010))
		assert.Error(t, err)
	})
}

// TestDataSet_Len tests counting elements
func TestDataSet_Len(t *testing.T) {
	t.Run("empty dataset", func(t *testing.T) {
		ds := dicom.NewDataSet()
		assert.Equal(t, 0, ds.Len())
	})

	t.Run("after adds", func(t *testing.T) {
		ds := dicom.NewDataSet()

		elem1 := mustNewElement(tag.New(0x0010, 0x0010), vr.PersonName,
			mustNewStringValue(vr.PersonName, []string{"Doe^John"}))
		elem2 := mustNewElement(tag.New(0x0010, 0x0020), vr.LongString,
			mustNewStringValue(vr.LongString, []string{"12345"}))

		require.NoError(t, ds.Add(elem1))
		assert.Equal(t, 1, ds.Len())

		require.NoError(t, ds.Add(elem2))
		assert.Equal(t, 2, ds.Len())
	})

	t.Run("after remove", func(t *testing.T) {
		ds := dicom.NewDataSet()

		elem := mustNewElement(tag.New(0x0010, 0x0010), vr.PersonName,
			mustNewStringValue(vr.PersonName, []string{"Doe^John"}))

		require.NoError(t, ds.Add(elem))
		assert.Equal(t, 1, ds.Len())

		require.NoError(t, ds.Remove(tag.New(0x0010, 0x0010)))
		assert.Equal(t, 0, ds.Len())
	})
}

// TestDataSet_Elements tests iteration over elements
func TestDataSet_Elements(t *testing.T) {
	t.Run("empty dataset", func(t *testing.T) {
		ds := dicom.NewDataSet()
		elements := ds.Elements()
		assert.Empty(t, elements)
	})

	t.Run("sorted by tag", func(t *testing.T) {
		ds := dicom.NewDataSet()

		// Add in non-sorted order
		elem1 := mustNewElement(tag.New(0x0020, 0x000D), vr.UniqueIdentifier,
			mustNewStringValue(vr.UniqueIdentifier, []string{"1.2.3"}))
		elem2 := mustNewElement(tag.New(0x0010, 0x0010), vr.PersonName,
			mustNewStringValue(vr.PersonName, []string{"Doe^John"}))
		elem3 := mustNewElement(tag.New(0x0010, 0x0020), vr.LongString,
			mustNewStringValue(vr.LongString, []string{"12345"}))

		require.NoError(t, ds.Add(elem1))
		require.NoError(t, ds.Add(elem2))
		require.NoError(t, ds.Add(elem3))

		elements := ds.Elements()
		require.Len(t, elements, 3)

		// Should be sorted by tag
		assert.Equal(t, tag.New(0x0010, 0x0010), elements[0].Tag())
		assert.Equal(t, tag.New(0x0010, 0x0020), elements[1].Tag())
		assert.Equal(t, tag.New(0x0020, 0x000D), elements[2].Tag())
	})
}

// TestDataSet_Tags tests getting all tags
func TestDataSet_Tags(t *testing.T) {
	t.Run("empty dataset", func(t *testing.T) {
		ds := dicom.NewDataSet()
		tags := ds.Tags()
		assert.Empty(t, tags)
	})

	t.Run("sorted tags", func(t *testing.T) {
		ds := dicom.NewDataSet()

		elem1 := mustNewElement(tag.New(0x0020, 0x000D), vr.UniqueIdentifier,
			mustNewStringValue(vr.UniqueIdentifier, []string{"1.2.3"}))
		elem2 := mustNewElement(tag.New(0x0010, 0x0010), vr.PersonName,
			mustNewStringValue(vr.PersonName, []string{"Doe^John"}))

		require.NoError(t, ds.Add(elem1))
		require.NoError(t, ds.Add(elem2))

		tags := ds.Tags()
		require.Len(t, tags, 2)

		// Should be sorted
		assert.Equal(t, tag.New(0x0010, 0x0010), tags[0])
		assert.Equal(t, tag.New(0x0020, 0x000D), tags[1])
	})
}

// TestDataSet_String tests string representation
func TestDataSet_String(t *testing.T) {
	t.Run("empty dataset", func(t *testing.T) {
		ds := dicom.NewDataSet()
		str := ds.String()
		assert.Contains(t, str, "DataSet")
		assert.Contains(t, str, "0 elements")
	})

	t.Run("single element", func(t *testing.T) {
		ds := dicom.NewDataSet()
		elem := mustNewElement(tag.New(0x0010, 0x0010), vr.PersonName,
			mustNewStringValue(vr.PersonName, []string{"Doe^John"}))

		require.NoError(t, ds.Add(elem))

		str := ds.String()
		assert.Contains(t, str, "1 element")
		assert.Contains(t, str, "0010,0010")
		assert.Contains(t, str, "Doe^John")
	})

	t.Run("multiple elements", func(t *testing.T) {
		ds := dicom.NewDataSet()

		elem1 := mustNewElement(tag.New(0x0010, 0x0010), vr.PersonName,
			mustNewStringValue(vr.PersonName, []string{"Doe^John"}))
		elem2 := mustNewElement(tag.New(0x0010, 0x0020), vr.LongString,
			mustNewStringValue(vr.LongString, []string{"12345"}))

		require.NoError(t, ds.Add(elem1))
		require.NoError(t, ds.Add(elem2))

		str := ds.String()
		assert.Contains(t, str, "2 elements")
	})
}

// TestDataSet_Copy tests copying a dataset
func TestDataSet_Copy(t *testing.T) {
	t.Run("copy empty dataset", func(t *testing.T) {
		ds := dicom.NewDataSet()
		copied := ds.Copy()

		assert.NotNil(t, copied)
		assert.Equal(t, 0, copied.Len())
		assert.NotSame(t, ds, copied) // Different instances
	})

	t.Run("copy with elements", func(t *testing.T) {
		ds := dicom.NewDataSet()
		elem := mustNewElement(tag.New(0x0010, 0x0010), vr.PersonName,
			mustNewStringValue(vr.PersonName, []string{"Doe^John"}))

		require.NoError(t, ds.Add(elem))

		copied := ds.Copy()
		assert.Equal(t, 1, copied.Len())

		// Modify copied should not affect original
		require.NoError(t, copied.Remove(tag.New(0x0010, 0x0010)))
		assert.Equal(t, 0, copied.Len())
		assert.Equal(t, 1, ds.Len())
	})
}

// TestDataSet_Merge tests merging two datasets
func TestDataSet_Merge(t *testing.T) {
	t.Run("merge into empty", func(t *testing.T) {
		ds1 := dicom.NewDataSet()
		ds2 := dicom.NewDataSet()

		elem := mustNewElement(tag.New(0x0010, 0x0010), vr.PersonName,
			mustNewStringValue(vr.PersonName, []string{"Doe^John"}))
		require.NoError(t, ds2.Add(elem))

		err := ds1.Merge(ds2)
		assert.NoError(t, err)
		assert.Equal(t, 1, ds1.Len())
	})

	t.Run("merge non-overlapping", func(t *testing.T) {
		ds1 := dicom.NewDataSet()
		ds2 := dicom.NewDataSet()

		elem1 := mustNewElement(tag.New(0x0010, 0x0010), vr.PersonName,
			mustNewStringValue(vr.PersonName, []string{"Doe^John"}))
		elem2 := mustNewElement(tag.New(0x0010, 0x0020), vr.LongString,
			mustNewStringValue(vr.LongString, []string{"12345"}))

		require.NoError(t, ds1.Add(elem1))
		require.NoError(t, ds2.Add(elem2))

		err := ds1.Merge(ds2)
		assert.NoError(t, err)
		assert.Equal(t, 2, ds1.Len())
	})

	t.Run("merge with overlap replaces", func(t *testing.T) {
		ds1 := dicom.NewDataSet()
		ds2 := dicom.NewDataSet()

		elem1 := mustNewElement(tag.New(0x0010, 0x0010), vr.PersonName,
			mustNewStringValue(vr.PersonName, []string{"Doe^John"}))
		elem2 := mustNewElement(tag.New(0x0010, 0x0010), vr.PersonName,
			mustNewStringValue(vr.PersonName, []string{"Smith^Jane"}))

		require.NoError(t, ds1.Add(elem1))
		require.NoError(t, ds2.Add(elem2))

		err := ds1.Merge(ds2)
		assert.NoError(t, err)
		assert.Equal(t, 1, ds1.Len())

		retrieved, err := ds1.Get(tag.New(0x0010, 0x0010))
		require.NoError(t, err)
		assert.Equal(t, "Smith^Jane", retrieved.Value().String())
	})
}
