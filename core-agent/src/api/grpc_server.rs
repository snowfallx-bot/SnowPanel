use std::net::SocketAddr;
use std::path::PathBuf;
use std::sync::Arc;

use anyhow::{Context, Result};
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
    ReadTextFileRequest, ReadTextFileResponse, ServiceActionRequest, ServiceActionResponse,
    ServiceInfo, SetCronTaskEnabledRequest, SetCronTaskEnabledResponse, UpdateCronTaskRequest,
    UpdateCronTaskResponse, WriteTextFileRequest, WriteTextFileResponse,
};
use crate::cron::service::{CronError, CronService};
use crate::docker::service::{DockerAction, DockerError, DockerService};
use crate::file::service::FileService as FileOperatorService;
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
            .add_service(HealthServiceServer::new(HealthServiceImpl))
            .add_service(SystemServiceServer::new(SystemServiceImpl {
                system_info_service: self.system_info_service.clone(),
            }))
            .add_service(FileServiceServer::new(FileServiceImpl {
                file_service: self.file_service.clone(),
            }))
            .add_service(ServiceManagerServiceServer::new(
                ServiceManagerServiceImpl {
                    service_manager: self.service_manager.clone(),
                },
            ))
            .add_service(DockerServiceServer::new(DockerServiceImpl {
                docker_service: self.docker_service.clone(),
            }))
            .add_service(CronServiceServer::new(CronServiceImpl {
                cron_service: self.cron_service.clone(),
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
    }

    async fn start_service(
        &self,
        request: Request<ServiceActionRequest>,
    ) -> Result<Response<ServiceActionResponse>, Status> {
        self.handle_action(ServiceAction::Start, request.into_inner())
    }

    async fn stop_service(
        &self,
        request: Request<ServiceActionRequest>,
    ) -> Result<Response<ServiceActionResponse>, Status> {
        self.handle_action(ServiceAction::Stop, request.into_inner())
    }

    async fn restart_service(
        &self,
        request: Request<ServiceActionRequest>,
    ) -> Result<Response<ServiceActionResponse>, Status> {
        self.handle_action(ServiceAction::Restart, request.into_inner())
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
    }

    async fn start_container(
        &self,
        request: Request<DockerContainerActionRequest>,
    ) -> Result<Response<DockerContainerActionResponse>, Status> {
        self.handle_action(DockerAction::Start, request.into_inner())
            .await
    }

    async fn stop_container(
        &self,
        request: Request<DockerContainerActionRequest>,
    ) -> Result<Response<DockerContainerActionResponse>, Status> {
        self.handle_action(DockerAction::Stop, request.into_inner())
            .await
    }

    async fn restart_container(
        &self,
        request: Request<DockerContainerActionRequest>,
    ) -> Result<Response<DockerContainerActionResponse>, Status> {
        self.handle_action(DockerAction::Restart, request.into_inner())
            .await
    }

    async fn list_images(
        &self,
        _request: Request<ListDockerImagesRequest>,
    ) -> Result<Response<ListDockerImagesResponse>, Status> {
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
    }

    async fn create_cron_task(
        &self,
        request: Request<CreateCronTaskRequest>,
    ) -> Result<Response<CreateCronTaskResponse>, Status> {
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
    }

    async fn update_cron_task(
        &self,
        request: Request<UpdateCronTaskRequest>,
    ) -> Result<Response<UpdateCronTaskResponse>, Status> {
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
    }

    async fn delete_cron_task(
        &self,
        request: Request<DeleteCronTaskRequest>,
    ) -> Result<Response<DeleteCronTaskResponse>, Status> {
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
    }

    async fn set_cron_task_enabled(
        &self,
        request: Request<SetCronTaskEnabledRequest>,
    ) -> Result<Response<SetCronTaskEnabledResponse>, Status> {
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
