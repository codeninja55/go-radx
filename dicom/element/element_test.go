package element_test

import (
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/dicom/element"
	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/value"
	"github.com/codeninja55/go-radx/dicom/vr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestElement_NewElement tests element creation with various value types
func TestElement_NewElement(t *testing.T) {
	tests := []struct {
		name    string
		tag     tag.Tag
		vr      vr.VR
		value   value.Value
		wantErr bool
	}{
		{
			name:    "PatientName with PN StringValue",
			tag:     tag.New(0x0010, 0x0010),
			vr:      vr.PersonName,
			value:   mustNewStringValue(vr.PersonName, []string{"Doe^John"}),
			wantErr: false,
		},
		{
			name:    "nil value should error",
			tag:     tag.New(0x0010, 0x0010),
			vr:      vr.PersonName,
			value:   nil,
			wantErr: true,
		},
		{
			name:    "VR mismatch should error",
			tag:     tag.New(0x0010, 0x0010),
			vr:      vr.PersonName,
			value:   mustNewStringValue(vr.LongString, []string{"test"}), // LO != PN
			wantErr: true,
		},
		{
			name:    "PatientID with LO StringValue",
			tag:     tag.New(0x0010, 0x0020),
			vr:      vr.LongString,
			value:   mustNewStringValue(vr.LongString, []string{"12345"}),
			wantErr: false,
		},
		{
			name:    "StudyDate with DA StringValue",
			tag:     tag.New(0x0008, 0x0020),
			vr:      vr.Date,
			value:   mustNewStringValue(vr.Date, []string{"20250109"}),
			wantErr: false,
		},
		{
			name:    "Rows with US IntValue",
			tag:     tag.New(0x0028, 0x0010),
			vr:      vr.UnsignedShort,
			value:   mustNewIntValue(vr.UnsignedShort, []int64{512}),
			wantErr: false,
		},
		{
			name:    "PixelData with OW BytesValue",
			tag:     tag.New(0x7FE0, 0x0010),
			vr:      vr.OtherWord,
			value:   mustNewBytesValue(vr.OtherWord, []byte{0x00, 0x01, 0x02, 0x03}),
			wantErr: false,
		},
		{
			name:    "ImagePositionPatient with DS (as FloatValue)",
			tag:     tag.New(0x0020, 0x0032),
			vr:      vr.FloatingPointDouble,
			value:   mustNewFloatValue(vr.FloatingPointDouble, []float64{100.0, 200.0, 50.0}),
			wantErr: false,
		},
		{
			name:    "empty StringValue",
			tag:     tag.New(0x0010, 0x0010),
			vr:      vr.PersonName,
			value:   mustNewStringValue(vr.PersonName, []string{}),
			wantErr: false,
		},
		{
			name:    "private tag (odd group)",
			tag:     tag.New(0x0009, 0x0010),
			vr:      vr.LongString,
			value:   mustNewStringValue(vr.LongString, []string{"Private Data"}),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elem, err := element.NewElement(tt.tag, tt.vr, tt.value)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, elem)
			} else {
				require.NoError(t, err)
				require.NotNil(t, elem)
				assert.Equal(t, tt.tag, elem.Tag())
				assert.Equal(t, tt.vr, elem.VR())
				assert.Equal(t, tt.value, elem.Value())
			}
		})
	}
}

// TestElement_AccessorMethods tests Tag(), VR(), and Value() accessors
func TestElement_AccessorMethods(t *testing.T) {
	testTag := tag.New(0x0010, 0x0010)
	testVR := vr.PersonName
	testValue := mustNewStringValue(testVR, []string{"Doe^John"})

	elem, err := element.NewElement(testTag, testVR, testValue)
	require.NoError(t, err)

	assert.Equal(t, testTag, elem.Tag(), "Tag() should return the element's tag")
	assert.Equal(t, testVR, elem.VR(), "VR() should return the element's VR")
	assert.Equal(t, testValue, elem.Value(), "Value() should return the element's value")
}

