package dicom

import "testing"

func TestTagAccessorsAndString(t *testing.T) {
	tests := []struct {
		name           string
		group, element uint16
		wantString     string
	}{
		{"patient name", 0x0010, 0x0010, "(0010,0010)"},
		{"study instance uid", 0x0020, 0x000D, "(0020,000D)"},
		{"pixel data", 0x7FE0, 0x0010, "(7FE0,0010)"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tag := NewTag(tc.group, tc.element)
			if got := tag.Group(); got != tc.group {
				t.Errorf("Group() = %#04x, want %#04x", got, tc.group)
			}
			if got := tag.Element(); got != tc.element {
				t.Errorf("Element() = %#04x, want %#04x", got, tc.element)
			}
			if got := tag.String(); got != tc.wantString {
				t.Errorf("String() = %q, want %q", got, tc.wantString)
			}
		})
	}
}

func TestTagPredicates(t *testing.T) {
	if !NewTag(0x0009, 0x0010).IsPrivate() {
		t.Error("odd group should be private")
	}
	if NewTag(0x0010, 0x0010).IsPrivate() {
		t.Error("even group should not be private")
	}
	if !NewTag(0x0009, 0x0010).IsPrivateCreator() {
		t.Error("(0009,0010) should be a private creator tag")
	}
	if NewTag(0x0009, 0x1001).IsPrivateCreator() {
		t.Error("(0009,1001) is a private data tag, not a creator")
	}
	if !NewTag(0x0010, 0x0000).IsGroupLength() {
		t.Error("element 0x0000 should be a group-length tag")
	}
}
