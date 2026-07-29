package rendering

import (
	"encoding/json"

	"github.com/bidon-io/bidon-backend/internal/adapter"
)

// Normalize returns a fully defaulted and sanitized config. A nil cfg yields DefaultConfig().
// When cfg is present, missing fields are backfilled and invalid sections fall back to that
// section's defaults while valid sections keep the DSP's customization (see sanitize).
// OpenRTB cannot reject a bid for bad rendering config, so mistakes degrade to defaults.
func Normalize(cfg *Config, demandID adapter.Key) *Config {
	if cfg == nil {
		return DefaultConfig()
	}
	if err := ApplyDefaults(cfg); err != nil {
		return DefaultConfig()
	}
	sanitize(cfg, demandID)
	return cfg
}

// ParseFromBidExt extracts rendering from bid.ext and Normalizes it. demandID identifies the
// DSP for log correlation when a section is rejected. Returns DefaultConfig when rendering
// is absent entirely or Ext is malformed JSON, which is equivalent to a present but empty
// "rendering": {}. Prefer DecodeBidExt when signaldata is needed from the same Ext bytes.
func ParseFromBidExt(ext json.RawMessage, demandID adapter.Key) *Config {
	_, cfg, err := DecodeBidExt(ext, demandID)
	if err != nil {
		return DefaultConfig()
	}
	return cfg
}

// DecodeBidExt unmarshals seatbid.bid.ext once into signaldata and a Normalized rendering
// config. Empty Ext yields ("", DefaultConfig(), nil). Malformed Ext returns an error so
// callers that treat Ext shape as required can fail; soft-fallback callers should use
// ParseFromBidExt instead.
func DecodeBidExt(ext json.RawMessage, demandID adapter.Key) (signaldata string, cfg *Config, err error) {
	if len(ext) == 0 {
		return "", DefaultConfig(), nil
	}
	var envelope struct {
		Signaldata string  `json:"signaldata"`
		Rendering  *Config `json:"rendering"`
	}
	if err := json.Unmarshal(ext, &envelope); err != nil {
		return "", nil, err
	}
	return envelope.Signaldata, Normalize(envelope.Rendering, demandID), nil
}
