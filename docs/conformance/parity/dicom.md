# DICOM data-layer parity: go-radx vs pydicom

This matrix compares the go-radx DICOM data layer (the `dicom/` package) against pydicom's documented public
feature surface, including the pylibjpeg plugin surface for pixel codecs. Every status was verified against
go-radx source and tests, not inferred from documentation; evidence cells name the file and symbol or line.
A feature is MET only when it is both implemented and exercised by tests. Network services (DIMSE), DICOMweb,
and the CLI/server are covered by sibling audits; rows here touch them only where the data layer is the owner.

Statuses: MET (implemented and tested), PARTIAL (a usable subset exists), NOT-MET (absent),
N-A (Python-specific with no sensible Go equivalent). Size estimates for PARTIAL/NOT-MET rows:
S (under 1 day), M (1-3 days), L (over 3 days).

## Summary

Row counts across all tables:

| Status | Count |
|--------|-------|
| MET | 66 |
| PARTIAL | 9 |
| NOT-MET | 7 |
| N-A | 6 |
| Total | 88 |

Top NOT-MET/PARTIAL items by impact:

1. **Private-creator dictionary breadth (PARTIAL, M).** The private block API and dictionary lookup mechanism are
   now MET (`dicom/private_block.go`, `dicom/private_dict.go`); the dictionary seed is minimal and attributed
   (the pydicom illustrative "ACME 3.1" creator). Vendor catalogues (Siemens/GE/Philips) are deferred to a
   `gdcmPrivateDict.xml` generator because no such source is vendored here to attribute against.

The charset table now covers the Latin/Cyrillic/Arabic/Greek/Hebrew 8859 sets, the Japanese ISO 2022 family,
UTF-8, GB18030, GBK, plus Korean (ISO 2022 IR 149), Simplified Chinese (ISO 2022 IR 58), Thai (ISO_IR 166 /
ISO 2022 IR 166), and bare ISO_IR 13 half-width katakana (PS3.3 C.12.1.1.2, PS3.5 Annex I.2/K.2).
Modality/VOI LUT and windowing (`apply_modality_lut`/`apply_voi_lut`/`apply_windowing`, `dicom/lut.go`), palette
colour expansion (`apply_color_lut`), colour-space conversion (`convert_color_space`, `dicom/colorspace.go`), and
overlay/waveform extraction (`overlay_array`/`waveform_array`, `dicom/overlay.go`, `dicom/waveform.go`) are now MET
(PS3.3 §C.11, C.7.9, C.7.6.3.1.2, C.10.9, PS3.5 §8.1.2).

Where go-radx exceeds pydicom: a full PS3.15 Table E.1-1 de-identification engine (pydicom core ships only
`remove_private_tags` and guidance), typed fail-closed errors, bounded hostile-input reading with fuzz targets
(`FuzzRead`, `FuzzReadPixelDataFrom` in `dicom/fuzz_test.go`), a deflate-bomb guard, and committed benchmark
baselines.

## File reading and writing

