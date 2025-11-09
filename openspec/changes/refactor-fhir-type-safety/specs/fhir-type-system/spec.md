# FHIR Type System Specification

## ADDED Requirements

### Requirement: Type-Safe Contained Resources

The system SHALL use `json.RawMessage` for contained resources instead of `interface{}` or `any` types to provide
type-safe lazy deserialization.

#### Scenario: Store contained resource as JSON

- **GIVEN** a FHIR DomainResource with a contained Organization resource
- **WHEN** the resource is serialized to JSON
- **THEN** the contained resource SHALL be stored as `json.RawMessage`
- **AND** the JSON SHALL match FHIR specification format
- **AND** deserialization SHALL be lazy (on-demand)

#### Scenario: Unmarshal contained resource to typed struct

- **GIVEN** a DomainResource with contained resources
- **WHEN** `UnmarshalContainedResource(idx int)` is called
- **THEN** the system SHALL unmarshal the specified contained resource to its typed struct
- **AND** the system SHALL use the resource factory pattern to determine the correct type
- **AND** an error SHALL be returned if the index is out of bounds

#### Scenario: Add contained resource

- **GIVEN** a DomainResource and a FHIR resource to contain
- **WHEN** `AddContainedResource(resource interface{})` is called
- **THEN** the system SHALL marshal the resource to JSON
- **AND** append it to the Contained array as `json.RawMessage`
- **AND** return an error if marshaling fails

---

### Requirement: Type-Safe Bundle Entry Resources

The system SHALL use `json.RawMessage` for Bundle entry resources to match the base implementation pattern and
provide consistent lazy deserialization.

#### Scenario: Store bundle entry resource

- **GIVEN** a Bundle with an entry containing a Patient resource
- **WHEN** the bundle is serialized to JSON
- **THEN** the entry resource SHALL be stored as `json.RawMessage`
- **AND** the JSON SHALL be valid FHIR format
- **AND** the R5 implementation SHALL match the base `fhir/bundle.go` pattern

#### Scenario: Unmarshal bundle entry resource with helper method

- **GIVEN** a BundleEntry with a resource
- **WHEN** `UnmarshalResource(v interface{})` is called with a target struct pointer
- **THEN** the system SHALL unmarshal the resource JSON into the provided struct
- **AND** return an error if the resource is nil or empty
- **AND** return an error if unmarshaling fails
- **AND** the helper method SHALL reduce boilerplate code for users

#### Scenario: Type-detect bundle entry resource

- **GIVEN** a BundleEntry with an unknown resource type
- **WHEN** the resource needs to be accessed
- **THEN** the system SHALL use the `UnmarshalResource` factory to detect the type
- **AND** unmarshal to the appropriate typed struct
- **AND** preserve all resource fields during unmarshaling

---

### Requirement: Expanded Choice Type Fields

The system SHALL expand FHIR choice types (e.g., `deceased[x]`) into multiple typed fields instead of using a
single `any` type field. This matches the FHIR JSON specification and provides compile-time type safety.

#### Scenario: Generate expanded fields for Patient.deceased[x]

- **GIVEN** the FHIR R5 Patient resource definition with `deceased[x]` choice type
- **WHEN** the code generator processes the definition
- **THEN** the system SHALL generate two fields: `DeceasedBoolean` and `DeceasedDateTime`
- **AND** each field SHALL have the correct JSON tag: `deceasedBoolean` and `deceasedDateTime`
- **AND** each field SHALL have a `fhir:"choice=deceased"` struct tag for validation
- **AND** the base field name `deceased` SHALL NOT be generated

#### Scenario: Serialize choice type to FHIR JSON

- **GIVEN** a Patient with `DeceasedBoolean` set to `false`
- **WHEN** the patient is marshaled to JSON
- **THEN** the JSON SHALL contain `"deceasedBoolean": false`
- **AND** the JSON SHALL NOT contain `"deceasedDateTime"`
- **AND** the JSON SHALL NOT contain a bare `"deceased"` field

