package catalog

import (
	"encoding/json"
	"strings"
)

// ModelOwner returns the upstream vendor for a catalog row (owned_by or id prefix).
func ModelOwner(entry ModelCatalogEntry) string {
	return DisplayModelOwner(entry.Owner, entry.ID, entry.LiveMetadata)
}

// DisplayModelLabel returns a UI-friendly model name. OpenRouter latest aliases use a
// leading ~ in the API id (e.g. ~anthropic/claude-haiku-latest); strip it for display only.
func DisplayModelLabel(id, displayName string) string {
	label := strings.TrimSpace(displayName)
	if label == "" {
		label = strings.TrimSpace(id)
	}
	return strings.TrimPrefix(label, "~")
}

// DisplayModelOwner returns a UI-friendly owner slug without OpenRouter ~ prefixes.
func DisplayModelOwner(owner, id string, liveMetadata ...json.RawMessage) string {
	owner = strings.TrimPrefix(strings.TrimSpace(owner), "~")
	if owner != "" {
		return owner
	}
	for _, raw := range liveMetadata {
		if o := ownerFromLiveMetadata(raw); o != "" {
			return strings.TrimPrefix(o, "~")
		}
	}
	return ownerFromModelID(strings.TrimPrefix(strings.TrimSpace(id), "~"))
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
