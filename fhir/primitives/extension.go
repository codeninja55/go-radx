package primitives

// PrimitiveExtension represents a FHIR extension on a primitive data type.
// In FHIR, primitive types can have extensions via the "_fieldName" pattern.
// For example, if a field "active" has an extension, it appears as "_active" in JSON.
type PrimitiveExtension struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`

	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
}

// Extension represents a FHIR extension.
type Extension struct {
	// Additional extensions
	Extension []Extension `json:"extension,omitempty"`

	// Value of extension - complex types
	ValueDate     *Date     `json:"valueDate,omitempty"`
	ValueDateTime *DateTime `json:"valueDateTime,omitempty"`
	ValueTime     *Time     `json:"valueTime,omitempty"`
	ValueInstant  *Instant  `json:"valueInstant,omitempty"`

	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`

	// Identifies the meaning of the extension
	URL string `json:"url"`

	// Value of extension - primitive types
	ValueBoolean      *bool    `json:"valueBoolean,omitempty"`
	ValueInteger      *int     `json:"valueInteger,omitempty"`
	ValueString       *string  `json:"valueString,omitempty"`
	ValueDecimal      *float64 `json:"valueDecimal,omitempty"`
	ValueUri          *string  `json:"valueUri,omitempty"`
	ValueUrl          *string  `json:"valueUrl,omitempty"`
	ValueCanonical    *string  `json:"valueCanonical,omitempty"`
	ValueBase64Binary *string  `json:"valueBase64Binary,omitempty"`
	ValueCode         *string  `json:"valueCode,omitempty"`
	// ... more value types can be added as needed
}

// HasExtension returns true if the PrimitiveExtension has any extensions.
func (p *PrimitiveExtension) HasExtension() bool {
	return p != nil && len(p.Extension) > 0
}

// GetExtensionByURL returns the first extension with the given URL, or nil if not found.
func (p *PrimitiveExtension) GetExtensionByURL(url string) *Extension {
	if p == nil {
		return nil
	}
	for i := range p.Extension {
		if p.Extension[i].URL == url {
			return &p.Extension[i]
		}
	}
	return nil
}

// AddExtension adds an extension to the primitive.
func (p *PrimitiveExtension) AddExtension(ext Extension) {
	if p == nil {
		return
	}
	p.Extension = append(p.Extension, ext)
}
