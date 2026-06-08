package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/codeninja55/go-radx/dicom"
)

// fileStore is the default ObjectStore: it persists each object as a Part 10 file under a configured
// root, in a study/series/instance directory layout (mirroring the pynetdicom qrscp reference app's
// shape, which the interop tests exercise). It is the runnable development default, not an archive:
// production storage, retention, and encryption at rest remain the consumer's responsibility
// (PRD §9.1, §9.5).
type fileStore struct {
	root string
}

// FileStore is the default ObjectStore: it persists each object as a Part 10 file under root, in a
// study/series/instance directory layout. It creates the tree lazily on the first Put. The root must
// be a usable directory path; a path that cannot be created is a typed error, never a silent fallback.
func FileStore(root string) (ObjectStore, error) {
	if root == "" {
		return nil, errors.New("server: file store root must not be empty")
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, fmt.Errorf("server: create file store root: %w", err)
	}
	return &fileStore{root: filepath.Clean(root)}, nil
}

// Put writes ds as a Part 10 file under root/{study}/{series}/{instance}.dcm. The UIDs are treated as
// untrusted input: each path component is validated as a conformant DICOM UID before it is used, so a
// component carrying path separators or "../" can never escape the root (PRD §9.1 input validation).
// The write is all-or-nothing per object: it writes to a temp file and renames into place, so a
// crash mid-write never leaves a partial object readable as complete (PRD §9.2 truncation-is-failure).
func (s *fileStore) Put(_ context.Context, ds *dicom.DataSet) error {
	study, series, instance, err := identity(ds)
	if err != nil {
		return err
	}
	dir, file, err := s.pathFor(study, series, instance)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("server: create object directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".radx-*.tmp")
	if err != nil {
		return fmt.Errorf("server: create temp object: %w", err)
	}
	tmpName := tmp.Name()
	_ = tmp.Close()
	// Clean up the temp file on any failure after this point so a failed Put leaves no debris.
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpName)
		}
	}()

	f := &dicom.File{
		Meta: &dicom.FileMeta{
			MediaStorageSOPClassUID:    sopClassOf(ds),
			MediaStorageSOPInstanceUID: dicom.SOPInstanceUID(instance),
			TransferSyntaxUID:          dicom.ExplicitVRLittleEndian,
		},
		DataSet: ds,
	}
	if err := dicom.WriteFile(tmpName, f); err != nil {
		return fmt.Errorf("server: write object: %w", err)
	}
	if err := os.Rename(tmpName, file); err != nil {
		return fmt.Errorf("server: commit object: %w", err)
	}
	committed = true
	return nil
}

// Get reads the stored object by SOP Instance UID. It returns ErrNotFound when the file is absent, so
// a caller distinguishes a genuine miss from a read fault. The UID is validated before it is used in
// a path, so a traversal-style UID never reads outside the root.
func (s *fileStore) Get(_ context.Context, instance dicom.SOPInstanceUID) (*dicom.DataSet, error) {
	file, err := s.findInstance(string(instance))
	if err != nil {
		return nil, err
	}
	f, err := dicom.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("server: read object: %w", err)
	}
	return f.DataSet, nil
}

// Exists reports presence without materialising the object.
func (s *fileStore) Exists(_ context.Context, instance dicom.SOPInstanceUID) (bool, error) {
	_, err := s.findInstance(string(instance))
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Delete removes one stored object, returning ErrNotFound when it is absent.
func (s *fileStore) Delete(_ context.Context, instance dicom.SOPInstanceUID) error {
	file, err := s.findInstance(string(instance))
	if err != nil {
		return err
	}
	if err := os.Remove(file); err != nil {
		return fmt.Errorf("server: delete object: %w", err)
	}
	return nil
}

// pathFor resolves the directory and file path for an object, validating every UID as a conformant
// DICOM UID first so an untrusted component cannot inject a path separator or "../". It then asserts
// the resolved path stays within the root as belt-and-braces against any future component source.
func (s *fileStore) pathFor(study, series, instance string) (dir, file string, err error) {
	for _, uid := range []string{study, series, instance} {
		if verr := dicom.UID(uid).Validate(); verr != nil {
			return "", "", fmt.Errorf("server: rejecting unsafe UID path component: %w", verr)
		}
	}
	dir = filepath.Join(s.root, study, series)
	file = filepath.Join(dir, instance+".dcm")
	if !s.withinRoot(file) {
		return "", "", fmt.Errorf("server: resolved object path escapes the store root")
	}
	return dir, file, nil
}

// findInstance locates the stored file for a SOP Instance UID by validating it and walking the
// study/series tree for the matching instance file. It returns ErrNotFound when no such file exists.
// Validating the UID first means a traversal-style identifier is rejected before any filesystem walk.
func (s *fileStore) findInstance(instance string) (string, error) {
	if err := dicom.UID(instance).Validate(); err != nil {
		return "", fmt.Errorf("server: rejecting unsafe instance UID: %w", err)
	}
	target := instance + ".dcm"
	var found string
	walkErr := filepath.WalkDir(s.root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == target {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if walkErr != nil {
		return "", fmt.Errorf("server: locate object: %w", walkErr)
	}
	if found == "" {
		return "", fmt.Errorf("%w: instance not stored", ErrNotFound)
	}
	return found, nil
}

// withinRoot reports whether p resolves to a location inside the store root, the defence-in-depth
// check that a validated UID could never break but a future component source might.
func (s *fileStore) withinRoot(p string) bool {
	rel, err := filepath.Rel(s.root, filepath.Clean(p))
	if err != nil {
		return false
	}
	return rel != ".." && !hasDotDotPrefix(rel)
}

// hasDotDotPrefix reports whether rel begins with a parent-directory traversal segment.
func hasDotDotPrefix(rel string) bool {
	return len(rel) >= 3 && rel[0] == '.' && rel[1] == '.' && (rel[2] == filepath.Separator)
}

// identity extracts the study, series, and instance UIDs an object is keyed by. A missing UID is a
// rejection, never a default path, so an object with no instance identity is never stored under a
// placeholder key it cannot be retrieved by (PRD §9.2).
func identity(ds *dicom.DataSet) (study, series, instance string, err error) {
	study, ok := ds.GetString(dicom.TagStudyInstanceUID)
	if !ok || study == "" {
		return "", "", "", errors.New("server: object has no StudyInstanceUID")
	}
	series, ok = ds.GetString(dicom.TagSeriesInstanceUID)
	if !ok || series == "" {
		return "", "", "", errors.New("server: object has no SeriesInstanceUID")
	}
	instance, ok = ds.GetString(dicom.TagSOPInstanceUID)
	if !ok || instance == "" {
		return "", "", "", errors.New("server: object has no SOPInstanceUID")
	}
	return study, series, instance, nil
}

// sopClassOf returns the object's SOP Class UID for the file meta, or the empty UID when absent.
func sopClassOf(ds *dicom.DataSet) dicom.SOPClassUID {
	v, _ := ds.GetString(dicom.TagSOPClassUID)
	return dicom.SOPClassUID(v)
}
