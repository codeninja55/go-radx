package dicomweb

import (
	"errors"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
)

func TestResourcePathLevel(t *testing.T) {
	cases := []struct {
		name string
		path ResourcePath
		want Level
	}{
		{"study", NewStudy("1.2.3"), LevelStudy},
		{"series", NewSeries("1.2.3", "1.2.3.4"), LevelSeries},
		{"instance", NewInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"), LevelInstance},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.path.Level(); got != tc.want {
				t.Fatalf("Level() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResourcePathRenders(t *testing.T) {
	cases := []struct {
		name string
		path ResourcePath
		want string
	}{
		{"study", NewStudy("1.2.3"), "/studies/1.2.3"},
		{"series", NewSeries("1.2.3", "1.2.3.4"), "/studies/1.2.3/series/1.2.3.4"},
		{
			"instance",
			NewInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"),
			"/studies/1.2.3/series/1.2.3.4/instances/1.2.3.4.5",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.path.Path()
			if err != nil {
				t.Fatalf("Path() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("Path() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestResourcePathRejectsInvalidUID is the named regression: a bad UID is rejected with
// ErrInvalidResource, never interpolated or escaped into the URL.
func TestResourcePathRejectsInvalidUID(t *testing.T) {
	cases := []struct {
		name string
		path ResourcePath
	}{
		{"empty study", NewStudy("")},
		{"non-numeric study", NewStudy("1.2.abc")},
		{"leading-zero component", NewStudy("1.02.3")},
		{"path-injection attempt", NewStudy("1.2.3/series/evil")},
		{"bad series", NewSeries("1.2.3", "1..4")},
		{"bad instance", NewInstance("1.2.3", "1.2.3.4", "1.2.3.4.x")},
		{"instance without series", ResourcePath{Study: "1.2.3", Instance: "1.2.3.4.5"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.path.Path(); !errors.Is(err, ErrInvalidResource) {
				t.Fatalf("Path() error = %v, want ErrInvalidResource", err)
			}
		})
	}
}

func TestResourcePathFrames(t *testing.T) {
	p := NewInstance("1.2.3", "1.2.3.4", "1.2.3.4.5")
	got, err := p.Frames(1, 4, 5)
	if err != nil {
		t.Fatalf("Frames() error = %v", err)
	}
	want := "/studies/1.2.3/series/1.2.3.4/instances/1.2.3.4.5/frames/1,4,5"
	if got != want {
		t.Fatalf("Frames() = %q, want %q", got, want)
	}

	if _, err := p.Frames(0); !errors.Is(err, ErrInvalidResource) {
		t.Fatalf("Frames(0) error = %v, want ErrInvalidResource", err)
	}
	if _, err := NewStudy("1.2.3").Frames(1); !errors.Is(err, ErrInvalidResource) {
		t.Fatalf("study-level Frames() error = %v, want ErrInvalidResource", err)
	}
}

func TestValidateUIDDelegatesToDicom(t *testing.T) {
	// A 65-character UID exceeds the PS3.5 limit; validateUID must reject it via the
	// dicom validator rather than admit it.
	long := dicom.UID("1." + repeat("2", 64))
	if err := validateUID(long, "StudyInstanceUID"); !errors.Is(err, ErrInvalidResource) {
		t.Fatalf("validateUID(long) = %v, want ErrInvalidResource", err)
	}
}

func repeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for range n {
		out = append(out, s...)
	}
	return string(out)
}
