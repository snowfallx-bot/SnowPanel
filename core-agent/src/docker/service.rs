use std::fmt::{Display, Formatter};

use bollard::container::{ListContainersOptions, RestartContainerOptions};
use bollard::image::ListImagesOptions;
use bollard::Docker;

#[derive(Clone, Debug)]
pub struct ContainerInfo {
    pub id: String,
    pub name: String,
    pub image: String,
    pub state: String,
    pub status: String,
}

#[derive(Clone, Debug)]
pub struct ImageInfo {
    pub id: String,
    pub repo_tags: Vec<String>,
    pub size: u64,
}

#[derive(Debug)]
pub struct DockerError {
    pub code: i32,
    pub message: String,
    pub detail: String,
}

impl Display for DockerError {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}: {}", self.message, self.detail)
    }
}

impl std::error::Error for DockerError {}

#[derive(Clone, Copy)]
pub enum DockerAction {
    Start,
    Stop,
    Restart,
}

#[derive(Clone)]
pub struct DockerService {
    docker: Option<Docker>,
}

impl DockerService {
    pub fn new() -> Result<Self, DockerError> {
        match Docker::connect_with_local_defaults() {
            Ok(docker) => Ok(Self {
                docker: Some(docker),
            }),
            Err(_) => Ok(Self { docker: None }),
        }
    }

    pub async fn list_containers(&self) -> Result<Vec<ContainerInfo>, DockerError> {
        let options = ListContainersOptions::<String> {
            all: true,
            ..Default::default()
        };
        let items = self
            .docker_client()?
            .list_containers(Some(options))
            .await
            .map_err(|err| DockerError::command_failed(format!("list containers failed: {err}")))?;

        Ok(items
            .into_iter()
            .map(|item| ContainerInfo {
                id: item.id.unwrap_or_default(),
                name: item
                    .names
                    .and_then(|names| names.into_iter().next())
                    .unwrap_or_default()
                    .trim_start_matches('/')
                    .to_string(),
                image: item.image.unwrap_or_default(),
                state: item.state.unwrap_or_default(),
                status: item.status.unwrap_or_default(),
            })
            .collect::<Vec<_>>())
    }

    pub async fn list_images(&self) -> Result<Vec<ImageInfo>, DockerError> {
        let options = ListImagesOptions::<String> {
            all: true,
            ..Default::default()
        };

        let items = self
            .docker_client()?
            .list_images(Some(options))
            .await
            .map_err(|err| DockerError::command_failed(format!("list images failed: {err}")))?;

        Ok(items
            .into_iter()
            .map(|item| {
                let size = if item.size < 0 { 0 } else { item.size as u64 };
                ImageInfo {
                    id: item.id,
                    repo_tags: item.repo_tags,
                    size,
                }
            })
            .collect::<Vec<_>>())
    }

    pub async fn run_action(
        &self,
        action: DockerAction,
        container_id: &str,
    ) -> Result<ContainerInfo, DockerError> {
        let id = normalize_container_id(container_id)?;
        let docker = self.docker_client()?;

        match action {
            DockerAction::Start => {
                docker
                    .start_container::<String>(&id, None)
                    .await
                    .map_err(|err| {
                        DockerError::command_failed(format!("start container '{id}' failed: {err}"))
                    })?
            }
            DockerAction::Stop => docker.stop_container(&id, None).await.map_err(|err| {
                DockerError::command_failed(format!("stop container '{id}' failed: {err}"))
            })?,
            DockerAction::Restart => docker
                .restart_container(&id, None::<RestartContainerOptions>)
                .await
                .map_err(|err| {
                    DockerError::command_failed(format!("restart container '{id}' failed: {err}"))
                })?,
        };

        self.inspect_container(&id).await
    }

    async fn inspect_container(&self, id: &str) -> Result<ContainerInfo, DockerError> {
        let details = self
            .docker_client()?
            .inspect_container(id, None)
            .await
            .map_err(|err| {
                DockerError::not_found(format!("inspect container '{id}' failed: {err}"))
            })?;

        Ok(ContainerInfo {
            id: details.id.unwrap_or_else(|| id.to_string()),
            name: details
                .name
                .unwrap_or_default()
                .trim_start_matches('/')
                .to_string(),
            image: details.config.and_then(|cfg| cfg.image).unwrap_or_default(),
            state: details
                .state
                .as_ref()
                .and_then(|state| state.status.clone().map(|status| status.to_string()))
                .unwrap_or_default(),
            status: details
                .state
                .as_ref()
                .and_then(|state| {
                    state
                        .health
                        .as_ref()
                        .and_then(|health| health.status.clone().map(|status| status.to_string()))
                })
                .unwrap_or_default(),
        })
    }

    fn docker_client(&self) -> Result<&Docker, DockerError> {
        self.docker.as_ref().ok_or_else(DockerError::unavailable)
    }
}

impl DockerError {
    fn unavailable() -> Self {
        Self {
            code: 6000,
            message: "docker unavailable".to_string(),
            detail: "Docker socket not found: /var/run/docker.sock".to_string(),
        }
    }

    fn bad_request(detail: String) -> Self {
        Self {
            code: 6001,
            message: "bad request".to_string(),
            detail,
        }
    }

    fn not_found(detail: String) -> Self {
        Self {
            code: 6002,
            message: "container not found".to_string(),
            detail,
        }
    }

    fn command_failed(detail: String) -> Self {
        Self {
            code: 6003,
            message: "docker command failed".to_string(),
            detail,
        }
    }
}

fn normalize_container_id(raw: &str) -> Result<String, DockerError> {
    let trimmed = raw.trim();
    if trimmed.is_empty() {
        return Err(DockerError::bad_request(
            "container id is empty".to_string(),
        ));
    }
    if trimmed.len() > 128 {
        return Err(DockerError::bad_request(
            "container id length exceeds 128".to_string(),
        ));
    }

    let valid = trimmed
        .chars()
        .all(|ch| ch.is_ascii_alphanumeric() || ch == '-' || ch == '_' || ch == '.' || ch == '/');
    if !valid {
        return Err(DockerError::bad_request(format!(
            "container id '{}' contains invalid characters",
            trimmed
        )));
    }

    Ok(trimmed.to_string())
}
