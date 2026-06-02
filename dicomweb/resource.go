package dicomweb

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/codeninja55/go-radx/dicom"
)

// Level is the depth of a DICOMweb resource path: study, series, or instance. PS3.18
// search begins at the study level, so there is no patient level here; a patient query
// is a study-level QIDO-RS search filtered by PatientID.
//
// This is a local enum rather than dimse.QueryLevel because the dimse query-level type
// is not yet present in this module; the reference doc maps the two one-to-one and a
// later increment reconciles them when the dimse type lands.
type Level int

const (
	// LevelStudy addresses /studies/{StudyInstanceUID}.
	LevelStudy Level = iota
	// LevelSeries addresses /studies/{study}/series/{SeriesInstanceUID}.
	LevelSeries
	// LevelInstance addresses /studies/{study}/series/{series}/instances/{SOPInstanceUID}.
	LevelInstance
)

// String renders the level name for diagnostics.
func (l Level) String() string {
	switch l {
	case LevelStudy:
		return "study"
	case LevelSeries:
		return "series"
	case LevelInstance:
		return "instance"
	default:
		return "unknown"
	}
}

// ResourcePath identifies a DICOMweb resource through the study/series/instance URL
// hierarchy. The deepest non-empty UID sets the level. Construct it through
// NewStudy/NewSeries/NewInstance so a caller never assembles URL fragments by hand and
// the mapping to a query level is explicit.
type ResourcePath struct {
	Study    dicom.UID // StudyInstanceUID;  required for series and instance paths
	Series   dicom.UID // SeriesInstanceUID; required for instance paths
	Instance dicom.UID // SOPInstanceUID
}

// NewStudy returns a study-level path.
func NewStudy(study dicom.UID) ResourcePath {
	return ResourcePath{Study: study}
}

// NewSeries returns a series-level path.
func NewSeries(study, series dicom.UID) ResourcePath {
	return ResourcePath{Study: study, Series: series}
}

// NewInstance returns an instance-level path.
func NewInstance(study, series, instance dicom.UID) ResourcePath {
	return ResourcePath{Study: study, Series: series, Instance: instance}
}

// Level reports the depth implied by the deepest UID set on the path.
func (p ResourcePath) Level() Level {
	switch {
	case p.Instance != "":
		return LevelInstance
	case p.Series != "":
		return LevelSeries
	default:
		return LevelStudy
	}
}

// Path renders the URL path segment for the resource, e.g.
// "/studies/{study}/series/{series}". Each UID present is validated as a DICOM UID
// before it is interpolated; an invalid or out-of-order path (a series without a study,
// an instance without a series) is rejected with ErrInvalidResource. A UID is never
// escaped blindly: a value that is not a conformant UID is rejected, not URL-encoded
// into the path, so a malformed identifier can never inject path segments.
func (p ResourcePath) Path() (string, error) {
	if err := validateUID(p.Study, "StudyInstanceUID"); err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("/studies/")
	b.WriteString(string(p.Study))

	if p.Series == "" {
		// A study-level path may not carry an instance UID without its series.
		if p.Instance != "" {
			return "", fmt.Errorf("%w: instance UID present without a series UID", ErrInvalidResource)
		}
		return b.String(), nil
	}

	if err := validateUID(p.Series, "SeriesInstanceUID"); err != nil {
		return "", err
	}
	b.WriteString("/series/")
	b.WriteString(string(p.Series))

	if p.Instance == "" {
		return b.String(), nil
	}
	if err := validateUID(p.Instance, "SOPInstanceUID"); err != nil {
		return "", err
	}
	b.WriteString("/instances/")
	b.WriteString(string(p.Instance))
	return b.String(), nil
}

// Frames addresses one or more pixel frames of an instance (1-based, per PS3.18
// §10.4.3). The path must be instance-level; a frame number below 1 is rejected. The
// rendered form is "/studies/.../instances/{uid}/frames/1,4,5".
func (p ResourcePath) Frames(frames ...int) (string, error) {
	if p.Level() != LevelInstance {
		return "", fmt.Errorf("%w: frames require an instance-level path", ErrInvalidResource)
	}
	if len(frames) == 0 {
		return "", fmt.Errorf("%w: no frame numbers given", ErrInvalidResource)
	}
	base, err := p.Path()
	if err != nil {
		return "", err
	}
	nums := make([]string, 0, len(frames))
	for _, f := range frames {
		if f < 1 {
			return "", fmt.Errorf("%w: frame number %d is below 1 (frames are 1-based)", ErrInvalidResource, f)
		}
		nums = append(nums, strconv.Itoa(f))
	}
	return base + "/frames/" + strings.Join(nums, ","), nil
}

// validateUID rejects an empty or non-conformant UID with a typed ErrInvalidResource
// naming the field, never the offending value: a malformed identifier is
// attacker-controlled and could carry PHI (PRD §9.1).
func validateUID(u dicom.UID, field string) error {
	if u == "" {
		return fmt.Errorf("%w: %s is empty", ErrInvalidResource, field)
	}
	if err := u.Validate(); err != nil {
		return fmt.Errorf("%w: %s is not a conformant DICOM UID", ErrInvalidResource, field)
	}
	return nil
}
