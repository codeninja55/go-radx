"""pydicom side: Part 10 full-file decode plus per-transfer-syntax pixel decode/encode.

Fixture-for-fixture mirror of the committed go-radx benchmarks: the Part 10 set
matches dicom/codec_bench_test.go part10DecodeFixtures, the decode set matches the
per-codec *_bench_test.go fixture tables, and the RLE encode geometries replicate
syntheticFrameRLE byte-for-byte. Compressed decodes pin decoding_plugin="pylibjpeg"
so the measured Python path is the pylibjpeg C codecs, the comparison the PRD names.
"""

from __future__ import annotations

from pathlib import Path

import numpy as np
import pydicom
from pydicom.pixels import pixel_array

from .common import (
    SIDE_PY,
    STATUS_ERROR,
    STATUS_UNSUPPORTED,
    BenchConfig,
    Result,
    measure,
)

PART10_FIXTURES = [
    "liver.dcm",
    "MR2_UNCI.dcm",
    "SC_rgb_expb.dcm",
    "MR-SIEMENS-DICOM-WithOverlays.dcm",
    "basic-text-sr.dcm",
]

# (comparison name, fixture, decoding plugin); names line up with the go test bench
# mapping in gorunner.py so the renderer can pair rows.
DECODE_FIXTURES = [
    ("pixel-decode-rle", "liver_rle.dcm", "pylibjpeg"),
    ("pixel-decode-jpegls", "MR_small_jpeg_ls_lossless.dcm", "pylibjpeg"),
    ("pixel-decode-jpegls", "JPEGLSNearLossless_08.dcm", "pylibjpeg"),
    ("pixel-decode-jpegls", "JPEGLSNearLossless_16.dcm", "pylibjpeg"),
    ("pixel-decode-jpegls", "SC_rgb_jls_lossy_sample.dcm", "pylibjpeg"),
    ("pixel-decode-jpeg", "SC_jpeg_no_color_transform.dcm", "pylibjpeg"),
    ("pixel-decode-jpeg", "JPGExtended.dcm", "pylibjpeg"),
    ("pixel-decode-jpeg", "JPGLosslessP14SV1_1s_1f_8b.dcm", "pylibjpeg"),
    ("pixel-decode-j2k", "liver_j2k.dcm", "pylibjpeg"),
    ("pixel-decode-htj2k", "HTJ2KLossless_08_RGB.dcm", "pylibjpeg"),
    ("pixel-decode-htj2k", "HTJ2K_08_RGB.dcm", "pylibjpeg"),
]

# Byte-for-byte replica of syntheticFrameRLE in dicom/codec_bench_test.go: a frame of
# (i // 7) % 251 bytes, mixing runs and literals so both PackBits paths are exercised.
RLE_ENCODE_GEOMETRIES = [
    ("8-bit_mono_256x256", 256, 256, 1, 8, "MONOCHROME2"),
    ("16-bit_mono_256x256", 256, 256, 1, 16, "MONOCHROME2"),
    ("8-bit_rgb_256x256", 256, 256, 3, 8, "RGB"),
]


