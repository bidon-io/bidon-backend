-- +goose Up
-- +goose StatementBegin

-- Create a test user for the apps
INSERT INTO users (id, email, password_hash, is_admin, created_at, updated_at, public_uid)
VALUES (
    1000,
    'demo@bidon.org',
    'AQIDBAUGBwgJCgsMDQ4PEA$bXTpgPF+hV8Z3JqzQqHFUa0oWxECgJ7LRy4f5Kw9NhM',
    false,
    NOW(),
    NOW(),
    1000
) ON CONFLICT (id) DO NOTHING;

-- Get BidMachine demand source ID (should exist from 20250325114723_demand_sources.sql)
DO $$
DECLARE
    bidmachine_id BIGINT;
    chess_app_id BIGINT := 2000;
    mahjong_app_id BIGINT := 2001;
    trivial_app_id BIGINT := 2002;
    bidmachine_account_id BIGINT := 3000;
    chess_profile_id BIGINT := 4000;
    mahjong_profile_id BIGINT := 4001;
    trivial_profile_id BIGINT := 4002;
BEGIN
    -- Get BidMachine demand source ID
    SELECT id INTO bidmachine_id FROM demand_sources WHERE api_key = 'bidmachine';

    IF bidmachine_id IS NULL THEN
        RAISE EXCEPTION 'BidMachine demand source not found. Please run demand_sources seed first.';
    END IF;

    -- Create BidMachine demand source account
    INSERT INTO demand_source_accounts (
        id, demand_source_id, user_id, type, extra, bidding, is_default, created_at, updated_at, label, public_uid
    ) VALUES (
        bidmachine_account_id,
        bidmachine_id,
        1000,
        'DemandSourceAccount::bidmachine',
        '{"seller_id": "1", "endpoint": "x.appbaqend.com", "mediation_config": ["rewarded", "interstitial", "banner"]}'::jsonb,
        false,
        true,
        NOW(),
        NOW(),
        'BidMachine Production Account',
        bidmachine_account_id
    ) ON CONFLICT (id) DO NOTHING;

    -- Create Chess App (Android)
    INSERT INTO apps (
        id, user_id, platform_id, human_name, package_name, app_key, settings, created_at, updated_at, public_uid
    ) VALUES (
        chess_app_id,
        1000,
        1, -- Android
        'Chess Master',
        'com.demo.chessmaster',
        'chess_' || chess_app_id,
        '{}'::jsonb,
        NOW(),
        NOW(),
        chess_app_id
    ) ON CONFLICT (id) DO NOTHING;

    -- Create Mahjong App (iOS)
    INSERT INTO apps (
        id, user_id, platform_id, human_name, package_name, app_key, settings, created_at, updated_at, public_uid
    ) VALUES (
        mahjong_app_id,
        1000,
        4, -- iOS
        'Mahjong Quest',
        'com.demo.mahjongquest',
        'mahjong_' || mahjong_app_id,
        '{}'::jsonb,
        NOW(),
        NOW(),
        mahjong_app_id
    ) ON CONFLICT (id) DO NOTHING;

    -- Create Trivial Pursuit App (Android)
    INSERT INTO apps (
        id, user_id, platform_id, human_name, package_name, app_key, settings, created_at, updated_at, public_uid
    ) VALUES (
        trivial_app_id,
        1000,
        1, -- Android
        'Trivial Pursuit Ultimate',
        'com.demo.trivialpursuit',
        'trivial_' || trivial_app_id,
        '{}'::jsonb,
        NOW(),
        NOW(),
        trivial_app_id
    ) ON CONFLICT (id) DO NOTHING;

    -- Create app demand profiles (links apps to BidMachine)
    INSERT INTO app_demand_profiles (
        id, app_id, account_type, account_id, demand_source_id, data, created_at, updated_at, public_uid, enabled
    ) VALUES
    (
        chess_profile_id,
        chess_app_id,
        'DemandSourceAccount::bidmachine',
        bidmachine_account_id,
        bidmachine_id,
        '{"source_id": "5"}'::jsonb,
        NOW(),
        NOW(),
        chess_profile_id,
        true
    ),
    (
        mahjong_profile_id,
        mahjong_app_id,
        'DemandSourceAccount::bidmachine',
        bidmachine_account_id,
        bidmachine_id,
        '{"source_id": "6"}'::jsonb,
        NOW(),
        NOW(),
        mahjong_profile_id,
        true
    ),
    (
        trivial_profile_id,
        trivial_app_id,
        'DemandSourceAccount::bidmachine',
        bidmachine_account_id,
        bidmachine_id,
        '{"source_id": "7"}'::jsonb,
        NOW(),
        NOW(),
        trivial_profile_id,
        true
    )
    ON CONFLICT (id) DO NOTHING;

    -- Create line items for Chess (Interstitial and Rewarded)
    INSERT INTO line_items (
        id, app_id, account_type, account_id, human_name, bid_floor, ad_type, extra,
        created_at, updated_at, width, height, format, public_uid, bidding
    ) VALUES
    -- Chess - Interstitial
    (
        5000,
        chess_app_id,
        'DemandSourceAccount::bidmachine',
        bidmachine_account_id,
        'Chess Interstitial',
        0.50,
        1, -- Interstitial
        '{}'::jsonb,
        NOW(),
        NOW(),
        0,
        0,
        '',
        5000,
        false
    ),
    -- Chess - Rewarded
    (
        5001,
        chess_app_id,
        'DemandSourceAccount::bidmachine',
        bidmachine_account_id,
        'Chess Rewarded Video',
        1.00,
        6, -- Rewarded
        '{}'::jsonb,
        NOW(),
        NOW(),
        0,
        0,
        '',
        5001,
        false
    ) ON CONFLICT (id) DO NOTHING;

    -- Create line items for Mahjong (Banner, Interstitial, and Rewarded)
    INSERT INTO line_items (
        id, app_id, account_type, account_id, human_name, bid_floor, ad_type, extra,
        created_at, updated_at, width, height, format, public_uid, bidding
    ) VALUES
    -- Mahjong - Banner
    (
        5010,
        mahjong_app_id,
        'DemandSourceAccount::bidmachine',
        bidmachine_account_id,
        'Mahjong Banner 320x50',
        0.10,
        3, -- Banner
        '{}'::jsonb,
        NOW(),
        NOW(),
        320,
        50,
        'BANNER',
        5010,
        false
    ),
    -- Mahjong - Interstitial
    (
        5011,
        mahjong_app_id,
        'DemandSourceAccount::bidmachine',
        bidmachine_account_id,
        'Mahjong Interstitial',
        0.75,
        1, -- Interstitial
        '{}'::jsonb,
        NOW(),
        NOW(),
        0,
        0,
        '',
        5011,
        false
    ),
    -- Mahjong - Rewarded
    (
        5012,
        mahjong_app_id,
        'DemandSourceAccount::bidmachine',
        bidmachine_account_id,
        'Mahjong Rewarded Video',
        1.50,
        6, -- Rewarded
        '{}'::jsonb,
        NOW(),
        NOW(),
        0,
        0,
        '',
        5012,
        false
    ) ON CONFLICT (id) DO NOTHING;

    -- Create line items for Trivial Pursuit (Banner and Interstitial)
    INSERT INTO line_items (
        id, app_id, account_type, account_id, human_name, bid_floor, ad_type, extra,
        created_at, updated_at, width, height, format, public_uid, bidding
    ) VALUES
    -- Trivial Pursuit - Banner Leaderboard
    (
        5020,
        trivial_app_id,
        'DemandSourceAccount::bidmachine',
        bidmachine_account_id,
        'Trivial Banner 728x90',
        0.20,
        3, -- Banner
        '{}'::jsonb,
        NOW(),
        NOW(),
        728,
        90,
        'LEADERBOARD',
        5020,
        false
    ),
    -- Trivial Pursuit - Banner MREC
    (
        5021,
        trivial_app_id,
        'DemandSourceAccount::bidmachine',
        bidmachine_account_id,
        'Trivial Banner 300x250',
        0.30,
        3, -- Banner
        '{}'::jsonb,
        NOW(),
        NOW(),
        300,
        250,
        'MREC',
        5021,
        false
    ),
    -- Trivial Pursuit - Interstitial
    (
        5022,
        trivial_app_id,
        'DemandSourceAccount::bidmachine',
        bidmachine_account_id,
        'Trivial Interstitial',
        0.60,
        1, -- Interstitial
        '{}'::jsonb,
        NOW(),
        NOW(),
        0,
        0,
        '',
        5022,
        false
    ) ON CONFLICT (id) DO NOTHING;

    -- Create auction configurations for Chess
    INSERT INTO auction_configurations (
        id, name, app_id, ad_type, rounds, status, settings, pricefloor,
        created_at, updated_at, segment_id, external_win_notifications, public_uid,
        timeout, demands, bidding, ad_unit_ids
    ) VALUES
    -- Chess - Interstitial Config
    (
        6000,
        'Chess Interstitial Config',
        chess_app_id,
        1, -- Interstitial
        '[]'::jsonb,
        1, -- Active
        '{"v2": true}'::jsonb,
        0.50,
        NOW(),
        NOW(),
        NULL,
        false,
        6000,
        15000, -- 15 second timeout
        ARRAY['bidmachine']::varchar[],
        ARRAY[]::varchar[],
        ARRAY[5000]::bigint[]
    ),
    -- Chess - Rewarded Config
    (
        6001,
        'Chess Rewarded Config',
        chess_app_id,
        6, -- Rewarded
        '[]'::jsonb,
        1, -- Active
        '{"v2": true}'::jsonb,
        1.00,
        NOW(),
        NOW(),
        NULL,
        false,
        6001,
        20000, -- 20 second timeout
        ARRAY['bidmachine']::varchar[],
        ARRAY[]::varchar[],
        ARRAY[5001]::bigint[]
    ) ON CONFLICT (id) DO NOTHING;

    -- Create auction configurations for Mahjong
    INSERT INTO auction_configurations (
        id, name, app_id, ad_type, rounds, status, settings, pricefloor,
        created_at, updated_at, segment_id, external_win_notifications, public_uid,
        timeout, demands, bidding, ad_unit_ids
    ) VALUES
    -- Mahjong - Banner Config
    (
        6010,
        'Mahjong Banner Config',
        mahjong_app_id,
        3, -- Banner
        '[]'::jsonb,
        1, -- Active
        '{"v2": true}'::jsonb,
        0.10,
        NOW(),
        NOW(),
        NULL,
        false,
        6010,
        10000, -- 10 second timeout
        ARRAY['bidmachine']::varchar[],
        ARRAY[]::varchar[],
        ARRAY[5010]::bigint[]
    ),
    -- Mahjong - Interstitial Config
    (
        6011,
        'Mahjong Interstitial Config',
        mahjong_app_id,
        1, -- Interstitial
        '[]'::jsonb,
        1, -- Active
        '{"v2": true}'::jsonb,
        0.75,
        NOW(),
        NOW(),
        NULL,
        false,
        6011,
        15000,
        ARRAY['bidmachine']::varchar[],
        ARRAY[]::varchar[],
        ARRAY[5011]::bigint[]
    ),
    -- Mahjong - Rewarded Config
    (
        6012,
        'Mahjong Rewarded Config',
        mahjong_app_id,
        6, -- Rewarded
        '[]'::jsonb,
        1, -- Active
        '{"v2": true}'::jsonb,
        1.50,
        NOW(),
        NOW(),
        NULL,
        false,
        6012,
        20000,
        ARRAY['bidmachine']::varchar[],
        ARRAY[]::varchar[],
        ARRAY[5012]::bigint[]
    ) ON CONFLICT (id) DO NOTHING;

    -- Create auction configurations for Trivial Pursuit
    INSERT INTO auction_configurations (
        id, name, app_id, ad_type, rounds, status, settings, pricefloor,
        created_at, updated_at, segment_id, external_win_notifications, public_uid,
        timeout, demands, bidding, ad_unit_ids
    ) VALUES
    -- Trivial Pursuit - Banner Config
    (
        6020,
        'Trivial Banner Config',
        trivial_app_id,
        3, -- Banner
        '[]'::jsonb,
        1, -- Active
        '{"v2": true}'::jsonb,
        0.20,
        NOW(),
        NOW(),
        NULL,
        false,
        6020,
        10000,
        ARRAY['bidmachine']::varchar[],
        ARRAY[]::varchar[],
        ARRAY[5020, 5021]::bigint[]
    ),
    -- Trivial Pursuit - Interstitial Config
    (
        6021,
        'Trivial Interstitial Config',
        trivial_app_id,
        1, -- Interstitial
        '[]'::jsonb,
        1, -- Active
        '{"v2": true}'::jsonb,
        0.60,
        NOW(),
        NOW(),
        NULL,
        false,
        6021,
        15000,
        ARRAY['bidmachine']::varchar[],
        ARRAY[]::varchar[],
        ARRAY[5022]::bigint[]
    ) ON CONFLICT (id) DO NOTHING;

END $$;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Clean up in reverse order
DELETE FROM auction_configurations WHERE id BETWEEN 6000 AND 6999;
DELETE FROM line_items WHERE id BETWEEN 5000 AND 5999;
DELETE FROM app_demand_profiles WHERE id BETWEEN 4000 AND 4999;
DELETE FROM apps WHERE id BETWEEN 2000 AND 2999;
DELETE FROM demand_source_accounts WHERE id BETWEEN 3000 AND 3999;
DELETE FROM users WHERE id = 1000;

-- +goose StatementEnd
