package adapter

// FieldMap maps a source JSON/config key onto a processed config key.
type FieldMap struct {
	From string
	To   string
}

// EnvSecret identifies demand secrets injected from process env / DemandConfig.
type EnvSecret int

const (
	EnvSecretNone EnvSecret = iota
	EnvSecretMeta
	EnvSecretMoloco
)

// Network is the canonical registration entry for a demand network.
type Network struct {
	Key               Key
	Label             string
	AccountType       string
	SupportsBidding   bool
	SupportsWaterfall bool

	AccountExtra []FieldMap
	AppData      []FieldMap
	AdUnitExtra  []FieldMap

	EnvSecret EnvSecret
}

// HasProcessedConfigMaps reports whether this network uses declarative remaps
// instead of passthrough account extra.
func (n Network) HasProcessedConfigMaps() bool {
	return len(n.AccountExtra) > 0 || len(n.AppData) > 0 || len(n.AdUnitExtra) > 0
}

// ApplyProcessedConfig writes remapped fields into dest.
// AdUnitExtra is applied only when adUnitExtra is non-nil (RTB ad unit present).
func (n Network) ApplyProcessedConfig(
	dest map[string]any,
	accountExtra map[string]any,
	appData map[string]any,
	adUnitExtra map[string]any,
) {
	for _, m := range n.AccountExtra {
		dest[m.To] = accountExtra[m.From]
	}
	for _, m := range n.AppData {
		dest[m.To] = appData[m.From]
	}
	if adUnitExtra == nil {
		return
	}
	for _, m := range n.AdUnitExtra {
		dest[m.To] = adUnitExtra[m.From]
	}
}
