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
}

// ReadOption configures a Read/ReadFile call.
type ReadOption func(*readConfig)

// defaultMaxSequenceDepth bounds SQ nesting unless the caller overrides it. It
// guards against a maliciously deep sequence (Increment 3 enforces it).
const defaultMaxSequenceDepth = 64

// newReadConfig resolves opts over the safe defaults.
func newReadConfig(opts ...ReadOption) readConfig {
	cfg := readConfig{
		maxElementLen:    defaultMaxElementLen,
		maxSequenceDepth: defaultMaxSequenceDepth,
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

// WithMaxSequenceDepth caps SQ nesting (default 64).
func WithMaxSequenceDepth(n int) ReadOption {
	return func(c *readConfig) { c.maxSequenceDepth = n }
}

// WithStopAtPixelData defers the pixel-data element for a partial read.
func WithStopAtPixelData() ReadOption {
	return func(c *readConfig) { c.stopAtPixelData = true }
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
