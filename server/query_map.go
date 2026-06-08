package server

import (
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicomweb"
	"github.com/codeninja55/go-radx/dimse"
)

// dimseLevelFromQIDO maps a QIDO-RS search level to the shared dimse.QueryLevel the Catalogue queries
// against, so both protocol planes hit one index API at one level vocabulary. DICOMweb search begins
// at the study level (there is no patient level in QIDO-RS), so QueryStudies maps to the study level.
func dimseLevelFromQIDO(l dicomweb.QueryLevel) dimse.QueryLevel {
	switch l {
	case dicomweb.QuerySeries:
		return dimse.QueryLevelSeries
	case dicomweb.QueryInstances:
		return dimse.QueryLevelImage
	default:
		return dimse.QueryLevelStudy
	}
}

// matchKeysFromQIDO projects a parsed QIDO-RS request into the catalogue's tag->value match map. The
// parent UIDs the URL path scoped (StudyUID, SeriesUID) become match constraints alongside the
// attribute-matching keys, so the catalogue scopes a series/instance search to its parent without the
// caller re-stating it. The match values may carry patient identifiers, so a caller that logs this
// map must redact the values (PRD §9.1).
func matchKeysFromQIDO(q dicomweb.QueryRequest) map[dicom.Tag]string {
	match := make(map[dicom.Tag]string, len(q.Match)+2)
	if q.StudyUID != "" {
		match[dicom.TagStudyInstanceUID] = string(q.StudyUID)
	}
	if q.SeriesUID != "" {
		match[dicom.TagSeriesInstanceUID] = string(q.SeriesUID)
	}
	for _, mk := range q.Match {
		match[mk.Tag] = mk.Value
	}
	return match
}
