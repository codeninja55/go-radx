"""python-hl7 side: ADT and ORU parse throughput over the repo HL7 v2 corpus.

The fixtures use CR segment terminators (the HL7 v2 wire form), which both
hl7v2.Parse and hl7.parse consume as-is, so the bytes are identical on both sides.
"""

from __future__ import annotations

from pathlib import Path

import hl7

from .common import SIDE_PY, BenchConfig, Result, measure

FIXTURES = ["adt-a01.hl7", "oru-r01.hl7"]


def run(testdata: Path, cfg: BenchConfig) -> list[Result]:
    out: list[Result] = []
    for fixture in FIXTURES:
        path = testdata / "hl7v2" / fixture
        raw = path.read_bytes()
        text = raw.decode("ascii")
        samples = measure(lambda t=text: hl7.parse(t), cfg.hl7_iters, cfg.repeats, cfg.warmup)
        out.append(
            Result(
                area="hl7v2",
                name="hl7v2-parse",
                fixture=fixture,
                side=SIDE_PY,
                library="python-hl7",
                ops=cfg.hl7_iters,
                bytes_per_op=float(len(raw)),
                samples_s=samples,
            )
        )
    return out
