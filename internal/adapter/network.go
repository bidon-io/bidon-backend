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

// DemandEnvSecrets holds env-backed demand secrets available at config-build time.
type DemandEnvSecrets struct {
	MetaAppSecret  string
	MetaPlatformID string
	MolocoAPIKey   string
}

// EnvSecretInjector writes env-backed secrets into a processed config.
// A nil injector means the network has no env secrets.
type EnvSecretInjector func(dest map[string]any, secrets DemandEnvSecrets)

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

	// InjectEnvSecrets is nil for networks that do not read demand env secrets.
	InjectEnvSecrets EnvSecretInjector
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

// ApplyEnvSecrets runs InjectEnvSecrets when both the injector and secrets are present.
func (n Network) ApplyEnvSecrets(dest map[string]any, secrets *DemandEnvSecrets) {
	if n.InjectEnvSecrets == nil || secrets == nil {
		return
	}
	n.InjectEnvSecrets(dest, *secrets)
}

func applyFieldMaps(dest, source map[string]any, maps []FieldMap) {
	for _, m := range maps {
		dest[m.TargetKey] = source[m.SourceKey]
	}
}

func injectMetaEnvSecrets(dest map[string]any, secrets DemandEnvSecrets) {
	dest["app_secret"] = secrets.MetaAppSecret
	dest["platform_id"] = secrets.MetaPlatformID
}

func injectMolocoEnvSecrets(dest map[string]any, secrets DemandEnvSecrets) {
	dest["api_key"] = secrets.MolocoAPIKey
}
