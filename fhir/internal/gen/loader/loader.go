// Package loader is the bottom stage of the FHIR generator pipeline. It reads the
// vendored, checksum-pinned HL7 FHIR definition bundle from a directory, verifies
// every file against the committed SHA256SUMS manifest before parsing, decodes the
// StructureDefinition / ValueSet / CodeSystem entries into raw records, and indexes
// them by canonical URL and by name for the model layer to resolve.
//
// The loader fails closed: a checksum mismatch, a missing required file, a
// malformed bundle, or an undecodable entry returns a typed *LoadError that names
// the offending file and refuses to proceed. It reads only from the local
// filesystem and never reaches the network — generation is reproducible from the
// pinned input alone.
package loader

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// sumsFile is the name of the checksum manifest committed beside the bundle
// files. It is in the shasum -a 256 format: "<hex>  <filename>" per line.
const sumsFile = "SHA256SUMS"

// verifyChecksums reads the SHA256SUMS manifest in dir and confirms every listed
// file's on-disk bytes hash to the recorded value. It returns a *LoadError on the
// first mismatch or missing file, and on a missing or malformed manifest. The
// error names the file but never embeds its bytes.
func verifyChecksums(dir string) error {
	sums, err := readSums(dir)
	if err != nil {
		return err
	}

	// Verify in a deterministic order so the first reported failure is stable.
	names := make([]string, 0, len(sums))
	for name := range sums {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		want := sums[name]
		got, err := sha256HexFile(filepath.Join(dir, name))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return &LoadError{File: name, Detail: "listed in " + sumsFile + " but missing on disk"}
			}
			return &LoadError{File: name, Detail: "read for checksum", Err: err}
		}
		if got != want {
			return &LoadError{File: name, Detail: "checksum mismatch (bundle drifted from pin)"}
		}
	}
	return nil
}

// readSums parses the SHA256SUMS manifest into a filename→hex map. The manifest
// itself is required; its absence is a fail-closed error.
func readSums(dir string) (map[string]string, error) {
	path := filepath.Join(dir, sumsFile)
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, &LoadError{File: sumsFile, Detail: "checksum manifest missing"}
		}
		return nil, &LoadError{File: sumsFile, Detail: "open checksum manifest", Err: err}
	}
	defer func() { _ = f.Close() }()

	sums := make(map[string]string)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, &LoadError{File: sumsFile, Detail: "malformed manifest line"}
		}
		sums[fields[1]] = strings.ToLower(fields[0])
	}
	if err := sc.Err(); err != nil {
		return nil, &LoadError{File: sumsFile, Detail: "scan checksum manifest", Err: err}
	}
	if len(sums) == 0 {
		return nil, &LoadError{File: sumsFile, Detail: "checksum manifest is empty"}
	}
	return sums, nil
}

// sha256HexFile streams a file through SHA-256 and returns the lowercase hex
// digest. Streaming keeps memory bounded for the large bundle files (the resource
// bundle is tens of megabytes), since the hash never needs the whole file in
// memory at once.
func sha256HexFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
