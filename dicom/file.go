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