| Feature | pydicom anchor | Status | go-radx evidence | Size | Notes |
|---------|---------------|--------|------------------|------|-------|
| Read Part 10 from path or stream (uncompressed TS) | `dcmread` | MET | dicom/reader_writer.go:Read, ReadFile; dicom/file_test.go | - | Honours declared TS; never assumes Implicit VR LE |
| Read compressed Part 10 with dataset retained | `dcmread` (any TS) | MET | dicom/dataset_codec.go:readDataSet (encapsulated branch); encapsulated_io_test.go TestReadEncapsulatedFixtureRetainsDataSet | - | The (7FE0,0010) fragment stream is retained verbatim, undecoded; unrecognised/private syntaxes stay rejected fail-closed |
| Write Part 10 (uncompressed TS) | `dcmwrite` / `Dataset.save_as` | MET | dicom/reader_writer.go:Write, WriteFile; dicom/dataset_writefile.go:9 | - | Encode-to-buffer first so errors surface before bytes hit the writer |
| Write compressed Part 10 (encapsulated pixel data) | `dcmwrite` with compressed TS | MET | dicom/dataset_codec.go:writeEncapsulatedElement; encapsulated_io_test.go TestEncapsulatedFixtureMainDataSetByteIdentical | - | Undefined-length element with Basic Offset Table (PS3.5 A.4); byte-identical round-trip on the compressed fixtures |
| Deflated Explicit VR LE read/write | Deflated TS support | MET | dicom/reader_writer.go:44-52,107-116; dicom/deflate_bomb_test.go | - | Inflate cap defends against decompression bombs (`WithMaxInflatedBytes`) |
| Stop before pixel data | `stop_before_pixels` | MET | dicom/file.go:98 WithStopAtPixelData; dicom/dataset_codec_test.go:114 | - | |
| Deferred loading of large elements | `defer_size` | MET | dicom/file.go WithDeferredValues; dicom/deferred.go DeferredValue.Load; deferred_test.go TestReadFileDefersLargeValues, TestDeferredRoundTripByteIdentical; deferred_hostile_test.go | - | ReadFile-only (the path is re-opened on demand); Read/DecodeDataSet and deflated TS reject the option fail-closed; loads re-validate the recorded window (typed *DeferredLoadError) |
| Selective-tag read | `specific_tags` | NOT-MET | no equivalent option in dicom/file.go readConfig | S | |
| Read without preamble / missing file meta | `force=True` | PARTIAL | dicom/dataset_stream.go:43 DecodeDataSet(r, ts) | S | Bare datasets readable when the caller supplies the TS; no preamble-less sniffing |
| Bare dataset stream encode/decode | `read_dataset` internals | MET | dicom/dataset_stream.go:17 EncodeDataSet, :43 DecodeDataSet; dataset_stream_test.go | - | |
| File meta generation and repair | `FileMetaDataset` / `fix_meta_info` | MET | dicom/file_meta.go:46 readFileMeta; dicom/file_meta_write.go:64 buildFileMetaGroup | - | Auto-supplies (0002,0001), recomputes group length, validates UIDs |

go-radx extras with no pydicom equivalent: bounded hostile-input reading (`WithMaxElementLen`,
`WithMaxSequenceDepth`, `WithMaxInflatedBytes` in dicom/file.go) and the truncation-is-failure contract
(dicom/element_header.go:86 midElementEOF).

## Dataset and element API

| Feature | pydicom anchor | Status | go-radx evidence | Size | Notes |
|---------|---------------|--------|------------------|------|-------|
| Element get/set/delete by tag | `Dataset.__getitem__` etc. | MET | dicom/dataset.go:34 Get, :40 Set, :49 Delete; dataset_test.go | - | |
| Keyword-to-tag lookup | keyword attribute access | MET | dicom/dictionary.go:26 Lookup, :51 LookupKeyword; dictionary_test.go | - | Go-idiomatic: lookup then typed getter, not dynamic attributes |
| Typed value accessors | `DataElement.value` typing | MET | dicom/dataset.go:96-196 GetString/GetInt/GetDecimal/GetUID/GetSequence/GetPersonName | - | |
| Top-level iteration in tag order | `iter(Dataset)` | MET | dicom/dataset.go:63 All | - | |
| Recursive iteration into sequences | `Dataset.iterall` | NOT-MET | only the internal walker dicom/deidentify_walk.go:52 | S | Export a recursive walk |
| Sequences (defined/undefined length, nesting, depth cap) | `Sequence` | MET | dicom/sequence.go, sequence_codec.go; sequence_test.go, sequence_regression_test.go | - | Original defined/undefined-length form preserved on write |
| Private element parse and round-trip | private elements | MET | dicom/tag.go:26 IsPrivate, :29 IsPrivateCreator; generic element path; tag_test.go | - | Parsed and written generically, no semantic claim |
| Private block API | `private_block`, `add_new_private`, `get_private_item` | MET | dicom/private_block.go:DataSet.PrivateBlock, DataSet.PrivateCreators, DataSet.GetPrivateItem, PrivateBlock.{Get,Set,Tag,Lookup}; private_block_test.go | - | Block = low byte of creator element (PS3.5 §7.8.1); create=true reserves the lowest free block; resolves real blocks in parsed datasets incl. UN-encoded creators |
| Private-creator dictionaries | pydicom private dicts | PARTIAL | dicom/private_dict.go:PrivateTagInfo, LookupPrivate, PrivateBlock.Lookup; private_block_test.go:TestPrivateDictLookup | M | Lookup mechanism complete; seed minimal and attributed (pydicom "ACME 3.1" illustrative creator). Vendor catalogues deferred to a gdcmPrivateDict.xml generator (TODO in private_dict.go); no such source vendored, so vendor tag meanings are not invented |
| Remove all private tags standalone | `Dataset.remove_private_tags` | PARTIAL | dicom/deidentify_walk.go:194 (inside Deidentify only) | S | Available only via the de-identification profile |
| DICOM JSON model (PS3.18) to/from | `Dataset.to_json` / `from_json` | MET | dicomweb/json.go:87 MarshalJSON, :516 UnmarshalJSON; dicomweb/json_test.go | - | Includes BulkDataURI threshold/resolver options (bulk data handler equivalent) |
| Dataset deep copy | `copy.deepcopy` | MET | dicom/dataset.go:197 Clone | - | |
| Tag-range slicing | `ds[0x00100000:0x00110000]` | N-A | iterate All() and filter on Tag.Group() | - | Pythonic slice sugar |
| Dynamic attribute access / `dir()` | `ds.PatientName` | N-A | typed getters by Tag constant (dicom/tag_values.go) | - | Go has no dynamic attributes; tag constants are generated for the full dictionary |