def _synthetic_frame(
    rows: int, cols: int, spp: int, bits: int
) -> tuple[np.ndarray, int]:
    n = rows * cols * spp * (bits // 8)
    buf = (np.arange(n, dtype=np.int64) // 7 % 251).astype(np.uint8).tobytes()
    if bits == 16:
        arr = np.frombuffer(buf, dtype="<u2").reshape(rows, cols)
    elif spp == 3:
        arr = np.frombuffer(buf, dtype=np.uint8).reshape(rows, cols, 3)
    else:
        arr = np.frombuffer(buf, dtype=np.uint8).reshape(rows, cols)
    return arr, n


def _result(name: str, fixture: str, **kw: object) -> Result:
    return Result(
        area="dicom", name=name, fixture=fixture, side=SIDE_PY, library="pydicom+pylibjpeg", **kw
    )


def bench_part10_read(testdata: Path, cfg: BenchConfig) -> list[Result]:
    out: list[Result] = []
    for fixture in PART10_FIXTURES:
        path = testdata / "dicom" / fixture
        size = path.stat().st_size
        samples = measure(
            lambda p=path: pydicom.dcmread(p), cfg.dicom_iters, cfg.repeats, cfg.warmup
        )
        out.append(
            _result(
                "dicom-part10-read",
                fixture,
                ops=cfg.dicom_iters,
                bytes_per_op=float(size),
                samples_s=samples,
                note="dcmread full dataset from disk",
            )
        )
    return out


def bench_pixel_decode(testdata: Path, cfg: BenchConfig) -> list[Result]:
    out: list[Result] = []
    for name, fixture, plugin in DECODE_FIXTURES:
        path = testdata / "dicom" / fixture
        ds = pydicom.dcmread(path)
        try:
            samples = measure(
                lambda d=ds, p=plugin: pixel_array(d, decoding_plugin=p),
                cfg.dicom_iters,
                cfg.repeats,
                cfg.warmup,
            )
        except Exception as exc:  # plugin/codec gaps surface as honest rows, not crashes
            out.append(_result(name, fixture, status=STATUS_ERROR, note=str(exc)[:200]))
            continue
        out.append(
            _result(
                name,
                fixture,
                ops=cfg.dicom_iters,
                samples_s=samples,
                note=f"pydicom.pixels.pixel_array, decoding_plugin={plugin}",
            )
        )
    return out


def bench_pixel_encode(testdata: Path, cfg: BenchConfig) -> list[Result]:
    from pydicom.pixels.encoders import (
        JPEGLSLosslessEncoder,
        RLELosslessEncoder,
    )

    out: list[Result] = []

    # RLE encode over the synthetic geometries (mirrors BenchmarkRLECodecEncode).
    for geom, rows, cols, spp, bits, photometric in RLE_ENCODE_GEOMETRIES:
        arr, nbytes = _synthetic_frame(rows, cols, spp, bits)
        kwargs = {
            "rows": rows,
            "columns": cols,
            "samples_per_pixel": spp,
            "bits_allocated": bits,
            "bits_stored": bits,
            "pixel_representation": 0,
            "photometric_interpretation": photometric,
            "number_of_frames": 1,
        }
        if spp > 1:
            kwargs["planar_configuration"] = 0
        try:
            samples = measure(
                lambda a=arr, k=kwargs: RLELosslessEncoder.encode(
                    a, encoding_plugin="pylibjpeg", **k
                ),
                cfg.dicom_iters,
                cfg.repeats,
                cfg.warmup,
            )
        except Exception as exc:
            out.append(_result("pixel-encode-rle", geom, status=STATUS_ERROR, note=str(exc)[:200]))
            continue
        out.append(
            _result(
                "pixel-encode-rle",
                geom,
                ops=cfg.dicom_iters,
                bytes_per_op=float(nbytes),
                samples_s=samples,
                note="RLELosslessEncoder, encoding_plugin=pylibjpeg (pylibjpeg-rle)",
            )
        )

    # Lossless re-encode of decoded fixture frames (mirrors the CharLS / OpenJPEG
    # encode benchmarks, which encode the decoded native frames of the same fixtures).
    for name, fixture, encoder, plugin in [
        ("pixel-encode-jpegls", "MR_small_jpeg_ls_lossless.dcm", JPEGLSLosslessEncoder, "pyjpegls"),
        ("pixel-encode-j2k", "liver_j2k.dcm", None, "pylibjpeg"),
    ]:
        if encoder is None:
            try:
                from pydicom.pixels.encoders import JPEG2000LosslessEncoder as encoder
            except ImportError:
                out.append(
                    _result(
                        name,
                        fixture,
                        status=STATUS_UNSUPPORTED,
                        note="pydicom build has no JPEG2000Lossless encoder",
                    )
                )
                continue
        path = testdata / "dicom" / fixture
        ds = pydicom.dcmread(path)
        if int(ds.BitsAllocated) < 8:
            # liver_j2k.dcm is 1-bit segmentation data; pydicom's encoders enforce the
            # PS3.5 section 8.2 encoding profiles (Bits Allocated 8/16/...) and refuse
            # it, while the committed go-radx benchmark encodes the same decoded frames.
            out.append(
                _result(
                    name,
                    fixture,
                    status=STATUS_UNSUPPORTED,
                    note=(
                        "pydicom encoder rejects 1-bit pixel data (PS3.5 8.2 profile); "
                        "go-radx row stands alone"
                    ),
                )
            )
            continue
        arr = pixel_array(ds, decoding_plugin="pylibjpeg")
        kwargs = {
            "rows": int(ds.Rows),
            "columns": int(ds.Columns),
            "samples_per_pixel": int(ds.SamplesPerPixel),
            "bits_allocated": int(ds.BitsAllocated),
            "bits_stored": int(ds.BitsStored),
            "pixel_representation": int(ds.PixelRepresentation),
            "photometric_interpretation": str(ds.PhotometricInterpretation),
            "number_of_frames": 1,
        }
        frames = (
            [arr[i] for i in range(arr.shape[0])]
            if int(getattr(ds, "NumberOfFrames", 1) or 1) > 1
            else [arr]
        )
        nbytes = sum(int(f.nbytes) for f in frames)

        def encode_all(
            fs: list[np.ndarray] = frames,
            enc: object = encoder,
            plg: str = plugin,
            k: dict[str, object] = kwargs,
        ) -> None:
            for f in fs:
                enc.encode(f, encoding_plugin=plg, **k)

        try:
            samples = measure(encode_all, cfg.dicom_iters, cfg.repeats, cfg.warmup)
        except Exception as exc:
            out.append(_result(name, fixture, status=STATUS_ERROR, note=str(exc)[:200]))
            continue
        out.append(
            _result(
                name,
                fixture,
                ops=cfg.dicom_iters,
                bytes_per_op=float(nbytes),
                samples_s=samples,
                note=f"lossless re-encode of decoded frames, encoding_plugin={plugin}",
            )
        )

    # Honest gaps, recorded as rows so the table shows WHY a comparison is absent.
    out.append(
        _result(
            "pixel-encode-jpeg",
            "n/a",
            status=STATUS_UNSUPPORTED,
            note="pydicom has no JPEG Baseline/Extended/Lossless encoder (decode only)",
        )
    )
    out.append(
        _result(
            "pixel-encode-htj2k",
            "n/a",
            status=STATUS_UNSUPPORTED,
            note="pylibjpeg-openjpeg (OpenJPEG) decodes HTJ2K but does not encode it",
        )
    )
    return out


def run(testdata: Path, cfg: BenchConfig) -> list[Result]:
    return (
        bench_part10_read(testdata, cfg)
        + bench_pixel_decode(testdata, cfg)
        + bench_pixel_encode(testdata, cfg)
    )
