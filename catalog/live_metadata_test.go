package catalog_test

import (
	"encoding/json"
	"testing"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/catalog/live"
)

func TestLiveEntriesToCatalog_PreservesFullJSONInOffering(t *testing.T) {
	raw := json.RawMessage(`{"id":"moonshotai/kimi-k2.6","owned_by":"moonshotai"}`)
	entries := catalog.LiveEntriesToCatalog([]live.Entry{{
		ID: "moonshotai/kimi-k2.6", DisplayName: "Kimi K2.6", RawJSON: raw,
	}})
	if len(entries) != 1 {
		t.Fatal("expected one entry")
	}
	if string(entries[0].LiveMetadata) != string(raw) {
		t.Fatalf("metadata = %s", entries[0].LiveMetadata)
	}
	c := catalog.CatalogV1FromLegacy(catalog.ModelCatalog{
		Source: "test",
		Providers: map[string][]catalog.ModelCatalogEntry{
			"canopywave": entries,
		},
	})
	var offering catalog.ModelOfferingV1
	for _, o := range c.Offerings {
		if o.DeploymentID == "canopywave" && o.NativeModelID == "moonshotai/kimi-k2.6" {
			offering = o
			break
		}
	}
	if offering.ID == "" {
		t.Fatal("canopywave offering not found")
	}
	if string(offering.LiveMetadata) != string(raw) {
		t.Fatalf("offering metadata = %s", offering.LiveMetadata)
	}
}
