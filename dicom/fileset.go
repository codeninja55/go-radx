package dicom

import (
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// MediaStorageDirectoryStorage is the SOP Class of the DICOMDIR file that anchors a
// File-set (PS3.4 Annex G; PS3.10 §8.6).
const MediaStorageDirectoryStorage SOPClassUID = "1.2.840.10008.1.3.10"

// FileSet is a loaded DICOM File-set: the parsed DICOMDIR plus the directory-record
// hierarchy it carries (PS3.10 §8; PS3.3 Annex F). Open one with OpenFileSet; build
// and write a new one with FileSetBuilder.
//
// FileSet is read-only after OpenFileSet returns and safe for concurrent reads; it is
// never mutated by its own methods.
type FileSet struct {
	path      string
	rootPath  string
	file      *File
	roots     []*DirectoryRecord
	instances []*FileInstance
}

// DirectoryRecord is one item of the Directory Record Sequence (0004,1220), placed in
// the Patient/Study/Series/Instance hierarchy by its offset links (PS3.3 §F.3). Type
// is the Directory Record Type (0004,1430); DataSet is the record's full dataset
// including its offset elements and record keys.
type DirectoryRecord struct {
	Type    string
	DataSet *DataSet

	parent   *DirectoryRecord
	children []*DirectoryRecord
	offset   int64
}

// Children returns the record's lower-level directory entity in chain order.
func (r *DirectoryRecord) Children() []*DirectoryRecord { return r.children }

// Parent returns the record's parent, or nil for a root-entity record.
func (r *DirectoryRecord) Parent() *DirectoryRecord { return r.parent }

// FileInstance is a directory record that references a file-set member through
// Referenced File ID (0004,1500).
type FileInstance struct {
	// Record is the referencing (leaf) directory record.
	Record *DirectoryRecord

	path string
}

// Path returns the member file's path under the file-set root. The path is resolved
// and traversal-guarded at OpenFileSet time; the file itself may or may not exist.
func (fi *FileInstance) Path() string { return fi.path }

// FileID returns the Referenced File ID (0004,1500) components.
func (fi *FileInstance) FileID() []string {
	comps, _ := fi.Record.DataSet.GetStrings(TagReferencedFileID)
	return comps
}

// Load reads the referenced member as a Part 10 file.
func (fi *FileInstance) Load(opts ...ReadOption) (*File, error) {
	return ReadFile(fi.path, opts...)
}

// OpenFileSet opens a File-set from its DICOMDIR. path names the DICOMDIR file or the
// directory that contains one. The DICOMDIR is parsed as a normal Part 10 file, its
// SOP Class is checked, and the Directory Record Sequence is resolved into a typed
// hierarchy by following the offset links (PS3.10 Table 8.1-1). An offset that does
// not land on a directory record, or a link chain that revisits a record (a cycle),
// is a *ValueError: the walk is bounded by the record count, so hostile offsets fail
// fast instead of looping (PRD §9.3).
func OpenFileSet(path string, opts ...ReadOption) (*FileSet, error) {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		path = filepath.Join(path, "DICOMDIR")
	}
	f, err := ReadFile(path, opts...)
	if err != nil {
		return nil, err
	}

	if f.Meta.MediaStorageSOPClassUID != MediaStorageDirectoryStorage {
		return nil, &ValueError{Tag: tagMediaStorageSOPClass, VR: VRUI,
			Msg: "not a DICOMDIR: Media Storage SOP Class is not Media Storage Directory Storage"}
	}
	seq, ok := f.DataSet.GetSequence(TagDirectoryRecordSequence)
	if !ok {
		return nil, &ValueError{Tag: TagDirectoryRecordSequence, VR: VRSQ,
			Msg: "DICOMDIR has no Directory Record Sequence"}
	}

	fs := &FileSet{
		path:     path,
		rootPath: filepath.Dir(path),
		file:     f,
	}
	if err := fs.resolveRecords(seq); err != nil {
		return nil, err
	}
	return fs, nil
}

// File returns the parsed DICOMDIR Part 10 file.
func (fs *FileSet) File() *File { return fs.file }

// Path returns the DICOMDIR file's path.
func (fs *FileSet) Path() string { return fs.path }

// RootPath returns the file-set root directory (the DICOMDIR's directory), the base
// every Referenced File ID resolves under.
func (fs *FileSet) RootPath() string { return fs.rootPath }

