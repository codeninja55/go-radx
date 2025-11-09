#!/usr/bin/env python3
"""
Generate pkg/uid/uid_definitions.go from DICOM Part 6 XML.

This script downloads the DICOM Standard Part 6 XML file and extracts UID definitions
from Tables A-1 (UID Values) and A-2 (Well-known Frames of Reference).

Based on pydicom's generate_uid_dict.py:
https://github.com/pydicom/pydicom/blob/main/util/generate_dict/generate_uid_dict.py

Usage:
    python3 generate_uid_definitions.py
    python3 generate_uid_definitions.py --local /path/to/dicom/xml
"""

import argparse
import re
import sys
from pathlib import Path
from urllib import request
import xml.etree.ElementTree as ET

DICOM_VERSION = "2024b"
XML_URL = "https://dicom.nema.org/medical/dicom/current/source/docbook/part06/part06.xml"
BR = "{http://docbook.org/ns/docbook}"

# Output file configuration
SCRIPT_DIR = Path(__file__).parent
OUTPUT_FILE = SCRIPT_DIR / "uid_definitions.go"
PACKAGE_NAME = "uid"

# MIT License text for innolitics compatibility (though we're using official DICOM)
MIT_LICENSE = """// Copyright (c) 2017 Innolitics, LLC.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
// of the Software, and to permit persons to whom the Software is furnished to do
// so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE."""


def sanitize_keyword(keyword):
    """
    Convert a UID Type string to a valid Go identifier with proper capitalization.

    Examples:
        "SOP Class" -> "SOPClass"
        "Transfer Syntax" -> "TransferSyntax"
        "Well-known SOP Instance" -> "WellKnownSOPInstance"
        "Well-known frame of reference" -> "WellKnownFrameOfReference"
    """
    # Preserve known acronyms
    acronyms = {
        "sop": "SOP",
        "dicom": "DICOM",
        "ldap": "LDAP",
        "oid": "OID",
        "uid": "UID",
        "uids": "UIDs",
    }

    # Split by word boundaries (spaces, hyphens, slashes, etc.)
    words = re.split(r'[\s\-/\.]+', keyword)

    # Capitalize each word, preserving acronyms
    result_words = []
    for word in words:
        if word:
            lower_word = word.lower()
            if lower_word in acronyms:
                result_words.append(acronyms[lower_word])
            else:
                result_words.append(word.capitalize())

    return ''.join(result_words)


def parse_row(column_names, row):
    """
    Parse a table row from the XML.

    Returns dict with column_name: value mappings.
    """
    cell_values = []
    for cell in row.iter(f"{BR}para"):
        # Check for emphasis tag
        emph_value = cell.find(f"{BR}emphasis")
        if emph_value is not None:
            if emph_value.text is not None:
                cell_values.append(emph_value.text.strip().replace("\u200b", ""))
            else:
                cell_values.append("")
        else:
            if cell.text is not None:
                cell_values.append(cell.text.strip().replace("\u200b", ""))
            else:
                cell_values.append("")

    # Pad with empty string if needed
    while len(cell_values) < len(column_names):
        cell_values.append("")

    return {k: v for k, v in zip(column_names, cell_values)}


def parse_table(root, labels, caption):
    """
    Find and parse a table by caption.

    Returns list of dicts with parsed row data.
    """
    for table in root.iter(f"{BR}table"):
        caption_elem = table.find(f"{BR}caption")
        if caption_elem is not None and caption_elem.text == caption:
            tbody = table.find(f"{BR}tbody")
            if tbody is None:
                continue
            return [parse_row(labels, row) for row in tbody.iter(f"{BR}tr")]

    raise ValueError(f"No table found with caption: {caption}")


