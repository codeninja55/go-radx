# FHIR Code Generator

This document explains the FHIR code generator that creates Go structs from FHIR StructureDefinitions.

## Overview

The generator reads FHIR StructureDefinition JSON files and generates type-safe Go code with:
- Validation struct tags (cardinality, required, enum, summary)
- Primitive extension fields (parallel field pattern)
- Choice type expansion (multiple typed fields)
- Resource inheritance (struct embedding)
- Complete JSON serialization support

## Installation

### Using mise tasks (Recommended)

```bash
# Build the generator
mise gen:build

# Download R5 schemas
mise schema:download
```

### Manual Build

```bash
cd fhir/scripts/gen
go build -o bin/fhirgen .
```

## Usage

### Using mise tasks (Recommended)

```bash
# Generate all R5 code (resources + types)
mise gen:r5-all

# Or generate individually
mise gen:r5-resources  # Just resources
mise gen:r5-types      # Just complex types

# Verify generated code compiles and passes tests
mise gen:verify

# Clean generated code
mise gen:clean
```

### Manual Usage

```bash
# Generate all R5 resources (146 resources)
./bin/fhirgen -version r5 \
  -input fhir_schemas/r5/profiles-resources.json \
  -output fhir/r5/resources \
  -verbose

# Generate all R5 complex types
./bin/fhirgen -version r5 \
  -input fhir_schemas/r5/profiles-types.json \
  -output fhir/r5/types \
  -verbose

# Generate specific resources only
./bin/fhirgen -version r5 \
  -input fhir_schemas/r5/profiles-resources.json \
  -output fhir/r5/resources \
  -filter Patient \
  -filter Observation \
  -verbose
```

### Command-Line Options

| Flag | Description | Default | Example |
|------|-------------|---------|---------|
| `-version` | FHIR version (r4, r4b, r5) | r5 | `-version r5` |
| `-input` | Path to StructureDefinitions JSON | (required) | `-input profiles-resources.json` |
| `-output` | Output directory | (required) | `-output fhir/r5/resources` |
| `-resources` | Comma-separated resource names | (all) | `-resources Patient,Observation` |
| `-package` | Go package name | resources | `-package myresources` |
| `-verbose` | Enable verbose logging | false | `-verbose` |

## Generated Code Features

### 1. Validation Struct Tags

The generator emits FHIR struct tags for validation:

```go
type Patient struct {
    ID     *string `json:"id,omitempty" fhir:"cardinality=0..1,summary"`
    Active *bool   `json:"active,omitempty" fhir:"cardinality=0..1,summary"`
    Name   []HumanName `json:"name,omitempty" fhir:"cardinality=0..*,summary"`
    Gender *string `json:"gender,omitempty" fhir:"cardinality=0..1,enum=male|female|other|unknown,summary"`
}
```

**Tag Components:**
- `cardinality=min..max` - Occurrence constraints (0..1, 1..*, etc.)
- `required` - Field is required (min >= 1)
- `enum=val1|val2` - Allowed enum values (pipe-separated)
- `summary` - FHIR summary element flag
- `choice=group` - Choice type group name

### 2. Primitive Extension Fields

For every primitive field, the generator creates a parallel extension field:

```go
type Patient struct {
    Active    *bool `json:"active,omitempty" fhir:"cardinality=0..1,summary"`
    ActiveExt *primitives.PrimitiveExtension `json:"_active,omitempty" fhir:"cardinality=0..1"`
    
    BirthDate    *primitives.Date `json:"birthDate,omitempty" fhir:"cardinality=0..1,summary"`
    BirthDateExt *primitives.PrimitiveExtension `json:"_birthDate,omitempty" fhir:"cardinality=0..1"`
}
```

**Pattern:** `FieldName` → `FieldNameExt` with JSON tag `_fieldName`

### 3. Choice Type Expansion

Choice types (`field[x]`) are expanded into multiple typed fields:

