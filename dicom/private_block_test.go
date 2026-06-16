package dicom

import (
	"bytes"
	"testing"
)

// privateGroup is an odd group used across the private-block tests.
const privateGroup uint16 = 0x0009

// buildPrivateDataSet returns a dataset carrying the "ACME 3.1" private creator at
// (0009,0010) and three data elements in its block (0009,1000..1002).
func buildPrivateDataSet() *DataSet {
	ds := NewDataSet()
	// A private-creator element is VR LO (PS3.5 §7.8.1); set it explicitly so it
	// round-trips as text rather than the VRUN dictVR fallback for unknown tags.
	ds.Set(Element{Tag: NewTag(privateGroup, 0x0010), VR: VRLO, Value: NewStrings(VRLO, "ACME 3.1")})
	ds.Set(Element{Tag: NewTag(privateGroup, 0x1001), VR: VRSH, Value: NewStrings(VRSH, "alpha")})
	ds.Set(Element{Tag: NewTag(privateGroup, 0x1002), VR: VRLO, Value: NewStrings(VRLO, "beta value")})
	ds.Set(Element{Tag: NewTag(privateGroup, 0x1003), VR: VRDS, Value: NewStrings(VRDS, "1.5")})
	return ds
}

func TestPrivateBlockResolvesExistingCreator(t *testing.T) {
	ds := buildPrivateDataSet()

	block, ok := ds.PrivateBlock(privateGroup, "ACME 3.1", false)
	if !ok {
		t.Fatalf("PrivateBlock: creator not resolved")
	}
	if got, want := block.BlockStart(), uint16(0x1000); got != want {
		t.Fatalf("BlockStart = %#04x, want %#04x", got, want)
	}
	if got, want := block.Tag(0x01), NewTag(privateGroup, 0x1001); got != want {
		t.Fatalf("Tag(0x01) = %s, want %s", got, want)
	}

	s, ok := block.GetString(0x01)
	if !ok || s != "alpha" {
		t.Fatalf("block.GetString(0x01) = %q, %v; want \"alpha\", true", s, ok)
	}
	s, ok = block.GetString(0x02)
	if !ok || s != "beta value" {
		t.Fatalf("block.GetString(0x02) = %q, %v; want \"beta value\", true", s, ok)
	}
}

func TestPrivateBlockAbsentCreator(t *testing.T) {
	ds := buildPrivateDataSet()
	if _, ok := ds.PrivateBlock(privateGroup, "OTHER VENDOR", false); ok {
		t.Fatalf("PrivateBlock resolved an absent creator")
	}
	if _, ok := ds.PrivateBlock(0x0008, "ACME 3.1", false); ok {
		t.Fatalf("PrivateBlock resolved an even (non-private) group")
	}
}

func TestPrivateCreatorsLists(t *testing.T) {
	ds := buildPrivateDataSet()
	ds.SetString(NewTag(privateGroup, 0x0011), "SECOND CREATOR")

	creators := ds.PrivateCreators(privateGroup)
	if len(creators) != 2 {
		t.Fatalf("PrivateCreators = %v, want 2 entries", creators)
	}
	if creators[0] != "ACME 3.1" || creators[1] != "SECOND CREATOR" {
		t.Fatalf("PrivateCreators = %v, want [ACME 3.1, SECOND CREATOR] in block order", creators)
	}
	if got := ds.PrivateCreators(0x0008); got != nil {
		t.Fatalf("PrivateCreators(even group) = %v, want nil", got)
	}
}

func TestGetPrivateItem(t *testing.T) {
	ds := buildPrivateDataSet()

	el, ok := ds.GetPrivateItem(privateGroup, 0x02, "ACME 3.1")
	if !ok {
		t.Fatalf("GetPrivateItem: not found")
	}
	if el.Tag != NewTag(privateGroup, 0x1002) {
		t.Fatalf("GetPrivateItem tag = %s, want (0009,1002)", el.Tag)
	}
	if el.VR != VRLO {
		t.Fatalf("GetPrivateItem VR = %s, want LO", el.VR)
	}

	if _, ok := ds.GetPrivateItem(privateGroup, 0x7F, "ACME 3.1"); ok {
		t.Fatalf("GetPrivateItem resolved an absent offset")
	}
}

