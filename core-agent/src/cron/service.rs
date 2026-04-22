use std::fmt::{Display, Formatter};
use std::io::Write;
use std::process::{Command, Stdio};
use std::time::{SystemTime, UNIX_EPOCH};

#[derive(Clone, Debug)]
pub struct CronTaskEntity {
    pub id: String,
    pub expression: String,
    pub command: String,
    pub enabled: bool,
}

#[derive(Debug)]
pub struct CronError {
    pub code: i32,
    pub message: String,
    pub detail: String,
}

impl Display for CronError {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}: {}", self.message, self.detail)
    }
}

impl std::error::Error for CronError {}

pub struct CronService;

impl CronService {
    pub fn new() -> Self {
        Self
    }

    pub fn list_tasks(&self) -> Result<Vec<CronTaskEntity>, CronError> {
        let content = read_current_crontab()?;
        let (_, tasks) = parse_crontab_document(&content);
        Ok(tasks)
    }

    pub fn create_task(
        &self,
        expression: &str,
        command: &str,
        enabled: bool,
    ) -> Result<CronTaskEntity, CronError> {
        validate_expression(expression)?;
        validate_command(command)?;

        let content = read_current_crontab()?;
        let (other_lines, mut tasks) = parse_crontab_document(&content);

        let task = CronTaskEntity {
            id: generate_task_id(),
            expression: expression.trim().to_string(),
            command: command.trim().to_string(),
            enabled,
        };
        tasks.push(task.clone());

        let next = render_crontab_document(&other_lines, &tasks);
        apply_crontab(&next)?;
        Ok(task)
    }

    pub fn update_task(
        &self,
        id: &str,
        expression: &str,
        command: &str,
        enabled: bool,
    ) -> Result<CronTaskEntity, CronError> {
        validate_task_id(id)?;
        validate_expression(expression)?;
        validate_command(command)?;

        let content = read_current_crontab()?;
        let (other_lines, mut tasks) = parse_crontab_document(&content);
        let item = tasks
            .iter_mut()
            .find(|task| task.id == id)
            .ok_or_else(|| CronError::not_found(format!("cron task '{}' not found", id)))?;

        item.expression = expression.trim().to_string();
        item.command = command.trim().to_string();
        item.enabled = enabled;
        let updated = item.clone();

        let next = render_crontab_document(&other_lines, &tasks);
        apply_crontab(&next)?;
        Ok(updated)
    }

    pub fn delete_task(&self, id: &str) -> Result<(), CronError> {
        validate_task_id(id)?;

        let content = read_current_crontab()?;
        let (other_lines, mut tasks) = parse_crontab_document(&content);
        let original_len = tasks.len();
        tasks.retain(|task| task.id != id);
        if tasks.len() == original_len {
            return Err(CronError::not_found(format!(
                "cron task '{}' not found",
                id
            )));
        }

        let next = render_crontab_document(&other_lines, &tasks);
        apply_crontab(&next)?;
        Ok(())
    }

    pub fn set_enabled(&self, id: &str, enabled: bool) -> Result<CronTaskEntity, CronError> {
        validate_task_id(id)?;

        let content = read_current_crontab()?;
        let (other_lines, mut tasks) = parse_crontab_document(&content);
        let item = tasks
            .iter_mut()
            .find(|task| task.id == id)
            .ok_or_else(|| CronError::not_found(format!("cron task '{}' not found", id)))?;
        item.enabled = enabled;
        let updated = item.clone();

        let next = render_crontab_document(&other_lines, &tasks);
        apply_crontab(&next)?;
        Ok(updated)
    }
}

impl CronError {
    fn bad_request(detail: String) -> Self {
        Self {
            code: 7000,
            message: "bad request".to_string(),
            detail,
        }
    }

    fn not_found(detail: String) -> Self {
        Self {
            code: 7001,
            message: "cron task not found".to_string(),
            detail,
        }
    }

    fn command_failed(detail: String) -> Self {
        Self {
            code: 7002,
            message: "crontab command failed".to_string(),
            detail,
        }
    }
}

fn validate_expression(raw: &str) -> Result<(), CronError> {
    let expression = raw.trim();
    if expression.is_empty() {
        return Err(CronError::bad_request(
            "cron expression is empty".to_string(),
        ));
    }

    let parts = expression.split_whitespace().collect::<Vec<_>>();
    if parts.len() != 5 {
        return Err(CronError::bad_request(
            "cron expression must contain 5 fields".to_string(),
        ));
    }

    for field in parts {
        let valid = field.chars().all(|ch| {
            ch.is_ascii_alphanumeric() || ch == '*' || ch == '/' || ch == ',' || ch == '-'
        });
        if !valid {
            return Err(CronError::bad_request(format!(
                "cron field '{}' contains unsupported characters",
                field
            )));
        }
    }

    Ok(())
}

