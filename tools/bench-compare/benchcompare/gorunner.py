"""go-radx side: drive the committed ``go test -bench`` suites and the gobench module.

The committed benchmarks are REUSED, not duplicated: the Part 10 and per-codec
pixel benchmarks under ./dicom/ already walk the same vendored fixtures the
Python runners read, so this module shells out to ``go test -bench`` (default
build, then the cgo codec build when the native libraries link) and maps each
benchmark line onto the shared comparison keys. Areas without a fixture-matched
committed benchmark (DIMSE loopback C-STORE, HL7 v2 parse, FHIR Bundle over the
file fixture) come from the gobench module.
"""

from __future__ import annotations

import json
import os
import re
import statistics
import subprocess
from pathlib import Path

from .common import (
    SIDE_GO,
    STATUS_UNAVAILABLE,
    STATUS_UNSUPPORTED,
    BenchConfig,
    Result,
)

CODEC_TAGS = "dicom_openjpeg dicom_libjpeg dicom_charls"

# go test benchmark name -> (comparison name, fixture). HTJ2K subbenchmarks are
# split from the J2K ones by fixture name below.
_BENCH_LINE = re.compile(
    r"^(Benchmark\S+?)(?:-\d+)?\s+\d+\s+([\d.]+) ns/op"
    r"(?:\s+([\d.]+) MB/s)?(?:\s+([\d.]+) B/op)?(?:\s+([\d.]+) allocs/op)?"
)


def _map_bench(bench: str) -> tuple[str, str] | None:
    name, _, sub = bench.partition("/")
    match name:
        case "BenchmarkReadFile":
            return ("dicom-part10-read", sub)
        case "BenchmarkRLECodecDecode":
            return ("pixel-decode-rle", "liver_rle.dcm")
        case "BenchmarkRLECodecEncode":
            return ("pixel-encode-rle", sub)
        case "BenchmarkCharLSCodecDecode":
            return ("pixel-decode-jpegls", sub)
        case "BenchmarkCharLSCodecEncode":
            return ("pixel-encode-jpegls", "MR_small_jpeg_ls_lossless.dcm")
        case "BenchmarkLibJPEGCodecDecode":
            return ("pixel-decode-jpeg", sub)
        case "BenchmarkOpenJPEGCodecDecode":
            kind = "pixel-decode-htj2k" if sub.startswith("HTJ2K") else "pixel-decode-j2k"
            return (kind, sub)
        case "BenchmarkOpenJPEGCodecEncode":
            return ("pixel-encode-j2k", "liver_j2k.dcm")
    return None


def _go_env(repo_root: Path) -> dict[str, str]:
    env = dict(os.environ)
    env["GOWORK"] = "off"
    return env


def go_version(repo_root: Path) -> str:
    try:
        out = subprocess.run(
            ["go", "version"], capture_output=True, text=True, check=True, cwd=repo_root
        ).stdout.strip()
        return out
    except (OSError, subprocess.CalledProcessError):
        return "unknown"


