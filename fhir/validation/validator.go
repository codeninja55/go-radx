package validation

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Error represents a validation error with context about where it occurred.
type Error struct {
	Field   string // Field path (e.g., "Patient.name[0].family")
	Message string // Human-readable error message
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

// Errors represents a collection of validation errors.
type Errors struct {
	errors []*Error
}

// Add adds a validation error.
func (e *Errors) Add(field, message string) {
	e.errors = append(e.errors, &Error{
		Field:   field,
		Message: message,
	})
}

// Addf adds a formatted validation error.
func (e *Errors) Addf(field, format string, args ...any) {
	e.Add(field, fmt.Sprintf(format, args...))
}

// HasErrors returns true if there are any validation errors.
func (e *Errors) HasErrors() bool {
	return len(e.errors) > 0
}

// Error implements the error interface.
func (e *Errors) Error() string {
	if len(e.errors) == 0 {
		return "no validation errors"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d validation error(s):\n", len(e.errors)))
	for i, err := range e.errors {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}
	return sb.String()
}

// Errors returns the list of validation errors.
func (e *Errors) List() []*Error {
	return e.errors
}

// Validator is an interface for types that can validate themselves.
type Validator interface {
	Validate() error
}

// ValidateCardinality checks if a slice meets the cardinality requirements.
// min: minimum number of elements (0 for optional)
// max: maximum number of elements (-1 for unlimited)
func ValidateCardinality(field string, count, min, max int) *Error {
	if min > 0 && count < min {
		if min == 1 {
			return &Error{
				Field:   field,
				Message: "required field is missing",
			}
		}
		return &Error{
			Field:   field,
			Message: fmt.Sprintf("requires at least %d element(s), got %d", min, count),
		}
	}

	if max >= 0 && count > max {
		return &Error{
			Field:   field,
			Message: fmt.Sprintf("requires at most %d element(s), got %d", max, count),
		}
	}

	return nil
}

// ValidateRequired checks if a required field is present.
func ValidateRequired(field string, value any) *Error {
	if value == nil {
		return &Error{
			Field:   field,
			Message: "required field is missing",
		}
	}

	// Check for empty string
	if s, ok := value.(string); ok && s == "" {
		return &Error{
			Field:   field,
			Message: "required field cannot be empty",
		}
	}

	return nil
}

// ValidateReference checks if a reference string is valid.
// FHIR references should be in the format "ResourceType/id" or absolute URLs.
func ValidateReference(field, ref string) error {
	if ref == "" {
		return nil // Empty reference is valid (will be caught by required check)
	}

	// Check for absolute URL
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return nil
	}

	// Check for relative reference format: "ResourceType/id"
	parts := strings.Split(ref, "/")
	if len(parts) < 2 {
		return &Error{
			Field:   field,
			Message: fmt.Sprintf("invalid reference format: %s (expected 'ResourceType/id')", ref),
		}
	}

	// First part should be a valid resource type name (starts with uppercase)
	resourceType := parts[0]
	if len(resourceType) == 0 || resourceType[0] < 'A' || resourceType[0] > 'Z' {
		return &Error{
			Field:   field,
			Message: fmt.Sprintf("invalid resource type in reference: %s", resourceType),
		}
	}

	return nil
}

// FHIRValidator provides comprehensive FHIR resource validation using struct tags.
type FHIRValidator struct {
	validate *validator.Validate
}

// NewFHIRValidator creates a new FHIR validator with custom validation rules.
func NewFHIRValidator() *FHIRValidator {
	v := validator.New()

	fv := &FHIRValidator{validate: v}

	// Register custom validators
	_ = v.RegisterValidation("fhir_cardinality", fv.validateCardinality)
	_ = v.RegisterValidation("fhir_enum", fv.validateEnum)
	_ = v.RegisterValidation("fhir_choice", fv.validateChoice)

	return fv
}

// Validate validates a FHIR resource using struct tags.
func (fv *FHIRValidator) Validate(resource any) error {
	if resource == nil {
		return fmt.Errorf("cannot validate nil resource")
	}

	errs := &Errors{}
	val := reflect.ValueOf(resource)

	// Validate struct fields
	fv.validateStruct(val, "", errs)

	// Validate choice type constraints
	fv.validateChoiceTypes(val, "", errs)

	if errs.HasErrors() {
		return errs
	}

	return nil
}

// validateStruct recursively validates a struct and its fields.
func (fv *FHIRValidator) validateStruct(v reflect.Value, path string, errs *Errors) {
	// Dereference pointers
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return
	}

	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		fieldPath := field.Name
		if path != "" {
			fieldPath = path + "." + field.Name
		}

		// Get FHIR struct tag
		fhirTag := field.Tag.Get("fhir")
		if fhirTag == "" {
			// No FHIR validation, but recurse into nested structs
			fv.validateField(fieldValue, fieldPath, errs)
			continue
		}

		// Parse and validate FHIR constraints
		fv.validateFHIRTag(fieldValue, fieldPath, fhirTag, errs)

		// Recurse into nested structs
		fv.validateField(fieldValue, fieldPath, errs)
	}
}

