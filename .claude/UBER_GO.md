# CLAUDE_GO.md

This file provides Go-specific coding guidelines based on the Uber Go Style Guide for use in this
repository.

## Core principles

- **Simplicity:** Prefer simple, clear solutions over clever or complex ones
- **Readability:** Code is read more often than written - optimize for readers
- **Consistency:** Follow existing patterns and conventions in the codebase
- **Performance:** Make informed decisions about performance trade-offs

## Interfaces and pointers

### Pointers to interfaces

- **NEVER** use pointers to interfaces (`*interface{}`)
- Interfaces already contain a pointer to the underlying data
- Pass interfaces as values

```go
// Bad
func processReader(r *io.Reader) { }

// Good
func processReader(r io.Reader) { }
```

### Verify interface compliance

- Verify interface compliance at compile time using zero-value assertions
- Place assertions near the type definition

```go
type Handler struct { }

var _ http.Handler = (*Handler)(nil)
```

### Receivers and interfaces

- Value receivers work with both pointers and values
- Pointer receivers only work with pointers or addressable values
- An interface can be satisfied by a pointer even if the method has a value receiver

```go
type Counter struct {
    value int
}

// Value receiver - works with both pointers and values
func (c Counter) Value() int {
    return c.value
}

// Pointer receiver - requires pointer or addressable value
func (c *Counter) Increment() {
    c.value++
}
```

## Concurrency and synchronization

### Mutexes

- Zero-value mutexes are valid - no need to initialize
- Use `sync.Mutex` or `sync.RWMutex` directly in structs
- **NEVER** embed mutexes in exported structs

```go
// Good
type SafeMap struct {
    mu   sync.Mutex
    data map[string]string
}
```

### Goroutines

- **NEVER** use fire-and-forget goroutines
- Always provide a way to wait for goroutines to exit
- Use `sync.WaitGroup` or context cancellation
- **NEVER** start goroutines in `init()`

```go
// Bad
func process() {
    go doWork() // fire-and-forget
}

// Good
func process(ctx context.Context) error {
    var wg sync.WaitGroup
    wg.Add(1)

    go func() {
        defer wg.Done()
        doWork(ctx)
    }()

    wg.Wait()
    return nil
}
```

### Channels

- Channel size should be one or unbuffered (zero)
- Buffered channels with size > 1 require strong justification
- Document why a specific buffer size is chosen

```go
// Preferred
ch := make(chan int)      // unbuffered
ch := make(chan int, 1)   // buffer of 1

// Requires justification
ch := make(chan int, 100) // why 100?
```

## Error handling

### Handle errors once

