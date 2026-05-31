package dicom

import (
	"math/big"
	"testing"
)

func TestDecimalPreservesLexicalForm(t *testing.T) {
	for _, s := range []string{"1.500", "-0.0", "3.14159265358979", "100", "+2.5"} {
		d, err := ParseDecimal(s)
		if err != nil {
			t.Fatalf("ParseDecimal(%q): %v", s, err)
		}
		if d.String() != s {
			t.Errorf("String() = %q, want preserved %q", d.String(), s)
		}
	}
}

func TestDecimalFloat64AndExact(t *testing.T) {
	d, _ := ParseDecimal("1.5")
	if f, ok := d.Float64(); !ok || f != 1.5 {
		t.Errorf("Float64() = (%v,%v), want (1.5,true)", f, ok)
	}
	if !d.Exact() {
		t.Error("1.5 is exactly representable as float64")
	}
	d2, _ := ParseDecimal("0.1")
	if _, ok := d2.Float64(); !ok {
		t.Error("0.1 should return ok == true (representable but rounded)")
	}
	if d2.Exact() {
		t.Error("0.1 is not exactly representable as float64")
	}
}

func TestDecimalBigFloat(t *testing.T) {
	d, _ := ParseDecimal("3.14159265358979")
	bf := d.BigFloat()
	want, _, _ := big.ParseFloat("3.14159265358979", 10, bf.Prec(), big.ToNearestEven)
	if bf.Cmp(want) != 0 {
		t.Errorf("BigFloat() = %v, want %v", bf, want)
	}
}

func TestDecimalDSLengthLimit(t *testing.T) {
	// DS is limited to 16 bytes per value (PS3.5).
	if _, err := ParseDecimal("12345678901234567"); err == nil {
		t.Error("a 17-byte DS value should be rejected")
	}
}

func TestDecimalInt64(t *testing.T) {
	d, _ := ParseDecimal("42")
	if n, ok := d.Int64(); !ok || n != 42 {
		t.Errorf("Int64() = (%d,%v), want (42,true)", n, ok)
	}
	d2, _ := ParseDecimal("4.2")
	if _, ok := d2.Int64(); ok {
		t.Error("4.2 is not integral")
	}
}

func TestDecimalJSONRoundTrip(t *testing.T) {
	d, _ := ParseDecimal("1.500")
	b, err := d.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "1.500" {
		t.Errorf("MarshalJSON() = %q, want unquoted 1.500 (FHIR decimal)", b)
	}
	var d2 Decimal
	if err := d2.UnmarshalJSON([]byte("1.500")); err != nil {
		t.Fatal(err)
	}
	if d2.String() != "1.500" {
		t.Errorf("round-trip lost lexical form: %q", d2.String())
	}
}
