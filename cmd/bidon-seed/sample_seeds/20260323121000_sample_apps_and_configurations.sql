-- +goose Up
-- +goose StatementBegin

-- Ensure a demo admin user exists with a known ID (1000).
-- If an admin user already exists in the system (e.g. created by init-admin.sh),
-- all seed resources will be assigned to that user instead (see DO $$ block below).
-- Password hash is a placeholder — this account is only used as a fallback owner.
INSERT INTO users (id, email, password_hash, is_admin, created_at, updated_at, public_uid)
VALUES (
    1000,
    'demo@bidon.org',
    'AQIDBAUGBwgJCgsMDQ4PEA$bXTpgPF+hV8Z3JqzQqHFUa0oWxECgJ7LRy4f5Kw9NhM',
    true,
    NOW(),
    NOW(),
    1000
) ON CONFLICT (id) DO NOTHING;

DO $$
DECLARE
    -- Demand source IDs (looked up dynamically)
    bidmachine_id BIGINT;
    applovin_id   BIGINT;
    admob_id      BIGINT;
    unityads_id   BIGINT;
    meta_id       BIGINT;
    adikteev_id   BIGINT;

    -- Owner: use the first existing admin user (e.g. from init-admin.sh),
    -- falling back to the demo admin user (1000) inserted above.
    owner_id BIGINT;

    -- App IDs
    chess_app_id       BIGINT := 2000;
    mahjong_app_id     BIGINT := 2001;
    trivial_app_id     BIGINT := 2002;
    wordpuzzle_app_id  BIGINT := 2003;
    spacerunner_app_id BIGINT := 2004;
    tetris_app_id      BIGINT := 2005;

    -- Demand source account IDs
    bidmachine_account_id BIGINT := 3000;
    applovin_account_id   BIGINT := 3001;
    admob_account_id      BIGINT := 3002;
    unityads_account_id   BIGINT := 3003;
    meta_account_id       BIGINT := 3004;
    adikteev_account_id   BIGINT := 3005;
