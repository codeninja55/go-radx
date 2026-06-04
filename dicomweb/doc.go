// Package dicomweb implements the RESTful DICOM services defined in DICOM PS3.18:
// WADO-RS (web access to DICOM objects), STOW-RS (store over the web), and QIDO-RS
// (query based on ID for DICOM objects). It ships both a client for talking to a
// remote DICOMweb origin server (Orthanc, dcm4chee-arc, a cloud archive) and an
// embeddable server that exposes the same services over pluggable storage and
// query backends. It is the HTTP-based counterpart to the DIMSE services in the
// dimse package and shares the DICOM data model: a retrieved or stored object is a
// *dicom.DataSet, identifiers are dicom.UID, and encodings are
// dicom.TransferSyntax, with no parallel object model.
//
// See docs/reference/dicomweb.md for the public API.
//
// Stability: experimental. Pre-1.0; the API may change between v0.x releases.
package dicomweb