## Value representations and value types

| Feature | pydicom anchor | Status | go-radx evidence | Size | Notes |
|---------|---------------|--------|------------------|------|-------|
| All PS3.5 VRs including OV/SV/UV | VR support | MET | dicom/vr.go:7-40; vr_test.go | - | 34 VRs incl. Other Very Long, Signed/Unsigned Very Long |
| PersonName three-group, five-component model | `valuerep.PersonName` | MET | dicom/person_name.go:12-58 ParsePersonName; person_name_test.go | - | Alphabetic/ideographic/phonetic groups modelled |
| DA/TM/DT typed parsing incl. timezone offsets | `valuerep.DA/TM/DT` | MET | dicom/date.go:38 ParseDA, time.go:50 ParseTM, datetime_dt.go:50 ParseDT; tests per file | - | Strict by default; `WithLenientDates` opts into partial DA forms |
| Decimal String handling | `valuerep.DS`, `use_DS_decimal` | MET | dicom/decimal.go:25 ParseDecimal, :39 ParseDecimalLexical; decimal_test.go | - | Always lossless; no float/decimal mode switch needed |
| Integer String values | `valuerep.IS` | MET | dicom/dataset.go:127 GetInt; value.go:85 NewInts | - | |
| Attribute Tag (AT) values | AT VR | MET | dicom/value.go:193 NewTags; value_test.go | - | |
| Bulk byte values (OB/OW/OD/OF/OL/OV) | bytes values | MET | dicom/value.go:217 NewBytes; value_codec_test.go | - | |

## Pixel data: decode, encode, and transforms

Decode/encode parity is measured against pydicom 3.x `pydicom.pixels` plus the pylibjpeg plugin family
(pylibjpeg-libjpeg, pylibjpeg-openjpeg, pylibjpeg-rle). go-radx codecs are build-tag-gated CGo
(`dicom_libjpeg`, `dicom_openjpeg`, `dicom_charls`) except RLE and native, which are pure Go; without the
tag, decode returns the typed `dicom.ErrCodecUnavailable` rather than failing the build.

