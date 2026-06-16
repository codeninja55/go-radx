package dimse

import (
	"context"
	"iter"

	"github.com/codeninja55/go-radx/dicom"
)

// ModalityWorklistInformationModelFind is the Modality Worklist Information Model — FIND SOP Class
// UID (PS3.4 K.6), the abstract syntax of an MWL C-FIND presentation context. It is the model a
// modality queries to discover the procedure steps scheduled for it (step 2 of the radiology
// workflow). It is exported so a caller may name it in WithQueryModel for a plain Find, or so an SCP
// recognises it; FindWorklist selects it implicitly.
const ModalityWorklistInformationModelFind = modalityWorklistSOPClass

// FindWorklist issues a Modality Worklist C-FIND and yields (Status, identifier) for each response,
// the same streaming contract as Find (PS3.4 K.4.1.2). It is step 2 of the radiology workflow: a
// modality queries the Modality Worklist Information Model to discover the procedure steps scheduled
// for it. The matching keys live under the Scheduled Procedure Step Sequence (0040,0100) of the
// query identifier and its nested attributes; the caller builds that identifier (see
// NewWorklistQuery for a starting point).
//
// The Modality Worklist is a FLAT information model with no hierarchy of levels, so — unlike Patient
// Root or Study Root C-FIND — FindWorklist does NOT write a Query/Retrieve Level (0008,0052) into
// the sent identifier (PS3.4 K.6.1.2.1, level suppression). It negotiates the Modality Worklist
// Information Model context (BasicWorklistContexts), categorises each response against the worklist
// service-class status table, and otherwise behaves exactly as Find: each Pending (0xFF00/0xFF01)
// yields a matching identifier, the terminal status yields a nil dataset and ends iteration,
// breaking the range loop or cancelling ctx sends a C-CANCEL-RQ, and a transport fault is surfaced
// via Association.LastError() read after the loop. A pre-flight fault (unestablished/released
// association, no negotiated worklist context) yields a single terminal Failure and sets LastError,
// never panicking.
//
// An Association is not safe for concurrent queries; run one Find/FindWorklist/Get/Move iterator per
// association at a time.
func (a *Association) FindWorklist(
	ctx context.Context,
	query *dicom.DataSet,
	opts ...QueryOption,
) iter.Seq2[Status, *dicom.DataSet] {
	// The Modality Worklist model is flat (no level), so the QueryLevel argument to Find is unused for
	// it: Find suppresses (0008,0052) for the worklist model and categorises against the worklist
	// status table. A worklist-specific WithQueryModel from the caller is ignored — FindWorklist always
	// targets the worklist model — so it is appended last and wins over any earlier model override.
	mwlOpts := make([]QueryOption, 0, len(opts)+1)
	mwlOpts = append(mwlOpts, opts...)
	mwlOpts = append(mwlOpts, WithQueryModel(modalityWorklistSOPClass))
	return a.Find(ctx, query, QueryLevelStudy, mwlOpts...)
}

// NewWorklistQuery builds an empty Modality Worklist query identifier carrying a single empty
// Scheduled Procedure Step Sequence (0040,0100) item, the universal-matching skeleton an MWL C-FIND
// starts from (PS3.4 K.6.1.2): the SCU adds the return keys it wants (as empty elements) and the
// match keys it constrains (with values) to the top level and to the sequence item, then sends it.
// The returned dataset is freshly allocated and owned by the caller. It carries no Query/Retrieve
// Level — the worklist model is flat.
func NewWorklistQuery() *dicom.DataSet {
	query := dicom.NewDataSet()
	step := dicom.NewDataSet()
	query.Set(dicom.Element{
		Tag:   dicom.TagScheduledProcedureStepSequence,
		VR:    dicom.VRSQ,
		Value: dicom.NewSequenceValue(dicom.NewSequence(step)),
	})
	return query
}

// worklistSPSTags is the SetWorklistMatch routing set: every attribute the current PS3.4
// Table K.6-1 names as a DIRECT child of the Scheduled Procedure Step Sequence (0040,0100).
// The Modality Worklist information model defines these inside the sequence item, not at the
// query's top level, so an SCP matches them against the item and one placed top-level would
// never constrain the query. Children of the nested protocol sequences (the table's ">>" rows
// under the Scheduled Protocol Code Sequence and the referenced protocol sequences) are NOT
// routed individually: a caller supplies those sequences whole and their items travel with
// them. The table's residual "All other Attributes of the Scheduled Procedure Step Sequence"
// row is deliberately not expressible here - an unlisted tag cannot be told apart from a
// top-level worklist key by tag alone, so unlisted keys stay top-level.
var worklistSPSTags = map[dicom.Tag]struct{}{
	dicom.TagModality:                            {}, // (0008,0060)
	dicom.TagReferencedDefinedProtocolSequence:   {}, // (0018,990C)
	dicom.TagReferencedPerformedProtocolSequence: {}, // (0018,990D)
	dicom.TagRequestedContrastAgent:              {}, // (0032,1070)
	dicom.TagScheduledStationAETitle:             {}, // (0040,0001)
	dicom.TagScheduledProcedureStepStartDate:     {}, // (0040,0002)
	dicom.TagScheduledProcedureStepStartTime:     {}, // (0040,0003)
	dicom.TagScheduledPerformingPhysicianName:    {}, // (0040,0006)
	dicom.TagScheduledProcedureStepDescription:   {}, // (0040,0007)
	dicom.TagScheduledProtocolCodeSequence:       {}, // (0040,0008)
	dicom.TagScheduledProcedureStepID:            {}, // (0040,0009)
	dicom.TagScheduledStationName:                {}, // (0040,0010)
	dicom.TagScheduledProcedureStepLocation:      {}, // (0040,0011)
	dicom.TagPreMedication:                       {}, // (0040,0012)
	dicom.TagScheduledProcedureStepStatus:        {}, // (0040,0020)
}

// SetWorklistMatch sets one match/return key on a Modality Worklist query identifier where the
// information model defines it (PS3.4 Table K.6-1): a Scheduled Procedure Step attribute
// (Modality, Scheduled Station AE Title, the SPS start date/time, performing physician, SPS
// description/ID, ...) is set inside the FIRST Scheduled Procedure Step Sequence (0040,0100)
// item — the universal-match item NewWorklistQuery seeds — and every other attribute is set at
// the query's top level. A query with no sequence item yet (a caller that did not start from
// NewWorklistQuery) has one seeded so the SPS key still lands where the SCP matches it.
func SetWorklistMatch(query *dicom.DataSet, e dicom.Element) {
	if _, sps := worklistSPSTags[e.Tag]; !sps {
		query.Set(e)
		return
	}
	if seq, ok := query.GetSequence(dicom.TagScheduledProcedureStepSequence); ok {
		for item := range seq.Items() {
			item.DataSet.Set(e)
			return
		}
	}
	step := dicom.NewDataSet()
	step.Set(e)
	query.Set(dicom.Element{
		Tag:   dicom.TagScheduledProcedureStepSequence,
		VR:    dicom.VRSQ,
		Value: dicom.NewSequenceValue(dicom.NewSequence(step)),
	})
}
