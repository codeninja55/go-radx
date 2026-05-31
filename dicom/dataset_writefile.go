package dicom

import "fmt"

// WriteFile wraps ds in a minimal Part 10 File and writes it to path in ts. The
// File Meta MediaStorageSOPClassUID/MediaStorageSOPInstanceUID are derived from the
// dataset's SOP Class UID (0008,0016) and SOP Instance UID (0008,0018). It returns
// a typed error if either is absent or ts is unsupported.
func (ds *DataSet) WriteFile(path string, ts TransferSyntax, opts ...WriteOption) error {
	sopClass, ok := ds.GetString(TagSOPClassUID)
	if !ok || sopClass == "" {
		return &ValueError{Tag: TagSOPClassUID, VR: VRUI, Msg: "dataset has no SOP Class UID to derive file meta from"}
	}
	sopInstance, ok := ds.GetString(TagSOPInstanceUID)
	if !ok || sopInstance == "" {
		return &ValueError{Tag: TagSOPInstanceUID, VR: VRUI, Msg: "dataset has no SOP Instance UID to derive file meta from"}
	}
	if err := checkWritableTransferSyntax(ts); err != nil {
		return err
	}

	f := &File{
		Meta: &FileMeta{
			MediaStorageSOPClassUID:    SOPClassUID(sopClass),
			MediaStorageSOPInstanceUID: SOPInstanceUID(sopInstance),
			TransferSyntaxUID:          ts,
		},
		DataSet: ds,
	}
	if err := WriteFile(path, f, opts...); err != nil {
		return fmt.Errorf("dicom: WriteFile: %w", err)
	}
	return nil
}
