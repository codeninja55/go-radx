package dicom

import (
	"bytes"
	"fmt"
	"io"
)

// tagFileMetaInformationVersion is (0002,0001), a Type 1 OB element whose standard
// value is the two-byte 0x0001.
var tagFileMetaInformationVersion = NewTag(0x0002, 0x0001)

// fileMetaInformationVersion is the standard (0002,0001) value: a two-byte big-end
// 0x0001 carried as OB (PS3.10 §7.1).
var fileMetaInformationVersion = []byte{0x00, 0x01}

// implementationClassUIDDefault identifies go-radx as the implementation when the
// caller supplies no (0002,0012). It is a 2.25.-rooted UUID-derived UID, which
// needs no organisational registration (PS3.5 §9.2).
const implementationClassUIDDefault UID = "2.25.0"

// writeFileMeta serialises the preamble, DICM magic, and the File Meta Information
// group (always Explicit VR LE) with an auto-recomputed (0002,0000) Group Length
// written first (Codex DCM-001). Any (0002,0000) in meta.Elements is ignored and
// recomputed, never trusted.
func writeFileMeta(w io.Writer, preamble [128]byte, meta *FileMeta) error {
	group, err := buildFileMetaGroup(meta)
	if err != nil {
		return err
	}

	// Serialise the group-0002 elements to count their bytes.
	var groupBuf bytes.Buffer
	if err := writeDataSet(&groupBuf, group, fileMetaTransferSyntax); err != nil {
		return err
	}

	// Prepend the recomputed group-length element.
	gl := NewDataSet()
	gl.Set(Element{Tag: tagFileMetaGroupLength, VR: VRUL, Value: NewInts(VRUL, int64(groupBuf.Len()))})
	var glBuf bytes.Buffer
	if err := writeDataSet(&glBuf, gl, fileMetaTransferSyntax); err != nil {
		return err
	}

	if _, err := w.Write(preamble[:]); err != nil {
		return err
	}
	if _, err := io.WriteString(w, dicmMagic); err != nil {
		return err
	}
	if _, err := w.Write(glBuf.Bytes()); err != nil {
		return err
	}
	_, err = w.Write(groupBuf.Bytes())
	return err
}

// buildFileMetaGroup assembles the group-0002 dataset (without the group-length
// element) from the typed required fields, overlaying any optional elements the
// caller supplied in meta.Elements. The typed fields take precedence so the file
// meta is internally consistent. Required UIDs are validated through the single
// ParseUID path (Codex DCM-009).
func buildFileMetaGroup(meta *FileMeta) (*DataSet, error) {
	group := NewDataSet()

	// Carry over optional/extra group-0002 elements, excluding the recomputed
	// group length and the fields the typed accessors own.
	if meta.Elements != nil {
		for e := range meta.Elements.All() {
			if e.Tag.Group() != 0x0002 {
				continue
			}
			switch e.Tag {
			case tagFileMetaGroupLength, tagMediaStorageSOPClass, tagMediaStorageSOPInst,
				tagTransferSyntax, tagImplementationClassID:
				continue
			default:
				group.Set(e)
			}
		}
	}

	// (0002,0001) File Meta Information Version (Type 1): supply the standard value
	// if the caller did not.
	if _, ok := group.Get(tagFileMetaInformationVersion); !ok {
		group.Set(Element{Tag: tagFileMetaInformationVersion, VR: VROB, Value: NewBytes(VROB, fileMetaInformationVersion)})
	}

	if err := setMetaUID(group, tagMediaStorageSOPClass, UID(meta.MediaStorageSOPClassUID), true); err != nil {
		return nil, err
	}
	if err := setMetaUID(group, tagMediaStorageSOPInst, UID(meta.MediaStorageSOPInstanceUID), true); err != nil {
		return nil, err
	}
	if err := setMetaUID(group, tagTransferSyntax, UID(meta.TransferSyntaxUID), false); err != nil {
		return nil, err
	}

	implClass := meta.ImplementationClassUID
	if implClass == "" {
		implClass = implementationClassUIDDefault
	}
	if err := setMetaUID(group, tagImplementationClassID, implClass, false); err != nil {
		return nil, err
	}

	if meta.TransferSyntaxUID == "" {
		return nil, fmt.Errorf("dicom: file meta is missing %s Transfer Syntax UID", tagTransferSyntax)
	}
	return group, nil
}

// setMetaUID validates u through ParseUID (the single validation path, Codex
// DCM-009) and sets it as a UI element. When optional is true an empty value is
// skipped rather than rejected.
func setMetaUID(group *DataSet, tag Tag, u UID, optional bool) error {
	if u == "" {
		if optional {
			return nil
		}
		return &ValueError{Tag: tag, VR: VRUI, Msg: "required file-meta UID is empty"}
	}
	if _, err := ParseUID(string(u)); err != nil {
		return &ValueError{Tag: tag, VR: VRUI, Msg: fmt.Sprintf("invalid file-meta UID: %v", err)}
	}
	group.Set(Element{Tag: tag, VR: VRUI, Value: NewStrings(VRUI, string(u))})
	return nil
}
