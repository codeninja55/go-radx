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
