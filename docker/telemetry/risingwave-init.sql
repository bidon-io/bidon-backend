-- Live funnel for the telemetry POC.
-- Issue 4 owns the compose services that apply this file.
-- Issue 5 added impression and notice sources and extended funnel_5m.
--
-- One source per protobuf message. A source decodes every record on
-- telemetry-events with one descriptor, so views filter on the
-- protobuf_message header. Joins are (app_id, auction_id).
--
-- Notice convention: each attempt emits win_notice_sent, billing_notice_sent,
-- or loss_notice_sent (TimeoutURL is loss_notice_sent). On failure the same
-- attempt also emits notice_delivery_failed, and *_sent.success is false.
-- funnel_5m counts billing_notice_sent where success is true.
-- Counts are raw rows. sampling_rate is 1.0 until sampling lands.

CREATE SOURCE IF NOT EXISTS src_auction_request_received
INCLUDE header 'protobuf_message' AS msg_type
WITH (
    connector = 'kafka',
    topic = 'telemetry-events',
    properties.bootstrap.server = 'redpanda:9092',
    scan.startup.mode = 'earliest'
) FORMAT PLAIN ENCODE PROTOBUF (
    message = 'org.bidon.telemetry.v1.AuctionRequestReceived',
    schema.registry = 'http://redpanda:8081',
    schema.registry.name.strategy = 'topic_record_name_strategy'
);

CREATE SOURCE IF NOT EXISTS src_auction_completed
INCLUDE header 'protobuf_message' AS msg_type
WITH (
    connector = 'kafka',
    topic = 'telemetry-events',
    properties.bootstrap.server = 'redpanda:9092',
    scan.startup.mode = 'earliest'
) FORMAT PLAIN ENCODE PROTOBUF (
    message = 'org.bidon.telemetry.v1.AuctionCompleted',
    schema.registry = 'http://redpanda:8081',
    schema.registry.name.strategy = 'topic_record_name_strategy'
);

CREATE SOURCE IF NOT EXISTS src_dsp_response_received
INCLUDE header 'protobuf_message' AS msg_type
WITH (
    connector = 'kafka',
    topic = 'telemetry-events',
    properties.bootstrap.server = 'redpanda:9092',
    scan.startup.mode = 'earliest'
) FORMAT PLAIN ENCODE PROTOBUF (
    message = 'org.bidon.telemetry.v1.DspResponseReceived',
    schema.registry = 'http://redpanda:8081',
    schema.registry.name.strategy = 'topic_record_name_strategy'
);

CREATE SOURCE IF NOT EXISTS src_ad_impression
INCLUDE header 'protobuf_message' AS msg_type
WITH (
    connector = 'kafka',
    topic = 'telemetry-events',
    properties.bootstrap.server = 'redpanda:9092',
    scan.startup.mode = 'earliest'
) FORMAT PLAIN ENCODE PROTOBUF (
    message = 'org.bidon.telemetry.v1.AdImpression',
    schema.registry = 'http://redpanda:8081',
    schema.registry.name.strategy = 'topic_record_name_strategy'
);

CREATE SOURCE IF NOT EXISTS src_billing_notice_sent
INCLUDE header 'protobuf_message' AS msg_type
WITH (
    connector = 'kafka',
    topic = 'telemetry-events',
    properties.bootstrap.server = 'redpanda:9092',
    scan.startup.mode = 'earliest'
) FORMAT PLAIN ENCODE PROTOBUF (
    message = 'org.bidon.telemetry.v1.BillingNoticeSent',
    schema.registry = 'http://redpanda:8081',
    schema.registry.name.strategy = 'topic_record_name_strategy'
);

CREATE SOURCE IF NOT EXISTS src_win_notice_sent
INCLUDE header 'protobuf_message' AS msg_type
WITH (
    connector = 'kafka',
    topic = 'telemetry-events',
    properties.bootstrap.server = 'redpanda:9092',
    scan.startup.mode = 'earliest'
) FORMAT PLAIN ENCODE PROTOBUF (
    message = 'org.bidon.telemetry.v1.WinNoticeSent',
    schema.registry = 'http://redpanda:8081',
    schema.registry.name.strategy = 'topic_record_name_strategy'
);

CREATE SOURCE IF NOT EXISTS src_loss_notice_sent
INCLUDE header 'protobuf_message' AS msg_type
WITH (
    connector = 'kafka',
    topic = 'telemetry-events',
    properties.bootstrap.server = 'redpanda:9092',
    scan.startup.mode = 'earliest'
) FORMAT PLAIN ENCODE PROTOBUF (
    message = 'org.bidon.telemetry.v1.LossNoticeSent',
    schema.registry = 'http://redpanda:8081',
    schema.registry.name.strategy = 'topic_record_name_strategy'
);

CREATE MATERIALIZED VIEW IF NOT EXISTS auction_requests AS
SELECT (envelope).app_id AS app_id, (envelope).auction_id AS auction_id,
       (envelope).event_ts AS event_ts, requested_price_floor