| Feature | pydicom anchor | Status | go-radx evidence | Size | Notes |
|---------|---------------|--------|------------------|------|-------|
| Native (uncompressed) frame decode and iteration | `pixel_array` / `iter_pixels` | MET | dicom/pixel_data.go:110 Frames; pixel_data_native_test.go | - | `iter.Seq2[Frame, error]` streaming iterator |
| Encapsulated frame extraction with Basic Offset Table | `encaps.generate_frames`, `parse_basic_offsets` | MET | dicom/encapsulated.go:42 parseEncapsulated, :119 basicOffsetTable; encapsulated_test.go | - | Frame-to-fragment mapping validated, hostile lengths bounded |
| Extended Offset Table (read) | `encapsulate_extended` read side | MET | dicom/extended_offset.go:22,59; extended_offset_test.go | - | |
| Public encapsulation builders (BOT/EOT write) | `encaps.encapsulate`, `encapsulate_extended` | NOT-MET | internal dicom/encapsulated.go encodeStream (used by File.SetPixelData) | S | Compressed write ships via retained streams and SetPixelData; no public byte-level builder API |
| Single-frame random access by index | `encaps.get_frame`, `pixel_array(index=)` | PARTIAL | dicom/pixel_data.go:110 (iterate to index) | S | Iterator carries Frame.Index; no direct indexed accessor |
| Typed sample access with metadata | `pixel_array` ndarray + `as_array` meta | PARTIAL | dicom/pixel_data.go:8 Frame.Pixels ([]byte) + PixelGeometry | S | Raw bytes plus geometry; no typed []uint16/[]int16 view helpers |
| JPEG Baseline / Extended decode (.50/.51) | pylibjpeg-libjpeg decode | MET | dicom/codec_libjpeg.go; codec_libjpeg_test.go | - | `dicom_libjpeg` tag (libjpeg-turbo) |
| JPEG Lossless P14 / SV1 decode (.57/.70) | pylibjpeg-libjpeg decode | MET | dicom/codec_libjpeg.go; codec_libjpeg_lossless_test.go | - | Predictor-agnostic |
| JPEG-LS lossless / near-lossless decode (.80/.81) | pylibjpeg / pyjpegls decode | MET | dicom/codec_charls.go; codec_charls_test.go | - | `dicom_charls` tag (CharLS) |
| JPEG 2000 lossless / lossy decode (.90/.91) | pylibjpeg-openjpeg decode | MET | dicom/codec_openjpeg.go; codec_openjpeg_test.go | - | `dicom_openjpeg` tag |
| HTJ2K decode (.201/.202/.203) | pylibjpeg-openjpeg decode | MET | dicom/codec_htj2k_test.go | - | Via OpenJPEG |
| RLE Lossless decode (.5) | pylibjpeg-rle / built-in | MET | dicom/rle.go, codec_rle.go; rle_test.go | - | Pure Go, always available |
| RLE Lossless encode | built-in RLE encoder | MET | dicom/codec_rle.go:9 CanEncode; rle_roundtrip_test.go | - | Pure Go |
| JPEG 2000 lossless encode | pylibjpeg-openjpeg encode | MET | dicom/codec_openjpeg.go:59 CanEncode; codec_bench_encode_test.go | - | Lossless only, pixel-exact |
| JPEG-LS lossless encode | pyjpegls encode | MET | dicom/codec_charls.go:69 CanEncode; codec_charls_test.go | - | Lossless only |
| Lossy encode (J2K ratios, JLS near-lossless, JPEG baseline) | `compress` with lossy params | NOT-MET | dicom/codec_libjpeg.go:67 CanEncode false; typed ErrEncodeUnsupported (codec.go:56) | M | Deliberate fidelity policy; revisit only with explicit opt-in design |
| Dataset-level compress/decompress in place | `Dataset.compress` / `decompress` | MET | dicom/transcode.go File.SetPixelData; encapsulated_io_test.go TestSetPixelDataTranscodeRoundTrip; store.go prepareForStore | - | NewPixelData -> Transcode -> File.SetPixelData -> Write; radx store --transcode-to decompresses on send |
| Pixel-layer transcode between syntaxes | decompress-then-compress flow | MET | dicom/transcode.go:12 Transcode; transcode_test.go | - | Explicit, opt-in; lossless targets only |
| Modality LUT / rescale application | `apply_modality_lut` | MET | dicom/lut.go:ApplyModalityLUT; lut_test.go TestApplyModalityLUTRescale, TestApplyModalityLUTTablePrecedence | - | ModalityLUTSequence table (precedence) or RescaleSlope/Intercept linear rescale (PS3.3 §C.11.1.1.2) |
| VOI LUT and windowing | `apply_voi_lut`, `apply_windowing` | MET | dicom/lut.go:ApplyVOILUT, ApplyWindowing; lut_test.go TestApplyWindowingLinear, TestApplyWindowingLinearExact, TestApplyWindowingSigmoid, TestApplyVOILUTTablePrecedence | - | VOILUTSequence table or LINEAR/LINEAR_EXACT/SIGMOID windowing (PS3.3 §C.11.2.1.2-3); indexed multi-pair WC/WW |
| Palette colour expansion | `apply_color_lut` | MET | dicom/colorspace.go ApplyColorLUT; colorspace_test.go (8/16-bit entries, first-mapped offset, zero-count=65536) | - | Non-segmented path (PS3.3 C.7.9, C.7.6.3.1.5); segmented LUT out of scope |
| Colour-space conversion utility | `convert_color_space` | MET | dicom/colorspace.go ConvertColorSpace; colorspace_test.go (PS3.3 C.7.6.3.1.2 pixel-exact) | - | YBR_FULL<->RGB, YBR_FULL_422->RGB/YBR_FULL, planar config 0/1; YBR_PARTIAL/ICT/RCT out of scope |
| 1-bit pixel pack/unpack | `pack_bits` / `unpack_bits` | PARTIAL | dicom/pixel_geometry.go:34 FrameLength (sub-byte sizing) | S | Frame sizing correct; no per-pixel pack/unpack helpers |
| Overlay data extraction | `Dataset.overlay_array` | MET | dicom/overlay.go:78 OverlayArray, OverlayGroups (LSB-first unpack PS3.5 §8.1.2); fixture + synthetic tests in overlay_test.go | - | 484x484 MR-SIEMENS fixture, 323 set bits exact; embedded (bit-position) overlays rejected as retired |
| Waveform decode | `Dataset.waveform_array` | MET | dicom/waveform.go:84 WaveformArray, WaveformGroups (C.10.9 scaling); synthetic exact tests in waveform_test.go | - | 8/16-bit SS/US/SB/UB, big/little-endian, per-channel sensitivity*correction+baseline |
| Streaming frame iteration | `pixels.iter_pixels` | MET | dicom/pixel_data.go:110 Frames | - | |

