package dicom

import (
	"fmt"
	"sort"
	"sync"

	"github.com/codeninja55/go-radx/dicom/tag"
)

// DataSetCollection represents a read-optimized collection of DICOM datasets with comprehensive indexing.
//
// This collection type is optimized for fast read operations with all indexes pre-built.
// It maintains 7 indexes for O(1) lookups by:
//   - SOPInstanceUID (0008,0018) - Primary key, unique per instance
//   - SeriesInstanceUID (0020,000E) - Groups instances into series
//   - StudyInstanceUID (0020,000D) - Groups series into studies
//   - PatientID (0010,0020) - Patient identifier
//   - AccessionNumber (0008,0050) - Study identifier
//   - SOPClassUID (0008,0016) - Type of DICOM object
//   - SeriesNumber (0020,0011) - Ordered access within series
//
// Thread-safe for concurrent access.
//
// Memory overhead: ~350-400 bytes per dataset for index pointers.
//
// Example usage:
//
//	// Create collection
//	coll := dicom.NewDataSetCollection()
//
//	// Add datasets
//	for _, ds := range datasets {
//	    if err := coll.Add(ds); err != nil {
//	        log.Printf("Failed to add dataset: %v", err)
//	    }
//	}
//
//	// Query by study
//	studyDatasets := coll.GetByStudyInstanceUID("1.2.840.113619.2.55.3.1234567890")
//	fmt.Printf("Found %d datasets in study\n", len(studyDatasets))
//
//	// Query by series with ordering
//	seriesDatasets := coll.GetSeriesNumberRange(1, 10)
//	for _, ds := range seriesDatasets {
//	    // Process in series number order
//	}
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part03.html#chapter_C
type DataSetCollection struct {
	mu sync.RWMutex

	// Primary storage indexed by SOPInstanceUID
	datasets map[string]*DataSet

	// Secondary indexes for O(1) lookups
	seriesInstanceIndex  map[string][]*DataSet // SeriesInstanceUID -> datasets
	studyInstanceIndex   map[string][]*DataSet // StudyInstanceUID -> datasets
	patientIDIndex       map[string][]*DataSet // PatientID -> datasets
	accessionNumberIndex map[string][]*DataSet // AccessionNumber -> datasets
	sopClassIndex        map[string][]*DataSet // SOPClassUID -> datasets
	seriesNumberIndex    map[int][]*DataSet    // SeriesNumber -> datasets (ordered)
}

// NewDataSetCollection creates a new empty dataset collection.
//
// Example:
//
//	coll := dicom.NewDataSetCollection()
//	fmt.Println(coll.Len())  // Output: 0
func NewDataSetCollection() *DataSetCollection {
	return &DataSetCollection{
		datasets:             make(map[string]*DataSet),
		seriesInstanceIndex:  make(map[string][]*DataSet),
		studyInstanceIndex:   make(map[string][]*DataSet),
		patientIDIndex:       make(map[string][]*DataSet),
		accessionNumberIndex: make(map[string][]*DataSet),
		sopClassIndex:        make(map[string][]*DataSet),
		seriesNumberIndex:    make(map[int][]*DataSet),
	}
}

// NewDataSetCollectionWithDataSets creates a new collection pre-populated with datasets.
//
// Returns an error if any dataset is missing required UIDs or if duplicate SOPInstanceUIDs are found.
//
// Example:
//
//	datasets := []*dicom.DataSet{ds1, ds2, ds3}
//	coll, err := dicom.NewDataSetCollectionWithDataSets(datasets)
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewDataSetCollectionWithDataSets(datasets []*DataSet) (*DataSetCollection, error) {
	coll := NewDataSetCollection()

	for _, ds := range datasets {
		if err := coll.Add(ds); err != nil {
			return nil, err
		}
	}

	return coll, nil
}

