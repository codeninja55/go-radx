package dimse

import (
	"bytes"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
)

// BenchmarkCommandSet_Encode benchmarks DIMSE command encoding
func BenchmarkCommandSet_Encode(b *testing.B) {
	cmd := &CommandSet{
		CommandField:           CommandCStoreRQ,
		MessageID:              1,
		Priority:               PriorityMedium,
		CommandDataSetType:     DataSetPresent,
		AffectedSOPClassUID:    "1.2.840.10008.5.1.4.1.1.2",
		AffectedSOPInstanceUID: "1.2.840.113619.2.55.3.1.1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cmd.ToDataSet()
	}
}

// BenchmarkCommandSet_Decode benchmarks DIMSE command decoding
func BenchmarkCommandSet_Decode(b *testing.B) {
	cmd := &CommandSet{
		CommandField:        CommandCEchoRQ,
		MessageID:           1,
		CommandDataSetType:  DataSetNotPresent,
		AffectedSOPClassUID: "1.2.840.10008.1.1",
	}

	ds, _ := cmd.ToDataSet()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = FromDataSet(ds)
	}
}

// BenchmarkMessage_Encode benchmarks message fragmentation into PDUs
func BenchmarkMessage_Encode(b *testing.B) {
	pduSizes := []uint32{4096, 16384, 65536}

	for _, maxPDU := range pduSizes {
		b.Run(string(rune(maxPDU)), func(b *testing.B) {
			ds := dicom.NewDataSet()
			_ = ds.SetPatientName("Test^Patient")
			_ = ds.SetPatientID("12345")
			_ = ds.SetStudyInstanceUID("1.2.840.113619.2.55.3.1.1")

			msg := &Message{
				CommandSet: &CommandSet{
					CommandField:        CommandCStoreRQ,
					MessageID:           1,
					CommandDataSetType:  DataSetPresent,
					AffectedSOPClassUID: "1.2.840.10008.5.1.4.1.1.2",
				},
				DataSet:               ds,
				PresentationContextID: 1,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = msg.Encode(maxPDU)
			}
		})
	}
}

// BenchmarkReassembler_AddPDU benchmarks message reassembly
func BenchmarkReassembler_AddPDU(b *testing.B) {
	// Create a message and fragment it
	msg := &Message{
		CommandSet: &CommandSet{
			CommandField:        CommandCEchoRQ,
			MessageID:           1,
			CommandDataSetType:  DataSetNotPresent,
			AffectedSOPClassUID: "1.2.840.10008.1.1",
		},
		PresentationContextID: 1,
	}

	pdus, _ := msg.Encode(16384)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reassembler := NewMessageReassembler()
		for _, pdu := range pdus {
			_, _ = reassembler.AddPDU(pdu)
		}
	}
}

// BenchmarkMessage_LargeDataset benchmarks encoding large datasets
func BenchmarkMessage_LargeDataset(b *testing.B) {
	// Create a dataset with multiple elements
	ds := dicom.NewDataSet()
	_ = ds.SetPatientName("LargeDataset^Test^Patient^Name")
	_ = ds.SetPatientID("LARGE123456789")
	_ = ds.SetStudyInstanceUID("1.2.840.113619.2.55.3.987654321.100")
	_ = ds.SetSeriesInstanceUID("1.2.840.113619.2.55.3.987654321.200")

	msg := &Message{
		CommandSet: &CommandSet{
			CommandField:        CommandCStoreRQ,
			MessageID:           1,
			CommandDataSetType:  DataSetPresent,
			AffectedSOPClassUID: "1.2.840.10008.5.1.4.1.1.2",
		},
		DataSet:               ds,
		PresentationContextID: 1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = msg.Encode(16384)
	}
}

// BenchmarkCommandSet_RoundTrip benchmarks full encode/decode cycle
func BenchmarkCommandSet_RoundTrip(b *testing.B) {
	cmd := &CommandSet{
		CommandField:        CommandCFindRQ,
		MessageID:           42,
		Priority:            PriorityHigh,
		CommandDataSetType:  DataSetPresent,
		AffectedSOPClassUID: "1.2.840.10008.5.1.4.1.2.1.1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Encode
		ds, _ := cmd.ToDataSet()

		// Decode
		_, _ = FromDataSet(ds)
	}
}

// BenchmarkMessage_Fragmentation benchmarks fragmentation at boundary conditions
func BenchmarkMessage_Fragmentation(b *testing.B) {
	// Create data that will require fragmentation
	largeData := bytes.Repeat([]byte("DICOM"), 10000) // ~50KB
	ds := dicom.NewDataSet()
	_ = ds.SetPatientName(string(largeData))

	msg := &Message{
		CommandSet: &CommandSet{
			CommandField:        CommandCStoreRQ,
			MessageID:           1,
			CommandDataSetType:  DataSetPresent,
			AffectedSOPClassUID: "1.2.840.10008.5.1.4.1.1.2",
		},
		DataSet:               ds,
		PresentationContextID: 1,
	}

	b.Run("SmallPDU_4KB", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = msg.Encode(4096)
		}
	})

	b.Run("MediumPDU_16KB", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = msg.Encode(16384)
		}
	})

	b.Run("LargePDU_64KB", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = msg.Encode(65536)
		}
	})
}

// BenchmarkStatusCode_Validation benchmarks status code checking
func BenchmarkStatusCode_Validation(b *testing.B) {
	statuses := []uint16{
		StatusSuccess,
		StatusPending,
		StatusCancel,
		0xA700, // Error
		0xC000, // Failure
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		status := statuses[i%len(statuses)]
		_ = status == StatusSuccess || status == StatusPending
	}
}