def run_go_test_benches(
    repo_root: Path, cfg: BenchConfig, *, tags: str | None
) -> tuple[list[Result], str | None]:
    """Run the dicom benchmarks and parse them into Results.

    Returns (results, error). With ``tags`` set this is the cgo codec build; a
    build/link failure (native libraries absent) is reported as the error string
    so the caller can mark those rows unavailable instead of crashing.
    """
    bench_re = (
        "BenchmarkCharLSCodec|BenchmarkLibJPEGCodec|BenchmarkOpenJPEGCodec"
        if tags
        else "BenchmarkReadFile|BenchmarkRLECodec"
    )
    cmd = ["go", "test", "-run=^$", f"-bench={bench_re}"]
    if tags:
        cmd += ["-tags", tags]
    cmd += [f"-benchtime={cfg.go_benchtime}", f"-count={cfg.go_count}", "./dicom/"]
    proc = subprocess.run(
        cmd, capture_output=True, text=True, cwd=repo_root, env=_go_env(repo_root)
    )
    if proc.returncode != 0:
        return [], (proc.stderr or proc.stdout)[-2000:]

    # Each -count repetition emits one line per benchmark; fold repetitions of the
    # same benchmark into one Result whose samples are the per-repetition ns/op
    # (ops=1: one sample = one operation's median time).
    series: dict[str, dict[str, object]] = {}
    for line in proc.stdout.splitlines():
        m = _BENCH_LINE.match(line.strip())
        if not m:
            continue
        bench, ns_op, mb_s, _b_op, allocs = m.groups()
        mapped = _map_bench(bench)
        if mapped is None:
            continue
        name, fixture = mapped
        entry = series.setdefault(
            bench,
            {"name": name, "fixture": fixture, "samples": [], "mb_s": mb_s, "allocs": allocs},
        )
        entry["samples"].append(float(ns_op) * 1e-9)

    results: list[Result] = []
    for entry in series.values():
        samples: list[float] = entry["samples"]  # type: ignore[assignment]
        median_s = statistics.median(samples)
        bytes_per_op = None
        if entry["mb_s"] is not None:
            bytes_per_op = float(entry["mb_s"]) * 1e6 * median_s  # invert go's MB/s
        results.append(
            Result(
                area="dicom",
                name=entry["name"],  # type: ignore[arg-type]
                fixture=entry["fixture"],  # type: ignore[arg-type]
                side=SIDE_GO,
                library="go-radx dicom" + (" (cgo codecs)" if tags else ""),
                ops=1,
                bytes_per_op=bytes_per_op,
                samples_s=samples,
                allocs_per_op=float(entry["allocs"]) if entry["allocs"] else None,
                note="go test -bench (committed benchmark)",
            )
        )
    return results, None


def codec_unavailable_rows(error: str) -> list[Result]:
    """Rows for the cgo codec benches when the native libraries do not link here."""
    rows = []
    for name, fixture in [
        ("pixel-decode-jpegls", "(codec build)"),
        ("pixel-decode-jpeg", "(codec build)"),
        ("pixel-decode-j2k", "(codec build)"),
        ("pixel-decode-htj2k", "(codec build)"),
        ("pixel-encode-jpegls", "(codec build)"),
        ("pixel-encode-j2k", "(codec build)"),
    ]:
        rows.append(
            Result(
                area="dicom",
                name=name,
                fixture=fixture,
                side=SIDE_GO,
                library="go-radx dicom (cgo codecs)",
                status=STATUS_UNAVAILABLE,
                note=f"cgo codec build failed on this host: {error[:160]}",
            )
        )
    return rows


def go_unsupported_rows() -> list[Result]:
    """go-radx-side honest gaps, mirrored against the pydicom-side unsupported rows."""
    return [
        Result(
            area="dicom",
            name="pixel-encode-jpeg",
            fixture="n/a",
            side=SIDE_GO,
            library="go-radx dicom (cgo codecs)",
            status=STATUS_UNSUPPORTED,
            note="libjpeg-turbo codecs are decode-only in go-radx",
        ),
        Result(
            area="dicom",
            name="pixel-encode-htj2k",
            fixture="n/a",
            side=SIDE_GO,
            library="go-radx dicom (cgo codecs)",
            status=STATUS_UNSUPPORTED,
            note="OpenJPEG has no HTJ2K encoder; HTJ2K is decode-only in go-radx",
        ),
    ]


def run_gobench(repo_root: Path, area: str, cfg: BenchConfig) -> list[Result]:
    gobench_dir = repo_root / "tools" / "bench-compare" / "gobench"
    cmd = [
        "go",
        "run",
        ".",
        "-area",
        area,
        "-repeats",
        str(cfg.repeats),
        "-warmup",
        str(cfg.warmup),
        "-testdata",
        str(repo_root / "testdata"),
    ]
    if area == "hl7":
        cmd += ["-iters", str(cfg.hl7_iters)]
    elif area == "fhir":
        cmd += ["-iters", str(cfg.fhir_iters)]
    elif area == "dimse":
        cmd += ["-small-count", str(cfg.dimse_small), "-medium-count", str(cfg.dimse_medium)]
    proc = subprocess.run(
        cmd, capture_output=True, text=True, cwd=gobench_dir, env=_go_env(repo_root), check=True
    )
    return [Result(**row) for row in json.loads(proc.stdout)]
