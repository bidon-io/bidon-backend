use crate::bidding::BiddingError;
use crate::bidding::BiddingService;
use crate::org::bidon::proto::v1::bidding_service_client::BiddingServiceClient;
use crate::com::iabtechlab::openrtb::v3::Openrtb;
use tonic::transport::Channel;
use tonic::Request;

#[derive(Debug, Clone)]
pub struct ProxyBiddingService {
    grpc_client: Option<BiddingServiceClient<Channel>>,
    grpc_url: &'static str,
}

impl ProxyBiddingService {
    pub fn new(grpc_url: &'static str,) -> Self {
        ProxyBiddingService {
            grpc_client: None,
            grpc_url,
        }
    }

    pub async fn connect(&mut self) -> Result<(), Box<dyn std::error::Error>> {
        let grpc_client = BiddingServiceClient::connect(self.grpc_url).await?;
        self.grpc_client = Some(grpc_client);
        Ok(())
    }
}

#[async_trait::async_trait]
impl BiddingService for ProxyBiddingService {
    async fn bid(&self, request: Openrtb) -> Result<Openrtb, BiddingError> {
        let grpc_request = Request::new(request);
        let grpc_client = self.grpc_client.as_ref().ok_or_else(|| BiddingError::new("gRPC client not initialized".to_string()))?;
        let grpc_response = grpc_client
            .clone() // Cloning is required here, because Tonic gRPC clients are mutable. Cloning is cheap.
            .bid(grpc_request)
            .await
            .map_err(|e| BiddingError::new(e.to_string()))?;
        Ok(grpc_response.into_inner())
    }
}