#### Scenario: Deserialize choice type from FHIR JSON

- **GIVEN** FHIR JSON with `"deceasedDateTime": "2023-10-15"`
- **WHEN** the JSON is unmarshaled to a Patient struct
- **THEN** the `DeceasedDateTime` field SHALL be populated
- **AND** the `DeceasedBoolean` field SHALL remain nil
- **AND** the value SHALL match the input JSON

#### Scenario: Access choice type with compile-time safety

- **GIVEN** a Patient resource
- **WHEN** the developer accesses the deceased information
- **THEN** the developer SHALL check `if patient.DeceasedBoolean != nil`
- **OR** check `if patient.DeceasedDateTime != nil`
- **AND** the IDE SHALL provide autocomplete for both fields
- **AND** the compiler SHALL catch type mismatches at compile time

---

### Requirement: Choice Type Validation

The system SHALL validate that only one field in a choice type group is set at runtime, enforcing FHIR mutual
exclusion constraints.

#### Scenario: Validate single choice field set

- **GIVEN** a Patient with only `DeceasedBoolean` set
- **WHEN** the resource is validated
- **THEN** validation SHALL pass
- **AND** no errors SHALL be returned

#### Scenario: Reject multiple choice fields set

- **GIVEN** a Patient with both `DeceasedBoolean` and `DeceasedDateTime` set
- **WHEN** the resource is validated
- **THEN** validation SHALL fail
- **AND** an error message SHALL indicate "choice type deceased has multiple values set"

#### Scenario: Allow zero choice fields set

- **GIVEN** a Patient with neither `DeceasedBoolean` nor `DeceasedDateTime` set
- **WHEN** the resource is validated
- **THEN** validation SHALL pass if the choice type is optional
- **AND** validation SHALL fail if the choice type is required

#### Scenario: Validate using struct tags

- **GIVEN** resource structs with `fhir:"choice=<base>"` tags
- **WHEN** the validation framework processes the struct
- **THEN** it SHALL group fields by choice base name
- **AND** use reflection to check how many fields in each group are non-nil
- **AND** enforce mutual exclusion per group

---

### Requirement: Expanded Polymorphic Value Fields

The system SHALL expand polymorphic value fields (like `UsageContext.value[x]`) into multiple typed fields for
type safety and consistency with choice type patterns.

#### Scenario: Generate expanded fields for UsageContext.value[x]

- **GIVEN** the FHIR UsageContext definition with `value[x]` accepting 4 types
- **WHEN** the code generator processes the definition
- **THEN** the system SHALL generate 4 fields: `ValueCodeableConcept`, `ValueQuantity`, `ValueRange`, `ValueReference`
- **AND** each SHALL have correct JSON tags matching FHIR spec
- **AND** each SHALL have `fhir:"choice=value"` struct tag

#### Scenario: Serialize polymorphic value to JSON

- **GIVEN** a UsageContext with `ValueQuantity` set
- **WHEN** marshaled to JSON
- **THEN** the JSON SHALL contain `"valueQuantity": {...}`
- **AND** other value fields SHALL be omitted
- **AND** the structure SHALL match FHIR specification

#### Scenario: Deserialize polymorphic value from JSON

- **GIVEN** FHIR JSON with `"valueCodeableConcept": {...}`
- **WHEN** unmarshaled to UsageContext
- **THEN** the `ValueCodeableConcept` field SHALL be populated
- **AND** other value fields SHALL remain nil

---

### Requirement: Code Generator Choice Type Expansion

The code generator SHALL automatically expand choice types into multiple typed fields during resource generation,
eliminating manual `any` type usage.

#### Scenario: Map choice type element to multiple fields

