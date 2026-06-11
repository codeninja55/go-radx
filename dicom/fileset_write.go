package dicom

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

// FileSetBuilder assembles a new DICOM File-set from Part 10 files or in-memory
// File values, then writes the DICOMDIR and the member files in the file-set layout
// (PS3.10 §8). Instances are grouped into the Patient/Study/Series/Instance record
// hierarchy by PatientID, StudyInstanceUID, and SeriesInstanceUID; member files are
// laid out under conformant generated File IDs (PS3.10 §8.5: components of 1-8
// characters from 0-9, A-Z, and underscore, at most 8 components).
//
// FileSetBuilder is NOT safe for concurrent use; the same single-threaded ownership
// rule as DataSet applies.
type FileSetBuilder struct {
	id     string
	staged []*stagedRecords
	seen   map[string]bool
}

// stagedRecords is one added instance with its pre-built directory records. The
// hierarchy records are built (and validated) at Add time so a missing required key
// fails fast; Write groups them by their key elements.
type stagedRecords struct {
	srcPath string // copy this Part 10 file verbatim; empty for an in-memory add
	file    *File  // encode this File; nil for an on-disk add

	patient, study, series, leaf *DataSet
}

// NewFileSetBuilder returns an empty builder.
func NewFileSetBuilder() *FileSetBuilder {
	return &FileSetBuilder{seen: make(map[string]bool)}
}

// SetID sets the File-set ID (0004,1130), validated at Write: at most 16 characters
// from 0-9, A-Z, and underscore (the PS3.10 §8.5 repertoire, which excludes SPACE).
// Empty (the default) writes a zero-length Type 2 element.
func (b *FileSetBuilder) SetID(id string) { b.id = id }

// Add stages the Part 10 file at path as a file-set member. The file is parsed to
// build and validate its directory records (a missing required record key is a
// *ValueError now, not at Write); at Write time the original file bytes are copied
// into the file-set verbatim, so the member is unchanged byte for byte.
func (b *FileSetBuilder) Add(path string, opts ...ReadOption) error {
	// The directory records need only the metadata that precedes the pixel data.
	f, err := ReadFile(path, append(opts, WithStopAtPixelData())...)
	if err != nil {
		return err
	}
	return b.stage(f, path)
}

// AddFile stages an in-memory Part 10 File as a file-set member. At Write time the
// member is encoded with Write, so f.Meta.TransferSyntaxUID must be writable.
func (b *FileSetBuilder) AddFile(f *File) error {
	if f == nil || f.Meta == nil || f.DataSet == nil {
		return fmt.Errorf("dicom: AddFile requires a File with Meta and DataSet")
	}
	return b.stage(f, "")
}

// stage validates f and builds its directory records.
func (b *FileSetBuilder) stage(f *File, srcPath string) error {
	if f.Meta == nil || f.Meta.TransferSyntaxUID == "" {
		return fmt.Errorf("dicom: file-set member has no transfer syntax in its file meta")
	}
	sopInst, err := requireRecordKey(f.DataSet, TagSOPInstanceUID)
	if err != nil {
		return err
	}
	if b.seen[sopInst] {
		return &ValueError{Tag: TagSOPInstanceUID, VR: VRUI,
			Msg: "duplicate SOP Instance UID in the file-set"}
	}

	s := &stagedRecords{srcPath: srcPath}
	if srcPath == "" {
		s.file = f
	}
	if s.patient, err = buildPatientRecord(f.DataSet); err != nil {
		return err
	}
	if s.study, err = buildStudyRecord(f.DataSet); err != nil {
		return err
	}
	if s.series, err = buildSeriesRecord(f.DataSet); err != nil {
		return err
	}
	if s.leaf, err = buildLeafRecord(f); err != nil {
		return err
	}
	b.seen[sopInst] = true
	b.staged = append(b.staged, s)
	return nil
}

// fileSetIDRE is the conformant File-set ID repertoire: at most 16 characters from
// 0-9, A-Z, and underscore (PS3.10 §8.4 with the §8.5 character set, no SPACE).
var fileSetIDRE = regexp.MustCompile(`^[0-9A-Z_]{0,16}$`)

