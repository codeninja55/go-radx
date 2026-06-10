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
	path, err := receivedInstancePath(root, ds)
	if err != nil {
		return err
	}
	if ts == "" {
		ts = dicom.ExplicitVRLittleEndian
	}
	return ds.WriteFile(path, ts)
}

// writeReceivedRawInstance writes the byte-exact Part 10 representation of a received instance under
// root in the Study/Series/SOP layout, without re-encoding. A WADO-RS retrieval uses this to
// preserve the transfer syntax the origin returned: re-encoding from the decoded dataset would
// silently transcode the object, so the captured bytes are written through unchanged. The path is
// derived from the dataset's UIDs and validated with the same path-safety guarantee as
// writeReceivedInstance, so a malformed sender UID can never escape root.
func writeReceivedRawInstance(root string, ds *dicom.DataSet, encoded []byte) error {
	path, err := receivedInstancePath(root, ds)
	if err != nil {
		return err
	}
	// 0o600: a received instance may carry PHI, so it is readable by the owner only.
	return os.WriteFile(path, encoded, 0o600)
}

// receivedInstancePath validates the dataset's Study/Series/SOP UIDs as path-safe segments and
// returns the per-instance output path under root, creating the Study/Series directory. Each UID is
// rejected if absent, non-conformant, or carrying a path separator, so a sender-controlled
// identifier can never escape root or traverse upward (RADX-016, the SCP path-safety rule).
func receivedInstancePath(root string, ds *dicom.DataSet) (string, error) {
	study, err := safeUIDSegment(ds, dicom.TagStudyInstanceUID, "StudyInstanceUID")
	if err != nil {
		return "", err
	}
	series, err := safeUIDSegment(ds, dicom.TagSeriesInstanceUID, "SeriesInstanceUID")
	if err != nil {
		return "", err
	}
	instance, err := safeUIDSegment(ds, dicom.TagSOPInstanceUID, "SOPInstanceUID")
	if err != nil {
		return "", err
	}

	dir := filepath.Join(root, study, series)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}
	return filepath.Join(dir, instance+".dcm"), nil
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
