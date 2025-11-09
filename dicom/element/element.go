// Package element provides DICOM data element structures and operations.
//
// A DICOM Data Element consists of a tag, VR (Value Representation), and value.
// This implementation follows pydicom's DataElement design adapted for Go idioms.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_7.1
package element

import (
	"fmt"
	"strings"

	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/value"
	"github.com/codeninja55/go-radx/dicom/vr"
)

// Element represents a DICOM data element.
//
// A Data Element is composed of:
//   - Tag: Unique identifier (group, element)
//   - VR: Value Representation (data type)
//   - Value: The actual data
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_7.1
type Element struct {
	tag   tag.Tag
	vr    vr.VR
	value value.Value
}

// NewElement creates a new DICOM data element.
//
// Parameters:
//   - t: DICOM tag (group, element)
//   - v: Value Representation
//   - val: Element value (must match VR type)
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_7.1
func NewElement(t tag.Tag, v vr.VR, val value.Value) (*Element, error) {
	if val == nil {
		return nil, fmt.Errorf("value cannot be nil")
	}

	// Verify VR matches the value's VR
	if val.VR() != v {
		return nil, fmt.Errorf("value VR %s does not match element VR %s", val.VR().String(), v.String())
	}

	return &Element{
		tag:   t,
		vr:    v,
		value: val,
	}, nil
}

// Tag returns the DICOM tag of this element.
// Similar to pydicom's DataElement.tag property.
func (e *Element) Tag() tag.Tag {
	return e.tag
}

// VR returns the Value Representation of this element.
// Similar to pydicom's DataElement.VR property.
func (e *Element) VR() vr.VR {
	return e.vr
}

// Value returns the value of this element.
// Similar to pydicom's DataElement.value property.
func (e *Element) Value() value.Value {
	return e.value
}

// Name returns the human-readable name of this element from the DICOM dictionary.
// Returns an empty string if the tag is not found (e.g., private tags).
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part06.html#chapter_6
func (e *Element) Name() string {
	info, err := tag.Find(e.tag)
	if err != nil {
		return "" // Unknown or private tag
	}
	return info.Name
}

// Keyword returns the keyword identifier of this element from the DICOM dictionary.
// Returns an empty string if the tag is not found (e.g., private tags).
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part06.html#chapter_6
func (e *Element) Keyword() string {
	info, err := tag.Find(e.tag)
	if err != nil {
		return "" // Unknown or private tag
	}
	return info.Keyword
}

// ValueMultiplicity returns the Value Multiplicity (number of values) as a string.
//
// For multivalued elements (like arrays), this returns the count.
// For single-valued elements, this returns "1".
// For empty elements, this returns "0".
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.4
func (e *Element) ValueMultiplicity() string {
	// Count values based on type
	switch v := e.value.(type) {
	case *value.StringValue:
		return fmt.Sprintf("%d", len(v.Strings()))
	case *value.IntValue:
		return fmt.Sprintf("%d", len(v.Ints()))
	case *value.FloatValue:
		return fmt.Sprintf("%d", len(v.Floats()))
	case *value.BytesValue:
		// Bytes are typically treated as a single value
		if len(v.Bytes()) == 0 {
			return "0"
		}
		return "1"
	default:
		return "1"
	}
}

// String returns a human-readable string representation of the element.
//
// Format: (GGGG,EEEE) VR [Name] = value
// Example: (0010,0010) PN [Patient's Name] = Doe^John
//
// For unknown tags, the name is omitted.
// Long values may be truncated for readability.
func (e *Element) String() string {
	var sb strings.Builder

	// Tag: (GGGG,EEEE)
	sb.WriteString(e.tag.String())
	sb.WriteString(" ")

	// VR
	sb.WriteString(e.vr.String())
	sb.WriteString(" ")

	// Name from a dictionary (if available)
	name := e.Name()
	if name != "" {
		sb.WriteString("[")
		sb.WriteString(name)
		sb.WriteString("] ")
	}

	// Value
	sb.WriteString("= ")
	valueStr := e.value.String()

	// Truncate very long values for display
	const maxValueLen = 80
	if len(valueStr) > maxValueLen {
		valueStr = valueStr[:maxValueLen] + "..."
	}

	sb.WriteString(valueStr)

	return sb.String()
}

// SetValue updates the value of this element.
//
// The new value must have the same VR as the element.
// Returns an error if the VR doesn't match or if the value is nil.
//
// Example:
//
//	elem, _ := ds.Get(tag.PatientName)
//	newValue := value.NewStringValue(vr.PersonName, []string{"Smith^Jane"})
//	if err := elem.SetValue(newValue); err != nil {
//	    log.Fatal(err)
//	}
func (e *Element) SetValue(val value.Value) error {
	if val == nil {
		return fmt.Errorf("value cannot be nil")
	}

	// Verify VR matches the value's VR
	if val.VR() != e.vr {
		return fmt.Errorf("value VR %s does not match element VR %s", val.VR().String(), e.vr.String())
	}

	e.value = val
	return nil
}

// Equals returns true if this element equals another element.
//
// Elements are equal if they have the same tag, VR, and value.
func (e *Element) Equals(other *Element) bool {
	if other == nil {
		return false
	}

	// Compare tags
	if !e.tag.Equals(other.tag) {
		return false
	}

	// Compare VRs
	if e.vr != other.vr {
		return false
	}

	// Compare values using Value.Equals()
	return e.value.Equals(other.value)
}