// validateField recursively validates a field value.
func (fv *FHIRValidator) validateField(v reflect.Value, path string, errs *Errors) {
	// Dereference pointers
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		fv.validateStruct(v, path, errs)
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			elemPath := fmt.Sprintf("%s[%d]", path, i)
			fv.validateField(v.Index(i), elemPath, errs)
		}
	}
}

// validateFHIRTag validates a field based on FHIR struct tag.
func (fv *FHIRValidator) validateFHIRTag(v reflect.Value, path, tag string, errs *Errors) {
	parts := strings.Split(tag, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		if part == "required" {
			fv.checkRequired(v, path, errs)
		} else if strings.HasPrefix(part, "cardinality=") {
			cardStr := strings.TrimPrefix(part, "cardinality=")
			fv.checkCardinality(v, path, cardStr, errs)
		} else if strings.HasPrefix(part, "enum=") {
			enumStr := strings.TrimPrefix(part, "enum=")
			fv.checkEnum(v, path, enumStr, errs)
		}
		// Note: choice validation is handled separately in validateChoice
		// summary is metadata only, no validation needed
	}
}

// checkRequired validates that a required field is present.
func (fv *FHIRValidator) checkRequired(v reflect.Value, path string, errs *Errors) {
	// Dereference pointer
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			errs.Add(path, "required field is missing")
			return
		}
		v = v.Elem()
	}

	// Check for zero values
	if v.IsZero() {
		errs.Add(path, "required field is missing or empty")
	}
}

// checkCardinality validates field cardinality (min..max).
func (fv *FHIRValidator) checkCardinality(v reflect.Value, path, cardinalityStr string, errs *Errors) {
	parts := strings.Split(cardinalityStr, "..")
	if len(parts) != 2 {
		errs.Addf(path, "invalid cardinality format: %s", cardinalityStr)
		return
	}

	min, err := strconv.Atoi(parts[0])
	if err != nil {
		errs.Addf(path, "invalid cardinality min: %s", parts[0])
		return
	}

	var max int
	if parts[1] == "*" {
		max = -1 // unlimited
	} else {
		max, err = strconv.Atoi(parts[1])
		if err != nil {
			errs.Addf(path, "invalid cardinality max: %s", parts[1])
			return
		}
	}

	// Get count
	var count int
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			count = 0
			break
		}
		v = v.Elem()
	}

	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		count = v.Len()
	} else if !v.IsZero() {
		count = 1
	}

	// Validate
	if min > 0 && count < min {
		if min == 1 {
			errs.Add(path, "required field is missing")
		} else {
			errs.Addf(path, "requires at least %d element(s), got %d", min, count)
		}
	}

	if max >= 0 && count > max {
		errs.Addf(path, "requires at most %d element(s), got %d", max, count)
	}
}

