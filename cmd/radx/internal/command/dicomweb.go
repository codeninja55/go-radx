package command

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/cmd/radx/internal/cli"
	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicomweb"
	"github.com/codeninja55/go-radx/logging"
)

// DICOMwebCmd groups the WADO-RS / STOW-RS / QIDO-RS clients. --url is the DICOMweb base; a bearer
// token comes from RADX_BEARER_TOKEN and is never logged (docs/reference/cli.md dicomweb).
type DICOMwebCmd struct {
	Wado DICOMwebWadoCmd `cmd:"" help:"Retrieve via WADO-RS."`
	Stow DICOMwebStowCmd `cmd:"" help:"Store via STOW-RS."`
	Qido DICOMwebQidoCmd `cmd:"" help:"Search via QIDO-RS."`
}

// newDICOMwebClient builds the shared client from the base URL and the bearer token (read from the
// environment, never logged). A nil/empty URL is a usage error.
func newDICOMwebClient(url, bearer string) (*dicomweb.Client, error) {
	if url == "" {
		return nil, &exitcode.UsageErr{Message: "--url (or RADX_DICOMWEB_URL) is required"}
	}
	var opts []dicomweb.ClientOption
	if bearer != "" {
		opts = append(opts, dicomweb.WithBearerToken(bearer))
	}
	return dicomweb.NewClient(url, opts...)
}

// DICOMwebWadoCmd retrieves objects (or, with --metadata, application/dicom+json metadata) addressed
// by the study/series/instance resource path, writing retrieved objects to --output-dir.
type DICOMwebWadoCmd struct {
	URL      string `name:"url" required:"" env:"RADX_DICOMWEB_URL" help:"DICOMweb base URL."`
	Study    string `name:"study" required:"" help:"Study Instance UID."`
	Series   string `name:"series" help:"Series Instance UID."`
	Instance string `name:"instance" help:"SOP Instance UID."`
	Metadata bool   `name:"metadata" help:"Retrieve metadata (application/dicom+json) instead of objects."`
	Output   string `name:"output-dir" default:"." help:"Where to write retrieved objects."`

	Bearer string `name:"bearer-token" env:"RADX_BEARER_TOKEN" hidden:"" help:"Bearer token (prefer the env var)."`
}

// wadoResult is the canonical machine shape for wado: the resource addressed and the count of
// objects (or metadata datasets) retrieved. It names counts and UIDs only.
type wadoResult struct {
	Status    string `json:"status"`
	Study     string `json:"study"`
	Series    string `json:"series,omitempty"`
	Instance  string `json:"instance,omitempty"`
	Retrieved int    `json:"retrieved"`
	OutputDir string `json:"output_dir,omitempty"`
}

// Run retrieves the addressed resource. With --metadata it fetches the DICOM-JSON metadata; without
// it streams the multipart/related objects and writes each to --output-dir under a Study/Series/SOP
// layout. A transport or HTTP fault is a network error (exit 4).
func (c *DICOMwebWadoCmd) Run(rc *RunContext) error {
	if rc.Out.Format == cli.FormatCSV {
		return &exitcode.UsageErr{Message: "dicomweb wado does not support --format csv; use human or json"}
	}
	client, err := newDICOMwebClient(c.URL, c.Bearer)
	if err != nil {
		return err
	}

	log := logging.FromContext(rc.Ctx)
	log.Debug("dicomweb wado: retrieving",
		zap.String("study", c.Study), zap.String("series", c.Series), zap.String("instance", c.Instance))

	if c.Metadata {
		return c.runMetadata(rc, client)
	}
	return c.runObjects(rc, client)
}

// runMetadata fetches the application/dicom+json metadata for the addressed resource and reports
// how many datasets it carried.
func (c *DICOMwebWadoCmd) runMetadata(rc *RunContext, client *dicomweb.Client) error {
	path := dicomweb.ResourcePath{
		Study:    dicom.UID(c.Study),
		Series:   dicom.UID(c.Series),
		Instance: dicom.UID(c.Instance),
	}
	datasets, err := client.RetrieveMetadata(rc.Ctx, path)
	if err != nil {
		return err
	}
	return c.emit(rc, wadoResult{
		Status:    "success",
		Study:     c.Study,
		Series:    c.Series,
		Instance:  c.Instance,
		Retrieved: len(datasets),
	})
}

