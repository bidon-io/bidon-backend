use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use std::time::Duration;

use serde::Serialize;

pub const EVENTS_PROTO: &str =
    include_str!("../../schemas/proto/org/bidon/telemetry/v1/events.proto");

#[derive(Default)]
pub struct SchemaCache {
    ids: RwLock<HashMap<String, i32>>,
}

impl SchemaCache {
    pub fn frame(&self, subject: &str, indexes: &[u64], payload: &[u8]) -> Vec<u8> {
        let guard = self.ids.read().expect("schema ids");
        match guard.get(subject).copied() {
            Some(id) => crate::frame::frame(id, indexes, payload),
            None => payload.to_vec(),
        }
    }

    fn insert(&self, subject: String, id: i32) {
        self.ids.write().expect("schema ids").insert(subject, id);
    }
}

/// Registers each subject in the background. Produce never waits on this.
/// Until a subject has an id, values stay raw protobuf.
pub fn spawn_register(cache: Arc<SchemaCache>, base_url: String, subjects: Vec<String>) {
    if base_url.is_empty() || subjects.is_empty() {
        return;
    }
    tokio::spawn(async move {
        let client = match reqwest::Client::builder()
            .timeout(Duration::from_secs(2))
            .build()
        {
            Ok(client) => client,
            Err(err) => {
                tracing::warn!(error = %err, "schema registry client");
                return;
            }
        };
        let mut pending = subjects;
        let mut backoff = Duration::from_secs(1);
        while !pending.is_empty() {
            let mut failed = Vec::new();
            for subject in pending.drain(..) {
                match register_one(&client, &base_url, &subject).await {
                    Ok(id) => cache.insert(subject, id),
                    Err(err) => {
                        tracing::warn!(
                            error = %err,
                            %subject,
                            "schema registry register; producing raw protobuf until it succeeds"
                        );
                        failed.push(subject);
                    }
                }
            }
            pending = failed;
            if pending.is_empty() {
                break;
            }
            tokio::time::sleep(backoff).await;
            backoff = (backoff * 2).min(Duration::from_secs(30));
        }
    });
}

#[derive(Serialize)]
struct RegisterBody<'a> {
    #[serde(rename = "schemaType")]
    schema_type: &'a str,
    schema: &'a str,
}

async fn register_one(
    client: &reqwest::Client,
    base_url: &str,
    subject: &str,
) -> anyhow::Result<i32> {
    let encoded: String = urlencoding_encode(subject);
    let url = format!(
        "{}/subjects/{encoded}/versions",
        base_url.trim_end_matches('/')
    );
    let response = client
        .post(&url)
        .json(&RegisterBody {
            schema_type: "PROTOBUF",
            schema: EVENTS_PROTO,
        })
        .send()
        .await?
        .error_for_status()?;
    let body: serde_json::Value = response.json().await?;
    let id = body
        .get("id")
        .and_then(|id| id.as_i64())
        .ok_or_else(|| anyhow::anyhow!("schema id missing"))?;
    let _ = client
        .put(format!(
            "{}/config/{encoded}",
            base_url.trim_end_matches('/')
        ))
        .json(&serde_json::json!({"compatibility": "BACKWARD"}))
        .send()
        .await;
    Ok(i32::try_from(id)?)
}

fn urlencoding_encode(subject: &str) -> String {
    let mut out = String::with_capacity(subject.len());
    for byte in subject.bytes() {
        match byte {
            b'A'..=b'Z' | b'a'..=b'z' | b'0'..=b'9' | b'-' | b'_' | b'.' | b'~' => {
                out.push(byte as char)
            }
            _ => out.push_str(&format!("%{byte:02X}")),
        }
    }
    out
}
