#!/usr/bin/env python3
"""Round-trip vendored DICOM fixtures through pydicom and assert byte stability.

For each canonical Explicit VR Little Endian fixture, the file is read with
pydicom and written back out; the re-encoded bytes must match the original
exactly. This guards the reader/writer contract that later increments rely on.

Behaviour when pydicom is not installed:
  * On a developer machine: print "SKIP: pydicom not installed" and exit 0, so
    the suite is green without pydicom.
  * In CI (CI=true): exit non-zero, so a missing gate cannot silently rot
    (PRD PS3 conformance, PRD section 11.1).

Usage:
    python3 tools/dicom-conformance/pydicom_roundtrip.py testdata/dicom
"""

import io
import os
import sys

# Explicit VR Little Endian. Only files with this Transfer Syntax are expected
# to be byte-stable through a pydicom read/write cycle.
EXPLICIT_VR_LE = "1.2.840.10008.1.2.1"


def main(argv):
    fixture_dir = argv[1] if len(argv) > 1 else "testdata/dicom"

    try:
        import pydicom
    except ImportError:
        msg = "SKIP: pydicom not installed (pip install pydicom to run the round-trip gate)"
        if os.environ.get("CI") == "true":
            print("FAIL: " + msg, file=sys.stderr)
            print(
                "FAIL: CI=true requires pydicom; the conformance gate must run in CI.",
                file=sys.stderr,
            )
            return 1
        print(msg)
        return 0

    if not os.path.isdir(fixture_dir):
        print(f"FAIL: fixture directory not found: {fixture_dir}", file=sys.stderr)
        return 1

    files = sorted(f for f in os.listdir(fixture_dir) if f.endswith(".dcm"))
    checked = 0
    failures = []

    for name in files:
        path = os.path.join(fixture_dir, name)
        with open(path, "rb") as fh:
            original = fh.read()

        ds = pydicom.dcmread(path)
        ts = getattr(ds.file_meta, "TransferSyntaxUID", None)
        if str(ts) != EXPLICIT_VR_LE:
            # Only canonical Explicit VR LE is asserted byte-stable.
            continue

        checked += 1
        buf = io.BytesIO()
        ds.save_as(buf, enforce_file_format=True)
        if buf.getvalue() != original:
            failures.append(name)

    if not files:
        print(f"SKIP: no .dcm files in {fixture_dir}")
        return 0

    if failures:
        for name in failures:
            print(f"FAIL: not byte-stable after round-trip: {name}", file=sys.stderr)
        return 1

    print(f"OK: {checked} Explicit VR LE fixture(s) round-tripped byte-stable")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