def parse_uid_values_table(root):
    """
    Parse Table A-1: UID Values

    Returns list of UID info dicts.
    """
    labels = ["UID Value", "UID Name", "UID Keyword", "UID Type", "UID Info", "Retired"]
    attrs = parse_table(root, labels, "UID Values")

    # Post-process
    for attr in attrs:
        name = attr["UID Name"]

        # Handle "(Retired)" in name
        if "(Retired)" in name:
            attr["Retired"] = "Retired"
            attr["UID Name"] = name.replace("(Retired)", "").strip()

        # Split name and info if colon present
        if ":" in name:
            parts = name.split(":", 1)
            attr["UID Name"] = parts[0].strip()
            attr["UID Info"] = parts[1].strip()

    return attrs


def parse_frames_of_reference_table(root):
    """
    Parse Table A-2: Well-known Frames of Reference

    Returns list of UID info dicts.
    """
    labels = ["UID Value", "UID Name", "UID Keyword", "Normative Reference"]
    attrs = parse_table(root, labels, "Well-known Frames of Reference")

    # Post-process
    for attr in attrs:
        attr["UID Type"] = "Well-known frame of reference"
        attr["UID Info"] = ""
        attr["Retired"] = ""
        del attr["Normative Reference"]

    return attrs


def generate_type_enum():
    """Generate the Type enum definition."""
    return '''// Type represents the category of a DICOM UID.
type Type string

const (
	TypeSOPClass                        Type = "SOP Class"
	TypeMetaSOPClass                    Type = "Meta SOP Class"
	TypeTransferSyntax                  Type = "Transfer Syntax"
	TypeWellKnownSOPInstance            Type = "Well-known SOP Instance"
	TypeWellKnownFrameOfReference       Type = "Well-known frame of reference"
	TypeCodingScheme                    Type = "Coding Scheme"
	TypeApplicationContextName          Type = "Application Context Name"
	TypeServiceClass                    Type = "Service Class"
	TypeDICOMUIDsAsACodingScheme        Type = "DICOM UIDs as a Coding Scheme"
	TypeLDAPOID                         Type = "LDAP OID"
	TypeSynchronizationFrameOfReference Type = "Synchronization Frame of Reference"
	TypeApplicationHostingModel         Type = "Application Hosting Model"
	TypeMappingResource                 Type = "Mapping Resource"
)
'''


def generate_info_struct():
    """Generate the Info struct definition."""
    return '''// Info holds metadata about a DICOM UID.
//
// Reference: DICOM PS3.6 Annex A - Registry of DICOM unique identifiers (UIDs)
// https://dicom.nema.org/medical/dicom/current/output/html/part06.html#chapter_A
type Info struct {
	UID     string // The UID value, e.g., "1.2.840.10008.1.2"
	Name    string // Human-readable name, e.g., "Implicit VR Little Endian"
	Type    Type   // Category of the UID
	Info    string // Additional information (optional)
	Retired bool   // True if the UID has been retired
}
'''


