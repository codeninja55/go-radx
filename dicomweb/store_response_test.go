package dicomweb

import (
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
)

func TestStoreResponseIsComplete(t *testing.T) {
	complete := &StoreResponse{Referenced: []StoredInstance{{}}}
	if !complete.IsComplete() {
		t.Fatal("IsComplete() = false for a response with no failures, want true")
	}
	partial := &StoreResponse{
		Referenced: []StoredInstance{{}},
		Failed:     []dicom.FailedSOPInstance{{FailureReason: 0xA700}},
	}
	if partial.IsComplete() {
		t.Fatal("IsComplete() = true for a response with a failure, want false")
	}
	var nilResp *StoreResponse
	if nilResp.IsComplete() {
		t.Fatal("IsComplete() = true for nil, want false")
	}
}

func TestParseStoreResponse(t *testing.T) {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagRetrieveURL, "https://pacs.example.org/dicom-web/studies/1.2.3")

	ref := dicom.NewDataSet()
	ref.SetString(dicom.TagReferencedSOPClassUID, "1.2.840.10008.5.1.4.1.1.4")
	ref.SetString(dicom.TagReferencedSOPInstanceUID, "1.2.3.4.5")
	ref.SetString(dicom.TagRetrieveURL, "https://pacs.example.org/dicom-web/studies/1.2.3/series/1.2.3.4/instances/1.2.3.4.5")
	ds.Set(dicom.Element{
		Tag:   dicom.TagReferencedSOPSequence,
		VR:    dicom.VRSQ,
		Value: dicom.NewSequenceValue(dicom.NewSequence(ref)),
	})

	failed := dicom.NewDataSet()
	failed.SetString(dicom.TagReferencedSOPClassUID, "1.2.840.10008.5.1.4.1.1.4")
	failed.SetString(dicom.TagReferencedSOPInstanceUID, "1.2.3.4.6")
	failed.Set(dicom.Element{Tag: dicom.TagFailureReason, VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, 0xA700)})
	ds.Set(dicom.Element{
		Tag:   dicom.TagFailedSOPSequence,
		VR:    dicom.VRSQ,
		Value: dicom.NewSequenceValue(dicom.NewSequence(failed)),
	})

	resp := parseStoreResponse(ds)
	if resp.RetrieveURL != "https://pacs.example.org/dicom-web/studies/1.2.3" {
		t.Fatalf("RetrieveURL = %q", resp.RetrieveURL)
	}
	if len(resp.Referenced) != 1 {
		t.Fatalf("Referenced len = %d, want 1", len(resp.Referenced))
	}
	if resp.Referenced[0].SOPInstanceUID != "1.2.3.4.5" {
		t.Fatalf("Referenced SOPInstanceUID = %q", resp.Referenced[0].SOPInstanceUID)
	}
	if len(resp.Failed) != 1 {
		t.Fatalf("Failed len = %d, want 1", len(resp.Failed))
	}
	if resp.Failed[0].FailureReason != 0xA700 {
		t.Fatalf("Failed FailureReason = 0x%04X, want 0xA700", resp.Failed[0].FailureReason)
	}
	if resp.IsComplete() {
		t.Fatal("IsComplete() = true with a failed instance")
	}
}

func TestStoreErrorNamesReasonWithoutPHI(t *testing.T) {
	err := &StoreError{
		Failed:   []dicom.FailedSOPInstance{{FailureReason: 0xA700}},
		Accepted: 1,
	}
	msg := err.Error()
	if !strings.Contains(msg, "out of resources") {
		t.Fatalf("StoreError message %q does not name the failure reason", msg)
	}
	if !strings.Contains(msg, "1 of 2") {
		t.Fatalf("StoreError message %q does not report the failed/total counts", msg)
	}
}

func TestFailureReasonNameUnknown(t *testing.T) {
	if got := failureReasonName(0x1234); !strings.Contains(got, "0x1234") {
		t.Fatalf("failureReasonName(0x1234) = %q, want the hex code", got)
	}
}
