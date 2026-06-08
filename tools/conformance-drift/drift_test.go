package conformancedrift

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/dimse"
)

// codePresetCounts is the code-side source of truth for presentation-context preset counts: it
// binds each documented preset name to the live count its dimse helper returns. A preset the
// dicom.md table marks NOT YET SHIPPED has no entry here, so the existence check confirms it is
// genuinely absent from the public API.
var codePresetCounts = map[string]PresetCounter{
	"VerificationContexts":             func() int { return len(dimse.VerificationContexts()) },
	"StorageContexts":                  func() int { return len(dimse.StorageContexts()) },
	"QueryRetrieveContexts":            func() int { return len(dimse.QueryRetrieveContexts()) },
	"QueryRetrieveWithStorageContexts": func() int { return len(dimse.QueryRetrieveWithStorageContexts()) },
	"BasicWorklistContexts":            func() int { return len(dimse.BasicWorklistContexts()) },
	"ModalityPerformedContexts":        func() int { return len(dimse.ModalityPerformedContexts()) },
	"StorageCommitmentContexts":        func() int { return len(dimse.StorageCommitmentContexts()) },
}

// TestNoConformanceDrift is the merge-blocking gate: it runs every drift class against the real
// repository tree and fails on any divergence between the conformance statements and the code.
func TestNoConformanceDrift(t *testing.T) {
	root := repoRoot(t)
	findings, err := Check(root, codePresetCounts)
	if err != nil {
		t.Fatalf("conformance-drift check errored: %v", err)
	}
	for _, f := range findings {
		t.Errorf("conformance drift: %s", f)
	}
}

// repoRoot walks up from the test working directory until it finds the go.mod whose module path
// is the go-radx root module, so the check resolves docs/ and the package sources regardless of
// where the test binary runs.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil && strings.Contains(string(data), "module github.com/codeninja55/go-radx\n") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate go-radx module root above %s", dir)
		}
		dir = parent
	}
}

// TestCheckDetectsDrift proves the check bites: each case mutates a temp copy of the real tree
// to introduce exactly one drift, and asserts the matching finding class is reported. The real
// docs are never mutated.
func TestCheckDetectsDrift(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(t *testing.T, root string) // introduces one drift into the temp tree
		counts    map[string]PresetCounter        // code-side counts to check against
		wantClass string
		wantSubj  string
	}{
		{
			name:      "wrong preset count",
			mutate:    func(t *testing.T, root string) {},
			counts:    withCount(codePresetCounts, "StorageContexts", func() int { return 99 }),
			wantClass: "preset-count",
			wantSubj:  "StorageContexts",
		},
		{
			name: "documented preset not defined in code",
			mutate: func(t *testing.T, root string) {
				removeCodePreset(t, root, "StorageContexts")
			},
			counts:    codePresetCounts,
			wantClass: "preset-missing",
			wantSubj:  "StorageContexts",
		},
		{
			name: "code preset absent from the table is surfaced",
			mutate: func(t *testing.T, root string) {
				addCodePreset(t, root, "ColorPaletteContexts")
			},
			counts:    codePresetCounts,
			wantClass: "preset-unexpected",
			wantSubj:  "ColorPaletteContexts",
		},
		{
			name: "shipped preset claimed as not-yet-shipped surfaces an unexpected preset",
			mutate: func(t *testing.T, root string) {
				markPresetNotYetShipped(t, root, "StorageContexts")
			},
			counts:    codePresetCounts,
			wantClass: "preset-unexpected",
			wantSubj:  "StorageContexts",
		},
		{
			name:      "documented preset has no live count wired in",
			mutate:    func(t *testing.T, root string) {},
			counts:    withoutPreset(codePresetCounts, "StorageContexts"),
			wantClass: "preset-count",
			wantSubj:  "StorageContexts",
		},
		{
			// Reproduces the audit's dicom.md over-claim: a negotiation row marked "Yes" names an option
			// function the dimse package does not export. The check must bite.
			name: "negotiation feature claimed as supported but the option function is absent",
			mutate: func(t *testing.T, root string) {
				addOverClaimNegotiationRow(t, root, "Fictional negotiation", "WithNonexistentNegotiation")
			},
			counts:    codePresetCounts,
			wantClass: "negotiation-missing",
			wantSubj:  "Fictional negotiation",
		},
		{
			name: "missing not-yet-shipped banner",
			mutate: func(t *testing.T, root string) {
				stripBanner(t, root, "dimse.md")
			},
			counts:    codePresetCounts,
			wantClass: "banner",
			wantSubj:  "dimse.md",
		},
		{
			// cross-cutting.md mentions NOT YET SHIPPED in its methodology prose, so a whole-file
			// phrase search would miss the removed header banner; the header-specific check catches it.
			name: "missing header banner despite the phrase appearing elsewhere",
			mutate: func(t *testing.T, root string) {
				stripBanner(t, root, "cross-cutting.md")
			},
			counts:    codePresetCounts,
			wantClass: "banner",
			wantSubj:  "cross-cutting.md",
		},
		{
			name: "missing stability marker",
			mutate: func(t *testing.T, root string) {
				stripStabilityMarker(t, root, "dimse")
			},
			counts:    codePresetCounts,
			wantClass: "stability",
			wantSubj:  "dimse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := copyTreeForCheck(t)
			tt.mutate(t, root)
			findings, err := Check(root, tt.counts)
			if err != nil {
				t.Fatalf("Check errored: %v", err)
			}
			if !hasFinding(findings, tt.wantClass, tt.wantSubj) {
				t.Fatalf("expected a %q finding for %q, got %v", tt.wantClass, tt.wantSubj, findings)
			}
		})
	}
}

