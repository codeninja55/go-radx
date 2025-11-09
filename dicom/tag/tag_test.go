package tag_test

import (
	"testing"

	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTag_NewTag(t *testing.T) {
	tests := []struct {
		name    string
		group   uint16
		element uint16
	}{
		{
			name:    "PatientName tag",
			group:   0x0010,
			element: 0x0010,
		},
		{
			name:    "StudyInstanceUID tag",
			group:   0x0020,
			element: 0x000D,
		},
		{
			name:    "PixelData tag",
			group:   0x7FE0,
			element: 0x0010,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tag := tag.New(tc.group, tc.element)
			assert.Equal(t, tc.group, tag.Group)
			assert.Equal(t, tc.element, tag.Element)
		})
	}
}

func TestTag_Equals(t *testing.T) {
	tests := []struct {
		name     string
		tag1     tag.Tag
		tag2     tag.Tag
		expected bool
	}{
		{
			name:     "equal tags",
			tag1:     tag.New(0x0008, 0x0020),
			tag2:     tag.New(0x0008, 0x0020),
			expected: true,
		},
		{
			name:     "different group",
			tag1:     tag.New(0x0008, 0x0020),
			tag2:     tag.New(0x0010, 0x0020),
			expected: false,
		},
		{
			name:     "different element",
			tag1:     tag.New(0x0008, 0x0020),
			tag2:     tag.New(0x0008, 0x0030),
			expected: false,
		},
		{
			name:     "both different",
			tag1:     tag.New(0x0008, 0x0020),
			tag2:     tag.New(0x0010, 0x0010),
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.tag1.Equals(tc.tag2)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTag_Compare(t *testing.T) {
	tests := []struct {
		name     string
		tag1     tag.Tag
		tag2     tag.Tag
		expected int
	}{
		{
			name:     "equal tags",
			tag1:     tag.New(0x0008, 0x0020),
			tag2:     tag.New(0x0008, 0x0020),
			expected: 0,
		},
		{
			name:     "tag1 less than tag2 by group",
			tag1:     tag.New(0x0008, 0x0020),
			tag2:     tag.New(0x0010, 0x0020),
			expected: -1,
		},
		{
			name:     "tag1 greater than tag2 by group",
			tag1:     tag.New(0x0010, 0x0020),
			tag2:     tag.New(0x0008, 0x0020),
			expected: 1,
		},
		{
			name:     "tag1 less than tag2 by element",
			tag1:     tag.New(0x0008, 0x0020),
			tag2:     tag.New(0x0008, 0x0030),
			expected: -1,
		},
		{
			name:     "tag1 greater than tag2 by element",
			tag1:     tag.New(0x0008, 0x0030),
			tag2:     tag.New(0x0008, 0x0020),
			expected: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.tag1.Compare(tc.tag2)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTag_String(t *testing.T) {
	tests := []struct {
		name     string
		tag      tag.Tag
		expected string
	}{
		{
			name:     "standard tag format",
			tag:      tag.New(0x0008, 0x0020),
			expected: "(0008,0020)",
		},
		{
			name:     "private tag format",
			tag:      tag.New(0x0009, 0x0010),
			expected: "(0009,0010)",
		},
		{
			name:     "pixel data tag",
			tag:      tag.New(0x7FE0, 0x0010),
			expected: "(7FE0,0010)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.tag.String()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTag_Uint32(t *testing.T) {
	tests := []struct {
		name     string
		tag      tag.Tag
		expected uint32
	}{
		{
			name:     "standard tag",
			tag:      tag.New(0x0008, 0x0020),
			expected: 0x00080020,
		},
		{
			name:     "pixel data tag",
			tag:      tag.New(0x7FE0, 0x0010),
			expected: 0x7FE00010,
		},
		{
			name:     "max values",
			tag:      tag.New(0xFFFF, 0xFFFF),
			expected: 0xFFFFFFFF,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.tag.Uint32()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTag_IsPrivate(t *testing.T) {
	tests := []struct {
		name     string
		tag      tag.Tag
		expected bool
	}{
		{
			name:     "standard tag (even group)",
			tag:      tag.New(0x0008, 0x0020),
			expected: false,
		},
		{
			name:     "private tag (odd group)",
			tag:      tag.New(0x0009, 0x0020),
			expected: true,
		},
		{
			name:     "another standard tag",
			tag:      tag.New(0x0010, 0x0010),
			expected: false,
		},
		{
			name:     "another private tag",
			tag:      tag.New(0x0011, 0x0010),
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.tag.IsPrivate()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTag_IsMetaElement(t *testing.T) {
	tests := []struct {
		name     string
		tag      tag.Tag
		expected bool
	}{
		{
			name:     "meta element group",
			tag:      tag.New(0x0002, 0x0010),
			expected: true,
		},
		{
			name:     "non-meta element group",
			tag:      tag.New(0x0008, 0x0020),
			expected: false,
		},
		{
			name:     "another meta element",
			tag:      tag.New(0x0002, 0x0001),
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.tag.IsMetaElement()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTag_Parse(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTag   tag.Tag
		wantError bool
	}{
		{
			name:      "valid tag with parentheses",
			input:     "(0008,0020)",
			wantTag:   tag.New(0x0008, 0x0020),
			wantError: false,
		},
		{
			name:      "valid tag without parentheses",
			input:     "0008,0020",
			wantTag:   tag.New(0x0008, 0x0020),
			wantError: false,
		},
		{
			name:      "valid tag with lowercase",
			input:     "(7fe0,0010)",
			wantTag:   tag.New(0x7FE0, 0x0010),
			wantError: false,
		},
		{
			name:      "invalid format",
			input:     "not-a-tag",
			wantTag:   tag.Tag{},
			wantError: true,
		},
		{
			name:      "empty string",
			input:     "",
			wantTag:   tag.Tag{},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tag.Parse(tc.input)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantTag, result)
			}
		})
	}
}

func TestFind(t *testing.T) {
	tests := []struct {
		name        string
		tag         tag.Tag
		wantErr     bool
		wantKeyword string
	}{
		{
			name:        "valid standard tag",
			tag:         tag.New(0x0008, 0x0005),
			wantErr:     false,
			wantKeyword: "SpecificCharacterSet",
		},
		{
			name:        "valid SOP Class UID tag",
			tag:         tag.New(0x0008, 0x0016),
			wantErr:     false,
			wantKeyword: "SOPClassUID",
		},
		{
			name:        "GenericGroupLength special case",
			tag:         tag.New(0x0008, 0x0000),
			wantErr:     false,
			wantKeyword: "GenericGroupLength",
		},
		{
			name:        "another GenericGroupLength",
			tag:         tag.New(0x0010, 0x0000),
			wantErr:     false,
			wantKeyword: "GenericGroupLength",
		},
		{
			name:    "unknown tag returns error",
			tag:     tag.New(0x9999, 0x9999),
			wantErr: true,
		},
		{
			name:    "private tag returns error",
			tag:     tag.New(0x0009, 0x0010),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tag.Find(tt.tag)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.tag, got.Tag)
				assert.Equal(t, tt.wantKeyword, got.Keyword)
			}
		})
	}
}

func TestFindByKeyword(t *testing.T) {
	tests := []struct {
		name    string
		keyword string
		wantErr bool
		wantTag tag.Tag
	}{
		{
			name:    "find by keyword",
			keyword: "SpecificCharacterSet",
			wantErr: false,
			wantTag: tag.New(0x0008, 0x0005),
		},
		{
			name:    "find by name (fallback)",
			keyword: "Specific Character Set",
			wantErr: false,
			wantTag: tag.New(0x0008, 0x0005),
		},
		{
			name:    "SOP Class UID by keyword",
			keyword: "SOPClassUID",
			wantErr: false,
			wantTag: tag.New(0x0008, 0x0016),
		},
		{
			name:    "unknown keyword",
			keyword: "NonExistentKeyword",
			wantErr: true,
		},
		{
			name:    "empty string",
			keyword: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tag.FindByKeyword(tt.keyword)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantTag, got.Tag)
			}
		})
	}
}

func TestFindByName(t *testing.T) {
	tests := []struct {
		name        string
		tagName     string
		wantErr     bool
		wantTag     tag.Tag
		wantKeyword string
	}{
		{
			name:        "find by name",
			tagName:     "Specific Character Set",
			wantErr:     false,
			wantTag:     tag.New(0x0008, 0x0005),
			wantKeyword: "SpecificCharacterSet",
		},
		{
			name:        "SOP Class UID by name",
			tagName:     "SOP Class UID",
			wantErr:     false,
			wantTag:     tag.New(0x0008, 0x0016),
			wantKeyword: "SOPClassUID",
		},
		{
			name:    "unknown name",
			tagName: "Non Existent Tag",
			wantErr: true,
		},
		{
			name:    "empty string",
			tagName: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tag.FindByName(tt.tagName)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantTag, got.Tag)
				assert.Equal(t, tt.wantKeyword, got.Keyword)
			}
		})
	}
}

func TestMustFind(t *testing.T) {
	t.Run("valid tag returns Info", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("MustFind should not panic for valid tag, but panicked with: %v", r)
			}
		}()

		result := tag.MustFind(tag.New(0x0008, 0x0005))
		assert.Equal(t, tag.New(0x0008, 0x0005), result.Tag)
		assert.Equal(t, "SpecificCharacterSet", result.Keyword)
	})

	t.Run("invalid tag panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustFind should panic for invalid tag, but did not panic")
			}
		}()

		tag.MustFind(tag.New(0x9999, 0x9999))
	})
}
