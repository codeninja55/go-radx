package hl7v2

import (
	"errors"
	"testing"
)

func TestParseAckCode(t *testing.T) {
	cases := map[string]AckCode{
		"AA": AckAccept,
		"AE": AckError,
		"AR": AckReject,
		"CA": AckCommitAccept,
		"CE": AckCommitError,
		"CR": AckCommitReject,
	}
	for raw, want := range cases {
		got, err := ParseAckCode(raw)
		if err != nil {
			t.Errorf("ParseAckCode(%q) error = %v, want nil", raw, err)
		}
		if got != want {
			t.Errorf("ParseAckCode(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestParseAckCodeUnknown(t *testing.T) {
	_, err := ParseAckCode("ZZ")
	if _, ok := errors.AsType[*ParseError](err); !ok {
		t.Fatalf("ParseAckCode(ZZ) error = %v, want *ParseError", err)
	}
}

func TestAckCodePredicates(t *testing.T) {
	positives := []AckCode{AckAccept, AckCommitAccept}
	errorsCodes := []AckCode{AckError, AckCommitError}
	rejects := []AckCode{AckReject, AckCommitReject}

	for _, c := range positives {
		if !c.IsPositive() || c.IsError() || c.IsReject() {
			t.Errorf("%q predicates wrong: positive want true", c)
		}
	}
	for _, c := range errorsCodes {
		if !c.IsError() || c.IsPositive() || c.IsReject() {
			t.Errorf("%q predicates wrong: error want true", c)
		}
	}
	for _, c := range rejects {
		if !c.IsReject() || c.IsPositive() || c.IsError() {
			t.Errorf("%q predicates wrong: reject want true", c)
		}
	}
}
