package dicom

// File is a parsed Part 10 object: the 128-byte preamble, the File Meta
// Information group, and the main dataset encoded in the transfer syntax named by
// (0002,0010).
type File struct {
	Preamble [128]byte
	Meta     *FileMeta
	DataSet  *DataSet
}

// FileMeta is the group-0002 File Meta Information, always encoded Explicit VR
// Little Endian. The required elements are surfaced as typed fields; Elements
// holds the full group-0002 dataset including any optional elements.
type FileMeta struct {
	MediaStorageSOPClassUID    SOPClassUID
	MediaStorageSOPInstanceUID SOPInstanceUID
	TransferSyntaxUID          TransferSyntax
	ImplementationClassUID     UID
	Elements                   *DataSet
}

// readConfig holds the resolved read options. There is no global mutable config;
// every knob is a functional option (PRD §9.4).
type readConfig struct {
	maxElementLen    uint32
	maxInflatedBytes int64
	maxSequenceDepth int
	stopAtPixelData  bool
	// defaultCharSet is the fallback Specific Character Set defined terms applied to
	// customisable text VRs when the dataset has no (0008,0005). Empty means the
	// default repertoire (ISO 646).
	defaultCharSet []string
	// activeCharset is the resolved Specific Character Set in force at the current
	// parse position. It flows through readConfig copies (a value type), so each
	// dataset and sequence-item scope can swap its own pointer without a global or a
	// shared mutation (PRD §9.4). nil means the default repertoire.
	activeCharset *SpecificCharacterSet
	// lenientDates relaxes DA parsing to accept the legacy YYYY and YYYYMM partial
	// forms. Strict YYYYMMDD is the default (Codex DCM-010).
	lenientDates bool
	// deferThreshold is the value-field size in bytes above which an element is
	// recorded as a *DeferredValue instead of materialised. Negative means deferral
	// is off (the default). deferPath is the re-openable source each deferred value
	// loads from; only ReadFile can supply it, so Read and DecodeDataSet reject the
	// option fail-closed.
	deferThreshold int64
	deferPath      string
}

// deferralEnabled reports whether this read records deferred values.
func (c *readConfig) deferralEnabled() bool { return c.deferThreshold >= 0 }

// shouldDefer reports whether h's value is recorded as deferred rather than
// materialised: deferral is on, the length is defined and over the threshold, the
// element is not a sequence, and it is not (0008,0005) — the active character set
// always materialises because it governs the decode of the text elements that
// follow it.
func (c *readConfig) shouldDefer(h elementHeader) bool {
	return c.deferralEnabled() &&
		h.vr != VRSQ &&
		h.length != undefinedLength &&
		int64(h.length) > c.deferThreshold &&
		h.tag != TagSpecificCharacterSet
}

// ReadOption configures a Read/ReadFile call.
type ReadOption func(*readConfig)

// defaultMaxSequenceDepth bounds SQ nesting unless the caller overrides it. It
// guards against a maliciously deep sequence (Increment 3 enforces it).
const defaultMaxSequenceDepth = 64

// defaultMaxInflatedBytes caps the total bytes inflated from a Deflated Explicit VR
// LE main dataset when the caller does not override it with WithMaxInflatedBytes. A
// DEFLATE stream lets a tiny input expand into a long run of small valid elements,
// so without a total bound the deflated read path can be driven to spin (a
// decompression bomb). The budget is generous: DICOM deflate ratios are modest (the
// metadata is mostly short text and small binary fields, well under an order of
// magnitude), so 4 GiB comfortably holds a legitimate deflated study's inflated main
// dataset — far larger than the 256 MiB per-element cap that bounds any single value
// — while still being finite, so a hostile stream fails fast instead of running
// without end.
const defaultMaxInflatedBytes int64 = 4 << 30 // 4 GiB

