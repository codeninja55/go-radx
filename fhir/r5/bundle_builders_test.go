package r5_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
)

func TestNewSearchSetSetsTotalAndType(t *testing.T) {
	bundle, err := r5.NewSearchSet(3,
		r5.SearchEntry{FullURL: "urn:uuid:obs-1", Resource: &r5.Observation{}},
		r5.SearchEntry{FullURL: "urn:uuid:obs-2", Resource: &r5.Observation{}},
	)
	if err != nil {
		t.Fatalf("NewSearchSet: %v", err)
	}
	if bundle.Type == nil || *bundle.Type != r5.BundleTypeSearchset {
		t.Errorf("type = %v, want searchset", bundle.Type)
	}
	if bundle.Total == nil || *bundle.Total != 3 {
		t.Errorf("total = %v, want 3", bundle.Total)
	}
	if len(bundle.Entry) != 2 {
		t.Fatalf("entry count = %d, want 2", len(bundle.Entry))
	}
}

func TestNewSearchSetCarriesSearchMetadata(t *testing.T) {
	mode := r5.SearchEntryModeMatch
	bundle, err := r5.NewSearchSet(1, r5.SearchEntry{
		FullURL:  "urn:uuid:obs-1",
		Resource: &r5.Observation{},
		Mode:     &mode,
	})
	if err != nil {
		t.Fatalf("NewSearchSet: %v", err)
	}
	if bundle.Entry[0].Search == nil || bundle.Entry[0].Search.Mode == nil {
		t.Fatalf("search metadata not carried through")
	}
	if *bundle.Entry[0].Search.Mode != r5.SearchEntryModeMatch {
		t.Errorf("search mode = %v, want match", *bundle.Entry[0].Search.Mode)
	}
}

func TestNewSearchSetRejectsNegativeTotal(t *testing.T) {
	_, err := r5.NewSearchSet(-1)
	if !errors.Is(err, r5.ErrInvalidBundle) {
		t.Fatalf("err = %v, want ErrInvalidBundle", err)
	}
}

func TestCollectionLeavesTotalUnset(t *testing.T) {
	bundle, err := r5.NewCollection(
		r5.CollectionEntry{FullURL: "urn:uuid:a", Resource: &r5.Observation{}},
	)
	if err != nil {
		t.Fatalf("NewCollection: %v", err)
	}
	if bundle.Total != nil {
		t.Errorf("collection total = %v, want unset (nil)", *bundle.Total)
	}
	if bundle.Type == nil || *bundle.Type != r5.BundleTypeCollection {
		t.Errorf("type = %v, want collection", bundle.Type)
	}
}

func TestNewTransactionRequiresRequest(t *testing.T) {
	bundle, err := r5.NewTransaction(
		r5.TransactionEntry{Resource: &r5.Patient{}, Method: r5.HTTPVerbPOST, URL: "Patient"},
	)
	if err != nil {
		t.Fatalf("NewTransaction: %v", err)
	}
	if bundle.Entry[0].Request == nil {
		t.Fatalf("transaction entry has no request")
	}
	if bundle.Entry[0].Request.Method == nil || *bundle.Entry[0].Request.Method != r5.HTTPVerbPOST {
		t.Errorf("request method = %v, want POST", bundle.Entry[0].Request.Method)
	}
	if bundle.Total != nil {
		t.Errorf("transaction total = %v, want unset", *bundle.Total)
	}
}

func TestNewTransactionRejectsEmptyURL(t *testing.T) {
	_, err := r5.NewTransaction(
		r5.TransactionEntry{Resource: &r5.Patient{}, Method: r5.HTTPVerbPOST, URL: ""},
	)
	if !errors.Is(err, r5.ErrInvalidBundle) {
		t.Fatalf("err = %v, want ErrInvalidBundle", err)
	}
	if !strings.Contains(err.Error(), "entry 0") {
		t.Errorf("error should name the offending entry index, got %q", err)
	}
}

func TestNewTransactionRejectsInvalidMethod(t *testing.T) {
	_, err := r5.NewTransaction(
		r5.TransactionEntry{Resource: &r5.Patient{}, Method: r5.HTTPVerb("TRACE"), URL: "Patient"},
	)
	if !errors.Is(err, r5.ErrInvalidBundle) {
		t.Fatalf("err = %v, want ErrInvalidBundle", err)
	}
}

