# Mirror of the mise tasks for contributors who do not use mise.
# See mise.toml for the canonical task definitions.

.PHONY: test-dicom test-dimse test-dicomweb test-hl7v2 test-fhir test-convert \
	test-skeleton test-fhir-gen interop-dimse interop-dicomweb dicom-dciodvfy dicom-pydicom \
	gen-fhir-r5 gen-fhir gen-verify fhir-refresh-r5

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

## Run the FHIR generator (fhir/internal) test suite (race).
test-fhir-gen:
	go test -race ./fhir/internal/...

## Regenerate the FHIR R5 release package from the pinned bundle (functional from M6 Increment 1).
gen-fhir-r5:
	cd fhir && go generate ./gen.go

## Regenerate every FHIR release package from the pinned bundles (functional from M6 Increment 1).
gen-fhir:
	go generate ./fhir/...

## Regenerate the FHIR release packages and fail on any drift (wired in M6 Increment 13).
gen-verify:
	go generate ./fhir/... && git diff --exit-code -- fhir/r4 fhir/r5

## Refresh-only: re-download and re-checksum the vendored HL7 FHIR R5 bundle (never at generate time).
fhir-refresh-r5:
	tools/fhir-definitions/refresh.sh r5

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