- **GIVEN** a FHIR element definition with multiple type options (e.g., `deceased[x]`)
- **WHEN** `MapElementToChoiceFields()` is called
- **THEN** the generator SHALL return multiple field definitions, one per type option
- **AND** each field SHALL have name format: `<BaseName><TypeName>` (e.g., `DeceasedBoolean`)
- **AND** each field SHALL have JSON tag format: `<baseName><TypeName>` (e.g., `deceasedBoolean`)
- **AND** each field SHALL have `fhir:"choice=<baseName>"` struct tag

#### Scenario: Handle choice type with 6+ variants

- **GIVEN** a choice type with more than 6 variants
- **WHEN** generating struct fields
- **THEN** the generator SHALL still expand all variants
- **AND** add a comment documenting the choice group
- **AND** maintain consistency with smaller choice types

#### Scenario: Update typemapper.go to avoid any types

- **GIVEN** the type mapper at `fhir/scripts/gen/parser/typemapper.go:122`
- **WHEN** it encounters multiple type codes
- **THEN** it SHALL NOT return `interface{}` or `any`
- **AND** it SHALL call `MapElementToChoiceFields()` to expand types
- **AND** maintain compatibility with primitive types

---

### Requirement: Validation Framework Integration

The validation framework SHALL support expanded choice types and enforce FHIR constraints using struct tags and
reflection.

#### Scenario: Register choice type validator

- **GIVEN** the FHIR validation framework
- **WHEN** initialized
- **THEN** it SHALL register a choice type validation function
- **AND** the function SHALL use reflection to process `fhir:"choice=<base>"` tags
- **AND** validate mutual exclusion per choice group

#### Scenario: Validate resource with choice types

- **GIVEN** a Patient resource with choice types
- **WHEN** `validator.Validate(patient)` is called
- **THEN** the system SHALL check all choice type groups
- **AND** return validation errors for any violations
- **AND** provide clear error messages identifying the choice group

#### Scenario: Performance of choice validation

- **GIVEN** a large resource with multiple choice types
- **WHEN** validated repeatedly
- **THEN** validation SHALL complete in <1ms for typical resources
- **AND** reflection overhead SHALL be minimal
- **AND** validation MAY cache reflection metadata for performance

---

### Requirement: JSON Round-trip Compatibility

The system SHALL maintain JSON round-trip compatibility, ensuring that JSON → struct → JSON produces identical
output and matches FHIR specification.

#### Scenario: Round-trip Patient with choice type

- **GIVEN** FHIR JSON: `{"resourceType": "Patient", "deceasedBoolean": false}`
- **WHEN** unmarshaled to Patient struct and marshaled back to JSON
- **THEN** the output JSON SHALL be identical to input
- **AND** field names SHALL match FHIR specification
- **AND** no data SHALL be lost

#### Scenario: Round-trip Bundle with entries

- **GIVEN** a Bundle JSON with multiple entry resources
- **WHEN** unmarshaled and marshaled back
- **THEN** all entry resources SHALL be preserved
- **AND** resource types SHALL be maintained
- **AND** the JSON SHALL be functionally equivalent

#### Scenario: Backward compatibility with v1 JSON

- **GIVEN** JSON generated by FHIR v1 structs
- **WHEN** unmarshaled to v2 structs
- **THEN** the data SHALL be correctly parsed
- **AND** choice types SHALL map to expanded fields
- **AND** no data loss SHALL occur

---

### Requirement: Developer Experience Improvements

The system SHALL provide helper methods and clear APIs to reduce boilerplate code and improve developer experience.

#### Scenario: Use BundleEntry.UnmarshalResource helper

- **GIVEN** a BundleEntry with a Patient resource
- **WHEN** a developer calls `entry.UnmarshalResource(&patient)`
- **THEN** the Patient SHALL be populated from the entry resource
- **AND** the developer SHALL NOT need to manually marshal/unmarshal
- **AND** the code SHALL be more readable and maintainable

#### Scenario: IDE autocomplete for choice types

