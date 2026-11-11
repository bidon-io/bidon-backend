use std::collections::BTreeMap;
use std::time::Duration;

use chrono::Utc;
use rskafka::client::partition::{Compression, PartitionClient, UnknownTopicHandling};
use rskafka::client::ClientBuilder;
use rskafka::record::Record;
use rskafka::BackoffConfig;
use tokio::sync::mpsc;

use bidon_telemetry::{Metrics, Outbound};

const MAX_BATCH_RECORDS: usize = 500;
/// Well under Redpanda's default 1 MiB batch limit, leaving room for headers.
const MAX_BATCH_BYTES: usize = 512 << 10;
/// rskafka retries forever by default. A deadline lets a failed produce return,
/// so it is counted and the next batch is not stuck behind it.
const PRODUCE_DEADLINE: Duration = Duration::from_secs(5);

/// Connects to partition 0 of `topic`. The topic is declared by compose
/// (`redpanda-topics`); this process does not create it.
pub async fn producer(
    brokers: &str,
    client_id: &str,
    topic: &str,
) -> anyhow::Result<PartitionClient> {
    let brokers = brokers
        .split(',')
        .map(str::trim)
        .filter(|broker| !broker.is_empty())
        .map(str::to_owned)
        .collect();
    let client = ClientBuilder::new(brokers)
        .client_id(client_id)
        .backoff_config(BackoffConfig {
            deadline: Some(PRODUCE_DEADLINE),
            ..Default::default()
        })
        .build()
        .await
        .map_err(|err| anyhow::anyhow!("kafka client: {err}"))?;
    client
        .partition_client(topic, 0, UnknownTopicHandling::Error)
        .await
        .map_err(|err| anyhow::anyhow!("topic {topic} partition 0 (create the topic first): {err}"))
}

/// Produces queued records until every sender is dropped and the queue is empty.
/// Whatever is already queued goes out in batches, so a shutdown drains it.
pub async fn run(
    producer: PartitionClient,
    metrics: Metrics,
    mut inbound: mpsc::Receiver<Outbound>,
) {
    let mut queued = Vec::with_capacity(MAX_BATCH_RECORDS);
    while inbound.recv_many(&mut queued, MAX_BATCH_RECORDS).await > 0 {
        for batch in batches(queued.drain(..), MAX_BATCH_BYTES) {
            let count = batch.len();
            if let Err(err) = producer.produce(batch, Compression::Lz4).await {
                metrics.produce_errors(count as u64);
                tracing::warn!(error = %err, records = count, "produce telemetry events");
            }
        }
    }
}

/// Splits records so each batch's values stay within `max_bytes`. A record
/// larger than the limit goes out alone.
fn batches(records: impl Iterator<Item = Outbound>, max_bytes: usize) -> Vec<Vec<Record>> {
    let mut out = Vec::new();
    let mut current = Vec::new();
    let mut bytes = 0;
    for record in records {
        let size = record.value.len();
        if !current.is_empty() && bytes + size > max_bytes {
            out.push(std::mem::take(&mut current));
            bytes = 0;
        }
        bytes += size;
        current.push(to_record(record));
    }
    if !current.is_empty() {
        out.push(current);
    }
    out
}

fn to_record(record: Outbound) -> Record {
    let mut headers = BTreeMap::new();
    headers.insert("event_name".to_owned(), record.event_name.into_bytes());
    headers.insert(
        "protobuf_message".to_owned(),
        record.message_type.into_bytes(),
    );
    Record {
        key: None,
        value: Some(record.value),
        headers,
        timestamp: Utc::now(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn outbound(size: usize) -> Outbound {
        Outbound {
            value: vec![0; size],
            event_name: "ad_loaded".to_owned(),
            message_type: "org.bidon.telemetry.v1.AdLoaded".to_owned(),
        }
    }

    #[test]
    fn batches_split_on_byte_limit() {
        let sizes: Vec<usize> = batches([40, 40, 40, 100, 10].map(outbound).into_iter(), 100)
            .iter()
            .map(Vec::len)
            .collect();
        assert_eq!(sizes, vec![2, 1, 1, 1]);
    }

    #[test]
    fn no_records_is_no_batches() {
        assert!(batches(std::iter::empty(), 100).is_empty());
    }
}
