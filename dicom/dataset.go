package dicom

import (
	"iter"
	"slices"
)

// Element is the atomic (Tag, VR, Value) unit of a DataSet.
type Element struct {
	Tag   Tag
	VR    VR
	Value Value
}

// DataSet is an ordered, Tag-keyed collection of Element values. It preserves
// ascending-tag order so a round-trip is byte-stable for canonical input.
//
// DataSet is NOT safe for concurrent mutation. A *DataSet may be read concurrently
// only if no goroutine mutates it; callers that share a dataset across goroutines
// must synchronise externally or pass independent Clone() copies. This is documented
// rather than locked because the hot path is single-threaded parse and write
// (PRD §9; Codex DCM-016).
type DataSet struct {
	elems map[Tag]Element
	order []Tag // sorted ascending; kept in step with elems
}

// NewDataSet returns an empty DataSet.
func NewDataSet() *DataSet {
	return &DataSet{elems: make(map[Tag]Element)}
}

// Get returns the element at t. ok is false if absent.
func (ds *DataSet) Get(t Tag) (Element, bool) {
	e, ok := ds.elems[t]
	return e, ok
}

// Set inserts or replaces the element for its tag, keeping ascending-tag order.
func (ds *DataSet) Set(e Element) {
	if _, exists := ds.elems[e.Tag]; !exists {
		i, _ := slices.BinarySearch(ds.order, e.Tag)
		ds.order = slices.Insert(ds.order, i, e.Tag)
	}
	ds.elems[e.Tag] = e
}

// Delete removes the element at t; it is not an error if absent.
func (ds *DataSet) Delete(t Tag) {
	if _, exists := ds.elems[t]; !exists {
		return
	}
	delete(ds.elems, t)
	if i, ok := slices.BinarySearch(ds.order, t); ok {
		ds.order = slices.Delete(ds.order, i, i+1)
	}
}

// Len returns the number of elements at this level (sequence items are not counted).
func (ds *DataSet) Len() int { return len(ds.order) }

// All iterates elements in ascending tag order.
func (ds *DataSet) All() iter.Seq[Element] {
	return func(yield func(Element) bool) {
		for _, t := range ds.order {
			if !yield(ds.elems[t]) {
				return
			}
		}
	}
}

// SetString looks up t's dictionary VR and inserts (or replaces) a text element
// carrying vals.
func (ds *DataSet) SetString(t Tag, vals ...string) {
	vr := dictVR(t)
	ds.Set(Element{Tag: t, VR: vr, Value: NewStrings(vr, vals...)})
}

// SetEmpty inserts (or replaces) a zero-length element at t under its dictionary VR.
func (ds *DataSet) SetEmpty(t Tag) {
	vr := dictVR(t)
	ds.Set(Element{Tag: t, VR: vr, Value: NewStrings(vr)})
}

// dictVR resolves the dictionary VR for t, defaulting to UN for unknown tags.
func dictVR(t Tag) VR {
	if info, ok := Lookup(t); ok {
		return info.VR
	}
	return VRUN
}

// GetString returns the first value of a text VR element. ok is false if t is absent
// or is not a text value.
func (ds *DataSet) GetString(t Tag) (string, bool) {
	vals, ok := ds.GetStrings(t)
	if !ok || len(vals) == 0 {
		return "", false
	}
	return vals[0], true
}

// GetStrings returns all backslash-separated values of a text VR element.
func (ds *DataSet) GetStrings(t Tag) ([]string, bool) {
	e, ok := ds.Get(t)
	if !ok {
		return nil, false
	}
	sv, ok := e.Value.(*Strings)
	if !ok {
		return nil, false
	}
	return sv.Strings(), true
}

// GetUID returns the first UI value parsed as a UID.
func (ds *DataSet) GetUID(t Tag) (UID, bool) {
	s, ok := ds.GetString(t)
	if !ok {
		return "", false
	}
	return UID(s), true
}

// GetInt returns the first integer value of an IS/SS/US/SL/UL/SV/UV element.
func (ds *DataSet) GetInt(t Tag) (int64, bool) {
	e, ok := ds.Get(t)
	if !ok {
		return 0, false
	}
	switch v := e.Value.(type) {
	case *Ints:
		ns := v.Ints()
		if len(ns) == 0 {
			return 0, false
		}
		return ns[0], true
	case *Decimals: // IS is carried as a Decimal
		ds := v.Decimals()
		if len(ds) == 0 {
			return 0, false
		}
		return ds[0].Int64()
	default:
		return 0, false
	}
}

// GetDecimal returns the first DS value.
func (ds *DataSet) GetDecimal(t Tag) (Decimal, bool) {
	e, ok := ds.Get(t)
	if !ok {
		return Decimal{}, false
	}
	dv, ok := e.Value.(*Decimals)
	if !ok {
		return Decimal{}, false
	}
	vals := dv.Decimals()
	if len(vals) == 0 {
		return Decimal{}, false
	}
	return vals[0], true
}

// GetPersonName returns the first PN value parsed into component groups.
func (ds *DataSet) GetPersonName(t Tag) (PersonName, bool) {
	s, ok := ds.GetString(t)
	if !ok {
		return PersonName{}, false
	}
	pn, err := ParsePersonName(s)
	if err != nil {
		return PersonName{}, false
	}
	return pn, true
}

// Clone returns a deep copy: every element's value slice is copied, not aliased, so
// de-identification and conversion never contaminate the source (Codex DCM-016).
func (ds *DataSet) Clone() *DataSet {
	out := NewDataSet()
	for e := range ds.All() {
		out.Set(Element{Tag: e.Tag, VR: e.VR, Value: cloneValue(e.Value)})
	}
	return out
}

// cloneValue reconstructs a value through its constructor so the copy owns its own
// backing slices. The value constructors already defensively copy their inputs.
func cloneValue(v Value) Value {
	switch t := v.(type) {
	case *Strings:
		return NewStrings(t.vr, t.vals...)
	case *Ints:
		return NewInts(t.vr, t.vals...)
	case *Floats:
		return NewFloats(t.vr, t.vals...)
	case *Decimals:
		return NewDecimals(t.vr, t.vals...)
	case *Tags:
		return NewTags(t.vals...)
	case *Bytes:
		return NewBytes(t.vr, t.b)
	default:
		return v
	}
}
