package rendering

import (
	"encoding/json"

	"github.com/bidon-io/bidon-backend/internal/adapter"
)

type bidExt struct {
	Rendering *Config `json:"rendering,omitempty"`
}

// ParseFromBidExt extracts rendering from bid.ext, backfills every section and field the DSP
// didn't set with its documented default, and validates the result per section. demandID
// identifies the DSP for log correlation when a section is rejected. Returns DefaultConfig
// when rendering is absent entirely or malformed JSON, which is equivalent to what a present
// but empty "rendering": {} produces. When rendering is present, a section that fails
// validation after defaulting falls back to that section's defaults while other, valid
// sections keep the DSP's customization (see sanitize). OpenRTB cannot reject a bid for bad
// rendering config, so mistakes degrade to defaults rather than being forwarded or discarding
// the whole config.
func ParseFromBidExt(ext json.RawMessage, demandID adapter.Key) *Config {
	if len(ext) == 0 {
		return DefaultConfig()
	}

	var envelope bidExt
	if err := json.Unmarshal(ext, &envelope); err != nil {
		return DefaultConfig()
	}
	if envelope.Rendering == nil {
		return DefaultConfig()
	}

	if err := ApplyDefaults(envelope.Rendering); err != nil {
		return DefaultConfig()
	}

	sanitize(envelope.Rendering, demandID)

	return envelope.Rendering
}