// ID returns the File-set ID (0004,1130), empty when the element is absent or empty.
func (fs *FileSet) ID() string {
	id, _ := fs.file.DataSet.GetString(TagFileSetID)
	return id
}

// UID returns the File-set's identifying UID: the DICOMDIR's Media Storage SOP
// Instance UID.
func (fs *FileSet) UID() UID {
	return UID(fs.file.Meta.MediaStorageSOPInstanceUID)
}

// Roots returns the root directory entity's records in chain order (typically the
// PATIENT records).
func (fs *FileSet) Roots() []*DirectoryRecord { return fs.roots }

// Records iterates every record in the hierarchy depth-first, parents before
// children, siblings in chain order.
func (fs *FileSet) Records() iter.Seq[*DirectoryRecord] {
	return func(yield func(*DirectoryRecord) bool) {
		// Iterative DFS: an explicit stack bounds the walk by the record count even
		// for a degenerate, maximally deep hierarchy (PRD §9.3).
		stack := make([]*DirectoryRecord, len(fs.roots))
		for i, r := range fs.roots {
			stack[len(fs.roots)-1-i] = r
		}
		for len(stack) > 0 {
			r := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if !yield(r) {
				return
			}
			for i := len(r.children) - 1; i >= 0; i-- {
				stack = append(stack, r.children[i])
			}
		}
	}
}

// Instances returns the file-set members: every record carrying a Referenced File ID,
// in hierarchy order.
func (fs *FileSet) Instances() []*FileInstance { return fs.instances }

// Find returns the instances whose record hierarchy matches every criterion. A
// criterion matches when the tag is present on the instance's record or any of its
// ancestor records with the given value (the record-level analogue of pydicom's
// FileSet.find). Values compare against the element's rendered string values.
func (fs *FileSet) Find(criteria map[Tag]string) []*FileInstance {
	var out []*FileInstance
	for _, fi := range fs.instances {
		match := true
		for t, want := range criteria {
			if !hierarchyHasValue(fi.Record, t, want) {
				match = false
				break
			}
		}
		if match {
			out = append(out, fi)
		}
	}
	return out
}

// FindValues returns the distinct values of t across every record in the hierarchy,
// in first-seen order (the record-level analogue of pydicom's FileSet.find_values).
func (fs *FileSet) FindValues(t Tag) []string {
	var out []string
	seen := make(map[string]bool)
	for r := range fs.Records() {
		vals, ok := recordValueStrings(r.DataSet, t)
		if !ok {
			continue
		}
		for _, v := range vals {
			if !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		}
	}
	return out
}

// hierarchyHasValue reports whether t carries value want on r or any ancestor of r.
func hierarchyHasValue(r *DirectoryRecord, t Tag, want string) bool {
	for ; r != nil; r = r.parent {
		vals, ok := recordValueStrings(r.DataSet, t)
		if !ok {
			continue
		}
		for _, v := range vals {
			if v == want {
				return true
			}
		}
	}
	return false
}

// recordValueStrings renders an element's values as strings for record-level
// matching: text VRs verbatim, binary integers and IS/DS through their decimal form.
func recordValueStrings(ds *DataSet, t Tag) ([]string, bool) {
	e, ok := ds.Get(t)
	if !ok {
		return nil, false
	}
	switch v := e.Value.(type) {
	case *Strings:
		return v.Strings(), true
	case *Ints:
		ns := v.Ints()
		out := make([]string, len(ns))
		for i, n := range ns {
			out[i] = strconv.FormatInt(n, 10)
		}
		return out, true
	case *Decimals:
		dv := v.Decimals()
		out := make([]string, len(dv))
		for i, d := range dv {
			out[i] = d.String()
		}
		return out, true
	default:
		return nil, false
	}
}

