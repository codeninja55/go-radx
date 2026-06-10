"""go-radx side: drive the committed ``go test -bench`` suites and the gobench module.

The committed benchmarks are REUSED, not duplicated: the Part 10 and per-codec
pixel benchmarks under ./dicom/ already walk the same vendored fixtures the
Python runners read, so this module shells out to ``go test -bench`` (default
build, then the cgo codec build when the native libraries link) and maps each
benchmark line onto the shared comparison keys. The raw per-codec decode rows
are go-internal (``codec.Decode`` over pre-extracted frames) and carry
``codec-decode-*`` names so they never pair with pydicom's ``pixel_array``
rows; the user-facing pixel decode comparison (``pixel-decode-*``) comes from
the gobench module's dicom area, which times the public ``dicom.ReadPixelData``
+ ``Frames()`` path. Areas without a fixture-matched committed benchmark
(DIMSE loopback C-STORE, HL7 v2 parse, FHIR Bundle over the file fixture) also
come from gobench.

``go test -count`` repetitions are treated like every other series: the first
``cfg.warmup`` repetitions are discarded (``-count`` is bumped accordingly) and
the published ns/op and MB/s both derive from the measured repetitions' median.
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
            return ("codec-decode-rle", "liver_rle.dcm")
        case "BenchmarkRLECodecEncode":
            return ("pixel-encode-rle", sub)
        case "BenchmarkCharLSCodecDecode":
            return ("codec-decode-jpegls", sub)
        case "BenchmarkCharLSCodecEncode":
            return ("pixel-encode-jpegls", "MR_small_jpeg_ls_lossless.dcm")
        case "BenchmarkLibJPEGCodecDecode":
            return ("codec-decode-jpeg", sub)
        case "BenchmarkOpenJPEGCodecDecode":
            kind = "codec-decode-htj2k" if sub.startswith("HTJ2K") else "codec-decode-j2k"
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
    # The first cfg.warmup repetitions of each benchmark are discarded below, so bump
    # -count to keep the measured sample count at cfg.go_count (warmup parity with the
    # Python runners and gobench).
    cmd += [f"-benchtime={cfg.go_benchtime}", f"-count={cfg.go_count + cfg.warmup}", "./dicom/"]
    proc = subprocess.run(
        cmd, capture_output=True, text=True, cwd=repo_root, env=_go_env(repo_root)
    )
    if proc.returncode != 0:
        return [], (proc.stderr or proc.stdout)[-2000:]

    # Each -count repetition emits one line per benchmark; fold repetitions of the
    # same benchmark into one Result. Every repetition's ns/op, MB/s, and allocs/op
    # is kept so all published figures derive from the measured (post-warmup)
    # repetitions, never from repetition 1 alone.
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
        entry = series.setdefault(bench, {"name": name, "fixture": fixture, "reps": []})
        entry["reps"].append(
            (
                float(ns_op) * 1e-9,
                float(mb_s) if mb_s is not None else None,
                float(allocs) if allocs is not None else None,
            )
        )

    results: list[Result] = []
    for entry in series.values():
        reps: list[tuple[float, float | None, float | None]] = entry["reps"]  # type: ignore[assignment]
        measured = reps[cfg.warmup :] if len(reps) > cfg.warmup else reps
        samples = [sec for sec, _, _ in measured]
        # bytes/op is constant per benchmark (b.SetBytes); recover it per repetition
        # (MB/s * sec/op) and take the median so throughput matches the ns/op statistic.
        byte_estimates = [mb * 1e6 * sec for sec, mb, _ in measured if mb is not None]
        alloc_values = [a for _, _, a in measured if a is not None]
        name = entry["name"]
        note = "go test -bench (committed benchmark)"
        if str(name).startswith("codec-decode-"):
            note = (
                "go-internal: raw codec.Decode over pre-extracted frames "
                "(committed benchmark); no Python pair"
            )
        results.append(
            Result(
                area="dicom",
                name=name,  # type: ignore[arg-type]
                fixture=entry["fixture"],  # type: ignore[arg-type]
                side=SIDE_GO,
                library="go-radx dicom" + (" (cgo codecs)" if tags else ""),
                ops=1,
                bytes_per_op=statistics.median(byte_estimates) if byte_estimates else None,
                samples_s=samples,
                allocs_per_op=statistics.median(alloc_values) if alloc_values else None,
                note=note,
            )
        )
    return results, None


def codec_unavailable_rows(error: str) -> list[Result]:
    """Rows for the cgo codec benches when the native libraries do not link here."""
    rows = []
    for name, fixture in [
        ("codec-decode-jpegls", "(codec build)"),
        ("codec-decode-jpeg", "(codec build)"),
        ("codec-decode-j2k", "(codec build)"),
        ("codec-decode-htj2k", "(codec build)"),
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


def _gobench_cmd(repo_root: Path, area: str, cfg: BenchConfig, *, tags: str | None) -> list[str]:
    cmd = ["go", "run"]
    if tags:
        cmd += ["-tags", tags]
    cmd += [
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
    elif area == "dicom":
        cmd += ["-iters", str(cfg.dicom_iters)]
    elif area == "dimse":
        cmd += ["-small-count", str(cfg.dimse_small), "-medium-count", str(cfg.dimse_medium)]
    return cmd


def run_gobench(repo_root: Path, area: str, cfg: BenchConfig) -> list[Result]:
    gobench_dir = repo_root / "tools" / "bench-compare" / "gobench"
    def run(cmd: list[str], check: bool) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            cwd=gobench_dir,
            env=_go_env(repo_root),
            check=check,
        )
    if area == "dicom":
        # The public pixel path needs the cgo codecs for the JPEG-family fixtures. When
        # the native libraries do not link, fall back to the default build: RLE still
        # measures and gobench reports the cgo fixtures as unavailable rows.
        proc = run(_gobench_cmd(repo_root, area, cfg, tags=CODEC_TAGS), False)
        if proc.returncode != 0:
            proc = run(_gobench_cmd(repo_root, area, cfg, tags=None), True)
    else:
        proc = run(_gobench_cmd(repo_root, area, cfg, tags=None), True)
    return [Result(**row) for row in json.loads(proc.stdout)]
