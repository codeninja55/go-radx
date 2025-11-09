# Testing Guide

This document provides comprehensive guidelines for testing the Radiology Cockpit BFF application when writing in Golang.

## Table-driven Tests

- Use table-driven tests for multiple test cases
- Makes adding test cases easy
- Reduces code duplication

```go
func TestParse(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    int
        wantErr bool
    }{
        {"valid", "123", 123, false},
        {"invalid", "abc", 0, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Parse(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("Parse() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Test-Driven Development (TDD)

When implementing new functionality, prefer Test-Driven Development:

**The TDD Cycle (Red-Green-Refactor):**

1. **Red Phase** - Write failing tests first
    - Write comprehensive test cases before implementation
    - Tests should fail because the functionality doesn't exist yet
    - This ensures tests are actually testing something

2. **Green Phase** - Implement minimum code to pass tests
    - Write the simplest code that makes tests pass
    - Don't worry about perfection yet
    - Focus on functionality first

3. **Refactor Phase** - Improve code quality
    - Clean up implementation while keeping tests green
    - Improve naming, structure, and efficiency
    - Remove duplication

**TDD Benefits:**
- Ensures testability - code is designed to be tested from the start
- Provides living documentation - tests show how code should be used
- Prevents over-engineering - only write code that's needed
- Catches regressions early - tests verify behavior doesn't break
- Increases confidence - comprehensive test coverage from day one

**Example TDD Workflow:**

```go
// 1. RED: Write failing test first
func TestFind(t *testing.T) {
    tag := tag.New(0x0008, 0x0005)
    info, err := tag.Find(tag)
    require.NoError(t, err)
    assert.Equal(t, "SpecificCharacterSet", info.Keyword)
}
// Run test: FAIL - tag.Find undefined

// 2. GREEN: Implement minimum code to pass
func Find(t Tag) (Info, error) {
    info, ok := TagDict[t]
    if !ok {
        return Info{}, fmt.Errorf("tag not found")
    }
    return info, nil
}
// Run test: PASS

// 3. REFACTOR: Improve while keeping tests green
func Find(t Tag) (Info, error) {
    // Add special case handling
    info, ok := TagDict[t]
    if !ok {
        if t.Group%2 == 0 && t.Element == 0x0000 {
            return GenericGroupLengthInfo(t), nil
        }
        return Info{}, fmt.Errorf("tag %s not found", t.String())
    }
    return info, nil
}
// Run test: PASS
```

**When to Use TDD:**
- Implementing new features or packages
- Adding complex business logic
- Building APIs or public interfaces
- Fixing bugs (write test that reproduces bug, then fix)

**When TDD May Not Be Needed:**
- Simple utility functions with obvious behavior
- Exploratory prototyping or spike solutions
- Trivial getters/setters
