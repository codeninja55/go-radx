"""Normalize raw samples into derived metrics and render the markdown tables.

The committed doc (docs/conformance/benchmarks/comparative.md) carries a marked
generated block; render_into_doc rewrites only that block, so regenerating the
published numbers on a quiet host is one command and the surrounding methodology
prose is never touched by the harness.
"""

from __future__ import annotations

import statistics
from pathlib import Path

from .common import STATUS_OK, Result

BEGIN_MARK = "<!-- bench-compare:begin generated tables -->"
END_MARK = "<!-- bench-compare:end generated tables -->"

AREA_TITLES = {
    "dicom": "DICOM: Part 10 decode and per-transfer-syntax pixel codecs",
    "dimse": "DIMSE: loopback C-STORE throughput (same-stack pairs)",
    "hl7v2": "HL7 v2: parse throughput",
    "fhir": "FHIR R5: Bundle unmarshal / marshal / validate",
}


def derive(r: Result) -> dict[str, float | None]:
    """ns/op, ops/sec, MB/s from the median sample - same arithmetic both sides."""
    if r.status != STATUS_OK or not r.samples_s or r.ops <= 0:
        return {"ns_per_op": None, "ops_per_sec": None, "mb_per_s": None}
    median_s = statistics.median(r.samples_s)
    ns_per_op = median_s / r.ops * 1e9
    ops_per_sec = r.ops / median_s
    mb_per_s = None
    if r.bytes_per_op:
        mb_per_s = r.bytes_per_op * r.ops / median_s / 1e6
    return {"ns_per_op": ns_per_op, "ops_per_sec": ops_per_sec, "mb_per_s": mb_per_s}


def _fmt(v: float | None, unit: str = "") -> str:
    if v is None:
        return "-"
    if v >= 1e9:
        s = f"{v / 1e9:.2f}G"
    elif v >= 1e6:
        s = f"{v / 1e6:.2f}M"
    elif v >= 1e3:
        s = f"{v / 1e3:.2f}k"
    else:
        s = f"{v:.2f}"
    return s + unit


def _row_key(r: Result) -> tuple[str, str, str]:
    return (r.area, r.name, r.fixture)


def _pair_rows(results: list[Result]) -> dict[tuple[str, str, str], dict[str, Result]]:
    """Pair go-radx and python rows on (area, name, fixture).

    Fixture-keyed names pair exactly; rows whose fixtures differ across sides
    (unsupported/unavailable markers) fall back to pairing on (area, name).
    """
    paired: dict[tuple[str, str, str], dict[str, Result]] = {}
    for r in results:
        paired.setdefault(_row_key(r), {})[r.side] = r
    # Fold one-sided placeholder fixtures ("n/a", "(codec build)") onto the real row.
    keys = list(paired.keys())
    for key in keys:
        area, name, fixture = key
        if fixture not in ("n/a", "(codec build)"):
            continue
        for other in keys:
            if other == key or other[0] != area or other[1] != name:
                continue
            for side, r in paired[key].items():
                paired[other].setdefault(side, r)
            paired.pop(key, None)
            break
    return paired


def render_tables(results: list[Result], mode: str) -> str:
    paired = _pair_rows(results)
    lines: list[str] = []
    lines.append(BEGIN_MARK)
    lines.append("")
    if mode != "full":
        lines.append(
            "> **PRELIMINARY** - smoke-run numbers from a loaded development host, captured only"
        )
        lines.append(
            "> to prove the harness executes end-to-end. Regenerate on a quiet host via"
        )
        lines.append("> `mise run bench:compare` before citing any figure.")
        lines.append("")
    for area in ("dicom", "dimse", "hl7v2", "fhir"):
        keys = sorted(k for k in paired if k[0] == area)
        if not keys:
            continue
        lines.append(f"### {AREA_TITLES[area]}")
        lines.append("")
        lines.append(
            "| Benchmark | Fixture | go-radx ns/op | go-radx MB/s | go allocs/op "
            "| Python ns/op | Python MB/s | Speedup (py/go) |"
        )
        lines.append("|---|---|---:|---:|---:|---:|---:|---:|")
        for key in keys:
            sides = paired[key]
            g, p = sides.get("go-radx"), sides.get("python")
            gd = derive(g) if g else {"ns_per_op": None, "mb_per_s": None}
            pd = derive(p) if p else {"ns_per_op": None, "mb_per_s": None}
            speedup = None
            if gd["ns_per_op"] and pd["ns_per_op"]:
                speedup = pd["ns_per_op"] / gd["ns_per_op"]
            note = ""
            for r in (g, p):
                if r is not None and r.status != STATUS_OK:
                    note = f" ({r.side}: {r.status})"
                    break
            lines.append(
                f"| {key[1]}{note} | {key[2]} "
                f"| {_fmt(gd['ns_per_op'])} | {_fmt(gd['mb_per_s'])} "
                f"| {_fmt(g.allocs_per_op) if g and g.allocs_per_op else '-'} "
                f"| {_fmt(pd['ns_per_op'])} | {_fmt(pd['mb_per_s'])} "
                f"| {f'{speedup:.1f}x' if speedup else '-'} |"
            )
        lines.append("")
        notes = sorted(
            {
                f"- `{r.name}` ({r.side}): {r.status} - {r.note}"
                for k in keys
                for r in paired[k].values()
                if r.status != STATUS_OK and r.note
            }
        )
        if notes:
            lines.append("Rows without numbers:")
            lines.append("")
            lines.extend(notes)
            lines.append("")
    lines.append(END_MARK)
    return "\n".join(lines)


def render_provenance(meta: dict[str, object]) -> str:
    pins = meta.get("python_pins", {})
    pin_str = ", ".join(f"{k} {v}" for k, v in sorted(pins.items()))  # type: ignore[union-attr]
    return "\n".join(
        [
            "<!-- bench-compare:begin provenance -->",
            "",
            f"- Generated: {meta.get('timestamp_utc')} (mode: {meta.get('mode')})",
            f"- Host: {meta.get('cpu')} / {meta.get('memory')} bytes RAM / {meta.get('os')}",
            f"- Go: {meta.get('go_version')}",
            f"- Python: {meta.get('python_version')}",
            f"- Python pins: {pin_str}",
            "",
            "<!-- bench-compare:end provenance -->",
        ]
    )


def _replace_block(text: str, begin: str, end: str, block: str) -> str:
    start = text.find(begin)
    stop = text.find(end)
    if start == -1 or stop == -1:
        raise ValueError(f"markers {begin!r}/{end!r} not found in doc")
    return text[:start] + block + text[stop + len(end) :]


def render_into_doc(doc_path: Path, results: list[Result], meta: dict[str, object]) -> None:
    text = doc_path.read_text(encoding="utf-8")
    text = _replace_block(
        text, BEGIN_MARK, END_MARK, render_tables(results, str(meta.get("mode")))
    )
    text = _replace_block(
        text,
        "<!-- bench-compare:begin provenance -->",
        "<!-- bench-compare:end provenance -->",
        render_provenance(meta),
    )
    doc_path.write_text(text, encoding="utf-8")
