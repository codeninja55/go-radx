# DICOM test fixtures

This directory holds a small, purposeful subset of vendored DICOM fixtures. Each file is chosen so a specific increment
of the DICOM data-layer plan has exactly the coverage it needs. Fixtures are test corpora only, vendored with upstream
license attribution per PRD §5.3 and §11.1.

Per PRD §9.1, committed fixtures carry no Protected Health Information that identifies a real patient: every upstream
file below was de-identified, synthetic, or binary-edited to remove patient identifiers before publication, and no
fixture filename encodes PHI.

## Provenance and coverage

The Transfer Syntax column is the actual `(0002,0010)` UID read from each committed file, not the upstream filename
convention. Origin records which upstream corpus the file came from; License records the governing license for that
origin (see the `LICENSE-*.txt` files in this directory).

| File | Transfer Syntax | Origin | License | Exercises |
| --- | --- | --- | --- | --- |
| `liver.dcm` | Explicit VR Little Endian (`1.2.840.10008.1.2.1`) | pydicom (DICOM SEG generated with dcmqi, contributed by Andrey Fedorov) | MIT (pydicom) | Canonical round-trip byte-stability (Increment 2) |
| `MR2_UNCI.dcm` | Explicit VR Little Endian (`1.2.840.10008.1.2.1`) | pydicom `MR2_*` family, derived from NEMA WG04 datasets | MIT (pydicom) | Nested sequence (`SQ`/`Item`) handling (Increment 3) |
| `SC_rgb_expb.dcm` | Explicit VR Big Endian, retired (`1.2.840.10008.1.2.2`) | pydicom `SC_rgb_*` family; Big Endian variant produced with dcmtk `dcmconv` | MIT (pydicom); dcmtk tooling | Byte-order handling (Increment 2, DCM-002) |
| `liver_rle.dcm` | RLE Lossless (`1.2.840.10008.1.2.5`) | pydicom liver corpus; RLE variant produced with GDCM `gdcmconv` | MIT (pydicom); GDCM tooling | Pure-Go RLE pixel decode (Increment 6) |
| `liver_j2k.dcm` | JPEG 2000 Lossless (`1.2.840.10008.1.2.4.90`) | pydicom liver corpus, JPEG 2000 variant | MIT (pydicom) | CGo-gated decode and `ErrCodecUnavailable` under `nocgo` (Increment 6) |
| `MR-SIEMENS-DICOM-WithOverlays.dcm` | Explicit VR Little Endian (`1.2.840.10008.1.2.1`) | GDCM | BSD-3-Clause (GDCM) | Overlays and repeating group `60xx` dictionary mask (Increment 1, DCM-012) |
| `explicit_VR-UN.dcm` | JPEG 2000 Lossless (`1.2.840.10008.1.2.4.90`) | pydicom (issue #968); original from The Cancer Imaging Archive, de-identified, recompressed with `gdcmconv --j2k` | MIT (pydicom) | Ambiguous/`UN` VR resolution (nearly all tags carry VR `UN`) (Increment 1) |

## Notes on upstream filename conventions

- `MR2_UNCI.dcm`: the `UNCI` suffix marks the uncompressed pydicom variant. The committed file is encoded as Explicit VR
  Little Endian, not Implicit VR Little Endian.
- `explicit_VR-UN.dcm`: the filename refers to its near-universal use of VR `UN`. Its Transfer Syntax is JPEG 2000
  Lossless because the upstream file was recompressed with `gdcmconv --j2k`; the `UN` VRs are the property under test.

## Upstream sources

- pydicom test data: <https://github.com/pydicom/pydicom> and the `pydicom-data` external store. Provenance for
  individual files is recorded in pydicom's `src/pydicom/data/test_files/README.txt`.
- GDCM (Grassroots DICOM): <https://github.com/malaterre/GDCM>.
- dcmtk: <https://github.com/DCMTK/dcmtk>.

## License attribution

- `LICENSE-pydicom.txt` — MIT license covering the pydicom-origin fixtures: `liver.dcm`, `MR2_UNCI.dcm`,
  `SC_rgb_expb.dcm`, `liver_rle.dcm`, `liver_j2k.dcm`, `explicit_VR-UN.dcm`.
- `LICENSE-gdcm.txt` — BSD-3-Clause license covering the GDCM-origin fixture `MR-SIEMENS-DICOM-WithOverlays.dcm`, and
  acknowledged for the GDCM tooling (`gdcmconv`) used to derive `liver_rle.dcm`.
- `LICENSE-dcmtk.txt` — OFFIS BSD-3-Clause license acknowledged for the dcmtk tooling (`dcmconv`) used to derive
  `SC_rgb_expb.dcm`.