// Write creates the file-set under root: the member files in their generated File ID
// layout and the DICOMDIR with offset-linked directory records (PS3.10 Table 8.1-1).
// It returns the written file-set re-opened through OpenFileSet, so every offset link
// in the DICOMDIR has been resolved before Write returns.
func (b *FileSetBuilder) Write(root string) (*FileSet, error) {
	if !fileSetIDRE.MatchString(b.id) {
		return nil, &ValueError{Tag: TagFileSetID, VR: VRCS,
			Msg: "File-set ID must be at most 16 characters from 0-9, A-Z, and underscore"}
	}

	patients := groupRecords(b.staged)
	flat, err := assignFileIDs(patients)
	if err != nil {
		return nil, err
	}

	meta, main, err := b.assembleDICOMDIR(patients, flat)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, err
	}
	for _, n := range flat {
		if n.inst == nil {
			continue
		}
		if err := writeMember(root, n); err != nil {
			return nil, err
		}
	}
	dicomdirPath := filepath.Join(root, "DICOMDIR")
	if err := WriteFile(dicomdirPath, &File{Meta: meta, DataSet: main}); err != nil {
		return nil, err
	}
	return OpenFileSet(dicomdirPath)
}

// recordNode is one directory record being assembled, with its position in the
// hierarchy and (for a leaf) the staged instance it references.
type recordNode struct {
	ds       *DataSet
	children []*recordNode
	inst     *stagedRecords
	fileID   []string // leaf only: the generated File ID components
	offset   int64
}

// groupRecords builds the Patient -> Study -> Series -> Instance tree, grouping by
// PatientID, StudyInstanceUID, and SeriesInstanceUID in first-seen order. The first
// instance of a group supplies the group's record dataset.
func groupRecords(staged []*stagedRecords) []*recordNode {
	var patients []*recordNode
	byPatient := make(map[string]*recordNode)
	byStudy := make(map[string]*recordNode)
	bySeries := make(map[string]*recordNode)

	for _, s := range staged {
		pid, _ := s.patient.GetString(TagPatientID)
		patient, ok := byPatient[pid]
		if !ok {
			patient = &recordNode{ds: s.patient}
			byPatient[pid] = patient
			patients = append(patients, patient)
		}

		suid, _ := s.study.GetString(TagStudyInstanceUID)
		study, ok := byStudy[pid+"\x00"+suid]
		if !ok {
			study = &recordNode{ds: s.study}
			byStudy[pid+"\x00"+suid] = study
			patient.children = append(patient.children, study)
		}

		seuid, _ := s.series.GetString(TagSeriesInstanceUID)
		series, ok := bySeries[pid+"\x00"+suid+"\x00"+seuid]
		if !ok {
			series = &recordNode{ds: s.series}
			bySeries[pid+"\x00"+suid+"\x00"+seuid] = series
			study.children = append(study.children, series)
		}

		series.children = append(series.children, &recordNode{ds: s.leaf, inst: s})
	}
	return patients
}

// assignFileIDs walks the tree depth-first, generating each leaf's File ID from its
// position (PT000000/ST000000/SE000000/IM000000), setting Referenced File ID
// (0004,1500) on the leaf record, and returning the records in file order. Each
// component is 8 characters from 0-9 and A-Z, so the IDs are conformant (PS3.10
// §8.5) up to 10^6 entries per level.
func assignFileIDs(patients []*recordNode) ([]*recordNode, error) {
	const perLevelCap = 1_000_000
	var flat []*recordNode
	if len(patients) > perLevelCap {
		return nil, fmt.Errorf("dicom: file-set exceeds %d patients (File ID space)", perLevelCap)
	}
	for pi, patient := range patients {
		flat = append(flat, patient)
		for si, study := range patient.children {
			flat = append(flat, study)
			for ei, series := range study.children {
				flat = append(flat, series)
				if len(patient.children) > perLevelCap || len(study.children) > perLevelCap ||
					len(series.children) > perLevelCap {
					return nil, fmt.Errorf("dicom: file-set exceeds %d entries at one hierarchy level (File ID space)", perLevelCap)
				}
				for ii, leaf := range series.children {
					leaf.fileID = []string{
						fmt.Sprintf("PT%06d", pi),
						fmt.Sprintf("ST%06d", si),
						fmt.Sprintf("SE%06d", ei),
						fmt.Sprintf("IM%06d", ii),
					}
					leaf.ds.Set(Element{Tag: TagReferencedFileID, VR: VRCS,
						Value: NewStrings(VRCS, leaf.fileID...)})
					flat = append(flat, leaf)
				}
			}
		}
	}
	return flat, nil
}