// TestElement_Name tests dictionary lookup for element names
func TestElement_Name(t *testing.T) {
	tests := []struct {
		name     string
		tag      tag.Tag
		vr       vr.VR
		value    value.Value
		wantName string
	}{
		{
			name:     "PatientName",
			tag:      tag.New(0x0010, 0x0010),
			vr:       vr.PersonName,
			value:    mustNewStringValue(vr.PersonName, []string{"Doe^John"}),
			wantName: "Patient's Name",
		},
		{
			name:     "PatientID",
			tag:      tag.New(0x0010, 0x0020),
			vr:       vr.LongString,
			value:    mustNewStringValue(vr.LongString, []string{"12345"}),
			wantName: "Patient ID",
		},
		{
			name:     "StudyDate",
			tag:      tag.New(0x0008, 0x0020),
			vr:       vr.Date,
			value:    mustNewStringValue(vr.Date, []string{"20250109"}),
			wantName: "Study Date",
		},
		{
			name:     "PixelData",
			tag:      tag.New(0x7FE0, 0x0010),
			vr:       vr.OtherWord,
			value:    mustNewBytesValue(vr.OtherWord, []byte{0x00, 0x01}),
			wantName: "Pixel Data",
		},
		{
			name:     "private tag (unknown in dictionary)",
			tag:      tag.New(0x0009, 0x0010),
			vr:       vr.LongString,
			value:    mustNewStringValue(vr.LongString, []string{"Private"}),
			wantName: "", // Private tags don't have standard names
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elem, err := element.NewElement(tt.tag, tt.vr, tt.value)
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, elem.Name())
		})
	}
}

// TestElement_Keyword tests dictionary lookup for element keywords
func TestElement_Keyword(t *testing.T) {
	tests := []struct {
		name        string
		tag         tag.Tag
		vr          vr.VR
		value       value.Value
		wantKeyword string
	}{
		{
			name:        "PatientName",
			tag:         tag.New(0x0010, 0x0010),
			vr:          vr.PersonName,
			value:       mustNewStringValue(vr.PersonName, []string{"Doe^John"}),
			wantKeyword: "PatientName",
		},
		{
			name:        "PatientID",
			tag:         tag.New(0x0010, 0x0020),
			vr:          vr.LongString,
			value:       mustNewStringValue(vr.LongString, []string{"12345"}),
			wantKeyword: "PatientID",
		},
		{
			name:        "StudyInstanceUID",
			tag:         tag.New(0x0020, 0x000D),
			vr:          vr.UniqueIdentifier,
			value:       mustNewStringValue(vr.UniqueIdentifier, []string{"1.2.3.4"}),
			wantKeyword: "StudyInstanceUID",
		},
		{
			name:        "private tag (no keyword)",
			tag:         tag.New(0x0009, 0x0010),
			vr:          vr.LongString,
			value:       mustNewStringValue(vr.LongString, []string{"Private"}),
			wantKeyword: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elem, err := element.NewElement(tt.tag, tt.vr, tt.value)
			require.NoError(t, err)
			assert.Equal(t, tt.wantKeyword, elem.Keyword())
		})
	}
}

// TestElement_VM tests Value Multiplicity reporting
func TestElement_VM(t *testing.T) {
	tests := []struct {
		name   string
		tag    tag.Tag
		vr     vr.VR
		value  value.Value
		wantVM string
	}{
		{
			name:   "single value string",
			tag:    tag.New(0x0010, 0x0010),
			vr:     vr.PersonName,
			value:  mustNewStringValue(vr.PersonName, []string{"Doe^John"}),
			wantVM: "1",
		},
		{
			name:   "multi-value (3) floats",
			tag:    tag.New(0x0020, 0x0032),
			vr:     vr.FloatingPointDouble,
			value:  mustNewFloatValue(vr.FloatingPointDouble, []float64{100.0, 200.0, 50.0}),
			wantVM: "3",
		},
		{
			name:   "multi-value (2) ints",
			tag:    tag.New(0x0028, 0x0030),
			vr:     vr.UnsignedShort,
			value:  mustNewIntValue(vr.UnsignedShort, []int64{1, 1}),
			wantVM: "2",
		},
		{
			name:   "empty value",
			tag:    tag.New(0x0010, 0x0010),
			vr:     vr.PersonName,
			value:  mustNewStringValue(vr.PersonName, []string{}),
			wantVM: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elem, err := element.NewElement(tt.tag, tt.vr, tt.value)
			require.NoError(t, err)
			assert.Equal(t, tt.wantVM, elem.ValueMultiplicity())
		})
	}
}

