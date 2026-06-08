package server

import (
	"context"
	"iter"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicomweb"
	"github.com/codeninja55/go-radx/dimse"
)

// queryCatalogue answers a query at q.Level, applying the full DICOM match across BOTH the catalogue's
// indexed attributes and any match key the catalogue does not index. The catalogue narrows on the
// indexed keys and the Go matcher decides; for a key the catalogue cannot index (a free-text attribute
// such as BodyPartExamined), the catalogue alone would yield candidates that lack the attribute and a
// downstream matcher would reject them all. To honour such a key faithfully, the candidates are
// fetched at instance granularity, the full stored dataset is read from the ObjectStore so the matcher
// sees the real attribute values, and the survivors are collapsed to q.Level — the collapse that runs
// AFTER matching so an instance the unindexed key matched is never dropped before it is tested.
//
// When every match key is indexed the catalogue's own match and collapse are complete, so the query is
// forwarded unchanged and no dataset is fetched. The iterator fails closed on a backend fault, never a
// laundered empty success (PRD §9.2).
func queryCatalogue(ctx context.Context, cat Catalogue, store ObjectStore, match map[dicom.Tag]string, level dimse.QueryLevel, fuzzy bool) iter.Seq2[*dicom.DataSet, error] {
	unindexed := unindexedKeys(match)
	if len(unindexed) == 0 {
		return cat.Query(ctx, CatalogueQuery{Level: level, Match: match, Fuzzy: fuzzy})
	}

	return func(yield func(*dicom.DataSet, error) bool) {
		// Query at instance granularity so each candidate carries its SOP Instance UID, the key the
		// ObjectStore fetch needs. The catalogue still narrows and matches the indexed keys.
		cq := CatalogueQuery{Level: dimse.QueryLevelImage, Match: match, Fuzzy: fuzzy}
		collapser := newLevelCollapser(level)
		for candidate, err := range cat.Query(ctx, cq) {
			if err != nil {
				yield(nil, err)
				return
			}
			instance, ok := candidate.GetString(dicom.TagSOPInstanceUID)
			if !ok || instance == "" {
				continue
			}
			full, err := store.Get(ctx, dicom.SOPInstanceUID(instance))
			if err != nil {
				yield(nil, err)
				return
			}
			if !dicomweb.MatchDataSet(full, unindexed, fuzzy) {
				continue
			}
			row, fresh := collapser.collapse(full)
			if !fresh {
				continue
			}
			// Carry the matched unindexed attributes onto the level row so a downstream re-matcher (the
			// DICOMweb server re-applies its matcher over every key) sees the value the candidate matched
			// on, rather than treating the projected row as missing the attribute and rejecting it.
			copyMatchedAttributes(row, full, unindexed)
			if !yield(row, nil) {
				return
			}
		}
	}
}

// copyMatchedAttributes copies the value of each key's attribute from src onto dst, so a level row a
// caller projected from the catalogue's indexed columns also carries the unindexed attributes the
// match decided against. A multi-valued attribute is copied whole so a downstream matcher sees every
// value.
func copyMatchedAttributes(dst, src *dicom.DataSet, keys []dicomweb.MatchKey) {
	for _, mk := range keys {
		if vals, ok := src.GetStrings(mk.Tag); ok && len(vals) > 0 {
			dst.SetString(mk.Tag, vals...)
		}
	}
}

// unindexedKeys returns the authoritative match keys for the attributes the catalogue does not index,
// so the caller knows which keys it must decide against the stored dataset. A universal (empty) value
// constrains nothing and is dropped.
func unindexedKeys(match map[dicom.Tag]string) []dicomweb.MatchKey {
	column := columnByTag()
	var keys []dicomweb.MatchKey
	for tag, value := range match {
		if value == "" {
			continue
		}
		if _, indexed := column[tag]; indexed {
			continue
		}
		keys = append(keys, dicomweb.NewMatchKey(tag, value))
	}
	return keys
}
