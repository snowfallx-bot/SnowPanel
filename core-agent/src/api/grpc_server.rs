use std::future::Future;
use std::net::SocketAddr;
use std::path::PathBuf;
use std::sync::Arc;

use anyhow::{Context, Result};
use tonic::metadata::MetadataMap;
use tonic::{Request, Response, Status};
use tracing::info;

use crate::api::proto::cron_service_server::{CronService as CronGrpcService, CronServiceServer};
use crate::api::proto::docker_service_server::{
    DockerService as DockerGrpcService, DockerServiceServer,
};
use crate::api::proto::file_service_server::{FileService, FileServiceServer};
use crate::api::proto::health_service_server::{HealthService, HealthServiceServer};
use crate::api::proto::service_manager_service_server::{
    ServiceManagerService, ServiceManagerServiceServer,
};
use crate::api::proto::system_service_server::{SystemService, SystemServiceServer};
use crate::api::proto::{
    CreateCronTaskRequest, CreateCronTaskResponse, CreateDirectoryRequest, CreateDirectoryResponse,
    CronTask, DeleteCronTaskRequest, DeleteCronTaskResponse, DeleteFileRequest, DeleteFileResponse,
    DockerContainerActionRequest, DockerContainerActionResponse, DockerContainerInfo,
    DockerImageInfo, Error, GetRealtimeResourceRequest, GetRealtimeResourceResponse,
    GetSystemOverviewRequest, GetSystemOverviewResponse, HealthCheckRequest, HealthCheckResponse,
    ListCronTasksRequest, ListCronTasksResponse, ListDockerContainersRequest,
    ListDockerContainersResponse, ListDockerImagesRequest, ListDockerImagesResponse,
    ListFilesRequest, ListFilesResponse, ListServicesRequest, ListServicesResponse,
    ReadFileChunkRequest, ReadFileChunkResponse, ReadTextFileRequest, ReadTextFileResponse,
    RenameFileRequest, RenameFileResponse, ServiceActionRequest, ServiceActionResponse,
    ServiceInfo, SetCronTaskEnabledRequest, SetCronTaskEnabledResponse, UpdateCronTaskRequest,
    UpdateCronTaskResponse, WriteFileChunkRequest, WriteFileChunkResponse, WriteTextFileRequest,
    WriteTextFileResponse,
};
use crate::cron::service::{CronError, CronService};
use crate::docker::service::{DockerAction, DockerError, DockerService};
use crate::file::service::FileService as FileOperatorService;
use crate::observability::metrics;
use crate::process::systemd_service::{ServiceAction, ServiceError, SystemdServiceManager};
use crate::security::path_validator::PathValidator;
use crate::service::system_info::SystemInfoService;

#[derive(Clone)]
pub struct GrpcServer {
    system_info_service: Arc<SystemInfoService>,
    file_service: Arc<FileOperatorService>,
    service_manager: Arc<SystemdServiceManager>,
    docker_service: Arc<DockerService>,
    cron_service: Arc<CronService>,
}

impl GrpcServer {
    pub fn new(
        allowed_roots: Vec<String>,
        max_read_bytes: usize,
        max_write_bytes: usize,
        service_whitelist: Vec<String>,
        cron_allowed_commands: Vec<String>,
    ) -> Result<Self> {
        let roots = allowed_roots
            .into_iter()
            .map(PathBuf::from)
            .collect::<Vec<_>>();
        let path_validator = PathValidator::new(roots);
        let docker_service = DockerService::new().context("initialize docker service failed")?;

        Ok(Self {
            system_info_service: Arc::new(SystemInfoService::new()),
            file_service: Arc::new(FileOperatorService::new(
                path_validator,
                max_read_bytes,
                max_write_bytes,
            )),
            service_manager: Arc::new(SystemdServiceManager::new(service_whitelist)),
            docker_service: Arc::new(docker_service),
            cron_service: Arc::new(CronService::new(cron_allowed_commands)),
        })
    }

