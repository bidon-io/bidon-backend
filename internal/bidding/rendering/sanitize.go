package rendering

import (
	"errors"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/bidon-io/bidon-backend/internal/adapter"
)

var assetIndexPattern = regexp.MustCompile(`Assets\[(\d+)\]`)

// sanitize validates cfg section by section and repairs it in place. A section that fails
// validation is replaced by its own defaults, as a whole, so one malformed field doesn't
// discard a sibling section's valid customization. Recovery is section-level rather than
// field-level because fields within a section are often connected via cross-field tags (e.g.
// StoreKitConfig.AppStoreID's required_if=Enabled, or CloseButtonConfig's custom_asset_url
// validator keying off Style) - a failure on one field can be a symptom of, or trigger, a
// related failure elsewhere in the section, so patching just the field that happened to error
// could still leave the section in a combination that was never valid together. Each
// replacement is logged so misconfigured bid responses are discoverable. EndCards.Assets is
// the one exception, repaired per-entry: each entry validates independently of its siblings
// and of the rest of the section, so a bad asset URL drops only that entry.
func sanitize(cfg *Config, demandID adapter.Key) {
	err := validate.Struct(cfg)
	if err == nil {
		return
	}

	var verrs validator.ValidationErrors
	ok := errors.As(err, &verrs)
	if !ok {
		logRenderingFailure(demandID, "config", err)
		*cfg = *DefaultConfig()
		return
	}

	bySection := make(map[string]validator.ValidationErrors)
	for _, fe := range verrs {
		section := topLevelSection(fe.Namespace())
		bySection[section] = append(bySection[section], fe)
	}

	for section, errs := range bySection {
		logRenderingFailure(demandID, section, errs)
	}

	if errs, ok := bySection["EndCards"]; ok {
		recoverEndCards(cfg, errs)
	}
	if _, ok := bySection["CloseButton"]; ok {
		cfg.CloseButton = defaultCloseButton()
	}
	if _, ok := bySection["Container"]; ok {
		cfg.Container = defaultContainer()
	}
	if _, ok := bySection["Creative"]; ok {
		cfg.Creative = defaultCreative()
	}
	if _, ok := bySection["StoreKit"]; ok {
		cfg.StoreKit = defaultStoreKit()
	}
}

// recoverEndCards drops only the invalid entries from EndCards.Assets when that's the sole
// source of error, preserving the rest of the DSP's endcard customization. Any other endcards
// field error (layout, count, etc.) is not repairable field-by-field, so the whole section
// falls back to defaults.
func recoverEndCards(cfg *Config, errs validator.ValidationErrors) {
	badIndexes := make(map[int]bool, len(errs))
	for _, fe := range errs {
		m := assetIndexPattern.FindStringSubmatch(fe.Namespace())
		if m == nil {
			cfg.EndCards = defaultEndCards()
			return
		}
		idx, convErr := strconv.Atoi(m[1])
		if convErr != nil {
			cfg.EndCards = defaultEndCards()
			return
		}
		badIndexes[idx] = true
	}

	kept := make([]EndcardAsset, 0, len(cfg.EndCards.Assets))
	for i, asset := range cfg.EndCards.Assets {
		if !badIndexes[i] {
			kept = append(kept, asset)
		}
	}
	cfg.EndCards.Assets = kept
}

// topLevelSection extracts the rendering section name (e.g. "CloseButton", "EndCards") from a
// go-playground/validator namespace such as "Config.EndCards.Assets[1].URL".
func topLevelSection(namespace string) string {
	parts := strings.SplitN(namespace, ".", 3)
	if len(parts) < 2 {
		return namespace
	}
	return parts[1]
}

func logRenderingFailure(demandID adapter.Key, section string, err error) {
	slog.Warn("rendering: invalid section in bid ext, falling back to section defaults",
		"demand_id", string(demandID),
		"section", section,
		"error", err,
	)
}
