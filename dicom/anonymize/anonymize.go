// Package anonymize implements DICOM PS3.15 compliant de-identification.
package anonymize

import (
	"fmt"
	"strings"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicom/element"
	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/uid"
	"github.com/codeninja55/go-radx/dicom/value"
	"github.com/codeninja55/go-radx/dicom/vr"
)

// Profile represents a DICOM PS3.15 de-identification profile.
type Profile int

const (
	// ProfileBasic is the Basic Application Level Confidentiality Profile (PS3.15 E.1).
	ProfileBasic Profile = iota

	// ProfileClean includes Basic profile with Clean Pixel Data and Clean Descriptors options.
	ProfileClean

	// ProfileRetainUIDs includes Basic profile but retains UIDs for longitudinal studies.
	ProfileRetainUIDs

	// ProfileRetainDeviceIdentity includes Basic profile but retains device/institution information.
	ProfileRetainDeviceIdentity

	// ProfileCustom allows full customization of anonymization actions.
	ProfileCustom
)

// Action represents the action to take for a DICOM attribute during anonymization.
//
// These actions follow DICOM PS3.15 Table E.1-1 notation.
type Action int

const (
	// ActionKeep preserves the attribute unchanged (K).
	ActionKeep Action = iota

	// ActionRemove deletes the attribute from the dataset (X).
	ActionRemove

	// ActionEmpty replaces with zero-length value or removes (Z).
	ActionEmpty

	// ActionDummy replaces with a dummy value maintaining VR validity (D).
	ActionDummy

	// ActionClean replaces with values of similar meaning without identification (C).
	ActionClean

	// ActionUID replaces UIDs with newly generated values (U).
	ActionUID

	// ActionEncrypt encrypts the value (for research use with key management).
	ActionEncrypt

	// ActionHash replaces with a one-way hash for consistency without identification.
	ActionHash

	// ActionCallback uses a custom callback function for the attribute.
	ActionCallback
)

// Options configures anonymization behavior beyond the base profile.
type Options struct {
	// RetainUIDs preserves original UIDs (for longitudinal studies).
	RetainUIDs bool

	// RetainDeviceIdentity preserves device and institution information.
	RetainDeviceIdentity bool

	// RetainPatientCharacteristics preserves age, sex, size, weight.
	RetainPatientCharacteristics bool

	// RetainLongitudinalTemporalInfo preserves dates/times with offset.
	RetainLongitudinalTemporalInfo bool

	// DateOffset is the offset to apply to dates when RetainLongitudinalTemporalInfo is true.
	DateOffset time.Duration

	// CleanPixelData removes burned-in annotations from pixel data.
	CleanPixelData bool

	// CleanDescriptors removes identifying information from text fields.
	CleanDescriptors bool

	// RemovePrivateTags removes all private tags.
	RemovePrivateTags bool

	// RemoveOverlays removes overlay planes (60xx groups).
	RemoveOverlays bool

	// RemoveCurves removes curve data (50xx groups).
	RemoveCurves bool
}

// Config contains the complete configuration for an Anonymizer.
type Config struct {
	// Profile is the base de-identification profile to use.
	Profile Profile

	// Options provides additional configuration.
	Options Options

	// PatientName is the replacement value for patient name.
	PatientName string

	// PatientID is the replacement value for patient ID.
	PatientID string

	// InstitutionName is the replacement value for institution name.
	InstitutionName string

	// CustomActions allows overriding actions for specific tags.
	CustomActions map[tag.Tag]Action

	// Callbacks provides custom functions for specific tags when using ActionCallback.
	Callbacks map[tag.Tag]func(*element.Element) (*element.Element, error)
}

// Anonymizer performs DICOM dataset de-identification.
type Anonymizer struct {
	config  Config
	actions map[tag.Tag]Action
}

// NewAnonymizer creates an anonymizer with the specified profile.
//
// Example:
//
//	anonymizer := anonymize.NewAnonymizer(anonymize.ProfileBasic)
func NewAnonymizer(profile Profile) *Anonymizer {
	config := Config{
		Profile:     profile,
		PatientName: "ANONYMOUS",
		PatientID:   fmt.Sprintf("ANON%d", time.Now().Unix()),
		Options:     defaultOptionsForProfile(profile),
	}
	return NewAnonymizerWithConfig(config)
}

// NewAnonymizerWithConfig creates an anonymizer with custom configuration.
//
// Example:
//
//	config := anonymize.Config{
//	    Profile: anonymize.ProfileBasic,
//	    Options: anonymize.Options{
//	        RetainUIDs: true,
//	        CleanDescriptors: true,
//	    },
//	    PatientName: "STUDY_001",
//	}
//	anonymizer := anonymize.NewAnonymizerWithConfig(config)
func NewAnonymizerWithConfig(config Config) *Anonymizer {
	a := &Anonymizer{
		config:  config,
		actions: make(map[tag.Tag]Action),
	}

	// Initialize actions based on profile
	a.initializeActions()

	// Apply custom actions
	for t, action := range config.CustomActions {
		a.actions[t] = action
	}

	return a
}

