# HL7 v2 test fixtures

This directory holds the M5 HL7 v2 test corpus: a small, purposeful set of HL7 v2.x messages, each chosen so a specific
increment of the HL7 v2 depth plan (`docs/plans/agentic-dev-2026-06-02-hl7-depth.md`) has exactly the coverage it
needs. Fixtures are test corpora only, vendored with upstream license attribution per PRD §5.3 and §11.1.

## Synthetic, fictitious PHI only

Per the project PHI rule (PRD §9.1) and the M5 plan's PHI constraint, every fixture in this corpus is **authored by
go-radx with entirely synthetic, fictitious patient data**. Names follow a `TESTPATIENT^<given>` (or an obviously
fictitious `O'BRIEN-SMITH`) scheme; medical record numbers (`MRN000NNNN`), account numbers, visit numbers, placer/filler
order numbers, control IDs, dates, addresses, and phone numbers are invented and do not identify any real person. No
fixture content or filename encodes real Protected Health Information.

These messages are **go-radx originals**, not copies of any upstream corpus. They are *shaped after* the canonical HL7
v2 message structures documented in `docs/reference/hl7v2.md` ("Worked examples") and demonstrated by the
[`python-hl7`](https://github.com/johnpaulett/python-hl7) test suite, which is the project's parity reference for
canonical message shapes. python-hl7's own sample messages derive from third-party sources (the HL7 Normative Edition
and a Victorian Health government sample set), so rather than redistribute that content, go-radx authored equivalent
synthetic messages with fictitious PHI and uses python-hl7 only as the structural/shape reference. The python-hl7
license (BSD-3-Clause) is vendored as `LICENSE-python-hl7.txt` to attribute that reference.

## Provenance and coverage

The Message type column is the `MSH-9` value the fixture carries (or the container header for the batch/file). Authorship
records that the message content is a go-radx synthetic original; Shape reference records the canonical structure it
follows. Exercises names the increment(s) that consume the fixture.

| File | Message type | Authorship | Shape reference | Exercises |
| --- | --- | --- | --- | --- |
| `adt-a01.hl7` | `ADT^A01` (admission) | go-radx synthetic | python-hl7 ADT shape; reference doc typed segments | `MSH`/`EVN`/`PID`/`PV1` typed segments, attending-doctor `XPN`, `PID-11` `XAD` (Inc 2), `ADT` lens + A01 trigger (Inc 3) |
| `adt-a03.hl7` | `ADT^A03` (discharge) | go-radx synthetic | python-hl7 ADT shape | A03 trigger-event scope and discharge `DTM` (Inc 3) |
| `oru-r01.hl7` | `ORU^R01` (results) | go-radx synthetic | reference doc ORU worked example | Two `OBR`+`OBX` result groups, `NM`/`TX` value types, repeated `OBX-5` (`~`) (Inc 2), `ORU.Results()` grouping (Inc 4) |
| `orm-o01.hl7` | `ORM^O01` (order) | go-radx synthetic | reference doc ORM worked example | `ORC`+`OBR` order group; the M2 `ORM` lens template for Inc 4 grouping |
| `omg-o19.hl7` | `OMG^O19` (imaging order) | go-radx synthetic | conformance doc OMG variant | `ORM` lens accepts both `ORM` and `OMG` codes (open question 5) |
| `ack.hl7` | `ACK^R01` | go-radx synthetic | reference doc ACK example | `MSA` with `AA` ack code echoing the `ORU` control ID (Inc 7) |
| `nonstandard-delim.hl7` | `ORU^R01` | go-radx synthetic | python-hl7 non-standard-delimiter case | Non-standard delimiters (`#@+$%`): encoding derivation and byte-exact round-trip (Inc 1, 6) |
| `escaped.hl7` | `ORU^R01` | go-radx synthetic | HL7 Chapter 2 §2.10 | `\F\`/`\S\`/`\T\`/`\E\`/`\Xdd\` escape sequences in field values (Inc 5, 6) |
| `batch.hl7` | `BHS`/`BTS` batch | go-radx synthetic | python-hl7 batch shape | `BHS`/`BTS` batch of two `ADT^A04` messages (Inc 11) |
| `file.hl7` | `FHS`/`FTS` file | go-radx synthetic | python-hl7 file shape | `FHS`/`FTS` file containing one batch (Inc 11) |

## Notes on fixture conventions

- **Segment terminators.** Every fixture uses the canonical HL7 carriage-return (`\r`, `0x0D`) segment terminator, with
  a trailing `\r` after the final segment, and contains no line-feed (`\n`) bytes. This is the form the parser
  round-trips byte-for-byte; the harness self-test asserts byte-exact `Parse`→`MarshalText` for every single-message
  fixture.
- **`nonstandard-delim.hl7`** declares field `#`, component `@`, repetition `+`, escape `$`, and subcomponent `%` in its
  `MSH-1`/`MSH-2`. It exercises the rule that delimiters are derived from the header per message and never hardcoded, so
  a non-standard sender round-trips correctly.
- **`escaped.hl7`** carries `\T\` in the patient family name and `\S\`, `\F\`, `\E\`, and `\X0D\` in an `OBX` note. The
  raw escape sequences round-trip byte-exact today; their *decoding* is exercised by the escape/unescape increment.
- **`batch.hl7` and `file.hl7`** are multi-message containers; `Parse` accepts only a single MSH-led message, so the
  harness loads their raw bytes and checks the container header/trailer. Their `ParseBatch`/`ParseFile` parsing lands in
  a later increment.

## How the harness loads these

The corpus harness lives in `hl7v2/corpus_test.go` (test code, not production). It registers every fixture with its
container kind and the increment it serves, and exposes `corpusRaw(t, name)` (exact bytes for any fixture) and
`corpusMessage(t, name)` (parsed `*Message` for the single-message fixtures). The later increments load their inputs
through these helpers rather than embedding raw HL7 in each test.

## Upstream sources

- python-hl7 (parity / shape reference): <https://github.com/johnpaulett/python-hl7>. The canonical message structures
  these synthetic fixtures follow are demonstrated in python-hl7's `tests/samples.py`.

## License attribution

- `LICENSE-python-hl7.txt` — BSD-3-Clause license of python-hl7 (Copyright © 2009-2020 John Paulett), attributed as the
  parity and message-shape reference for this corpus. No python-hl7 message content is redistributed here; the fixtures
  are go-radx synthetic originals shaped after python-hl7's canonical structures.
