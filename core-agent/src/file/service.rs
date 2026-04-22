use std::fs::{self, OpenOptions};
use std::io::Write;
use std::path::Path;
use std::time::UNIX_EPOCH;

use crate::api::proto::{
    CreateDirectoryResponse, DeleteFileResponse, Error, FileEntry, ListFilesResponse,
    PathSafetyContext, ReadTextFileResponse, WriteTextFileResponse,
};
use crate::security::path_validator::{FileOperation, PathValidationError, PathValidator};

const DEFAULT_TEXT_EXTENSIONS: &[&str] = &[
    "txt", "log", "md", "json", "yaml", "yml", "toml", "ini", "conf", "xml", "csv", "go", "rs",
    "ts", "tsx", "js", "jsx", "css", "scss", "html", "sh", "py", "sql", "proto",
];

#[derive(Clone)]
pub struct FileService {
    path_validator: PathValidator,
    max_read_bytes: usize,
    max_write_bytes: usize,
}

impl FileService {
    pub fn new(
        path_validator: PathValidator,
        max_read_bytes: usize,
        max_write_bytes: usize,
    ) -> Self {
        Self {
            path_validator,
            max_read_bytes,
            max_write_bytes,
        }
    }

    pub fn list_files(&self, path: &str, safety: Option<PathSafetyContext>) -> ListFilesResponse {
        let safety = safety.unwrap_or_default();
        let normalized = match self.path_validator.validate(
            path,
            &safety.allowed_roots,
            true,
            FileOperation::List,
        ) {
            Ok(value) => value,
            Err(err) => return ListFilesResponse::from_error(file_error_from_validation(err)),
        };

        let metadata = match fs::metadata(&normalized) {
            Ok(value) => value,
            Err(err) => {
                return ListFilesResponse::from_error(FileError::io(format!(
                    "cannot read metadata for '{}': {err}",
                    normalized.display()
                )))
            }
        };

        if !metadata.is_dir() {
            return ListFilesResponse::from_error(FileError::bad_request(format!(
                "'{}' is not a directory",
                normalized.display()
            )));
        }

        let mut entries = Vec::new();
        let dir = match fs::read_dir(&normalized) {
            Ok(value) => value,
            Err(err) => {
                return ListFilesResponse::from_error(FileError::io(format!(
                    "cannot list directory '{}': {err}",
                    normalized.display()
                )))
            }
        };

        for item in dir {
            let item = match item {
                Ok(value) => value,
                Err(err) => {
                    return ListFilesResponse::from_error(FileError::io(format!(
                        "failed to iterate directory '{}': {err}",
                        normalized.display()
                    )))
                }
            };

            let item_meta = match item.metadata() {
                Ok(value) => value,
                Err(err) => {
                    return ListFilesResponse::from_error(FileError::io(format!(
                        "failed to read metadata '{}': {err}",
                        item.path().display()
                    )))
                }
            };

            let modified_at_unix = item_meta
                .modified()
                .ok()
                .and_then(|value| value.duration_since(UNIX_EPOCH).ok())
                .map(|value| value.as_secs() as i64)
                .unwrap_or(0);

            entries.push(FileEntry {
                name: item.file_name().to_string_lossy().into_owned(),
                path: item.path().to_string_lossy().into_owned(),
                is_dir: item_meta.is_dir(),
                size: item_meta.len(),
                modified_at_unix,
            });
        }

        entries.sort_by(|left, right| left.name.to_lowercase().cmp(&right.name.to_lowercase()));

        ListFilesResponse {
            error: Some(Error::ok()),
            current_path: normalized.to_string_lossy().into_owned(),
            entries,
        }
    }

    pub fn read_text_file(
        &self,
        path: &str,
        max_bytes: i64,
        encoding: &str,
        safety: Option<PathSafetyContext>,
    ) -> ReadTextFileResponse {
        let safety = safety.unwrap_or_default();
        let normalized = match self.path_validator.validate(
            path,
            &safety.allowed_roots,
            true,
            FileOperation::Read,
        ) {
            Ok(value) => value,
            Err(err) => return ReadTextFileResponse::from_error(file_error_from_validation(err)),
        };

        if !is_supported_encoding(encoding) {
            return ReadTextFileResponse::from_error(FileError::unsupported_encoding(encoding));
        }

        let metadata = match fs::metadata(&normalized) {
            Ok(value) => value,
            Err(err) => {
                return ReadTextFileResponse::from_error(FileError::io(format!(
                    "cannot read metadata for '{}': {err}",
                    normalized.display()
                )))
            }
        };

        if !metadata.is_file() {
            return ReadTextFileResponse::from_error(FileError::bad_request(format!(
                "'{}' is not a file",
                normalized.display()
            )));
        }

        let buffer = match fs::read(&normalized) {
            Ok(value) => value,
            Err(err) => {
                return ReadTextFileResponse::from_error(FileError::io(format!(
                    "cannot read file '{}': {err}",
                    normalized.display()
                )))
            }
        };

        let max_allowed = if max_bytes > 0 {
            std::cmp::min(max_bytes as usize, self.max_read_bytes)
        } else {
            self.max_read_bytes
        };

        let truncated = buffer.len() > max_allowed;
        let effective = if truncated {
            &buffer[..max_allowed]
        } else {
            &buffer[..]
        };

        let content = match String::from_utf8(effective.to_vec()) {
            Ok(value) => value,
            Err(_) => {
                return ReadTextFileResponse::from_error(FileError::not_text_file(format!(
                    "'{}' is not utf-8 text",
                    normalized.display()
                )))
            }
        };

        ReadTextFileResponse {
            error: Some(Error::ok()),
            path: normalized.to_string_lossy().into_owned(),
            content,
            size: metadata.len(),
            truncated,
            encoding: "utf-8".to_string(),
        }
    }

