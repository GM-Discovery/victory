package playerprofile

import "sort"

// IsReservedFieldKey reports whether a key belongs to the account layer and
// can therefore never be a catalogue fact or a Face override target
// (Kernel 61 §3.2, §6.6, §8.2).
func IsReservedFieldKey(key string) bool {
	switch key {
	case ReservedFieldStageName, ReservedFieldHandle, ReservedFieldEmail, ReservedFieldAccountUUID:
		return true
	default:
		return false
	}
}

// BuildProjectedFields turns effective facts into projected fields using the
// catalogue's face eligibility/region/default-priority plus any owner
// overrides. Facts for fields the catalogue no longer knows about (or that
// are not face_eligible) are omitted -- only typed eligible facts may appear
// on Face (Kernel 61 §6.6).
func BuildProjectedFields(cat Catalogue, facts map[string]ProfileFact, overrides map[string]FaceOverride) []ProjectedField {
	out := make([]ProjectedField, 0, len(facts))

	for _, page := range cat.Pages {
		for _, field := range page.Fields {
			if !field.FaceEligible {
				continue
			}
			fact, ok := facts[field.FieldKey]
			if !ok {
				continue
			}

			visibilityMode := VisibilityInferred
			priorityMode := PriorityInferred
			priorityScore := field.DefaultPriority

			if override, hasOverride := overrides[field.FieldKey]; hasOverride {
				if override.VisibilityMode != "" {
					visibilityMode = override.VisibilityMode
				}
				if override.PriorityMode == PriorityManual {
					priorityMode = PriorityManual
					priorityScore = override.PriorityScore
				}
			}

			out = append(out, ProjectedField{
				FieldKey:           field.FieldKey,
				Label:              field.FieldLabel,
				DisplayValue:       fact.DisplayValue,
				Region:             field.FaceRegion,
				FaceVisibilityMode: visibilityMode,
				FaceVisible:        visibilityMode != VisibilityHidden,
				PriorityMode:       priorityMode,
				PriorityScore:      priorityScore,
			})
		}
	}

	return out
}

// VisibleSortedFields filters to Face-visible fields and sorts by descending
// priority score, then label, mirroring the Kernel 59A character Face
// projector's ordering so both Face systems behave the same way to a user
// who's used one already.
func VisibleSortedFields(fields []ProjectedField) []ProjectedField {
	visible := make([]ProjectedField, 0, len(fields))
	for _, f := range fields {
		if f.FaceVisible {
			visible = append(visible, f)
		}
	}
	sort.SliceStable(visible, func(i, j int) bool {
		if visible[i].PriorityScore != visible[j].PriorityScore {
			return visible[i].PriorityScore > visible[j].PriorityScore
		}
		return visible[i].Label < visible[j].Label
	})
	return visible
}

// GroupByRegion buckets already-filtered/sorted fields into the five fixed
// Face regions (Kernel 61 §6.8). A region with no fields is simply absent
// from the map -- the Face omits empty regions.
func GroupByRegion(fields []ProjectedField) map[string][]ProjectedField {
	out := map[string][]ProjectedField{}
	for _, f := range fields {
		out[f.Region] = append(out[f.Region], f)
	}
	return out
}

// BuildTrailerFace assembles the compiled social projection: stage name
// (always present, always in the identity header) plus every Face-visible
// fact grouped by region (Kernel 61 §6.6, §6.8).
func BuildTrailerFace(projectionVersion, stageName string, fields []ProjectedField) TrailerFace {
	visible := VisibleSortedFields(fields)
	return TrailerFace{
		ProjectionVersion: projectionVersion,
		StageName:         stageName,
		Regions:           GroupByRegion(visible),
	}
}
