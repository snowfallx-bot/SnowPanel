use std::net::SocketAddr;
use std::path::PathBuf;
use std::sync::Arc;

use anyhow::{Context, Result};
use tonic::{Request, Response, Status};
use tracing::info;

use crate::api::proto::file_service_server::{FileService, FileServiceServer};
use crate::api::proto::health_service_server::{HealthService, HealthServiceServer};
use crate::api::proto::system_service_server::{SystemService, SystemServiceServer};
use crate::api::proto::{
    CreateDirectoryRequest, CreateDirectoryResponse, DeleteFileRequest, DeleteFileResponse, Error,
    GetRealtimeResourceRequest, GetRealtimeResourceResponse, GetSystemOverviewRequest,
    GetSystemOverviewResponse, HealthCheckRequest, HealthCheckResponse, ListFilesRequest,
    ListFilesResponse, ReadTextFileRequest, ReadTextFileResponse, WriteTextFileRequest,
    WriteTextFileResponse,
};
use crate::file::service::FileService as FileOperatorService;
use crate::security::path_validator::PathValidator;
use crate::service::system_info::SystemInfoService;

#[derive(Clone)]
pub struct GrpcServer {
    system_info_service: Arc<SystemInfoService>,
    file_service: Arc<FileOperatorService>,
}

impl GrpcServer {
    pub fn new(allowed_roots: Vec<String>, max_read_bytes: usize, max_write_bytes: usize) -> Self {
        let roots = allowed_roots.into_iter().map(PathBuf::from).collect::<Vec<_>>();
        let path_validator = PathValidator::new(roots);

        Self {
            system_info_service: Arc::new(SystemInfoService::new()),
            file_service: Arc::new(FileOperatorService::new(
                path_validator,
                max_read_bytes,
                max_write_bytes,
            )),
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
            .add_service(FileServiceServer::new(FileServiceImpl {
                file_service: self.file_service.clone(),
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
            error: Some(ok_error()),
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
            error: Some(ok_error()),
            overview: Some(overview),
        }))
    }

    async fn get_realtime_resource(
        &self,
        _request: Request<GetRealtimeResourceRequest>,
    ) -> Result<Response<GetRealtimeResourceResponse>, Status> {
        let resource = self.system_info_service.get_realtime_resource();
        Ok(Response::new(GetRealtimeResourceResponse {
            error: Some(ok_error()),
            resource: Some(resource),
        }))
    }
}

#[derive(Clone)]
struct FileServiceImpl {
    file_service: Arc<FileOperatorService>,
}

#[tonic::async_trait]
impl FileService for FileServiceImpl {
    async fn list_files(
        &self,
        request: Request<ListFilesRequest>,
    ) -> Result<Response<ListFilesResponse>, Status> {
        let payload = request.into_inner();
        Ok(Response::new(
            self.file_service.list_files(&payload.path, payload.safety),
        ))
    }

    async fn read_text_file(
        &self,
        request: Request<ReadTextFileRequest>,
    ) -> Result<Response<ReadTextFileResponse>, Status> {
        let payload = request.into_inner();
        Ok(Response::new(self.file_service.read_text_file(
            &payload.path,
            payload.max_bytes,
            &payload.encoding,
            payload.safety,
        )))
    }

    async fn write_text_file(
        &self,
        request: Request<WriteTextFileRequest>,
    ) -> Result<Response<WriteTextFileResponse>, Status> {
        let payload = request.into_inner();
        Ok(Response::new(self.file_service.write_text_file(
            &payload.path,
            &payload.content,
            payload.create_if_not_exists,
            payload.truncate,
            &payload.encoding,
            payload.safety,
        )))
    }

    async fn create_directory(
        &self,
        request: Request<CreateDirectoryRequest>,
    ) -> Result<Response<CreateDirectoryResponse>, Status> {
        let payload = request.into_inner();
        Ok(Response::new(self.file_service.create_directory(
            &payload.path,
            payload.create_parents,
            payload.safety,
        )))
    }

    async fn delete_file(
        &self,
        request: Request<DeleteFileRequest>,
    ) -> Result<Response<DeleteFileResponse>, Status> {
        let payload = request.into_inner();
        Ok(Response::new(self.file_service.delete_path(
            &payload.path,
            payload.recursive,
            payload.safety,
        )))
    }
}

fn ok_error() -> Error {
    Error {
        code: 0,
        message: "ok".to_string(),
        detail: String::new(),
    }
}
