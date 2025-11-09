# FHIR Code Generator

This directory contains the code generator that creates Go types from FHIR StructureDefinitions.

## Overview

The FHIR code generator reads FHIR StructureDefinition JSON files and generates idiomatic Go structs for
FHIR resources and complex types. It supports both FHIR R4 and R5 specifications.

## Features

- **Complete R4 and R5 support**: Generates types for all FHIR resources and complex types
- **Type-safe primitives**: Uses custom primitive types (Date, DateTime, Time, Instant) with validation
- **BackboneElements**: Generates nested struct types for complex inline structures
- **Resource type constants**: Generates constants like `ResourceTypePatient = "Patient"`
- **Selective generation**: Generate only specific resources with `-resources` flag
- **Verbose logging**: Track progress with `-verbose` flag

## Building the Generator

```bash
cd fhir/scripts/gen
go build -o bin/fhir-gen .
```

## Usage

### Basic Usage

Generate all R4 resources and types:

```bash
./fhir/scripts/gen/bin/fhir-gen \
  -version r4 \
  -input fhir_schemas/profiles-resources.json \
  -output fhir/r4/resources
```

Generate all R5 resources and types:

```bash
./fhir/scripts/gen/bin/fhir-gen \
  -version r5 \
  -input fhir_schemas/r5/profiles-resources.json \
  -output fhir/r5/resources
```

### Selective Generation

Generate only specific resources (useful for development and testing):

```bash
./fhir/scripts/gen/bin/fhir-gen \
  -version r4 \
  -input fhir_schemas/profiles-resources.json \
  -output /tmp/test-output \
  -resources "Patient,Observation,Bundle"
```

### Verbose Mode

Enable verbose output to see detailed progress:

```bash
./fhir/scripts/gen/bin/fhir-gen \
  -version r4 \
  -input fhir_schemas/profiles-resources.json \
  -output fhir/r4/resources \
  -verbose
```

## Command-Line Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-version` | string | `r4` | FHIR version (`r4` or `r5`) |
| `-input` | string | `fhir_schemas/profiles-resources.json` | Path to FHIR StructureDefinitions bundle |
| `-output` | string | **(required)** | Output directory for generated Go files |
| `-resources` | string | `""` (all) | Comma-separated list of resources to generate |
| `-verbose` | bool | `false` | Enable verbose logging |

## Generated Code Structure

The generator creates one Go file per resource or complex type:

```
fhir/r4/resources/
├── patient.go              # Patient resource
├── observation.go          # Observation resource
├── bundle.go              # Bundle resource
├── humanname.go           # HumanName complex type
├── codeableconcept.go     # CodeableConcept complex type
└── ...
```

Each file contains:

1. **Resource type constant** (for resources only):
   ```go
   const ResourceTypePatient = "Patient"
   ```

2. **BackboneElement types** (nested structs):
   ```go
   type PatientContact struct {
       Name     *HumanName `json:"name,omitempty"`
       // ...
   }
   ```

3. **Main resource/type struct**:
   ```go
   type Patient struct {
       ID        *string           `json:"id,omitempty"`
       Name      []HumanName       `json:"name,omitempty"`
       BirthDate *primitives.Date  `json:"birthDate,omitempty"`
       // ...
   }
   ```

## How It Works

1. **Parse**: Reads FHIR StructureDefinition bundle JSON file
2. **Filter**: Applies resource filter if `-resources` flag is provided
3. **Extract**: Extracts element definitions and builds internal model
4. **Map**: Maps FHIR types to Go types (e.g., `date` → `primitives.Date`)
5. **Generate**: Creates Go source code with proper imports and formatting
6. **Write**: Writes formatted Go files to output directory

## Type Mappings

The generator maps FHIR types to Go types as follows:

| FHIR Type | Go Type | Notes |
|-----------|---------|-------|
| `boolean` | `bool` | |
| `integer` | `int` | |
| `positiveInt` | `int` | |
| `unsignedInt` | `uint` | |
| `integer64` | `int64` | R5 only |
| `decimal` | `float64` | |
| `string` | `string` | |
| `uri` | `string` | |
| `url` | `string` | |
| `canonical` | `string` | |
| `code` | `string` | |
| `id` | `string` | |
| `markdown` | `string` | |
| `base64Binary` | `string` | |
| `date` | `primitives.Date` | Custom type with partial precision |
| `dateTime` | `primitives.DateTime` | Custom type with optional timezone |
| `time` | `primitives.Time` | Custom type (HH:MM:SS) |
| `instant` | `primitives.Instant` | Custom type (requires timezone) |
| `Reference` | `Reference` | Complex type |
| `CodeableConcept` | `CodeableConcept` | Complex type |
| `Resource` | `any` | Polymorphic contained resources |
| BackboneElement | Nested struct | Inline definitions |

## Regenerating Types

After updating FHIR schemas or modifying the generator:

1. **Rebuild the generator**:
   ```bash
   cd fhir/scripts/gen
   go build -o bin/fhir-gen .
   ```

2. **Regenerate R4 types**:
   ```bash
   cd ../../..  # Back to project root
   ./fhir/scripts/gen/bin/fhir-gen \
     -version r4 \
     -input fhir_schemas/profiles-resources.json \
     -output fhir/r4/resources
   ```

3. **Regenerate R5 types**:
   ```bash
   ./fhir/scripts/gen/bin/fhir-gen \
     -version r5 \
     -input fhir_schemas/r5/profiles-resources.json \
     -output fhir/r5/resources
   ```

4. **Run tests**:
   ```bash
   cd fhir/r4/resources && go test -v
   cd ../../r5/resources && go test -v
   ```

## Project Structure

```
fhir/scripts/gen/
├── README.md                # This file
├── main.go                  # CLI entry point
├── model/
│   └── definition.go        # Internal data models
├── parser/
│   ├── structdef.go         # FHIR StructureDefinition parser
│   └── typemapper.go        # FHIR to Go type mapper
└── codegen/
    ├── builder.go           # High-level builder
    └── generator.go         # Low-level code generator
```

## Development

### Adding New Type Mappings

To add support for new FHIR types, update `parser/typemapper.go`:

```go
primitiveMap: map[string]string{
    "newFhirType": "primitives.NewGoType",
    // ...
}
```

### Customizing Code Generation

The code generation template is in `codegen/generator.go` as the `fileTemplate` constant.
Modify this template to change the structure of generated files.

### Testing the Generator

Generate a small subset of resources for testing:

```bash
./fhir/scripts/gen/bin/fhir-gen \
  -version r4 \
  -input fhir_schemas/profiles-resources.json \
  -output /tmp/test-gen \
  -resources "Patient,Observation" \
  -verbose
```

## Troubleshooting

### "file not found" error

Ensure the input file path is correct relative to your current directory:

```bash
# From project root
./fhir/scripts/gen/bin/fhir-gen -input fhir_schemas/profiles-resources.json ...

# From generator directory
./bin/fhir-gen -input ../../../fhir_schemas/profiles-resources.json ...
```

### Generated code doesn't compile

1. Check that primitive types are correctly imported
2. Ensure all referenced types exist in the output directory
3. Run `go fmt` on the generated files
4. Check for circular dependencies in BackboneElements

### Missing resources

Use `-verbose` flag to see which resources are being skipped:

```bash
./fhir/scripts/gen/bin/fhir-gen ... -verbose 2>&1 | grep "Skipping"
```

Abstract resources (like `Resource`, `DomainResource`) are intentionally skipped.

## License

This generator is part of the go-radx project. See the main project LICENSE file.