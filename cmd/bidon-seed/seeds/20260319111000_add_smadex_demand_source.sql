-- +goose Up
-- +goose StatementBegin
INSERT INTO demand_sources (api_key, human_name, created_at, updated_at)
VALUES ('smadex', 'Smadex', NOW(), NOW())
ON CONFLICT (api_key) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM demand_sources WHERE api_key = 'smadex';
-- +goose StatementEnd
