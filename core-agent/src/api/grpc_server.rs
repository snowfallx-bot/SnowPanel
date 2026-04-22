use std::net::SocketAddr;
use std::sync::Arc;

use anyhow::{Context, Result};
use tonic::{Request, Response, Status};
use tracing::info;

use crate::api::proto::health_service_server::{HealthService, HealthServiceServer};
use crate::api::proto::system_service_server::{SystemService, SystemServiceServer};
use crate::api::proto::{
    Error, GetRealtimeResourceRequest, GetRealtimeResourceResponse, GetSystemOverviewRequest,
    GetSystemOverviewResponse, HealthCheckRequest, HealthCheckResponse,
};
use crate::service::system_info::SystemInfoService;

#[derive(Clone)]
pub struct GrpcServer {
    system_info_service: Arc<SystemInfoService>,
}

impl GrpcServer {
    pub fn new() -> Self {
        Self {
            system_info_service: Arc::new(SystemInfoService::new()),
        }
    }

    pub async fn run(&self, addr: &str) -> Result<()> {
        let socket_addr: SocketAddr = addr
            .parse()
            .with_context(|| format!("invalid listen address: {addr}"))?;

        info!("core-agent grpc server listening on {}", socket_addr);

        tonic::transport::Server::builder()
            .add_service(HealthServiceServer::new(HealthServiceImpl))
            .add_service(SystemServiceServer::new(SystemServiceImpl {
                system_info_service: self.system_info_service.clone(),
            }))
            .serve(socket_addr)
            .await
            .with_context(|| format!("grpc server terminated unexpectedly on {socket_addr}"))?;

        Ok(())
    }
}

#[derive(Default)]
struct HealthServiceImpl;

#[tonic::async_trait]
impl HealthService for HealthServiceImpl {
    async fn check(
        &self,
        _request: Request<HealthCheckRequest>,
    ) -> Result<Response<HealthCheckResponse>, Status> {
        Ok(Response::new(HealthCheckResponse {
            error: None,
            status: "SERVING".to_string(),
        }))
    }
}

#[derive(Clone)]
struct SystemServiceImpl {
    system_info_service: Arc<SystemInfoService>,
}

#[tonic::async_trait]
impl SystemService for SystemServiceImpl {
    async fn get_system_overview(
        &self,
        _request: Request<GetSystemOverviewRequest>,
    ) -> Result<Response<GetSystemOverviewResponse>, Status> {
        let overview = self.system_info_service.get_overview();
        Ok(Response::new(GetSystemOverviewResponse {
            error: Some(Error {
                code: 0,
                message: "ok".to_string(),
                detail: String::new(),
            }),
            overview: Some(overview),
        }))
    }

    async fn get_realtime_resource(
        &self,
        _request: Request<GetRealtimeResourceRequest>,
    ) -> Result<Response<GetRealtimeResourceResponse>, Status> {
        let resource = self.system_info_service.get_realtime_resource();
        Ok(Response::new(GetRealtimeResourceResponse {
            error: Some(Error {
                code: 0,
                message: "ok".to_string(),
                detail: String::new(),
            }),
            resource: Some(resource),
        }))
    }
}
