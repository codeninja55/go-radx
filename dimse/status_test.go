package dimse

import "testing"

// TestStatusEchoSuccess is the named regression: StatusEchoSuccess must categorise as Success.
func TestStatusEchoSuccess(t *testing.T) {
	if !StatusEchoSuccess.IsSuccess() {
		t.Errorf("StatusEchoSuccess.IsSuccess() = false, want true (category %v)", StatusEchoSuccess.Category())
	}
	if StatusEchoSuccess.Code != 0x0000 {
		t.Errorf("StatusEchoSuccess.Code = %#04x, want 0x0000", StatusEchoSuccess.Code)
	}
}

// TestStatusSuccess checks the general-class success constant.
func TestStatusSuccess(t *testing.T) {
	if !StatusSuccess.IsSuccess() {
		t.Errorf("StatusSuccess.IsSuccess() = false, want true")
	}
	if StatusSuccess.Category() != StatusCategorySuccess {
		t.Errorf("StatusSuccess.Category() = %v, want Success", StatusSuccess.Category())
	}
}

// TestStatusCategoriesAcrossClasses checks the category methods on a representative status from
// each modelled class, exercising the categorisation tables.
func TestStatusCategoriesAcrossClasses(t *testing.T) {
	cases := []struct {
		name string
		s    Status
		want StatusCategory
	}{
		{"general success", NewStatus(0x0000, ServiceClassGeneral), StatusCategorySuccess},
		{"general cancel", NewStatus(0xFE00, ServiceClassGeneral), StatusCategoryCancel},
		{"verification success", NewStatus(0x0000, ServiceClassVerification), StatusCategorySuccess},
		{"sop class not supported", NewStatus(0x0122, ServiceClassVerification), StatusCategoryFailure},
		{"refused out of resources", NewStatus(0xA700, ServiceClassGeneral), StatusCategoryFailure},
	}
	for _, c := range cases {
		if got := c.s.Category(); got != c.want {
			t.Errorf("%s: Category() = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestStatusUnknownPreservesCode is the named regression: a code with no registered meaning in
// the active service class resolves to StatusCategoryUnknown with its raw code preserved, never
// coerced to success.
func TestStatusUnknownPreservesCode(t *testing.T) {
	s := NewStatus(0x1234, ServiceClassVerification)
	if s.Category() != StatusCategoryUnknown {
		t.Errorf("unknown code Category() = %v, want StatusCategoryUnknown", s.Category())
	}
	if s.IsSuccess() {
		t.Error("an unknown code must not report IsSuccess()")
	}
	if s.Code != 0x1234 {
		t.Errorf("unknown code preserved as %#04x, want 0x1234", s.Code)
	}
}

// TestStatusString checks the "0xNNNN Category[: Meaning]" rendering, never bare hex.
func TestStatusString(t *testing.T) {
	cases := []struct {
		s    Status
		want string
	}{
		{StatusSuccess, "0x0000 Success"},
		{StatusEchoSuccess, "0x0000 Success"},
		{NewStatus(0x0122, ServiceClassVerification), "0x0122 Failure: Refused: SOP Class Not Supported"},
		{NewStatus(0x1234, ServiceClassVerification), "0x1234 Unknown"},
	}
	for _, c := range cases {
		if got := c.s.String(); got != c.want {
			t.Errorf("Status{%#04x}.String() = %q, want %q", c.s.Code, got, c.want)
		}
	}
}

// TestStatusCategoryPredicates checks the boolean predicates track Category().
func TestStatusCategoryPredicates(t *testing.T) {
	if !NewStatus(0xFE00, ServiceClassGeneral).IsCancel() {
		t.Error("0xFE00 should report IsCancel()")
	}
	if !NewStatus(0x0122, ServiceClassVerification).IsFailure() {
		t.Error("0x0122 should report IsFailure()")
	}
	if NewStatus(0x0000, ServiceClassVerification).IsFailure() {
		t.Error("0x0000 must not report IsFailure()")
	}
}