// Anonymize performs de-identification on a DICOM dataset.
//
// Returns a new anonymized dataset. The original dataset is not modified.
//
// Example:
//
//	anonymizedDS, err := anonymizer.Anonymize(originalDS)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (a *Anonymizer) Anonymize(ds *dicom.DataSet) (*dicom.DataSet, error) {
	// Create a copy of the dataset
	newDS, err := a.copyDataSet(ds)
	if err != nil {
		return nil, fmt.Errorf("failed to copy dataset: %w", err)
	}

	// Apply profile actions to each element
	err = newDS.WalkModify(func(elem *element.Element) (bool, error) {
		action, ok := a.actions[elem.Tag()]
		if !ok {
			// Default action for unspecified tags
			if isPrivateTag(elem.Tag()) && a.config.Options.RemovePrivateTags {
				return false, dicom.ErrRemoveElement
			}
			// Keep by default
			return false, nil
		}

		return a.applyAction(elem, action)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to apply anonymization: %w", err)
	}

	// Remove overlays if configured
	if a.config.Options.RemoveOverlays {
		if err := newDS.RemoveGroupTags(0x6000); err != nil {
			return nil, fmt.Errorf("failed to remove overlays: %w", err)
		}
	}

	// Remove curves if configured
	if a.config.Options.RemoveCurves {
		if err := newDS.RemoveGroupTags(0x5000); err != nil {
			return nil, fmt.Errorf("failed to remove curves: %w", err)
		}
	}

	// Generate new UIDs if needed
	if !a.config.Options.RetainUIDs {
		if err := newDS.GenerateNewUIDs(); err != nil {
			return nil, fmt.Errorf("failed to generate new UIDs: %w", err)
		}
	}

	return newDS, nil
}

// applyAction applies the specified action to an element.
func (a *Anonymizer) applyAction(elem *element.Element, action Action) (bool, error) {
	switch action {
	case ActionKeep:
		return false, nil

	case ActionRemove:
		return false, dicom.ErrRemoveElement

	case ActionEmpty:
		return a.replaceWithEmpty(elem)

	case ActionDummy:
		return a.replaceWithDummy(elem)

	case ActionClean:
		return a.cleanElement(elem)

	case ActionUID:
		return a.replaceUID(elem)

	case ActionHash:
		return a.hashElement(elem)

	case ActionCallback:
		callback, ok := a.config.Callbacks[elem.Tag()]
		if !ok {
			return false, fmt.Errorf("no callback defined for tag %s", elem.Tag())
		}
		newElem, err := callback(elem)
		if err != nil {
			return false, err
		}
		if newElem == nil {
			return false, dicom.ErrRemoveElement
		}
		return true, elem.SetValue(newElem.Value())

	default:
		return false, nil
	}
}

// replaceWithEmpty replaces the element value with an empty value.
func (a *Anonymizer) replaceWithEmpty(elem *element.Element) (bool, error) {
	var val value.Value
	var err error

	switch elem.VR() {
	case vr.PersonName, vr.LongString, vr.ShortString, vr.UnlimitedText,
		vr.ShortText, vr.LongText, vr.CodeString:
		val, err = value.NewStringValue(elem.VR(), []string{""})
	case vr.Date, vr.Time, vr.DateTime:
		val, err = value.NewStringValue(elem.VR(), []string{""})
	case vr.AgeString:
		val, err = value.NewStringValue(vr.AgeString, []string{""})
	case vr.IntegerString:
		val, err = value.NewIntValue(vr.IntegerString, []int64{})
	case vr.DecimalString:
		val, err = value.NewFloatValue(vr.DecimalString, []float64{})
	default:
		// For other VRs, use empty bytes
		val, err = value.NewBytesValue(elem.VR(), []byte{})
	}

	if err != nil {
		return false, fmt.Errorf("failed to create empty value: %w", err)
	}

	return true, elem.SetValue(val)
}

// replaceWithDummy replaces the element value with a dummy value.
func (a *Anonymizer) replaceWithDummy(elem *element.Element) (bool, error) {
	var val value.Value
	var err error

	// Special handling for specific tags
	switch elem.Tag() {
	case tag.PatientName:
		val, err = value.NewStringValue(vr.PersonName, []string{a.config.PatientName})
	case tag.PatientID:
		val, err = value.NewStringValue(vr.LongString, []string{a.config.PatientID})
	case tag.InstitutionName:
		val, err = value.NewStringValue(vr.LongString, []string{a.config.InstitutionName})
	default:
		// Generic dummy values based on VR
		switch elem.VR() {
		case vr.PersonName:
			val, err = value.NewStringValue(vr.PersonName, []string{"ANONYMOUS"})
		case vr.Date:
			val, err = value.NewStringValue(vr.Date, []string{"19000101"})
		case vr.Time:
			val, err = value.NewStringValue(vr.Time, []string{"000000"})
		case vr.DateTime:
			val, err = value.NewStringValue(vr.DateTime, []string{"19000101000000"})
		case vr.AgeString:
			val, err = value.NewStringValue(vr.AgeString, []string{"000Y"})
		case vr.LongString, vr.ShortString:
			val, err = value.NewStringValue(elem.VR(), []string{"REMOVED"})
		default:
			return a.replaceWithEmpty(elem)
		}
	}

	if err != nil {
		return false, fmt.Errorf("failed to create dummy value: %w", err)
	}

	return true, elem.SetValue(val)
}

