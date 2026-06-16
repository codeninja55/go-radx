package dicom

// VR is the two-letter Value Representation from PS3.5 Table 6.2-1.
type VR uint8

const (
	VRAE VR = iota // Application Entity
	VRAS           // Age String
	VRAT           // Attribute Tag
	VRCS           // Code String
	VRDA           // Date
	VRDS           // Decimal String
	VRDT           // Date Time
	VRFL           // Floating Point Single
	VRFD           // Floating Point Double
	VRIS           // Integer String
	VRLO           // Long String
	VRLT           // Long Text
	VROB           // Other Byte
	VROD           // Other Double
	VROF           // Other Float
	VROL           // Other Long
	VROV           // Other Very Long
	VROW           // Other Word
	VRPN           // Person Name
	VRSH           // Short String
	VRSL           // Signed Long
	VRSQ           // Sequence of Items
	VRSS           // Signed Short
	VRST           // Short Text
	VRSV           // Signed Very Long
	VRTM           // Time
	VRUC           // Unlimited Characters
	VRUI           // Unique Identifier
	VRUL           // Unsigned Long
	VRUN           // Unknown
	VRUR           // URI/URL
	VRUS           // Unsigned Short
	VRUT           // Unlimited Text
	VRUV           // Unsigned Very Long
)

// Ambiguous parse-time placeholders (PS3.6 dictionary VRs the reader resolves from
// context). They never appear on the wire.
const (
	VRUSorSS     VR = iota + 34 // "US or SS"
	VRUSorOW                    // "US or OW"
	VROBorOW                    // "OB or OW"
	VRUSorSSorOW                // "US or SS or OW"
)

var vrNames = [...]string{
	VRAE: "AE", VRAS: "AS", VRAT: "AT", VRCS: "CS", VRDA: "DA", VRDS: "DS",
	VRDT: "DT", VRFL: "FL", VRFD: "FD", VRIS: "IS", VRLO: "LO", VRLT: "LT",
	VROB: "OB", VROD: "OD", VROF: "OF", VROL: "OL", VROV: "OV", VROW: "OW",
	VRPN: "PN", VRSH: "SH", VRSL: "SL", VRSQ: "SQ", VRSS: "SS", VRST: "ST",
	VRSV: "SV", VRTM: "TM", VRUC: "UC", VRUI: "UI", VRUL: "UL", VRUN: "UN",
	VRUR: "UR", VRUS: "US", VRUT: "UT", VRUV: "UV",
	VRUSorSS: "US or SS", VRUSorOW: "US or OW", VROBorOW: "OB or OW",
	VRUSorSSorOW: "US or SS or OW",
}

// resolveExplicitVR maps an ambiguous dictionary placeholder to the concrete 2-letter
// VR an explicit-VR stream must carry. The placeholders only arise from a value read
// under Implicit VR LE (where no VR is on the wire); without resolution vrToBytes would
// emit "UN", losing the element's true type on an implicit->explicit transcode.
//
// "US or SS" signedness is defined by Pixel Representation (0028,0103): 0 is unsigned
// (US), 1 is two's-complement signed (SS) (PS3.5 §8.1.1, PS3.6 dictionary). Resolving it
// unconditionally to US corrupts the signed semantics of a value such as a Modality LUT
// Descriptor first-mapped -1000 (held as 0xFC18) or a signed Pixel Padding Value: a later
// explicit-VR read of a concrete US decodes 0xFC18 as 64536 and never sign-reinterprets
// it. The value bytes are byte-identical either way, so signed carries the dataset's
// PixelRepresentation==1 state to pick SS over US. When PixelRepresentation is absent the
// standard's default is unsigned, so signed is false and the pair resolves to US.
//
// The word/byte placeholders (US or OW, US or SS or OW, OB or OW) resolve to OW: every
// candidate carries raw 16-bit words or bytes, so the field is byte-preserving and no
// signedness applies. A concrete VR passes through unchanged.
func resolveExplicitVR(vr VR, signed bool) VR {
	switch vr {
	case VRUSorSS:
		if signed {
			return VRSS
		}
		return VRUS
	case VRUSorOW, VRUSorSSorOW:
		return VROW
	case VROBorOW:
		return VROW
	default:
		return vr
	}
}

func (vr VR) String() string {
	if int(vr) < len(vrNames) && vrNames[vr] != "" {
		return vrNames[vr]
	}
	return "??"
}

// Is32BitLength reports whether the VR uses the 4-byte explicit-VR length form.
func (vr VR) Is32BitLength() bool {
	switch vr {
	case VROB, VROW, VROD, VROF, VROL, VROV, VRSQ, VRUC, VRUR, VRUT, VRUN:
		return true
	default:
		return false
	}
}

// usesSpecificCharacterSet reports whether the VR's text is decoded through the
// dataset's (0008,0005) Specific Character Set. PS3.5 §6.1.2.3 names exactly these
// VRs as customisable; every other text VR (AE CS DA DS IS TM UI UR AS) is always the
// default repertoire (ISO 646) and ignores (0008,0005).
func (vr VR) usesSpecificCharacterSet() bool {
	switch vr {
	case VRSH, VRLO, VRST, VRLT, VRPN, VRUC, VRUT:
		return true
	default:
		return false
	}
}

// PadByte returns the value-field pad byte: NULL (0x00) for UI, SPACE (0x20) for
// other string VRs. ok is false for binary VRs (their natural length is even).
func (vr VR) PadByte() (byte, bool) {
	switch vr {
	case VRUI:
		return 0x00, true
	case VRAE, VRAS, VRCS, VRDA, VRDS, VRDT, VRIS, VRLO, VRLT,
		VRPN, VRSH, VRST, VRTM, VRUC, VRUR, VRUT:
		return 0x20, true
	default:
		return 0, false
	}
}
