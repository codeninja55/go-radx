//go:build !(cgo && dicom_charls)

package dicom

// This file is the companion to codec_charls.go for every build that does NOT enable
// both cgo and the dicom_charls tag: the default `go build ./...`, a CGO_ENABLED=0
// build, and a cgo build without the tag. It registers no JPEG-LS codec, so a
// JPEG-LS instance degrades to the typed ErrCodecUnavailable (PRD §7.3) rather than
// pulling in CharLS. Building with -tags dicom_charls against an installed CharLS
// swaps in the real codec.
//
// There is intentionally nothing to register here; the pure-Go pixel pipeline is
// complete on its own and the JPEG-LS syntaxes have no pure-Go decoder.