// cleanElement cleans identifying information while preserving clinical meaning.
func (a *Anonymizer) cleanElement(elem *element.Element) (bool, error) {
	// Implementation depends on element type
	// For text fields, remove identifying patterns
	// For structured fields, preserve structure but remove identifiers

	switch elem.VR() {
	case vr.LongText, vr.ShortText, vr.UnlimitedText:
		// Clean text by removing common identifying patterns
		cleaned := cleanText(elem.Value().String())
		val, err := value.NewStringValue(elem.VR(), []string{cleaned})
		if err != nil {
			return false, fmt.Errorf("failed to create cleaned value: %w", err)
		}
		return true, elem.SetValue(val)
	default:
		// For other types, use dummy replacement
		return a.replaceWithDummy(elem)
	}
}

// replaceUID generates a new UID for the element.
func (a *Anonymizer) replaceUID(elem *element.Element) (bool, error) {
	if elem.VR() != vr.UniqueIdentifier {
		return false, fmt.Errorf("cannot replace UID for non-UI VR: %s", elem.VR())
	}

	newUID := uid.Generate()
	val, err := value.NewStringValue(vr.UniqueIdentifier, []string{newUID})
	if err != nil {
		return false, fmt.Errorf("failed to create UID value: %w", err)
	}
	return true, elem.SetValue(val)
}

// hashElement replaces the value with a one-way hash.
func (a *Anonymizer) hashElement(elem *element.Element) (bool, error) {
	// Simple hash implementation - in production, use proper cryptographic hash
	original := elem.Value().String()
	hashed := fmt.Sprintf("HASH_%d", hashString(original))

	val, err := value.NewStringValue(elem.VR(), []string{hashed})
	if err != nil {
		return false, fmt.Errorf("failed to create hash value: %w", err)
	}
	return true, elem.SetValue(val)
}

// copyDataSet creates a deep copy of a dataset.
func (a *Anonymizer) copyDataSet(ds *dicom.DataSet) (*dicom.DataSet, error) {
	newDS := dicom.NewDataSet()

	// Copy all elements
	err := ds.Walk(func(elem *element.Element) error {
		// Create a copy of the element
		newElem, err := element.NewElement(elem.Tag(), elem.VR(), elem.Value())
		if err != nil {
			return err
		}
		return newDS.Add(newElem)
	})

	return newDS, err
}

// Helper functions

func defaultOptionsForProfile(profile Profile) Options {
	switch profile {
	case ProfileBasic:
		return Options{
			RemovePrivateTags: true,
		}
	case ProfileClean:
		return Options{
			RemovePrivateTags: true,
			CleanPixelData:    true,
			CleanDescriptors:  true,
		}
	case ProfileRetainUIDs:
		return Options{
			RemovePrivateTags: true,
			RetainUIDs:        true,
		}
	case ProfileRetainDeviceIdentity:
		return Options{
			RemovePrivateTags:    true,
			RetainDeviceIdentity: true,
		}
	default:
		return Options{}
	}
}

func isPrivateTag(t tag.Tag) bool {
	return t.Group%2 == 1
}

func cleanText(text string) string {
	// Simple text cleaning - remove common patterns
	// In production, use more sophisticated NLP-based cleaning
	cleaned := text

	// Remove phone numbers (simple pattern)
	cleaned = strings.ReplaceAll(cleaned, "phone:", "REMOVED")

	// Remove email addresses (simple pattern)
	if strings.Contains(cleaned, "@") {
		cleaned = "CLEANED_TEXT"
	}

	return cleaned
}

func hashString(s string) uint32 {
	// Simple hash function - in production, use crypto hash
	var h uint32
	for i := 0; i < len(s); i++ {
		h = h*31 + uint32(s[i])
	}
	return h
}

// initializeActions sets up the action map based on the profile.
func (a *Anonymizer) initializeActions() {
	switch a.config.Profile {
	case ProfileBasic:
		a.initializeBasicProfile()

	case ProfileClean:
		a.initializeBasicProfile()
		a.initializeCleanDescriptorsProfile()
		if a.config.Options.CleanPixelData {
			a.initializeCleanPixelDataProfile()
		}

	case ProfileRetainUIDs:
		a.initializeBasicProfile()
		// RetainUIDs option is handled in initializeBasicProfile

	case ProfileRetainDeviceIdentity:
		a.initializeBasicProfile()
		// RetainDeviceIdentity option is handled in initializeBasicProfile

	case ProfileCustom:
		// For custom profiles, only use explicitly provided CustomActions
		// No automatic initialization

	default:
		a.initializeBasicProfile()
	}
}