// TestElement_String tests human-readable string representation
func TestElement_String(t *testing.T) {
	tests := []struct {
		name         string
		tag          tag.Tag
		vr           vr.VR
		value        value.Value
		wantPrefix   string   // Check if string starts with this
		wantContains []string // Check if string contains these substrings
	}{
		{
			name:         "PatientName",
			tag:          tag.New(0x0010, 0x0010),
			vr:           vr.PersonName,
			value:        mustNewStringValue(vr.PersonName, []string{"Doe^John"}),
			wantPrefix:   "(0010,0010)",
			wantContains: []string{"PN", "Patient's Name", "Doe^John"},
		},
		{
			name:         "PatientID",
			tag:          tag.New(0x0010, 0x0020),
			vr:           vr.LongString,
			value:        mustNewStringValue(vr.LongString, []string{"12345"}),
			wantPrefix:   "(0010,0020)",
			wantContains: []string{"LO", "Patient ID", "12345"},
		},
		{
			name:         "Rows (numeric)",
			tag:          tag.New(0x0028, 0x0010),
			vr:           vr.UnsignedShort,
			value:        mustNewIntValue(vr.UnsignedShort, []int64{512}),
			wantPrefix:   "(0028,0010)",
			wantContains: []string{"US", "Rows", "512"},
		},
		{
			name:         "empty value",
			tag:          tag.New(0x0010, 0x0010),
			vr:           vr.PersonName,
			value:        mustNewStringValue(vr.PersonName, []string{}),
			wantPrefix:   "(0010,0010)",
			wantContains: []string{"PN", "Patient's Name"},
		},
		{
			name:         "private tag",
			tag:          tag.New(0x0009, 0x0010),
			vr:           vr.LongString,
			value:        mustNewStringValue(vr.LongString, []string{"Private"}),
			wantPrefix:   "(0009,0010)",
			wantContains: []string{"LO", "Private"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elem, err := element.NewElement(tt.tag, tt.vr, tt.value)
			require.NoError(t, err)

			str := elem.String()
			assert.True(t, len(str) > 0, "String() should not be empty")

			if tt.wantPrefix != "" {
				assert.Contains(t, str, tt.wantPrefix, "String should contain tag")
			}

			for _, substr := range tt.wantContains {
				assert.Contains(t, str, substr, "String should contain: %s", substr)
			}
		})
	}

	// Test long value truncation
	t.Run("long value truncation", func(t *testing.T) {
		// Create a very long string (>80 chars) using UT (Unlimited Text) which has no max length
		longString := strings.Repeat("A", 100)
		elem, err := element.NewElement(
			tag.New(0x0008, 0x0080), // Institution Name, but we'll use UT for unlimited length
			vr.UnlimitedText,
			mustNewStringValue(vr.UnlimitedText, []string{longString}),
		)
		require.NoError(t, err)

		str := elem.String()
		// Should be truncated with "..."
		assert.Contains(t, str, "...", "Long values should be truncated")
		assert.Less(t, len(str), len(longString)+50, "String should be shorter than original")
	})
}

