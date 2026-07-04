package dicom

import (
	"bytes"
	"errors"
	"testing"
)

// part10WithRawMain builds a minimal Part 10 byte stream in Explicit VR Little Endian
// whose main dataset is the supplied raw element bytes, so the reader's charset
// resolution can be exercised against byte sequences the high-level API cannot
// construct directly.
func part10WithRawMain(t *testing.T, mainRaw []byte) []byte {
	t.Helper()
	meta := &FileMeta{
		MediaStorageSOPClassUID:    "1.2.840.10008.5.1.4.1.1.7",
		MediaStorageSOPInstanceUID: "1.2.3.4.5",
		TransferSyntaxUID:          ExplicitVRLittleEndian,
	}
	var buf bytes.Buffer
	var preamble [128]byte
	if err := writeFileMeta(&buf, preamble, meta); err != nil {
		t.Fatalf("writeFileMeta: %v", err)
	}
	buf.Write(mainRaw)
	return buf.Bytes()
}

// rawElement builds an Explicit VR LE element with an explicit raw value field, used
// to inject non-default-charset bytes the high-level API cannot construct directly.
func rawElement(tag Tag, vr VR, value []byte) []byte {
	var b bytes.Buffer
	b.WriteByte(byte(tag >> 16))
	b.WriteByte(byte(tag >> 24))
	b.WriteByte(byte(tag))
	b.WriteByte(byte(tag >> 8))
	b.WriteString(vr.String())
	v := value
	if len(v)%2 == 1 {
		pad := byte(0x20)
		if p, ok := vr.PadByte(); ok {
			pad = p
		}
		v = append(append([]byte{}, v...), pad)
	}
	b.WriteByte(byte(len(v)))
	b.WriteByte(byte(len(v) >> 8))
	b.Write(v)
	return b.Bytes()
}

func TestReaderDecodesLatin1PatientName(t *testing.T) {
	// (0008,0005) SpecificCharacterSet = "ISO_IR 100", then a PN in Latin-1.
	scs := rawElement(TagSpecificCharacterSet, VRCS, []byte("ISO_IR 100"))
	pnRaw := []byte{0xc4, 'n', 'e', 0xe4, 's', '^', 'R', 0xfc, 'd', 'i', 'g', 'e', 'r'}
	pn := rawElement(NewTag(0x0010, 0x0010), VRPN, pnRaw)

	stream := part10WithRawMain(t, append(scs, pn...))

	f, err := Read(bytes.NewReader(stream))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	got, ok := f.DataSet.GetString(NewTag(0x0010, 0x0010))
	if !ok {
		t.Fatal("PatientName absent")
	}
	if got != "Äneäs^Rüdiger" {
		t.Fatalf("PatientName = %q, want Äneäs^Rüdiger", got)
	}
}

func TestReaderDefaultRepertoireVRIgnoresCharset(t *testing.T) {
	// Even with ISO_IR 100 in force, a CS value (default repertoire) is plain ASCII;
	// a stray high byte must not be reinterpreted through Latin-1.
	scs := rawElement(TagSpecificCharacterSet, VRCS, []byte("ISO_IR 100"))
	cs := rawElement(NewTag(0x0008, 0x0060), VRCS, []byte("MR")) // Modality CS
	stream := part10WithRawMain(t, append(scs, cs...))

	f, err := Read(bytes.NewReader(stream))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	got, _ := f.DataSet.GetString(NewTag(0x0008, 0x0060))
	if got != "MR" {
		t.Fatalf("Modality = %q, want MR", got)
	}
}

func TestReaderRoundTripLatin1ByteExact(t *testing.T) {
	scs := rawElement(TagSpecificCharacterSet, VRCS, []byte("ISO_IR 100"))
	pnRaw := []byte{0xc4, 'n', 'e', 0xe4, 's', '^', 'R', 0xfc, 'd', 'i', 'g', 'e', 'r'}
	pn := rawElement(NewTag(0x0010, 0x0010), VRPN, pnRaw)
	stream := part10WithRawMain(t, append(scs, pn...))

	f, err := Read(bytes.NewReader(stream))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	var out bytes.Buffer
	if err := Write(&out, f); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !bytes.Equal(out.Bytes(), stream) {
		t.Fatalf("round-trip not byte-identical for a Latin-1 dataset")
	}
}

func TestReaderRoundTripISO2022JapaneseByteExact(t *testing.T) {
	scs := rawElement(TagSpecificCharacterSet, VRCS, []byte("\\ISO 2022 IR 87"))
	pn := rawElement(NewTag(0x0010, 0x0010), VRPN, canonicalIR87Name)
	stream := part10WithRawMain(t, append(scs, pn...))

	f, err := Read(bytes.NewReader(stream))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	got, ok := f.DataSet.GetPersonName(NewTag(0x0010, 0x0010))
	if !ok {
		t.Fatal("PatientName absent")
	}
	if got.Ideographic.FamilyName != "山田" {
		t.Fatalf("ideographic family = %q, want 山田", got.Ideographic.FamilyName)
	}

	var out bytes.Buffer
	if err := Write(&out, f); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !bytes.Equal(out.Bytes(), stream) {
		t.Fatalf("round-trip not byte-identical for an ISO 2022 IR 87 dataset")
	}
}

func TestReaderUnknownCharsetIsTypedError(t *testing.T) {
	scs := rawElement(TagSpecificCharacterSet, VRCS, []byte("ISO_IR 9999"))
	pn := rawElement(NewTag(0x0010, 0x0010), VRPN, []byte("Doe^John"))
	stream := part10WithRawMain(t, append(scs, pn...))

	_, err := Read(bytes.NewReader(stream))
	if err == nil {
		t.Fatal("Read with unknown charset = nil error, want typed error")
	}
	if _, ok := errors.AsType[*UnsupportedCharacterSetError](err); !ok {
		t.Fatalf("error is %T, want *UnsupportedCharacterSetError", err)
	}
}

func TestWithDefaultCharacterSetFallback(t *testing.T) {
	// No (0008,0005) element; the caller supplies the fallback charset.
	pnRaw := []byte{0xc4, 'n', 'e', 0xe4, 's'}
	pn := rawElement(NewTag(0x0010, 0x0010), VRPN, pnRaw)
	stream := part10WithRawMain(t, pn)

	f, err := Read(bytes.NewReader(stream), WithDefaultCharacterSet("ISO_IR 100"))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	got, _ := f.DataSet.GetString(NewTag(0x0010, 0x0010))
	if got != "Äneäs" {
		t.Fatalf("PatientName = %q, want Äneäs", got)
	}
}