- Handle each error exactly once
- **NEVER** log and return an error (that's handling twice)
- Either handle (log/retry) or return, not both

```go
// Bad
if err != nil {
    log.Printf("error: %v", err)
    return err  // caller will log again
}

// Good - return for caller to handle
if err != nil {
    return fmt.Errorf("process failed: %w", err)
}

// Good - handle and don't return
if err != nil {
    log.Printf("error: %v", err)
    return nil
}
```

### Error wrapping

- Use `fmt.Errorf` with `%w` verb to wrap errors
- Provides error chain for debugging
- Enables error unwrapping with `errors.Is` and `errors.As`

```go
if err != nil {
    return fmt.Errorf("read config: %w", err)
}
```

### Error naming

- Error variables should start with `Err` or `err` prefix
- Error types should end with `Error` suffix

```go
var ErrNotFound = errors.New("not found")

type ValidationError struct {
    Field string
}
```

## Performance

### String conversions

- Prefer `strconv` over `fmt` for primitive type conversions
- `strconv` is significantly faster

```go
// Bad
s := fmt.Sprintf("%d", 123)

// Good
s := strconv.Itoa(123)
```

### Avoid string-to-byte conversions

- Avoid repeated conversions between `string` and `[]byte`
- Use consistent types throughout the call chain

```go
// Bad
func process(s string) {
    b := []byte(s)  // allocation
    // ...
}

// Good - accept []byte directly
func process(b []byte) {
    // ...
}
```

### Specify container capacity

- Pre-allocate slices and maps when size is known
- Reduces memory allocations and copies

```go
// Good
const size = 100
m := make(map[string]int, size)
s := make([]string, 0, size)
```

## Code style and formatting

### Import ordering

Group imports in three sections:
1. Standard library
2. Third-party packages
3. Local packages

```go
import (
    // Standard library
    "context"
    "fmt"
    "os"

    // Third-party
    "go.uber.org/zap"
    "github.com/pkg/errors"

    // Local
    "github.com/harrison-ai/lumina/internal/config"
)
```

### Group similar declarations

- Group related constants, variables, and types
- Use parentheses for multiple declarations

```go
// Good
const (
    StatusPending  = "pending"
    StatusActive   = "active"
    StatusComplete = "complete"
)

type (
    UserID    string
    RequestID string
)
```

### Reduce nesting

- Use early returns to reduce nesting
- Handle error cases first

```go
// Bad
if condition {
    if anotherCondition {
        // deep nesting
    }
}

// Good
if !condition {
    return err
}
if !anotherCondition {
    return err
}
// happy path at lowest indent
```

### Line length

- Keep lines under 120 characters when possible
- Break long lines at logical points

## Naming conventions

### Package names

- Use short, lowercase, single-word names
- Avoid underscores, hyphens, or mixedCaps
- Package name should match directory name

```go
// Good
package user
package http
package time
```

### Function names

- Use MixedCaps for exported functions
- Use mixedCaps for unexported functions
- Avoid redundancy with package name

```go
// Bad
func UserGetByID(id string) {}  // redundant "User"

// Good
func GetByID(id string) {}      // in package "user"
```

### Variable declarations

- Use short, descriptive names for local variables
- Longer names for package-level variables
- Single-letter names ok for very local scope (loop counters)

```go
// Good for local scope
func process(r io.Reader) {
    b := make([]byte, 1024)
    // ...
}

// Good for package level
var defaultTimeout = 30 * time.Second
```

## Resource management

### Use defer for cleanup

- Use `defer` to ensure resources are released
- Defer has minimal overhead
- Defer improves readability

```go
// Good
func readFile(path string) error {
    f, err := os.Open(path)
    if err != nil {
        return err
    }
    defer f.Close()

    // ... use file
    return nil
}
```

### Exit handling

- Only exit in `main()` or `init()`
- Prefer returning errors over calling `os.Exit` or `log.Fatal`
- Exit at most once in `main()`

```go
// Good
func main() {
    if err := run(); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
}

func run() error {
    // actual logic that returns errors
    return nil
}
```

## Anti-patterns and things to avoid

### Don't panic

- **NEVER** use panic for normal error handling
- Panic is for unrecoverable errors only
- Prefer returning errors

```go
// Bad
if err != nil {
    panic(err)
}

// Good
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

### Avoid init()

- **AVOID** `init()` functions when possible
- Makes control flow less obvious
- Can't handle errors
- Prefer explicit initialization

```go
// Bad
func init() {
    config = loadConfig()  // no error handling
}

// Good
func NewApp() (*App, error) {
    config, err := loadConfig()
    if err != nil {
        return nil, err
    }
    return &App{config: config}, nil
}
```

### Avoid mutable globals

- Mutable global state makes testing difficult
- Causes unexpected behavior in concurrent code
- Use dependency injection instead

```go
// Bad
var cache = make(map[string]string)

func Get(key string) string {
    return cache[key]  // race condition
}

// Good
type Cache struct {
    mu   sync.RWMutex
    data map[string]string
}

func (c *Cache) Get(key string) string {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.data[key]
}
```

### Avoid embedding types in public structs

- Embedded types expose internal implementation
- Makes API changes difficult
- Prefer explicit fields

```go
// Bad
type Server struct {
    http.Server  // exposes all http.Server methods
}

// Good
type Server struct {
    server *http.Server
}
```

### Don't use built-in names

- **NEVER** shadow built-in identifiers
- Makes code confusing

```go
// Bad
func process() {
    true := false    // shadows built-in
    len := 5         // shadows built-in
}

// Good
func process() {
    isValid := false
    length := 5
}
```

## Enumerations

### Start enums at one

- Start `iota` enums at 1, not 0
- Zero value should be invalid or have special meaning
- Makes zero value bugs obvious

```go
// Good
type Status int

const (
    StatusInvalid Status = iota  // 0
    StatusPending                // 1
    StatusActive                 // 2
    StatusComplete               // 3
)
```

## Time handling

### Use time package

- Use `time.Time` for instants
- Use `time.Duration` for periods
- **NEVER** use `int` or `float64` for durations

```go
// Bad
func wait(seconds int) {
    time.Sleep(time.Duration(seconds) * time.Second)
}

// Good
func wait(d time.Duration) {
    time.Sleep(d)
}

// Usage
wait(30 * time.Second)
```

## Slices and maps

### Copy at boundaries

- Copy slices and maps when crossing boundaries
- Prevents unintended mutation
- Protects internal state

```go
// Good - defensive copy on return
func (s *Store) GetItems() []Item {
    s.mu.RLock()
    defer s.mu.RUnlock()

    items := make([]Item, len(s.items))
    copy(items, s.items)
    return items
}
```

## Testing

### Table-driven tests

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

## Additional resources

- Full Uber Go Style Guide: https://github.com/uber-go/guide
- Effective Go: https://go.dev/doc/effective_go
- Go Code Review Comments: https://github.com/golang/go/wiki/CodeReviewComments
