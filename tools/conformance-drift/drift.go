// Package conformancedrift checks that the go-radx conformance statements in docs/conformance
// stay honest as the code moves: countable claims in the statements must match what the code
// actually exposes, every unimplemented-surface statement must carry its "NOT YET SHIPPED"
// banner, and every public standard/server package must carry its one-line stability marker.
//
// The check runs as a test (go test ./tools/conformance-drift/...) so a drift between a
// statement and the code it describes fails the build. The pure functions here take the repo
// root as an argument so the same logic runs against the real tree and against temp fixtures
// that prove each drift class is detected.
package conformancedrift

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// notYetShippedBanner is the marker text every scaffold conformance statement must carry so a
// consumer is never misled into treating an unimplemented surface as conformance-guaranteed.
const notYetShippedBanner = "NOT YET SHIPPED"

// stabilityMarker is the one-line godoc marker every public standard/server package carries to
// declare its API-stability posture (cross-cutting.md "Governance and stability posture").
const stabilityMarker = "Stability:"

// presetTableRow matches a row of the "Presentation-context preset summary" table in
// docs/conformance/dicom.md: the preset name in the first cell (a `Name()` code span) and the
// context count in the third. A count cell of "NOT YET SHIPPED ..." marks a preset the
// statement names but does not claim as shipped, so the existence check exempts it.
var presetTableRow = regexp.MustCompile(`^\|\s*` + "`" + `([A-Za-z]+)\(\)` + "`" + `\s*\|[^|]*\|\s*([^|]+?)\s*\|`)

// leadingIntRE pulls the first integer out of a preset count cell (e.g. "36 (table above)" -> 36).
var leadingIntRE = regexp.MustCompile(`^(\d+)`)

// PresetClaim is a single countable claim from the dicom.md preset-count table: a preset
// helper and the number of presentation contexts the statement says it returns. NotYetShipped
// records a preset the statement names but does not claim as shipped; for those the code-side
// function is expected to be absent and no count is asserted.
type PresetClaim struct {
	Name          string
	Count         int
	NotYetShipped bool
}

// PresetCounter returns the number of presentation contexts a code-side preset exposes. The
// code registry maps each documented preset name to one of these; a nil entry means the preset
// is not implemented in code.
type PresetCounter func() int

// Finding is a single detected drift. The check fails when any finding is produced.
type Finding struct {
	Class   string // "preset-count", "preset-missing", "preset-unexpected", "banner", "stability"
	Subject string // the preset, doc, or package the finding concerns
	Detail  string
}

func (f Finding) String() string {
	return fmt.Sprintf("[%s] %s: %s", f.Class, f.Subject, f.Detail)
}

// scaffoldStatements are the conformance statements whose surfaces are not yet implemented;
// each must carry the NOT YET SHIPPED banner until its surface ships.
var scaffoldStatements = []string{
	"dicomweb.md",
	"dimse.md",
	"convert.md",
	"cli-server.md",
}

// stabilityPackages are the top-level public packages that must each carry a one-line
// "Stability:" godoc marker on their package comment.
var stabilityPackages = []string{
	"convert",
	"dicom",
	"dicomweb",
	"dimse",
	"fhir",
	"hl7v2",
	"server",
}

// Check runs every drift class against the repo rooted at root using codeCounts as the
// code-side source of truth for preset context counts. It returns the union of all findings;
// an empty slice means the statements and the code agree.
func Check(root string, codeCounts map[string]PresetCounter) ([]Finding, error) {
	var findings []Finding

	presetFindings, err := checkPresetCounts(root, codeCounts)
	if err != nil {
		return nil, err
	}
	findings = append(findings, presetFindings...)

	bannerFindings, err := checkBanners(root)
	if err != nil {
		return nil, err
	}
	findings = append(findings, bannerFindings...)

	stabilityFindings, err := checkStabilityMarkers(root)
	if err != nil {
		return nil, err
	}
	findings = append(findings, stabilityFindings...)

	return findings, nil
}

