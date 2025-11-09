# Design Document: FHIR Type Safety Refactoring

## Context

The current FHIR implementation uses `any` types in 620+ locations to handle polymorphic FHIR resources and choice
types. While this provides flexibility, it sacrifices type safety and creates poor developer experience.

After analyzing real-world FHIR examples in testdata/ and comparing against the FHIR R5 specification, we identified
that:
1. The base implementation (`fhir/bundle.go`) already uses correct patterns with `json.RawMessage`
2. The FHIR spec requires expanded field names for choice types (e.g., `deceasedBoolean` not `deceased`)
3. R4 resources partially implement expanded fields, but R5 reverted to `any` types
4. Users are requesting helper methods like `UnmarshalResource()` to reduce boilerplate

This design addresses all 620+ `any` type usages across 9 distinct categories.

## Goals

### Primary Goals
1. **Eliminate all `any` types** from FHIR resource structs (620+ occurrences)
2. **Match FHIR specification** - Use expanded choice field names as per spec
3. **Type safety** - Catch type errors at compile time instead of runtime
4. **Better DX** - Reduce boilerplate, improve IDE support, clearer APIs
5. **Consistency** - R5 should match base implementation patterns
6. **Performance** - Lazy deserialization with `json.RawMessage`, no overhead

### Non-Goals
1. Not changing FHIR JSON serialization format (remains spec-compliant)
2. Not adding runtime type introspection beyond existing patterns
3. Not implementing full FHIR path or FHIRPath query language
4. Not supporting FHIR versions other than R5 (R4B handled separately)

## Decisions

### Decision 1: Use json.RawMessage for Polymorphic Resources

**Context**: Bundle entries and contained resources can hold any FHIR resource type.

**Decision**: Use `json.RawMessage` instead of `*any` or `interface{}`.

**Rationale**:
- ✅ Type-safe at the JSON level (valid JSON byte slice)
- ✅ Lazy deserialization - only parse when needed
- ✅ Zero overhead - no double marshaling
- ✅ Existing pattern in `fhir/bundle.go` (proven to work)
- ✅ Easy to add helper methods for common operations

**Implementation**:
```go
type BundleEntry struct {
    Resource json.RawMessage `json:"resource,omitempty"`
}

// Helper method
func (e *BundleEntry) UnmarshalResource(v interface{}) error {
    if len(e.Resource) == 0 {
        return errors.New("resource is nil")
    }

    // Use existing factory for type detection
    resource, err := UnmarshalResource(e.Resource)
    if err != nil {
        return err
    }

    // Type assert to target
    // Or directly unmarshal if v is provided
    return json.Unmarshal(e.Resource, v)
}
```

**Alternatives Considered**:

**Alternative A: Keep `*any` with type assertion helpers**
```go
Resource *any `json:"resource,omitempty"`

func (e *BundleEntry) GetResource() (any, error) {
    // Type assertion logic
}
```
- ❌ Still requires runtime type assertions
- ❌ No compile-time safety
- ❌ Marshaling overhead (any → JSON → struct)

**Alternative B: Use generics**
```go
type BundleEntry[T Resource] struct {
    Resource *T `json:"resource,omitempty"`
}
```
- ❌ Can't mix different resource types in same bundle
- ❌ Breaks JSON serialization
- ❌ Too complex for this use case

**Alternative C: Interface-based approach**
```go
type FHIRResource interface {
    ResourceType() string
}

Resource FHIRResource `json:"resource,omitempty"`
```
- ❌ Custom marshaling required for all 202 resources
- ❌ Performance overhead
- ❌ Doesn't match FHIR JSON structure

### Decision 2: Expand Choice Type Fields

**Context**: FHIR choice types like `deceased[x]` can be one of multiple types (`deceasedBoolean`, `deceasedDateTime`).

**Decision**: Generate multiple typed fields instead of single `any` field.

**Rationale**:
- ✅ Matches FHIR JSON specification exactly
- ✅ Compile-time type checking
- ✅ Better IDE autocomplete
- ✅ Validation can enforce mutual exclusion
- ✅ Proven pattern in `fhir/CHOICE_TYPES.md` documentation

**Implementation**:
```go
type Patient struct {
    // Choice type: deceased[x]
    DeceasedBoolean  *bool                `json:"deceasedBoolean,omitempty" fhir:"choice=deceased"`
    DeceasedDateTime *primitives.DateTime `json:"deceasedDateTime,omitempty" fhir:"choice=deceased"`
}
```

**Validation**:
```go
func validateChoiceTypes(val reflect.Value) error {
    // Group fields by choice base name
    // Ensure only one field per group is non-nil
    choiceGroups := make(map[string][]reflect.Value)

    for i := 0; i < val.NumField(); i++ {
        field := val.Type().Field(i)
        if choiceTag := field.Tag.Get("fhir"); strings.Contains(choiceTag, "choice=") {
            choiceName := extractChoiceName(choiceTag)
            if !val.Field(i).IsNil() {
                choiceGroups[choiceName] = append(choiceGroups[choiceName], val.Field(i))
            }
        }
    }

    for choiceName, fields := range choiceGroups {
        if len(fields) > 1 {
            return fmt.Errorf("choice type %s has multiple values set", choiceName)
        }
    }
    return nil
}
```