// resolveRecords builds the record hierarchy from the Directory Record Sequence by
// following the offset links. Each item's byte offset was captured at parse time, so
// resolution is a map lookup, never a re-scan of the file.
func (fs *FileSet) resolveRecords(seq *Sequence) error {
	byOffset := make(map[int64]*DirectoryRecord, seq.Len())
	for it := range seq.Items() {
		rec := &DirectoryRecord{DataSet: it.DataSet, offset: it.fileOffset}
		rec.Type, _ = it.DataSet.GetString(TagDirectoryRecordType)
		byOffset[it.fileOffset] = rec
	}

	visited := make(map[int64]bool, seq.Len())
	first := recordOffset(fs.file.DataSet, TagOffsetOfTheFirstDirectoryRecordOfTheRootDirectoryEntity)
	roots, err := followChain(byOffset, visited, first, nil,
		TagOffsetOfTheFirstDirectoryRecordOfTheRootDirectoryEntity)
	if err != nil {
		return err
	}
	fs.roots = roots

	// Resolve each record's lower-level entity breadth-first. The visited set spans
	// the whole walk: a record has exactly one parent and one predecessor in a
	// conformant file-set, so any second arrival is a cycle or a shared-tail
	// corruption and the walk stays bounded by the record count.
	queue := append([]*DirectoryRecord(nil), roots...)
	for len(queue) > 0 {
		rec := queue[0]
		queue = queue[1:]
		childOff := recordOffset(rec.DataSet, TagOffsetOfReferencedLowerLevelDirectoryEntity)
		children, err := followChain(byOffset, visited, childOff, rec,
			TagOffsetOfReferencedLowerLevelDirectoryEntity)
		if err != nil {
			return err
		}
		rec.children = children
		queue = append(queue, children...)
	}

	for r := range fs.Records() {
		comps, ok := r.DataSet.GetStrings(TagReferencedFileID)
		if !ok || len(comps) == 0 {
			continue
		}
		p, err := resolveFileID(fs.rootPath, comps)
		if err != nil {
			return err
		}
		fs.instances = append(fs.instances, &FileInstance{Record: r, path: p})
	}
	return nil
}

// followChain resolves the sibling chain that starts at off: each record's Offset of
// the Next Directory Record (0004,1400) names its successor; zero ends the chain
// (PS3.3 §F.2.2). linkTag names the element the entry offset came from, for
// diagnostics. An offset that is not a record start or that revisits a record is a
// *ValueError; the shared visited set bounds every chain by the total record count.
func followChain(byOffset map[int64]*DirectoryRecord, visited map[int64]bool,
	off int64, parent *DirectoryRecord, linkTag Tag) ([]*DirectoryRecord, error) {
	var out []*DirectoryRecord
	for off != 0 {
		rec, ok := byOffset[off]
		if !ok {
			return nil, &ValueError{Tag: linkTag, VR: VRUL,
				Msg: fmt.Sprintf("directory record offset %d does not match any record in the Directory Record Sequence", off)}
		}
		if visited[off] {
			return nil, &ValueError{Tag: linkTag, VR: VRUL,
				Msg: fmt.Sprintf("directory record offset chain revisits the record at offset %d (cycle)", off)}
		}
		visited[off] = true
		rec.parent = parent
		out = append(out, rec)
		linkTag = TagOffsetOfTheNextDirectoryRecord
		off = recordOffset(rec.DataSet, linkTag)
	}
	return out, nil
}

// recordOffset reads a UL offset element, treating an absent or non-integer element
// as 0 ("no record", PS3.10 Table 8.1-1).
func recordOffset(ds *DataSet, t Tag) int64 {
	n, ok := ds.GetInt(t)
	if !ok || n < 0 {
		return 0
	}
	return n
}

// resolveFileID joins the Referenced File ID components under the file-set root.
// Components are validated before joining: an empty, dot, dot-dot, or
// separator-bearing component could escape the root, so it is rejected as a
// *ValueError without echoing the value (PRD §9.1). Legacy file-sets with lowercase
// or over-long components are accepted on read; conformant IDs (PS3.10 §8.5) are
// enforced on write.
func resolveFileID(root string, comps []string) (string, error) {
	for _, c := range comps {
		if c == "" || c == "." || c == ".." ||
			strings.ContainsAny(c, `/\:`) || strings.ContainsRune(c, 0) {
			return "", &ValueError{Tag: TagReferencedFileID, VR: VRCS,
				Msg: "file ID component would escape the file-set root"}
		}
	}
	if len(comps) == 0 {
		return "", &ValueError{Tag: TagReferencedFileID, VR: VRCS, Msg: "empty file ID"}
	}
	p := filepath.Join(append([]string{root}, comps...)...)
	rel, err := filepath.Rel(root, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", &ValueError{Tag: TagReferencedFileID, VR: VRCS,
			Msg: "file ID resolves outside the file-set root"}
	}
	return p, nil
}
