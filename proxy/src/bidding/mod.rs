mod echo;
mod proxy;

use derive_more::Display;
use derive_new::new;
pub use echo::EchoBiddingService;
pub use proxy::ProxyBiddingService;
use thiserror::Error;
use tonic::Status;

use crate::com::iabtechlab::openrtb::v3::Openrtb;

#[async_trait::async_trait]
pub trait BiddingService {
    /// Bidding service
    async fn bid(&self, request: Openrtb) -> Result<Openrtb, Status>;
}
