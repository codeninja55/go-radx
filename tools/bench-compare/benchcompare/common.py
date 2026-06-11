"""Shared result schema, timing harness, and environment capture.

Both sides emit the same shape: raw wall-clock samples (seconds per sample, each
sample timing ``ops`` back-to-back operations). All derived metrics (ns/op,
ops/sec, MB/s) are computed once, by the renderer, with the same arithmetic for
both sides.
"""

from __future__ import annotations

import dataclasses
import platform
import subprocess
import sys
import time
from collections.abc import Callable
from dataclasses import dataclass, field
from importlib import metadata

SIDE_GO = "go-radx"
SIDE_PY = "python"

STATUS_OK = "ok"
STATUS_UNSUPPORTED = "unsupported"
STATUS_UNAVAILABLE = "unavailable"
STATUS_ERROR = "error"


@dataclass
class Result:
    """One benchmark series on one side. Mirrors the gobench JSON schema."""

    area: str
    name: str
    fixture: str
    side: str
    library: str
    status: str = STATUS_OK
    note: str = ""
    ops: int = 0
    bytes_per_op: float | None = None
    samples_s: list[float] = field(default_factory=list)
    allocs_per_op: float | None = None

    def to_dict(self) -> dict[str, object]:
        return dataclasses.asdict(self)


@dataclass
class BenchConfig:
    """Per-mode knobs. ``smoke`` validates the harness end-to-end with tiny N;

    ``full`` is the publishing configuration (median of N >= 5 samples).
    """

    mode: str
    repeats: int
    warmup: int
    hl7_iters: int
    fhir_iters: int
    dicom_iters: int
    dimse_small: int
    dimse_medium: int
    go_benchtime: str
    go_count: int


SMOKE = BenchConfig(
    mode="smoke",
    repeats=2,
    warmup=1,
    hl7_iters=50,
    fhir_iters=20,
    dicom_iters=3,
    dimse_small=5,
    dimse_medium=2,
    go_benchtime="3x",
    go_count=2,
)

FULL = BenchConfig(
    mode="full",
    repeats=7,
    warmup=2,
    hl7_iters=2000,
    fhir_iters=200,
    dicom_iters=20,
    dimse_small=200,
    dimse_medium=20,
    go_benchtime="50x",
    go_count=7,
)


def measure(fn: Callable[[], object], ops: int, repeats: int, warmup: int) -> list[float]:
    """Time ``ops`` calls of fn per sample, ``repeats`` times after ``warmup`` samples."""
    for _ in range(warmup):
        for _ in range(ops):
            fn()
    samples: list[float] = []
    for _ in range(repeats):
        start = time.perf_counter()
        for _ in range(ops):
            fn()
        samples.append(time.perf_counter() - start)
    return samples


def _sysctl(key: str) -> str:
    try:
        return subprocess.run(
            ["sysctl", "-n", key], capture_output=True, text=True, check=True
        ).stdout.strip()
    except (OSError, subprocess.CalledProcessError):
        return ""


def _linux_cpu() -> str:
    try:
        with open("/proc/cpuinfo", encoding="utf-8") as f:
            for line in f:
                if line.lower().startswith("model name"):
                    return line.split(":", 1)[1].strip()
    except OSError:
        pass
    return platform.processor() or platform.machine()


def capture_environment(go_version: str) -> dict[str, object]:
    """Record the hardware and toolchain the run was produced on (methodology)."""
    if sys.platform == "darwin":
        cpu = _sysctl("machdep.cpu.brand_string")
        mem = _sysctl("hw.memsize")
    else:
        cpu = _linux_cpu()
        mem = ""
        try:
            with open("/proc/meminfo", encoding="utf-8") as f:
                first = f.readline()
            mem = first.split(":", 1)[1].strip() if ":" in first else ""
        except OSError:
            pass

    pins: dict[str, str] = {}
    for dist in (
        "pydicom",
        "pylibjpeg",
        "pylibjpeg-libjpeg",
        "pylibjpeg-openjpeg",
        "pylibjpeg-rle",
        "pyjpegls",
        "pynetdicom",
        "hl7",
        "fhir.resources",
        "numpy",
    ):
        try:
            pins[dist] = metadata.version(dist)
        except metadata.PackageNotFoundError:
            pins[dist] = "absent"

    return {
        "timestamp_utc": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "os": f"{platform.system()} {platform.release()} ({platform.machine()})",
        "cpu": cpu,
        "memory": mem,
        "go_version": go_version,
        "python_version": platform.python_version(),
        "python_pins": pins,
    }