## Character sets

| Feature | pydicom anchor | Status | go-radx evidence | Size | Notes |
|---------|---------------|--------|------------------|------|-------|
| Default repertoire + ISO 8859 supplements (IR 100/101/109/110/144/127/126/138/148) | `charset` term table | MET | dicom/specific_character_set.go:67-92; charset_integration_test.go | - | Bare and ISO 2022 forms both mapped |
| UTF-8 (ISO_IR 192), GB18030, GBK | multi-byte stand-alone sets | MET | dicom/specific_character_set.go:104-106; charset_regression_test.go | - | |
| ISO 2022 code extensions (Latin + Japanese IR 13/14/87/159), multi-valued (0008,0005), delimiter resets, per-item charset in sequences | code-extension decode | MET | dicom/iso2022.go:41 decodeISO2022, :21 isISO2022Reset; dicom/sequence_codec.go:150; iso2022_test.go | - | |
| Korean (ISO 2022 IR 149) | `euc_kr` mapping | MET | dicom/specific_character_set.go:definedTermTable "ISO 2022 IR 149" (korean.EUCKR, familyDoubleByteG1); dicom/iso2022.go:decodeSingleByteSegment double-byte G1 run; iso2022_test.go:TestISO2022KoreanPersonNameDecode | - | PS3.5 Annex I.2 worked example "Hong^Gildong=洪^吉洞=홍^길동" |
| Simplified Chinese code extension (ISO 2022 IR 58) | `GB2312` mapping | MET | dicom/specific_character_set.go:definedTermTable "ISO 2022 IR 58" (simplifiedchinese.GBK, familyDoubleByteG1); iso2022_test.go:TestISO2022SimplifiedChinesePersonNameDecode | - | PS3.5 Annex K.2 worked example "Zhang^XiaoDong=张^小东"; GB2312 is the 8-bit subset of GBK |
| Thai (ISO_IR 166 / ISO 2022 IR 166) | `TIS-620` mapping | MET | dicom/specific_character_set.go:definedTermTable "ISO_IR 166"/"ISO 2022 IR 166" (charmap.Windows874); iso2022_test.go:TestBareThaiDecode, TestISO2022ThaiDesignationDecode | - | Bare and ESC - T forms both mapped; pydicom test_charset.py vector |
| Bare ISO_IR 13 (JIS X 0201 without escapes) | `shift_jis` mapping | MET | dicom/specific_character_set.go:definedTermTable "ISO_IR 13" (japanese.ShiftJIS); iso2022_test.go:TestBareKatakanaDecode | - | Half-width katakana 0xA1-0xDF -> U+FF61-U+FF9F; pydicom test_charset.py vector |
| Encode with code extensions (write side) | `charset.encode_string` | MET | dicom/iso2022.go:258 encodeISO2022; iso2022_test.go | - | |
| Lenient handling of malformed charset terms | `handle_encoding_errors`, spelling repair | PARTIAL | dicom/specific_character_set.go:157 normaliseDefinedTerm (trim only); typed UnsupportedCharacterSetError | S | Fail-closed by design; no replace/ignore modes, no misspelling repair |