    pub async fn run(&self, addr: &str) -> Result<()> {
        let socket_addr: SocketAddr = addr
            .parse()
            .with_context(|| format!("invalid listen address: {addr}"))?;

        info!("core-agent grpc server listening on {}", socket_addr);

        tonic::transport::Server::builder()
            .add_service(HealthServiceServer::with_interceptor(
                HealthServiceImpl,
                request_logging_interceptor,
            ))
            .add_service(SystemServiceServer::with_interceptor(
                SystemServiceImpl {
                    system_info_service: self.system_info_service.clone(),
                },
                request_logging_interceptor,
            ))
            .add_service(FileServiceServer::with_interceptor(
                FileServiceImpl {
                    file_service: self.file_service.clone(),
                },
                request_logging_interceptor,
            ))
            .add_service(ServiceManagerServiceServer::with_interceptor(
                ServiceManagerServiceImpl {
                    service_manager: self.service_manager.clone(),
                },
                request_logging_interceptor,
            ))
            .add_service(DockerServiceServer::with_interceptor(
                DockerServiceImpl {
                    docker_service: self.docker_service.clone(),
                },
                request_logging_interceptor,
            ))
            .add_service(CronServiceServer::with_interceptor(
                CronServiceImpl {
                    cron_service: self.cron_service.clone(),
                },
                request_logging_interceptor,
            ))
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
        observe_grpc_call("/snowpanel.agent.v1.HealthService/Check", async move {
            Ok(Response::new(HealthCheckResponse {
                error: Some(ok_error()),
                status: "SERVING".to_string(),
            }))
        })
        .await
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
        observe_grpc_call("/snowpanel.agent.v1.SystemService/GetSystemOverview", async move {
            let overview = self.system_info_service.get_overview();
            Ok(Response::new(GetSystemOverviewResponse {
                error: Some(ok_error()),
                overview: Some(overview),
            }))
        })
        .await
    }

    async fn get_realtime_resource(
        &self,
        _request: Request<GetRealtimeResourceRequest>,
    ) -> Result<Response<GetRealtimeResourceResponse>, Status> {
        observe_grpc_call("/snowpanel.agent.v1.SystemService/GetRealtimeResource", async move {
            let resource = self.system_info_service.get_realtime_resource();
            Ok(Response::new(GetRealtimeResourceResponse {
                error: Some(ok_error()),
                resource: Some(resource),
            }))
        })
        .await
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
        observe_grpc_call("/snowpanel.agent.v1.FileService/ListFiles", async move {
            let payload = request.into_inner();
            Ok(Response::new(
                self.file_service.list_files(&payload.path, payload.safety),
            ))
        })
        .await
    }

    async fn read_text_file(
        &self,
        request: Request<ReadTextFileRequest>,
    ) -> Result<Response<ReadTextFileResponse>, Status> {
        observe_grpc_call("/snowpanel.agent.v1.FileService/ReadTextFile", async move {
            let payload = request.into_inner();
            Ok(Response::new(self.file_service.read_text_file(
                &payload.path,
                payload.max_bytes,
                &payload.encoding,
                payload.safety,
            )))
        })
        .await
    }

    async fn read_file_chunk(
        &self,
        request: Request<ReadFileChunkRequest>,
    ) -> Result<Response<ReadFileChunkResponse>, Status> {
        observe_grpc_call("/snowpanel.agent.v1.FileService/ReadFileChunk", async move {
            let payload = request.into_inner();
            Ok(Response::new(self.file_service.read_file_chunk(
                &payload.path,
                payload.offset,
                payload.limit,
                payload.safety,
            )))
        })
        .await
    }

    async fn write_text_file(
        &self,
        request: Request<WriteTextFileRequest>,
    ) -> Result<Response<WriteTextFileResponse>, Status> {
        observe_grpc_call("/snowpanel.agent.v1.FileService/WriteTextFile", async move {
            let payload = request.into_inner();
            Ok(Response::new(self.file_service.write_text_file(
                &payload.path,
                &payload.content,
                payload.create_if_not_exists,
                payload.truncate,
                &payload.encoding,
                payload.safety,
            )))
        })
        .await
    }

    async fn write_file_chunk(
        &self,
        request: Request<WriteFileChunkRequest>,
    ) -> Result<Response<WriteFileChunkResponse>, Status> {
        observe_grpc_call("/snowpanel.agent.v1.FileService/WriteFileChunk", async move {
            let payload = request.into_inner();
            Ok(Response::new(self.file_service.write_file_chunk(
                &payload.path,
                payload.offset,
                &payload.chunk,
                payload.create_if_not_exists,
                payload.truncate,
                payload.safety,
            )))
        })
        .await
    }

    async fn create_directory(
        &self,
        request: Request<CreateDirectoryRequest>,
    ) -> Result<Response<CreateDirectoryResponse>, Status> {
        observe_grpc_call("/snowpanel.agent.v1.FileService/CreateDirectory", async move {
            let payload = request.into_inner();
            Ok(Response::new(self.file_service.create_directory(
                &payload.path,
                payload.create_parents,
                payload.safety,
            )))
        })
        .await
    }

