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
	"encoding/json"
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

// requiredFiles are the three bundle files Load reads. Complex and primitive
// datatypes, resources, and value sets live in separate bundles, so all three are
// required; a missing one is a fail-closed error.
var requiredFiles = []string{
	"profiles-types.json",
	"profiles-resources.json",
	"valuesets.json",
}

// Bundle is the loaded and indexed definition set. It holds the raw decoded
// StructureDefinition / ValueSet / CodeSystem records indexed by canonical URL and
// by name, for the model layer to resolve type references and value-set bindings.
// It is read-only after Load returns and carries no mutable shared state.
type Bundle struct {
	sdByName  map[string]*StructureDefinition
	sdByURL   map[string]*StructureDefinition
	vsByURL   map[string]*ValueSet
	vsByName  map[string]*ValueSet
	csByURL   map[string]*CodeSystem
	csByName  map[string]*CodeSystem
	resources int // count of kind=="resource" StructureDefinitions
	datatypes int // count of kind=="complex-type"/"primitive-type" StructureDefinitions
}

// StructureDefinition returns the StructureDefinition indexed by its FHIR name
// (for example "Patient" or "Period").
func (b *Bundle) StructureDefinition(name string) (*StructureDefinition, bool) {
	sd, ok := b.sdByName[name]
	return sd, ok
}

// StructureDefinitionByURL returns the StructureDefinition indexed by its
// canonical URL (for example "http://hl7.org/fhir/StructureDefinition/Patient").
func (b *Bundle) StructureDefinitionByURL(url string) (*StructureDefinition, bool) {
	sd, ok := b.sdByURL[url]
	return sd, ok
}

// ValueSet returns the ValueSet indexed by its canonical URL.
func (b *Bundle) ValueSet(url string) (*ValueSet, bool) {
	vs, ok := b.vsByURL[url]
	return vs, ok
}

// ValueSetByName returns the ValueSet indexed by its FHIR name.
func (b *Bundle) ValueSetByName(name string) (*ValueSet, bool) {
	vs, ok := b.vsByName[name]
	return vs, ok
}

// CodeSystem returns the CodeSystem indexed by its canonical URL.
func (b *Bundle) CodeSystem(url string) (*CodeSystem, bool) {
	cs, ok := b.csByURL[url]
	return cs, ok
}

// CodeSystemByName returns the CodeSystem indexed by its FHIR name.
func (b *Bundle) CodeSystemByName(name string) (*CodeSystem, bool) {
	cs, ok := b.csByName[name]
	return cs, ok
}

// ResourceCount returns the number of resource-kind StructureDefinitions loaded.
func (b *Bundle) ResourceCount() int { return b.resources }

// DatatypeCount returns the number of datatype (complex-type or primitive-type)
// StructureDefinitions loaded.
func (b *Bundle) DatatypeCount() int { return b.datatypes }

// ValueSetCount returns the number of ValueSets loaded.
func (b *Bundle) ValueSetCount() int { return len(b.vsByURL) }

// CodeSystemCount returns the number of CodeSystems loaded.
func (b *Bundle) CodeSystemCount() int { return len(b.csByURL) }

// Load reads the vendored definition bundle in dir, verifies every file against
// the committed SHA256SUMS, decodes the StructureDefinition / ValueSet /
// CodeSystem entries, and indexes them by canonical URL and by name. It fails
// closed: a checksum mismatch, a missing required file, a malformed bundle, or an
// undecodable entry returns a typed *LoadError. Load never reaches the network.
func Load(dir string) (*Bundle, error) {
	if err := verifyChecksums(dir); err != nil {
		return nil, err
	}

	b := &Bundle{
		sdByName: make(map[string]*StructureDefinition),
		sdByURL:  make(map[string]*StructureDefinition),
		vsByURL:  make(map[string]*ValueSet),
		vsByName: make(map[string]*ValueSet),
		csByURL:  make(map[string]*CodeSystem),
		csByName: make(map[string]*CodeSystem),
	}

	for _, name := range requiredFiles {
		if err := b.loadFile(filepath.Join(dir, name), name); err != nil {
			return nil, err
		}
	}
	return b, nil
}