## De-identification

pydicom core ships only `remove_private_tags` and anonymisation guidance; the full profile work lives in
third-party tools. go-radx ships a PS3.15 Annex E Basic Application Level Confidentiality Profile engine,
so this area exceeds the pydicom reference surface.

| Feature | pydicom anchor | Status | go-radx evidence | Size | Notes |
|---------|---------------|--------|------------------|------|-------|
| Basic profile engine (Table E.1-1) | anonymisation guidance | MET | dicom/deidentify.go NewProfile/Deidentify; deidentify_actions.go (213 keywords); deidentify_test.go | - | Count gated by tools/conformance-drift |
| Retain sub-options (CG 7050 coded) | n/a (exceeds pydicom) | MET | dicom/deidentify.go options; deidentify_options_test.go | - | Patient characteristics, temporal, device, UIDs, safe private |
| Burned-in pixel annotation fail-closed | n/a | MET | dicom/deidentify.go:15 ErrBurnedInPixelData; deidentify_burnedin_test.go | - | |
| Consistent UID remapping | n/a | MET | dicom/deidentify_walk.go:152 remapUID; deidentify_uid_test.go | - | Per-call map preserves the reference graph |
| Date shifting with interval preservation | n/a | MET | dicom/deidentify_walk.go:166 applyDate; deidentify_dates_test.go | - | Offset derived per StudyInstanceUID |

## DICOMDIR and file-sets

