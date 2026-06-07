package dicom

import "fmt"

// ValueType is the SR content-item value type from PS3.3 C.17.3 (the VR CS value of
// Value Type (0040,A040)). It names how a content item's value field is encoded.
type ValueType uint8

const (
	ValueTypeContainer ValueType = iota // CONTAINER
	ValueTypeText                       // TEXT
	ValueTypeCode                       // CODE
	ValueTypeNum                        // NUM (measured value + units)
	ValueTypePName                      // PNAME (person name)
	ValueTypeDate                       // DATE
	ValueTypeTime                       // TIME
	ValueTypeDateTime                   // DATETIME
	ValueTypeUIDRef                     // UIDREF
	ValueTypeComposite                  // COMPOSITE (referenced SOP instance)
	ValueTypeImage                      // IMAGE (referenced image)
	ValueTypeSCoord                     // SCOORD (spatial coordinates)
	ValueTypeSCoord3D                   // SCOORD3D
	ValueTypeTCoord                     // TCOORD (temporal coordinates)
	ValueTypeWaveform                   // WAVEFORM
)

// valueTypeTerms maps each ValueType to its PS3.3 CS defined term, indexed by the
// enum value so String is a direct lookup.
var valueTypeTerms = [...]string{
	ValueTypeContainer: "CONTAINER",
	ValueTypeText:      "TEXT",
	ValueTypeCode:      "CODE",
	ValueTypeNum:       "NUM",
	ValueTypePName:     "PNAME",
	ValueTypeDate:      "DATE",
	ValueTypeTime:      "TIME",
	ValueTypeDateTime:  "DATETIME",
	ValueTypeUIDRef:    "UIDREF",
	ValueTypeComposite: "COMPOSITE",
	ValueTypeImage:     "IMAGE",
	ValueTypeSCoord:    "SCOORD",
	ValueTypeSCoord3D:  "SCOORD3D",
	ValueTypeTCoord:    "TCOORD",
	ValueTypeWaveform:  "WAVEFORM",
}

// String renders the PS3.3 CS defined term written to (0040,A040).
func (v ValueType) String() string {
	if int(v) < len(valueTypeTerms) && valueTypeTerms[v] != "" {
		return valueTypeTerms[v]
	}
	return "??"
}

// parseValueType resolves a (0040,A040) CS defined term to its ValueType. It rejects
// an unknown or empty term with a typed ValueError so a malformed SR fails closed.
func parseValueType(s string) (ValueType, error) {
	for vt, term := range valueTypeTerms {
		if term != "" && term == s {
			return ValueType(vt), nil
		}
	}
	return 0, &ValueError{Tag: TagValueType, VR: VRCS, Msg: fmt.Sprintf("unknown SR value type (%d bytes)", len(s))}
}

// RelationshipType is the SR parent-child relationship from PS3.3 C.17.3.2.4 (the VR
// CS value of Relationship Type (0040,A010)).
type RelationshipType uint8

const (
	RelationshipContains      RelationshipType = iota // CONTAINS
	RelationshipHasObsContext                         // HAS OBS CONTEXT
	RelationshipHasConceptMod                         // HAS CONCEPT MOD
	RelationshipHasProperties                         // HAS PROPERTIES
	RelationshipHasAcqContext                         // HAS ACQ CONTEXT
	RelationshipInferredFrom                          // INFERRED FROM
	RelationshipSelectedFrom                          // SELECTED FROM
)

// relationshipTerms maps each RelationshipType to its PS3.3 CS defined term.
var relationshipTerms = [...]string{
	RelationshipContains:      "CONTAINS",
	RelationshipHasObsContext: "HAS OBS CONTEXT",
	RelationshipHasConceptMod: "HAS CONCEPT MOD",
	RelationshipHasProperties: "HAS PROPERTIES",
	RelationshipHasAcqContext: "HAS ACQ CONTEXT",
	RelationshipInferredFrom:  "INFERRED FROM",
	RelationshipSelectedFrom:  "SELECTED FROM",
}

// String renders the PS3.3 CS defined term written to (0040,A010).
func (r RelationshipType) String() string {
	if int(r) < len(relationshipTerms) && relationshipTerms[r] != "" {
		return relationshipTerms[r]
	}
	return "??"
}

// parseRelationshipType resolves a (0040,A010) CS defined term to its
// RelationshipType, rejecting an unknown or empty term with a typed ValueError.
func parseRelationshipType(s string) (RelationshipType, error) {
	for rt, term := range relationshipTerms {
		if term != "" && term == s {
			return RelationshipType(rt), nil
		}
	}
	return 0, &ValueError{Tag: TagRelationshipType, VR: VRCS, Msg: fmt.Sprintf("unknown SR relationship type (%d bytes)", len(s))}
}
