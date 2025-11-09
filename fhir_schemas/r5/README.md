# FHIR R5 Schemas

This directory contains FHIR R5 StructureDefinition files used for code generation.

## Files

- **profiles-resources.json** - FHIR R5 resource definitions (146 resources)
- **profiles-types.json** - FHIR R5 complex type definitions
- **profiles-others.json** - Other FHIR profiles (extensions, etc.)
- **valuesets.json** - FHIR value sets for coded elements
- **search-parameters.json** - FHIR search parameter definitions
- **conceptmaps.json** - FHIR concept maps
- **dataelements.json** - FHIR data element definitions

## Source

- **FHIR Version**: R5 (5.0.0)
- **Source URL**: https://hl7.org/fhir/R5/
- **Download Date**: October 15, 2024
- **Schema Bundle**: https://hl7.org/fhir/R5/fhir.schema.json.zip

## Usage

These schemas are used by the FHIR code generator to produce type-safe Go structs:

```bash
# Generate R5 resources
./fhir/scripts/gen/bin/fhirgen -version r5 \
  -input fhir_schemas/r5/profiles-resources.json \
  -output fhir/r5/resources \
  -verbose

# Generate R5 complex types
./fhir/scripts/gen/bin/fhirgen -version r5 \
  -input fhir_schemas/r5/profiles-types.json \
  -output fhir/r5/types \
  -verbose
```

## Updating Schemas

To update to a newer R5 version:

```bash
# Download latest R5 schemas
curl -L -o fhir_schemas/r5/fhir.schema.json.zip https://hl7.org/fhir/R5/fhir.schema.json.zip

# Extract
cd fhir_schemas/r5
unzip -o fhir.schema.json.zip
rm fhir.schema.json.zip

# Update this README with new version and date
```

## Version Control

These schema files are tracked in version control to ensure:
- Reproducible builds across environments
- Ability to regenerate code with identical schemas
- Clear documentation of which FHIR version is being used