// Add inserts a dataset into the collection and updates all indexes.
//
// Returns an error if:
//   - The dataset is nil
//   - Required UIDs are missing (SOPInstanceUID, SeriesInstanceUID, StudyInstanceUID, PatientID, SOPClassUID)
//   - A dataset with the same SOPInstanceUID already exists
//
// Example:
//
//	if err := coll.Add(dataset); err != nil {
//	    log.Printf("Failed to add dataset: %v", err)
//	}
func (c *DataSetCollection) Add(ds *DataSet) error {
	if ds == nil {
		return fmt.Errorf("cannot add nil dataset")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Extract required UIDs
	sopInstanceUID, err := c.extractStringValue(ds, tag.New(0x0008, 0x0018), "SOPInstanceUID")
	if err != nil {
		return err
	}

	seriesInstanceUID, err := c.extractStringValue(ds, tag.New(0x0020, 0x000E), "SeriesInstanceUID")
	if err != nil {
		return err
	}

	studyInstanceUID, err := c.extractStringValue(ds, tag.New(0x0020, 0x000D), "StudyInstanceUID")
	if err != nil {
		return err
	}

	patientID, err := c.extractStringValue(ds, tag.New(0x0010, 0x0020), "PatientID")
	if err != nil {
		return err
	}

	sopClassUID, err := c.extractStringValue(ds, tag.New(0x0008, 0x0016), "SOPClassUID")
	if err != nil {
		return err
	}

	// Check for duplicate
	if _, exists := c.datasets[sopInstanceUID]; exists {
		return fmt.Errorf("duplicate SOPInstanceUID: %s", sopInstanceUID)
	}

	// Extract optional fields
	accessionNumber, _ := c.extractOptionalStringValue(ds, tag.New(0x0008, 0x0050)) //nolint:errcheck // Optional field
	seriesNumber, _ := c.extractOptionalIntValue(ds, tag.New(0x0020, 0x0011))       //nolint:errcheck // Optional field

	// Add to primary storage
	c.datasets[sopInstanceUID] = ds

	// Update all indexes
	c.seriesInstanceIndex[seriesInstanceUID] = append(c.seriesInstanceIndex[seriesInstanceUID], ds)
	c.studyInstanceIndex[studyInstanceUID] = append(c.studyInstanceIndex[studyInstanceUID], ds)
	c.patientIDIndex[patientID] = append(c.patientIDIndex[patientID], ds)
	c.accessionNumberIndex[accessionNumber] = append(c.accessionNumberIndex[accessionNumber], ds)
	c.sopClassIndex[sopClassUID] = append(c.sopClassIndex[sopClassUID], ds)
	c.seriesNumberIndex[seriesNumber] = append(c.seriesNumberIndex[seriesNumber], ds)

	return nil
}

// GetBySOPInstanceUID retrieves a dataset by its SOPInstanceUID.
//
// Returns an error if the dataset is not found.
//
// Example:
//
//	ds, err := coll.GetBySOPInstanceUID("1.2.840.113619.2.55.3.1234567890.123")
//	if err != nil {
//	    log.Printf("Dataset not found: %v", err)
//	}
func (c *DataSetCollection) GetBySOPInstanceUID(uid string) (*DataSet, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ds, exists := c.datasets[uid]
	if !exists {
		return nil, fmt.Errorf("dataset with SOPInstanceUID %s not found", uid)
	}

	return ds, nil
}

// GetBySeriesInstanceUID retrieves all datasets in a series.
//
// Returns an empty slice if no datasets are found.
//
// Example:
//
//	datasets := coll.GetBySeriesInstanceUID("1.2.840.113619.2.55.3.1234567890")
//	fmt.Printf("Found %d datasets in series\n", len(datasets))
func (c *DataSetCollection) GetBySeriesInstanceUID(uid string) []*DataSet {
	c.mu.RLock()
	defer c.mu.RUnlock()

	datasets := c.seriesInstanceIndex[uid]
	if datasets == nil {
		return []*DataSet{}
	}

	// Return a copy to prevent external modification
	result := make([]*DataSet, len(datasets))
	copy(result, datasets)
	return result
}

// GetByStudyInstanceUID retrieves all datasets in a study.
//
// Returns an empty slice if no datasets are found.
//
// Example:
//
//	datasets := coll.GetByStudyInstanceUID("1.2.840.113619.2.55.3.1234567890")
//	fmt.Printf("Found %d datasets in study\n", len(datasets))
func (c *DataSetCollection) GetByStudyInstanceUID(uid string) []*DataSet {
	c.mu.RLock()
	defer c.mu.RUnlock()

	datasets := c.studyInstanceIndex[uid]
	if datasets == nil {
		return []*DataSet{}
	}

	// Return a copy to prevent external modification
	result := make([]*DataSet, len(datasets))
	copy(result, datasets)
	return result
}

// GetByPatientID retrieves all datasets for a patient.
//
// Returns an empty slice if no datasets are found.
//
// Example:
//
//	datasets := coll.GetByPatientID("P12345")
//	fmt.Printf("Found %d datasets for patient\n", len(datasets))
func (c *DataSetCollection) GetByPatientID(patientID string) []*DataSet {
	c.mu.RLock()
	defer c.mu.RUnlock()

	datasets := c.patientIDIndex[patientID]
	if datasets == nil {
		return []*DataSet{}
	}

	// Return a copy to prevent external modification
	result := make([]*DataSet, len(datasets))
	copy(result, datasets)
	return result
}

// GetByAccessionNumber retrieves all datasets with the given accession number.
//
// Returns an empty slice if no datasets are found.
//
// Example:
//
//	datasets := coll.GetByAccessionNumber("ACC12345")
//	fmt.Printf("Found %d datasets for accession number\n", len(datasets))
func (c *DataSetCollection) GetByAccessionNumber(accessionNumber string) []*DataSet {
	c.mu.RLock()
	defer c.mu.RUnlock()

	datasets := c.accessionNumberIndex[accessionNumber]
	if datasets == nil {
		return []*DataSet{}
	}

	// Return a copy to prevent external modification
	result := make([]*DataSet, len(datasets))
	copy(result, datasets)
	return result
}

// GetBySOPClassUID retrieves all datasets of a specific SOP Class.
//
// Returns an empty slice if no datasets are found.
//
// Example:
//
//	// Get all CT Image Storage datasets
//	datasets := coll.GetBySOPClassUID("1.2.840.10008.5.1.4.1.1.2")
//	fmt.Printf("Found %d CT images\n", len(datasets))
func (c *DataSetCollection) GetBySOPClassUID(uid string) []*DataSet {
	c.mu.RLock()
	defer c.mu.RUnlock()

	datasets := c.sopClassIndex[uid]
	if datasets == nil {
		return []*DataSet{}
	}

	// Return a copy to prevent external modification
	result := make([]*DataSet, len(datasets))
	copy(result, datasets)
	return result
}

// GetBySeriesNumber retrieves all datasets with the given series number.
//
// Returns an empty slice if no datasets are found.
//
// Example:
//
//	datasets := coll.GetBySeriesNumber(1)
//	fmt.Printf("Found %d datasets with series number 1\n", len(datasets))
func (c *DataSetCollection) GetBySeriesNumber(seriesNumber int) []*DataSet {
	c.mu.RLock()
	defer c.mu.RUnlock()

	datasets := c.seriesNumberIndex[seriesNumber]
	if datasets == nil {
		return []*DataSet{}
	}

	// Return a copy to prevent external modification
	result := make([]*DataSet, len(datasets))
	copy(result, datasets)
	return result
}

// GetSeriesNumberRange retrieves all datasets with series numbers in the given range (inclusive).
//
// The returned datasets are sorted by series number in ascending order.
// Returns an empty slice if no datasets are found or if start > end.
//
// Example:
//
//	// Get series 1 through 10
//	datasets := coll.GetSeriesNumberRange(1, 10)
//	for i, ds := range datasets {
//	    fmt.Printf("Dataset %d from series\n", i)
//	}
func (c *DataSetCollection) GetSeriesNumberRange(start, end int) []*DataSet {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if start > end {
		return []*DataSet{}
	}

	var result []*DataSet
	for num := start; num <= end; num++ {
		if datasets, exists := c.seriesNumberIndex[num]; exists {
			result = append(result, datasets...)
		}
	}

	return result
}

// Remove removes a dataset from the collection by SOPInstanceUID.
//
// Returns an error if the dataset is not found.
// Updates all indexes to remove references to the dataset.
//
// Example:
//
//	if err := coll.Remove("1.2.840.113619.2.55.3.1234567890.123"); err != nil {
//	    log.Printf("Failed to remove dataset: %v", err)
//	}
func (c *DataSetCollection) Remove(sopInstanceUID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	ds, exists := c.datasets[sopInstanceUID]
	if !exists {
		return fmt.Errorf("dataset with SOPInstanceUID %s not found", sopInstanceUID)
	}

	// Extract UIDs for index cleanup
	seriesInstanceUID, _ := c.extractStringValue(ds, tag.New(0x0020, 0x000E), "SeriesInstanceUID") //nolint:errcheck // Dataset already validated during Add
	studyInstanceUID, _ := c.extractStringValue(ds, tag.New(0x0020, 0x000D), "StudyInstanceUID")     //nolint:errcheck // Dataset already validated during Add
	patientID, _ := c.extractStringValue(ds, tag.New(0x0010, 0x0020), "PatientID")                   //nolint:errcheck // Dataset already validated during Add
	sopClassUID, _ := c.extractStringValue(ds, tag.New(0x0008, 0x0016), "SOPClassUID")               //nolint:errcheck // Dataset already validated during Add
	accessionNumber, _ := c.extractOptionalStringValue(ds, tag.New(0x0008, 0x0050))                  //nolint:errcheck // Optional field
	seriesNumber, _ := c.extractOptionalIntValue(ds, tag.New(0x0020, 0x0011))                        //nolint:errcheck // Optional field

	// Remove from primary storage
	delete(c.datasets, sopInstanceUID)

	// Remove from all indexes
	c.seriesInstanceIndex[seriesInstanceUID] = c.removeFromSlice(c.seriesInstanceIndex[seriesInstanceUID], ds)
	c.studyInstanceIndex[studyInstanceUID] = c.removeFromSlice(c.studyInstanceIndex[studyInstanceUID], ds)
	c.patientIDIndex[patientID] = c.removeFromSlice(c.patientIDIndex[patientID], ds)
	c.accessionNumberIndex[accessionNumber] = c.removeFromSlice(c.accessionNumberIndex[accessionNumber], ds)
	c.sopClassIndex[sopClassUID] = c.removeFromSlice(c.sopClassIndex[sopClassUID], ds)
	c.seriesNumberIndex[seriesNumber] = c.removeFromSlice(c.seriesNumberIndex[seriesNumber], ds)

	return nil
}

// Contains checks if a dataset with the given SOPInstanceUID exists in the collection.
//
// Example:
//
//	if coll.Contains("1.2.840.113619.2.55.3.1234567890.123") {
//	    fmt.Println("Dataset exists")
//	}
func (c *DataSetCollection) Contains(sopInstanceUID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, exists := c.datasets[sopInstanceUID]
	return exists
}

// Len returns the number of datasets in the collection.
//
// Example:
//
//	fmt.Printf("Collection contains %d datasets\n", coll.Len())
func (c *DataSetCollection) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.datasets)
}

