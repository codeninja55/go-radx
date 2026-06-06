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
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// notYetShippedBanner is the marker text a scaffold conformance statement carries so a consumer
// is never misled into treating an unimplemented surface as conformance-guaranteed. It is also
// the token that flags a NOT YET SHIPPED preset in the dicom.md count table.
const notYetShippedBanner = "NOT YET SHIPPED"

// scaffoldBannerRE matches the leading blockquote scaffold banner a scaffold statement opens
// with — `> **Implementation status: NOT YET SHIPPED.**`. The banner check matches this specific
// header line rather than the bare phrase, so removing the top banner is detected even when the
// phrase appears elsewhere in the document (for example in prose about the drift methodology).
var scaffoldBannerRE = regexp.MustCompile(`(?m)^>\s+\*\*Implementation status:\s+` + notYetShippedBanner + `\.`)

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

// scaffoldStatements are the conformance statements whose surfaces are not yet implemented or
// not yet authored; each must carry the NOT YET SHIPPED banner until its surface ships. A
// statement leaves this list when its surface is published (the banner is removed and the
// scope contract is filled), as dicomweb.md did once WADO/QIDO/STOW and client auth shipped.
var scaffoldStatements = []string{
	"dimse.md",
	"convert.md",
	"cli-server.md",
	"cross-cutting.md",
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

// checkPresetCounts reconciles the preset-count table in dicom.md with the presets the code
// actually exposes. Preset *existence* is decided structurally by discovering the exported
// preset functions in the dimse source (DiscoverCodePresets), so a preset added to the code is
// surfaced even if nobody updates the count registry. The *counts* are then compared against
// codeCounts, the live len() of each shipped preset. The reconciliation:
//
//   - a shipped (not-deferred) preset named in the table that the code does not define is a
//     preset-missing finding;
//   - a preset the table marks NOT YET SHIPPED that the code does define is a preset-unexpected
//     finding (a deferred surface was shipped without updating the statement);
//   - a preset the code defines that the table does not name is a preset-unexpected finding (a
//     code-only preset that escaped the statement);
//   - a shipped, defined, documented preset whose live count differs from the documented count
//     is a preset-count finding. If such a preset has no live counter wired into codeCounts the
//     count cannot be checked, which is itself reported so the registry cannot quietly fall
//     behind the code.
func checkPresetCounts(root string, codeCounts map[string]PresetCounter) ([]Finding, error) {
	claims, err := ParsePresetClaims(filepath.Join(root, "docs", "conformance", "dicom.md"))
	if err != nil {
		return nil, err
	}
	if len(claims) == 0 {
		return nil, fmt.Errorf("no preset claims parsed from dicom.md: the preset-count table is missing or its format changed")
	}

	codePresets, err := DiscoverCodePresets(filepath.Join(root, "dimse"))
	if err != nil {
		return nil, err
	}

	var findings []Finding
	documented := make(map[string]bool, len(claims))
	for _, claim := range claims {
		documented[claim.Name] = true
		inCode := codePresets[claim.Name]

		if claim.NotYetShipped {
			if inCode {
				findings = append(findings, Finding{
					Class:   "preset-unexpected",
					Subject: claim.Name,
					Detail:  "documented as NOT YET SHIPPED but is implemented in code",
				})
			}
			continue
		}

		if !inCode {
			findings = append(findings, Finding{
				Class:   "preset-missing",
				Subject: claim.Name,
				Detail:  fmt.Sprintf("named in the dicom.md preset table (%d contexts) but not defined in the dimse package", claim.Count),
			})
			continue
		}

		counter := codeCounts[claim.Name]
		if counter == nil {
			findings = append(findings, Finding{
				Class:   "preset-count",
				Subject: claim.Name,
				Detail:  fmt.Sprintf("dicom.md claims %d contexts but no live count is wired into the check for this preset", claim.Count),
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

	for name := range codePresets {
		if !documented[name] {
			findings = append(findings, Finding{
				Class:   "preset-unexpected",
				Subject: name,
				Detail:  "defined in the dimse package but absent from the dicom.md preset table",
			})
		}
	}

	return findings, nil
}

// presetReturnType is the slice element type a presentation-context preset returns; an exported
// dimse function with this result and no parameters is treated as a preset for discovery.
const presetReturnType = "PresentationContext"

// DiscoverCodePresets parses the Go source in dimseDir and returns the set of exported,
// parameterless functions that return []PresentationContext — the code-side preset surface. It
// reads the source rather than a hand-maintained list so the existence checks reflect the code
// itself. Test files are skipped.
func DiscoverCodePresets(dimseDir string) (map[string]bool, error) {
	entries, err := os.ReadDir(dimseDir)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	presets := make(map[string]bool)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dimseDir, e.Name()), nil, 0)
		if err != nil {
			return nil, err
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !fn.Name.IsExported() {
				continue
			}
			if isPresetSignature(fn.Type) {
				presets[fn.Name.Name] = true
			}
		}
	}
	return presets, nil
}

// isPresetSignature reports whether fn takes no parameters and returns a single
// []PresentationContext result, the shape every preset helper shares.
func isPresetSignature(fn *ast.FuncType) bool {
	if fn.Params != nil && len(fn.Params.List) != 0 {
		return false
	}
	if fn.Results == nil || len(fn.Results.List) != 1 {
		return false
	}
	slice, ok := fn.Results.List[0].Type.(*ast.ArrayType)
	if !ok || slice.Len != nil {
		return false
	}
	ident, ok := slice.Elt.(*ast.Ident)
	return ok && ident.Name == presetReturnType
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

// checkBanners verifies every scaffold conformance statement opens with the NOT YET SHIPPED
// scaffold banner, so an unimplemented surface can never be silently presented as
// conformance-guaranteed by deleting the banner.
func checkBanners(root string) ([]Finding, error) {
	var findings []Finding
	for _, name := range scaffoldStatements {
		path := filepath.Join(root, "docs", "conformance", name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if !scaffoldBannerRE.Match(data) {
			findings = append(findings, Finding{
				Class:   "banner",
				Subject: name,
				Detail:  fmt.Sprintf("scaffold statement is missing the %q scaffold banner", notYetShippedBanner),
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
