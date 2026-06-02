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

// TestStatusStoreSuccess is the named regression: StatusStoreSuccess must categorise as Success
// against the Storage service class.
func TestStatusStoreSuccess(t *testing.T) {
	if !StatusStoreSuccess.IsSuccess() {
		t.Errorf("StatusStoreSuccess.IsSuccess() = false, want true (category %v)", StatusStoreSuccess.Category())
	}
	if StatusStoreSuccess.Code != 0x0000 {
		t.Errorf("StatusStoreSuccess.Code = %#04x, want 0x0000", StatusStoreSuccess.Code)
	}
	if StatusStoreSuccess.ServiceClass() != ServiceClassStorage {
		t.Errorf("StatusStoreSuccess service class = %v, want Storage", StatusStoreSuccess.ServiceClass())
	}
}

// TestStorageStatusTable exercises the Storage service-class status table (PS3.4 B.2.3, verified
// against pynetdicom STORAGE_SERVICE_CLASS_STATUS): the named failure codes, the warning codes,
// and the ranged bands resolve to the right categories and meanings.
func TestStorageStatusTable(t *testing.T) {
	cases := []struct {
		name     string
		s        Status
		category StatusCategory
		meaning  string
	}{
		{"cannot understand", StatusStoreCannotUnderstand, StatusCategoryFailure, "Cannot Understand"},
		{"out of resources", StatusStoreOutOfResources, StatusCategoryFailure, "Refused: Out of Resources"},
		{"dataset mismatch (failure)", StatusStoreDataSetDoesNotMatchSOPClass, StatusCategoryFailure, "Data Set Does Not Match SOP Class"},
		{"coercion warning", StatusStoreCoercionOfDataElements, StatusCategoryWarning, "Coercion of Data Elements"},
		{"element discarded", StatusStoreElementDiscarded, StatusCategoryWarning, "Element Discarded"},
		{"dataset mismatch (warning)", StatusStoreDataSetDoesNotMatchSOPClassWarning, StatusCategoryWarning, "Data Set Does Not Match SOP Class"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.s.Category() != c.category {
				t.Errorf("%s category = %v, want %v", c.name, c.s.Category(), c.category)
			}
			if c.s.Meaning() != c.meaning {
				t.Errorf("%s meaning = %q, want %q", c.name, c.s.Meaning(), c.meaning)
			}
			if c.s.ServiceClass() != ServiceClassStorage {
				t.Errorf("%s service class = %v, want Storage", c.name, c.s.ServiceClass())
			}
		})
	}

	// A code inside a Storage failure band resolves to the band meaning even though it is not a
	// named code, against the Storage class.
	band := NewStatus(0xA7FF, ServiceClassStorage)
	if !band.IsFailure() {
		t.Errorf("0xA7FF (Out of Resources band) IsFailure() = false, want true")
	}
}

// TestQueryRetrieveStatusCategories exercises the C-FIND/C-GET/C-MOVE service-class tables
// (verified against pynetdicom QR_FIND/QR_MOVE/QR_GET_SERVICE_CLASS_STATUS): the same numeric
// code categorises differently from Storage — 0xFF00/0xFF01 are Pending, 0xB000 is the
// C-GET/C-MOVE "one or more sub-operations failed" Warning (not a failure), and 0xA801 is the
// C-MOVE "Move Destination Unknown" Failure.
func TestQueryRetrieveStatusCategories(t *testing.T) {
	cases := []struct {
		name string
		s    Status
		want StatusCategory
	}{
		{"find pending", StatusFindPending, StatusCategoryPending},
		{"find pending optional-keys", NewStatus(0xFF01, ServiceClassFind), StatusCategoryPending},
		{"find success", StatusFindSuccess, StatusCategorySuccess},
		{"move sub-ops failure warning", NewStatus(0xB000, ServiceClassMove), StatusCategoryWarning},
		{"move destination unknown", StatusMoveDestinationUnknown, StatusCategoryFailure},
		{"get sub-ops failure warning", NewStatus(0xB000, ServiceClassGet), StatusCategoryWarning},
		{"cancel", NewStatus(0xFE00, ServiceClassMove), StatusCategoryCancel},
	}
	for _, c := range cases {
		if got := c.s.Category(); got != c.want {
			t.Errorf("%s: Category() = %s, want %s", c.name, got, c.want)
		}
	}
	// A pending status must NEVER read as success, the laundering bug the typed model prevents.
	if StatusFindPending.IsSuccess() {
		t.Error("StatusFindPending.IsSuccess() must be false")
	}
}
