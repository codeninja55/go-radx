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
// retain names attributes the caller needs on every returned (collapsed) row beyond the level's own
// projection: every match-key tag (so a downstream re-matcher sees the value the candidate matched on
// rather than a projected row that dropped the attribute) plus any caller-requested return field
// (QIDO-RS includefield). includeAll requests every available attribute. Because the level collapse
// projects a row to its level's identifying columns, a retained attribute that the level does not
// project — or any attribute when includeAll is set — would otherwise be lost; it is preserved by
// merging it back onto the collapsed row from the matched full instance row.
//
// When every match key is indexed, no retained tag is unindexed, and includeAll is not set, the
// catalogue's own match and collapse are complete (the collapser carries the retained indexed columns),
// so the query is forwarded unchanged and no dataset is fetched. The iterator fails closed on a backend
// fault, never a laundered empty success (PRD §9.2).
func queryCatalogue(ctx context.Context, cat Catalogue, store ObjectStore, match map[dicom.Tag]string, level dimse.QueryLevel, retain []dicom.Tag, includeAll bool, fuzzy bool) iter.Seq2[*dicom.DataSet, error] {
	retainTags := retainedTags(match, retain)
	if !includeAll && len(unindexedKeys(match)) == 0 && allIndexed(retainTags) {
		return cat.Query(ctx, CatalogueQuery{Level: level, Match: match, Return: retainTags, Fuzzy: fuzzy})
	}

	unindexed := unindexedKeys(match)
	return func(yield func(*dicom.DataSet, error) bool) {
		// Query at instance granularity so each candidate carries its SOP Instance UID, the key the
		// ObjectStore fetch needs. The catalogue still narrows and matches the indexed keys, and projects
		// the retained indexed columns onto the candidate row.
		cq := CatalogueQuery{Level: dimse.QueryLevelImage, Match: match, Return: retainTags, Fuzzy: fuzzy}
		collapser := newLevelCollapser(level, retainTags)
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
			// Merge the attributes the caller will re-match or project onto the level row from the matched
			// full instance row, so a downstream re-matcher (the DICOMweb server re-applies its matcher over
			// every key) and projection (includefield) see the real values rather than treating the projected
			// row as missing the attribute. includeAll carries every available attribute through.
			if includeAll {
				mergeAll(row, full)
			} else {
				mergeTags(row, full, retainTags)
			}
			if !yield(row, nil) {
				return
			}
		}
	}
}

// retainedTags is the set of attributes a returned row must carry beyond its level's own projection:
// every match-key tag (so the value a candidate matched on survives a downstream re-match) unioned with
// the caller's explicit return fields. A universal (empty) match value constrains nothing and so does
// not need to be carried back. The result is deduplicated, preserving caller order then match order.
func retainedTags(match map[dicom.Tag]string, retain []dicom.Tag) []dicom.Tag {
	seen := make(map[dicom.Tag]struct{}, len(match)+len(retain))
	out := make([]dicom.Tag, 0, len(match)+len(retain))
	add := func(tag dicom.Tag) {
		if _, ok := seen[tag]; ok {
			return
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	for _, tag := range retain {
		add(tag)
	}
	for tag, value := range match {
		if value == "" {
			continue
		}
		add(tag)
	}
	return out
}

// allIndexed reports whether every tag is an attribute the catalogue indexes, so a retained tag can be
// projected from the catalogue row without fetching the stored dataset. A tag the catalogue does not
// store can only be supplied from the ObjectStore, forcing the instance-fetch path.
func allIndexed(tags []dicom.Tag) bool {
	column := columnByTag()
	for _, tag := range tags {
		if _, ok := column[tag]; !ok {
			return false
		}
	}
	return true
}

// mergeTags copies the value of each tag from src onto dst when src carries it, so a level row a caller
// projected from the catalogue's indexed columns also carries the attributes a downstream matcher or
// includefield projection needs. The element is cloned so dst never aliases src's backing storage.
func mergeTags(dst, src *dicom.DataSet, tags []dicom.Tag) {
	for _, tag := range tags {
		if e, ok := src.Get(tag); ok {
			dst.Set(cloneElement(e))
		}
	}
}

// mergeAll copies every attribute src carries onto dst (includefield=all), so the returned row carries
// the full available attribute set rather than only the level's projection. Each element is cloned so
// dst never aliases src's backing storage.
func mergeAll(dst, src *dicom.DataSet) {
	for e := range src.All() {
		dst.Set(cloneElement(e))
	}
}

// cloneElement deep-copies an element through a single-element dataset clone, so a merged attribute
// never aliases the stored dataset's backing storage.
func cloneElement(e dicom.Element) dicom.Element {
	tmp := dicom.NewDataSet()
	tmp.Set(e)
	cloned := tmp.Clone()
	out, _ := cloned.Get(e.Tag)
	return out
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