// TestElement_Equals tests equality comparison
func TestElement_Equals(t *testing.T) {
	tag1 := tag.New(0x0010, 0x0010)
	tag2 := tag.New(0x0010, 0x0020)

	value1 := mustNewStringValue(vr.PersonName, []string{"Doe^John"})
	value2 := mustNewStringValue(vr.PersonName, []string{"Smith^Jane"})
	value3 := mustNewStringValue(vr.LongString, []string{"12345"})

	tests := []struct {
		name      string
		elem1Tag  tag.Tag
		elem1VR   vr.VR
		elem1Val  value.Value
		elem2Tag  tag.Tag
		elem2VR   vr.VR
		elem2Val  value.Value
		wantEqual bool
	}{
		{
			name:      "identical elements",
			elem1Tag:  tag1,
			elem1VR:   vr.PersonName,
			elem1Val:  value1,
			elem2Tag:  tag1,
			elem2VR:   vr.PersonName,
			elem2Val:  value1,
			wantEqual: true,
		},
		{
			name:      "same tag and VR, equal values",
			elem1Tag:  tag1,
			elem1VR:   vr.PersonName,
			elem1Val:  mustNewStringValue(vr.PersonName, []string{"Doe^John"}),
			elem2Tag:  tag1,
			elem2VR:   vr.PersonName,
			elem2Val:  mustNewStringValue(vr.PersonName, []string{"Doe^John"}),
			wantEqual: true,
		},
		{
			name:      "different tags",
			elem1Tag:  tag1,
			elem1VR:   vr.PersonName,
			elem1Val:  value1,
			elem2Tag:  tag2,
			elem2VR:   vr.PersonName,
			elem2Val:  value1,
			wantEqual: false,
		},
		{
			name:      "different VRs",
			elem1Tag:  tag1,
			elem1VR:   vr.PersonName,
			elem1Val:  value1,
			elem2Tag:  tag1,
			elem2VR:   vr.LongString,
			elem2Val:  value3,
			wantEqual: false,
		},
		{
			name:      "different values",
			elem1Tag:  tag1,
			elem1VR:   vr.PersonName,
			elem1Val:  value1,
			elem2Tag:  tag1,
			elem2VR:   vr.PersonName,
			elem2Val:  value2,
			wantEqual: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elem1, err := element.NewElement(tt.elem1Tag, tt.elem1VR, tt.elem1Val)
			require.NoError(t, err)

			elem2, err := element.NewElement(tt.elem2Tag, tt.elem2VR, tt.elem2Val)
			require.NoError(t, err)

			assert.Equal(t, tt.wantEqual, elem1.Equals(elem2))
		})
	}

	// Test nil comparison separately
	t.Run("nil comparison", func(t *testing.T) {
		elem, err := element.NewElement(tag1, vr.PersonName, value1)
		require.NoError(t, err)
		assert.False(t, elem.Equals(nil), "Element should not equal nil")
	})
}

// TestElement_StandardTags tests creation with well-known DICOM tags
func TestElement_StandardTags(t *testing.T) {
	tests := []struct {
		name    string
		tag     tag.Tag
		vr      vr.VR
		value   value.Value
		wantErr bool
	}{
		{
			name:    "SOPClassUID",
			tag:     tag.New(0x0008, 0x0016),
			vr:      vr.UniqueIdentifier,
			value:   mustNewStringValue(vr.UniqueIdentifier, []string{"1.2.840.10008.5.1.4.1.1.2"}),
			wantErr: false,
		},
		{
			name:    "SOPInstanceUID",
			tag:     tag.New(0x0008, 0x0018),
			vr:      vr.UniqueIdentifier,
			value:   mustNewStringValue(vr.UniqueIdentifier, []string{"1.2.3.4.5"}),
			wantErr: false,
		},
		{
			name:    "StudyInstanceUID",
			tag:     tag.New(0x0020, 0x000D),
			vr:      vr.UniqueIdentifier,
			value:   mustNewStringValue(vr.UniqueIdentifier, []string{"1.2.3"}),
			wantErr: false,
		},
		{
			name:    "SeriesInstanceUID",
			tag:     tag.New(0x0020, 0x000E),
			vr:      vr.UniqueIdentifier,
			value:   mustNewStringValue(vr.UniqueIdentifier, []string{"1.2.3.4"}),
			wantErr: false,
		},
		{
			name:    "Modality",
			tag:     tag.New(0x0008, 0x0060),
			vr:      vr.CodeString,
			value:   mustNewStringValue(vr.CodeString, []string{"CT"}),
			wantErr: false,
		},
		{
			name:    "Manufacturer",
			tag:     tag.New(0x0008, 0x0070),
			vr:      vr.LongString,
			value:   mustNewStringValue(vr.LongString, []string{"ACME Corp"}),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elem, err := element.NewElement(tt.tag, tt.vr, tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, elem)
			}
		})
	}
}

// Helper functions to create values for tests
func mustNewStringValue(v vr.VR, values []string) *value.StringValue {
	val, err := value.NewStringValue(v, values)
	if err != nil {
		panic(err)
	}
	return val
}

func mustNewBytesValue(v vr.VR, data []byte) *value.BytesValue {
	val, err := value.NewBytesValue(v, data)
	if err != nil {
		panic(err)
	}
	return val
}

func mustNewIntValue(v vr.VR, values []int64) *value.IntValue {
	val, err := value.NewIntValue(v, values)
	if err != nil {
		panic(err)
	}
	return val
}

func mustNewFloatValue(v vr.VR, values []float64) *value.FloatValue {
	val, err := value.NewFloatValue(v, values)
	if err != nil {
		panic(err)
	}
	return val
}