// loadFile streams one bundle file's entries into the index. It decodes the outer
// Bundle object and iterates entry[].resource one element at a time so the large
// resource bundle (tens of megabytes) is never fully materialised as a single
// in-memory structure; each entry resource is decoded, indexed, and released.
func (b *Bundle) loadFile(path, name string) error {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &LoadError{File: name, Detail: "required bundle file missing"}
		}
		return &LoadError{File: name, Detail: "open bundle file", Err: err}
	}
	defer func() { _ = f.Close() }()

	dec := json.NewDecoder(bufio.NewReaderSize(f, 1<<20))

	// Walk to the "entry" array, decoding the outer Bundle object's keys as tokens
	// and only diving into entry; everything else (resourceType, type, meta, ...)
	// is skipped without allocating.
	if err := expectDelim(dec, '{', name); err != nil {
		return err
	}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return &LoadError{File: name, Detail: "decode bundle object", Err: err}
		}
		key, ok := keyTok.(string)
		if !ok {
			return &LoadError{File: name, Detail: "malformed bundle object key"}
		}
		if key != "entry" {
			if err := skipValue(dec, name); err != nil {
				return err
			}
			continue
		}
		if err := b.loadEntries(dec, name); err != nil {
			return err
		}
	}
	return nil
}

// loadEntries decodes the entry array, indexing each entry's resource by its
// resourceType. A malformed entry is a hard error, not a silent skip.
func (b *Bundle) loadEntries(dec *json.Decoder, name string) error {
	if err := expectDelim(dec, '[', name); err != nil {
		return err
	}
	for dec.More() {
		var entry bundleEntry
		if err := dec.Decode(&entry); err != nil {
			return &LoadError{File: name, Detail: "decode bundle entry", Err: err}
		}
		if len(entry.Resource) == 0 {
			continue
		}
		if err := b.indexResource(entry.Resource, name); err != nil {
			return err
		}
	}
	// Consume the closing ']' of the entry array.
	if _, err := dec.Token(); err != nil {
		return &LoadError{File: name, Detail: "close entry array", Err: err}
	}
	return nil
}

// indexResource decodes one raw entry resource into the record matching its
// resourceType and adds it to the indexes. Resource types the generator does not
// consume are ignored; an undecodable known type is a hard error.
func (b *Bundle) indexResource(raw json.RawMessage, name string) error {
	var hdr resourceHeader
	if err := json.Unmarshal(raw, &hdr); err != nil {
		return &LoadError{File: name, Detail: "peek entry resourceType", Err: err}
	}

	switch hdr.ResourceType {
	case "StructureDefinition":
		var sd StructureDefinition
		if err := json.Unmarshal(raw, &sd); err != nil {
			return &LoadError{File: name, Detail: "decode StructureDefinition", Err: err}
		}
		if sd.Name != "" {
			b.sdByName[sd.Name] = &sd
		}
		if sd.URL != "" {
			b.sdByURL[sd.URL] = &sd
		}
		switch sd.Kind {
		case "resource":
			b.resources++
		case "complex-type", "primitive-type":
			b.datatypes++
		}
	case "ValueSet":
		var vs ValueSet
		if err := json.Unmarshal(raw, &vs); err != nil {
			return &LoadError{File: name, Detail: "decode ValueSet", Err: err}
		}
		if vs.URL != "" {
			b.vsByURL[vs.URL] = &vs
		}
		if vs.Name != "" {
			b.vsByName[vs.Name] = &vs
		}
	case "CodeSystem":
		var cs CodeSystem
		if err := json.Unmarshal(raw, &cs); err != nil {
			return &LoadError{File: name, Detail: "decode CodeSystem", Err: err}
		}
		if cs.URL != "" {
			b.csByURL[cs.URL] = &cs
		}
		if cs.Name != "" {
			b.csByName[cs.Name] = &cs
		}
	default:
		// A resource type the generator does not consume (for example a
		// SearchParameter or a CompartmentDefinition); ignore it.
	}
	return nil
}

// expectDelim reads the next token and asserts it is the given delimiter, so a
// truncated or non-object/array bundle fails closed with a named error.
func expectDelim(dec *json.Decoder, want json.Delim, name string) error {
	tok, err := dec.Token()
	if err != nil {
		return &LoadError{File: name, Detail: "decode bundle structure", Err: err}
	}
	if d, ok := tok.(json.Delim); !ok || d != want {
		return &LoadError{File: name, Detail: "unexpected bundle structure"}
	}
	return nil
}

// skipValue consumes the next JSON value (scalar, object, or array) from the
// decoder without retaining it, so non-entry keys of the outer Bundle object are
// passed over without allocation.
func skipValue(dec *json.Decoder, name string) error {
	tok, err := dec.Token()
	if err != nil {
		return &LoadError{File: name, Detail: "skip bundle value", Err: err}
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil // scalar
	}
	if delim != '{' && delim != '[' {
		return nil
	}
	depth := 1
	for depth > 0 {
		t, err := dec.Token()
		if err != nil {
			return &LoadError{File: name, Detail: "skip bundle value", Err: err}
		}
		switch d, isDelim := t.(json.Delim); {
		case isDelim && (d == '{' || d == '['):
			depth++
		case isDelim && (d == '}' || d == ']'):
			depth--
		}
	}
	return nil
}

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
