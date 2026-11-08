use std::sync::Arc;

use axum::body::Body;
use axum::extract::State;
use axum::http::{header, Request, StatusCode};
use axum::response::IntoResponse;
use axum::routing::{get, post};
use axum::Router;
use futures_util::StreamExt;

use crate::ingest::{problem, HttpReply, Ingestor, MAX_BYTES};

pub fn router(ingestor: Arc<Ingestor>) -> Router {
    Router::new()
        .route("/v2/telemetry", post(post_telemetry))
        .route("/metrics", get(metrics))
        .route("/health", get(health))
        .with_state(ingestor)
}

async fn health() -> &'static str {
    "ok"
}

async fn metrics(State(ingestor): State<Arc<Ingestor>>) -> impl IntoResponse {
    (
        StatusCode::OK,
        [(header::CONTENT_TYPE, "text/plain; version=0.0.4")],
        ingestor.metrics().encode(),
    )
}

async fn post_telemetry(
    State(ingestor): State<Arc<Ingestor>>,
    request: Request<Body>,
) -> impl IntoResponse {
    let sdk_version = request
        .headers()
        .get("x-bidon-version")
        .and_then(|value| value.to_str().ok())
        .unwrap_or("")
        .to_owned();
    let body = match read_limited(request.into_body()).await {
        Ok(body) => body,
        Err(BodyError::TooLarge) => {
            return reply_response(&problem(
                StatusCode::PAYLOAD_TOO_LARGE,
                "telemetry batch too large",
            ));
        }
        Err(BodyError::Malformed) => {
            return reply_response(&problem(
                StatusCode::BAD_REQUEST,
                "malformed telemetry batch",
            ));
        }
    };
    reply_response(&ingestor.handle(&body, &sdk_version))
}

enum BodyError {
    TooLarge,
    Malformed,
}

async fn read_limited(body: Body) -> Result<Vec<u8>, BodyError> {
    let mut buf = Vec::new();
    let mut stream = body.into_data_stream();
    while let Some(next) = stream.next().await {
        let data = next.map_err(|_| BodyError::Malformed)?;
        if buf.len().saturating_add(data.len()) > MAX_BYTES {
            return Err(BodyError::TooLarge);
        }
        buf.extend_from_slice(&data);
    }
    Ok(buf)
}

fn reply_response(reply: &HttpReply) -> impl IntoResponse {
    (
        reply.status,
        [(header::CONTENT_TYPE, reply.content_type)],
        reply.body.clone(),
    )
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ingest::{Ingestor, RecordingOutbox};
    use crate::metrics::Metrics;
    use crate::registry::SchemaCache;
    use http_body_util::BodyExt;
    use tower::ServiceExt;

    #[tokio::test]
    async fn health_and_metrics_routes() {
        let app = router(Arc::new(Ingestor::new(
            Arc::new(RecordingOutbox::new()),
            Arc::new(SchemaCache::default()),
            Metrics::new(),
            "telemetry-events".to_owned(),
            true,
        )));
        let health = app
            .clone()
            .oneshot(
                Request::builder()
                    .uri("/health")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(health.status(), StatusCode::OK);
        let metrics = app
            .oneshot(
                Request::builder()
                    .uri("/metrics")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(metrics.status(), StatusCode::OK);
        let bytes = metrics.into_body().collect().await.unwrap().to_bytes();
        assert!(String::from_utf8_lossy(&bytes).contains("ingest_total"));
    }

    fn app() -> Router {
        router(Arc::new(Ingestor::new(
            Arc::new(RecordingOutbox::new()),
            Arc::new(SchemaCache::default()),
            Metrics::new(),
            "telemetry-events".to_owned(),
            true,
        )))
    }

    async fn post(app: Router, body: Body) -> (StatusCode, String) {
        let response = app
            .oneshot(
                Request::builder()
                    .method("POST")
                    .uri("/v2/telemetry")
                    .header("content-type", "application/json")
                    .header("x-bidon-version", "0.7.0")
                    .body(body)
                    .unwrap(),
            )
            .await
            .unwrap();
        let status = response.status();
        let bytes = response.into_body().collect().await.unwrap().to_bytes();
        (status, String::from_utf8_lossy(&bytes).into_owned())
    }

    #[tokio::test]
    async fn post_valid_batch_is_202() {
        let body = include_str!("../testdata/valid_request.json");
        let (status, text) = post(app(), Body::from(body)).await;
        assert_eq!(status, StatusCode::ACCEPTED);
        assert!(text.starts_with(r#"{"accepted":"#));
    }

    #[tokio::test]
    async fn post_malformed_batch_is_400_problem_json() {
        let (status, text) = post(app(), Body::from("not-json")).await;
        assert_eq!(status, StatusCode::BAD_REQUEST);
        assert!(text.contains(r#""code":400"#));
    }

    #[tokio::test]
    async fn post_oversized_streamed_body_is_413() {
        let chunk = vec![b' '; 64 << 10];
        let chunks: Vec<Result<Vec<u8>, std::io::Error>> =
            (0..5).map(|_| Ok(chunk.clone())).collect();
        let body = Body::from_stream(futures_util::stream::iter(chunks));
        let (status, text) = post(app(), body).await;
        assert_eq!(status, StatusCode::PAYLOAD_TOO_LARGE);
        assert!(text.contains("too large"));
    }
}
