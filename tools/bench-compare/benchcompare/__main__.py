"""Orchestrator: run both sides over the same fixtures, write JSON, render markdown.

Usage (from tools/bench-compare):

    uv run python -m benchcompare --mode smoke   # harness validation, tiny N
    uv run python -m benchcompare --mode full    # publishing run (quiet host)
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

from . import bench_dicom, bench_dimse, bench_fhir, bench_hl7, gorunner, render
from .common import FULL, SMOKE, Result, capture_environment

REPO_ROOT = Path(__file__).resolve().parents[3]


def main() -> int:
    parser = argparse.ArgumentParser(prog="benchcompare")
    parser.add_argument("--mode", choices=["smoke", "full"], default="full")
    parser.add_argument(
        "--out",
        type=Path,
        default=REPO_ROOT / "tools" / "bench-compare" / "results",
        help="directory for comparative.json and comparative.md",
    )
    parser.add_argument(
        "--doc",
        type=Path,
        default=REPO_ROOT / "docs" / "conformance" / "benchmarks" / "comparative.md",
        help="committed doc whose generated block is rewritten",
    )
    args = parser.parse_args()
    cfg = SMOKE if args.mode == "smoke" else FULL
    testdata = REPO_ROOT / "testdata"

    results: list[Result] = []

    print(f"bench-compare: mode={cfg.mode} repo={REPO_ROOT}", flush=True)

    print("bench-compare: go test -bench (default build: Part 10 + RLE)...", flush=True)
    default_rows, err = gorunner.run_go_test_benches(REPO_ROOT, cfg, tags=None)
    if err:
        print(f"bench-compare: FATAL default-build go benches failed:\n{err}", file=sys.stderr)
        return 1
    results += default_rows

    print("bench-compare: go test -bench (cgo codec build)...", flush=True)
    codec_rows, err = gorunner.run_go_test_benches(REPO_ROOT, cfg, tags=gorunner.CODEC_TAGS)
    if err:
        print("bench-compare: cgo codec build unavailable on this host; rows marked", flush=True)
        results += gorunner.codec_unavailable_rows(err)
    else:
        results += codec_rows
    results += gorunner.go_unsupported_rows()

    for area in ("dicom", "dimse", "hl7", "fhir"):
        print(f"bench-compare: gobench -area {area}...", flush=True)
        results += gorunner.run_gobench(REPO_ROOT, area, cfg)

    print("bench-compare: pydicom (Part 10 + pixel decode/encode)...", flush=True)
    results += bench_dicom.run(testdata, cfg)
    print("bench-compare: pynetdicom (loopback C-STORE)...", flush=True)
    results += bench_dimse.run(testdata, cfg)
    print("bench-compare: python-hl7 (parse)...", flush=True)
    results += bench_hl7.run(testdata, cfg)
    print("bench-compare: fhir.resources (Bundle)...", flush=True)
    results += bench_fhir.run(testdata, cfg)

    meta = capture_environment(gorunner.go_version(REPO_ROOT))
    meta["mode"] = cfg.mode
    meta["repeats"] = cfg.repeats
    meta["warmup"] = cfg.warmup

    args.out.mkdir(parents=True, exist_ok=True)
    json_path = args.out / "comparative.json"
    json_path.write_text(
        json.dumps({"meta": meta, "results": [r.to_dict() for r in results]}, indent=2) + "\n",
        encoding="utf-8",
    )
    md_path = args.out / "comparative.md"
    md_path.write_text(
        render.render_provenance(meta) + "\n\n" + render.render_tables(results, cfg.mode) + "\n",
        encoding="utf-8",
    )
    render.render_into_doc(args.doc, results, meta)

    ok = sum(1 for r in results if r.status == "ok")
    other = [(r.name, r.side, r.status) for r in results if r.status != "ok"]
    print(f"bench-compare: wrote {json_path}")
    print(f"bench-compare: wrote {md_path}")
    print(f"bench-compare: updated {args.doc}")
    print(f"bench-compare: {ok} measured series; {len(other)} non-measured rows: {other}")
    if cfg.mode == "smoke":
        print(
            "bench-compare: SMOKE RUN - numbers are preliminary; rerun with --mode full "
            "on a quiet host before publishing"
        )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
