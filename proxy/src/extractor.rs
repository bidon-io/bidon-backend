use axum::async_trait;
use axum::extract::rejection::JsonRejection;
use axum::extract::{FromRequest, Json, Request};
use axum::http::StatusCode;
use axum::response::{IntoResponse, Response};

use crate::adapter;
use crate::com::iabtechlab::openrtb::v3::Openrtb;
use crate::models::AuctionRequest;

pub struct BidonOpenRtbExtractor(pub Openrtb);

pub enum OpenRtbExtractorRejection {
    MissingBidonVersionHeader,
    InvalidJson(JsonRejection),
    InvalidBiddingRequest,
}

impl IntoResponse for OpenRtbExtractorRejection {
    fn into_response(self) -> Response {
        match self {
            OpenRtbExtractorRejection::InvalidJson(json_error) => (
                StatusCode::BAD_REQUEST,
                format!("Invalid JSON in request body: {:?}", json_error),
            )
                .into_response(),
            OpenRtbExtractorRejection::InvalidBiddingRequest => {
                (StatusCode::BAD_REQUEST, "Invalid bidding request").into_response()
            }
            OpenRtbExtractorRejection::MissingBidonVersionHeader => {
                (StatusCode::BAD_REQUEST, "Missing x-bidon-version header").into_response()
            }
        }
    }
}

#[async_trait]
impl<B> FromRequest<B> for BidonOpenRtbExtractor
where
    B: Send + Sync + 'static,
{
    type Rejection = OpenRtbExtractorRejection;

    async fn from_request(req: Request, state: &B) -> Result<Self, Self::Rejection> {
        /// Header - `x-bidon-version` - version of the bidon server.
        const X_BIDON_VERSION_HEADER: &str = "x-bidon-version";

        let bidon_version = req
            .headers()
            .get(X_BIDON_VERSION_HEADER)
            .and_then(|x| x.to_str().ok())
            .map(|bidon_version| bidon_version.to_string())
            // Return an error response if the header is missing
            .ok_or(OpenRtbExtractorRejection::InvalidBiddingRequest)?;

        let Json(auction_request) = Json::<AuctionRequest>::from_request(req, state)
            .await
            .map_err(OpenRtbExtractorRejection::InvalidJson)?;

        let openrtb_request = adapter::try_from(auction_request, &bidon_version)
            .map_err(|_| OpenRtbExtractorRejection::InvalidBiddingRequest)?;

        Ok(BidonOpenRtbExtractor(openrtb_request))
    }
}
