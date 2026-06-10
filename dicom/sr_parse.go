package dicom

// ParseSR reads the SR content tree from ds, starting at the root content item and
// recursing through the Content Sequence (0040,A730). The root's attributes live at
// the dataset's top level: its ValueType (0040,A040) must be CONTAINER and its concept
// name comes from the Concept Name Code Sequence (0040,A043).
//
// ParseSR returns a typed *ValueError when ds is not a supported SR IOD (its SOP Class
// is not Basic Text, Enhanced, or Comprehensive SR) or the root is not a CONTAINER.
// The recursion is bounded by the package sequence-depth cap (default 64): a
// maliciously deep tree returns a *LimitExceededError rather than overflowing the
// stack. The same bounds checks the SQ reader enforces apply, since the tree is read
// from already-parsed Sequence values.
func ParseSR(ds *DataSet) (*ContentItem, error) {
	if ds == nil {
		return nil, &ValueError{Tag: TagSOPClassUID, VR: VRUI, Msg: "SR dataset is nil"}
	}
	sopClass, ok := ds.GetUID(TagSOPClassUID)
	if !ok {
		return nil, &ValueError{Tag: TagSOPClassUID, VR: VRUI, Msg: "SR dataset has no SOP Class UID"}
	}
	if !isSupportedSRSOPClass(sopClass) {
		return nil, &ValueError{
			Tag: TagSOPClassUID, VR: VRUI,
			Msg: "SOP Class " + sopClass.Name() + " is not a supported SR IOD",
		}
	}

	root, err := parseContentItem(ds, true, 0, defaultMaxSequenceDepth)
	if err != nil {
		return nil, err
	}
	if root.ValueType != ValueTypeContainer {
		return nil, &ValueError{
			Tag: TagValueType, VR: VRCS,
			Msg: "SR document root value type is " + root.ValueType.String() + ", want CONTAINER",
		}
	}
	return &root, nil
}

// parseContentItem reads one content item from the dataset item, recursing through the
// Content Sequence. isRoot suppresses the relationship read for the document root,
// which carries no relationship to a parent. depth is the current nesting level; a
// level beyond maxDepth returns a LimitExceededError before recursing further so a
// pathological tree cannot overflow the stack.
func parseContentItem(item *DataSet, isRoot bool, depth, maxDepth int) (ContentItem, error) {
	if depth > maxDepth {
		return ContentItem{}, &LimitExceededError{
			Tag:    TagContentSequence,
			Limit:  uint64(maxDepth), // #nosec G115 -- small non-negative recursion limit
			Actual: uint64(depth),    // #nosec G115 -- small non-negative recursion counter
			Kind:   "sequence-depth",
		}
	}

	var ci ContentItem

	vtStr, ok := item.GetString(TagValueType)
	if !ok {
		return ContentItem{}, &ValueError{Tag: TagValueType, VR: VRCS, Msg: "content item has no value type"}
	}
	vt, err := parseValueType(vtStr)
	if err != nil {
		return ContentItem{}, err
	}
	ci.ValueType = vt

	if !isRoot {
		relStr, ok := item.GetString(TagRelationshipType)
		if !ok {
			return ContentItem{}, &ValueError{Tag: TagRelationshipType, VR: VRCS, Msg: "child content item has no relationship type"}
		}
		rel, err := parseRelationshipType(relStr)
		if err != nil {
			return ContentItem{}, err
		}
		ci.Relationship = rel
	}

	if code, ok := readCodeSeq(item, TagConceptNameCodeSequence); ok {
		ci.ConceptName = code
	}

	if err := parseItemValue(&ci, item); err != nil {
		return ContentItem{}, err
	}

	if err := parseChildren(&ci, item, depth, maxDepth); err != nil {
		return ContentItem{}, err
	}
	return ci, nil
}

// parseItemValue extracts the value-type-specific value into ci from item.
func parseItemValue(ci *ContentItem, item *DataSet) error {
	switch ci.ValueType {
	case ValueTypeText:
		ci.Text, _ = item.GetString(TagTextValue)
	case ValueTypeCode:
		if code, ok := readCodeSeq(item, TagConceptCodeSequence); ok {
			ci.Code = code
		}
	case ValueTypeNum:
		parseNumValue(ci, item)
	case ValueTypePName:
		if pn, ok := item.GetPersonName(TagPersonName); ok {
			ci.PersonName = pn
		}
	case ValueTypeDate:
		ci.DateTime = srDateTime(item, TagDate)
	case ValueTypeTime:
		ci.DateTime = srDateTime(item, TagTime)
	case ValueTypeDateTime:
		ci.DateTime = srDateTime(item, TagDateTime)
	case ValueTypeUIDRef:
		if u, ok := item.GetUID(TagUID); ok {
			ci.UID = u
		}
	case ValueTypeComposite, ValueTypeImage:
		if refs, ok := readReferencedSOPSeq(item); ok {
			ci.Referenced = refs
		}
	}
	// CONTAINER, SCOORD, SCOORD3D, TCOORD, and WAVEFORM carry no scalar value field on
	// the ContentItem model; their content lives in children or is out of v1 scope.
	return nil
}

// parseNumValue reads the NUM Measured Value Sequence (0040,A300): its single item
// holds the Numeric Value (0040,A30A) and the Measurement Units Code Sequence
// (0040,08EA).
func parseNumValue(ci *ContentItem, item *DataSet) {
	seq, ok := item.GetSequence(TagMeasuredValueSequence)
	if !ok || seq.Len() == 0 {
		return
	}
	for mv := range seq.Items() {
		if d, ok := mv.DataSet.GetDecimal(TagNumericValue); ok {
			ci.MeasuredValue = d
		}
		if units, ok := readCodeSeq(mv.DataSet, TagMeasurementUnitsCodeSequence); ok {
			ci.MeasurementUnits = units
		}
		return
	}
}

// srDateTime reads a DATE/TIME/DATETIME content value, preserving its lexical form. A
// value that does not parse as a DT yields the zero DT rather than an error, since the
// preserved string round-trips even when not resolvable to a time.Time.
func srDateTime(item *DataSet, t Tag) DT {
	s, ok := item.GetString(t)
	if !ok {
		return DT{}
	}
	dt, err := ParseDT(s)
	if err != nil {
		return DT{lexical: s}
	}
	return dt
}

// parseChildren recurses the Content Sequence (0040,A730), appending each child item.
func parseChildren(ci *ContentItem, item *DataSet, depth, maxDepth int) error {
	seq, ok := item.GetSequence(TagContentSequence)
	if !ok {
		return nil
	}
	for child := range seq.Items() {
		childItem, err := parseContentItem(child.DataSet, false, depth+1, maxDepth)
		if err != nil {
			return err
		}
		ci.Children = append(ci.Children, childItem)
	}
	return nil
}
