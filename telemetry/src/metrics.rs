use std::collections::HashMap;
use std::sync::Arc;

use prometheus::{HistogramOpts, HistogramVec, IntCounterVec, Opts, Registry, TextEncoder};

const BUCKETS: &[f64] = &[0.05, 0.1, 0.25, 0.5, 1.0, 2.0, 4.0, 8.0];

#[derive(Clone)]
pub struct Metrics {
    registry: Arc<Registry>,
    ingest: IntCounterVec,
    auction: HistogramVec,
    tokens: HistogramVec,
    ad_load: HistogramVec,
}

impl Metrics {
    pub fn new() -> Self {
        let registry = Registry::new();
        let ingest = IntCounterVec::new(
            Opts::new("ingest_total", "SDK telemetry ingest results."),
            &["result"],
        )
        .expect("ingest counter");
        let auction = histogram(
            "client_auction_duration_seconds",
            "Client-observed auction duration.",
            &["ad_type"],
        );
        let tokens = histogram(
            "client_token_collection_duration_seconds",
            "Client-observed token collection duration.",
            &["ad_type"],
        );
        let ad_load = histogram(
            "client_ad_load_duration_seconds",
            "Client-observed ad load duration.",
            &["ad_type", "result"],
        );
        for collector in [
            Box::new(ingest.clone()) as Box<dyn prometheus::core::Collector>,
            Box::new(auction.clone()),
            Box::new(tokens.clone()),
            Box::new(ad_load.clone()),
        ] {
            registry.register(collector).expect("register metric");
        }
        for result in [
            "accepted",
            "dropped_name",
            "dropped_metric",
            "buffer_full",
            "produce_error",
        ] {
            ingest.with_label_values(&[result]);
        }
        Self {
            registry: Arc::new(registry),
            ingest,
            auction,
            tokens,
            ad_load,
        }
    }

    pub fn accepted(&self) {
        self.bump("accepted");
    }

    pub fn dropped_name(&self) {
        self.bump("dropped_name");
    }

    pub fn dropped_metric(&self) {
        self.bump("dropped_metric");
    }

    pub fn buffer_full(&self) {
        self.bump("buffer_full");
    }

    pub fn produce_error(&self) {
        self.bump("produce_error");
    }

    /// Records an allow-listed histogram observation.
    /// An unknown name, a non-histogram type, or an extra label key is rejected.
    pub fn observe(&self, name: &str, value: f64, labels: &HashMap<String, String>) -> bool {
        let (series, keys): (&HistogramVec, &[&str]) = match name {
            "client_auction_duration_seconds" => (&self.auction, &["ad_type"]),
            "client_token_collection_duration_seconds" => (&self.tokens, &["ad_type"]),
            "client_ad_load_duration_seconds" => (&self.ad_load, &["ad_type", "result"]),
            _ => return false,
        };
        if labels.keys().any(|key| !keys.contains(&key.as_str())) {
            return false;
        }
        let values: Vec<&str> = keys
            .iter()
            .map(|key| labels.get(*key).map(String::as_str).unwrap_or(""))
            .collect();
        series.with_label_values(&values).observe(value);
        true
    }

    pub fn encode(&self) -> String {
        let families = self.registry.gather();
        let mut buf = String::new();
        TextEncoder::new()
            .encode_utf8(&families, &mut buf)
            .expect("encode metrics");
        buf
    }

    fn bump(&self, result: &str) {
        self.ingest.with_label_values(&[result]).inc();
    }
}

impl Default for Metrics {
    fn default() -> Self {
        Self::new()
    }
}

fn histogram(name: &str, help: &str, labels: &[&str]) -> HistogramVec {
    HistogramVec::new(
        HistogramOpts::new(name, help).buckets(BUCKETS.to_vec()),
        labels,
    )
    .expect("histogram")
}
