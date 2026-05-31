package dicom

import (
	"errors"
	"testing"
)

func TestNewSpecificCharacterSetEmptyIsDefault(t *testing.T) {
	cs, err := NewSpecificCharacterSet()
	if err != nil {
		t.Fatalf("NewSpecificCharacterSet(): %v", err)
	}
	got, err := cs.Decode([]byte("Doe^John"))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got != "Doe^John" {
		t.Errorf("Decode = %q, want %q", got, "Doe^John")
	}
}

func TestNewSpecificCharacterSetEmptyDefinedTermIsDefault(t *testing.T) {
	// (0008,0005) present but with a zero-length value designates the default
	// repertoire (ISO_IR 6 / ISO 646).
	cs, err := NewSpecificCharacterSet("")
	if err != nil {
		t.Fatalf("NewSpecificCharacterSet(%q): %v", "", err)
	}
	got, err := cs.Decode([]byte("ABC"))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got != "ABC" {
		t.Errorf("Decode = %q, want ABC", got)
	}
}

func TestSpecificCharacterSetLatin1RoundTrip(t *testing.T) {
	cs, err := NewSpecificCharacterSet("ISO_IR 100")
	if err != nil {
		t.Fatalf("NewSpecificCharacterSet: %v", err)
	}
	// "Äneäs^Rüdiger" encoded in ISO 8859-1 (Latin-1).
	raw := []byte{0xc4, 'n', 'e', 0xe4, 's', '^', 'R', 0xfc, 'd', 'i', 'g', 'e', 'r'}
	want := "Äneäs^Rüdiger"

	got, err := cs.Decode(raw)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got != want {
		t.Fatalf("Decode = %q, want %q", got, want)
	}

	back, err := cs.Encode(got)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if string(back) != string(raw) {
		t.Errorf("Encode round-trip = % x, want % x", back, raw)
	}
}

func TestSpecificCharacterSetUTF8RoundTrip(t *testing.T) {
	cs, err := NewSpecificCharacterSet("ISO_IR 192")
	if err != nil {
		t.Fatalf("NewSpecificCharacterSet: %v", err)
	}
	want := "Wang^XiaoDong=王^小東"
	raw := []byte(want) // UTF-8 bytes verbatim

	got, err := cs.Decode(raw)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got != want {
		t.Fatalf("Decode = %q, want %q", got, want)
	}

	back, err := cs.Encode(got)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if string(back) != string(raw) {
		t.Errorf("Encode round-trip = % x, want % x", back, raw)
	}
}

func TestSpecificCharacterSetGB18030RoundTrip(t *testing.T) {
	cs, err := NewSpecificCharacterSet("GB18030")
	if err != nil {
		t.Fatalf("NewSpecificCharacterSet: %v", err)
	}
	want := "Wang^XiaoDong=王^小东"

	enc, err := cs.Encode(want)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	got, err := cs.Decode(enc)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got != want {
		t.Fatalf("round-trip = %q, want %q", got, want)
	}
}

func TestSpecificCharacterSetGBKRoundTrip(t *testing.T) {
	cs, err := NewSpecificCharacterSet("GBK")
	if err != nil {
		t.Fatalf("NewSpecificCharacterSet: %v", err)
	}
	want := "汉字"
	enc, err := cs.Encode(want)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	got, err := cs.Decode(enc)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got != want {
		t.Fatalf("round-trip = %q, want %q", got, want)
	}
}

func TestNewSpecificCharacterSetUnknownTermErrors(t *testing.T) {
	_, err := NewSpecificCharacterSet("ISO_IR 9999")
	if err == nil {
		t.Fatal("NewSpecificCharacterSet(ISO_IR 9999) = nil error, want typed error")
	}
	var ue *UnsupportedCharacterSetError
	if !errors.As(err, &ue) {
		t.Fatalf("error is %T, want *UnsupportedCharacterSetError", err)
	}
	if ue.DefinedTerm != "ISO_IR 9999" {
		t.Errorf("DefinedTerm = %q, want %q", ue.DefinedTerm, "ISO_IR 9999")
	}
}

func TestSpecificCharacterSetWhitespaceTolerant(t *testing.T) {
	// Defined terms may carry leading/trailing whitespace from padding or sloppy
	// producers; resolution trims them.
	cs, err := NewSpecificCharacterSet(" ISO_IR 100 ")
	if err != nil {
		t.Fatalf("NewSpecificCharacterSet: %v", err)
	}
	got, err := cs.Decode([]byte{0xc4})
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got != "Ä" {
		t.Errorf("Decode = %q, want Ä", got)
	}
}
