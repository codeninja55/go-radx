package fhir

import (
	"encoding/json"
	"reflect"
	"strings"
)

// SummaryMode indicates whether to include only summary elements.
type SummaryMode int

const (
	// SummaryModeAll includes all fields (default JSON behavior).
	SummaryModeAll SummaryMode = iota

	// SummaryModeTrue includes only fields marked as summary elements.
	SummaryModeTrue

	// SummaryModeFalse includes all fields except summary elements.
	SummaryModeFalse

	// SummaryModeText includes only the text element and minimal metadata.
	SummaryModeText

	// SummaryModeData includes all fields except the text element.
	SummaryModeData
)

// MarshalSummaryJSON marshals a resource to JSON including only summary elements.
// This is equivalent to the FHIR _summary=true parameter.
func MarshalSummaryJSON(resource interface{}) ([]byte, error) {
	return MarshalWithSummaryMode(resource, SummaryModeTrue)
}

// MarshalWithSummaryMode marshals a resource to JSON with the specified summary mode.
func MarshalWithSummaryMode(resource interface{}, mode SummaryMode) ([]byte, error) {
	if mode == SummaryModeAll {
		// No filtering needed
		return json.Marshal(resource)
	}

	// Create a filtered representation
	filtered := filterBySummary(reflect.ValueOf(resource), mode)
	return json.Marshal(filtered)
}

// filterBySummary recursively filters a struct based on summary mode.
func filterBySummary(v reflect.Value, mode SummaryMode) interface{} {
	// Dereference pointers
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		return filterStruct(v, mode)
	case reflect.Slice:
		return filterSlice(v, mode)
	case reflect.Map:
		return filterMap(v, mode)
	default:
		return v.Interface()
	}
}

// filterStruct filters a struct based on summary mode.
func filterStruct(v reflect.Value, mode SummaryMode) map[string]interface{} {
	result := make(map[string]interface{})
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Handle embedded fields
		if field.Anonymous {
			// Recursively include embedded struct fields
			embedded := filterBySummary(fieldValue, mode)
			if embeddedMap, ok := embedded.(map[string]interface{}); ok {
				for k, v := range embeddedMap {
					result[k] = v
				}
			}
			continue
		}

		// Get JSON tag
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Parse JSON tag
		jsonName := strings.Split(jsonTag, ",")[0]
		if jsonName == "" {
			jsonName = field.Name
		}

		// Check if field should be included based on summary mode
		if !shouldIncludeField(field, mode) {
			continue
		}

		// Get field value, recursively filter if needed
		filteredValue := filterBySummary(fieldValue, mode)

		// Skip zero values with omitempty
		if strings.Contains(jsonTag, "omitempty") && isZeroValue(fieldValue) {
			continue
		}

		result[jsonName] = filteredValue
	}

	return result
}

// filterSlice filters a slice based on summary mode.
func filterSlice(v reflect.Value, mode SummaryMode) []interface{} {
	if v.IsNil() {
		return nil
	}

	result := make([]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		result[i] = filterBySummary(v.Index(i), mode)
	}

	return result
}

// filterMap filters a map based on summary mode.
func filterMap(v reflect.Value, mode SummaryMode) map[string]interface{} {
	if v.IsNil() {
		return nil
	}

	result := make(map[string]interface{})
	iter := v.MapRange()
	for iter.Next() {
		key := iter.Key().String()
		value := filterBySummary(iter.Value(), mode)
		result[key] = value
	}

	return result
}

// shouldIncludeField determines if a field should be included based on summary mode.
func shouldIncludeField(field reflect.StructField, mode SummaryMode) bool {
	fhirTag := field.Tag.Get("fhir")
	isSummary := strings.Contains(fhirTag, "summary")
	isText := field.Name == "Text"

	switch mode {
	case SummaryModeTrue:
		// Include only summary fields + always include resourceType and id
		return isSummary || field.Name == "ResourceType" || field.Name == "ID" || field.Name == "Meta"

	case SummaryModeFalse:
		// Include everything except summary fields
		return !isSummary

	case SummaryModeText:
		// Include only text + minimal metadata (resourceType, id, meta)
		return isText || field.Name == "ResourceType" || field.Name == "ID" || field.Name == "Meta"

	case SummaryModeData:
		// Include everything except text
		return !isText

	default:
		return true
	}
}

// isZeroValue checks if a reflect.Value is a zero value.
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map:
		return v.IsNil() || v.Len() == 0
	case reflect.String:
		return v.String() == ""
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	default:
		return v.IsZero()
	}
}

// GetSummaryFields returns a list of field names that are marked as summary elements.
func GetSummaryFields(resource interface{}) []string {
	var summaryFields []string

	v := reflect.ValueOf(resource)
	// Dereference pointer
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		fhirTag := field.Tag.Get("fhir")
		if strings.Contains(fhirTag, "summary") {
			jsonTag := field.Tag.Get("json")
			jsonName := strings.Split(jsonTag, ",")[0]
			if jsonName != "" && jsonName != "-" {
				summaryFields = append(summaryFields, jsonName)
			}
		}
	}

	return summaryFields
}