| Feature | pydicom anchor | Status | go-radx evidence | Size | Notes |
|---------|---------------|--------|------------------|------|-------|
| Load, iterate, and query an existing file-set | `fileset.FileSet`, `find` | MET | dicom/fileset.go OpenFileSet/Roots/Records/Instances/Find/FindValues; fileset_test.go, fileset_hostile_test.go | - | Offset links resolved with cycle/range checks (typed errors, bounded walk); DICOMDIR must be Explicit VR LE (PS3.10 §8.6, typed error otherwise); Referenced File IDs read permissively but traversal-safe — lowercase/over-long components accepted as pydicom does, strict §8.2/§8.5 IDs enforced on write only; cross-read against a dcmtk `dcmmkdir` DICOMDIR |
| Create, write, add, and remove instances | `FileSet.add/remove/write` | PARTIAL | dicom/fileset_write.go FileSetBuilder Add/AddFile/SetID/Write; fileset_test.go round-trip; dcmtk `dcmmkdir --append` re-links the written DICOMDIR | S | Create-from-scratch only: no remove, no staged mutation of an existing file-set; leaf records are IMAGE or SR DOCUMENT (not pydicom's full DIRECTORY_RECORDERS table); PS3.11 media application profiles not enforced |

## UIDs

| Feature | pydicom anchor | Status | go-radx evidence | Size | Notes |
|---------|---------------|--------|------------------|------|-------|
| Generate UIDs under an organisation root | `generate_uid(prefix=)` | MET | dicom/uid_generator.go:31 NewUIDGenerator; uid_generator_test.go | - | No default root shipped: explicit root or the 2.25 form |
| Random UUID-derived UIDs (2.25) | `generate_uid()` default | MET | dicom/uid_generator.go:48 NewRandomUIDGenerator | - | |
| Deterministic entropy-derived UIDs | `generate_uid(entropy_srcs=)` | NOT-MET | no equivalent in uid_generator.go | S | Useful for reproducible de-identification pipelines |
| UID validation | `UID.is_valid` | MET | dicom/uid.go:21 Validate, :53 IsValid; uid_test.go | - | |
| UID registry names | `UID.name` / `keyword` | MET | dicom/uid.go:59 Name; uid_values.go | - | |
| Transfer syntax introspection | `is_implicit_VR`, `is_compressed`, etc. | MET | dicom/transfer_syntax.go:32-67 IsImplicitVR/IsBigEndian/IsDeflated/IsEncapsulated/Name | - | |
| Private transfer syntax registration | `UID.set_private_encoding` | NOT-MET | dicom/reader_writer.go supportedEncapsulated allowlist rejects unrecognised syntaxes | S | `RegisterCodec` (codec.go:90) covers pixel codecs only |

## Configuration and utilities

| Feature | pydicom anchor | Status | go-radx evidence | Size | Notes |
|---------|---------------|--------|------------------|------|-------|
| Read/write validation modes | `settings.reading_validation_mode` (RAISE/WARN/IGNORE) | PARTIAL | strict by default; per-call opt-outs dicom/file.go:106 WithLenientDates, :114 WithDefaultCharacterSet | S | Deliberate fail-closed posture; no warn-and-continue mode for sloppy files |
| Global debug logging | `config.debug` | N-A | structured logging is the caller's concern (zap upstream) | - | |
| Raw-element conversion hooks | `pydicom.hooks` | N-A | typed errors and functional options replace callback hooks | - | |
| Future-behaviour switch | `config._use_future` | N-A | versioned Go module API | - | |
| Inspect/dump a file from the CLI | `pydicom show` | MET | cmd/radx/internal/command/dump.go (radx dump); compressed_io_test.go TestDumpCompressedFile | - | CLI parity owned by audit A6; works on compressed files |
| Generate code from a file | `pydicom codify` | NOT-MET | no equivalent radx command | S | Dev tooling; CLI scope owned by audit A6 |
| Bundled example/test data module | `pydicom.examples`, `get_testdata_file` | N-A | dicom/testdata serves the test suite; not a public API commitment | - | |

## Methodology

Reference inventory built on 2026-06-10 from current documentation, not training data:

- pydicom: Context7 ID `/pydicom/pydicom` (High reputation, 1324 snippets), queried per topic area:
  reading/writing (`dcmread`/`dcmwrite` options), Dataset/DataElement API and private tags, pixel data
  (decode/encode, `apply_voi_lut`/`apply_modality_lut`/`convert_color_space`, `encaps`), character sets,
  UIDs and config (`settings.reading_validation_mode`, hooks), FileSet/DICOMDIR, waveform/overlay, CLI.
  Source pages cited by ctx7 resolve to https://pydicom.github.io/pydicom/stable/ guides and reference docs.
- pylibjpeg plugin surface: Context7 ID `/pydicom/pylibjpeg` (decode: JPEG, JPEG-LS, JPEG 2000/HTJ2K, RLE;
  encode: JPEG 2000 via openjpeg, RLE), cross-checked against the README transfer-syntax table it cites.

go-radx verification: direct reads and symbol greps across `dicom/` (and `dicomweb/json.go`,
`cmd/radx/internal/command/` where a row touches them), with test files confirmed present for every MET row.
Existing conformance statement `docs/conformance/dicom.md` was used as a map but every claim rowed here was
re-verified against source; one tension was found and is recorded below.

Caveats and unverified areas, stated honestly:

- The ctx7 snapshots are extracts, not the complete pydicom reference; minor utility functions (e.g. dataset
  pretty-printing internals, `Dataset.formatted_lines`, benchmark helpers) and every individual `config`
  attribute were not exhaustively rowed. The major documented surfaces (user guide topic areas) are covered.
- Tests were inspected for existence and subject matter, not executed in this audit (no `go test` run).
- Value-multiplicity (VM) enforcement on write was not deeply verified in either direction; the dictionary
  carries VM strings (dicom/tag_values.go TagInfo) but no row claims VM validation parity.
- DIMSE-layer behaviour for compressed instances (transport pass-through) is out of this audit's scope. The
  former doc-accuracy point about `docs/conformance/dicom.md` ("for transport always" misleading readers about
  `dicom.Read`) is resolved: `dicom.Read`/`Write` now handle the recognised encapsulated syntaxes with the
  pixel stream retained, and the statement says so.
- Fuzzing posture confirmed: 22 repo-wide fuzz targets include the data-layer `FuzzRead` and
  `FuzzReadPixelDataFrom` in `dicom/fuzz_test.go`; fuzz coverage is not a gap.