// DataSets returns all datasets in the collection.
//
// The returned slice is a copy and can be safely modified without affecting the collection.
// The order is not guaranteed.
//
// Example:
//
//	for _, ds := range coll.DataSets() {
//	    fmt.Printf("Dataset: %s\n", ds.String())
//	}
func (c *DataSetCollection) DataSets() []*DataSet {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.datasets) == 0 {
		return []*DataSet{}
	}

	result := make([]*DataSet, 0, len(c.datasets))
	for _, ds := range c.datasets {
		result = append(result, ds)
	}

	// Sort by SOPInstanceUID for deterministic behavior
	sort.Slice(result, func(i, j int) bool {
		//nolint:errcheck // Used for sorting only, error indicates missing tag
		sopI, _ := c.extractStringValue(result[i], tag.New(0x0008, 0x0018), "SOPInstanceUID")
		//nolint:errcheck // Used for sorting only, error indicates missing tag
		sopJ, _ := c.extractStringValue(result[j], tag.New(0x0008, 0x0018), "SOPInstanceUID")
		return sopI < sopJ
	})

	return result
}

// Helper methods

// extractStringValue extracts a string value from a dataset element.
func (c *DataSetCollection) extractStringValue(ds *DataSet, t tag.Tag, name string) (string, error) {
	elem, err := ds.Get(t)
	if err != nil {
		return "", fmt.Errorf("required element %s (%s) not found: %w", name, t, err)
	}

	return elem.Value().String(), nil
}