    async fn delete_file(
        &self,
        request: Request<DeleteFileRequest>,
    ) -> Result<Response<DeleteFileResponse>, Status> {
        observe_grpc_call("/snowpanel.agent.v1.FileService/DeleteFile", async move {
            let payload = request.into_inner();
            Ok(Response::new(self.file_service.delete_path(
                &payload.path,
                payload.recursive,
                payload.safety,
            )))
        })
        .await
    }

    async fn rename_file(
        &self,
        request: Request<RenameFileRequest>,
    ) -> Result<Response<RenameFileResponse>, Status> {
        observe_grpc_call("/snowpanel.agent.v1.FileService/RenameFile", async move {
            let payload = request.into_inner();
            Ok(Response::new(self.file_service.rename_file(
                &payload.source_path,
                &payload.target_path,
                payload.safety,
            )))
        })
        .await
    }
}

fn ok_error() -> Error {
    Error {
        code: 0,
        message: "ok".to_string(),
        detail: String::new(),
    }
}

async fn observe_grpc_call<T, F>(grpc_method: &str, call: F) -> Result<Response<T>, Status>
where
    F: Future<Output = Result<Response<T>, Status>>,
{
    let guard = metrics::start_grpc_call(grpc_method);
    let result = call.await;
    if result.is_ok() {
        guard.finish("ok");
    } else {
        guard.finish("error");
    }
    result
}

fn request_logging_interceptor(request: Request<()>) -> Result<Request<()>, Status> {
    let grpc_method = request.uri().path().to_string();
    let request_id = request_id_from_metadata(request.metadata());

    info!(
        request_id = request_id.as_str(),
        grpc_method = grpc_method.as_str(),
        "core-agent grpc request"
    );

    Ok(request)
}

fn request_id_from_metadata(metadata: &MetadataMap) -> String {
    const HEADER: &str = "x-request-id";
    const MAX_LEN: usize = 128;

    let Some(raw) = metadata.get(HEADER) else {
        return "missing".to_string();
    };

    let Ok(value) = raw.to_str() else {
        return "invalid".to_string();
    };

    let trimmed = value.trim();
    if trimmed.is_empty() {
        return "missing".to_string();
    }
    if trimmed.len() > MAX_LEN {
        return trimmed.chars().take(MAX_LEN).collect::<String>();
    }
    trimmed.to_string()
}

#[derive(Clone)]
struct ServiceManagerServiceImpl {
    service_manager: Arc<SystemdServiceManager>,
}

#[tonic::async_trait]
impl ServiceManagerService for ServiceManagerServiceImpl {
    async fn list_services(
        &self,
        request: Request<ListServicesRequest>,
    ) -> Result<Response<ListServicesResponse>, Status> {
        observe_grpc_call(
            "/snowpanel.agent.v1.ServiceManagerService/ListServices",
            async move {
            let payload = request.into_inner();
            let result = self.service_manager.list_services(&payload.keyword);
            match result {
                Ok(items) => Ok(Response::new(ListServicesResponse {
                    error: Some(ok_error()),
                    services: items
                        .into_iter()
                        .map(|item| ServiceInfo {
                            name: item.name,
                            display_name: item.display_name,
                            status: item.status,
                        })
                        .collect::<Vec<_>>(),
                })),
                Err(err) => Ok(Response::new(ListServicesResponse {
                    error: Some(to_error(err)),
                    services: Vec::new(),
                })),
            }
        },
        )
        .await
    }

    async fn start_service(
        &self,
        request: Request<ServiceActionRequest>,
    ) -> Result<Response<ServiceActionResponse>, Status> {
        observe_grpc_call("/snowpanel.agent.v1.ServiceManagerService/StartService", async move {
            self.handle_action(ServiceAction::Start, request.into_inner())
        })
        .await
    }

    async fn stop_service(
        &self,
        request: Request<ServiceActionRequest>,
    ) -> Result<Response<ServiceActionResponse>, Status> {
        observe_grpc_call("/snowpanel.agent.v1.ServiceManagerService/StopService", async move {
            self.handle_action(ServiceAction::Stop, request.into_inner())
        })
        .await
    }

    async fn restart_service(
        &self,
        request: Request<ServiceActionRequest>,
    ) -> Result<Response<ServiceActionResponse>, Status> {
        observe_grpc_call(
            "/snowpanel.agent.v1.ServiceManagerService/RestartService",
            async move { self.handle_action(ServiceAction::Restart, request.into_inner()) },
        )
        .await
    }
}

