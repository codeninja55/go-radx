# Mirror of the mise tasks for contributors who do not use mise.
# See mise.toml for the canonical task definitions.

.PHONY: test-dicom dicom-dciodvfy dicom-pydicom

## Run the dicom package test suite (race + coverage).
test-dicom:
	go test -race -cover ./dicom/...

## Validate written DICOM files with dcmtk dciodvfy (skips if absent).
dicom-dciodvfy:
	tools/dicom-conformance/dciodvfy.sh

## Round-trip vendored fixtures through pydicom (skips if absent).
dicom-pydicom:
	python3 tools/dicom-conformance/pydicom_roundtrip.py testdata/dicom