```go
// From: deceased[x] with types boolean, dateTime
type Patient struct {
    DeceasedBoolean  *bool `json:"deceasedBoolean,omitempty" fhir:"cardinality=0..1,summary,choice=deceased"`
    DeceasedBooleanExt *primitives.PrimitiveExtension `json:"_deceasedBoolean,omitempty"`
    
    DeceasedDateTime *primitives.DateTime `json:"deceasedDateTime,omitempty" fhir:"cardinality=0..1,summary,choice=deceased"`
    DeceasedDateTimeExt *primitives.PrimitiveExtension `json:"_deceasedDateTime,omitempty"`
}
```

**Pattern:** Base name + type suffix, all tagged with same `choice=group`

### 4. Resource Inheritance

Resources extending DomainResource use struct embedding:

```go
type Patient struct {
    fhir.DomainResource
    // Patient-specific fields only
    Active *bool `json:"active,omitempty"`
    // ...
}
```

Base fields (ID, Meta, Text, Extension, etc.) are inherited from the embedded type.

### 5. BackboneElements

Nested structures become their own types:

```go
type PatientContact struct {
    ID           *string `json:"id,omitempty"`
    Relationship []CodeableConcept `json:"relationship,omitempty"`
    Name         *HumanName `json:"name,omitempty"`
    Telecom      []ContactPoint `json:"telecom,omitempty"`
}

type Patient struct {
    fhir.DomainResource
    Contact []PatientContact `json:"contact,omitempty"`
}
```

## Generator Architecture

### Components

```
fhir/scripts/gen/
├── main.go           # CLI entry point
├── parser/
│   ├── parser.go     # Loads and parses StructureDefinitions
│   ├── typemapper.go # Maps FHIR types to Go types
│   └── structdef.go  # StructureDefinition parsing
├── model/
│   └── definition.go # Intermediate representation (IR)
├── codegen/
│   ├── generator.go  # Go code generation
│   └── builder.go    # Builds IR from parsed definitions
└── bin/
    └── fhirgen       # Compiled binary
```

### Processing Pipeline

1. **Parse** - Read and parse StructureDefinitions JSON
2. **Map** - Convert FHIR types to Go types
3. **Build** - Create intermediate representation (IR)
4. **Filter** - Remove base fields for inherited types
5. **Expand** - Expand choice types to multiple fields
6. **Generate** - Emit Go code from IR
7. **Format** - Apply `go fmt`

### Type Mapping

| FHIR Type | Go Type | Package |
|-----------|---------|---------|
| boolean | bool | builtin |
| integer | int | builtin |
| string | string | builtin |
| decimal | float64 | builtin |
| uri, url, canonical | string | builtin |
| date | primitives.Date | primitives |
| dateTime | primitives.DateTime | primitives |
| time | primitives.Time | primitives |
| instant | primitives.Instant | primitives |
| code | string | builtin |
| base64Binary | string | builtin |
| Complex types | TypeName | Same package |
| Resources | TypeName | resources |

## Customization

### Filtering Resources

Generate only specific resources:

```bash
# Patient and related resources
./bin/fhirgen -version r5 \
  -input profiles-resources.json \
  -output r5/resources \
  -resources Patient,Practitioner,Organization
```

### Custom Package Name

```bash
# Generate into custom package
./bin/fhirgen -version r5 \
  -input profiles-resources.json \
  -output myapp/fhir \
  -package fhir
```

### Verbose Output

See detailed generation progress:

```bash
./bin/fhirgen -version r5 \
  -input profiles-resources.json \
  -output r5/resources \
  -verbose
```

Output:
```
Generating 146 resources...
  Generating Patient...
  Generating Observation...
  ...
Successfully generated 146 files in r5/resources
```

## Extending the Generator

### Adding Custom Type Mappings

Edit `parser/typemapper.go`:

```go
func (tm *TypeMapper) MapType(fhirType string) string {
    switch fhirType {
    case "myCustomType":
        return "CustomGoType"
    default:
        return tm.primitiveMap[fhirType]
    }
}
```

### Adding Custom Struct Tags

Edit `model/definition.go`:

```go
func (f *Field) FHIRTag() string {
    var parts []string
    
    // Add your custom tags
    if f.CustomProperty {
        parts = append(parts, "custom=true")
    }
    
    return strings.Join(parts, ",")
}
```

