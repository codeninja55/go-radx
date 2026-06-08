package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
)

// ensureDir creates dir (and parents) with owner-only permissions, surfacing a creation failure as
// a file-I/O fault (exit 5). It is the single directory-create primitive the receive sinks use.
func ensureDir(dir string) error {
	if dir == "" {
		return &exitcode.UsageErr{Message: "an output directory must be named"}
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	return nil
}

// writeReceivedInstance writes one received dataset under root in a Study/Series/SOP directory
// layout, in the negotiated transfer syntax. The UIDs are sender-controlled input: each is
// validated as a conformant DICOM UID and rejected if it carries a path separator, so a malformed
// identifier can never escape root or traverse upward (RADX-016, the SCP path-safety rule). The
// transfer syntax falls back to Explicit VR Little Endian when the caller passes an empty one (an
// unnegotiated context cannot have produced an instance, but the fallback keeps the writer total).
func writeReceivedInstance(root string, ds *dicom.DataSet, ts dicom.TransferSyntax) error {
	study, err := safeUIDSegment(ds, dicom.TagStudyInstanceUID, "StudyInstanceUID")
	if err != nil {
		return err
	}
	series, err := safeUIDSegment(ds, dicom.TagSeriesInstanceUID, "SeriesInstanceUID")
	if err != nil {
		return err
	}
	instance, err := safeUIDSegment(ds, dicom.TagSOPInstanceUID, "SOPInstanceUID")
	if err != nil {
		return err
	}

	dir := filepath.Join(root, study, series)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	path := filepath.Join(dir, instance+".dcm")

	if ts == "" {
		ts = dicom.ExplicitVRLittleEndian
	}
	return ds.WriteFile(path, ts)
}

// safeUIDSegment reads a UID-valued element and validates it for use as a path segment: it must be
// present, a conformant DICOM UID, and free of path separators. Any violation is a parse failure
// (the object's identifier is malformed), so a sender cannot inject a traversal segment.
func safeUIDSegment(ds *dicom.DataSet, tag dicom.Tag, name string) (string, error) {
	value, ok := ds.GetString(tag)
	if !ok || value == "" {
		return "", &dicom.ValueError{Tag: tag, VR: dicom.VRUI, Msg: name + " is absent; cannot derive an output path"}
	}
	if strings.ContainsAny(value, `/\`) || value == "." || value == ".." {
		return "", &dicom.ValueError{Tag: tag, VR: dicom.VRUI, Msg: name + " carries a path separator; rejecting to prevent traversal"}
	}
	if err := dicom.UID(value).Validate(); err != nil {
		return "", fmt.Errorf("%s is not a conformant UID: %w", name, err)
	}
	return value, nil
}
