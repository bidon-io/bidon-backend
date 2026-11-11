mod kafka;

use std::time::Duration;

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
    let producer_task = tokio::spawn(async move {
        kafka::run(producer, metrics, inbound).await;
    });

    let listener = tokio::net::TcpListener::bind(("0.0.0.0", config.port)).await?;
    tracing::info!(port = config.port, "bidon-telemetry listening");
    let (stop_tx, mut stop_rx) = tokio::sync::watch::channel(false);
    tokio::spawn(async move {
        shutdown_signal().await;
        let _ = stop_tx.send(true);
    });
    let mut serve_stop = stop_rx.clone();
    let server = axum::serve(listener, router(ingestor)).with_graceful_shutdown(async move {
        let _ = serve_stop.wait_for(|stop| *stop).await;
    });
    // When serve() returns it has dropped the router and with it the last queue
    // sender, so the producer finishes what is queued and returns.
    let work = async {
        server.await?;
        let _ = producer_task.await;
        anyhow::Ok(())
    };
    tokio::pin!(work);
    tokio::select! {
        result = &mut work => return result,
        _ = stop_rx.wait_for(|stop| *stop) => {}
    }
    // One deadline for open connections and the queue drain, inside Docker's
    // 10s stop timeout.
    match tokio::time::timeout(SHUTDOWN_TIMEOUT, work).await {
        Ok(result) => result,
        Err(_) => {
            tracing::warn!("shutdown deadline passed before the queue drained");
            Ok(())
        }
    }
}

const SHUTDOWN_TIMEOUT: Duration = Duration::from_secs(8);

async fn shutdown_signal() {
    use tokio::signal::unix::{signal, SignalKind};

    match signal(SignalKind::terminate()) {
        Ok(mut term) => {
            tokio::select! {
                _ = tokio::signal::ctrl_c() => {}
                _ = term.recv() => {}
            }
        }
        Err(err) => {
            tracing::warn!(error = %err, "SIGTERM handler; only Ctrl-C stops the process");
            let _ = tokio::signal::ctrl_c().await;
        }
    }
    tracing::info!("shutting down");
}
