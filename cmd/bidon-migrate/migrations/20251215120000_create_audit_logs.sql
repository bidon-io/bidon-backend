-- +goose Up
-- +goose StatementBegin

-- Create audit logs table
CREATE TABLE audit_logs (
    id          BIGSERIAL PRIMARY KEY,
    table_name  TEXT NOT NULL,
    record_id   BIGINT,
    operation   TEXT NOT NULL CHECK (operation IN ('INSERT', 'UPDATE', 'DELETE')),
    user_id     BIGINT,
    old_data    JSONB,
    new_data    JSONB,
    meta        JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes
CREATE INDEX idx_audit_logs_table_record ON audit_logs (table_name, record_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs USING BRIN (created_at);
CREATE INDEX idx_audit_logs_user_id ON audit_logs (user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_audit_logs_meta_auth_method ON audit_logs ((meta->>'auth_method'));

-- Trigger function: reads session variables and logs changes
CREATE OR REPLACE FUNCTION audit_log_changes() RETURNS TRIGGER AS $$
DECLARE
    v_user_id     BIGINT;
    v_auth_method TEXT;
    v_api_key_id  TEXT;
    v_meta        JSONB;
    v_old_data    JSONB;
    v_new_data    JSONB;
    v_record_id   BIGINT;
BEGIN
    -- Read audit context from session variables
    v_user_id := NULLIF(current_setting('audit.user_id', true), '')::BIGINT;
    v_auth_method := NULLIF(current_setting('audit.auth_method', true), '');
    v_api_key_id := NULLIF(current_setting('audit.api_key_id', true), '');

    -- Build meta JSON
    v_meta := '{}'::JSONB;
    IF v_auth_method IS NOT NULL THEN
        v_meta := v_meta || jsonb_build_object('auth_method', v_auth_method);
    END IF;
    IF v_api_key_id IS NOT NULL THEN
        v_meta := v_meta || jsonb_build_object('api_key_id', v_api_key_id);
    END IF;

    -- Build audit record based on operation
    IF TG_OP = 'DELETE' THEN
        v_record_id := OLD.id;
        v_old_data  := to_jsonb(OLD);
        v_new_data  := NULL;
    ELSIF TG_OP = 'UPDATE' THEN
        v_record_id := NEW.id;
        v_old_data  := to_jsonb(OLD);
        v_new_data  := to_jsonb(NEW);
    ELSIF TG_OP = 'INSERT' THEN
        v_record_id := NEW.id;
        v_old_data  := NULL;
        v_new_data  := to_jsonb(NEW);
    END IF;

    -- Insert audit log entry
    INSERT INTO audit_logs (table_name, record_id, operation, user_id, old_data, new_data, meta)
    VALUES (TG_TABLE_NAME, v_record_id, TG_OP, v_user_id, v_old_data, v_new_data, v_meta);

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Enable auditing on specific tables
CREATE TRIGGER audit_trigger AFTER INSERT OR UPDATE OR DELETE ON apps
    FOR EACH ROW EXECUTE FUNCTION audit_log_changes();
CREATE TRIGGER audit_trigger AFTER INSERT OR UPDATE OR DELETE ON app_demand_profiles
    FOR EACH ROW EXECUTE FUNCTION audit_log_changes();
CREATE TRIGGER audit_trigger AFTER INSERT OR UPDATE OR DELETE ON auction_configurations
    FOR EACH ROW EXECUTE FUNCTION audit_log_changes();
CREATE TRIGGER audit_trigger AFTER INSERT OR UPDATE OR DELETE ON line_items
    FOR EACH ROW EXECUTE FUNCTION audit_log_changes();
CREATE TRIGGER audit_trigger AFTER INSERT OR UPDATE OR DELETE ON segments
    FOR EACH ROW EXECUTE FUNCTION audit_log_changes();
CREATE TRIGGER audit_trigger AFTER INSERT OR UPDATE OR DELETE ON demand_source_accounts
    FOR EACH ROW EXECUTE FUNCTION audit_log_changes();
CREATE TRIGGER audit_trigger AFTER INSERT OR UPDATE OR DELETE ON api_keys
    FOR EACH ROW EXECUTE FUNCTION audit_log_changes();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Disable triggers
DROP TRIGGER IF EXISTS audit_trigger ON apps;
DROP TRIGGER IF EXISTS audit_trigger ON app_demand_profiles;
DROP TRIGGER IF EXISTS audit_trigger ON auction_configurations;
DROP TRIGGER IF EXISTS audit_trigger ON line_items;
DROP TRIGGER IF EXISTS audit_trigger ON segments;
DROP TRIGGER IF EXISTS audit_trigger ON demand_source_accounts;
DROP TRIGGER IF EXISTS audit_trigger ON api_keys;

-- Drop function and table
DROP FUNCTION IF EXISTS audit_log_changes();
DROP TABLE IF EXISTS audit_logs;

-- +goose StatementEnd
