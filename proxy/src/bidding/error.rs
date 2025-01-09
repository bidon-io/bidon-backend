use axum::{
    http::StatusCode,
    response::{IntoResponse, Response},
    Json,
};
use derive_more::Display;
use serde_json::json;
use tonic::Status;

#[derive(Debug, Display)]
pub enum BiddingError {
    #[display("gRPC error: {}", _0)]
    GrpcError(Status),
    #[display("HTTP status: {}, error: {}", _0, _1)]
    HttpError(StatusCode, String),
    #[display("Validation error: {}", _0)]
    ValidationError(String),
    #[display("Serialization error: {}", _0)]
    SerializationError(String),
    #[display("Unexpected error: {}", _0)]
    UnexpectedError(String),
}

impl std::error::Error for BiddingError {}

impl BiddingError {
    /// Maps gRPC status codes to appropriate HTTP status codes
    /// Source: https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto
    fn grpc_to_http_status(code: tonic::Code) -> StatusCode {
        match code {
            tonic::Code::Cancelled => StatusCode::from_u16(499).unwrap(),
            tonic::Code::Unknown => StatusCode::INTERNAL_SERVER_ERROR,
            tonic::Code::InvalidArgument => StatusCode::BAD_REQUEST,
            tonic::Code::DeadlineExceeded => StatusCode::GATEWAY_TIMEOUT,
            tonic::Code::NotFound => StatusCode::NOT_FOUND,
            tonic::Code::AlreadyExists => StatusCode::CONFLICT,
            tonic::Code::PermissionDenied => StatusCode::FORBIDDEN,
            tonic::Code::ResourceExhausted => StatusCode::TOO_MANY_REQUESTS,
            tonic::Code::FailedPrecondition => StatusCode::BAD_REQUEST,
            tonic::Code::Aborted => StatusCode::CONFLICT,
            tonic::Code::OutOfRange => StatusCode::BAD_REQUEST,
            tonic::Code::Unimplemented => StatusCode::NOT_IMPLEMENTED,
            tonic::Code::Internal => StatusCode::INTERNAL_SERVER_ERROR,
            tonic::Code::Unavailable => StatusCode::SERVICE_UNAVAILABLE,
            tonic::Code::DataLoss => StatusCode::INTERNAL_SERVER_ERROR,
            tonic::Code::Unauthenticated => StatusCode::UNAUTHORIZED,
            _ => StatusCode::INTERNAL_SERVER_ERROR,
        }
    }
}

impl IntoResponse for BiddingError {
    fn into_response(self) -> Response {
        let (status_code, error_message) = match self {
            BiddingError::GrpcError(status) => {
                // Try to parse the status message as JSON
                let code = serde_json::from_str::<GrpcError>(status.message())
                    .inspect_err(|e| tracing::debug!("Error parsing gRPC error: {}", e))
                    .ok()
                    .and_then(|e| {
                        StatusCode::from_u16(e.code as u16)
                            .inspect_err(|e| {
                                tracing::debug!("Error parsing gRPC error code: {}", e)
                            })
                            .ok()
                    })
                    .unwrap_or_else(|| Self::grpc_to_http_status(status.code()));
                (code, status.message().to_string())
            }
            BiddingError::HttpError(status, msg) => {
                (status, format!("Upstream HTTP service error: {}", msg))
            }
            BiddingError::ValidationError(msg) => {
                (StatusCode::BAD_REQUEST, format!("Invalid request: {}", msg))
            }
            BiddingError::SerializationError(msg) => (
                StatusCode::BAD_REQUEST,
                format!("Serialization error: {}", msg),
            ),
            BiddingError::UnexpectedError(msg) => (
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("Internal server error: {}", msg),
            ),
        };

        let body = Json(json!({
            "error": error_message,
            "code": status_code.as_u16()
        }));

        (status_code, body).into_response()
    }
}

#[derive(serde::Deserialize)]
pub struct GrpcError {
    pub message: String,
    pub code: i32,
}