FROM src_auction_request_received
WHERE convert_from(msg_type, 'utf8') = 'org.bidon.telemetry.v1.AuctionRequestReceived';

CREATE MATERIALIZED VIEW IF NOT EXISTS auctions_completed AS
SELECT (envelope).app_id AS app_id, (envelope).auction_id AS auction_id,
       (envelope).event_ts AS event_ts, winner_dsp, price, price_floor,
       participant_count, total_latency_ms, error_code
FROM src_auction_completed
WHERE convert_from(msg_type, 'utf8') = 'org.bidon.telemetry.v1.AuctionCompleted';

CREATE MATERIALIZED VIEW IF NOT EXISTS dsp_responses AS
SELECT (envelope).app_id AS app_id, (envelope).auction_id AS auction_id,
       (envelope).event_ts AS event_ts, dsp, outcome, http_status, latency_ms,
       price, nbr, creative_id, crtype
FROM src_dsp_response_received
WHERE convert_from(msg_type, 'utf8') = 'org.bidon.telemetry.v1.DspResponseReceived';

CREATE MATERIALIZED VIEW IF NOT EXISTS ad_impressions AS
SELECT (envelope).app_id AS app_id, (envelope).auction_id AS auction_id,
       (envelope).event_ts AS event_ts, dsp, price
FROM src_ad_impression
WHERE convert_from(msg_type, 'utf8') = 'org.bidon.telemetry.v1.AdImpression';

CREATE MATERIALIZED VIEW IF NOT EXISTS billing_notices AS
SELECT (envelope).app_id AS app_id, (envelope).auction_id AS auction_id,
       (envelope).event_ts AS event_ts, dsp, success, http_status, retry_count
FROM src_billing_notice_sent
WHERE convert_from(msg_type, 'utf8') = 'org.bidon.telemetry.v1.BillingNoticeSent';

CREATE MATERIALIZED VIEW IF NOT EXISTS win_notices AS
SELECT (envelope).app_id AS app_id, (envelope).auction_id AS auction_id,
       (envelope).event_ts AS event_ts, dsp, success, http_status
FROM src_win_notice_sent
WHERE convert_from(msg_type, 'utf8') = 'org.bidon.telemetry.v1.WinNoticeSent';

CREATE MATERIALIZED VIEW IF NOT EXISTS loss_notices AS
SELECT (envelope).app_id AS app_id, (envelope).auction_id AS auction_id,
       (envelope).event_ts AS event_ts, dsp, success, http_status
FROM src_loss_notice_sent
WHERE convert_from(msg_type, 'utf8') = 'org.bidon.telemetry.v1.LossNoticeSent';

CREATE MATERIALIZED VIEW IF NOT EXISTS auctions_in_flight AS
SELECT r.app_id, r.auction_id, r.event_ts AS started_ts
FROM auction_requests r
LEFT JOIN auctions_completed c
  ON r.app_id = c.app_id AND r.auction_id = c.auction_id
WHERE c.auction_id IS NULL;

CREATE MATERIALIZED VIEW IF NOT EXISTS dsp_outcomes_5m AS
SELECT window_start, dsp, outcome, COUNT(*) AS n
FROM TUMBLE(dsp_responses, to_timestamp(event_ts / 1000.0), INTERVAL '5 minutes')
GROUP BY window_start, dsp, outcome;

-- billing_ok is billing_notice_sent with success. Impressions are ad_impression
-- rows, synthesized by /v2/show until the SDK sends the client event.
CREATE MATERIALIZED VIEW IF NOT EXISTS funnel_5m AS
SELECT
    window_start,
    COUNT(DISTINCT r.auction_id) AS requests,
    COUNT(DISTINCT c.auction_id) AS completed,
    COUNT(DISTINCT c.auction_id) FILTER (WHERE c.winner_dsp <> '') AS server_fills,
    COUNT(DISTINCT i.auction_id) AS impressions,
    COUNT(DISTINCT b.auction_id) AS billing_ok
FROM TUMBLE(auction_requests, to_timestamp(event_ts / 1000.0), INTERVAL '5 minutes') r
LEFT JOIN auctions_completed c
  ON r.app_id = c.app_id AND r.auction_id = c.auction_id
LEFT JOIN ad_impressions i
  ON r.app_id = i.app_id AND r.auction_id = i.auction_id
LEFT JOIN billing_notices b
  ON r.app_id = b.app_id AND r.auction_id = b.auction_id AND b.success
GROUP BY window_start;

CREATE MATERIALIZED VIEW IF NOT EXISTS notice_sends AS
SELECT event_ts, 'win' AS notice_type, success FROM win_notices
UNION ALL
SELECT event_ts, 'billing' AS notice_type, success FROM billing_notices
UNION ALL
SELECT event_ts, 'loss' AS notice_type, success FROM loss_notices;

CREATE MATERIALIZED VIEW IF NOT EXISTS notice_delivery_5m AS
SELECT window_start, notice_type, success, COUNT(*) AS n
FROM TUMBLE(notice_sends, to_timestamp(event_ts / 1000.0), INTERVAL '5 minutes')
GROUP BY window_start, notice_type, success;