impl ServiceManagerServiceImpl {
    fn handle_action(
        &self,
        action: ServiceAction,
        req: ServiceActionRequest,
    ) -> Result<Response<ServiceActionResponse>, Status> {
        let result = self.service_manager.run_action(action, &req.name);
        match result {
            Ok(item) => Ok(Response::new(ServiceActionResponse {
                error: Some(ok_error()),
                name: item.name,
                status: item.status,
            })),
            Err(err) => Ok(Response::new(ServiceActionResponse {
                error: Some(to_error(err)),
                name: req.name,
                status: String::new(),
            })),
        }
    }
}

fn to_error(err: ServiceError) -> Error {
    Error {
        code: err.code,
        message: err.message,
        detail: err.detail,
    }
}

#[derive(Clone)]
struct DockerServiceImpl {
    docker_service: Arc<DockerService>,
}

#[tonic::async_trait]
impl DockerGrpcService for DockerServiceImpl {
    async fn list_containers(
        &self,
        _request: Request<ListDockerContainersRequest>,
    ) -> Result<Response<ListDockerContainersResponse>, Status> {
        observe_grpc_call("/snowpanel.agent.v1.DockerService/ListContainers", async move {
            let result = self.docker_service.list_containers().await;
            match result {
                Ok(containers) => Ok(Response::new(ListDockerContainersResponse {
                    error: Some(ok_error()),
                    containers: containers
                        .into_iter()
                        .map(|item| DockerContainerInfo {
                            id: item.id,
                            name: item.name,
                            image: item.image,
                            state: item.state,
                            status: item.status,
                        })
                        .collect::<Vec<_>>(),
                })),
                Err(err) => Ok(Response::new(ListDockerContainersResponse {
                    error: Some(to_docker_error(err)),
                    containers: Vec::new(),
                })),
            }
        })
        .await
    }

    async fn start_container(
        &self,
        request: Request<DockerContainerActionRequest>,
    ) -> Result<Response<DockerContainerActionResponse>, Status> {
        observe_grpc_call("/snowpanel.agent.v1.DockerService/StartContainer", async move {
            self.handle_action(DockerAction::Start, request.into_inner())
                .await
        })
        .await
    }

    async fn stop_container(
        &self,
        request: Request<DockerContainerActionRequest>,
    ) -> Result<Response<DockerContainerActionResponse>, Status> {
        observe_grpc_call("/snowpanel.agent.v1.DockerService/StopContainer", async move {
            self.handle_action(DockerAction::Stop, request.into_inner())
                .await
        })
        .await
    }

    async fn restart_container(
        &self,
        request: Request<DockerContainerActionRequest>,
    ) -> Result<Response<DockerContainerActionResponse>, Status> {
        observe_grpc_call("/snowpanel.agent.v1.DockerService/RestartContainer", async move {
            self.handle_action(DockerAction::Restart, request.into_inner())
                .await
        })
        .await
    }

    async fn list_images(
        &self,
        _request: Request<ListDockerImagesRequest>,
    ) -> Result<Response<ListDockerImagesResponse>, Status> {
        observe_grpc_call("/snowpanel.agent.v1.DockerService/ListImages", async move {
            let result = self.docker_service.list_images().await;
            match result {
                Ok(images) => Ok(Response::new(ListDockerImagesResponse {
                    error: Some(ok_error()),
                    images: images
                        .into_iter()
                        .map(|item| DockerImageInfo {
                            id: item.id,
                            repo_tags: item.repo_tags,
                            size: item.size,
                        })
                        .collect::<Vec<_>>(),
                })),
                Err(err) => Ok(Response::new(ListDockerImagesResponse {
                    error: Some(to_docker_error(err)),
                    images: Vec::new(),
                })),
            }
        })
        .await
    }
}

impl DockerServiceImpl {
    async fn handle_action(
        &self,
        action: DockerAction,
        payload: DockerContainerActionRequest,
    ) -> Result<Response<DockerContainerActionResponse>, Status> {
        let result = self.docker_service.run_action(action, &payload.id).await;
        match result {
            Ok(container) => Ok(Response::new(DockerContainerActionResponse {
                error: Some(ok_error()),
                id: container.id,
                state: container.state,
            })),
            Err(err) => Ok(Response::new(DockerContainerActionResponse {
                error: Some(to_docker_error(err)),
                id: payload.id,
                state: String::new(),
            })),
        }
    }
}

