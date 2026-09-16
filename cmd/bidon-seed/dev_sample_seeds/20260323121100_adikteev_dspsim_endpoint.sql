-- +goose Up
-- +goose StatementBegin

-- Dev-only overlay: point the sample Adikteev account at bidon-dspsim.
-- Shared sample_seeds use the real DSP URL (staging / Coolify previews).
UPDATE demand_source_accounts
SET extra = jsonb_set(
        COALESCE(extra, '{}'::jsonb),
        '{endpoint}',
        '"http://bidon-dspsim:1325/openrtb/bid"'
    ),
    updated_at = NOW()
WHERE id = 3005;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

UPDATE demand_source_accounts
SET extra = jsonb_set(
        COALESCE(extra, '{}'::jsonb),
        '{endpoint}',
        '"http://appodeal-eu.dsp.adikteev.com"'
    ),
    updated_at = NOW()
WHERE id = 3005;

-- +goose StatementEnd