BEGIN
    -- Resolve owner: prefer an existing admin user, fall back to demo user 1000.
    SELECT id INTO owner_id FROM users WHERE is_admin = true ORDER BY id LIMIT 1;
    IF owner_id IS NULL THEN
        owner_id := 1000;
    END IF;

    -- Look up demand source IDs from the demand_sources seed
    SELECT id INTO bidmachine_id FROM demand_sources WHERE api_key = 'bidmachine';
    SELECT id INTO applovin_id   FROM demand_sources WHERE api_key = 'applovin';
    SELECT id INTO admob_id      FROM demand_sources WHERE api_key = 'admob';
    SELECT id INTO unityads_id   FROM demand_sources WHERE api_key = 'unityads';
    SELECT id INTO meta_id       FROM demand_sources WHERE api_key = 'meta';
    SELECT id INTO adikteev_id   FROM demand_sources WHERE api_key = 'adikteev';

    IF bidmachine_id IS NULL THEN RAISE EXCEPTION 'BidMachine demand source not found. Run demand_sources seed first.'; END IF;
    IF applovin_id   IS NULL THEN RAISE EXCEPTION 'AppLovin demand source not found. Run demand_sources seed first.'; END IF;
    IF admob_id      IS NULL THEN RAISE EXCEPTION 'AdMob demand source not found. Run demand_sources seed first.'; END IF;
    IF unityads_id   IS NULL THEN RAISE EXCEPTION 'Unity Ads demand source not found. Run demand_sources seed first.'; END IF;
    IF meta_id       IS NULL THEN RAISE EXCEPTION 'Meta demand source not found. Run demand_sources seed first.'; END IF;
    IF adikteev_id   IS NULL THEN RAISE EXCEPTION 'Adikteev demand source not found. Run demand_sources seed first.'; END IF;

    -- =========================================================
    -- Demand Source Accounts (all owned by owner_id)
    -- =========================================================
    INSERT INTO demand_source_accounts (
        id, demand_source_id, user_id, type, extra, bidding, is_default, created_at, updated_at, label, public_uid
    ) VALUES
    (
        bidmachine_account_id, bidmachine_id, owner_id,
        'DemandSourceAccount::bidmachine',
        '{"seller_id": "1", "endpoint": "x.appbaqend.com", "mediation_config": ["rewarded", "interstitial", "banner"]}'::jsonb,
        false, false, NOW(), NOW(), 'BidMachine Production', bidmachine_account_id
    ),
    (
        applovin_account_id, applovin_id, owner_id,
        'DemandSourceAccount::applovin',
        '{"sdk_key": "applovin-demo-sdk-key-ABC123XYZ"}'::jsonb,
        false, false, NOW(), NOW(), 'AppLovin MAX', applovin_account_id
    ),
    (
        admob_account_id, admob_id, owner_id,
        'DemandSourceAccount::admob',
        '{}'::jsonb,
        false, false, NOW(), NOW(), 'AdMob Production', admob_account_id
    ),
    (
        unityads_account_id, unityads_id, owner_id,
        'DemandSourceAccount::unityads',
        '{}'::jsonb,
        false, false, NOW(), NOW(), 'Unity Ads Production', unityads_account_id
    ),
    (
        meta_account_id, meta_id, owner_id,
        'DemandSourceAccount::meta',
        '{}'::jsonb,
        false, false, NOW(), NOW(), 'Meta Audience Network', meta_account_id
    ),
    (
        adikteev_account_id, adikteev_id, owner_id,
        'DemandSourceAccount::adikteev',
        '{"endpoint": "rubicon-eu.dsp.adikteev.com", "seller_id":  "1"}'::jsonb,
        true, false, NOW(), NOW(), 'Adikteev Audience Network', adikteev_account_id
    )
    ON CONFLICT (id) DO NOTHING;

    -- =========================================================
    -- Apps (all owned by owner_id)
    -- =========================================================
    INSERT INTO apps (
        id, user_id, platform_id, human_name, package_name, app_key, settings, created_at, updated_at, public_uid
    ) VALUES
    (chess_app_id,       owner_id, 1, 'Chess Master',            'com.demo.chessmaster',    'chess_'       || chess_app_id,       '{}'::jsonb, NOW(), NOW(), chess_app_id),
    (mahjong_app_id,     owner_id, 4, 'Mahjong Quest',           'com.demo.mahjongquest',   'mahjong_'     || mahjong_app_id,     '{}'::jsonb, NOW(), NOW(), mahjong_app_id),
    (trivial_app_id,     owner_id, 1, 'Trivial Pursuit Ultimate', 'com.demo.trivialpursuit', 'trivial_'     || trivial_app_id,     '{}'::jsonb, NOW(), NOW(), trivial_app_id),
    (wordpuzzle_app_id,  owner_id, 4, 'Word Puzzle Pro',         'com.demo.wordpuzzlepro',  'wordpuzzle_'  || wordpuzzle_app_id,  '{}'::jsonb, NOW(), NOW(), wordpuzzle_app_id),
    (spacerunner_app_id, owner_id, 1, 'Space Runner',            'com.demo.spacerunner',    'spacerunner_' || spacerunner_app_id, '{}'::jsonb, NOW(), NOW(), spacerunner_app_id),
    (tetris_app_id,      owner_id, 1, 'Tetris',                  'com.demo.tetris',         'tetris_' || tetris_app_id,           '{}'::jsonb, NOW(), NOW(), tetris_app_id)
    ON CONFLICT (id) DO NOTHING;

    -- =========================================================
    -- App Demand Profiles
    -- Chess:        BidMachine, AppLovin, AdMob
    -- Mahjong:      BidMachine, AppLovin, AdMob
    -- Trivial:      BidMachine, AdMob
    -- Word Puzzle:  Unity Ads, Meta
    -- Space Runner: Unity Ads, Meta
    -- Tetris:       Adikteev
    -- =========================================================
    INSERT INTO app_demand_profiles (
        id, app_id, account_type, account_id, demand_source_id, data, created_at, updated_at, public_uid, enabled
    ) VALUES
    -- Chess Master
    (4000, chess_app_id, 'DemandSourceAccount::BidMachine', bidmachine_account_id, bidmachine_id,
     '{"source_id": "5"}'::jsonb, NOW(), NOW(), 4000, true),
    (4001, chess_app_id, 'DemandSourceAccount::Applovin', applovin_account_id, applovin_id,
     '{"ad_unit_ids": ["chess_applovin_inter", "chess_applovin_rewarded"], "mediator": "Bidon"}'::jsonb, NOW(), NOW(), 4001, true),
    (4002, chess_app_id, 'DemandSourceAccount::Admob', admob_account_id, admob_id,
     '{"app_id": "ca-app-pub-3940256099942544~2001003922"}'::jsonb, NOW(), NOW(), 4002, true),
    -- Mahjong Quest
    (4003, mahjong_app_id, 'DemandSourceAccount::BidMachine', bidmachine_account_id, bidmachine_id,
     '{"source_id": "6"}'::jsonb, NOW(), NOW(), 4003, true),
    (4004, mahjong_app_id, 'DemandSourceAccount::Applovin', applovin_account_id, applovin_id,
     '{"ad_unit_ids": ["mahjong_applovin_banner", "mahjong_applovin_inter"], "mediator": "Bidon"}'::jsonb, NOW(), NOW(), 4004, true),
    (4005, mahjong_app_id, 'DemandSourceAccount::Admob', admob_account_id, admob_id,
     '{"app_id": "ca-app-pub-3940256099942544~3001003931"}'::jsonb, NOW(), NOW(), 4005, true),
    -- Trivial Pursuit
    (4006, trivial_app_id, 'DemandSourceAccount::BidMachine', bidmachine_account_id, bidmachine_id,
     '{"source_id": "7"}'::jsonb, NOW(), NOW(), 4006, true),
    (4007, trivial_app_id, 'DemandSourceAccount::Admob', admob_account_id, admob_id,
     '{"app_id": "ca-app-pub-3940256099942544~4001003940"}'::jsonb, NOW(), NOW(), 4007, true),
    -- Word Puzzle Pro
    (4008, wordpuzzle_app_id, 'DemandSourceAccount::UnityAds', unityads_account_id, unityads_id,
     '{"game_id": 4968857}'::jsonb, NOW(), NOW(), 4008, true),
    (4009, wordpuzzle_app_id, 'DemandSourceAccount::Meta', meta_account_id, meta_id,
     '{"app_id": 987654321}'::jsonb, NOW(), NOW(), 4009, true),
    -- Space Runner
    (4010, spacerunner_app_id, 'DemandSourceAccount::UnityAds', unityads_account_id, unityads_id,
     '{"game_id": 5123456}'::jsonb, NOW(), NOW(), 4010, true),
    (4011, spacerunner_app_id, 'DemandSourceAccount::Meta', meta_account_id, meta_id,
     '{"app_id": 876543210}'::jsonb, NOW(), NOW(), 4011, true),
    -- Tetris
    (4012, tetris_app_id, 'DemandSourceAccount::Adikteev', adikteev_account_id, adikteev_id,
     '{"app_id": 195876}'::jsonb, NOW(), NOW(), 4012, true)
    ON CONFLICT (id) DO NOTHING;

    -- =========================================================
    -- CHESS MASTER
    --
    -- Auction strategy:
    --   Interstitial — hybrid: 3 BidMachine + 2 AppLovin + 3 AdMob waterfall tiers,
    --                          BidMachine & AppLovin also in RTB bidding
    --   Rewarded      — waterfall + single BidMachine bidding stream
    -- =========================================================

    -- 5000-5009: Chess Interstitial line items
    INSERT INTO line_items (
        id, app_id, account_type, account_id, human_name, bid_floor, ad_type, extra,
        created_at, updated_at, width, height, format, public_uid, bidding
    ) VALUES
    -- BidMachine waterfall tiers
    (5000, chess_app_id, 'DemandSourceAccount::bidmachine', bidmachine_account_id,
     'Chess BidMachine Interstitial $0.50', 0.50, 1, '{"placement": "chess_inter_low"}'::jsonb,   NOW(), NOW(), 0, 0, '', 5000, false),
    (5001, chess_app_id, 'DemandSourceAccount::bidmachine', bidmachine_account_id,
     'Chess BidMachine Interstitial $1.00', 1.00, 1, '{"placement": "chess_inter_mid"}'::jsonb,   NOW(), NOW(), 0, 0, '', 5001, false),
    (5002, chess_app_id, 'DemandSourceAccount::bidmachine', bidmachine_account_id,
     'Chess BidMachine Interstitial $2.00', 2.00, 1, '{"placement": "chess_inter_high"}'::jsonb,  NOW(), NOW(), 0, 0, '', 5002, false),
    -- AppLovin waterfall tiers
    (5003, chess_app_id, 'DemandSourceAccount::applovin', applovin_account_id,
     'Chess AppLovin Interstitial $0.40', 0.40, 1, '{"zone_id": "chess_al_inter_low"}'::jsonb,    NOW(), NOW(), 0, 0, '', 5003, false),
    (5004, chess_app_id, 'DemandSourceAccount::applovin', applovin_account_id,
     'Chess AppLovin Interstitial $0.80', 0.80, 1, '{"zone_id": "chess_al_inter_high"}'::jsonb,   NOW(), NOW(), 0, 0, '', 5004, false),
    -- AdMob waterfall tiers
    (5005, chess_app_id, 'DemandSourceAccount::admob', admob_account_id,
     'Chess AdMob Interstitial $0.30', 0.30, 1, '{"ad_unit_id": "ca-app-pub-3940256099942544/1033173712"}'::jsonb, NOW(), NOW(), 0, 0, '', 5005, false),
    (5006, chess_app_id, 'DemandSourceAccount::admob', admob_account_id,
     'Chess AdMob Interstitial $0.60', 0.60, 1, '{"ad_unit_id": "ca-app-pub-3940256099942544/4411468910"}'::jsonb, NOW(), NOW(), 0, 0, '', 5006, false),
    (5007, chess_app_id, 'DemandSourceAccount::admob', admob_account_id,
     'Chess AdMob Interstitial $1.20', 1.20, 1, '{"ad_unit_id": "ca-app-pub-3940256099942544/8691691433"}'::jsonb, NOW(), NOW(), 0, 0, '', 5007, false),
    -- BidMachine & AppLovin RTB bidding
    (5008, chess_app_id, 'DemandSourceAccount::bidmachine', bidmachine_account_id,
     'Chess BidMachine Interstitial [Bidding]', 0.01, 1, '{"placement": "chess_inter_bid"}'::jsonb, NOW(), NOW(), 0, 0, '', 5008, true),
    (5009, chess_app_id, 'DemandSourceAccount::applovin', applovin_account_id,
     'Chess AppLovin Interstitial [Bidding]',   0.01, 1, '{"zone_id": "chess_al_inter_bid"}'::jsonb, NOW(), NOW(), 0, 0, '', 5009, true)
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO auction_configurations (
        id, name, app_id, ad_type, rounds, status, settings, pricefloor,
        created_at, updated_at, segment_id, external_win_notifications, public_uid,
        timeout, demands, bidding, ad_unit_ids
    ) VALUES (
        6000, 'Chess Interstitial Config', chess_app_id, 1, '[]'::jsonb, 1, '{"v2": true}'::jsonb, 0.30,
        NOW(), NOW(), NULL, false, 6000, 15000,
        ARRAY['bidmachine', 'applovin', 'admob']::varchar[],
        ARRAY['bidmachine', 'applovin']::varchar[],
        ARRAY[5000, 5001, 5002, 5003, 5004, 5005, 5006, 5007, 5008, 5009]::bigint[]
    ) ON CONFLICT (id) DO NOTHING;

    -- 5020-5026: Chess Rewarded line items
    INSERT INTO line_items (
        id, app_id, account_type, account_id, human_name, bid_floor, ad_type, extra,
        created_at, updated_at, width, height, format, public_uid, bidding
    ) VALUES
    -- BidMachine waterfall tiers
    (5020, chess_app_id, 'DemandSourceAccount::bidmachine', bidmachine_account_id,
     'Chess BidMachine Rewarded $1.00', 1.00, 6, '{"placement": "chess_rew_low"}'::jsonb,  NOW(), NOW(), 0, 0, '', 5020, false),
    (5021, chess_app_id, 'DemandSourceAccount::bidmachine', bidmachine_account_id,
     'Chess BidMachine Rewarded $2.00', 2.00, 6, '{"placement": "chess_rew_mid"}'::jsonb,  NOW(), NOW(), 0, 0, '', 5021, false),
    (5022, chess_app_id, 'DemandSourceAccount::bidmachine', bidmachine_account_id,
     'Chess BidMachine Rewarded $3.00', 3.00, 6, '{"placement": "chess_rew_high"}'::jsonb, NOW(), NOW(), 0, 0, '', 5022, false),
    -- AdMob waterfall tiers
    (5023, chess_app_id, 'DemandSourceAccount::admob', admob_account_id,
     'Chess AdMob Rewarded $0.80', 0.80, 6, '{"ad_unit_id": "ca-app-pub-3940256099942544/5224354917"}'::jsonb, NOW(), NOW(), 0, 0, '', 5023, false),
    (5024, chess_app_id, 'DemandSourceAccount::admob', admob_account_id,
     'Chess AdMob Rewarded $1.50', 1.50, 6, '{"ad_unit_id": "ca-app-pub-3940256099942544/1712485313"}'::jsonb, NOW(), NOW(), 0, 0, '', 5024, false),
    (5025, chess_app_id, 'DemandSourceAccount::admob', admob_account_id,
     'Chess AdMob Rewarded $2.50', 2.50, 6, '{"ad_unit_id": "ca-app-pub-3940256099942544/9214589741"}'::jsonb, NOW(), NOW(), 0, 0, '', 5025, false),
    -- BidMachine RTB bidding
    (5026, chess_app_id, 'DemandSourceAccount::bidmachine', bidmachine_account_id,
     'Chess BidMachine Rewarded [Bidding]', 0.01, 6, '{"placement": "chess_rew_bid"}'::jsonb, NOW(), NOW(), 0, 0, '', 5026, true)
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO auction_configurations (
        id, name, app_id, ad_type, rounds, status, settings, pricefloor,
        created_at, updated_at, segment_id, external_win_notifications, public_uid,
        timeout, demands, bidding, ad_unit_ids
    ) VALUES (
        6001, 'Chess Rewarded Config', chess_app_id, 6, '[]'::jsonb, 1, '{"v2": true}'::jsonb, 0.80,
        NOW(), NOW(), NULL, false, 6001, 20000,
        ARRAY['bidmachine', 'admob']::varchar[],
        ARRAY['bidmachine']::varchar[],
        ARRAY[5020, 5021, 5022, 5023, 5024, 5025, 5026]::bigint[]
    ) ON CONFLICT (id) DO NOTHING;

    -- =========================================================
    -- MAHJONG QUEST
    --
    -- Auction strategy:
    --   Banner        — hybrid: 2 BidMachine + 2 AppLovin waterfall, BidMachine bidding
    --   Interstitial  — hybrid: 3 AppLovin + 3 AdMob waterfall, AppLovin bidding
    --   Rewarded      — pure waterfall only (no bidding), 3 BidMachine + 3 AdMob tiers
    -- =========================================================

    -- 5040-5044: Mahjong Banner line items
    INSERT INTO line_items (
        id, app_id, account_type, account_id, human_name, bid_floor, ad_type, extra,
        created_at, updated_at, width, height, format, public_uid, bidding
    ) VALUES
    -- BidMachine waterfall tiers
    (5040, mahjong_app_id, 'DemandSourceAccount::bidmachine', bidmachine_account_id,
     'Mahjong BidMachine Banner $0.10', 0.10, 3, '{"placement": "mahjong_banner_low"}'::jsonb,  NOW(), NOW(), 320, 50, 'BANNER', 5040, false),
    (5041, mahjong_app_id, 'DemandSourceAccount::bidmachine', bidmachine_account_id,
     'Mahjong BidMachine Banner $0.20', 0.20, 3, '{"placement": "mahjong_banner_high"}'::jsonb, NOW(), NOW(), 320, 50, 'BANNER', 5041, false),
    -- AppLovin waterfall tiers
    (5042, mahjong_app_id, 'DemandSourceAccount::applovin', applovin_account_id,
     'Mahjong AppLovin Banner $0.08', 0.08, 3, '{"zone_id": "mahjong_al_banner_low"}'::jsonb,  NOW(), NOW(), 320, 50, 'BANNER', 5042, false),
    (5043, mahjong_app_id, 'DemandSourceAccount::applovin', applovin_account_id,
     'Mahjong AppLovin Banner $0.15', 0.15, 3, '{"zone_id": "mahjong_al_banner_high"}'::jsonb, NOW(), NOW(), 320, 50, 'BANNER', 5043, false),
    -- BidMachine RTB bidding
    (5044, mahjong_app_id, 'DemandSourceAccount::bidmachine', bidmachine_account_id,
     'Mahjong BidMachine Banner [Bidding]', 0.01, 3, '{"placement": "mahjong_banner_bid"}'::jsonb, NOW(), NOW(), 320, 50, 'BANNER', 5044, true)
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO auction_configurations (
        id, name, app_id, ad_type, rounds, status, settings, pricefloor,
        created_at, updated_at, segment_id, external_win_notifications, public_uid,
        timeout, demands, bidding, ad_unit_ids
    ) VALUES (
        6010, 'Mahjong Banner Config', mahjong_app_id, 3, '[]'::jsonb, 1, '{"v2": true}'::jsonb, 0.08,
        NOW(), NOW(), NULL, false, 6010, 10000,
        ARRAY['bidmachine', 'applovin']::varchar[],
        ARRAY['bidmachine']::varchar[],
        ARRAY[5040, 5041, 5042, 5043, 5044]::bigint[]
    ) ON CONFLICT (id) DO NOTHING;

    -- 5060-5066: Mahjong Interstitial line items
    INSERT INTO line_items (
        id, app_id, account_type, account_id, human_name, bid_floor, ad_type, extra,
        created_at, updated_at, width, height, format, public_uid, bidding
    ) VALUES
    -- AppLovin waterfall tiers
    (5060, mahjong_app_id, 'DemandSourceAccount::applovin', applovin_account_id,
     'Mahjong AppLovin Interstitial $0.50', 0.50, 1, '{"zone_id": "mahjong_al_inter_low"}'::jsonb,  NOW(), NOW(), 0, 0, '', 5060, false),
    (5061, mahjong_app_id, 'DemandSourceAccount::applovin', applovin_account_id,
     'Mahjong AppLovin Interstitial $1.00', 1.00, 1, '{"zone_id": "mahjong_al_inter_mid"}'::jsonb,  NOW(), NOW(), 0, 0, '', 5061, false),
    (5062, mahjong_app_id, 'DemandSourceAccount::applovin', applovin_account_id,
     'Mahjong AppLovin Interstitial $1.75', 1.75, 1, '{"zone_id": "mahjong_al_inter_high"}'::jsonb, NOW(), NOW(), 0, 0, '', 5062, false),
    -- AdMob waterfall tiers
    (5063, mahjong_app_id, 'DemandSourceAccount::admob', admob_account_id,
     'Mahjong AdMob Interstitial $0.40', 0.40, 1, '{"ad_unit_id": "ca-app-pub-3940256099942544/1033173712"}'::jsonb, NOW(), NOW(), 0, 0, '', 5063, false),
    (5064, mahjong_app_id, 'DemandSourceAccount::admob', admob_account_id,
     'Mahjong AdMob Interstitial $0.80', 0.80, 1, '{"ad_unit_id": "ca-app-pub-3940256099942544/4411468910"}'::jsonb, NOW(), NOW(), 0, 0, '', 5064, false),
    (5065, mahjong_app_id, 'DemandSourceAccount::admob', admob_account_id,
     'Mahjong AdMob Interstitial $1.50', 1.50, 1, '{"ad_unit_id": "ca-app-pub-3940256099942544/8691691433"}'::jsonb, NOW(), NOW(), 0, 0, '', 5065, false),
    -- AppLovin RTB bidding
    (5066, mahjong_app_id, 'DemandSourceAccount::applovin', applovin_account_id,
     'Mahjong AppLovin Interstitial [Bidding]', 0.01, 1, '{"zone_id": "mahjong_al_inter_bid"}'::jsonb, NOW(), NOW(), 0, 0, '', 5066, true)
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO auction_configurations (
        id, name, app_id, ad_type, rounds, status, settings, pricefloor,
        created_at, updated_at, segment_id, external_win_notifications, public_uid,
        timeout, demands, bidding, ad_unit_ids
    ) VALUES (
        6011, 'Mahjong Interstitial Config', mahjong_app_id, 1, '[]'::jsonb, 1, '{"v2": true}'::jsonb, 0.40,
        NOW(), NOW(), NULL, false, 6011, 15000,
        ARRAY['applovin', 'admob']::varchar[],
        ARRAY['applovin']::varchar[],
        ARRAY[5060, 5061, 5062, 5063, 5064, 5065, 5066]::bigint[]
    ) ON CONFLICT (id) DO NOTHING;

    -- 5080-5085: Mahjong Rewarded — pure waterfall, no bidding
    INSERT INTO line_items (
        id, app_id, account_type, account_id, human_name, bid_floor, ad_type, extra,
        created_at, updated_at, width, height, format, public_uid, bidding
    ) VALUES
    -- BidMachine waterfall tiers
    (5080, mahjong_app_id, 'DemandSourceAccount::bidmachine', bidmachine_account_id,
     'Mahjong BidMachine Rewarded $1.50', 1.50, 6, '{"placement": "mahjong_rew_low"}'::jsonb,  NOW(), NOW(), 0, 0, '', 5080, false),
    (5081, mahjong_app_id, 'DemandSourceAccount::bidmachine', bidmachine_account_id,
     'Mahjong BidMachine Rewarded $2.50', 2.50, 6, '{"placement": "mahjong_rew_mid"}'::jsonb,  NOW(), NOW(), 0, 0, '', 5081, false),
    (5082, mahjong_app_id, 'DemandSourceAccount::bidmachine', bidmachine_account_id,
     'Mahjong BidMachine Rewarded $4.00', 4.00, 6, '{"placement": "mahjong_rew_high"}'::jsonb, NOW(), NOW(), 0, 0, '', 5082, false),
    -- AdMob waterfall tiers
    (5083, mahjong_app_id, 'DemandSourceAccount::admob', admob_account_id,
     'Mahjong AdMob Rewarded $1.20', 1.20, 6, '{"ad_unit_id": "ca-app-pub-3940256099942544/5224354917"}'::jsonb, NOW(), NOW(), 0, 0, '', 5083, false),
    (5084, mahjong_app_id, 'DemandSourceAccount::admob', admob_account_id,
     'Mahjong AdMob Rewarded $2.00', 2.00, 6, '{"ad_unit_id": "ca-app-pub-3940256099942544/1712485313"}'::jsonb, NOW(), NOW(), 0, 0, '', 5084, false),
    (5085, mahjong_app_id, 'DemandSourceAccount::admob', admob_account_id,
     'Mahjong AdMob Rewarded $3.00', 3.00, 6, '{"ad_unit_id": "ca-app-pub-3940256099942544/9214589741"}'::jsonb, NOW(), NOW(), 0, 0, '', 5085, false)
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO auction_configurations (
        id, name, app_id, ad_type, rounds, status, settings, pricefloor,
        created_at, updated_at, segment_id, external_win_notifications, public_uid,
        timeout, demands, bidding, ad_unit_ids
    ) VALUES (
        6012, 'Mahjong Rewarded Config', mahjong_app_id, 6, '[]'::jsonb, 1, '{"v2": true}'::jsonb, 1.20,
        NOW(), NOW(), NULL, false, 6012, 20000,
        ARRAY['bidmachine', 'admob']::varchar[],
        ARRAY[]::varchar[],
        ARRAY[5080, 5081, 5082, 5083, 5084, 5085]::bigint[]
    ) ON CONFLICT (id) DO NOTHING;

    -- =========================================================
    -- TRIVIAL PURSUIT ULTIMATE
    --
    -- Auction strategy:
    --   Banner        — mixed formats (leaderboard + MREC), hybrid with BidMachine bidding
    --   Interstitial  — hybrid: 3 BidMachine + 2 AdMob waterfall, BidMachine bidding
    -- =========================================================

    -- 5100-5105: Trivial Banner — mixed ad formats, BidMachine bidding
    INSERT INTO line_items (
        id, app_id, account_type, account_id, human_name, bid_floor, ad_type, extra,
        created_at, updated_at, width, height, format, public_uid, bidding
    ) VALUES
    -- BidMachine leaderboard waterfall tiers
    (5100, trivial_app_id, 'DemandSourceAccount::bidmachine', bidmachine_account_id,
     'Trivial BidMachine Leaderboard $0.20', 0.20, 3, '{"placement": "trivial_leader_low"}'::jsonb,  NOW(), NOW(), 728, 90, 'LEADERBOARD', 5100, false),
    (5101, trivial_app_id, 'DemandSourceAccount::bidmachine', bidmachine_account_id,
     'Trivial BidMachine Leaderboard $0.40', 0.40, 3, '{"placement": "trivial_leader_high"}'::jsonb, NOW(), NOW(), 728, 90, 'LEADERBOARD', 5101, false),
    -- AdMob MREC waterfall tiers
    (5102, trivial_app_id, 'DemandSourceAccount::admob', admob_account_id,
     'Trivial AdMob MREC $0.15', 0.15, 3, '{"ad_unit_id": "ca-app-pub-3940256099942544/6300978111"}'::jsonb, NOW(), NOW(), 300, 250, 'MREC', 5102, false),
    (5103, trivial_app_id, 'DemandSourceAccount::admob', admob_account_id,
     'Trivial AdMob MREC $0.30', 0.30, 3, '{"ad_unit_id": "ca-app-pub-3940256099942544/2247696110"}'::jsonb, NOW(), NOW(), 300, 250, 'MREC', 5103, false),
    (5104, trivial_app_id, 'DemandSourceAccount::admob', admob_account_id,
     'Trivial AdMob MREC $0.50', 0.50, 3, '{"ad_unit_id": "ca-app-pub-3940256099942544/9214589741"}'::jsonb, NOW(), NOW(), 300, 250, 'MREC', 5104, false),
    -- BidMachine RTB bidding
    (5105, trivial_app_id, 'DemandSourceAccount::bidmachine', bidmachine_account_id,
     'Trivial BidMachine Banner [Bidding]', 0.01, 3, '{"placement": "trivial_banner_bid"}'::jsonb, NOW(), NOW(), 320, 50, 'BANNER', 5105, true)
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO auction_configurations (
        id, name, app_id, ad_type, rounds, status, settings, pricefloor,
        created_at, updated_at, segment_id, external_win_notifications, public_uid,
        timeout, demands, bidding, ad_unit_ids
    ) VALUES (
        6020, 'Trivial Banner Config', trivial_app_id, 3, '[]'::jsonb, 1, '{"v2": true}'::jsonb, 0.15,
        NOW(), NOW(), NULL, false, 6020, 10000,
        ARRAY['bidmachine', 'admob']::varchar[],
        ARRAY['bidmachine']::varchar[],
        ARRAY[5100, 5101, 5102, 5103, 5104, 5105]::bigint[]
    ) ON CONFLICT (id) DO NOTHING;

    -- 5120-5125: Trivial Interstitial — 3 BidMachine + 2 AdMob waterfall, BidMachine bidding
    INSERT INTO line_items (
        id, app_id, account_type, account_id, human_name, bid_floor, ad_type, extra,
        created_at, updated_at, width, height, format, public_uid, bidding
    ) VALUES
    -- BidMachine waterfall tiers
    (5120, trivial_app_id, 'DemandSourceAccount::bidmachine', bidmachine_account_id,
     'Trivial BidMachine Interstitial $0.60', 0.60, 1, '{"placement": "trivial_inter_low"}'::jsonb,  NOW(), NOW(), 0, 0, '', 5120, false),
    (5121, trivial_app_id, 'DemandSourceAccount::bidmachine', bidmachine_account_id,
     'Trivial BidMachine Interstitial $1.20', 1.20, 1, '{"placement": "trivial_inter_mid"}'::jsonb,  NOW(), NOW(), 0, 0, '', 5121, false),
    (5122, trivial_app_id, 'DemandSourceAccount::bidmachine', bidmachine_account_id,
     'Trivial BidMachine Interstitial $2.00', 2.00, 1, '{"placement": "trivial_inter_high"}'::jsonb, NOW(), NOW(), 0, 0, '', 5122, false),
    -- AdMob waterfall tiers
    (5123, trivial_app_id, 'DemandSourceAccount::admob', admob_account_id,
     'Trivial AdMob Interstitial $0.50', 0.50, 1, '{"ad_unit_id": "ca-app-pub-3940256099942544/1033173712"}'::jsonb, NOW(), NOW(), 0, 0, '', 5123, false),
    (5124, trivial_app_id, 'DemandSourceAccount::admob', admob_account_id,
     'Trivial AdMob Interstitial $1.00', 1.00, 1, '{"ad_unit_id": "ca-app-pub-3940256099942544/4411468910"}'::jsonb, NOW(), NOW(), 0, 0, '', 5124, false),
    -- BidMachine RTB bidding
    (5125, trivial_app_id, 'DemandSourceAccount::bidmachine', bidmachine_account_id,
     'Trivial BidMachine Interstitial [Bidding]', 0.01, 1, '{"placement": "trivial_inter_bid"}'::jsonb, NOW(), NOW(), 0, 0, '', 5125, true)
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO auction_configurations (
        id, name, app_id, ad_type, rounds, status, settings, pricefloor,
        created_at, updated_at, segment_id, external_win_notifications, public_uid,
        timeout, demands, bidding, ad_unit_ids
    ) VALUES (
        6021, 'Trivial Interstitial Config', trivial_app_id, 1, '[]'::jsonb, 1, '{"v2": true}'::jsonb, 0.50,
        NOW(), NOW(), NULL, false, 6021, 15000,
        ARRAY['bidmachine', 'admob']::varchar[],
        ARRAY['bidmachine']::varchar[],
        ARRAY[5120, 5121, 5122, 5123, 5124, 5125]::bigint[]
    ) ON CONFLICT (id) DO NOTHING;

    -- =========================================================
    -- WORD PUZZLE PRO
    --
    -- Auction strategy:
    --   Banner   — bidding-first: 2 Meta waterfall tiers + Meta RTB bidding
    --   Rewarded — full hybrid: 2 UnityAds + 2 Meta waterfall, both in RTB bidding
    -- =========================================================

    -- 5140-5142: Word Puzzle Banner — Meta waterfall + bidding
    INSERT INTO line_items (
        id, app_id, account_type, account_id, human_name, bid_floor, ad_type, extra,
        created_at, updated_at, width, height, format, public_uid, bidding
    ) VALUES
    -- Meta waterfall tiers
    (5140, wordpuzzle_app_id, 'DemandSourceAccount::meta', meta_account_id,
     'Word Puzzle Meta Banner $0.15', 0.15, 3, '{"placement_id": "987654321_banner_low"}'::jsonb,  NOW(), NOW(), 320, 50, 'BANNER', 5140, false),
    (5141, wordpuzzle_app_id, 'DemandSourceAccount::meta', meta_account_id,
     'Word Puzzle Meta Banner $0.30', 0.30, 3, '{"placement_id": "987654321_banner_high"}'::jsonb, NOW(), NOW(), 320, 50, 'BANNER', 5141, false),
    -- Meta RTB bidding
    (5142, wordpuzzle_app_id, 'DemandSourceAccount::meta', meta_account_id,
     'Word Puzzle Meta Banner [Bidding]', 0.01, 3, '{"placement_id": "987654321_banner_bid"}'::jsonb, NOW(), NOW(), 320, 50, 'BANNER', 5142, true)
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO auction_configurations (
        id, name, app_id, ad_type, rounds, status, settings, pricefloor,
        created_at, updated_at, segment_id, external_win_notifications, public_uid,
        timeout, demands, bidding, ad_unit_ids
    ) VALUES (
        6030, 'Word Puzzle Banner Config', wordpuzzle_app_id, 3, '[]'::jsonb, 1, '{"v2": true}'::jsonb, 0.15,
        NOW(), NOW(), NULL, false, 6030, 10000,
        ARRAY['meta']::varchar[],
        ARRAY['meta']::varchar[],
        ARRAY[5140, 5141, 5142]::bigint[]
    ) ON CONFLICT (id) DO NOTHING;

    -- 5160-5165: Word Puzzle Rewarded — UnityAds + Meta waterfall, both in bidding
    INSERT INTO line_items (
        id, app_id, account_type, account_id, human_name, bid_floor, ad_type, extra,
        created_at, updated_at, width, height, format, public_uid, bidding
    ) VALUES
    -- UnityAds waterfall tiers
    (5160, wordpuzzle_app_id, 'DemandSourceAccount::unityads', unityads_account_id,
     'Word Puzzle UnityAds Rewarded $0.90', 0.90, 6, '{"placement_id": "rewardedVideo_low"}'::jsonb,  NOW(), NOW(), 0, 0, '', 5160, false),
    (5161, wordpuzzle_app_id, 'DemandSourceAccount::unityads', unityads_account_id,
     'Word Puzzle UnityAds Rewarded $1.80', 1.80, 6, '{"placement_id": "rewardedVideo_high"}'::jsonb, NOW(), NOW(), 0, 0, '', 5161, false),
    -- Meta waterfall tiers
    (5162, wordpuzzle_app_id, 'DemandSourceAccount::meta', meta_account_id,
     'Word Puzzle Meta Rewarded $0.75', 0.75, 6, '{"placement_id": "987654321_rew_low"}'::jsonb,  NOW(), NOW(), 0, 0, '', 5162, false),
    (5163, wordpuzzle_app_id, 'DemandSourceAccount::meta', meta_account_id,
     'Word Puzzle Meta Rewarded $1.50', 1.50, 6, '{"placement_id": "987654321_rew_high"}'::jsonb, NOW(), NOW(), 0, 0, '', 5163, false),
    -- UnityAds & Meta RTB bidding
    (5164, wordpuzzle_app_id, 'DemandSourceAccount::unityads', unityads_account_id,
     'Word Puzzle UnityAds Rewarded [Bidding]', 0.01, 6, '{"placement_id": "rewardedVideo_bid"}'::jsonb, NOW(), NOW(), 0, 0, '', 5164, true),
    (5165, wordpuzzle_app_id, 'DemandSourceAccount::meta', meta_account_id,
     'Word Puzzle Meta Rewarded [Bidding]',     0.01, 6, '{"placement_id": "987654321_rew_bid"}'::jsonb, NOW(), NOW(), 0, 0, '', 5165, true)
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO auction_configurations (
        id, name, app_id, ad_type, rounds, status, settings, pricefloor,
        created_at, updated_at, segment_id, external_win_notifications, public_uid,
        timeout, demands, bidding, ad_unit_ids
    ) VALUES (
        6031, 'Word Puzzle Rewarded Config', wordpuzzle_app_id, 6, '[]'::jsonb, 1, '{"v2": true}'::jsonb, 0.75,
        NOW(), NOW(), NULL, false, 6031, 20000,
        ARRAY['unityads', 'meta']::varchar[],
        ARRAY['unityads', 'meta']::varchar[],
        ARRAY[5160, 5161, 5162, 5163, 5164, 5165]::bigint[]
    ) ON CONFLICT (id) DO NOTHING;

    -- =========================================================
    -- SPACE RUNNER
    --
    -- Auction strategy:
    --   Interstitial — full hybrid: 2 UnityAds + 2 Meta waterfall, both in RTB bidding
    --   Rewarded     — waterfall + bidding: 3 UnityAds waterfall tiers, UnityAds bidding
    -- =========================================================

    -- 5180-5185: Space Runner Interstitial — UnityAds + Meta waterfall, both in bidding
    INSERT INTO line_items (
        id, app_id, account_type, account_id, human_name, bid_floor, ad_type, extra,
        created_at, updated_at, width, height, format, public_uid, bidding
    ) VALUES
    -- UnityAds waterfall tiers
    (5180, spacerunner_app_id, 'DemandSourceAccount::unityads', unityads_account_id,
     'Space Runner UnityAds Interstitial $0.70', 0.70, 1, '{"placement_id": "Interstitial_Android_low"}'::jsonb,  NOW(), NOW(), 0, 0, '', 5180, false),
    (5181, spacerunner_app_id, 'DemandSourceAccount::unityads', unityads_account_id,
     'Space Runner UnityAds Interstitial $1.40', 1.40, 1, '{"placement_id": "Interstitial_Android_high"}'::jsonb, NOW(), NOW(), 0, 0, '', 5181, false),
    -- Meta waterfall tiers
    (5182, spacerunner_app_id, 'DemandSourceAccount::meta', meta_account_id,
     'Space Runner Meta Interstitial $0.55', 0.55, 1, '{"placement_id": "876543210_inter_low"}'::jsonb,  NOW(), NOW(), 0, 0, '', 5182, false),
    (5183, spacerunner_app_id, 'DemandSourceAccount::meta', meta_account_id,
     'Space Runner Meta Interstitial $1.10', 1.10, 1, '{"placement_id": "876543210_inter_high"}'::jsonb, NOW(), NOW(), 0, 0, '', 5183, false),
    -- UnityAds & Meta RTB bidding
    (5184, spacerunner_app_id, 'DemandSourceAccount::unityads', unityads_account_id,
     'Space Runner UnityAds Interstitial [Bidding]', 0.01, 1, '{"placement_id": "Interstitial_Android_bid"}'::jsonb, NOW(), NOW(), 0, 0, '', 5184, true),
    (5185, spacerunner_app_id, 'DemandSourceAccount::meta', meta_account_id,
     'Space Runner Meta Interstitial [Bidding]',     0.01, 1, '{"placement_id": "876543210_inter_bid"}'::jsonb,     NOW(), NOW(), 0, 0, '', 5185, true)
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO auction_configurations (
        id, name, app_id, ad_type, rounds, status, settings, pricefloor,
        created_at, updated_at, segment_id, external_win_notifications, public_uid,
        timeout, demands, bidding, ad_unit_ids
    ) VALUES (
        6040, 'Space Runner Interstitial Config', spacerunner_app_id, 1, '[]'::jsonb, 1, '{"v2": true}'::jsonb, 0.55,
        NOW(), NOW(), NULL, false, 6040, 15000,
        ARRAY['unityads', 'meta']::varchar[],
        ARRAY['unityads', 'meta']::varchar[],
        ARRAY[5180, 5181, 5182, 5183, 5184, 5185]::bigint[]
    ) ON CONFLICT (id) DO NOTHING;

    -- 5200-5203: Space Runner Rewarded — 3 UnityAds waterfall tiers + UnityAds bidding
    INSERT INTO line_items (
        id, app_id, account_type, account_id, human_name, bid_floor, ad_type, extra,
        created_at, updated_at, width, height, format, public_uid, bidding
    ) VALUES
    -- UnityAds waterfall tiers
    (5200, spacerunner_app_id, 'DemandSourceAccount::unityads', unityads_account_id,
     'Space Runner UnityAds Rewarded $1.10', 1.10, 6, '{"placement_id": "rewardedVideo_low"}'::jsonb,  NOW(), NOW(), 0, 0, '', 5200, false),
    (5201, spacerunner_app_id, 'DemandSourceAccount::unityads', unityads_account_id,
     'Space Runner UnityAds Rewarded $2.20', 2.20, 6, '{"placement_id": "rewardedVideo_mid"}'::jsonb,  NOW(), NOW(), 0, 0, '', 5201, false),
    (5202, spacerunner_app_id, 'DemandSourceAccount::unityads', unityads_account_id,
     'Space Runner UnityAds Rewarded $3.50', 3.50, 6, '{"placement_id": "rewardedVideo_high"}'::jsonb, NOW(), NOW(), 0, 0, '', 5202, false),
    -- UnityAds RTB bidding
    (5203, spacerunner_app_id, 'DemandSourceAccount::unityads', unityads_account_id,
     'Space Runner UnityAds Rewarded [Bidding]', 0.01, 6, '{"placement_id": "rewardedVideo_bid"}'::jsonb, NOW(), NOW(), 0, 0, '', 5203, true)
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO auction_configurations (
        id, name, app_id, ad_type, rounds, status, settings, pricefloor,
        created_at, updated_at, segment_id, external_win_notifications, public_uid,
        timeout, demands, bidding, ad_unit_ids
    ) VALUES (
        6041, 'Space Runner Rewarded Config', spacerunner_app_id, 6, '[]'::jsonb, 1, '{"v2": true}'::jsonb, 1.10,
        NOW(), NOW(), NULL, false, 6041, 20000,
        ARRAY['unityads']::varchar[],
        ARRAY['unityads']::varchar[],
        ARRAY[5200, 5201, 5202, 5203]::bigint[]
    ) ON CONFLICT (id) DO NOTHING;

    -- =========================================================
    -- TETRIS
    --
    -- Auction strategy:
    --   Banner        — Adikteev bidding
    -- =========================================================
    INSERT INTO line_items (
        id, app_id, account_type, account_id, human_name, bid_floor, ad_type, extra,
        created_at, updated_at, width, height, format, public_uid, bidding
    ) VALUES
    (5301, tetris_app_id, 'DemandSourceAccount::adikteev', adikteev_account_id,
     'Tetris Adikteev Banner [Bidding]', 0.01, 3, '{"placement_id": "4567822365_banner_bid"}'::jsonb, NOW(), NOW(), 320, 480, 'BANNER', 5301, true)
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO auction_configurations (
        id, name, app_id, ad_type, rounds, status, settings, pricefloor,
        created_at, updated_at, segment_id, external_win_notifications, public_uid,
        timeout, demands, bidding, ad_unit_ids
    ) VALUES (
    6050, 'Tetris Banner Auction', tetris_app_id, 3, '[]'::jsonb, 1, '{"v2": true}'::jsonb, 0.15,
    NOW(), NOW(), NULL, false, 6050, 10000,
    ARRAY['adikteev']::varchar[],
    ARRAY['adikteev']::varchar[],
    ARRAY[5301]::bigint[]
    ) ON CONFLICT (id) DO NOTHING;

END $$;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Delete dependents by app_id so any records created outside the seed ID ranges are also removed.
DELETE FROM auction_configurations WHERE app_id  BETWEEN 2000 AND 2099;
DELETE FROM line_items              WHERE app_id  BETWEEN 2000 AND 2099;
DELETE FROM app_demand_profiles     WHERE app_id  BETWEEN 2000 AND 2099;
DELETE FROM apps                    WHERE id      BETWEEN 2000 AND 2099;
DELETE FROM demand_source_accounts  WHERE id      BETWEEN 3000 AND 3099;
-- Only remove user 1000 if it was the fallback demo user; never remove a real admin user.
-- Clear api_keys owned by user 1000 first to satisfy the FK constraint.
DELETE FROM api_keys WHERE user_id = 1000;
DELETE FROM users    WHERE id = 1000 AND email = 'demo@bidon.org';

-- +goose StatementEnd