    pub fn write_text_file(
        &self,
        path: &str,
        content: &str,
        create_if_not_exists: bool,
        truncate: bool,
        encoding: &str,
        safety: Option<PathSafetyContext>,
    ) -> WriteTextFileResponse {
        let safety = safety.unwrap_or_default();
        let normalized = match self.path_validator.validate(
            path,
            &safety.allowed_roots,
            true,
            FileOperation::Write,
        ) {
            Ok(value) => value,
            Err(err) => return WriteTextFileResponse::from_error(file_error_from_validation(err)),
        };

        if !is_supported_encoding(encoding) {
            return WriteTextFileResponse::from_error(FileError::unsupported_encoding(encoding));
        }

        if !is_probably_text_file(&normalized) {
            return WriteTextFileResponse::from_error(FileError::not_text_file(format!(
                "'{}' is not an allowed text file",
                normalized.display()
            )));
        }

        if content.len() > self.max_write_bytes {
            return WriteTextFileResponse::from_error(FileError::file_too_large(format!(
                "payload size exceeds max_write_bytes={}",
                self.max_write_bytes
            )));
        }

        if !normalized.exists() && !create_if_not_exists {
            return WriteTextFileResponse::from_error(FileError::not_found(format!(
                "target '{}' does not exist",
                normalized.display()
            )));
        }

        if let Some(parent) = normalized.parent() {
            if !parent.exists() {
                return WriteTextFileResponse::from_error(FileError::bad_request(format!(
                    "parent path '{}' does not exist",
                    parent.display()
                )));
            }
        }

        let mut options = OpenOptions::new();
        options
            .write(true)
            .create(create_if_not_exists)
            .truncate(truncate);
        let mut file = match options.open(&normalized) {
            Ok(value) => value,
            Err(err) => {
                return WriteTextFileResponse::from_error(FileError::io(format!(
                    "cannot open '{}': {err}",
                    normalized.display()
                )))
            }
        };

        if let Err(err) = file.write_all(content.as_bytes()) {
            return WriteTextFileResponse::from_error(FileError::io(format!(
                "cannot write '{}': {err}",
                normalized.display()
            )));
        }

        WriteTextFileResponse {
            error: Some(Error::ok()),
            path: normalized.to_string_lossy().into_owned(),
            written_bytes: content.len() as u64,
        }
    }

    pub fn create_directory(
        &self,
        path: &str,
        create_parents: bool,
        safety: Option<PathSafetyContext>,
    ) -> CreateDirectoryResponse {
        let safety = safety.unwrap_or_default();
        let normalized = match self.path_validator.validate(
            path,
            &safety.allowed_roots,
            true,
            FileOperation::Mkdir,
        ) {
            Ok(value) => value,
            Err(err) => {
                return CreateDirectoryResponse::from_error(file_error_from_validation(err))
            }
        };

        let result = if create_parents {
            fs::create_dir_all(&normalized)
        } else {
            fs::create_dir(&normalized)
        };
        if let Err(err) = result {
            return CreateDirectoryResponse::from_error(FileError::io(format!(
                "cannot create directory '{}': {err}",
                normalized.display()
            )));
        }

        CreateDirectoryResponse {
            error: Some(Error::ok()),
            path: normalized.to_string_lossy().into_owned(),
        }
    }

    pub fn delete_path(
        &self,
        path: &str,
        recursive: bool,
        safety: Option<PathSafetyContext>,
    ) -> DeleteFileResponse {
        let safety = safety.unwrap_or_default();
        let normalized = match self.path_validator.validate(
            path,
            &safety.allowed_roots,
            true,
            FileOperation::Delete,
        ) {
            Ok(value) => value,
            Err(err) => return DeleteFileResponse::from_error(file_error_from_validation(err)),
        };

        if !normalized.exists() {
            return DeleteFileResponse::from_error(FileError::not_found(format!(
                "'{}' does not exist",
                normalized.display()
            )));
        }

        let metadata = match fs::metadata(&normalized) {
            Ok(value) => value,
            Err(err) => {
                return DeleteFileResponse::from_error(FileError::io(format!(
                    "cannot read metadata '{}': {err}",
                    normalized.display()
                )))
            }
        };

        if metadata.is_dir() {
            if recursive {
                if let Err(err) = fs::remove_dir_all(&normalized) {
                    return DeleteFileResponse::from_error(FileError::io(format!(
                        "cannot delete directory '{}': {err}",
                        normalized.display()
                    )));
                }
            } else if let Err(err) = fs::remove_dir(&normalized) {
                return DeleteFileResponse::from_error(FileError::io(format!(
                    "cannot delete directory '{}': {err}",
                    normalized.display()
                )));
            }
        } else if let Err(err) = fs::remove_file(&normalized) {
            return DeleteFileResponse::from_error(FileError::io(format!(
                "cannot delete file '{}': {err}",
                normalized.display()
            )));
        }

        DeleteFileResponse {
            error: Some(Error::ok()),
            path: normalized.to_string_lossy().into_owned(),
        }
    }
}

