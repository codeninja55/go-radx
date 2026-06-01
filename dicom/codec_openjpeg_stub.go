//go:build !(cgo && dicom_openjpeg)

package dicom

// This file is the companion to codec_openjpeg.go for every build that does NOT
// enable both cgo and the dicom_openjpeg tag: the default `go build ./...`, a
// CGO_ENABLED=0 build, and a cgo build without the tag. It registers no JPEG 2000
// codec, so a JPEG 2000 instance degrades to the typed ErrCodecUnavailable
// (PRD §7.3) rather than pulling in OpenJPEG. Building with -tags dicom_openjpeg
// against an installed libopenjp2 swaps in the real codec.
//
// There is intentionally nothing to register here; the pure-Go pixel pipeline is
// complete on its own and the JPEG 2000 syntaxes have no pure-Go decoder.
