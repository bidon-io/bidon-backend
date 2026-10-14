use std::sync::Arc;

use axum::http::StatusCode;
use serde::Deserialize;
use serde_json::Value;
use tokio::sync::mpsc;
use uuid::Uuid;

use crate::catalog::Catalog;
use crate::metrics::Metrics;
use crate::registry::SchemaCache;

pub const MAX_BYTES: usize = 256 << 10;
pub const MAX_EVENTS: usize = 100;
pub const MAX_METRICS: usize = 50;
const QUEUE_CAPACITY: usize = 4096;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Outbound {
    pub value: Vec<u8>,
    pub event_name: String,
    pub message_type: String,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum EnqueueError {
    Full,
    Closed,
}

pub trait Outbox: Send + Sync {
    fn try_enqueue(&self, record: Outbound) -> Result<(), EnqueueError>;
}

pub struct ChannelOutbox {
    tx: mpsc::Sender<Outbound>,
}

impl ChannelOutbox {
    pub fn pair() -> (Self, mpsc::Receiver<Outbound>) {
        let (tx, rx) = mpsc::channel(QUEUE_CAPACITY);
        (Self { tx }, rx)
    }
}

impl Outbox for ChannelOutbox {
    fn try_enqueue(&self, record: Outbound) -> Result<(), EnqueueError> {
        self.tx.try_send(record).map_err(|err| match err {
            mpsc::error::TrySendError::Full(_) => EnqueueError::Full,
            mpsc::error::TrySendError::Closed(_) => EnqueueError::Closed,
        })
    }
}

#[cfg(test)]
pub struct RecordingOutbox {
    pub records: std::sync::Mutex<Vec<Outbound>>,
    pub fail: Option<EnqueueError>,
}

#[cfg(test)]
impl RecordingOutbox {
    pub fn new() -> Self {
        Self {
            records: std::sync::Mutex::new(Vec::new()),
            fail: None,
        }
    }
}

#[cfg(test)]
impl Outbox for RecordingOutbox {
    fn try_enqueue(&self, record: Outbound) -> Result<(), EnqueueError> {
        if let Some(err) = self.fail {
            return Err(err);
        }
        self.records.lock().expect("records").push(record);
        Ok(())
    }
}

pub struct HttpReply {
    pub status: StatusCode,
    pub content_type: &'static str,
    pub body: String,
}

pub struct Ingestor {
    catalog: Catalog,
    out: Arc<dyn Outbox>,
    schemas: Arc<SchemaCache>,
    metrics: Metrics,
    topic: String,
    enabled: bool,
}

impl Ingestor {
    pub fn new(
        out: Arc<dyn Outbox>,
        schemas: Arc<SchemaCache>,
        metrics: Metrics,
        topic: String,
        enabled: bool,
    ) -> Self {
        Self {
            catalog: Catalog::load(),
            out,
            schemas,
            metrics,
            topic,
            enabled,
        }
    }

    pub fn subjects(&self) -> Vec<String> {
        self.catalog.subjects(&self.topic)
    }

    pub fn metrics(&self) -> &Metrics {
        &self.metrics
    }

    pub fn handle(&self, body: &[u8], sdk_version: &str) -> HttpReply {
        if body.len() > MAX_BYTES {
            return problem(StatusCode::PAYLOAD_TOO_LARGE, "telemetry batch too large");
        }
        let batch: Batch = match serde_json::from_slice(body) {
            Ok(batch) => batch,
            Err(_) => return problem(StatusCode::BAD_REQUEST, "malformed telemetry batch"),
        };
        let Some(app) = batch.app else {
            return problem(
                StatusCode::BAD_REQUEST,
                "telemetry request requires app.key, app.bundle, and session.id",
            );
        };
        if app.key.is_empty()
            || app.bundle.is_empty()
            || batch
                .session
                .as_ref()
                .is_none_or(|session| session.id.is_empty())
        {
            return problem(
                StatusCode::BAD_REQUEST,
                "telemetry request requires app.key, app.bundle, and session.id",
            );
        }
        let session = batch.session.unwrap();
        if Uuid::parse_str(&session.id).is_err() {
            return problem(StatusCode::BAD_REQUEST, "malformed telemetry batch");
        }
        if batch.events.len() > MAX_EVENTS || batch.metrics.len() > MAX_METRICS {
            return problem(StatusCode::PAYLOAD_TOO_LARGE, "telemetry batch too large");
        }
        if !self.enabled {
            tracing::info!(sdk_version, app_key = %app.key, bytes = body.len(), accepted = 0, "telemetry ingest disabled");
            return accepted(0);
        }

        let mut kept = 0usize;
        for event in &batch.events {
            let name = event
                .get("envelope")
                .and_then(|envelope| envelope.get("event_name"))
                .and_then(|name| name.as_str())
                .unwrap_or("");
            if !self.catalog.contains(name) {
                self.metrics.dropped_name();
                continue;
            }
            let Some(encoded) =
                self.catalog
                    .encode_event(event, &app.key, &app.bundle, &session.id)
            else {
                self.metrics.dropped_name();
                continue;
            };
            let subject = format!("{}-{}", self.topic, encoded.message_type);
            let value = self
                .schemas
                .frame(&subject, &encoded.indexes, &encoded.payload);
            match self.out.try_enqueue(Outbound {
                value,
                event_name: encoded.event_name,
                message_type: encoded.message_type,
            }) {
                Ok(()) => {
                    self.metrics.accepted();
                    kept += 1;
                }
                Err(EnqueueError::Full) => self.metrics.buffer_full(),
                Err(EnqueueError::Closed) => self.metrics.produce_error(),
            }
        }

        for metric in &batch.metrics {
            if apply_metric(&self.metrics, metric) {
                self.metrics.accepted();
                kept += 1;
            } else {
                self.metrics.dropped_metric();
            }
        }

        tracing::info!(sdk_version, app_key = %app.key, bytes = body.len(), accepted = kept, "telemetry ingest");
        accepted(kept)
    }
}

fn apply_metric(metrics: &Metrics, raw: &Value) -> bool {
    let name = raw
        .get("name")
        .and_then(|value| value.as_str())
        .unwrap_or("");
    let kind = raw
        .get("type")
        .and_then(|value| value.as_str())
        .unwrap_or("");
    let Some(value) = raw.get("value").and_then(json_f64) else {
        return false;
    };
    if kind != "histogram" {
        return false;
    }
    let mut labels = std::collections::HashMap::new();
    if let Some(object) = raw.get("labels").and_then(|value| value.as_object()) {
        for (key, label) in object {
            let Some(text) = label.as_str() else {
                return false;
            };
            labels.insert(key.clone(), text.to_owned());
        }
    }
    metrics.observe(name, value, &labels)
}

fn json_f64(value: &Value) -> Option<f64> {
    value
        .as_f64()
        .or_else(|| value.as_i64().map(|number| number as f64))
}

fn accepted(count: usize) -> HttpReply {
    HttpReply {
        status: StatusCode::ACCEPTED,
        content_type: "application/json",
        body: format!(r#"{{"accepted":{count}}}"#),
    }
}

fn problem(status: StatusCode, message: &str) -> HttpReply {
    HttpReply {
        status,
        content_type: "application/problem+json",
        body: format!(
            r#"{{"error":{{"message":"{message}","code":{}}}}}"#,
            status.as_u16()
        ),
    }
}

#[derive(Deserialize)]
struct Batch {
    app: Option<AppIds>,
    session: Option<SessionIds>,
    #[serde(default)]
    events: Vec<Value>,
    #[serde(default)]
    metrics: Vec<Value>,
}

#[derive(Deserialize)]
struct AppIds {
    #[serde(default)]
    key: String,
    #[serde(default)]
    bundle: String,
}

#[derive(Deserialize)]
struct SessionIds {
    #[serde(default)]
    id: String,
}

#[cfg(test)]
mod tests {
    use super::*;

    fn ingestor(enabled: bool, fail: Option<EnqueueError>) -> (Ingestor, Arc<RecordingOutbox>) {
        let out = Arc::new(RecordingOutbox {
            fail,
            ..RecordingOutbox::new()
        });
        let ingestor = Ingestor::new(
            out.clone(),
            Arc::new(SchemaCache::default()),
            Metrics::new(),
            "telemetry-events".to_owned(),
            enabled,
        );
        (ingestor, out)
    }

    const VALID: &str = r#"{
      "app": {"key": "app_abc123", "bundle": "com.example.game"},
      "session": {"id": "550e8400-e29b-41d4-a716-446655440000"},
      "events": [
        {
          "envelope": {
            "event_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
            "event_name": "ad_request_started",
            "event_ts": 1710000000123,
            "auction_id": "auc_01J8Z3K4N5",
            "ad_type": "banner",
            "ad_format": "BANNER",
            "sampling_rate": 1
          },
          "price_floor": 0.05
        },
        {
          "envelope": {
            "event_id": "0f14d0ab-9605-4a62-a9e4-90e086941e9f",
            "event_name": "ad_filled",
            "event_ts": 1710000000450,
            "auction_id": "auc_01J8Z3K4N5",
            "ad_type": "banner",
            "ad_format": "BANNER",
            "sampling_rate": 1
          },
          "demand_id": "bidmachine",
          "price": 1.24,
          "bid_type": "bidding"
        }
      ],
      "metrics": [
        {
          "name": "client_auction_duration_seconds",
          "type": "histogram",
          "value": 0.327,
          "labels": {"ad_type": "banner"}
        }
      ]
    }"#;

    fn batch(events: &str) -> String {
        format!(
            r#"{{"app":{{"key":"app_abc123","bundle":"com.example.game"}},"session":{{"id":"550e8400-e29b-41d4-a716-446655440000"}},{events}}}"#
        )
    }

    #[tokio::test]
    async fn valid_batch_is_accepted_and_enqueued() {
        let (ingestor, out) = ingestor(true, None);
        let reply = ingestor.handle(VALID.as_bytes(), "0.7.0");
        assert_eq!(reply.status, StatusCode::ACCEPTED);
        assert_eq!(reply.body, r#"{"accepted":3}"#);
        let records = out.records.lock().unwrap();
        assert_eq!(records.len(), 2);
        assert_eq!(records[0].event_name, "ad_request_started");
        assert_eq!(
            records[0].message_type,
            "org.bidon.telemetry.v1.AdRequestStarted"
        );
        assert!(!records[0].value.is_empty());
        assert!(ingestor.metrics().encode().contains("ingest_total"));
        assert!(ingestor
            .metrics()
            .encode()
            .contains("client_auction_duration_seconds"));
    }

    #[tokio::test]
    async fn unknown_event_and_server_event_are_dropped() {
        let (ingestor, out) = ingestor(true, None);
        let body = batch(
            r#""events":[
                {"envelope":{"event_id":"11111111-1111-4111-8111-111111111111","event_name":"not_in_catalogue","event_ts":1}},
                {"envelope":{"event_id":"22222222-2222-4222-8222-222222222222","event_name":"dsp_request_sent","event_ts":1}}
            ]"#,
        );
        let reply = ingestor.handle(body.as_bytes(), "");
        assert_eq!(reply.body, r#"{"accepted":0}"#);
        assert!(out.records.lock().unwrap().is_empty());
    }

    #[tokio::test]
    async fn extra_metric_label_is_dropped() {
        let (ingestor, _) = ingestor(true, None);
        let body = batch(
            r#""metrics":[{"name":"client_auction_duration_seconds","type":"histogram","value":1,"labels":{"ad_type":"banner","auction_id":"x"}}]"#,
        );
        let reply = ingestor.handle(body.as_bytes(), "");
        assert_eq!(reply.body, r#"{"accepted":0}"#);
    }

    #[tokio::test]
    async fn missing_app_is_400() {
        let (ingestor, _) = ingestor(true, None);
        let reply = ingestor.handle(
            br#"{"session":{"id":"550e8400-e29b-41d4-a716-446655440000"}}"#,
            "",
        );
        assert_eq!(reply.status, StatusCode::BAD_REQUEST);
    }

    #[tokio::test]
    async fn malformed_json_is_400() {
        let (ingestor, _) = ingestor(true, None);
        let reply = ingestor.handle(b"not-json", "");
        assert_eq!(reply.status, StatusCode::BAD_REQUEST);
    }

    #[tokio::test]
    async fn too_many_events_is_413() {
        let (ingestor, _) = ingestor(true, None);
        let mut events = String::from(r#""events":["#);
        for i in 0..101 {
            if i > 0 {
                events.push(',');
            }
            events.push_str(&format!(
                r#"{{"envelope":{{"event_id":"00000000-0000-4000-8000-{i:012}","event_name":"auction_requested","event_ts":1}}}}"#
            ));
        }
        events.push(']');
        let reply = ingestor.handle(batch(&events).as_bytes(), "");
        assert_eq!(reply.status, StatusCode::PAYLOAD_TOO_LARGE);
    }

    #[tokio::test]
    async fn oversized_body_is_413() {
        let (ingestor, _) = ingestor(true, None);
        let body = vec![b' '; MAX_BYTES + 1];
        let reply = ingestor.handle(&body, "");
        assert_eq!(reply.status, StatusCode::PAYLOAD_TOO_LARGE);
    }

    #[tokio::test]
    async fn kill_switch_accepts_nothing() {
        let (ingestor, out) = ingestor(false, None);
        let reply = ingestor.handle(VALID.as_bytes(), "");
        assert_eq!(reply.status, StatusCode::ACCEPTED);
        assert_eq!(reply.body, r#"{"accepted":0}"#);
        assert!(out.records.lock().unwrap().is_empty());
    }

    #[tokio::test]
    async fn full_buffer_is_not_accepted() {
        let (ingestor, out) = ingestor(true, Some(EnqueueError::Full));
        let reply = ingestor.handle(VALID.as_bytes(), "");
        assert_eq!(reply.body, r#"{"accepted":1}"#);
        assert!(out.records.lock().unwrap().is_empty());
        assert!(ingestor
            .metrics()
            .encode()
            .contains(r#"result="buffer_full""#));
    }
}