#[derive(Debug)]
struct FileError {
    code: i32,
    message: String,
    detail: String,
}

impl FileError {
    fn bad_request(detail: String) -> Self {
        Self {
            code: 4000,
            message: "bad request".to_string(),
            detail,
        }
    }

    fn path_unsafe(detail: String) -> Self {
        Self {
            code: 4001,
            message: "unsafe path".to_string(),
            detail,
        }
    }

    fn not_found(detail: String) -> Self {
        Self {
            code: 4002,
            message: "path not found".to_string(),
            detail,
        }
    }

    fn not_text_file(detail: String) -> Self {
        Self {
            code: 4003,
            message: "text file required".to_string(),
            detail,
        }
    }

    fn file_too_large(detail: String) -> Self {
        Self {
            code: 4004,
            message: "file too large".to_string(),
            detail,
        }
    }

    fn io(detail: String) -> Self {
        Self {
            code: 4005,
            message: "io error".to_string(),
            detail,
        }
    }

    fn unsupported_encoding(encoding: &str) -> Self {
        Self {
            code: 4006,
            message: "unsupported encoding".to_string(),
            detail: format!("encoding '{encoding}' is not supported"),
        }
    }

    fn dangerous_path(detail: String) -> Self {
        Self {
            code: 4007,
            message: "dangerous path".to_string(),
            detail,
        }
    }

    fn into_proto(self) -> Error {
        Error {
            code: self.code,
            message: self.message,
            detail: self.detail,
        }
    }
}

fn file_error_from_validation(err: PathValidationError) -> FileError {
    match err {
        PathValidationError::EmptyPath | PathValidationError::InvalidPath(_) => {
            FileError::bad_request(err.to_string())
        }
        PathValidationError::UnsafePath(_) => FileError::path_unsafe(err.to_string()),
        PathValidationError::DangerousPath(_) => FileError::dangerous_path(err.to_string()),
        PathValidationError::IOError(message) => {
            if message.contains("No such file or directory") {
                FileError::not_found(message)
            } else {
                FileError::io(message)
            }
        }
    }
}

fn is_supported_encoding(encoding: &str) -> bool {
    let normalized = encoding.trim().to_lowercase();
    normalized.is_empty() || normalized == "utf-8" || normalized == "utf8"
}

fn is_probably_text_file(path: &Path) -> bool {
    path.extension()
        .and_then(|value| value.to_str())
        .map(|ext| {
            DEFAULT_TEXT_EXTENSIONS
                .iter()
                .any(|item| item.eq_ignore_ascii_case(ext))
        })
        .unwrap_or(false)
}

trait ErrorProtoExt {
    fn ok() -> Self;
}

impl ErrorProtoExt for Error {
    fn ok() -> Self {
        Self {
            code: 0,
            message: "ok".to_string(),
            detail: String::new(),
        }
    }
}

trait ListFilesResponseExt {
    fn from_error(err: FileError) -> Self;
}

impl ListFilesResponseExt for ListFilesResponse {
    fn from_error(err: FileError) -> Self {
        Self {
            error: Some(err.into_proto()),
            current_path: String::new(),
            entries: Vec::new(),
        }
    }
}

trait ReadTextFileResponseExt {
    fn from_error(err: FileError) -> Self;
}

impl ReadTextFileResponseExt for ReadTextFileResponse {
    fn from_error(err: FileError) -> Self {
        Self {
            error: Some(err.into_proto()),
            path: String::new(),
            content: String::new(),
            size: 0,
            truncated: false,
            encoding: "utf-8".to_string(),
        }
    }
}

trait WriteTextFileResponseExt {
    fn from_error(err: FileError) -> Self;
}

impl WriteTextFileResponseExt for WriteTextFileResponse {
    fn from_error(err: FileError) -> Self {
        Self {
            error: Some(err.into_proto()),
            path: String::new(),
            written_bytes: 0,
        }
    }
}

trait CreateDirectoryResponseExt {
    fn from_error(err: FileError) -> Self;
}

impl CreateDirectoryResponseExt for CreateDirectoryResponse {
    fn from_error(err: FileError) -> Self {
        Self {
            error: Some(err.into_proto()),
            path: String::new(),
        }
    }
}

trait DeleteFileResponseExt {
    fn from_error(err: FileError) -> Self;
}

impl DeleteFileResponseExt for DeleteFileResponse {
    fn from_error(err: FileError) -> Self {
        Self {
            error: Some(err.into_proto()),
            path: String::new(),
        }
    }
}
