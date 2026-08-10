package adapter

// FieldMap projects one source config key onto a processed-config key.
// Prefer CopyKey / RenameKey constructors over literal structs.
type FieldMap struct {
	SourceKey string
	TargetKey string
}

// CopyKey allowlists a field onto the processed config under the same name.
func CopyKey(key string) FieldMap {
	return FieldMap{SourceKey: key, TargetKey: key}
}

// RenameKey copies a source field onto a differently named processed-config key.
func RenameKey(sourceKey, targetKey string) FieldMap {
	return FieldMap{SourceKey: sourceKey, TargetKey: targetKey}
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

// HasProcessedConfigMaps reports whether this network uses declarative field
// projections instead of passthrough account extra.
func (n Network) HasProcessedConfigMaps() bool {
	return len(n.AccountExtra) > 0 || len(n.AppData) > 0 || len(n.AdUnitExtra) > 0
}

// ApplyProcessedConfig writes projected fields into dest.
// AdUnitExtra is applied only when adUnitExtra is non-nil (RTB ad unit present).
func (n Network) ApplyProcessedConfig(
	dest map[string]any,
	accountExtra map[string]any,
	appData map[string]any,
	adUnitExtra map[string]any,
) {
	applyFieldMaps(dest, accountExtra, n.AccountExtra)
	applyFieldMaps(dest, appData, n.AppData)
	if adUnitExtra == nil {
		return
	}
	applyFieldMaps(dest, adUnitExtra, n.AdUnitExtra)
}

func applyFieldMaps(dest, source map[string]any, maps []FieldMap) {
	for _, m := range maps {
		dest[m.TargetKey] = source[m.SourceKey]
	}
}
