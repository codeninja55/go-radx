//go:build !(cgo && dicom_libjpeg)

package dicom

// This file is the companion to codec_libjpeg.go for every build that does NOT
// enable both cgo and the dicom_libjpeg tag: the default `go build ./...`, a
// CGO_ENABLED=0 build, and a cgo build without the tag. It registers no JPEG
// (baseline/extended) codec, so a JPEG Baseline or Extended instance degrades to the
// typed ErrCodecUnavailable (PRD §7.3) rather than pulling in libjpeg-turbo.
// Building with -tags dicom_libjpeg against an installed libjpeg-turbo swaps in the
// real codec.
//
// There is intentionally nothing to register here; the pure-Go pixel pipeline is
// complete on its own and the lossy JPEG syntaxes have no pure-Go decoder.