// checkEnum validates that a field value is in the allowed enum values.
func (fv *FHIRValidator) checkEnum(v reflect.Value, path, enumStr string, errs *Errors) {
	// Dereference pointer
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return // nil is valid for optional enums
		}
		v = v.Elem()
	}

	// Get string value
	var strValue string
	switch v.Kind() {
	case reflect.String:
		strValue = v.String()
	default:
		strValue = fmt.Sprintf("%v", v.Interface())
	}

	if strValue == "" {
		return // empty is valid for optional enums
	}

	// Parse allowed values
	allowedValues := strings.Split(enumStr, "|")
	for _, allowed := range allowedValues {
		if strings.TrimSpace(allowed) == strValue {
			return // valid
		}
	}

	errs.Addf(path, "invalid enum value '%s', must be one of: %s", strValue, enumStr)
}

// validateCardinality is a custom validator for go-playground/validator.
func (fv *FHIRValidator) validateCardinality(fl validator.FieldLevel) bool {
	// This is a placeholder for go-playground/validator integration
	// The actual validation is done in checkCardinality above
	return true
}

// validateEnum is a custom validator for go-playground/validator.
func (fv *FHIRValidator) validateEnum(fl validator.FieldLevel) bool {
	// This is a placeholder for go-playground/validator integration
	// The actual validation is done in checkEnum above
	return true
}

// validateChoice is a custom validator for choice type mutual exclusion.
func (fv *FHIRValidator) validateChoice(fl validator.FieldLevel) bool {
	// This is a placeholder - choice validation requires struct-level logic
	// which is handled separately
	return true
}

// validateChoiceTypes validates that only one field in each choice group is set.
func (fv *FHIRValidator) validateChoiceTypes(v reflect.Value, path string, errs *Errors) {
	// Dereference pointers
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return
	}

	t := v.Type()

	// Group fields by choice group
	choiceGroups := make(map[string][]string) // choiceGroup -> list of field paths with non-nil values

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		fieldPath := field.Name
		if path != "" {
			fieldPath = path + "." + field.Name
		}

		// Get FHIR struct tag
		fhirTag := field.Tag.Get("fhir")
		if fhirTag == "" {
			// Recurse into nested structs
			if field.Type.Kind() == reflect.Struct || (field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct) {
				fv.validateChoiceTypes(fieldValue, fieldPath, errs)
			}
			continue
		}

		// Parse choice group from tag
		choiceGroup := getChoiceGroup(fhirTag)
		if choiceGroup == "" {
			// Not a choice type, but recurse into nested structs
			if field.Type.Kind() == reflect.Struct || (field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct) {
				fv.validateChoiceTypes(fieldValue, fieldPath, errs)
			}
			continue
		}

		// Check if field is set (non-nil and non-zero)
		if isFieldSet(fieldValue) {
			choiceGroups[choiceGroup] = append(choiceGroups[choiceGroup], fieldPath)
		}
	}

	// Check that each choice group has at most one field set
	for choiceGroup, fields := range choiceGroups {
		if len(fields) > 1 {
			errs.Addf(strings.Join(fields, ", "),
				"choice type '%s' has multiple fields set, only one is allowed: %s",
				choiceGroup, strings.Join(fields, ", "))
		}
	}
}

// getChoiceGroup extracts the choice group name from a FHIR struct tag.
// Example: "cardinality=0..1,choice=deceased" returns "deceased"
func getChoiceGroup(tag string) string {
	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "choice=") {
			return strings.TrimPrefix(part, "choice=")
		}
	}
	return ""
}

// isFieldSet checks if a field has a non-zero/non-nil value.
func isFieldSet(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return !v.IsNil()
	case reflect.Slice, reflect.Map:
		return !v.IsNil() && v.Len() > 0
	case reflect.String:
		return v.String() != ""
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() != 0
	case reflect.Float32, reflect.Float64:
		return v.Float() != 0
	default:
		return !v.IsZero()
	}
}
