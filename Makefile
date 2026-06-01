# Mirror of the mise tasks for contributors who do not use mise.
# See mise.toml for the canonical task definitions.

.PHONY: test-dicom test-dimse test-dicomweb test-hl7v2 test-fhir test-convert \
	test-skeleton interop-dimse interop-dicomweb dicom-dciodvfy dicom-pydicom

## Run the dicom package test suite (race + coverage).
test-dicom:
	go test -race -cover ./dicom/...

## Run the dimse package test suite (race + coverage).
test-dimse:
	go test -race -cover ./dimse/...

## Run the dicomweb package test suite (race + coverage).
test-dicomweb:
	go test -race -cover ./dicomweb/...

## Run the hl7v2 package test suite (race + coverage).
test-hl7v2:
	go test -race -cover ./hl7v2/...

## Run the fhir package test suite (race + coverage).
test-fhir:
	go test -race -cover ./fhir/...

## Run the convert package test suite (race + coverage).
test-convert:
	go test -race -cover ./convert/...

## Run the M2 walking-skeleton package suites (race + coverage).
test-skeleton:
	go test -race -cover ./dimse/... ./dicomweb/... ./hl7v2/... ./fhir/... ./convert/... ./server/...

## Run the DIMSE interop gate against the Orthanc/dcm4chee-arc containers.
interop-dimse:
	go test -tags interop -v ./dimse/integration/...

## Run the DICOMweb interop gate against the Orthanc container.
interop-dicomweb:
	go test -tags interop -v ./dicomweb/integration/...

## Validate written DICOM files with dcmtk dciodvfy (skips if absent).
dicom-dciodvfy:
	tools/dicom-conformance/dciodvfy.sh

## Round-trip vendored fixtures through pydicom (skips if absent).
dicom-pydicom:
	python3 tools/dicom-conformance/pydicom_roundtrip.py testdata/dicom
