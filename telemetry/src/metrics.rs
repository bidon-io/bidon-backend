use std::collections::HashMap;
use std::sync::Arc;

use prometheus::{HistogramOpts, HistogramVec, IntCounterVec, Opts, Registry, TextEncoder};

const BUCKETS: &[f64] = &[0.05, 0.1, 0.25, 0.5, 1.0, 2.0, 4.0, 8.0];

const AD_TYPES: &[&str] = &["banner", "interstitial", "rewarded"];
/// A load ends in `ad_loaded` or `ad_load_failed`; the failure detail is on that event.
const AD_LOAD_RESULTS: &[&str] = &["loaded", "failed"];

type LabelSpec = (&'static str, &'static [&'static str]);

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
            "dropped_invalid",
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

    pub fn dropped_invalid(&self) {
        self.bump("dropped_invalid");
    }

    pub fn buffer_full(&self) {
        self.bump("buffer_full");
    }

    pub fn produce_error(&self) {
        self.bump("produce_error");
    }

    pub fn produce_errors(&self, count: u64) {
        self.ingest
            .with_label_values(&["produce_error"])
            .inc_by(count);
    }

    /// Records an allow-listed histogram observation.
    /// An unknown name, a negative or non-finite value, an extra label key, or a
    /// label value outside the series' fixed set is rejected, so a client cannot
    /// mint new series.
    pub fn observe(&self, name: &str, value: f64, labels: &HashMap<String, String>) -> bool {
        let (series, spec): (&HistogramVec, &[LabelSpec]) = match name {
            "client_auction_duration_seconds" => (&self.auction, &[("ad_type", AD_TYPES)]),
            "client_token_collection_duration_seconds" => (&self.tokens, &[("ad_type", AD_TYPES)]),
            "client_ad_load_duration_seconds" => (
                &self.ad_load,
                &[("ad_type", AD_TYPES), ("result", AD_LOAD_RESULTS)],
            ),
            _ => return false,
        };
        if !value.is_finite() || value < 0.0 {
            return false;
        }
        if labels
            .keys()
            .any(|key| !spec.iter().any(|(allowed, _)| *allowed == key.as_str()))
        {
            return false;
        }
        let mut values: Vec<&str> = Vec::with_capacity(spec.len());
        for (key, allowed) in spec {
            match labels.get(*key).map(String::as_str) {
                Some(value) if allowed.contains(&value) => values.push(value),
                Some(_) => return false,
                None => values.push(""),
            }
        }
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

#[cfg(test)]
mod tests {
    use super::*;

    fn labels(pairs: &[(&str, &str)]) -> HashMap<String, String> {
        pairs
            .iter()
            .map(|(k, v)| ((*k).to_owned(), (*v).to_owned()))
            .collect()
    }

    #[test]
    fn ad_load_result_is_loaded_or_failed() {
        let metrics = Metrics::new();
        for (result, kept) in [("loaded", true), ("failed", true), ("no_fill", false)] {
            assert_eq!(
                metrics.observe(
                    "client_ad_load_duration_seconds",
                    0.4,
                    &labels(&[("ad_type", "banner"), ("result", result)]),
                ),
                kept,
                "{result}"
            );
        }
    }

    #[test]
    fn unknown_label_value_is_rejected() {
        let metrics = Metrics::new();
        assert!(!metrics.observe(
            "client_auction_duration_seconds",
            0.4,
            &labels(&[("ad_type", "made-up-123")]),
        ));
        assert!(!metrics.encode().contains("made-up-123"));
    }

    #[test]
    fn negative_and_non_finite_values_are_rejected() {
        let metrics = Metrics::new();
        let ok = labels(&[("ad_type", "banner")]);
        assert!(!metrics.observe("client_auction_duration_seconds", -1.0, &ok));
        assert!(!metrics.observe("client_auction_duration_seconds", f64::NAN, &ok));
        assert!(!metrics.observe("client_auction_duration_seconds", f64::INFINITY, &ok));
    }
}
