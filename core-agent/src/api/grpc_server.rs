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
    ReadFileChunkRequest, ReadFileChunkResponse, ReadTextFileRequest, ReadTextFileResponse,
    RenameFileRequest, RenameFileResponse, ServiceActionRequest, ServiceActionResponse,
    ServiceInfo, SetCronTaskEnabledRequest, SetCronTaskEnabledResponse, UpdateCronTaskRequest,
    UpdateCronTaskResponse, WriteFileChunkRequest, WriteFileChunkResponse, WriteTextFileRequest,
    WriteTextFileResponse,
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

    async fn read_file_chunk(
        &self,
        request: Request<ReadFileChunkRequest>,
    ) -> Result<Response<ReadFileChunkResponse>, Status> {
        let payload = request.into_inner();
        Ok(Response::new(self.file_service.read_file_chunk(
            &payload.path,
            payload.offset,
            payload.limit,
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

    async fn write_file_chunk(
        &self,
        request: Request<WriteFileChunkRequest>,
    ) -> Result<Response<WriteFileChunkResponse>, Status> {
        let payload = request.into_inner();
        Ok(Response::new(self.file_service.write_file_chunk(
            &payload.path,
            payload.offset,
            &payload.chunk,
            payload.create_if_not_exists,
            payload.truncate,
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

    async fn rename_file(
        &self,
        request: Request<RenameFileRequest>,
    ) -> Result<Response<RenameFileResponse>, Status> {
        let payload = request.into_inner();
        Ok(Response::new(self.file_service.rename_file(
            &payload.source_path,
            &payload.target_path,
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

#[cfg(test)]
mod tests {
    use super::{FileOperatorService, FileServiceImpl};
    use crate::api::proto::file_service_server::FileService as FileGrpcService;
    use crate::api::proto::{
        CreateDirectoryRequest, DeleteFileRequest, ListFilesRequest, PathSafetyContext,
        ReadFileChunkRequest, ReadTextFileRequest, RenameFileRequest, WriteFileChunkRequest,
        WriteTextFileRequest,
    };
    use crate::security::path_validator::PathValidator;
    use std::fs;
    use std::path::{Path, PathBuf};
    use std::sync::Arc;
    use std::time::{SystemTime, UNIX_EPOCH};
    use tonic::Request;

    struct TempDir {
        path: PathBuf,
    }

    impl TempDir {
        fn new(prefix: &str) -> Self {
            let mut path = std::env::temp_dir();
            let nonce = SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .expect("system time must be after unix epoch")
                .as_nanos();
            path.push(format!("snowpanel_grpc_file_{prefix}_{nonce}"));
            fs::create_dir_all(&path).expect("failed to create temp dir");
            Self { path }
        }

        fn path(&self) -> &Path {
            &self.path
        }
    }

    impl Drop for TempDir {
        fn drop(&mut self) {
            let _ = fs::remove_dir_all(&self.path);
        }
    }

    fn file_service() -> FileServiceImpl {
        FileServiceImpl {
            file_service: Arc::new(FileOperatorService::new(
                PathValidator::new(Vec::new()),
                16,
                32,
            )),
        }
    }

    fn safety(root: &Path) -> Option<PathSafetyContext> {
        Some(PathSafetyContext {
            requested_path: String::new(),
            allowed_roots: vec![path_string(root)],
            enforce_safe_root: true,
            follow_symlink: false,
            normalized_path: String::new(),
        })
    }

    fn path_string(path: &Path) -> String {
        path.to_str()
            .expect("test paths should be valid utf-8")
            .to_string()
    }

    fn canonical_string(path: &Path) -> String {
        if path.exists() {
            return fs::canonicalize(path)
                .expect("path should canonicalize")
                .to_string_lossy()
                .into_owned();
        }

        let parent = path.parent().expect("path should have parent");
        let file_name = path.file_name().expect("path should have file name");
        fs::canonicalize(parent)
            .expect("path parent should canonicalize")
            .join(file_name)
            .to_string_lossy()
            .into_owned()
    }

    fn assert_ok(error: Option<crate::api::proto::Error>) {
        let error = error.expect("response should include error envelope");
        assert_eq!(error.code, 0, "response error should be ok: {:?}", error);
    }

    #[tokio::test]
    async fn file_grpc_service_forwards_proto_request_fields() {
        let temp = TempDir::new("forwarding");
        let service = file_service();

        let text_file = temp.path().join("note.txt");
        let write_text = service
            .write_text_file(Request::new(WriteTextFileRequest {
                path: path_string(&text_file),
                content: "hello grpc".to_string(),
                create_if_not_exists: true,
                truncate: true,
                encoding: "utf-8".to_string(),
                safety: safety(temp.path()),
            }))
            .await
            .expect("write_text_file should not return transport error")
            .into_inner();
        assert_ok(write_text.error);
        assert_eq!(write_text.path, canonical_string(&text_file));
        assert_eq!(write_text.written_bytes, "hello grpc".len() as u64);

        let read_text = service
            .read_text_file(Request::new(ReadTextFileRequest {
                path: path_string(&text_file),
                max_bytes: 5,
                encoding: "utf-8".to_string(),
                safety: safety(temp.path()),
            }))
            .await
            .expect("read_text_file should not return transport error")
            .into_inner();
        assert_ok(read_text.error);
        assert_eq!(read_text.path, canonical_string(&text_file));
        assert_eq!(read_text.content, "hello");
        assert!(read_text.truncated);

        let read_chunk = service
            .read_file_chunk(Request::new(ReadFileChunkRequest {
                path: path_string(&text_file),
                offset: 6,
                limit: 4,
                safety: safety(temp.path()),
            }))
            .await
            .expect("read_file_chunk should not return transport error")
            .into_inner();
        assert_ok(read_chunk.error);
        assert_eq!(read_chunk.offset, 6);
        assert_eq!(read_chunk.chunk, b"grpc".to_vec());
        assert!(read_chunk.eof);

        let chunk_file = temp.path().join("chunk.bin");
        let write_chunk = service
            .write_file_chunk(Request::new(WriteFileChunkRequest {
                path: path_string(&chunk_file),
                offset: 0,
                chunk: b"abc".to_vec(),
                create_if_not_exists: true,
                truncate: true,
                safety: safety(temp.path()),
            }))
            .await
            .expect("write_file_chunk should not return transport error")
            .into_inner();
        assert_ok(write_chunk.error);
        assert_eq!(write_chunk.path, canonical_string(&chunk_file));
        assert_eq!(write_chunk.offset, 0);
        assert_eq!(write_chunk.written_bytes, 3);
        assert_eq!(write_chunk.total_size, 3);

        let nested_parent = temp.path().join("nested");
        fs::create_dir_all(&nested_parent).expect("failed to create nested parent");
        let nested_dir = nested_parent.join("cache");
        let created = service
            .create_directory(Request::new(CreateDirectoryRequest {
                path: path_string(&nested_dir),
                create_parents: true,
                safety: safety(temp.path()),
            }))
            .await
            .expect("create_directory should not return transport error")
            .into_inner();
        assert_ok(created.error);
        assert_eq!(created.path, canonical_string(&nested_dir));

        let listed = service
            .list_files(Request::new(ListFilesRequest {
                path: path_string(temp.path()),
                safety: safety(temp.path()),
            }))
            .await
            .expect("list_files should not return transport error")
            .into_inner();
        assert_ok(listed.error);
        assert_eq!(listed.current_path, canonical_string(temp.path()));
        assert!(listed.entries.iter().any(|entry| entry.name == "note.txt"));
        assert!(listed.entries.iter().any(|entry| entry.name == "nested"));

        let renamed_file = temp.path().join("renamed.txt");
        let expected_text_path = canonical_string(&text_file);
        let expected_renamed_path = canonical_string(&renamed_file);
        let renamed = service
            .rename_file(Request::new(RenameFileRequest {
                source_path: path_string(&text_file),
                target_path: path_string(&renamed_file),
                safety: safety(temp.path()),
            }))
            .await
            .expect("rename_file should not return transport error")
            .into_inner();
        assert_ok(renamed.error);
        assert_eq!(renamed.source_path, expected_text_path);
        assert_eq!(renamed.target_path, expected_renamed_path);
        assert_eq!(renamed.moved_bytes, "hello grpc".len() as u64);

        let delete_dir = temp.path().join("delete-me");
        fs::create_dir_all(delete_dir.join("child")).expect("failed to create nested delete dir");
        let expected_delete_dir_path = canonical_string(&delete_dir);
        let deleted = service
            .delete_file(Request::new(DeleteFileRequest {
                path: path_string(&delete_dir),
                recursive: true,
                safety: safety(temp.path()),
            }))
            .await
            .expect("delete_file should not return transport error")
            .into_inner();
        assert_ok(deleted.error);
        assert_eq!(deleted.path, expected_delete_dir_path);
        assert!(!delete_dir.exists());
    }
}
