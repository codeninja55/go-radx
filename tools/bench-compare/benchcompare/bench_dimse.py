"""pynetdicom side: loopback C-STORE throughput, same-stack SCU -> SCP.

Mirror of the gobench dimse area: one sample = one association carrying
``dimse_small`` small + ``dimse_medium`` medium sequential C-STOREs over
127.0.0.1; association setup/release sit outside the timed window; the SCP
handler forces the dataset decode (event.dataset) and returns success without
persisting. This measures stack throughput (PDU framing + dataset codec), not
interop - each side talks to itself.
"""

from __future__ import annotations

import time
from pathlib import Path

import pydicom
from pynetdicom import AE, AllStoragePresentationContexts, evt
from pynetdicom.presentation import build_context

from .common import SIDE_PY, BenchConfig, Result

SMALL_FIXTURE = "liver.dcm"
MEDIUM_FIXTURE = "MR2_UNCI.dcm"


def _handle_store(event: evt.Event) -> int:
    ds = event.dataset  # forces the dataset decode, mirroring the go-radx dispatcher
    ds.file_meta = event.file_meta
    _ = ds.SOPInstanceUID
    return 0x0000


def run(testdata: Path, cfg: BenchConfig) -> list[Result]:
    small_path = testdata / "dicom" / SMALL_FIXTURE
    medium_path = testdata / "dicom" / MEDIUM_FIXTURE
    small = pydicom.dcmread(small_path)
    medium = pydicom.dcmread(medium_path)

    scp = AE(ae_title="BENCHSCP")
    scp.supported_contexts = AllStoragePresentationContexts
    server = scp.start_server(
        ("127.0.0.1", 0), block=False, evt_handlers=[(evt.EVT_C_STORE, _handle_store)]
    )
    port = server.socket.getsockname()[1]

    scu = AE(ae_title="BENCHSCU")
    for ds in (small, medium):
        scu.requested_contexts = scu.requested_contexts + [
            build_context(ds.SOPClassUID, [str(ds.file_meta.TransferSyntaxUID)])
        ]

    ops = cfg.dimse_small + cfg.dimse_medium
    total_bytes = (
        cfg.dimse_small * small_path.stat().st_size + cfg.dimse_medium * medium_path.stat().st_size
    )

    def sample() -> float:
        assoc = scu.associate("127.0.0.1", port, ae_title="BENCHSCP")
        if not assoc.is_established:
            raise RuntimeError("pynetdicom association refused")
        try:
            start = time.perf_counter()
            for _ in range(cfg.dimse_small):
                status = assoc.send_c_store(small)
                if not status or status.Status != 0x0000:
                    raise RuntimeError(f"send_c_store small failed: {status}")
            for _ in range(cfg.dimse_medium):
                status = assoc.send_c_store(medium)
                if not status or status.Status != 0x0000:
                    raise RuntimeError(f"send_c_store medium failed: {status}")
            return time.perf_counter() - start
        finally:
            assoc.release()

    try:
        for _ in range(cfg.warmup):
            sample()
        samples = [sample() for _ in range(cfg.repeats)]
    finally:
        server.shutdown()

    return [
        Result(
            area="dimse",
            name="dimse-cstore-loopback",
            fixture=f"{cfg.dimse_small}x {SMALL_FIXTURE} + {cfg.dimse_medium}x {MEDIUM_FIXTURE}",
            side=SIDE_PY,
            library="pynetdicom",
            ops=ops,
            bytes_per_op=total_bytes / ops,
            samples_s=samples,
            note="same-stack SCU->SCP over loopback; association setup excluded; no persistence",
        )
    ]