// newReadConfig resolves opts over the safe defaults.
func newReadConfig(opts ...ReadOption) readConfig {
	cfg := readConfig{
		maxElementLen:    defaultMaxElementLen,
		maxInflatedBytes: defaultMaxInflatedBytes,
		maxSequenceDepth: defaultMaxSequenceDepth,
		deferThreshold:   -1,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// WithMaxElementLen caps a single element's value field. A length over n is
// rejected before allocation (Codex DCM-004).
func WithMaxElementLen(n uint32) ReadOption {
	return func(c *readConfig) { c.maxElementLen = n }
}

// WithMaxInflatedBytes caps the total bytes inflated from a Deflated Explicit VR LE
// main dataset (default 4 GiB). A stream that inflates past n is rejected with a
// *LimitExceededError rather than allowed to spin the element loop, which is how a
// tiny crafted DEFLATE stream would otherwise mount a decompression-bomb denial of
// service. A non-positive n disables the bound. It has no effect on the four
// uncompressed transfer syntaxes, which carry no DEFLATE stream.
func WithMaxInflatedBytes(n int64) ReadOption {
	return func(c *readConfig) { c.maxInflatedBytes = n }
}

// WithMaxSequenceDepth caps SQ nesting (default 64).
func WithMaxSequenceDepth(n int) ReadOption {
	return func(c *readConfig) { c.maxSequenceDepth = n }
}

// WithStopAtPixelData defers the pixel-data element for a partial read.
func WithStopAtPixelData() ReadOption {
	return func(c *readConfig) { c.stopAtPixelData = true }
}

// WithDeferredValues defers element values larger than threshold bytes (pydicom's
// defer_size analogue, PRD §6.2): instead of materialising the value, the reader
// records its byte window as a *DeferredValue and skips it, so memory stays bounded
// while reading large objects. The value is loaded from the source file on first
// access — transparently through the dataset accessors and the write path, or
// explicitly through (*DeferredValue).Load when the load error matters. A threshold
// of 0 defers every non-empty value; a negative threshold disables deferral.
//
// Deferral requires a re-openable source, so it works only with ReadFile: Read and
// DecodeDataSet reject the option fail-closed because a generic io.Reader cannot be
// re-read on demand. Deflated Explicit VR LE is also rejected fail-closed: element
// offsets there address the inflated stream, which is not seekable in the source
// file. Under an encapsulated transfer syntax the (7FE0,0010) fragment stream is
// always deferred (its delimited length is unknown until it has been scanned); the
// scan still validates the item structure byte for byte. (0008,0005) Specific
// Character Set is never deferred — it governs the decode of the elements that
// follow it.
//
// The source file must still be present and unmodified when a deferred value loads;
// the recorded window is re-validated against the file on every load and a source
// that shrank or stopped parsing is a typed *DeferredLoadError, never a panic.
func WithDeferredValues(threshold int64) ReadOption {
	return func(c *readConfig) { c.deferThreshold = threshold }
}

// WithLenientDates accepts the legacy partial DA forms (YYYY and YYYYMM) in
// addition to the strict YYYYMMDD. Strict parsing is the default: the prototype
// accepted partial dates unconditionally, silently treating them as valid clinical
// metadata (Codex DCM-010).
func WithLenientDates() ReadOption {
	return func(c *readConfig) { c.lenientDates = true }
}

// WithDefaultCharacterSet sets the Specific Character Set defined terms applied to
// customisable text VRs when the dataset itself carries no (0008,0005). A dataset's
// own (0008,0005) always takes precedence (PS3.5 §6.1.2.3). With no terms the default
// repertoire (ISO 646) is used.
func WithDefaultCharacterSet(cs ...string) ReadOption {
	return func(c *readConfig) { c.defaultCharSet = append([]string(nil), cs...) }
}

// writeConfig holds the resolved write options.
type writeConfig struct{}

// WriteOption configures a Write/WriteFile call.
type WriteOption func(*writeConfig)

// newWriteConfig resolves opts over the defaults.
func newWriteConfig(opts ...WriteOption) writeConfig {
	var cfg writeConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}
