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

// part10ItemCharsetStream builds a Part 10 stream whose top level uses ISO_IR 100
// while the first sequence item carries its own (0008,0005) = ISO 2022 IR 87
// (PS3.5 §7.5.3). Item 1 holds the Annex H.3 Japanese PN; item 2 has no charset
// element of its own; Latin-1 PN elements sit before and after the sequence.
func part10ItemCharsetStream(t *testing.T) []byte {
	t.Helper()
	latin1Name := []byte{0xc4, 'n', 'e', 0xe4, 's'} // "Äneäs" in Latin-1

	var main bytes.Buffer
	main.Write(rawElement(TagSpecificCharacterSet, VRCS, []byte("ISO_IR 100")))
	main.Write(rawElement(NewTag(0x0008, 0x0090), VRPN, latin1Name))
	main.Write(sqHeaderUndefinedLE(NewTag(0x0008, 0x1120)))
	main.Write(itemHeaderUndefinedLE())
	main.Write(rawElement(TagSpecificCharacterSet, VRCS, []byte("\\ISO 2022 IR 87")))
	main.Write(rawElement(NewTag(0x0010, 0x0010), VRPN, canonicalIR87Name))
	main.Write(leTag(0xFFFE, 0xE00D))
	main.Write(le32(0))
	main.Write(itemHeaderUndefinedLE())
	main.Write(rawElement(NewTag(0x0010, 0x0010), VRPN, latin1Name))
	main.Write(leTag(0xFFFE, 0xE00D))
	main.Write(le32(0))
	main.Write(seqDelim())
	main.Write(rawElement(NewTag(0x0010, 0x0010), VRPN, latin1Name))
	return part10WithRawMain(t, main.Bytes())
}

func TestReaderSequenceItemCharsetGovernsOnlyItsItem(t *testing.T) {
	f, err := Read(bytes.NewReader(part10ItemCharsetStream(t)))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	seq, ok := f.DataSet.GetSequence(NewTag(0x0008, 0x1120))
	if !ok {
		t.Fatal("sequence (0008,1120) absent")
	}
	if seq.Len() != 2 {
		t.Fatalf("sequence has %d items, want 2", seq.Len())
	}

	var items []Item
	for item := range seq.Items() {
		items = append(items, item)
	}

	// Item 1 decodes under its own ISO 2022 IR 87, not the top-level Latin-1.
	pn, ok := items[0].DataSet.GetPersonName(NewTag(0x0010, 0x0010))
	if !ok {
		t.Fatal("item 1 PatientName absent")
	}
	if pn.Alphabetic.FamilyName != "Yamada" || pn.Ideographic.FamilyName != "山田" || pn.Phonetic.FamilyName != "やまだ" {
		t.Errorf("item 1 PN = %+v, want Yamada/山田/やまだ family names via the item's IR 87 charset", pn)
	}

	// Item 2 carries no (0008,0005), so it inherits the enclosing dataset's Latin-1.
	got, _ := items[1].DataSet.GetString(NewTag(0x0010, 0x0010))
	if got != "Äneäs" {
		t.Errorf("item 2 PatientName = %q, want Äneäs via the inherited top-level charset", got)
	}
}

func TestReaderSequenceItemCharsetDoesNotLeakToTopLevel(t *testing.T) {
	f, err := Read(bytes.NewReader(part10ItemCharsetStream(t)))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	// Before the sequence: top-level Latin-1.
	got, _ := f.DataSet.GetString(NewTag(0x0008, 0x0090))
	if got != "Äneäs" {
		t.Errorf("ReferringPhysicianName = %q, want Äneäs", got)
	}

	// After the sequence: still top-level Latin-1, untouched by item 1's IR 87.
	got, _ = f.DataSet.GetString(NewTag(0x0010, 0x0010))
	if got != "Äneäs" {
		t.Errorf("top-level PatientName after the sequence = %q, want Äneäs (item charset leaked)", got)
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
