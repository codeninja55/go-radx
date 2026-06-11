"""fhir.resources side: R5 Bundle unmarshal, marshal, validate over the repo fixture.

Object-model caveat the renderer carries into the doc: fhir.resources is pydantic,
so model_validate_json IS parse + validate in one step - there is no separate
post-parse validation pass to time. The "validate" row is therefore go-radx-only,
with the Python side recorded as folded into unmarshal.
"""

from __future__ import annotations

from pathlib import Path

from fhir.resources.bundle import Bundle

from .common import SIDE_PY, STATUS_UNSUPPORTED, BenchConfig, Result, measure

FIXTURE = "Bundle.json"


def run(testdata: Path, cfg: BenchConfig) -> list[Result]:
    raw = (testdata / "fhir" / "r5" / FIXTURE).read_bytes()
    bundle = Bundle.model_validate_json(raw)
    size = float(len(raw))

    def mk(name: str, samples: list[float], note: str = "") -> Result:
        return Result(
            area="fhir",
            name=name,
            fixture=FIXTURE,
            side=SIDE_PY,
            library="fhir.resources",
            ops=cfg.fhir_iters,
            bytes_per_op=size,
            samples_s=samples,
            note=note,
        )

    unmarshal = mk(
        "fhir-bundle-unmarshal",
        measure(
            lambda: Bundle.model_validate_json(raw), cfg.fhir_iters, cfg.repeats, cfg.warmup
        ),
        note="model_validate_json (pydantic: parse and validate are one step)",
    )
    marshal = mk(
        "fhir-bundle-marshal",
        measure(lambda: bundle.model_dump_json(), cfg.fhir_iters, cfg.repeats, cfg.warmup),
        note="model_dump_json",
    )
    validate = Result(
        area="fhir",
        name="fhir-bundle-validate",
        fixture=FIXTURE,
        side=SIDE_PY,
        library="fhir.resources",
        status=STATUS_UNSUPPORTED,
        note="pydantic validates during parse; covered by the unmarshal row",
    )
    return [unmarshal, marshal, validate]
