#!/usr/bin/env python3
"""
Generate tag_values.go with all DICOM tag definitions.

This script fetches DICOM tag definitions from the innolitics JSON representation
and generates a Go file with exported variables and a TagDict map.
"""
import json
import logging
import urllib.request
from typing import IO, NamedTuple, List

logging.basicConfig(level=logging.DEBUG)

INNOLITICS_VERSION_HASH = "7f4749d09ed3ef2fa70637d376d423a4b13523cd"  # rev2024b

INNOLITICS_CREDITS = f'''// This file's contents are derived from the innolitics json representation of the dicom standard.
// The innolitics source is licensed as follows:
// https://github.com/innolitics/dicom-standard/blob/{INNOLITICS_VERSION_HASH}/LICENSE.txt
//
// Copyright (c) 2017 Innolitics, LLC.
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
// SOFTWARE.'''

Tag = NamedTuple('Tag', [
    ('group', int),
    ('elem', int),
    ('vr', List[str]),
    ('name', str),
    ('vm', str),
    ('keyword', str),
    ('retired', bool)])


def read_tags_from_innolitics(version_hash: str) -> List[Tag]:
    """Fetch DICOM tags from innolitics GitHub repository."""
    logging.info(f"Fetching tags from innolitics (version {version_hash[:8]})")
    response = urllib.request.urlopen(
        f"https://raw.githubusercontent.com/innolitics/dicom-standard/{version_hash}/standard/attributes.json"
    )
    attrs = json.loads(response.read().decode())
    allowed_vrs_separator = " or "

    tags = [
        Tag(
            # The id field should always follow format "ggggeeee", so this should be safe.
            group=int(resolvable_tag_id[:4], 16),
            elem=int(resolvable_tag_id[4:], 16),
            # To understand this ternary expression, see: https://dicom.nema.org/medical/dicom/2024a/output/html/part06.html#note_6_2.
            vr=(e["valueRepresentation"] if resolvable_tag_id[:4].lower() != 'fffe' else "NA").split(allowed_vrs_separator),
            name=e["name"],
            vm=e["valueMultiplicity"],
            keyword=e["keyword"],
            retired=e["retired"] == "Y")
        for e in attrs
        if (resolvable_tag_id := e["id"].replace("x", "0")) and len(e["keyword"]) > 0
    ]

    logging.info(f"Found {len(tags)} tags")
    return tags


def vr_string_to_go_const(vr: str) -> str:
    """Convert VR string to Go VR constant name."""
    vr_map = {
        "AE": "vr.ApplicationEntity",
        "AS": "vr.AgeString",
        "AT": "vr.AttributeTag",
        "CS": "vr.CodeString",
        "DA": "vr.Date",
        "DS": "vr.DecimalString",
        "DT": "vr.DateTime",
        "FD": "vr.FloatingPointDouble",
        "FL": "vr.FloatingPointSingle",
        "IS": "vr.IntegerString",
        "LO": "vr.LongString",
        "LT": "vr.LongText",
        "OB": "vr.OtherByte",
        "OD": "vr.OtherDouble",
        "OF": "vr.OtherFloat",
        "OL": "vr.OtherLong",
        "OV": "vr.OtherVeryLong",
        "OW": "vr.OtherWord",
        "PN": "vr.PersonName",
        "SH": "vr.ShortString",
        "SL": "vr.SignedLong",
        "SQ": "vr.SequenceOfItems",
        "SS": "vr.SignedShort",
        "ST": "vr.ShortText",
        "SV": "vr.SignedVeryLong",
        "TM": "vr.Time",
        "UC": "vr.UnlimitedCharacters",
        "UI": "vr.UniqueIdentifier",
        "UL": "vr.UnsignedLong",
        "UN": "vr.Unknown",
        "UR": "vr.UniversalResourceIdentifier",
        "US": "vr.UnsignedShort",
        "UT": "vr.UnlimitedText",
        "UV": "vr.UnsignedVeryLong",
        # Special case for Item tags
        "NA": "vr.Unknown",  # Item-related tags use UN as VR
    }
    return vr_map.get(vr, "vr.Unknown")


def tag_dict_entry(t: Tag) -> str:
    """Generate a TagDict entry for a single tag."""
    start_indent = '\t'
    vr_list = ", ".join(vr_string_to_go_const(v) for v in t.vr)
    return f'{start_indent}{t.keyword}: Info{{Tag: {t.keyword}, VRs: []vr.VR{{{vr_list}}}, Name: "{t.name}", Keyword: "{t.keyword}", VM: "{t.vm}", Retired: {str(t.retired).lower()}}},'


def generate(out: IO[str]):
    """Generate the complete tag_values.go file."""
    tags = read_tags_from_innolitics(INNOLITICS_VERSION_HASH)
    new_line = chr(10)

    print(f'''// AUTO-GENERATED from generate_tag_values.py. DO NOT EDIT.
{INNOLITICS_CREDITS}
package tag

import "github.com/harrison-ai/go-radx/dicom/vr"

// Standard DICOM tags as exported variables for convenient access.
// These are all tags defined in the DICOM standard Part 6, Chapter 6.
// https://dicom.nema.org/medical/dicom/current/output/html/part06.html#chapter_6
{new_line.join(f"var {t.keyword} = New(0x{t.group:04x}, 0x{t.elem:04x})" for t in tags)}

// TagDict is a map of all standard DICOM tags to their metadata.
// It provides VR information, names, keywords, value multiplicity, and retirement status.
var TagDict = map[Tag]Info{{
{new_line.join(tag_dict_entry(t) for t in tags)}
}}''', file=out)


def main():
    """Main entry point."""
    with open("tag_values.go", "w") as out:
        generate(out)
    logging.info("Successfully generated tag_values.go")


if __name__ == "__main__":
    main()