// runObjects streams the addressed resource's instances and writes each to --output-dir. An
// instance level retrieve fetches one object; a study or series level streams every instance.
func (c *DICOMwebWadoCmd) runObjects(rc *RunContext, client *dicomweb.Client) error {
	if err := ensureDir(c.Output); err != nil {
		return err
	}

	count := 0
	write := func(ds *dicom.DataSet) error {
		if err := writeReceivedInstance(c.Output, ds, dicom.ExplicitVRLittleEndian); err != nil {
			return err
		}
		count++
		return nil
	}

	switch {
	case c.Instance != "":
		ds, err := client.RetrieveInstance(rc.Ctx, dicomweb.NewInstance(dicom.UID(c.Study), dicom.UID(c.Series), dicom.UID(c.Instance)))
		if err != nil {
			return err
		}
		if err := write(ds); err != nil {
			return err
		}
	case c.Series != "":
		for ds, err := range client.RetrieveSeries(rc.Ctx, dicom.UID(c.Study), dicom.UID(c.Series)) {
			if err != nil {
				return err
			}
			if err := write(ds); err != nil {
				return err
			}
		}
	default:
		for ds, err := range client.RetrieveStudy(rc.Ctx, dicom.UID(c.Study)) {
			if err != nil {
				return err
			}
			if err := write(ds); err != nil {
				return err
			}
		}
	}

	return c.emit(rc, wadoResult{
		Status:    "success",
		Study:     c.Study,
		Series:    c.Series,
		Instance:  c.Instance,
		Retrieved: count,
		OutputDir: c.Output,
	})
}

// emit renders the wado result in the resolved format.
func (c *DICOMwebWadoCmd) emit(rc *RunContext, r wadoResult) error {
	if rc.Out.Format == cli.FormatJSON {
		return rc.Out.EmitJSON(r)
	}
	_, err := fmt.Fprintf(rc.Out.Machine, "retrieved %d objects for study %s\n", r.Retrieved, r.Study)
	return err
}

// DICOMwebStowCmd POSTs one or more DICOM objects to the origin as multipart/related (STOW-RS).
type DICOMwebStowCmd struct {
	Paths []string `arg:"" name:"path" help:"DICOM files to store."`

	URL       string `name:"url" required:"" env:"RADX_DICOMWEB_URL" help:"DICOMweb base URL."`
	Study     string `name:"study" help:"Target Study Instance UID (optional)."`
	Recursive bool   `short:"R" name:"recursive" help:"Descend into directories for *.dcm files."`

	Bearer string `name:"bearer-token" env:"RADX_BEARER_TOKEN" hidden:"" help:"Bearer token (prefer the env var)."`
}

// stowResult is the canonical machine shape for stow: the per-outcome tally of accepted and failed
// instances the origin reported.
type stowResult struct {
	Status   string `json:"status"`
	Accepted int    `json:"accepted"`
	Failed   int    `json:"failed"`
}

// Run reads the input files, POSTs them as a STOW-RS multipart body, and reports the origin's store
// response. A partial or total store failure (a 202/409 or a non-empty Failed SOP Sequence) is a
// network-class failure: stow exits non-zero, never reading a partial store as success (PRD §9.2).
func (c *DICOMwebStowCmd) Run(rc *RunContext) error {
	if rc.Out.Format == cli.FormatCSV {
		return &exitcode.UsageErr{Message: "dicomweb stow does not support --format csv; use human or json"}
	}
	client, err := newDICOMwebClient(c.URL, c.Bearer)
	if err != nil {
		return err
	}

	files, err := resolveDICOMPaths(c.Paths, c.Recursive)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return &exitcode.UsageErr{Message: "no DICOM files to store"}
	}

	instances := make([]*dicom.DataSet, 0, len(files))
	for _, path := range files {
		f, readErr := dicom.ReadFile(path)
		if readErr != nil {
			// A file that cannot be read or parsed is a fail-closed fault: stow never silently drops a
			// requested instance (PRD §9.2).
			return readErr
		}
		instances = append(instances, f.DataSet)
	}

	log := logging.FromContext(rc.Ctx)
	log.Debug("dicomweb stow: storing", zap.Int("instances", len(instances)))

	// When --study is set, target the study-scoped /studies/{study} STOW path so the origin
	// constrains the request to that StudyInstanceUID and rejects an instance from a different
	// study; an empty --study posts to the unconstrained root /studies target.
	store := client.Store
	if c.Study != "" {
		store = func(ctx context.Context, ds ...*dicom.DataSet) (*dicomweb.StoreResponse, error) {
			return client.StoreToStudy(ctx, dicom.UID(c.Study), ds...)
		}
	}
	resp, storeErr := store(rc.Ctx, instances...)
	result := stowResult{Status: "success"}
	if resp != nil {
		result.Accepted = len(resp.Referenced)
		result.Failed = len(resp.Failed)
	}
	if storeErr != nil {
		result.Status = "failure"
		if emitErr := c.emit(rc, result); emitErr != nil {
			return emitErr
		}
		return storeErr
	}
	return c.emit(rc, result)
}

