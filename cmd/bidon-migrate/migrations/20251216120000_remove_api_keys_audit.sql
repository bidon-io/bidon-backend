-- +goose Up
-- +goose StatementBegin

-- Remove audit trigger from api_keys table (has UUID primary key, not compatible with BIGINT record_id)
DROP TRIGGER IF EXISTS audit_api_keys ON api_keys;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Re-add audit trigger to api_keys table
CREATE TRIGGER audit_api_keys
    AFTER INSERT OR UPDATE OR DELETE ON api_keys
    FOR EACH ROW EXECUTE FUNCTION audit_log_changes();

-- +goose StatementEnd