**Alternatives Considered**:

**Alternative A: Union type with custom marshaling**
```go
type DeceasedChoice struct {
    Boolean  *bool
    DateTime *primitives.DateTime
}

func (d *DeceasedChoice) MarshalJSON() ([]byte, error) {
    if d.Boolean != nil {
        return json.Marshal(map[string]any{"deceasedBoolean": d.Boolean})
    }
    // ...
}
```
- ❌ Extra wrapper type adds verbosity
- ❌ Custom marshaling complexity
- ❌ Doesn't match FHIR struct patterns

**Alternative B: Keep `any` with accessor methods**
```go
Deceased *any `json:"deceased,omitempty"`

func (p *Patient) GetDeceasedBoolean() (*bool, error)
func (p *Patient) GetDeceasedDateTime() (*primitives.DateTime, error)
func (p *Patient) SetDeceasedBoolean(v bool) error
```
- ❌ Still using `any` internally
- ❌ Lots of boilerplate methods
- ❌ Runtime type assertions required

**Alternative C: Interface with type switching**
```go
type DeceasedValue interface {
    isDeceasedValue()
}

func (b *bool) isDeceasedValue() {}
func (d *primitives.DateTime) isDeceasedValue() {}
```
- ❌ Custom marshaling required
- ❌ Doesn't match JSON structure
- ❌ Overly complex for this use case

### Decision 3: Generate Tagged Unions for Complex Polymorphic Types

**Context**: Types like `UsageContext.value[x]` accept 4+ unrelated types that aren't simple primitives.

**Decision**: Generate tagged union wrapper types with flattened JSON marshaling.

**Rationale**:
- ✅ Type-safe access to each variant
- ✅ Matches FHIR JSON structure (flattened fields)
- ✅ Clear which variant is set
- ✅ Custom marshaling hides complexity

**Implementation**:
```go
type UsageContextValue struct {
    CodeableConcept *CodeableConcept `json:"-"`
    Quantity        *Quantity        `json:"-"`
    Range           *Range           `json:"-"`
    Reference       *Reference       `json:"-"`
}

func (v *UsageContextValue) MarshalJSON() ([]byte, error) {
    // Flatten to parent level with correct field name
    if v.CodeableConcept != nil {
        return json.Marshal(map[string]any{
            "valueCodeableConcept": v.CodeableConcept,
        })
    }
    if v.Quantity != nil {
        return json.Marshal(map[string]any{
            "valueQuantity": v.Quantity,
        })
    }
    // ...
    return []byte("{}"), nil
}

func (v *UsageContextValue) UnmarshalJSON(data []byte) error {
    // Detect which field is present and unmarshal accordingly
    var raw map[string]json.RawMessage
    if err := json.Unmarshal(data, &raw); err != nil {
        return err
    }

    if rawCC, ok := raw["valueCodeableConcept"]; ok {
        v.CodeableConcept = &CodeableConcept{}
        return json.Unmarshal(rawCC, v.CodeableConcept)
    }
    // ...
}

type UsageContext struct {
    Code  Coding              `json:"code"`
    Value *UsageContextValue  `json:"-"`  // Custom marshaler handles flattening
}

// Embed value marshaling in parent
func (u *UsageContext) MarshalJSON() ([]byte, error) {
    type Alias UsageContext
    data, _ := json.Marshal((*Alias)(u))

    if u.Value != nil {
        valueData, _ := u.Value.MarshalJSON()
        // Merge maps
    }
    return data, nil
}
```

**Alternatives Considered**:

**Alternative A: json.RawMessage with accessor methods**
```go
ValueRaw json.RawMessage `json:"-"`

func (u *UsageContext) GetValueCodeableConcept() (*CodeableConcept, error)
func (u *UsageContext) SetValueCodeableConcept(v *CodeableConcept) error
```
- ✅ Simpler implementation
- ❌ Still requires runtime type detection
- ❌ Less type-safe

**Alternative B: Expand all fields (like choice types)**
```go
ValueCodeableConcept *CodeableConcept `json:"valueCodeableConcept,omitempty"`
ValueQuantity        *Quantity        `json:"valueQuantity,omitempty"`
ValueRange           *Range           `json:"valueRange,omitempty"`
ValueReference       *Reference       `json:"valueReference,omitempty"`
```
- ✅ Simplest, most consistent with choice types
- ✅ No custom marshaling
- ✅ Type-safe
- ⚠️ Slightly verbose for 4+ variants

**Decision**: Use Alternative B (expanded fields) for consistency with choice types. Only use tagged unions if
>6 variants make it too verbose.