### Modifying Templates

The generator uses Go templates in `codegen/generator.go`. To customize output format, modify the `fileTemplate` constant.

## Testing

### Unit Tests

```bash
cd fhir/scripts/gen
go test ./...
```

### Integration Testing

Generate a known resource and verify output:

```bash
# Generate Patient
./bin/fhirgen -version r5 \
  -input test/profiles-resources.json \
  -output test/output \
  -resources Patient

# Verify compilation
cd test/output
go build .
```

### Validation

Generated code should:
1. Compile without errors
2. Pass `go vet`
3. Format correctly with `go fmt`
4. Include all expected fields from StructureDefinition
5. Have correct struct tags

## Troubleshooting

### Common Issues

**Issue: "resource not found"**
```
Error: resource Patient not found in StructureDefinitions
```
**Solution:** Verify the resource name matches exactly (case-sensitive). Use `-verbose` to see available resources.

**Issue: "failed to parse StructureDefinition"**
```
Error: failed to parse StructureDefinition: unexpected JSON structure
```
**Solution:** Ensure input file is valid FHIR StructureDefinitions JSON from official FHIR spec.

**Issue: Generated code doesn't compile**
```
Error: undefined: primitives.Date
```
**Solution:** Ensure `primitives` package is available in the output directory's parent. Import path should be correct.

**Issue: Missing fields in generated code**
```
Generated Patient is missing 'name' field
```
**Solution:** Check that the StructureDefinition contains the expected elements. Use `-verbose` to see parsing details.

## Performance

### Generation Speed

- **Single resource:** ~10ms
- **All 146 R5 resources:** ~2 seconds
- **All R5 types + resources:** ~5 seconds

### Output Size

- **Average resource:** ~100 lines, 3KB
- **Large resource (Patient):** ~300 lines, 10KB
- **All R5 resources:** ~60,000 lines, 2MB

## Examples

### Complete R5 Generation Workflow (Recommended)

Using mise tasks for the complete workflow:

```bash
# Download R5 schemas
mise schema:download

# Generate all R5 code (resources + types)
mise gen:r5-all

# Verify generated code compiles and passes tests
mise gen:verify
```

### Generate R5 Resources Only

```bash
# Using mise (recommended)
mise gen:r5-resources

# Or manually
cd fhir/scripts/gen
go build -o bin/fhirgen .
./bin/fhirgen -version r5 \
  -input ../../fhir_schemas/r5/profiles-resources.json \
  -output ../../r5/resources \
  -verbose
```

### Generate R5 Complex Types Only

```bash
# Using mise (recommended)
mise gen:r5-types

# Or manually
./fhir/scripts/gen/bin/fhirgen -version r5 \
  -input fhir_schemas/r5/profiles-types.json \
  -output fhir/r5/types \
  -verbose
```

### Generate Specific Resources for Testing

```bash
# Build generator first
mise gen:build

# Generate minimal set manually
./fhir/scripts/gen/bin/fhirgen -version r5 \
  -input fhir_schemas/r5/profiles-resources.json \
  -output /tmp/fhir-test \
  -resources Patient,Observation,Practitioner \
  -verbose
```

### Clean Generated Code

```bash
# Remove all generated files
mise gen:clean
```

## Future Enhancements

Planned improvements:
- [ ] Support for FHIR R4B
- [ ] Option to generate interfaces for resources
- [ ] Generate helper methods (getters/setters)
- [ ] Support for custom templates
- [ ] Generate validation methods
- [ ] Support for profiled resources
- [ ] Generate resource-specific helpers

## Contributing

When modifying the generator:

1. Understand the pipeline: Parse → Map → Build → Generate
2. Add tests for new features
3. Verify generated code compiles
4. Update this documentation
5. Run full generation to ensure no regressions

## Summary

The FHIR code generator provides:
- **Automated** type-safe Go code from FHIR spec
- **Complete** R5 support with validation tags
- **Flexible** filtering and customization
- **Fast** generation (seconds for full spec)
- **Maintainable** clear architecture and extensibility

Use it to generate FHIR resources that are fully compatible with validation, extensions, choice types, and summary mode!
