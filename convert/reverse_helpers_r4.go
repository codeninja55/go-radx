package convert

import (
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir/r4"
)

// conceptNameForR4 maps the first Coding of an R4 CodeableConcept back to a DICOM
// ConceptNameCode triplet, the R4 twin of conceptNameFor. The Coding.system maps
// back to its scheme designator via the shared schemeDesignatorFor; the
// Coding.display, or the CodeableConcept.text when the Coding has none, becomes the
// code meaning. A concept with no Coding yields the zero ConceptNameCode.
func conceptNameForR4(cc *r4.CodeableConcept) dicom.ConceptNameCode {
	if cc == nil || len(cc.Coding) == 0 {
		return dicom.ConceptNameCode{}
	}
	coding := cc.Coding[0]
	out := dicom.ConceptNameCode{}
	if coding.Code != nil {
		out.CodeValue = *coding.Code
	}
	if coding.System != nil {
		out.CodingSchemeDesignator = schemeDesignatorFor(*coding.System)
	}
	switch {
	case coding.Display != nil:
		out.CodeMeaning = *coding.Display
	case cc.Text != nil:
		out.CodeMeaning = *cc.Text
	}
	return out
}
