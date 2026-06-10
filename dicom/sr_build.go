package dicom

// BuildSR encodes root into ds: the root content item's attributes are written at the
// dataset's top level (ValueType, Concept Name Code Sequence, and its value) and its
// children are encoded recursively into the Content Sequence (0040,A730). Each child
// node also carries its Relationship Type (0040,A010); the root, having no parent,
// writes none.
//
// BuildSR returns a typed *ValueError when root is nil, ds is nil, or root is not a
// CONTAINER. The recursion is bounded by the package sequence-depth cap (default 64):
// a tree deeper than the cap returns a *LimitExceededError rather than emitting an
// unbounded nest.
func BuildSR(ds *DataSet, root *ContentItem) error {
	if ds == nil {
		return &ValueError{Tag: TagContentSequence, VR: VRSQ, Msg: "BuildSR dataset is nil"}
	}
	if root == nil {
		return &ValueError{Tag: TagValueType, VR: VRCS, Msg: "BuildSR root content item is nil"}
	}
	if root.ValueType != ValueTypeContainer {
		return &ValueError{
			Tag: TagValueType, VR: VRCS,
			Msg: "SR document root value type is " + root.ValueType.String() + ", want CONTAINER",
		}
	}
	return encodeContentItem(ds, root, true, 0, defaultMaxSequenceDepth)
}

// encodeContentItem writes one content item's attributes into item and recurses into
// the Content Sequence. isRoot suppresses the Relationship Type write for the document
// root. depth bounds the recursion against maxDepth.
func encodeContentItem(item *DataSet, ci *ContentItem, isRoot bool, depth, maxDepth int) error {
	if depth > maxDepth {
		return &LimitExceededError{
			Tag:    TagContentSequence,
			Limit:  uint64(maxDepth), // #nosec G115 -- small non-negative recursion limit
			Actual: uint64(depth),    // #nosec G115 -- small non-negative recursion counter
			Kind:   "sequence-depth",
		}
	}

	item.SetString(TagValueType, ci.ValueType.String())
	if !isRoot {
		item.SetString(TagRelationshipType, ci.Relationship.String())
	}
	if !ci.ConceptName.IsZero() {
		writeCodeSeq(item, TagConceptNameCodeSequence, ci.ConceptName)
	}

	encodeItemValue(item, ci)

	return encodeChildren(item, ci, depth, maxDepth)
}

// encodeItemValue writes the value-type-specific attribute(s) for ci.
func encodeItemValue(item *DataSet, ci *ContentItem) {
	switch ci.ValueType {
	case ValueTypeText:
		item.SetString(TagTextValue, ci.Text)
	case ValueTypeCode:
		if !ci.Code.IsZero() {
			writeCodeSeq(item, TagConceptCodeSequence, ci.Code)
		}
	case ValueTypeNum:
		encodeNumValue(item, ci)
	case ValueTypePName:
		item.SetString(TagPersonName, ci.PersonName.String())
	case ValueTypeDate:
		item.SetString(TagDate, ci.DateTime.String())
	case ValueTypeTime:
		item.SetString(TagTime, ci.DateTime.String())
	case ValueTypeDateTime:
		item.SetString(TagDateTime, ci.DateTime.String())
	case ValueTypeUIDRef:
		item.SetString(TagUID, string(ci.UID))
	case ValueTypeComposite, ValueTypeImage:
		writeReferencedSOPSeq(item, ci.Referenced)
	}
	// CONTAINER and the spatial/temporal/waveform types carry no scalar value field.
}

// encodeNumValue writes the NUM Measured Value Sequence (0040,A300): a single item
// holding the Numeric Value (0040,A30A) and the Measurement Units Code Sequence
// (0040,08EA).
func encodeNumValue(item *DataSet, ci *ContentItem) {
	measured := NewDataSet()
	measured.Set(Element{Tag: TagNumericValue, VR: VRDS, Value: NewDecimals(VRDS, ci.MeasuredValue)})
	if !ci.MeasurementUnits.IsZero() {
		writeCodeSeq(measured, TagMeasurementUnitsCodeSequence, ci.MeasurementUnits)
	}
	item.Set(Element{Tag: TagMeasuredValueSequence, VR: VRSQ, Value: NewSequenceValue(NewSequence(measured))})
}

// encodeChildren writes ci's children into the Content Sequence (0040,A730), one item
// per child, recursing through encodeContentItem.
func encodeChildren(item *DataSet, ci *ContentItem, depth, maxDepth int) error {
	if len(ci.Children) == 0 {
		return nil
	}
	items := make([]*DataSet, 0, len(ci.Children))
	for i := range ci.Children {
		childDS := NewDataSet()
		if err := encodeContentItem(childDS, &ci.Children[i], false, depth+1, maxDepth); err != nil {
			return err
		}
		items = append(items, childDS)
	}
	item.Set(Element{Tag: TagContentSequence, VR: VRSQ, Value: NewSequenceValue(NewSequence(items...))})
	return nil
}
