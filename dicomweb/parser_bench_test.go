package dicomweb

import (
	"bytes"
	"io"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
)

// benchMetadataDataSet builds a metadata-shaped dataset roughly the size a QIDO-RS result
// row or a WADO-RS metadata instance carries: study/series/instance identity, the common
// patient and study attributes, image geometry, and a one-item Referenced Image Sequence so
// the SQ marshal/unmarshal path is on the hot path. Every value is synthetic; the patient
// name is the obvious sentinel, never real PHI.
func benchMetadataDataSet() *dicom.DataSet {
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x0008, 0x0018), VR: dicom.VRUI, Value: dicom.NewStrings(dicom.VRUI, "1.2.840.10008.1.2.1.99.1")})
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x0008, 0x0016), VR: dicom.VRUI, Value: dicom.NewStrings(dicom.VRUI, "1.2.840.10008.5.1.4.1.1.2")})
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x0020, 0x000D), VR: dicom.VRUI, Value: dicom.NewStrings(dicom.VRUI, "1.2.840.10008.1.2.1.99.2")})
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x0020, 0x000E), VR: dicom.VRUI, Value: dicom.NewStrings(dicom.VRUI, "1.2.840.10008.1.2.1.99.3")})
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x0010, 0x0010), VR: dicom.VRPN, Value: dicom.NewStrings(dicom.VRPN, "ZZZTEST^SENTINEL")})
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x0010, 0x0020), VR: dicom.VRLO, Value: dicom.NewStrings(dicom.VRLO, "SYNTH-0001")})
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x0008, 0x0020), VR: dicom.VRDA, Value: dicom.NewStrings(dicom.VRDA, "20240101")})
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x0008, 0x0060), VR: dicom.VRCS, Value: dicom.NewStrings(dicom.VRCS, "CT")})
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x0028, 0x0010), VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, 512)})
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x0028, 0x0011), VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, 512)})
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x0028, 0x0030), VR: dicom.VRDS, Value: dicom.NewDecimals(dicom.VRDS, mustDecimal("0.5"), mustDecimal("0.5"))})

	item := dicom.NewDataSet()
	item.Set(dicom.Element{Tag: dicom.NewTag(0x0008, 0x1150), VR: dicom.VRUI, Value: dicom.NewStrings(dicom.VRUI, "1.2.840.10008.5.1.4.1.1.2")})
	item.Set(dicom.Element{Tag: dicom.NewTag(0x0008, 0x1155), VR: dicom.VRUI, Value: dicom.NewStrings(dicom.VRUI, "1.2.840.10008.1.2.1.99.4")})
	seq := dicom.NewSequence()
	seq.Append(item)
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x0008, 0x1140), VR: dicom.VRSQ, Value: dicom.NewSequenceValue(seq)})

	return ds
}

// mustDecimal parses a DS lexical form for the benchmark fixtures, panicking on a malformed
// literal because the inputs are compile-time constants in the benchmark itself.
func mustDecimal(s string) dicom.Decimal {
	d, err := dicom.ParseDecimal(s)
	if err != nil {
		panic("benchMetadataDataSet: bad decimal literal " + s)
	}
	return d
}

// BenchmarkMarshalJSON measures encoding a metadata-shaped dataset to DICOM JSON, the
// per-instance encode the QIDO-RS result and WADO-RS metadata paths run. SetBytes reports
// the throughput and ReportAllocs the encode allocation profile (PRD §9.3) against the
// committed baseline.
func BenchmarkMarshalJSON(b *testing.B) {
	ds := benchMetadataDataSet()
	encoded, err := MarshalJSON(ds)
	if err != nil {
		b.Fatalf("warm-up MarshalJSON: %v", err)
	}
	b.SetBytes(int64(len(encoded)))
	b.ReportAllocs()
	for b.Loop() {
		if _, err := MarshalJSON(ds); err != nil {
			b.Fatalf("MarshalJSON: %v", err)
		}
	}
}

// BenchmarkUnmarshalJSON measures decoding a metadata-shaped DICOM-JSON document back into a
// dataset, the trust-boundary decode every QIDO/WADO/STOW JSON payload passes through. The
// bytes are produced once outside the timed loop; SetBytes reports throughput and
// ReportAllocs the decode allocation profile.
func BenchmarkUnmarshalJSON(b *testing.B) {
	encoded, err := MarshalJSON(benchMetadataDataSet())
	if err != nil {
		b.Fatalf("encode bench dataset: %v", err)
	}
	b.SetBytes(int64(len(encoded)))
	b.ReportAllocs()
	for b.Loop() {
		if _, err := UnmarshalJSON(encoded); err != nil {
			b.Fatalf("UnmarshalJSON: %v", err)
		}
	}
}

// benchMultipartParts is the part count of the multipart/related body the multipart
// benchmarks iterate. It is sized like a small WADO-RS series retrieve — several instances
// returned as separate parts — so the per-part header parse and bounded-body drain dominate
// over fixed setup.
const benchMultipartParts = 16

// benchPartBody is one part's payload: a kilobyte of synthetic octets standing in for a
// small Part 10 object or frame fragment.
var benchPartBody = bytes.Repeat([]byte{0xAB}, 1024)

// benchMultipartBody assembles a multipart/related body of benchMultipartParts parts and
// returns it with its media type, both built once for the read benchmark.
func benchMultipartBody(b *testing.B) (body []byte, mediaType string) {
	b.Helper()
	var buf bytes.Buffer
	mw := NewMultipartWriter(&buf, mediaTypeDICOM)
	for range benchMultipartParts {
		if err := mw.AddPart(mediaTypeDICOM, bytes.NewReader(benchPartBody)); err != nil {
			b.Fatalf("AddPart: %v", err)
		}
	}
	boundary, err := mw.Close()
	if err != nil {
		b.Fatalf("Close: %v", err)
	}
	return buf.Bytes(), `multipart/related; type="` + mediaTypeDICOM + `"; boundary="` + boundary + `"`
}

// BenchmarkMultipartRead measures iterating a multipart/related body part-by-part through
// the bounded reader, the WADO-RS retrieve and STOW-RS store hot path: each NextPart parses
// a part header and each body is drained through the per-part cap. SetBytes reports
// throughput and ReportAllocs the per-part allocation profile against the committed baseline.
func BenchmarkMultipartRead(b *testing.B) {
	body, mediaType := benchMultipartBody(b)
	b.SetBytes(int64(len(body)))
	b.ReportAllocs()
	for b.Loop() {
		mr, err := NewMultipartReader(bytes.NewReader(body), mediaType)
		if err != nil {
			b.Fatalf("NewMultipartReader: %v", err)
		}
		for {
			_, partBody, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				b.Fatalf("NextPart: %v", err)
			}
			if _, err := io.Copy(io.Discard, partBody); err != nil {
				b.Fatalf("drain part: %v", err)
			}
		}
	}
}
