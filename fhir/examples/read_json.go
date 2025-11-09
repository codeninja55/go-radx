//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/codeninja55/go-radx/fhir/r4/resources"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <fhir-json-file>\n", os.Args[0])
		os.Exit(1)
	}

	filename := os.Args[1]

	// Read the FHIR JSON file
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Unmarshal into Patient struct
	var patient resources.Patient
	if err := json.Unmarshal(data, &patient); err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshaling JSON: %v\n", err)
		os.Exit(1)
	}

	// Access and print patient information
	fmt.Println("=== Patient Information ===")

	if patient.ID != nil {
		fmt.Printf("ID: %s\n", *patient.ID)
	}

	if patient.Active != nil {
		fmt.Printf("Active: %v\n", *patient.Active)
	}

	// Print names
	if len(patient.Name) > 0 {
		fmt.Println("\nNames:")
		for i, name := range patient.Name {
			fmt.Printf("  %d. ", i+1)
			if name.Use != nil {
				fmt.Printf("[%s] ", *name.Use)
			}
			if len(name.Given) > 0 {
				for _, given := range name.Given {
					fmt.Printf("%s ", given)
				}
			}
			if name.Family != nil {
				fmt.Printf("%s", *name.Family)
			}
			fmt.Println()
		}
	}

	// Print gender
	if patient.Gender != nil {
		fmt.Printf("\nGender: %s\n", *patient.Gender)
	}

	// Print birth date with precision info
	if patient.BirthDate != nil {
		fmt.Printf("\nBirth Date: %s\n", patient.BirthDate.String())
		fmt.Printf("Precision: %s\n", patient.BirthDate.Precision())

		// Convert to time.Time
		if t, err := patient.BirthDate.Time(); err == nil {
			fmt.Printf("As time.Time: %v\n", t)
		}
	}

	// Print contact information
	if len(patient.Telecom) > 0 {
		fmt.Println("\nContact Information:")
		for i, telecom := range patient.Telecom {
			fmt.Printf("  %d. ", i+1)
			if telecom.System != nil {
				fmt.Printf("%s: ", *telecom.System)
			}
			if telecom.Value != nil {
				fmt.Printf("%s", *telecom.Value)
			}
			if telecom.Use != nil {
				fmt.Printf(" (%s)", *telecom.Use)
			}
			fmt.Println()
		}
	}

	// Print addresses
	if len(patient.Address) > 0 {
		fmt.Println("\nAddresses:")
		for i, addr := range patient.Address {
			fmt.Printf("  %d. ", i+1)
			if addr.Use != nil {
				fmt.Printf("[%s] ", *addr.Use)
			}
			if len(addr.Line) > 0 {
				fmt.Printf("%s, ", addr.Line[0])
			}
			if addr.City != nil {
				fmt.Printf("%s, ", *addr.City)
			}
			if addr.State != nil {
				fmt.Printf("%s ", *addr.State)
			}
			if addr.PostalCode != nil {
				fmt.Printf("%s", *addr.PostalCode)
			}
			fmt.Println()
		}
	}

	// Print identifiers
	if len(patient.Identifier) > 0 {
		fmt.Println("\nIdentifiers:")
		for i, id := range patient.Identifier {
			fmt.Printf("  %d. ", i+1)
			if id.System != nil {
				fmt.Printf("System: %s, ", *id.System)
			}
			if id.Value != nil {
				fmt.Printf("Value: %s", *id.Value)
			}
			if id.Use != nil {
				fmt.Printf(" [%s]", *id.Use)
			}
			fmt.Println()
		}
	}

	// Print emergency contacts
	if len(patient.Contact) > 0 {
		fmt.Println("\nEmergency Contacts:")
		for i, contact := range patient.Contact {
			fmt.Printf("  %d. ", i+1)
			if contact.Name != nil {
				if len(contact.Name.Given) > 0 {
					fmt.Printf("%s ", contact.Name.Given[0])
				}
				if contact.Name.Family != nil {
					fmt.Printf("%s", *contact.Name.Family)
				}
			}
			if len(contact.Telecom) > 0 && contact.Telecom[0].Value != nil {
				fmt.Printf(" - %s", *contact.Telecom[0].Value)
			}
			fmt.Println()
		}
	}
}
