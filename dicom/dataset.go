// Package dicom provides Go implementations of DICOM data structures and operations.
//
// This is the root package containing the primary DataSet type and collection types
// for working with DICOM datasets.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html
package dicom

import (
	"fmt"
	"sort"
	"strings"

	"github.com/codeninja55/go-radx/dicom/element"
	"github.com/codeninja55/go-radx/dicom/tag"
)

// DataSet represents a collection of DICOM data elements.
//
// A DataSet stores DataElements indexed by their tags, providing dictionary-like
// access to DICOM attributes. This follows pydicom's Dataset design adapted for Go.
//
// Example usage:
//
//	// Create a new dataset
//	ds := dicom.NewDataSet()
//
//	// Add elements
//	patientName := element.NewElement(
//	    tag.New(0x0010, 0x0010),
//	    vr.PersonName,
//	    value.NewStringValue(vr.PersonName, []string{"Doe^John"}),
//	)
//	ds.Add(patientName)
//
//	// Retrieve by tag
//	elem, err := ds.Get(tag.New(0x0010, 0x0010))
//
//	// Retrieve by keyword
//	elem, err := ds.GetByKeyword("PatientName")
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_7.1
type DataSet struct {
	elements map[tag.Tag]*element.Element
}

// NewDataSet creates a new empty DICOM dataset.
//
// Example:
//
//	ds := dicom.NewDataSet()
//	fmt.Println(ds.Len())  // Output: 0
func NewDataSet() *DataSet {
	return &DataSet{
		elements: make(map[tag.Tag]*element.Element),
	}
}

// NewDataSetWithElements creates a new dataset pre-populated with elements.
//
// Returns an error if any element is nil or if duplicate tags are found.
//
// Example:
//
//	elements := []*element.Element{patientName, patientID, studyDate}
//	ds, err := dicom.NewDataSetWithElements(elements)
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewDataSetWithElements(elements []*element.Element) (*DataSet, error) {
	ds := NewDataSet()

	for _, elem := range elements {
		if elem == nil {
			return nil, fmt.Errorf("cannot add nil element")
		}

		// Check for duplicates
		if ds.Contains(elem.Tag()) {
			return nil, fmt.Errorf("duplicate tag %s in elements", elem.Tag())
		}

		if err := ds.Add(elem); err != nil {
			return nil, err
		}
	}

	return ds, nil
}

// Add inserts or replaces an element in the dataset.
//
// If an element with the same tag already exists, it will be replaced.
// Returns an error if the element is nil.
//
// Example:
//
//	elem := element.NewElement(tag.New(0x0010, 0x0010), vr.PersonName, value)
//	if err := ds.Add(elem); err != nil {
//	    log.Fatal(err)
//	}
func (ds *DataSet) Add(elem *element.Element) error {
	if elem == nil {
		return fmt.Errorf("cannot add nil element")
	}

	ds.elements[elem.Tag()] = elem
	return nil
}

// Get retrieves an element by its DICOM tag.
//
// Returns an error if the tag is not found in the dataset.
//
// Example:
//
//	elem, err := ds.Get(tag.New(0x0010, 0x0010))
//	if err != nil {
//	    log.Printf("PatientName not found: %v", err)
//	}
func (ds *DataSet) Get(t tag.Tag) (*element.Element, error) {
	elem, exists := ds.elements[t]
	if !exists {
		return nil, fmt.Errorf("element with tag %s not found", t)
	}

	return elem, nil
}

// GetByKeyword retrieves an element by its DICOM keyword.
//
// The keyword is looked up in the DICOM dictionary to find the corresponding tag.
// Returns an error if the keyword is unknown or the element is not in the dataset.
//
// Example:
//
//	elem, err := ds.GetByKeyword("PatientName")
//	if err != nil {
//	    log.Printf("Element not found: %v", err)
//	}
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part06.html#chapter_6
func (ds *DataSet) GetByKeyword(keyword string) (*element.Element, error) {
	// Find the tag for this keyword
	info, err := tag.FindByKeyword(keyword)
	if err != nil {
		return nil, fmt.Errorf("unknown keyword %q: %w", keyword, err)
	}

	return ds.Get(info.Tag)
}

// Contains checks if an element with the given tag exists in the dataset.
//
// Example:
//
//	if ds.Contains(tag.New(0x0010, 0x0010)) {
//	    fmt.Println("PatientName is present")
//	}
func (ds *DataSet) Contains(t tag.Tag) bool {
	_, exists := ds.elements[t]
	return exists
}

// Remove removes an element from the dataset by its tag.
//
// Returns an error if the tag is not found.
//
// Example:
//
//	if err := ds.Remove(tag.New(0x0010, 0x0010)); err != nil {
//	    log.Printf("Could not remove PatientName: %v", err)
//	}
func (ds *DataSet) Remove(t tag.Tag) error {
	if !ds.Contains(t) {
		return fmt.Errorf("element with tag %s not found", t)
	}

	delete(ds.elements, t)
	return nil
}