// emit renders the stow result in the resolved format.
func (c *DICOMwebStowCmd) emit(rc *RunContext, r stowResult) error {
	if rc.Out.Format == cli.FormatJSON {
		return rc.Out.EmitJSON(r)
	}
	_, err := fmt.Fprintf(rc.Out.Machine, "stored %d accepted, %d failed\n", r.Accepted, r.Failed)
	return err
}

// DICOMwebQidoCmd searches the origin (QIDO-RS) at a level and streams one match per result.
type DICOMwebQidoCmd struct {
	URL     string   `name:"url" required:"" env:"RADX_DICOMWEB_URL" help:"DICOMweb base URL."`
	Level   string   `name:"level" enum:"studies,series,instances" default:"studies" help:"Search level."`
	Study   string   `name:"study" help:"Scope a series/instance search to this study."`
	Series  string   `name:"series" help:"Scope an instance search to this series."`
	Match   []string `name:"match" help:"Attribute match (key=value); repeat to add keys."`
	Include []string `name:"include" help:"Additional return attributes (keyword or GGGGEEEE)."`
	Limit   int      `name:"limit" default:"0" help:"Maximum matches (0 = server default)."`

	Bearer string `name:"bearer-token" env:"RADX_BEARER_TOKEN" hidden:"" help:"Bearer token (prefer the env var)."`
}

// Run searches the origin and streams matches in the resolved format, one JSON Line per match.
func (c *DICOMwebQidoCmd) Run(rc *RunContext) error {
	client, err := newDICOMwebClient(c.URL, c.Bearer)
	if err != nil {
		return err
	}
	query, err := c.buildSearchQuery()
	if err != nil {
		return err
	}

	results, searchErr := c.search(rc.Ctx, client, query)
	if searchErr != nil {
		return searchErr
	}

	em := &matchEmitter{out: rc.Out, columns: c.Match}
	if startErr := em.start(); startErr != nil {
		return startErr
	}
	for _, res := range results {
		if emitErr := em.emit(datasetAttributes(res.DataSet)); emitErr != nil {
			return emitErr
		}
	}
	return em.finish()
}

// search dispatches to the level-appropriate QIDO-RS search.
func (c *DICOMwebQidoCmd) search(ctx context.Context, client *dicomweb.Client, query dicomweb.SearchQuery) ([]dicomweb.SearchResult, error) {
	switch c.Level {
	case "series":
		return client.SearchSeries(ctx, dicom.UID(c.Study), query)
	case "instances":
		return client.SearchInstances(ctx, dicom.UID(c.Study), dicom.UID(c.Series), query)
	default:
		return client.SearchStudies(ctx, query)
	}
}

// buildSearchQuery turns the --match, --include, and --limit flags into a SearchQuery, validating
// each match/include key against the dictionary so a malformed key is a usage error.
func (c *DICOMwebQidoCmd) buildSearchQuery() (dicomweb.SearchQuery, error) {
	q := dicomweb.SearchQuery{Limit: c.Limit}
	if len(c.Match) > 0 {
		q.Match = make(map[string]string, len(c.Match))
	}
	for _, raw := range c.Match {
		key, value, ok := splitKeyValue(raw)
		if !ok {
			return dicomweb.SearchQuery{}, &exitcode.UsageErr{Message: fmt.Sprintf("invalid --match %q (use key=value)", raw)}
		}
		if _, resolvable := parseTagSpec(key); !resolvable {
			return dicomweb.SearchQuery{}, &exitcode.UsageErr{Message: fmt.Sprintf("invalid --match key %q", key)}
		}
		q.Match[key] = value
	}
	for _, inc := range c.Include {
		if _, resolvable := parseTagSpec(inc); !resolvable {
			return dicomweb.SearchQuery{}, &exitcode.UsageErr{Message: fmt.Sprintf("invalid --include key %q", inc)}
		}
		q.IncludeFields = append(q.IncludeFields, inc)
	}
	return q, nil
}