// TestCheckCleanOnFixture proves the check passes on an unmutated copy of the real tree, so the
// detection cases above are isolating a single introduced drift rather than a pre-existing one.
func TestCheckCleanOnFixture(t *testing.T) {
	root := copyTreeForCheck(t)
	findings, err := Check(root, codePresetCounts)
	if err != nil {
		t.Fatalf("Check errored: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings on a clean fixture, got %v", findings)
	}
}

// copyTreeForCheck builds a temp tree carrying just the inputs Check reads — the conformance
// statements, the dimse preset source the existence check parses, and one stability-marker
// source file per package — so a self-test can mutate it without touching the real repository.
func copyTreeForCheck(t *testing.T) string {
	t.Helper()
	src := repoRoot(t)
	dst := t.TempDir()

	confDir := filepath.Join(dst, "docs", "conformance")
	mkdirAll(t, confDir)
	confFiles := append([]string{"dicom.md"}, scaffoldStatements...)
	for _, name := range confFiles {
		copyFile(t, filepath.Join(src, "docs", "conformance", name), filepath.Join(confDir, name))
	}

	for _, pkg := range stabilityPackages {
		markerFile := stabilityMarkerFile(t, filepath.Join(src, pkg))
		dstPkg := filepath.Join(dst, pkg)
		mkdirAll(t, dstPkg)
		copyFile(t, markerFile, filepath.Join(dstPkg, filepath.Base(markerFile)))
	}

	// DiscoverCodePresets and DiscoverDimseFuncs parse the dimse package source, so the fixture
	// carries the real presets file (preset existence) and the option-function sources the
	// negotiation table names (negotiation existence) alongside the dimse stability marker above:
	// negotiation.go declares WithAsyncOps/WithUserIdentity/WithExtendedNegotiation/
	// WithCommonExtendedNegotiation and server.go declares WithAuthenticator.
	for _, name := range []string{"presets.go", "ae.go", "association.go", "negotiation.go", "server.go"} {
		copyFile(t, filepath.Join(src, "dimse", name), filepath.Join(dst, "dimse", name))
	}
	return dst
}

// removeCodePreset deletes the exported func <preset>() declaration from the fixture's dimse
// presets source, simulating a preset documented in the table but removed from the code.
func removeCodePreset(t *testing.T, root, preset string) {
	t.Helper()
	path := filepath.Join(root, "dimse", "presets.go")
	data := readFile(t, path)
	marker := "func " + preset + "()"
	start := strings.Index(data, marker)
	if start < 0 {
		t.Fatalf("preset function %q not found in fixture presets.go", preset)
	}
	end := strings.Index(data[start:], "\n}\n")
	if end < 0 {
		t.Fatalf("could not find end of %q in fixture presets.go", preset)
	}
	writeFile(t, path, data[:start]+data[start+end+len("\n}\n"):])
}

// addCodePreset appends an exported, table-absent preset function to the fixture's dimse presets
// source, simulating a preset that exists in code but is missing from the dicom.md table.
func addCodePreset(t *testing.T, root, preset string) {
	t.Helper()
	path := filepath.Join(root, "dimse", "presets.go")
	data := readFile(t, path)
	stub := "\n\nfunc " + preset + "() []PresentationContext {\n\treturn nil\n}\n"
	writeFile(t, path, data+stub)
}