// checkPresetCounts compares the preset-count table in dicom.md against codeCounts: every
// shipped preset the table names must exist in code with the documented count, every preset
// marked NOT YET SHIPPED must be absent from code, and codeCounts may not name a preset the
// table omits (so a code-only preset is also surfaced).
func checkPresetCounts(root string, codeCounts map[string]PresetCounter) ([]Finding, error) {
	claims, err := ParsePresetClaims(filepath.Join(root, "docs", "conformance", "dicom.md"))
	if err != nil {
		return nil, err
	}
	if len(claims) == 0 {
		return nil, fmt.Errorf("no preset claims parsed from dicom.md: the preset-count table is missing or its format changed")
	}

	var findings []Finding
	documented := make(map[string]bool, len(claims))
	for _, claim := range claims {
		documented[claim.Name] = true
		counter, inCode := codeCounts[claim.Name]

		if claim.NotYetShipped {
			if inCode && counter != nil {
				findings = append(findings, Finding{
					Class:   "preset-unexpected",
					Subject: claim.Name,
					Detail:  "documented as NOT YET SHIPPED but is implemented in code",
				})
			}
			continue
		}

		if !inCode || counter == nil {
			findings = append(findings, Finding{
				Class:   "preset-missing",
				Subject: claim.Name,
				Detail:  fmt.Sprintf("named in the dicom.md preset table (%d contexts) but not found in code", claim.Count),
			})
			continue
		}

		if got := counter(); got != claim.Count {
			findings = append(findings, Finding{
				Class:   "preset-count",
				Subject: claim.Name,
				Detail:  fmt.Sprintf("dicom.md claims %d contexts, code returns %d", claim.Count, got),
			})
		}
	}

	for name, counter := range codeCounts {
		if counter == nil {
			continue
		}
		if !documented[name] {
			findings = append(findings, Finding{
				Class:   "preset-unexpected",
				Subject: name,
				Detail:  "implemented in code but absent from the dicom.md preset table",
			})
		}
	}

	return findings, nil
}

// ParsePresetClaims reads the preset-count table out of the conformance statement at path. It
// returns one claim per table row, skipping the header and separator rows.
func ParsePresetClaims(path string) ([]PresetClaim, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var claims []PresetClaim
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		m := presetTableRow.FindStringSubmatch(scanner.Text())
		if m == nil {
			continue
		}
		name, countCell := m[1], strings.TrimSpace(m[2])
		if strings.Contains(countCell, notYetShippedBanner) {
			claims = append(claims, PresetClaim{Name: name, NotYetShipped: true})
			continue
		}
		intMatch := leadingIntRE.FindStringSubmatch(countCell)
		if intMatch == nil {
			return nil, fmt.Errorf("preset %q in %s has an unparseable count cell %q", name, filepath.Base(path), countCell)
		}
		count, err := strconv.Atoi(intMatch[1])
		if err != nil {
			return nil, fmt.Errorf("preset %q count %q in %s: %w", name, intMatch[1], filepath.Base(path), err)
		}
		claims = append(claims, PresetClaim{Name: name, Count: count})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return claims, nil
}

// checkBanners verifies every scaffold conformance statement carries the NOT YET SHIPPED
// banner, so an unimplemented surface can never be silently presented as conformance-guaranteed.
func checkBanners(root string) ([]Finding, error) {
	var findings []Finding
	for _, name := range scaffoldStatements {
		path := filepath.Join(root, "docs", "conformance", name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if !strings.Contains(string(data), notYetShippedBanner) {
			findings = append(findings, Finding{
				Class:   "banner",
				Subject: name,
				Detail:  fmt.Sprintf("scaffold statement is missing the %q banner", notYetShippedBanner),
			})
		}
	}
	return findings, nil
}

// checkStabilityMarkers verifies every public standard/server package carries a one-line
// "Stability:" godoc marker. It scans the package's .go files (excluding tests) for a comment
// line carrying the marker immediately within the package's doc comment block.
func checkStabilityMarkers(root string) ([]Finding, error) {
	var findings []Finding
	for _, pkg := range stabilityPackages {
		ok, err := packageHasStabilityMarker(filepath.Join(root, pkg))
		if err != nil {
			return nil, err
		}
		if !ok {
			findings = append(findings, Finding{
				Class:   "stability",
				Subject: pkg,
				Detail:  fmt.Sprintf("package is missing its %q godoc marker", stabilityMarker),
			})
		}
	}
	return findings, nil
}

// packageHasStabilityMarker reports whether any non-test .go file directly in dir carries the
// stability marker in a comment line that precedes the package clause.
func packageHasStabilityMarker(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		ok, err := fileHasStabilityMarker(filepath.Join(dir, e.Name()))
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

// fileHasStabilityMarker reports whether the file at path carries a "// ... Stability:" comment
// line that appears before the package clause (i.e. within the package doc comment block).
func fileHasStabilityMarker(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "package ") {
			return false, scanner.Err()
		}
		if strings.HasPrefix(line, "//") && strings.Contains(line, stabilityMarker) {
			return true, scanner.Err()
		}
	}
	return false, scanner.Err()
}