// TestTransactionRejectsResponseEntry is the FHIR-010 regression: a transaction entry
// must carry a request, never a response. The typed TransactionEntry has no response
// field at all, so a response is unrepresentable in a transaction built through the
// builder; this test pins that a transaction the builder produces never serialises a
// response on any entry.
func TestTransactionRejectsResponseEntry(t *testing.T) {
	bundle, err := r5.NewTransaction(
		r5.TransactionEntry{Resource: &r5.Patient{}, Method: r5.HTTPVerbPUT, URL: "Patient/p1"},
	)
	if err != nil {
		t.Fatalf("NewTransaction: %v", err)
	}
	for i := range bundle.Entry {
		if bundle.Entry[i].Response != nil {
			t.Fatalf("transaction entry %d carries a response, want none", i)
		}
	}
	encoded, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(encoded), `"response"`) {
		t.Errorf("transaction bundle serialised a response: %s", encoded)
	}
}

func TestNewBatchRequiresRequest(t *testing.T) {
	_, err := r5.NewBatch(
		r5.TransactionEntry{Resource: &r5.Patient{}, Method: r5.HTTPVerbGET, URL: ""},
	)
	if !errors.Is(err, r5.ErrInvalidBundle) {
		t.Fatalf("err = %v, want ErrInvalidBundle", err)
	}
}

func TestTransactionRejectsDuplicateFullURL(t *testing.T) {
	_, err := r5.NewTransaction(
		r5.TransactionEntry{FullURL: "urn:uuid:dup", Resource: &r5.Patient{}, Method: r5.HTTPVerbPOST, URL: "Patient"},
		r5.TransactionEntry{FullURL: "urn:uuid:dup", Resource: &r5.Patient{}, Method: r5.HTTPVerbPOST, URL: "Patient"},
	)
	if !errors.Is(err, r5.ErrInvalidBundle) {
		t.Fatalf("err = %v, want ErrInvalidBundle", err)
	}
	if !strings.Contains(err.Error(), "fullUrl") {
		t.Errorf("error should mention fullUrl, got %q", err)
	}
}

func TestNewDocumentRequiresCompositionFirst(t *testing.T) {
	bundle, err := r5.NewDocument(&r5.Composition{},
		r5.DocumentEntry{FullURL: "urn:uuid:pat-1", Resource: &r5.Patient{}},
	)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	if bundle.Type == nil || *bundle.Type != r5.BundleTypeDocument {
		t.Errorf("type = %v, want document", bundle.Type)
	}
	if len(bundle.Entry) != 2 {
		t.Fatalf("entry count = %d, want 2", len(bundle.Entry))
	}
	first, ok := fhir.As[*r5.Composition](*bundle.Entry[0].Resource)
	if !ok || first == nil {
		t.Errorf("first entry is not a Composition")
	}
}

func TestNewDocumentRejectsNonComposition(t *testing.T) {
	_, err := r5.NewDocument(&r5.Patient{})
	if !errors.Is(err, r5.ErrInvalidBundle) {
		t.Fatalf("err = %v, want ErrInvalidBundle", err)
	}
	if !strings.Contains(err.Error(), "Composition") {
		t.Errorf("error should name Composition, got %q", err)
	}
}

func TestNewDocumentRejectsNilComposition(t *testing.T) {
	_, err := r5.NewDocument(nil)
	if !errors.Is(err, r5.ErrInvalidBundle) {
		t.Fatalf("err = %v, want ErrInvalidBundle", err)
	}
	// A typed-nil pointer must also be rejected, not dereferenced.
	var typedNil *r5.Composition
	_, err = r5.NewDocument(typedNil)
	if !errors.Is(err, r5.ErrInvalidBundle) {
		t.Fatalf("typed-nil err = %v, want ErrInvalidBundle", err)
	}
}