// assembleDICOMDIR computes every record's byte offset and returns the DICOMDIR's
// file meta and main dataset. Offsets are byte positions of each record's item tag
// from the start of the file (PS3.10 Table 8.1-1), computed analytically: the offset
// elements are fixed-width UL, so a record's encoded length never changes when its
// placeholder offsets are replaced with the final values.
func (b *FileSetBuilder) assembleDICOMDIR(patients, flat []*recordNode) (*FileMeta, *DataSet, error) {
	meta := &FileMeta{
		MediaStorageSOPClassUID:    MediaStorageDirectoryStorage,
		MediaStorageSOPInstanceUID: SOPInstanceUID(NewRandomUIDGenerator().Generate()),
		TransferSyntaxUID:          ExplicitVRLittleEndian,
	}
	var metaBuf bytes.Buffer
	if err := writeFileMeta(&metaBuf, [128]byte{}, meta); err != nil {
		return nil, nil, err
	}

	main := NewDataSet()
	main.Set(Element{Tag: TagFileSetID, VR: VRCS, Value: NewStrings(VRCS, fileSetIDValues(b.id)...)})
	main.Set(ulElement(TagOffsetOfTheFirstDirectoryRecordOfTheRootDirectoryEntity, 0))
	main.Set(ulElement(TagOffsetOfTheLastDirectoryRecordOfTheRootDirectoryEntity, 0))
	main.Set(Element{Tag: TagFileSetConsistencyFlag, VR: VRUS, Value: NewInts(VRUS, 0)})
	var prefixBuf bytes.Buffer
	if err := writeDataSet(&prefixBuf, main, ExplicitVRLittleEndian); err != nil {
		return nil, nil, err
	}

	// The Directory Record Sequence element header under Explicit VR LE: 4-byte tag,
	// 2-byte VR, 2-byte reserved, 4-byte defined length (PS3.5 §7.1.2).
	const sqHeaderLen = 12
	const itemHeaderLen = 8
	cursor := int64(metaBuf.Len()) + int64(prefixBuf.Len()) + sqHeaderLen
	for _, n := range flat {
		n.offset = cursor
		var cw lengthCounter
		if err := writeDataSet(&cw, n.ds, ExplicitVRLittleEndian); err != nil {
			return nil, nil, err
		}
		cursor += itemHeaderLen + cw.n
	}
	if cursor > maxValueFieldLen {
		return nil, nil, &LimitExceededError{Tag: TagDirectoryRecordSequence,
			Limit: uint64(maxValueFieldLen), Actual: uint64(cursor), Kind: "directory-record-offset"}
	}

	// Link the records: each record's next-sibling and first-child offsets, and the
	// root entity's first and last record offsets (zero means no record).
	linkSiblings(patients)
	if len(patients) > 0 {
		main.Set(ulElement(TagOffsetOfTheFirstDirectoryRecordOfTheRootDirectoryEntity, patients[0].offset))
		main.Set(ulElement(TagOffsetOfTheLastDirectoryRecordOfTheRootDirectoryEntity, patients[len(patients)-1].offset))
	}

	seq := &Sequence{}
	for _, n := range flat {
		seq.items = append(seq.items, Item{DataSet: n.ds})
	}
	main.Set(Element{Tag: TagDirectoryRecordSequence, VR: VRSQ, Value: NewSequenceValue(seq)})
	return meta, main, nil
}

// linkSiblings sets (0004,1400) and (0004,1420) through the tree from the computed
// offsets: sibling chains within each entity, and each record's lower-level entity.
func linkSiblings(siblings []*recordNode) {
	for i, n := range siblings {
		var next int64
		if i+1 < len(siblings) {
			next = siblings[i+1].offset
		}
		n.ds.Set(ulElement(TagOffsetOfTheNextDirectoryRecord, next))
		var lower int64
		if len(n.children) > 0 {
			lower = n.children[0].offset
		}
		n.ds.Set(ulElement(TagOffsetOfReferencedLowerLevelDirectoryEntity, lower))
		linkSiblings(n.children)
	}
}

