package dicom

// ContentItem is one node of a DICOM Structured Report content tree (PS3.3 C.17.3).
// Each node carries a concept name (what it states or measures), a value typed by its
// ValueType, a relationship to its parent, and any nested children. Only the value
// field matching ValueType is populated; the others hold their zero value. The root
// item carries the zero RelationshipType.
//
// A ContentItem tree is plain data and is NOT safe for concurrent mutation; the same
// single-threaded ownership rule as DataSet applies (PRD §9; Codex DCM-016).
type ContentItem struct {
	ValueType    ValueType
	Relationship RelationshipType // relationship to the parent; root carries the zero value
	ConceptName  ConceptNameCode  // (0040,A043) what this item measures or states

	// Value fields; only the field matching ValueType is populated.
	Text             string                  // TEXT: (0040,A160)
	Code             ConceptNameCode         // CODE: the coded value (0040,A168)
	MeasuredValue    Decimal                 // NUM: numeric value (0040,A30A)
	MeasurementUnits ConceptNameCode         // NUM: units of measurement (0040,08EA)
	PersonName       PersonName              // PNAME: (0040,A123)
	DateTime         DT                      // DATE/TIME/DATETIME, preserved lexical form
	UID              UID                     // UIDREF: (0040,A124)
	Referenced       []ReferencedSOPInstance // COMPOSITE/IMAGE referenced instances

	Children []ContentItem // nested content (CONTAINS and other relationships)
}

// srSOPClasses is the set of SR SOP Class UIDs go-radx parses and builds as IOD-aware
// content trees: Basic Text SR, Enhanced SR, and Comprehensive SR (PS3.3; declared in
// docs/conformance/dicom.md). Other IODs are rejected by ParseSR with a typed error.
var srSOPClasses = map[UID]bool{
	"1.2.840.10008.5.1.4.1.1.88.11": true, // Basic Text SR Storage
	"1.2.840.10008.5.1.4.1.1.88.22": true, // Enhanced SR Storage
	"1.2.840.10008.5.1.4.1.1.88.33": true, // Comprehensive SR Storage
}

// isSupportedSRSOPClass reports whether u is one of the supported SR SOP Classes.
func isSupportedSRSOPClass(u UID) bool { return srSOPClasses[u] }
