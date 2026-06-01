package dimse

import (
	"time"

	"github.com/codeninja55/go-radx/dicom"
)

// Default timeouts mirror the pynetdicom defaults (dimse.md "Application Entity"). All are
// also overridable per association via context deadlines.
const (
	defaultACSETimeout    = 30 * time.Second // association negotiation/release
	defaultDIMSETimeout   = 30 * time.Second // awaiting a DIMSE response
	defaultNetworkTimeout = 60 * time.Second // idle association
	defaultMaxPDULength   = MaxPDULength(16382)
)

// aeConfig holds the per-AE knobs set by functional options. There is no global mutable
// state (PRD §9.4); every AE carries its own configuration.
type aeConfig struct {
	acseTimeout       time.Duration
	dimseTimeout      time.Duration
	networkTimeout    time.Duration
	connectionTimeout time.Duration // 0 = no explicit TCP-connect timeout
	maxPDULength      MaxPDULength

	implementationClassUID dicom.UID
	implementationVersion  string
}

// AEOption configures an AE at construction.
type AEOption func(*aeConfig)

// WithACSETimeout sets the association negotiation/release timeout (default 30s).
func WithACSETimeout(d time.Duration) AEOption {
	return func(c *aeConfig) { c.acseTimeout = d }
}

// WithDIMSETimeout sets the timeout awaiting a DIMSE response (default 30s).
func WithDIMSETimeout(d time.Duration) AEOption {
	return func(c *aeConfig) { c.dimseTimeout = d }
}

// WithNetworkTimeout sets the idle-association timeout (default 60s).
func WithNetworkTimeout(d time.Duration) AEOption {
	return func(c *aeConfig) { c.networkTimeout = d }
}

// WithConnectionTimeout sets the TCP-connect timeout (default none).
func WithConnectionTimeout(d time.Duration) AEOption {
	return func(c *aeConfig) { c.connectionTimeout = d }
}

// WithMaxPDULength sets the maximum PDU length to advertise (default 16382; 0 = unlimited,
// resolved at fragmentation time against the local send cap — Codex DIMSE-005).
func WithMaxPDULength(n MaxPDULength) AEOption {
	return func(c *aeConfig) { c.maxPDULength = n }
}

// WithImplementationClassUID sets the Implementation Class UID advertised in the
// A-ASSOCIATE user information (PS3.7 D.3.3.2).
func WithImplementationClassUID(uid dicom.UID) AEOption {
	return func(c *aeConfig) { c.implementationClassUID = uid }
}

// WithImplementationVersionName sets the Implementation Version Name advertised in the
// A-ASSOCIATE user information (PS3.7 D.3.3.2).
func WithImplementationVersionName(name string) AEOption {
	return func(c *aeConfig) { c.implementationVersion = name }
}

// AE is a local DICOM Application Entity: the factory for outbound associations (SCU) and
// inbound listeners (SCP). It carries no global mutable state; every knob is per-AE and set
// with functional options. Safe for concurrent use (the config is immutable after NewAE).
type AE struct {
	title AETitle
	cfg   aeConfig
}

// NewAE constructs an AE with the given local AE Title and options. Zero options yield the
// pynetdicom-default timeouts and a 16382-byte maximum PDU length. It returns a typed
// *ValidationError if the title is not a conformant AE Title.
func NewAE(title AETitle, opts ...AEOption) (*AE, error) {
	if _, err := ParseAETitle(string(title)); err != nil {
		return nil, err
	}
	cfg := aeConfig{
		acseTimeout:    defaultACSETimeout,
		dimseTimeout:   defaultDIMSETimeout,
		networkTimeout: defaultNetworkTimeout,
		maxPDULength:   defaultMaxPDULength,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &AE{title: title, cfg: cfg}, nil
}

// Title returns the local AE Title.
func (ae *AE) Title() AETitle { return ae.title }

// config returns the resolved configuration (internal; used by Associate and Server).
func (ae *AE) config() aeConfig { return ae.cfg }
