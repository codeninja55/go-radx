package r5_test

import (
	"encoding/json"
	"testing"

	"github.com/codeninja55/go-radx/fhir/r5"
)

// TestGeneratedPeriodRoundTrips exercises the first generated datatype end to end:
// the emitted r5.Period must marshal to the expected JSON and decode back unchanged,
// proving the generator produces usable Go, not just compiling Go.
func TestGeneratedPeriodRoundTrips(t *testing.T) {
	p := r5.Period{Start: strptr("2024-01-01T00:00:00Z"), End: strptr("2024-12-31T23:59:59Z")}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"start":"2024-01-01T00:00:00Z","end":"2024-12-31T23:59:59Z"}`
	if string(b) != want {
		t.Errorf("Period JSON = %s, want %s", b, want)
	}

	var back r5.Period
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.Start == nil || *back.Start != *p.Start || back.End == nil || *back.End != *p.End {
		t.Errorf("round-tripped Period = %+v, want %+v", back, p)
	}
}

// TestGeneratedPeriodOmitsEmpty asserts an absent optional scalar is omitted from the
// wire form, the omitempty contract the planner records for optional fields.
func TestGeneratedPeriodOmitsEmpty(t *testing.T) {
	b, err := json.Marshal(r5.Period{Start: strptr("2024-01-01")})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"start":"2024-01-01"}` {
		t.Errorf("Period with only Start = %s, want {\"start\":\"2024-01-01\"}", b)
	}
}
