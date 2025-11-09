package scp

import (
	"fmt"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicom/tag"
)

// Common DICOM tags used by SCP services
var (
	TagSOPClassUID    = tag.New(0x0008, 0x0016)
	TagSOPInstanceUID = tag.New(0x0008, 0x0018)
)

// getStringFromDataSet extracts a string value from a DICOM dataset
func getStringFromDataSet(ds *dicom.DataSet, t tag.Tag) (string, error) {
	elem, err := ds.Get(t)
	if err != nil {
		return "", fmt.Errorf("get tag %s: %w", t, err)
	}

	value := elem.Value()
	if value == nil {
		return "", fmt.Errorf("tag %s has nil value", t)
	}

	return value.String(), nil
}
