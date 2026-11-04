use std::collections::BTreeMap;

use chrono::Utc;
use rskafka::client::error::{Error, ProtocolError};
use rskafka::client::partition::{Compression, PartitionClient, UnknownTopicHandling};
use rskafka::client::ClientBuilder;
use rskafka::record::Record;
use tokio::sync::mpsc;

use bidon_telemetry::{Metrics, Outbound};

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
        .build()
        .await
        .map_err(|err| anyhow::anyhow!("kafka client: {err}"))?;
    let known = client
        .list_topics()
        .await
        .map_err(|err| anyhow::anyhow!("list topics: {err}"))?;
    if !known.iter().any(|existing| existing.name == topic) {
        let controller = client
            .controller_client()
            .map_err(|err| anyhow::anyhow!("kafka controller: {err}"))?;
        match controller.create_topic(topic, 1, 1, 5_000).await {
            Ok(()) => {}
            Err(Error::ServerError {
                protocol_error: ProtocolError::TopicAlreadyExists,
                ..
            }) => {}
            Err(err) => return Err(anyhow::anyhow!("create topic {topic}: {err}")),
        }
    }
    client
        .partition_client(topic, 0, UnknownTopicHandling::Error)
        .await
        .map_err(|err| anyhow::anyhow!("kafka partition: {err}"))
}

pub async fn run(
    producer: PartitionClient,
    metrics: Metrics,
    mut inbound: mpsc::Receiver<Outbound>,
) {
    while let Some(record) = inbound.recv().await {
        let mut headers = BTreeMap::new();
        headers.insert(
            "event_name".to_owned(),
            record.event_name.as_bytes().to_vec(),
        );
        headers.insert(
            "protobuf_message".to_owned(),
            record.message_type.as_bytes().to_vec(),
        );
        let event_name = record.event_name.clone();
        let kafka_record = Record {
            key: None,
            value: Some(record.value),
            headers,
            timestamp: Utc::now(),
        };
        if let Err(err) = producer
            .produce(vec![kafka_record], Compression::NoCompression)
            .await
        {
            metrics.produce_error();
            tracing::warn!(error = %err, event_name = %event_name, "produce telemetry event");
        }
    }
}