### Decision 4: Update Code Generator Instead of Manual Changes

**Context**: 620+ `any` types across 202 R5 resource files.

**Decision**: Modify code generator and regenerate all resources.

**Rationale**:
- ✅ Consistent patterns across all resources
- ✅ Easier to maintain (generator is source of truth)
- ✅ Can regenerate if FHIR spec changes
- ✅ Reduces manual error risk

**Generator Changes**:
1. `fhir/scripts/gen/parser/typemapper.go:122` - Expand choice types
2. `fhir/scripts/gen/codegen/builder.go:213` - Generate choice fields
3. Add struct tags for validation (`fhir:"choice=<base>"`)
4. Generate correct JSON field names

**Alternatives Considered**:

**Alternative A: Manual refactoring**
- ❌ Too error-prone for 620+ locations
- ❌ Hard to maintain consistency
- ❌ Can't easily regenerate

**Alternative B: AST rewriting script**
- ⚠️ Complex to write correctly
- ❌ Doesn't update generator (next gen overwrites)
- ❌ One-time use only

## Risks & Trade-offs

### Risk 1: Breaking Changes for All Users

**Risk**: Every user of FHIR R5 resources will need to update their code.

**Mitigation**:
1. Comprehensive migration guide with clear before/after examples
2. Detailed release notes documenting all breaking changes
3. Side-by-side examples showing old vs new patterns
4. No third-party dependencies exist, so coordination is not required

**Trade-off**: Short-term migration effort for long-term type safety gains. Since there are no external dependencies, the breaking change is acceptable.

### Risk 2: Code Generator Complexity

**Risk**: Generator becomes more complex, harder to maintain.

**Mitigation**:
1. Add comprehensive generator tests
2. Document generator architecture
3. Use clear, modular generator design
4. Add validation step in generator

**Trade-off**: One-time generator complexity vs 620+ manual changes.

### Risk 3: Performance Regression

**Risk**: json.RawMessage or custom marshaling might be slower.

**Mitigation**:
1. Benchmark before/after
2. json.RawMessage should be faster (lazy parsing)
3. Expanded fields avoid type assertions (faster)
4. Profile and optimize if needed

**Expected**: Same or better performance due to lazy parsing.

### Risk 4: JSON Compatibility

**Risk**: New structs might not parse old JSON correctly.

**Mitigation**:
1. Expanded fields use same JSON names as spec
2. Test with real-world examples from testdata/
3. Roundtrip tests (JSON → struct → JSON)
4. Backward compatibility tests

**Trade-off**: None - JSON format is unchanged.

## Migration Plan

### Phase 1: Preparation (Week 1)
1. Create comprehensive test suite
2. Baseline benchmarks
3. Migration guide skeleton

### Phase 2: Base Types (Week 1-2)
1. Update Contained to json.RawMessage
2. Add helper methods
3. Test and validate

### Phase 3: Bundle Consistency (Week 2)
1. Fix R5 Bundle to match base
2. Add UnmarshalResource helper
3. Update tests

### Phase 4: Generator Updates (Week 2-3)
1. Modify type mapper for choice expansion
2. Update builder for new patterns
3. Test generator

### Phase 5: Regeneration (Week 3-4)
1. Regenerate all R5 resources
2. Verify compilation
3. Run linters
4. Review generated code

### Phase 6: Validation & Testing (Week 4-5)
1. Update validation framework
2. Write comprehensive tests
3. Integration tests with real data
4. Performance benchmarks

### Phase 7: Documentation (Week 5)
1. Complete migration guide
2. Update all FHIR docs
3. Update examples
4. Create detailed release notes

### Phase 8: Release (Week 5-6)
1. Final validation
2. Tag release with breaking change version
3. Release announcement with migration guide
4. No backward compatibility maintenance required

**Total Effort: ~5-6 weeks**

## Open Questions

1. **Q: Should we support both v1 and v2 simultaneously?**
   - A: No, this will be a direct breaking release. No simultaneous v1/v2 support needed.

2. **Q: How do we handle third-party libraries depending on v1?**
   - A: No third-party libraries currently depend on v1. Direct breaking change is acceptable.

3. **Q: Should we provide an automated migration tool?**
   - A: Nice-to-have, not critical. Manual migration is straightforward with guide.

4. **Q: What about R4B resources?**
   - A: Apply same patterns, but as separate effort (tracked in different change)

5. **Q: How do we validate choice types at runtime?**
   - A: Use struct tags + reflection in validation framework (see Decision 2)

6. **Q: Performance impact of validation?**
   - A: Validation is opt-in, users can skip if not needed

## Success Metrics

1. ✅ Zero `any` types in resource structs (except factory methods)
2. ✅ All tests pass (existing + new)
3. ✅ Performance within 10% of baseline
4. ✅ Migration guide complete with examples
5. ✅ Positive user feedback on type safety
6. ✅ IDE autocomplete works for choice types