// Len returns the number of elements in the dataset.
//
// Example:
//
//	fmt.Printf("Dataset contains %d elements\n", ds.Len())
func (ds *DataSet) Len() int {
	return len(ds.elements)
}

// Elements returns all elements in the dataset sorted by tag.
//
// The returned slice is a copy and can be safely modified without affecting
// the dataset.
//
// Example:
//
//	for _, elem := range ds.Elements() {
//	    fmt.Printf("%s = %s\n", elem.Tag(), elem.Value())
//	}
func (ds *DataSet) Elements() []*element.Element {
	if len(ds.elements) == 0 {
		return []*element.Element{}
	}

	// Collect and sort by tag
	tags := ds.Tags()
	elements := make([]*element.Element, len(tags))

	for i, t := range tags {
		elements[i] = ds.elements[t]
	}

	return elements
}

// Tags returns all tags in the dataset sorted in ascending order.
//
// The returned slice is a copy and can be safely modified without affecting
// the dataset.
//
// Example:
//
//	for _, t := range ds.Tags() {
//	    elem, _ := ds.Get(t)
//	    fmt.Printf("%s: %s\n", t, elem.Name())
//	}
func (ds *DataSet) Tags() []tag.Tag {
	if len(ds.elements) == 0 {
		return []tag.Tag{}
	}

	tags := make([]tag.Tag, 0, len(ds.elements))
	for t := range ds.elements {
		tags = append(tags, t)
	}

	// Sort by tag value
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Compare(tags[j]) < 0
	})

	return tags
}

// String returns a human-readable string representation of the dataset.
//
// Format:
//
//	DataSet with N elements:
//	(GGGG,EEEE) VR [Name] = value
//	...
//
// Example:
//
//	fmt.Println(ds.String())
//	// Output:
//	// DataSet with 2 elements:
//	// (0010,0010) PN [Patient's Name] = Doe^John
//	// (0010,0020) LO [Patient ID] = 12345
func (ds *DataSet) String() string {
	var sb strings.Builder

	count := ds.Len()
	if count == 0 {
		sb.WriteString("DataSet with 0 elements")
		return sb.String()
	}

	if count == 1 {
		sb.WriteString("DataSet with 1 element:\n")
	} else {
		sb.WriteString(fmt.Sprintf("DataSet with %d elements:\n", count))
	}

	// Print elements in sorted order
	for _, elem := range ds.Elements() {
		sb.WriteString("  ")
		sb.WriteString(elem.String())
		sb.WriteString("\n")
	}

	return sb.String()
}

// Copy creates a deep copy of the dataset.
//
// The returned dataset is independent and modifications will not affect
// the original.
//
// Example:
//
//	original := dicom.NewDataSet()
//	// ... add elements ...
//	copy := original.Copy()
//	copy.Remove(tag.New(0x0010, 0x0010))  // Does not affect original
func (ds *DataSet) Copy() *DataSet {
	copied := NewDataSet()

	for t, elem := range ds.elements {
		copied.elements[t] = elem
	}

	return copied
}

// Merge merges elements from another dataset into this one.
//
// Elements with the same tag will be replaced by the other dataset's values.
//
// Example:
//
//	ds1 := dicom.NewDataSet()
//	ds2 := dicom.NewDataSet()
//	// ... populate both datasets ...
//	ds1.Merge(ds2)  // ds2's elements are merged into ds1
func (ds *DataSet) Merge(other *DataSet) error {
	if other == nil {
		return fmt.Errorf("cannot merge nil dataset")
	}

	for t, elem := range other.elements {
		ds.elements[t] = elem
	}

	return nil
}

// FileMetaInformation returns a new DataSet containing only File Meta Information elements.
//
// File Meta Information consists of all elements in Group 0x0002, which includes:
// - Transfer Syntax UID (0002,0010)
// - Media Storage SOP Class UID (0002,0002)
// - Media Storage SOP Instance UID (0002,0003)
// - Implementation Class UID (0002,0012)
// - Implementation Version Name (0002,0013)
//
// Returns nil if no File Meta Information elements are present.
//
// Example:
//
//	fileMeta := ds.FileMetaInformation()
//	if fileMeta != nil {
//	    tsElem, err := fileMeta.Get(tag.TransferSyntaxUID)
//	    if err == nil {
//	        fmt.Printf("Transfer Syntax: %s\n", tsElem.Value())
//	    }
//	}
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part10.html#sect_7.1
func (ds *DataSet) FileMetaInformation() *DataSet {
	fileMeta := NewDataSet()
	hasElements := false

	// File Meta Information is Group 0x0002
	const fileMetaGroup = 0x0002

	// Collect all elements from Group 0x0002
	for t, elem := range ds.elements {
		if t.Group == fileMetaGroup {
			fileMeta.elements[t] = elem
			hasElements = true
		}
	}

	if !hasElements {
		return nil
	}

	return fileMeta
}