// extractOptionalStringValue extracts an optional string value from a dataset element.
func (c *DataSetCollection) extractOptionalStringValue(ds *DataSet, t tag.Tag) (string, error) {
	elem, err := ds.Get(t)
	if err != nil {
		return "", nil // Optional field missing
	}

	return elem.Value().String(), nil
}

// extractOptionalIntValue extracts an optional int value from a dataset element.
func (c *DataSetCollection) extractOptionalIntValue(ds *DataSet, t tag.Tag) (int, error) {
	elem, err := ds.Get(t)
	if err != nil {
		return 0, nil // Optional field missing
	}

	// Parse the int from string representation
	valueStr := elem.Value().String()
	var intValue int
	_, err = fmt.Sscanf(valueStr, "%d", &intValue)
	if err != nil {
		return 0, nil // Invalid int, treat as missing
	}

	return intValue, nil
}

// removeFromSlice removes a dataset from a slice and returns the modified slice.
func (c *DataSetCollection) removeFromSlice(slice []*DataSet, ds *DataSet) []*DataSet {
	if slice == nil {
		return nil
	}

	for i, d := range slice {
		if d == ds {
			// Remove by swapping with last element and truncating
			slice[i] = slice[len(slice)-1]
			return slice[:len(slice)-1]
		}
	}

	return slice
}
