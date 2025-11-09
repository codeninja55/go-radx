#!/usr/bin/env python3
"""
Generate Go constant exports for all Transfer Syntax and SOP Class UIDs.

This script reads uid_values.go and generates two new files:
- transfer_syntax_uids.go: All Transfer Syntax UID constants
- sop_class_uids.go: All SOP Class UID constants
"""

import re
import sys
from pathlib import Path
from typing import List, Tuple


def to_go_constant_name(name: str) -> str:
    """
    Convert a DICOM UID name to a Go constant name.

    Example: "Implicit VR Little Endian" -> "ImplicitVRLittleEndian"
    """
    # Remove special characters and replace with spaces
    name = re.sub(r'[^\w\s]+', ' ', name)

    # Split into words
    words = name.split()

    # Special acronyms that should be uppercase
    acronyms = {
        'vr', 'uid', 'sop', 'ct', 'mr', 'us', 'pet', 'nm', 'xa', 'rf',
        'dx', 'mg', 'io', 'px', 'gm', 'sm', 'au', 'hd', 'sr', 'ko', 'pr',
        'rt', 'rwv', 'seg', 'fid', 'reg', 'jpeg', 'rle', 'mpeg', 'smpte',
        'dicom', 'hl7', 'ihe', 'iso', 'qr', 'mpps', 'gp', 'pps', 'pdf',
        'cda', 'stl', 'mtl', 'obj', 'iv', 'uv', 'roi', 'rtss', 'dvh',
        'mwl', 'ups', 'nud', 'fda', 'css', 'html', 'xml', 'json', 'uri'
    }

    result = []
    for word in words:
        word_lower = word.lower()
        if word_lower in acronyms:
            result.append(word.upper())
        elif word_lower.isdigit():
            result.append(word)
        else:
            # Capitalize first letter
            result.append(word.capitalize())

    const_name = ''.join(result)

    # Handle edge cases
    # If starts with a digit, prepend with 'UID'
    if const_name and const_name[0].isdigit():
        const_name = 'UID' + const_name

    return const_name


def parse_uid_map(uid_values_path: Path) -> List[Tuple[str, str, str, bool]]:
    """
    Parse uid_values.go and extract all UIDs from uidMap.

    Returns: List of (uid, name, type, retired) tuples
    """
    content = uid_values_path.read_text()

    # Find the uidMap declaration - match until the closing brace
    # Need to be careful with nested braces
    uid_map_start = content.find('var uidMap = map[string]Info{')
    if uid_map_start == -1:
        raise ValueError("Could not find uidMap in uid_values.go")

    # Find the closing brace by counting braces
    brace_count = 0
    start_pos = uid_map_start + len('var uidMap = map[string]Info')
    for i, char in enumerate(content[start_pos:], start=start_pos):
        if char == '{':
            brace_count += 1
        elif char == '}':
            brace_count -= 1
            if brace_count == 0:
                uid_map_content = content[uid_map_start:i+1]
                break
    else:
        raise ValueError("Could not find closing brace for uidMap")

    # Parse each UID entry
    # Format: "UID": {UID: "UID", Name: "NAME", Type: TypeXXX, Info: "INFO", Retired: bool},
    pattern = r'"([^"]+)":\s*\{\s*UID:\s*"[^"]*",\s*Name:\s*"([^"]*)",\s*Type:\s*(Type\w+),\s*Info:\s*"[^"]*",\s*Retired:\s*(true|false)\s*\}'

    uids = []
    for match in re.finditer(pattern, uid_map_content):
        uid = match.group(1)
        name = match.group(2)
        uid_type = match.group(3).strip()
        retired = match.group(4) == 'true'
        uids.append((uid, name, uid_type, retired))

    return uids