func TestPrivateBlockCreateReservesBlock(t *testing.T) {
	ds := NewDataSet()
	// Occupy block 0x10 with a different creator so create must pick 0x11.
	ds.SetString(NewTag(privateGroup, 0x0010), "FIRST")

	block, ok := ds.PrivateBlock(privateGroup, "ACME 3.1", true)
	if !ok {
		t.Fatalf("PrivateBlock create=true failed")
	}
	if got, want := block.BlockStart(), uint16(0x1100); got != want {
		t.Fatalf("BlockStart = %#04x, want %#04x (lowest free block)", got, want)
	}

	creatorEl, ok := ds.GetString(NewTag(privateGroup, 0x0011))
	if !ok || creatorEl != "ACME 3.1" {
		t.Fatalf("creator element not written: %q, %v", creatorEl, ok)
	}

	block.SetString(0x05, VRSH, "gamma")
	got, ok := ds.GetString(NewTag(privateGroup, 0x1105))
	if !ok || got != "gamma" {
		t.Fatalf("block.SetString did not land at (0009,1105): %q, %v", got, ok)
	}

	// Re-resolving an existing creator must not allocate a new block.
	again, ok := ds.PrivateBlock(privateGroup, "ACME 3.1", true)
	if !ok || again.BlockStart() != 0x1100 {
		t.Fatalf("re-resolve reserved a new block: %#04x", again.BlockStart())
	}
}

func TestPrivateDictLookup(t *testing.T) {
	ds := buildPrivateDataSet()
	block, ok := ds.PrivateBlock(privateGroup, "ACME 3.1", false)
	if !ok {
		t.Fatalf("PrivateBlock failed")
	}

	info, ok := block.Lookup(0x01)
	if !ok {
		t.Fatalf("block.Lookup(0x01): seeded entry not found")
	}
	if info.VR != VRSH || info.Keyword != "ACMEPrivateData01" {
		t.Fatalf("Lookup(0x01) = %+v, want VR=SH Keyword=ACMEPrivateData01", info)
	}

	if _, ok := LookupPrivate("ACME 3.1", privateGroup, 0x03); !ok {
		t.Fatalf("LookupPrivate offset 0x03 not seeded")
	}
	if _, ok := LookupPrivate("ACME 3.1", 0x0008, 0x01); ok {
		t.Fatalf("LookupPrivate resolved an even group")
	}
	if _, ok := LookupPrivate("UNKNOWN VENDOR", privateGroup, 0x01); ok {
		t.Fatalf("LookupPrivate resolved an unseeded creator")
	}
}

func TestPrivateBlockRoundTrip(t *testing.T) {
	ds := buildPrivateDataSet()

	var buf bytes.Buffer
	if err := EncodeDataSet(&buf, ds, ExplicitVRLittleEndian); err != nil {
		t.Fatalf("EncodeDataSet: %v", err)
	}
	got, err := DecodeDataSet(bytes.NewReader(buf.Bytes()), ExplicitVRLittleEndian)
	if err != nil {
		t.Fatalf("DecodeDataSet: %v", err)
	}

	block, ok := got.PrivateBlock(privateGroup, "ACME 3.1", false)
	if !ok {
		t.Fatalf("round-tripped dataset: creator not resolved")
	}
	if s, ok := block.GetString(0x01); !ok || s != "alpha" {
		t.Fatalf("round-trip offset 0x01 = %q, %v; want \"alpha\"", s, ok)
	}
	if s, ok := block.GetString(0x02); !ok || s != "beta value" {
		t.Fatalf("round-trip offset 0x02 = %q, %v; want \"beta value\"", s, ok)
	}
	if got, want := got.PrivateCreators(privateGroup), 1; len(got) != want {
		t.Fatalf("round-trip PrivateCreators = %v, want %d entry", got, want)
	}
}
