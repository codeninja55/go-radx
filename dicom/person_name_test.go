package dicom

import "testing"

func TestParsePersonNameAlphabetic(t *testing.T) {
	pn, err := ParsePersonName("Doe^John^Q^Dr^Jr")
	if err != nil {
		t.Fatal(err)
	}
	a := pn.Alphabetic
	if a.FamilyName != "Doe" || a.GivenName != "John" || a.MiddleName != "Q" ||
		a.Prefix != "Dr" || a.Suffix != "Jr" {
		t.Errorf("components = %+v", a)
	}
}

func TestParsePersonNameThreeGroups(t *testing.T) {
	// alphabetic=Yamada^Tarou, ideographic, phonetic.
	pn, err := ParsePersonName("Yamada^Tarou=山田^太郎=yamada^tarou")
	if err != nil {
		t.Fatal(err)
	}
	if pn.Alphabetic.FamilyName != "Yamada" {
		t.Errorf("alphabetic family = %q", pn.Alphabetic.FamilyName)
	}
	if pn.Ideographic.FamilyName == "" {
		t.Error("ideographic group should be populated")
	}
	if pn.Phonetic.FamilyName != "yamada" {
		t.Errorf("phonetic family = %q", pn.Phonetic.FamilyName)
	}
}

func TestParsePersonNameRejectsTooManyGroups(t *testing.T) {
	if _, err := ParsePersonName("a=b=c=d"); err == nil {
		t.Error("more than three component groups should error")
	}
}

func TestPersonNameStringDropsTrailingEmpties(t *testing.T) {
	pn, _ := ParsePersonName("Doe^John^^^")
	if got := pn.String(); got != "Doe^John" {
		t.Errorf("String() = %q, want Doe^John", got)
	}
}