def generate_transfer_syntax_file(uids: List[Tuple[str, str, str, bool]], output_path: Path):
    """Generate transfer_syntax_uids.go with all Transfer Syntax UID constants."""
    # Filter for Transfer Syntax UIDs
    transfer_syntaxes = [(uid, name, retired) for uid, name, typ, retired in uids
                         if typ == 'TypeTransferSyntax']

    # Sort by UID
    transfer_syntaxes.sort(key=lambda x: x[0])

    # Generate constants
    lines = [
        '// AUTO-GENERATED - DO NOT EDIT',
        '// Generated from DICOM PS3.6 Part 6 - Data Dictionary',
        '// DICOM Standard Version: 2024b',
        '//',
        '// This file contains all Transfer Syntax UID constants for convenient access.',
        '// Total: {} Transfer Syntax UIDs'.format(len(transfer_syntaxes)),
        '',
        'package uid',
        '',
        '// Transfer Syntax UIDs',
        '// These are all the transfer syntaxes defined in the DICOM standard.',
        'var (',
    ]

    # Track duplicate constant names
    seen_names = {}
    skipped = 0

    for uid, name, retired in transfer_syntaxes:
        # Skip UIDs with empty names
        if not name or name.strip() == '':
            skipped += 1
            continue

        const_name = to_go_constant_name(name)

        # Handle duplicates
        if const_name in seen_names:
            # Append UID suffix for uniqueness
            suffix = uid.replace('.', '_')
            const_name = const_name + '_' + suffix.split('_')[-1]
        seen_names[const_name] = uid

        # Add deprecation notice for retired UIDs
        if retired:
            lines.append(f'\t// {name} (RETIRED)')
            lines.append(f'\t//')
            lines.append(f'\t// Deprecated: This UID has been retired from the DICOM standard.')
        else:
            lines.append(f'\t// {name}')
        lines.append(f'\t{const_name} = MustParse("{uid}")')
        lines.append('')

    lines.append(')')
    lines.append('')

    output_path.write_text('\n'.join(lines))
    print(f'Generated {output_path} with {len(transfer_syntaxes) - skipped} Transfer Syntax UIDs')
    if skipped > 0:
        print(f'  (Skipped {skipped} UIDs with empty names)')


def generate_sop_class_file(uids: List[Tuple[str, str, str, bool]], output_path: Path):
    """Generate sop_class_uids.go with all SOP Class UID constants."""
    # Filter for SOP Class UIDs
    sop_classes = [(uid, name, retired) for uid, name, typ, retired in uids
                   if typ in ('TypeSOPClass', 'TypeMetaSOPClass')]

    # Sort by UID
    sop_classes.sort(key=lambda x: x[0])

    # Generate constants
    lines = [
        '// AUTO-GENERATED - DO NOT EDIT',
        '// Generated from DICOM PS3.6 Part 6 - Data Dictionary',
        '// DICOM Standard Version: 2024b',
        '//',
        '// This file contains all SOP Class UID constants for convenient access.',
        '// Total: {} SOP Class UIDs'.format(len(sop_classes)),
        '',
        'package uid',
        '',
        '// SOP Class UIDs (including Meta SOP Classes)',
        '// These are all the SOP classes defined in the DICOM standard.',
        'var (',
    ]

    # Track duplicate constant names
    seen_names = {}
    skipped = 0

    for uid, name, retired in sop_classes:
        # Skip UIDs with empty names
        if not name or name.strip() == '':
            skipped += 1
            continue

        const_name = to_go_constant_name(name)

        # Handle duplicates
        if const_name in seen_names:
            # Append UID suffix for uniqueness
            suffix = uid.replace('.', '_')
            const_name = const_name + '_' + suffix.split('_')[-1]
        seen_names[const_name] = uid

        # Add deprecation notice for retired UIDs
        if retired:
            lines.append(f'\t// {name} (RETIRED)')
            lines.append(f'\t//')
            lines.append(f'\t// Deprecated: This UID has been retired from the DICOM standard.')
        else:
            lines.append(f'\t// {name}')
        lines.append(f'\t{const_name} = MustParse("{uid}")')
        lines.append('')

    lines.append(')')
    lines.append('')

    output_path.write_text('\n'.join(lines))
    print(f'Generated {output_path} with {len(sop_classes) - skipped} SOP Class UIDs')
    if skipped > 0:
        print(f'  (Skipped {skipped} UIDs with empty names)')


def main():
    # Get script directory
    script_dir = Path(__file__).parent

    # Input file
    uid_values_path = script_dir / 'uid_values.go'
    if not uid_values_path.exists():
        print(f'Error: {uid_values_path} not found', file=sys.stderr)
        sys.exit(1)

    # Parse UIDs
    print(f'Parsing {uid_values_path}...')
    uids = parse_uid_map(uid_values_path)
    print(f'Found {len(uids)} total UIDs')

    # Generate Transfer Syntax file
    transfer_syntax_path = script_dir / 'transfer_syntax_uids.go'
    generate_transfer_syntax_file(uids, transfer_syntax_path)

    # Generate SOP Class file
    sop_class_path = script_dir / 'sop_class_uids.go'
    generate_sop_class_file(uids, sop_class_path)

    print('Done!')


if __name__ == '__main__':
    main()
