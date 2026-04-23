use std::fmt::{Display, Formatter};
use std::process::Command;

#[derive(Clone, Debug)]
pub struct ManagedService {
    pub name: String,
    pub display_name: String,
    pub status: String,
}

#[derive(Clone)]
pub struct SystemdServiceManager {
    whitelist: Vec<String>,
}

#[derive(Debug)]
pub struct ServiceError {
    pub code: i32,
    pub message: String,
    pub detail: String,
}

impl Display for ServiceError {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}: {}", self.message, self.detail)
    }
}

impl std::error::Error for ServiceError {}

#[derive(Clone, Copy)]
pub enum ServiceAction {
    Start,
    Stop,
    Restart,
}

impl SystemdServiceManager {
    pub fn new(whitelist: Vec<String>) -> Self {
        Self { whitelist }
    }

    pub fn list_services(&self, keyword: &str) -> Result<Vec<ManagedService>, ServiceError> {
        let output = Command::new("systemctl")
            .args([
                "list-units",
                "--type=service",
                "--all",
                "--no-legend",
                "--no-pager",
            ])
            .output()
            .map_err(|err| {
                ServiceError::io(format!("execute systemctl list-units failed: {err}"))
            })?;

        if !output.status.success() {
            return Err(ServiceError::command_failed(format!(
                "systemctl list-units failed: {}",
                String::from_utf8_lossy(&output.stderr)
            )));
        }

        let keyword = keyword.trim().to_lowercase();
        let mut services = Vec::new();
        for line in String::from_utf8_lossy(&output.stdout).lines() {
            let columns = line.split_whitespace().collect::<Vec<_>>();
            if columns.len() < 5 {
                continue;
            }

            let name = columns[0].to_string();
            let status = columns[3].to_string();
            let display_name = columns[4..].join(" ");
            let service = ManagedService {
                name: name.clone(),
                display_name,
                status,
            };

            if !self.whitelist.is_empty() && !self.is_allowed(&name) {
                continue;
            }

            if keyword.is_empty()
                || service.name.to_lowercase().contains(&keyword)
                || service.display_name.to_lowercase().contains(&keyword)
            {
                services.push(service);
            }
        }

        Ok(services)
    }

    pub fn run_action(
        &self,
        action: ServiceAction,
        raw_name: &str,
    ) -> Result<ManagedService, ServiceError> {
        let name = normalize_service_name(raw_name)?;
        self.ensure_allowed(&name)?;

        let action_str = match action {
            ServiceAction::Start => "start",
            ServiceAction::Stop => "stop",
            ServiceAction::Restart => "restart",
        };

        let output = Command::new("systemctl")
            .args([action_str, &name, "--no-pager"])
            .output()
            .map_err(|err| {
                ServiceError::io(format!(
                    "execute systemctl {} '{}' failed: {err}",
                    action_str, name
                ))
            })?;

        if !output.status.success() {
            return Err(ServiceError::command_failed(format!(
                "systemctl {} '{}' failed: {}",
                action_str,
                name,
                String::from_utf8_lossy(&output.stderr)
            )));
        }

        let status = self.query_status(&name)?;
        Ok(ManagedService {
            name: name.clone(),
            display_name: name,
            status,
        })
    }

    pub fn query_status(&self, name: &str) -> Result<String, ServiceError> {
        let output = Command::new("systemctl")
            .args(["is-active", name, "--no-pager"])
            .output()
            .map_err(|err| {
                ServiceError::io(format!(
                    "execute systemctl is-active '{}' failed: {err}",
                    name
                ))
            })?;

        if !output.status.success() {
            let stderr = String::from_utf8_lossy(&output.stderr).trim().to_string();
            if stderr.is_empty() {
                return Ok("inactive".to_string());
            }
            return Err(ServiceError::command_failed(format!(
                "systemctl is-active '{}' failed: {}",
                name, stderr
            )));
        }

        Ok(String::from_utf8_lossy(&output.stdout).trim().to_string())
    }

    #[allow(dead_code)]
    pub fn tail_logs_placeholder(&self, _name: &str) -> Result<String, ServiceError> {
        Err(ServiceError {
            code: 5005,
            message: "not implemented".to_string(),
            detail: "service log query interface is reserved for a later stage".to_string(),
        })
    }

    fn ensure_allowed(&self, name: &str) -> Result<(), ServiceError> {
        if self.whitelist.is_empty() {
            return Ok(());
        }
        if self.is_allowed(name) {
            return Ok(());
        }
        Err(ServiceError {
            code: 5002,
            message: "service not allowed".to_string(),
            detail: format!("service '{}' is outside whitelist", name),
        })
    }

    fn is_allowed(&self, name: &str) -> bool {
        self.whitelist.iter().any(|item| item == name)
    }
}

impl ServiceError {
    fn bad_request(detail: String) -> Self {
        Self {
            code: 5000,
            message: "bad request".to_string(),
            detail,
        }
    }

    fn command_failed(detail: String) -> Self {
        Self {
            code: 5001,
            message: "systemctl command failed".to_string(),
            detail,
        }
    }

    fn io(detail: String) -> Self {
        Self {
            code: 5003,
            message: "io error".to_string(),
            detail,
        }
    }
}

fn normalize_service_name(raw_name: &str) -> Result<String, ServiceError> {
    let trimmed = raw_name.trim();
    if trimmed.is_empty() {
        return Err(ServiceError::bad_request(
            "service name is empty".to_string(),
        ));
    }
    if trimmed.len() > 128 {
        return Err(ServiceError::bad_request(
            "service name length exceeds 128".to_string(),
        ));
    }

    let valid = trimmed
        .chars()
        .all(|ch| ch.is_ascii_alphanumeric() || ch == '-' || ch == '_' || ch == '.' || ch == '@');
    if !valid {
        return Err(ServiceError::bad_request(format!(
            "service name '{}' contains invalid characters",
            trimmed
        )));
    }

    if trimmed.ends_with(".service") {
        return Ok(trimmed.to_string());
    }
    Ok(format!("{trimmed}.service"))
}