// stabilityMarkerFile returns the .go file in dir that carries the package stability marker, so
// the fixture copies a real marker-bearing file rather than guessing the filename.
func stabilityMarkerFile(t *testing.T, dir string) string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir %s: %v", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		ok, err := fileHasStabilityMarker(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("scan %s: %v", e.Name(), err)
		}
		if ok {
			return filepath.Join(dir, e.Name())
		}
	}
	t.Fatalf("no stability-marker source file found in %s", dir)
	return ""
}

func markPresetNotYetShipped(t *testing.T, root, preset string) {
	t.Helper()
	path := filepath.Join(root, "docs", "conformance", "dicom.md")
	data := readFile(t, path)
	span := "`" + preset + "()`"
	lines := strings.Split(data, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "|") && strings.Contains(line, span) {
			cells := strings.Split(line, "|")
			cells[len(cells)-2] = " " + notYetShippedBanner + " (would be 99) "
			lines[i] = strings.Join(cells, "|")
			writeFile(t, path, strings.Join(lines, "\n"))
			return
		}
	}
	t.Fatalf("preset row for %q not found in dicom.md", preset)
}

// addOverClaimNegotiationRow inserts a new negotiation-table row marked "Yes" that names an option
// function the dimse package does not export, reproducing an over-claim: a feature marked supported
// whose `With...(` knob does not exist. The row is inserted inside the "## Association negotiation"
// section (before the next "###" sub-heading) so the section-scoped parser picks it up. Now that
// every real negotiation feature ships, the over-claim is injected with a fictional function rather
// than by flipping a real row.
func addOverClaimNegotiationRow(t *testing.T, root, feature, fn string) {
	t.Helper()
	path := filepath.Join(root, "docs", "conformance", "dicom.md")
	lines := strings.Split(readFile(t, path), "\n")
	inSection := false
	row := "| " + feature + " | Yes | `" + fn + "(...)` |"
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## Association negotiation") {
			inSection = true
			continue
		}
		if inSection && strings.HasPrefix(trimmed, "###") {
			// Insert the over-claim row just before the next sub-heading, still inside the section.
			lines = append(lines[:i], append([]string{row, ""}, lines[i:]...)...)
			writeFile(t, path, strings.Join(lines, "\n"))
			return
		}
	}
	t.Fatalf("association-negotiation section not found in dicom.md")
}

// stripBanner removes only the leading blockquote scaffold banner line, leaving any other
// occurrence of the phrase in place, so the test proves the banner check matches the header
// banner specifically rather than the bare phrase anywhere in the file.
func stripBanner(t *testing.T, root, doc string) {
	t.Helper()
	path := filepath.Join(root, "docs", "conformance", doc)
	lines := strings.Split(readFile(t, path), "\n")
	for i, line := range lines {
		if scaffoldBannerRE.MatchString(line) {
			lines[i] = "> This statement is authored and conformance-backed."
			writeFile(t, path, strings.Join(lines, "\n"))
			return
		}
	}
	t.Fatalf("no scaffold banner line found in %s", doc)
}

func stripStabilityMarker(t *testing.T, root, pkg string) {
	t.Helper()
	dir := filepath.Join(root, pkg)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir %s: %v", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		ok, err := fileHasStabilityMarker(path)
		if err != nil {
			t.Fatalf("scan %s: %v", path, err)
		}
		if ok {
			data := readFile(t, path)
			writeFile(t, path, strings.ReplaceAll(data, stabilityMarker, "API posture:"))
			return
		}
	}
	t.Fatalf("no stability marker to strip in %s", pkg)
}

func withCount(base map[string]PresetCounter, name string, counter PresetCounter) map[string]PresetCounter {
	out := make(map[string]PresetCounter, len(base))
	for k, v := range base {
		out[k] = v
	}
	out[name] = counter
	return out
}

func withoutPreset(base map[string]PresetCounter, name string) map[string]PresetCounter {
	out := make(map[string]PresetCounter, len(base))
	for k, v := range base {
		if k == name {
			continue
		}
		out[k] = v
	}
	return out
}

func hasFinding(findings []Finding, class, subject string) bool {
	for _, f := range findings {
		if f.Class == class && f.Subject == subject {
			return true
		}
	}
	return false
}

func mkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
