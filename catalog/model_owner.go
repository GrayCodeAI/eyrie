package catalog

import (
	"encoding/json"
	"strings"
)

// ModelOwner returns the upstream vendor for a catalog row (owned_by or id prefix).
func ModelOwner(entry ModelCatalogEntry) string {
	if o := strings.TrimSpace(entry.Owner); o != "" {
		return o
	}
	if o := ownerFromLiveMetadata(entry.LiveMetadata); o != "" {
		return o
	}
	return ownerFromModelID(entry.ID)
}

func ownerFromLiveMetadata(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var meta struct {
		OwnedBy string `json:"owned_by"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return ""
	}
	return strings.TrimSpace(meta.OwnedBy)
}

func ownerFromModelID(id string) string {
	id = strings.TrimSpace(id)
	if i := strings.Index(id, "/"); i > 0 {
		return id[:i]
	}
	return ""
}
