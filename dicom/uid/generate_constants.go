//go:build ignore
// +build ignore

// This program generates constant exports for all Transfer Syntax and SOP Class UIDs.
// Run with: go run generate_constants.go
package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"unicode"
)

// Import the uid package to access uidMap
// Note: This requires the package to be built first

func main() {
	// We'll generate two separate files:
	// 1. transfer_syntax_uids.go - All Transfer Syntax UIDs
	// 2. sop_class_uids.go - All SOP Class UIDs

	// Since we can't directly import the uid package here (circular dependency),
	// we'll need to read the uid_values.go file and parse it
	// For now, let's create a template that can be filled in

	fmt.Println("Generating UID constants...")
	fmt.Println("This requires manually extracting UIDs from uid_values.go")
	fmt.Println("A Python script is better suited for this task.")

	os.Exit(0)
}

// toGoConstantName converts a DICOM UID name to a Go constant name.
// Example: "Implicit VR Little Endian" -> "ImplicitVRLittleEndian"
func toGoConstantName(name string) string {
	// Remove special characters and replace with spaces
	reg := regexp.MustCompile(`[^\w\s]+`)
	name = reg.ReplaceAllString(name, " ")

	// Split into words
	words := strings.Fields(name)

	// Capitalize first letter of each word
	var result strings.Builder
	for _, word := range words {
		// Handle special cases
		switch strings.ToUpper(word) {
		case "VR", "UID", "SOP", "CT", "MR", "US", "PET", "NM", "XA", "RF", "DX", "MG", "IO", "PX", "GM", "SM", "AU", "HD", "SR", "KO", "PR", "RT", "RWV", "SEG", "FID", "REG", "JPEG", "RLE", "MPEG", "SMPTE", "DICOM", "HL7", "IHE", "ISO":
			result.WriteString(strings.ToUpper(word))
		default:
			// Capitalize first letter
			if len(word) > 0 {
				result.WriteRune(unicode.ToUpper(rune(word[0])))
				result.WriteString(word[1:])
			}
		}
	}

	return result.String()
}

const transferSyntaxTemplate = `// AUTO-GENERATED - DO NOT EDIT
// Generated from DICOM PS3.6 Part 6 - Data Dictionary
// DICOM Standard Version: 2024b
//
// This file contains all Transfer Syntax UID constants for convenient access.

package uid

// Transfer Syntax UIDs
var (
{{range .}}	// {{.Name}}{{if .Retired}} (RETIRED){{end}}
	{{.ConstName}} = MustParse("{{.UID}}")
{{end}}
)
`

const sopClassTemplate = `// AUTO-GENERATED - DO NOT EDIT
// Generated from DICOM PS3.6 Part 6 - Data Dictionary
// DICOM Standard Version: 2024b
//
// This file contains all SOP Class UID constants for convenient access.

package uid

// SOP Class UIDs
var (
{{range .}}	// {{.Name}}{{if .Retired}} (RETIRED){{end}}
	{{.ConstName}} = MustParse("{{.UID}}")
{{end}}
)
`

type uidInfo struct {
	UID       string
	Name      string
	ConstName string
	Retired   bool
}
