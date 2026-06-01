package dicom

// ConceptNameCode is a coded concept — one item of a Code Sequence — carrying the
// (code value, coding scheme, code meaning) triple from PS3.3 Section 8.1. It backs
// Concept Name Code Sequence (0040,A043), the CODE content-item value
// (Concept Code Sequence (0040,A168)), and the NUM Measurement Units Code Sequence
// (0040,08EA).
type ConceptNameCode struct {
	CodeValue              string // (0008,0100)
	CodingSchemeDesignator string // (0008,0102), e.g. "DCM", "SCT", "LN"
	CodeMeaning            string // (0008,0104)
}

// IsZero reports whether the code is the empty value (no code, scheme, or meaning).
func (c ConceptNameCode) IsZero() bool {
	return c.CodeValue == "" && c.CodingSchemeDesignator == "" && c.CodeMeaning == ""
}

// ReferencedSOPInstance pairs a referenced SOP Class with its SOP Instance. It is the
// single shape reused by dimse and dicomweb; SR COMPOSITE/IMAGE items reference
// instances through it via Referenced SOP Sequence (0008,1199).
type ReferencedSOPInstance struct {
	SOPClassUID    SOPClassUID
	SOPInstanceUID SOPInstanceUID
}

// codeItem builds the single Code Sequence Macro item dataset for c: the code value
// (0008,0100), coding scheme designator (0008,0102), and code meaning (0008,0104).
func codeItem(c ConceptNameCode) *DataSet {
	item := NewDataSet()
	item.SetString(TagCodeValue, c.CodeValue)
	item.SetString(TagCodingSchemeDesignator, c.CodingSchemeDesignator)
	item.SetString(TagCodeMeaning, c.CodeMeaning)
	return item
}

// readCode reads a ConceptNameCode from one Code Sequence Macro item dataset. ok is
// false when the item carries none of the three code attributes.
func readCode(item *DataSet) (ConceptNameCode, bool) {
	if item == nil {
		return ConceptNameCode{}, false
	}
	value, _ := item.GetString(TagCodeValue)
	scheme, _ := item.GetString(TagCodingSchemeDesignator)
	meaning, _ := item.GetString(TagCodeMeaning)
	c := ConceptNameCode{CodeValue: value, CodingSchemeDesignator: scheme, CodeMeaning: meaning}
	if c.IsZero() {
		return ConceptNameCode{}, false
	}
	return c, true
}

// readCodeSeq reads a ConceptNameCode from the first item of the code sequence at t.
// ok is false when the sequence is absent, empty, or its first item carries no code.
func readCodeSeq(ds *DataSet, t Tag) (ConceptNameCode, bool) {
	seq, ok := ds.GetSequence(t)
	if !ok || seq.Len() == 0 {
		return ConceptNameCode{}, false
	}
	for it := range seq.Items() {
		return readCode(it.DataSet)
	}
	return ConceptNameCode{}, false
}

// writeCodeSeq encodes c as a single-item code sequence at t.
func writeCodeSeq(ds *DataSet, t Tag, c ConceptNameCode) {
	ds.Set(Element{Tag: t, VR: VRSQ, Value: NewSequenceValue(NewSequence(codeItem(c)))})
}

// readReferencedSOPSeq reads the Referenced SOP Sequence (0008,1199) as a list of
// ReferencedSOPInstance. ok is false when the sequence is absent or empty.
func readReferencedSOPSeq(ds *DataSet) ([]ReferencedSOPInstance, bool) {
	seq, ok := ds.GetSequence(TagReferencedSOPSequence)
	if !ok || seq.Len() == 0 {
		return nil, false
	}
	refs := make([]ReferencedSOPInstance, 0, seq.Len())
	for it := range seq.Items() {
		class, _ := it.DataSet.GetString(TagReferencedSOPClassUID)
		instance, _ := it.DataSet.GetString(TagReferencedSOPInstanceUID)
		refs = append(refs, ReferencedSOPInstance{
			SOPClassUID:    SOPClassUID(class),
			SOPInstanceUID: SOPInstanceUID(instance),
		})
	}
	if len(refs) == 0 {
		return nil, false
	}
	return refs, true
}

// writeReferencedSOPSeq encodes refs as the Referenced SOP Sequence (0008,1199), one
// item per referenced instance with its SOP Class UID (0008,1150) and SOP Instance
// UID (0008,1155).
func writeReferencedSOPSeq(ds *DataSet, refs []ReferencedSOPInstance) {
	if len(refs) == 0 {
		return
	}
	items := make([]*DataSet, 0, len(refs))
	for _, ref := range refs {
		item := NewDataSet()
		item.SetString(TagReferencedSOPClassUID, string(ref.SOPClassUID))
		item.SetString(TagReferencedSOPInstanceUID, string(ref.SOPInstanceUID))
		items = append(items, item)
	}
	ds.Set(Element{Tag: TagReferencedSOPSequence, VR: VRSQ, Value: NewSequenceValue(NewSequence(items...))})
}
