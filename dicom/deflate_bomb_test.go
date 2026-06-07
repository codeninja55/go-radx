package dicom

import (
	"bytes"
	"compress/flate"
	"errors"
	"io"
	"testing"
	"time"
)

// deflateBombMainDataSet returns the raw, uncompressed Explicit VR LE bytes of a bare
// main dataset large enough to exceed a small inflated-bytes budget when fed through the
// deflated read path. The elements carry synthetic sentinels, never real PHI, and span
// distinct tags so each survives DataSet.Set and the encoded stream grows with the count.
func deflateBombMainDataSet(t *testing.T, elements int) []byte {
	t.Helper()
	ds := NewDataSet()
	for i := 0; i < elements; i++ {
		// (0009,xxxx) is a private-ish odd group; the value is a fixed sentinel so the
		// uncompressed bytes are highly compressible (a tiny DEFLATE stream inflates into
		// this long run) yet every element is structurally valid.
		tag := NewTag(0x0009, uint16(i))
		ds.Set(Element{Tag: tag, VR: VRLO, Value: NewStrings(VRLO, "ZZZSENTINEL")})
	}
	var raw bytes.Buffer
	if err := writeDataSet(&raw, ds, ExplicitVRLittleEndian); err != nil {
		t.Fatalf("writeDataSet: %v", err)
	}
	return raw.Bytes()
}

// deflate compresses raw as a raw DEFLATE stream, the encoding the Deflated Explicit VR
// LE main dataset uses after the file-meta group.
func deflate(t *testing.T, raw []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	fw, err := flate.NewWriter(&out, flate.BestCompression)
	if err != nil {
		t.Fatalf("flate.NewWriter: %v", err)
	}
	if _, err := fw.Write(raw); err != nil {
		t.Fatalf("flate write: %v", err)
	}
	if err := fw.Close(); err != nil {
		t.Fatalf("flate close: %v", err)
	}
	return out.Bytes()
}

// deflateBombPart10 assembles a Part 10 stream whose Deflated Explicit VR LE main
// dataset inflates well past a small budget: a real file-meta group followed by a raw
// DEFLATE stream of a long run of valid elements.
func deflateBombPart10(t *testing.T, elements int) []byte {
	t.Helper()
	meta := &FileMeta{
		MediaStorageSOPClassUID:    "1.2.840.10008.5.1.4.1.1.7",
		MediaStorageSOPInstanceUID: "1.2.3.4.5.6.7.8.9",
		TransferSyntaxUID:          DeflatedExplicitVRLittleEndian,
	}
	var out bytes.Buffer
	if err := writeFileMeta(&out, [128]byte{}, meta); err != nil {
		t.Fatalf("writeFileMeta: %v", err)
	}
	out.Write(deflate(t, deflateBombMainDataSet(t, elements)))
	return out.Bytes()
}

