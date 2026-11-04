mod kafka;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env().unwrap_or_else(|_| "info".into()),
        )
        .json()
        .init();
    run().await
}

async fn run() -> anyhow::Result<()> {
    use std::sync::Arc;

    use bidon_telemetry::{
        router, spawn_register, ChannelOutbox, Config, Ingestor, Metrics, SchemaCache,
    };

    let config = Config::from_env()?;
    let (outbox, inbound) = ChannelOutbox::pair();
    let metrics = Metrics::new();
    let schemas = Arc::new(SchemaCache::default());
    let ingestor = Arc::new(Ingestor::new(
        Arc::new(outbox),
        schemas.clone(),
        metrics.clone(),
        config.topic.clone(),
        config.ingest_enabled,
    ));
    spawn_register(
        schemas,
        config.schema_registry_url.clone(),
        ingestor.subjects(),
    );
    let producer = kafka::producer(&config.brokers, &config.client_id, &config.topic).await?;
    tokio::spawn(async move {
        kafka::run(producer, metrics, inbound).await;
    });

    let listener = tokio::net::TcpListener::bind(("0.0.0.0", config.port)).await?;
    tracing::info!(port = config.port, "bidon-telemetry listening");
    axum::serve(listener, router(ingestor))
        .with_graceful_shutdown(async {
            let _ = tokio::signal::ctrl_c().await;
        })
        .await?;
    Ok(())
}