func TestNewMessageRequiresMessageHeaderFirst(t *testing.T) {
	bundle, err := r5.NewMessage(&r5.MessageHeader{},
		r5.MessageEntry{FullURL: "urn:uuid:pat-1", Resource: &r5.Patient{}},
	)
	if err != nil {
		t.Fatalf("NewMessage: %v", err)
	}
	if bundle.Type == nil || *bundle.Type != r5.BundleTypeMessage {
		t.Errorf("type = %v, want message", bundle.Type)
	}
	first, ok := fhir.As[*r5.MessageHeader](*bundle.Entry[0].Resource)
	if !ok || first == nil {
		t.Errorf("first entry is not a MessageHeader")
	}
}

func TestNewMessageRejectsNonMessageHeader(t *testing.T) {
	_, err := r5.NewMessage(&r5.Patient{})
	if !errors.Is(err, r5.ErrInvalidBundle) {
		t.Fatalf("err = %v, want ErrInvalidBundle", err)
	}
	if !strings.Contains(err.Error(), "MessageHeader") {
		t.Errorf("error should name MessageHeader, got %q", err)
	}
}

func TestDocumentRejectsDuplicateFullURL(t *testing.T) {
	_, err := r5.NewDocument(&r5.Composition{},
		r5.DocumentEntry{FullURL: "urn:uuid:dup", Resource: &r5.Patient{}},
		r5.DocumentEntry{FullURL: "urn:uuid:dup", Resource: &r5.Patient{}},
	)
	if !errors.Is(err, r5.ErrInvalidBundle) {
		t.Fatalf("err = %v, want ErrInvalidBundle", err)
	}
}

func TestNewMessageRejectsTypedNilHeader(t *testing.T) {
	var typedNil *r5.MessageHeader
	_, err := r5.NewMessage(typedNil)
	if !errors.Is(err, r5.ErrInvalidBundle) {
		t.Fatalf("typed-nil header err = %v, want ErrInvalidBundle", err)
	}
}

func TestContentBuildersRejectNilResource(t *testing.T) {
	var nilObs *r5.Observation

	if _, err := r5.NewSearchSet(0, r5.SearchEntry{Resource: nilObs}); !errors.Is(err, r5.ErrInvalidBundle) {
		t.Errorf("NewSearchSet nil resource err = %v, want ErrInvalidBundle", err)
	}
	if _, err := r5.NewCollection(r5.CollectionEntry{Resource: nilObs}); !errors.Is(err, r5.ErrInvalidBundle) {
		t.Errorf("NewCollection nil resource err = %v, want ErrInvalidBundle", err)
	}
	if _, err := r5.NewCollection(r5.CollectionEntry{}); !errors.Is(err, r5.ErrInvalidBundle) {
		t.Errorf("NewCollection absent resource err = %v, want ErrInvalidBundle", err)
	}
	if _, err := r5.NewDocument(&r5.Composition{}, r5.DocumentEntry{Resource: nilObs}); !errors.Is(err, r5.ErrInvalidBundle) {
		t.Errorf("NewDocument nil trailing resource err = %v, want ErrInvalidBundle", err)
	}
	if _, err := r5.NewMessage(&r5.MessageHeader{}, r5.MessageEntry{Resource: nilObs}); !errors.Is(err, r5.ErrInvalidBundle) {
		t.Errorf("NewMessage nil trailing resource err = %v, want ErrInvalidBundle", err)
	}
}

func TestSearchSetRejectsDuplicateFullURL(t *testing.T) {
	_, err := r5.NewSearchSet(2,
		r5.SearchEntry{FullURL: "urn:uuid:dup", Resource: &r5.Observation{}},
		r5.SearchEntry{FullURL: "urn:uuid:dup", Resource: &r5.Observation{}},
	)
	if !errors.Is(err, r5.ErrInvalidBundle) {
		t.Fatalf("err = %v, want ErrInvalidBundle", err)
	}
}

func TestCollectionRejectsDuplicateFullURL(t *testing.T) {
	_, err := r5.NewCollection(
		r5.CollectionEntry{FullURL: "urn:uuid:dup", Resource: &r5.Observation{}},
		r5.CollectionEntry{FullURL: "urn:uuid:dup", Resource: &r5.Observation{}},
	)
	if !errors.Is(err, r5.ErrInvalidBundle) {
		t.Fatalf("err = %v, want ErrInvalidBundle", err)
	}
}
