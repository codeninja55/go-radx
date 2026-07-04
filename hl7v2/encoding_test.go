package hl7v2

import (
	"errors"
	"testing"
)

func TestDefaultEncoding(t *testing.T) {
	enc := DefaultEncoding()
	want := EncodingCharacters{Field: '|', Component: '^', Repetition: '~', Escape: '\\', Subcomponent: '&'}
	if enc != want {
		t.Fatalf("DefaultEncoding() = %+v, want %+v", enc, want)
	}
}

func TestDeriveEncoding(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   EncodingCharacters
	}{
		{
			name:   "complete standard header",
			header: `MSH|^~\&|`,
			want:   EncodingCharacters{Field: '|', Component: '^', Repetition: '~', Escape: '\\', Subcomponent: '&'},
		},
		{
			name:   "short MSH-2 fills repetition/escape/subcomponent from defaults",
			header: `MSH|^`,
			want:   EncodingCharacters{Field: '|', Component: '^', Repetition: '~', Escape: '\\', Subcomponent: '&'},
		},
		{
			name:   "non-standard field separator",
			header: `MSH#^~\&#`,
			want:   EncodingCharacters{Field: '#', Component: '^', Repetition: '~', Escape: '\\', Subcomponent: '&'},
		},
		{
			name:   "non-standard component and repetition",
			header: `MSH|@!\&|`,
			want:   EncodingCharacters{Field: '|', Component: '@', Repetition: '!', Escape: '\\', Subcomponent: '&'},
		},
		{
			name:   "MSH-2 absent fills all four from defaults",
			header: `MSH|`,
			want:   EncodingCharacters{Field: '|', Component: '^', Repetition: '~', Escape: '\\', Subcomponent: '&'},
		},
		{
			name:   "MSH-2 runs to end of header with no trailing field separator",
			header: `MSH|^~\&`,
			want:   EncodingCharacters{Field: '|', Component: '^', Repetition: '~', Escape: '\\', Subcomponent: '&'},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DeriveEncoding([]byte(tc.header))
			if err != nil {
				t.Fatalf("DeriveEncoding(%q) error = %v", tc.header, err)
			}
			if got != tc.want {
				t.Fatalf("DeriveEncoding(%q) = %+v, want %+v", tc.header, got, tc.want)
			}
		})
	}
}

func TestDeriveEncodingNotMSH(t *testing.T) {
	for _, header := range []string{"PID|", "MS", "", "BHS|^~\\&"} {
		_, err := DeriveEncoding([]byte(header))
		if _, ok := errors.AsType[*ParseError](err); !ok {
			t.Fatalf("DeriveEncoding(%q) error = %v, want *ParseError", header, err)
		}
	}
}