- **GIVEN** a Patient resource variable
- **WHEN** the developer types `patient.Deceased`
- **THEN** the IDE SHALL suggest `DeceasedBoolean` and `DeceasedDateTime`
- **AND** show the correct types for each field
- **AND** provide documentation for each field

#### Scenario: Compile-time error for type mismatch

- **GIVEN** a Patient struct with expanded choice fields
- **WHEN** a developer tries to assign a string to `DeceasedBoolean`
- **THEN** the compiler SHALL produce an error
- **AND** the error SHALL be caught before runtime
- **AND** the error message SHALL be clear

---

### Requirement: Documentation and Migration Support

The system SHALL provide comprehensive documentation and migration guides to help users transition from `any`
types to expanded typed fields.

#### Scenario: Migration guide for choice types

- **GIVEN** a user with existing code using v1 FHIR structs
- **WHEN** they read the migration guide
- **THEN** the guide SHALL show before/after code examples
- **AND** explain how to update choice type access patterns
- **AND** list all breaking changes
- **AND** provide clear migration steps

#### Scenario: Updated CHOICE_TYPES.md documentation

- **GIVEN** the `fhir/CHOICE_TYPES.md` documentation file
- **WHEN** updated for v2
- **THEN** it SHALL document the expanded field pattern
- **AND** show examples of each choice type
- **AND** explain validation rules
- **AND** link to migration guide

#### Scenario: Helper method examples

- **GIVEN** the `fhir/examples/` directory
- **WHEN** updated for v2
- **THEN** examples SHALL demonstrate `UnmarshalResource` usage
- **AND** show choice type access patterns
- **AND** demonstrate contained resource operations
- **AND** include error handling best practices

---

### Requirement: Performance Characteristics

The system SHALL maintain or improve performance compared to the current `any` type implementation, particularly
for lazy deserialization scenarios.

#### Scenario: Lazy deserialization with json.RawMessage

- **GIVEN** a Bundle with 100 entries
- **WHEN** the bundle is unmarshaled
- **THEN** entry resources SHALL NOT be fully parsed until accessed
- **AND** memory usage SHALL be lower than eager deserialization
- **AND** unmarshaling SHALL be faster than with `any` types

#### Scenario: Choice type field access performance

- **GIVEN** a Patient with expanded choice fields
- **WHEN** accessing `patient.DeceasedBoolean`
- **THEN** access SHALL be direct field access (no reflection)
- **AND** performance SHALL be equivalent to any struct field access
- **AND** no type assertion overhead SHALL exist

#### Scenario: Benchmark comparison

- **GIVEN** performance benchmarks for marshal/unmarshal operations
- **WHEN** comparing v1 (any types) vs v2 (typed fields)
- **THEN** v2 SHALL be within 10% of v1 performance for marshal
- **AND** v2 SHALL be faster for unmarshal (lazy loading)
- **AND** v2 SHALL use less memory for large bundles

---

### Requirement: Consistency Across R5 Resources

The system SHALL apply type-safe patterns consistently across all 202 R5 resource types, ensuring uniform behavior
and developer experience.

#### Scenario: All resources use json.RawMessage for contained

- **GIVEN** all 202 R5 resource type definitions
- **WHEN** generated by the code generator
- **THEN** all SHALL inherit from DomainResource
- **AND** all SHALL use `Contained []json.RawMessage`
- **AND** none SHALL use `interface{}` or `any` for contained resources

#### Scenario: All choice types are expanded

- **GIVEN** resources with choice type fields
- **WHEN** generated
- **THEN** all choice types SHALL be expanded to typed fields
- **AND** all SHALL have consistent naming: `<Base><Type>`
- **AND** all SHALL have validation struct tags
- **AND** none SHALL use `any` for choice types

#### Scenario: Generator produces consistent output

- **GIVEN** the FHIR code generator
- **WHEN** run multiple times on the same input
- **THEN** generated code SHALL be identical
- **AND** formatting SHALL be consistent via gofmt
- **AND** all generated files SHALL pass golangci-lint