def generate_common_constants(attrs):
    """
    Generate exported constants for commonly-used UIDs.

    Focuses on Transfer Syntaxes and important SOP Classes.
    """
    # Define which UIDs to export as constants
    important_uids = {
        # Transfer Syntaxes
        "1.2.840.10008.1.2": "ImplicitVRLittleEndian",
        "1.2.840.10008.1.2.1": "ExplicitVRLittleEndian",
        "1.2.840.10008.1.2.2": "ExplicitVRBigEndian",
        "1.2.840.10008.1.2.1.99": "DeflatedExplicitVRLittleEndian",
        "1.2.840.10008.1.2.5": "RLELossless",
        "1.2.840.10008.1.2.4.50": "JPEGBaseline8Bit",
        "1.2.840.10008.1.2.4.51": "JPEGExtended12Bit",
        "1.2.840.10008.1.2.4.57": "JPEGLossless",
        "1.2.840.10008.1.2.4.70": "JPEGLosslessSV1",
        "1.2.840.10008.1.2.4.80": "JPEGLSLossless",
        "1.2.840.10008.1.2.4.81": "JPEGLSNearLossless",
        "1.2.840.10008.1.2.4.90": "JPEG2000Lossless",
        "1.2.840.10008.1.2.4.91": "JPEG2000",
        # Common SOP Classes
        "1.2.840.10008.1.1": "VerificationSOPClass",
        "1.2.840.10008.5.1.4.1.1.1": "ComputedRadiographyImageStorage",
        "1.2.840.10008.5.1.4.1.1.2": "CTImageStorage",
        "1.2.840.10008.5.1.4.1.1.4": "MRImageStorage",
        "1.2.840.10008.5.1.4.1.1.7": "SecondaryCaptureImageStorage",
        # Query/Retrieve
        "1.2.840.10008.5.1.4.1.2.1.1": "PatientRootQRFind",
        "1.2.840.10008.5.1.4.1.2.1.2": "PatientRootQRMove",
        "1.2.840.10008.5.1.4.1.2.1.3": "PatientRootQRGet",
        "1.2.840.10008.5.1.4.1.2.2.1": "StudyRootQRFind",
        "1.2.840.10008.5.1.4.1.2.2.2": "StudyRootQRMove",
        "1.2.840.10008.5.1.4.1.2.2.3": "StudyRootQRGet",
        # Modality Worklist
        "1.2.840.10008.5.1.4.31": "ModalityWorklistInformationFind",
    }

    lines = ['// Common DICOM UIDs exported as package-level variables for convenient access.', 'var (']

    # Create map for quick lookup
    uid_map = {attr["UID Value"]: attr for attr in attrs}

    for uid, name in sorted(important_uids.items()):
        if uid in uid_map:
            attr = uid_map[uid]
            comment = attr["UID Name"]
            if attr["Retired"]:
                comment += " (Retired)"
            lines.append(f'\t// {name} - {comment}')
            lines.append(f'\t{name} = MustParse("{uid}")')
            lines.append('')

    lines.append(')')

    return '\n'.join(lines)


def generate_uid_map(attrs):
    """Generate the uidMap variable."""
    lines = ['// uidMap contains metadata for all standard DICOM UIDs.', 'var uidMap = map[string]Info{']

    for attr in attrs:
        uid = attr["UID Value"]
        name = attr["UID Name"].replace('"', '\\"')
        uid_type = attr["UID Type"]
        info = attr.get("UID Info", "").replace('"', '\\"')
        retired = "true" if attr["Retired"] == "Retired" else "false"

        # Single line format
        lines.append(f'\t"{uid}": {{UID: "{uid}", Name: "{name}", Type: Type{sanitize_keyword(uid_type)}, Info: "{info}", Retired: {retired}}},')

    lines.append('}')

    return '\n'.join(lines)


def generate_helper_functions():
    """Generate helper functions for UID lookup."""
    return '''// Lookup returns the Info for the given UID string.
// Returns false if the UID is not found in the standard dictionary.
func Lookup(uid string) (Info, bool) {
	info, ok := uidMap[uid]
	return info, ok
}

// Name returns the human-readable name for the given UID.
// Returns empty string if the UID is not found.
func Name(uid string) string {
	if info, ok := uidMap[uid]; ok {
		return info.Name
	}
	return ""
}

// IsRetired returns true if the given UID has been retired from the DICOM standard.
// Returns false if the UID is not found or is not retired.
func IsRetired(uid string) bool {
	if info, ok := uidMap[uid]; ok {
		return info.Retired
	}
	return false
}

// GetType returns the Type category for the given UID.
// Returns empty Type if the UID is not found.
func GetType(uid string) Type {
	if info, ok := uidMap[uid]; ok {
		return info.Type
	}
	return ""
}

// IsTransferSyntax returns true if the given UID represents a Transfer Syntax.
func IsTransferSyntax(uid string) bool {
	if info, ok := uidMap[uid]; ok {
		return info.Type == TypeTransferSyntax
	}
	return false
}

// IsSOPClass returns true if the given UID represents a SOP Class.
func IsSOPClass(uid string) bool {
	if info, ok := uidMap[uid]; ok {
		return info.Type == TypeSOPClass || info.Type == TypeMetaSOPClass
	}
	return false
}
'''