fn validate_command(raw: &str) -> Result<(), CronError> {
    let command = raw.trim();
    if command.is_empty() {
        return Err(CronError::bad_request("command is empty".to_string()));
    }
    if command.contains('\n') || command.contains('\r') {
        return Err(CronError::bad_request(
            "command cannot contain newlines".to_string(),
        ));
    }
    if command.len() > 1024 {
        return Err(CronError::bad_request(
            "command length exceeds 1024".to_string(),
        ));
    }
    Ok(())
}

fn validate_task_id(raw: &str) -> Result<(), CronError> {
    let id = raw.trim();
    if id.is_empty() {
        return Err(CronError::bad_request("task id is empty".to_string()));
    }
    let valid = id
        .chars()
        .all(|ch| ch.is_ascii_alphanumeric() || ch == '-' || ch == '_');
    if !valid {
        return Err(CronError::bad_request(format!(
            "task id '{}' contains invalid characters",
            id
        )));
    }
    Ok(())
}

fn generate_task_id() -> String {
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|value| value.as_millis())
        .unwrap_or(0);
    format!("task-{now}")
}

fn read_current_crontab() -> Result<String, CronError> {
    let output = Command::new("crontab")
        .arg("-l")
        .output()
        .map_err(|err| CronError::command_failed(format!("run 'crontab -l' failed: {err}")))?;

    if output.status.success() {
        return Ok(String::from_utf8_lossy(&output.stdout).to_string());
    }

    let stderr = String::from_utf8_lossy(&output.stderr).to_lowercase();
    if stderr.contains("no crontab for") {
        return Ok(String::new());
    }

    Err(CronError::command_failed(format!(
        "'crontab -l' failed: {}",
        String::from_utf8_lossy(&output.stderr)
    )))
}

fn apply_crontab(content: &str) -> Result<(), CronError> {
    let mut child = Command::new("crontab")
        .arg("-")
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(|err| CronError::command_failed(format!("spawn 'crontab -' failed: {err}")))?;

    if let Some(stdin) = child.stdin.as_mut() {
        stdin.write_all(content.as_bytes()).map_err(|err| {
            CronError::command_failed(format!("write cron content failed: {err}"))
        })?;
    }

    let output = child
        .wait_with_output()
        .map_err(|err| CronError::command_failed(format!("wait 'crontab -' failed: {err}")))?;

    if !output.status.success() {
        return Err(CronError::command_failed(format!(
            "'crontab -' failed: {}",
            String::from_utf8_lossy(&output.stderr)
        )));
    }

    Ok(())
}

fn parse_crontab_document(content: &str) -> (Vec<String>, Vec<CronTaskEntity>) {
    let mut others = Vec::new();
    let mut tasks = Vec::new();

    let mut lines = content.lines().peekable();
    while let Some(line) = lines.next() {
        if let Some((id, enabled_from_header)) = parse_metadata_header(line) {
            if let Some(entry_line) = lines.next() {
                if let Some(task) = parse_task_line(entry_line, &id, enabled_from_header) {
                    tasks.push(task);
                    continue;
                }

                others.push(line.to_string());
                others.push(entry_line.to_string());
                continue;
            }

            others.push(line.to_string());
            continue;
        }
        others.push(line.to_string());
    }

    (others, tasks)
}

fn parse_metadata_header(line: &str) -> Option<(String, bool)> {
    let prefix = "# snowpanel:id=";
    if !line.starts_with(prefix) {
        return None;
    }
    let raw = line.trim_start_matches(prefix);
    let (id_part, enabled_part) = raw.split_once(";enabled=")?;
    let enabled = matches!(enabled_part.trim(), "1" | "true");
    Some((id_part.trim().to_string(), enabled))
}

fn parse_task_line(line: &str, id: &str, enabled_from_header: bool) -> Option<CronTaskEntity> {
    let mut enabled = enabled_from_header;
    let mut body = line.trim().to_string();
    if let Some(rest) = body.strip_prefix('#') {
        body = rest.trim_start().to_string();
        enabled = false;
    }

    let parts = body.split_whitespace().collect::<Vec<_>>();
    if parts.len() < 6 {
        return None;
    }

    let expression = parts[..5].join(" ");
    let command = parts[5..].join(" ");
    Some(CronTaskEntity {
        id: id.to_string(),
        expression,
        command,
        enabled,
    })
}

fn render_crontab_document(other_lines: &[String], tasks: &[CronTaskEntity]) -> String {
    let mut lines = Vec::new();
    lines.extend(
        other_lines
            .iter()
            .filter(|item| !item.trim().is_empty())
            .cloned(),
    );

    if !lines.is_empty() {
        lines.push(String::new());
    }

    for task in tasks {
        lines.push(format!(
            "# snowpanel:id={};enabled={}",
            task.id,
            if task.enabled { 1 } else { 0 }
        ));
        let entry = format!("{} {}", task.expression, task.command);
        if task.enabled {
            lines.push(entry);
        } else {
            lines.push(format!("# {}", entry));
        }
    }

    if lines.is_empty() {
        String::new()
    } else {
        format!("{}\n", lines.join("\n"))
    }
}