fn to_docker_error(err: DockerError) -> Error {
    Error {
        code: err.code,
        message: err.message,
        detail: err.detail,
    }
}

#[derive(Clone)]
struct CronServiceImpl {
    cron_service: Arc<CronService>,
}

#[tonic::async_trait]
impl CronGrpcService for CronServiceImpl {
    async fn list_cron_tasks(
        &self,
        _request: Request<ListCronTasksRequest>,
    ) -> Result<Response<ListCronTasksResponse>, Status> {
        observe_grpc_call("/snowpanel.agent.v1.CronService/ListCronTasks", async move {
            let result = self.cron_service.list_tasks();
            match result {
                Ok(tasks) => Ok(Response::new(ListCronTasksResponse {
                    error: Some(ok_error()),
                    tasks: tasks
                        .into_iter()
                        .map(to_proto_cron_task)
                        .collect::<Vec<_>>(),
                })),
                Err(err) => Ok(Response::new(ListCronTasksResponse {
                    error: Some(to_cron_error(err)),
                    tasks: Vec::new(),
                })),
            }
        })
        .await
    }

    async fn create_cron_task(
        &self,
        request: Request<CreateCronTaskRequest>,
    ) -> Result<Response<CreateCronTaskResponse>, Status> {
        observe_grpc_call("/snowpanel.agent.v1.CronService/CreateCronTask", async move {
            let payload = request.into_inner();
            let result =
                self.cron_service
                    .create_task(&payload.expression, &payload.command, payload.enabled);

            match result {
                Ok(task) => Ok(Response::new(CreateCronTaskResponse {
                    error: Some(ok_error()),
                    task: Some(to_proto_cron_task(task)),
                })),
                Err(err) => Ok(Response::new(CreateCronTaskResponse {
                    error: Some(to_cron_error(err)),
                    task: None,
                })),
            }
        })
        .await
    }

    async fn update_cron_task(
        &self,
        request: Request<UpdateCronTaskRequest>,
    ) -> Result<Response<UpdateCronTaskResponse>, Status> {
        observe_grpc_call("/snowpanel.agent.v1.CronService/UpdateCronTask", async move {
            let payload = request.into_inner();
            let result = self.cron_service.update_task(
                &payload.id,
                &payload.expression,
                &payload.command,
                payload.enabled,
            );

            match result {
                Ok(task) => Ok(Response::new(UpdateCronTaskResponse {
                    error: Some(ok_error()),
                    task: Some(to_proto_cron_task(task)),
                })),
                Err(err) => Ok(Response::new(UpdateCronTaskResponse {
                    error: Some(to_cron_error(err)),
                    task: None,
                })),
            }
        })
        .await
    }

    async fn delete_cron_task(
        &self,
        request: Request<DeleteCronTaskRequest>,
    ) -> Result<Response<DeleteCronTaskResponse>, Status> {
        observe_grpc_call("/snowpanel.agent.v1.CronService/DeleteCronTask", async move {
            let payload = request.into_inner();
            let result = self.cron_service.delete_task(&payload.id);
            match result {
                Ok(()) => Ok(Response::new(DeleteCronTaskResponse {
                    error: Some(ok_error()),
                    id: payload.id,
                })),
                Err(err) => Ok(Response::new(DeleteCronTaskResponse {
                    error: Some(to_cron_error(err)),
                    id: String::new(),
                })),
            }
        })
        .await
    }

    async fn set_cron_task_enabled(
        &self,
        request: Request<SetCronTaskEnabledRequest>,
    ) -> Result<Response<SetCronTaskEnabledResponse>, Status> {
        observe_grpc_call("/snowpanel.agent.v1.CronService/SetCronTaskEnabled", async move {
            let payload = request.into_inner();
            let result = self.cron_service.set_enabled(&payload.id, payload.enabled);
            match result {
                Ok(task) => Ok(Response::new(SetCronTaskEnabledResponse {
                    error: Some(ok_error()),
                    task: Some(to_proto_cron_task(task)),
                })),
                Err(err) => Ok(Response::new(SetCronTaskEnabledResponse {
                    error: Some(to_cron_error(err)),
                    task: None,
                })),
            }
        })
        .await
    }
}

fn to_cron_error(err: CronError) -> Error {
    Error {
        code: err.code,
        message: err.message,
        detail: err.detail,
    }
}

fn to_proto_cron_task(task: crate::cron::service::CronTaskEntity) -> CronTask {
    CronTask {
        id: task.id,
        expression: task.expression,
        command: task.command,
        enabled: task.enabled,
    }
}
