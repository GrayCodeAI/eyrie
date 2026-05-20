package catalog

import "sort"

// ModelEntriesForProvider lists models from a compiled v1 catalog for one provider.
// New models appear here automatically when the eyrie catalog is updated — hosts must not hardcode IDs.
func ModelEntriesForProvider(compiled *CompiledCatalogV1, provider string) []ModelCatalogEntry {
	if compiled == nil {
		return nil
	}
	provider = CanonicalProviderID(provider)
	if provider == "" {
		return nil
	}
	seen := map[string]bool{}
	var out []ModelCatalogEntry
	add := func(model ModelV1, offering ModelOfferingV1) {
		if model.ID == "" || seen[model.ID] {
			return
		}
		seen[model.ID] = true
		inPrice, outPrice := 0.0, 0.0
		if offering.Pricing.RatesPer1M != nil {
			inPrice = offering.Pricing.RatesPer1M["input_tokens"]
			outPrice = offering.Pricing.RatesPer1M["output_tokens"]
		}
		out = append(out, ModelCatalogEntry{
			ID:               model.ID,
			DisplayName:      model.Name,
			ContextWindow:    model.ContextWindow,
			MaxOutput:        model.MaxOutput,
			InputPricePer1M:  inPrice,
			OutputPricePer1M: outPrice,
		})
	}
	if provider == "openrouter" {
		for _, offering := range compiled.OfferingsByDeployment["openrouter"] {
			model, ok := compiled.ModelsByID[offering.CanonicalModelID]
			if !ok {
				continue
			}
			add(model, offering)
		}
	} else {
		ids := make([]string, 0, len(compiled.ModelsByID))
		for id, model := range compiled.ModelsByID {
			if CanonicalProviderID(model.ProviderID) == provider {
				ids = append(ids, id)
			}
		}
		sort.Strings(ids)
		for _, id := range ids {
			add(compiled.ModelsByID[id], firstOfferingForModel(compiled, id))
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func firstOfferingForModel(compiled *CompiledCatalogV1, canonicalModelID string) ModelOfferingV1 {
	offerings := compiled.OfferingsByCanonicalModel[canonicalModelID]
	if len(offerings) == 0 {
		return ModelOfferingV1{}
	}
	sort.SliceStable(offerings, func(i, j int) bool {
		return offerings[i].DeploymentID < offerings[j].DeploymentID
	})
	return offerings[0]
}
