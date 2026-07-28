package rendering

import "github.com/creasty/defaults"

// DefaultConfig returns the documented baseline rendering configuration, fully populated.
func DefaultConfig() *Config {
	cfg := &Config{}
	if err := ApplyDefaults(cfg); err != nil {
		panic(err)
	}
	return cfg
}

// ApplyDefaults backfills every section the DSP omitted (and the nested ImpressionTracking
// section) with a fresh, fully-defaulted value, then fills any remaining zero-valued fields
// within sections the DSP did provide from their struct default tags. A DSP only needs to set
// the fields it wants to override; everything else, whole sections included, defaults.
func ApplyDefaults(cfg *Config) error {
	if cfg == nil {
		return nil
	}
	if cfg.CloseButton == nil {
		cfg.CloseButton = &CloseButtonConfig{}
	}
	if cfg.EndCards == nil {
		cfg.EndCards = &EndCardsConfig{}
	}
	if cfg.Container == nil {
		cfg.Container = &ContainerConfig{}
	}
	if cfg.Container.ImpressionTracking == nil {
		cfg.Container.ImpressionTracking = &ImpressionTrackingConfig{}
	}
	if cfg.Creative == nil {
		cfg.Creative = &CreativeConfig{}
	}
	if cfg.StoreKit == nil {
		cfg.StoreKit = &StoreKitConfig{}
	}
	return defaults.Set(cfg)
}

func defaultCloseButton() *CloseButtonConfig {
	cfg := &CloseButtonConfig{}
	setDefaults(cfg)
	return cfg
}

func defaultEndCards() *EndCardsConfig {
	cfg := &EndCardsConfig{}
	setDefaults(cfg)
	return cfg
}

func defaultContainer() *ContainerConfig {
	cfg := &ContainerConfig{ImpressionTracking: &ImpressionTrackingConfig{}}
	setDefaults(cfg)
	return cfg
}

func defaultCreative() *CreativeConfig {
	cfg := &CreativeConfig{}
	setDefaults(cfg)
	return cfg
}

func defaultStoreKit() *StoreKitConfig {
	cfg := &StoreKitConfig{}
	setDefaults(cfg)
	return cfg
}

// setDefaults fills a single section's zero-valued fields from its struct default tags.
// Only called with the package's own section types, whose tags are fixed at compile time,
// so a failure here means a tag itself is malformed rather than anything request-dependent.
func setDefaults(section any) {
	if err := defaults.Set(section); err != nil {
		panic(err)
	}
}
