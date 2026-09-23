-- +goose Up
-- +goose StatementBegin

-- Dev-only overlay: point the sample Adikteev account at bidon-dspsim.
-- Shared sample_seeds use the real DSP URL (staging / Coolify previews).
DO $$
DECLARE
    smadex_account_id     BIGINT := 3006;
BEGIN
    UPDATE demand_source_accounts
    SET extra = jsonb_set(
            COALESCE(extra, '{}'::jsonb),
            '{endpoint}',
            '"http://bidon-dspsim:1325/openrtb/bid"'
        ),
        updated_at = NOW()
    WHERE id = smadex_account_id;
END $$;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DO $$
DECLARE
    smadex_account_id     BIGINT := 3006;
BEGIN
    UPDATE demand_source_accounts
    SET extra = jsonb_set(
            COALESCE(extra, '{}'::jsonb),
            '{endpoint}',
            '"https://bon-use1.smadex.com/hyperad/rtb/437617/bid"'
        ),
        updated_at = NOW()
    WHERE id = smadex_account_id;
END $$;

-- +goose StatementEnd