// writeMember writes one staged instance to its File ID path under root: an on-disk
// add is copied byte for byte, an in-memory add is encoded with Write.
func writeMember(root string, n *recordNode) error {
	dst := filepath.Join(append([]string{root}, n.fileID...)...)
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return err
	}
	if n.inst.srcPath != "" {
		return copyFile(n.inst.srcPath, dst)
	}
	return WriteFile(dst, n.inst.file)
}

// copyFile copies src to dst byte for byte.
func copyFile(src, dst string) error {
	in, err := os.Open(src) // #nosec G304 -- copying a caller-supplied member path is this API's contract
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(dst) // #nosec G304 -- dst is root joined with generated PT/ST/SE/IM components
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// lengthCounter counts bytes without retaining them, for analytic length
// computation.
type lengthCounter struct{ n int64 }

func (w *lengthCounter) Write(p []byte) (int, error) {
	w.n += int64(len(p))
	return len(p), nil
}

// ulElement builds a UL element carrying v.
func ulElement(t Tag, v int64) Element {
	return Element{Tag: t, VR: VRUL, Value: NewInts(VRUL, v)}
}

// fileSetIDValues renders the File-set ID as element values: a zero-length Type 2
// element when unset.
func fileSetIDValues(id string) []string {
	if id == "" {
		return nil
	}
	return []string{id}
}

// srDocumentSOPClasses are the storage classes recorded as SR DOCUMENT directory
// records (PS3.3 §F.5.21); every other leaf is recorded as IMAGE. pydicom's full
// per-class record-type table is not implemented.
var srDocumentSOPClasses = map[string]bool{
	"1.2.840.10008.5.1.4.1.1.88.11": true, // Basic Text SR
	"1.2.840.10008.5.1.4.1.1.88.22": true, // Enhanced SR
	"1.2.840.10008.5.1.4.1.1.88.33": true, // Comprehensive SR
}

// newRecord starts a directory record of the given type with its required linkage
// elements as placeholders (PS3.3 §F.3.2.2). The Specific Character Set is carried
// over when present so copied text keys keep their repertoire.
func newRecord(recordType string, src *DataSet) *DataSet {
	rec := NewDataSet()
	rec.Set(ulElement(TagOffsetOfTheNextDirectoryRecord, 0))
	rec.Set(Element{Tag: TagRecordInUseFlag, VR: VRUS, Value: NewInts(VRUS, 0xFFFF)})
	rec.Set(ulElement(TagOffsetOfReferencedLowerLevelDirectoryEntity, 0))
	rec.Set(Element{Tag: TagDirectoryRecordType, VR: VRCS, Value: NewStrings(VRCS, recordType)})
	copyOptionalRecordElement(rec, src, TagSpecificCharacterSet)
	return rec
}

// buildPatientRecord builds a PATIENT record (PS3.3 §F.5.1): PatientID required,
// PatientName Type 2.
func buildPatientRecord(src *DataSet) (*DataSet, error) {
	rec := newRecord("PATIENT", src)
	copyRecordElement(rec, src, TagPatientName)
	if err := copyRequiredRecordElement(rec, src, TagPatientID); err != nil {
		return nil, err
	}
	return rec, nil
}

// buildStudyRecord builds a STUDY record (PS3.3 §F.5.2): StudyDate, StudyTime,
// StudyID, and StudyInstanceUID required; StudyDescription and AccessionNumber
// Type 2.
func buildStudyRecord(src *DataSet) (*DataSet, error) {
	rec := newRecord("STUDY", src)
	for _, t := range []Tag{TagStudyDate, TagStudyTime, TagStudyID, TagStudyInstanceUID} {
		if err := copyRequiredRecordElement(rec, src, t); err != nil {
			return nil, err
		}
	}
	copyRecordElement(rec, src, TagStudyDescription)
	copyRecordElement(rec, src, TagAccessionNumber)
	return rec, nil
}

// buildSeriesRecord builds a SERIES record (PS3.3 §F.5.3): Modality,
// SeriesInstanceUID, and SeriesNumber required.
func buildSeriesRecord(src *DataSet) (*DataSet, error) {
	rec := newRecord("SERIES", src)
	for _, t := range []Tag{TagModality, TagSeriesInstanceUID, TagSeriesNumber} {
		if err := copyRequiredRecordElement(rec, src, t); err != nil {
			return nil, err
		}
	}
	return rec, nil
}

// buildLeafRecord builds the instance-level record: IMAGE (PS3.3 §F.5.4) or
// SR DOCUMENT (PS3.3 §F.5.21) by SOP Class, with the referenced-SOP elements every
// leaf carries (PS3.3 Table F.3-3). Referenced File ID is set at Write when the
// File ID is assigned.
func buildLeafRecord(f *File) (*DataSet, error) {
	src := f.DataSet
	sopClass, err := requireRecordKey(src, TagSOPClassUID)
	if err != nil {
		return nil, err
	}
	recordType := "IMAGE"
	if srDocumentSOPClasses[sopClass] {
		recordType = "SR DOCUMENT"
	}
	rec := newRecord(recordType, src)

	if err := copyRequiredRecordElement(rec, src, TagSOPClassUID); err != nil {
		return nil, err
	}
	rec.Set(Element{Tag: TagReferencedSOPClassUIDInFile, VR: VRUI,
		Value: cloneValue(mustGet(src, TagSOPClassUID).Value)})
	rec.Delete(TagSOPClassUID)
	if err := copyRequiredRecordElement(rec, src, TagSOPInstanceUID); err != nil {
		return nil, err
	}
	rec.Set(Element{Tag: TagReferencedSOPInstanceUIDInFile, VR: VRUI,
		Value: cloneValue(mustGet(src, TagSOPInstanceUID).Value)})
	rec.Delete(TagSOPInstanceUID)
	rec.Set(Element{Tag: TagReferencedTransferSyntaxUIDInFile, VR: VRUI,
		Value: NewStrings(VRUI, string(f.Meta.TransferSyntaxUID))})

	if err := copyRequiredRecordElement(rec, src, TagInstanceNumber); err != nil {
		return nil, err
	}
	if recordType == "SR DOCUMENT" {
		for _, t := range []Tag{TagContentDate, TagContentTime,
			TagCompletionFlag, TagVerificationFlag, TagConceptNameCodeSequence} {
			if err := copyRequiredRecordElement(rec, src, t); err != nil {
				return nil, err
			}
		}
		copyOptionalRecordElement(rec, src, TagVerificationDateTime)
	}
	return rec, nil
}

// mustGet returns the element at t; callers have already validated presence.
func mustGet(ds *DataSet, t Tag) Element {
	e, _ := ds.Get(t)
	return e
}

// requireRecordKey returns t's first rendered value, or a *ValueError naming the
// missing or empty required key (the message never carries the value, PRD §9.1).
// A sequence-valued key (e.g. Concept Name Code Sequence) needs a non-empty
// sequence, not a string value.
func requireRecordKey(ds *DataSet, t Tag) (string, error) {
	if seq, ok := ds.GetSequence(t); ok && seq.Len() > 0 {
		return "", nil
	}
	vals, ok := recordValueStrings(ds, t)
	if ok && len(vals) > 0 && vals[0] != "" {
		return vals[0], nil
	}
	return "", &ValueError{Tag: t, VR: dictVR(t),
		Msg: "required directory-record key is missing or empty"}
}

// copyRequiredRecordElement copies a required (Type 1) key from src into rec.
func copyRequiredRecordElement(rec, src *DataSet, t Tag) error {
	if _, err := requireRecordKey(src, t); err != nil {
		return err
	}
	e, _ := src.Get(t)
	rec.Set(Element{Tag: t, VR: e.VR, Value: cloneValue(e.Value)})
	return nil
}

// copyRecordElement copies a Type 2 key from src into rec, writing a zero-length
// element when src lacks it.
func copyRecordElement(rec, src *DataSet, t Tag) {
	e, ok := src.Get(t)
	if !ok {
		rec.SetEmpty(t)
		return
	}
	rec.Set(Element{Tag: t, VR: e.VR, Value: cloneValue(e.Value)})
}

// copyOptionalRecordElement copies a conditional (Type 1C/3) key from src into rec,
// omitting it when absent.
func copyOptionalRecordElement(rec, src *DataSet, t Tag) {
	e, ok := src.Get(t)
	if !ok {
		return
	}
	rec.Set(Element{Tag: t, VR: e.VR, Value: cloneValue(e.Value)})
}