func TestReadDeflateBombReturnsLimitExceeded(t *testing.T) {
	bomb := deflateBombPart10(t, 4000)
	// A few KiB of inflated dataset against a 1 KiB budget: enough to trip the bound but
	// small enough that the test allocates nothing close to the default 4 GiB.
	const budget = 1 << 10

	done := make(chan error, 1)
	go func() {
		_, err := Read(bytes.NewReader(bomb), WithMaxInflatedBytes(budget))
		done <- err
	}()

	select {
	case err := <-done:
		var lim *LimitExceededError
		if !errors.As(err, &lim) {
			t.Fatalf("Read deflate bomb: got %v, want *LimitExceededError", err)
		}
		if lim.Kind != "inflated-bytes" {
			t.Errorf("LimitExceededError.Kind = %q, want %q", lim.Kind, "inflated-bytes")
		}
		if lim.Limit != budget {
			t.Errorf("LimitExceededError.Limit = %d, want %d", lim.Limit, budget)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Read deflate bomb did not return within 10s: the inflated-bytes bound did not fire")
	}
}

func TestDecodeDataSetDeflateBombReturnsLimitExceeded(t *testing.T) {
	bomb := deflate(t, deflateBombMainDataSet(t, 4000))
	const budget = 1 << 10

	done := make(chan error, 1)
	go func() {
		_, err := DecodeDataSet(bytes.NewReader(bomb), DeflatedExplicitVRLittleEndian, WithMaxInflatedBytes(budget))
		done <- err
	}()

	select {
	case err := <-done:
		var lim *LimitExceededError
		if !errors.As(err, &lim) {
			t.Fatalf("DecodeDataSet deflate bomb: got %v, want *LimitExceededError", err)
		}
		if lim.Kind != "inflated-bytes" {
			t.Errorf("LimitExceededError.Kind = %q, want %q", lim.Kind, "inflated-bytes")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("DecodeDataSet deflate bomb did not return within 10s: the inflated-bytes bound did not fire")
	}
}

// TestInflateLimitReaderStopsAtBudget asserts the limiter reads at most budget+1 bytes
// from the source before reporting the typed error, so a hostile stream cannot drive an
// unbounded read before the bound fires.
func TestInflateLimitReaderStopsAtBudget(t *testing.T) {
	const budget = 64
	src := &countingReader{data: bytes.Repeat([]byte{0xAB}, 4096)}
	lr := newInflateLimitReader(src, budget)

	var total int
	buf := make([]byte, 32)
	for {
		n, err := lr.Read(buf)
		total += n
		if err != nil {
			var lim *LimitExceededError
			if !errors.As(err, &lim) {
				t.Fatalf("inflateLimitReader: got %v, want *LimitExceededError", err)
			}
			break
		}
	}
	if total != budget {
		t.Errorf("delivered %d inflated bytes, want exactly the budget %d", total, budget)
	}
	if src.read > budget+1 {
		t.Errorf("read %d source bytes, want at most budget+1 = %d", src.read, budget+1)
	}
}

// TestInflateLimitReaderAcceptsExactBudget confirms a stream whose inflated size equals the
// budget exactly ends with a clean io.EOF, not a LimitExceededError. The dataset parser
// issues one more read after the last element to observe EOF; that probe must not be misread
// as a byte beyond the budget, which would reject a valid object sized exactly at the cap.
func TestInflateLimitReaderAcceptsExactBudget(t *testing.T) {
	const budget = 64
	src := &countingReader{data: bytes.Repeat([]byte{0xAB}, budget)}
	lr := newInflateLimitReader(src, budget)

	var total int
	buf := make([]byte, 32)
	for {
		n, err := lr.Read(buf)
		total += n
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			var lim *LimitExceededError
			if errors.As(err, &lim) {
				t.Fatalf("exact-budget stream wrongly rejected as over-limit: %v", err)
			}
			t.Fatalf("inflateLimitReader: got %v, want io.EOF", err)
		}
	}
	if total != budget {
		t.Errorf("delivered %d inflated bytes, want exactly the budget %d", total, budget)
	}
}

// countingReader records how many bytes were read from it.
type countingReader struct {
	data []byte
	pos  int
	read int
}

func (c *countingReader) Read(p []byte) (int, error) {
	if c.pos >= len(c.data) {
		return 0, io.EOF
	}
	n := copy(p, c.data[c.pos:])
	c.pos += n
	c.read += n
	return n, nil
}

// TestReadDeflatedRoundTripUnderBudgetSucceeds confirms a legitimate deflated object
// round-trips when its inflated dataset stays under the budget: the bomb guard must not
// reject valid deflated studies.
func TestReadDeflatedRoundTripUnderBudgetSucceeds(t *testing.T) {
	f := sampleFile(DeflatedExplicitVRLittleEndian)
	var buf bytes.Buffer
	if err := Write(&buf, f); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(bytes.NewReader(buf.Bytes()), WithMaxInflatedBytes(1<<20))
	if err != nil {
		t.Fatalf("Read legitimate deflated object under budget: %v", err)
	}
	if v, ok := got.DataSet.GetString(NewTag(0x0010, 0x0010)); !ok || v != "Doe^Jane" {
		t.Errorf("round-trip PatientName = %q,%v", v, ok)
	}
}
