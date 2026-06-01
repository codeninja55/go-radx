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
| `HTJ2KLossless_08_RGB.dcm` | High-Throughput JPEG 2000 (Lossless Only) (`1.2.840.10008.1.2.4.201`) | pydicom-data store; 480x640 RGB 8-bit synthetic | MIT (pydicom) | CGo-gated HTJ2K decode via OpenJPEG 2.5 and `ErrCodecUnavailable` under `nocgo` (Increment 6c) |
| `HTJ2K_08_RGB.dcm` | High-Throughput JPEG 2000 (`1.2.840.10008.1.2.4.203`) | pydicom-data store; 480x640 RGB 8-bit synthetic | MIT (pydicom) | CGo-gated HTJ2K decode via OpenJPEG 2.5 and `ErrCodecUnavailable` under `nocgo` (Increment 6c) |
| `SC_jpeg_no_color_transform.dcm` | JPEG Baseline (Process 1) (`1.2.840.10008.1.2.4.50`) | pydicom `SC_*` family; 256x256 RGB 8-bit, no colour transform | MIT (pydicom) | CGo-gated JPEG Baseline decode via libjpeg-turbo and `ErrCodecUnavailable` under `nocgo` (Increment 6c) |
| `SC_rgb_small_odd_jpeg.dcm` | JPEG Baseline (Process 1) (`1.2.840.10008.1.2.4.50`) | pydicom `SC_*` family; 3x3 YBR_FULL 8-bit (JFIF/YCbCr) | MIT (pydicom) | YBR_FULL JPEG decode keeps native YCbCr samples; pixel-exact vs pydicom reference (Increment 6c) |
| `JPGExtended.dcm` | JPEG Extended (Process 2 & 4) (`1.2.840.10008.1.2.4.51`) | pydicom test files; 1024x256 MONOCHROME2 16-bit allocated / 12-bit stored | MIT (pydicom) | CGo-gated JPEG Extended 12-bit decode via libjpeg-turbo (Increment 6c) |
| `MR-SIEMENS-DICOM-WithOverlays.dcm` | Explicit VR Little Endian (`1.2.840.10008.1.2.1`) | GDCM | BSD-3-Clause (GDCM) | Overlays and repeating group `60xx` dictionary mask (Increment 1, DCM-012) |
| `explicit_VR-UN.dcm` | JPEG 2000 Lossless (`1.2.840.10008.1.2.4.90`) | pydicom (issue #968); original from The Cancer Imaging Archive, de-identified, recompressed with `gdcmconv --j2k` | MIT (pydicom) | Ambiguous/`UN` VR resolution (nearly all tags carry VR `UN`) (Increment 1) |
| `basic-text-sr.dcm` | Explicit VR Little Endian (`1.2.840.10008.1.2.1`) | pydicom `reportsi.dcm` (Basic Text SR Storage; synthetic placeholder patient identifiers) | MIT (pydicom) | Structured Report content-item tree parse and round-trip (Increment 8) |

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
  `SC_rgb_expb.dcm`, `liver_rle.dcm`, `liver_j2k.dcm`, `explicit_VR-UN.dcm`, `basic-text-sr.dcm`,
  `HTJ2KLossless_08_RGB.dcm`, `HTJ2K_08_RGB.dcm`, `SC_jpeg_no_color_transform.dcm`, `SC_rgb_small_odd_jpeg.dcm`,
  `JPGExtended.dcm`.
- `LICENSE-gdcm.txt` — BSD-3-Clause license covering the GDCM-origin fixture `MR-SIEMENS-DICOM-WithOverlays.dcm`, and
  acknowledged for the GDCM tooling (`gdcmconv`) used to derive `liver_rle.dcm`.
- `LICENSE-dcmtk.txt` — OFFIS BSD-3-Clause license acknowledged for the dcmtk tooling (`dcmconv`) used to derive
  `SC_rgb_expb.dcm`.
