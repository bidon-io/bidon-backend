use crate::adapter;
use crate::bidding::BiddingService;
use crate::extract::AuctionRequestPayload;
use axum::extract::State;
use axum::{
    http::StatusCode,
    response::{IntoResponse, Json},
};

pub async fn get_auction_handler<S>(
    State(bidding_service): State<Box<S>>,
    AuctionRequestPayload(request): AuctionRequestPayload,
) -> impl IntoResponse
where
    S: BiddingService + Send + Sync,
{
    match bidding_service.bid(request).await {
        Ok(response) => {
            tracing::trace!("Bidding response: {:?}", response);
            match adapter::try_into(response) {
                Ok(auction_response) => Json::from(auction_response).into_response(),
                Err(err) => (StatusCode::INTERNAL_SERVER_ERROR, err.to_string()).into_response(),
            }
        }
        Err(err) => {
            tracing::error!("Bidding error: {:?}", err);

            let http_status = match err.code() {
                // Map gRPC status codes to HTTP status codes, sorted by gRPC status code.
                // Source: https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto
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
            };
            let message = err.message().to_string();

            (http_status, Json::from(message)).into_response()
        }
    }
}