def write_output_file(attrs, output_path):
    """Generate and write the complete Go source file."""
    with open(output_path, 'w') as f:
        # File header
        f.write(f'// AUTO-GENERATED by {Path(__file__).name}. DO NOT EDIT.\n')
        f.write('// Generated from DICOM PS3.6 Part 6 - Data Dictionary\n')
        f.write(f'// DICOM Standard Version: {DICOM_VERSION}\n')
        f.write('// Source: https://dicom.nema.org/medical/dicom/current/source/docbook/part06/part06.xml\n')
        f.write('//\n')
        f.write(MIT_LICENSE)
        f.write('\n\n')

        f.write(f'package {PACKAGE_NAME}\n\n')

        # Type enum
        f.write(generate_type_enum())
        f.write('\n')

        # Info struct
        f.write(generate_info_struct())
        f.write('\n')

        # Common constants
        f.write(generate_common_constants(attrs))
        f.write('\n\n')

        # UID map
        f.write(generate_uid_map(attrs))
        f.write('\n\n')

        # Helper functions
        f.write(generate_helper_functions())


def setup_argparse():
    """Configure command-line argument parsing."""
    parser = argparse.ArgumentParser(
        description="Generate uid_definitions.go from DICOM PS3.6 XML",
        usage="generate_uid_definitions.py [options]"
    )

    parser.add_argument(
        "--local",
        help="Path to directory containing part06.xml (instead of downloading)",
        type=str,
    )

    return parser.parse_args()


def main():
    """Main execution function."""
    args = setup_argparse()

    # Get XML source
    if args.local:
        print(f"Using local XML from: {args.local}")
        part06_path = Path(args.local) / "part06.xml"
        if not part06_path.exists():
            print(f"Error: File not found: {part06_path}", file=sys.stderr)
            sys.exit(1)
        tree = ET.parse(part06_path)
    else:
        print(f"Downloading: {XML_URL}")
        try:
            with request.urlopen(XML_URL) as response:
                tree = ET.parse(response)
            print("Download complete, processing...")
        except Exception as e:
            print(f"Error downloading XML: {e}", file=sys.stderr)
            sys.exit(1)

    root = tree.getroot()

    # Parse tables
    print("Parsing Table A-1: UID Values...")
    uid_values = parse_uid_values_table(root)
    print(f"  Found {len(uid_values)} UID entries")

    print("Parsing Table A-2: Well-known Frames of Reference...")
    frames_of_ref = parse_frames_of_reference_table(root)
    print(f"  Found {len(frames_of_ref)} frame of reference entries")

    # Combine all UIDs
    all_uids = uid_values + frames_of_ref
    print(f"Total UIDs: {len(all_uids)}")

    # Clean up data
    for attr in all_uids:
        # Replace ampersands
        attr["UID Name"] = attr["UID Name"].replace("&", "and")
        # Remove soft hyphens
        attr["UID Value"] = attr["UID Value"].replace("\u00ad", "")

    # Generate output
    print(f"Writing output to: {OUTPUT_FILE}")
    write_output_file(all_uids, OUTPUT_FILE)

    print(f"✓ Successfully generated {OUTPUT_FILE}")
    print(f"  Total UIDs: {len(all_uids)}")
    print(f"  Transfer Syntaxes: {sum(1 for a in all_uids if 'Transfer Syntax' in a['UID Type'])}")
    print(f"  SOP Classes: {sum(1 for a in all_uids if 'SOP Class' in a['UID Type'])}")
    print(f"  Retired UIDs: {sum(1 for a in all_uids if a['Retired'] == 'Retired')}")


if __name__ == "__main__":
    main()