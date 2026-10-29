use anyhow::Context;

pub const DEFAULT_PORT: u16 = 1326;
pub const SCHEMA_VERSION: &str = "1";

#[derive(Debug, Clone)]
pub struct Config {
    pub port: u16,
    pub brokers: String,
    pub topic: String,
    pub schema_registry_url: String,
    pub ingest_enabled: bool,
    pub client_id: String,
}

impl Config {
    pub fn from_env() -> anyhow::Result<Self> {
        Ok(Self {
            port: env_u16("TELEMETRY_PORT", DEFAULT_PORT)?,
            brokers: required("KAFKA_BROKERS_LIST")?,
            topic: env_string("KAFKA_TELEMETRY_EVENTS_TOPIC", "telemetry-events"),
            schema_registry_url: env_string("SCHEMA_REGISTRY_URL", ""),
            ingest_enabled: env_bool("TELEMETRY_INGEST_ENABLED", true),
            client_id: env_string("KAFKA_CLIENT_ID", "bidon-telemetry"),
        })
    }
}

fn required(key: &str) -> anyhow::Result<String> {
    std::env::var(key).with_context(|| format!("{key} is required"))
}

fn env_string(key: &str, default: &str) -> String {
    match std::env::var(key) {
        Ok(value) if !value.is_empty() => value,
        _ => default.to_owned(),
    }
}

fn env_u16(key: &str, default: u16) -> anyhow::Result<u16> {
    match std::env::var(key) {
        Ok(value) if !value.is_empty() => value
            .parse()
            .with_context(|| format!("{key} is not a port")),
        _ => Ok(default),
    }
}

fn env_bool(key: &str, default: bool) -> bool {
    match std::env::var(key) {
        Ok(value) => !matches!(value.as_str(), "0" | "false" | "FALSE" | "no"),
        Err(_) => default,
    }
}
