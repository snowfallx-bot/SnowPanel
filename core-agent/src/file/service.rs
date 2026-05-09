use std::fs::{self, File, OpenOptions};
use std::io::{Read, Seek, SeekFrom, Write};
use std::path::Path;
use std::time::UNIX_EPOCH;

use crate::api::proto::{
    CreateDirectoryResponse, DeleteFileResponse, Error, FileEntry, ListFilesResponse,
    PathSafetyContext, ReadFileChunkResponse, ReadTextFileResponse, RenameFileResponse,
    WriteFileChunkResponse, WriteTextFileResponse,
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

    pub fn read_file_chunk(
        &self,
        path: &str,
        offset: u64,
        limit: u32,
        safety: Option<PathSafetyContext>,
    ) -> ReadFileChunkResponse {
        let safety = safety.unwrap_or_default();
        let normalized = match self.path_validator.validate(
            path,
            &safety.allowed_roots,
            true,
            FileOperation::Read,
        ) {
            Ok(value) => value,
            Err(err) => return ReadFileChunkResponse::from_error(file_error_from_validation(err)),
        };

        let metadata = match fs::metadata(&normalized) {
            Ok(value) => value,
            Err(err) => {
                return ReadFileChunkResponse::from_error(FileError::io(format!(
                    "cannot read metadata for '{}': {err}",
                    normalized.display()
                )))
            }
        };

        if !metadata.is_file() {
            return ReadFileChunkResponse::from_error(FileError::bad_request(format!(
                "'{}' is not a file",
                normalized.display()
            )));
        }

        let total_size = metadata.len();
        if offset > total_size {
            return ReadFileChunkResponse::from_error(FileError::bad_request(format!(
                "offset {offset} exceeds file size {total_size}",
            )));
        }

        if offset == total_size {
            return ReadFileChunkResponse {
                error: Some(Error::ok()),
                path: normalized.to_string_lossy().into_owned(),
                offset,
                chunk: Vec::new(),
                total_size,
                eof: true,
            };
        }

        let max_read_bytes = self.max_read_bytes.max(1);
        let max_allowed = if limit > 0 {
            std::cmp::min(limit as usize, max_read_bytes)
        } else {
            max_read_bytes
        };

        let mut file = match File::open(&normalized) {
            Ok(value) => value,
            Err(err) => {
                return ReadFileChunkResponse::from_error(FileError::io(format!(
                    "cannot open '{}': {err}",
                    normalized.display()
                )))
            }
        };

        if let Err(err) = file.seek(SeekFrom::Start(offset)) {
            return ReadFileChunkResponse::from_error(FileError::io(format!(
                "cannot seek '{}' to offset {offset}: {err}",
                normalized.display()
            )));
        }

        let mut buffer = vec![0u8; max_allowed];
        let read_len = match file.read(&mut buffer) {
            Ok(value) => value,
            Err(err) => {
                return ReadFileChunkResponse::from_error(FileError::io(format!(
                    "cannot read file '{}': {err}",
                    normalized.display()
                )))
            }
        };
        buffer.truncate(read_len);

        let eof = offset.saturating_add(read_len as u64) >= total_size;

        ReadFileChunkResponse {
            error: Some(Error::ok()),
            path: normalized.to_string_lossy().into_owned(),
            offset,
            chunk: buffer,
            total_size,
            eof,
        }
    }

    pub fn write_file_chunk(
        &self,
        path: &str,
        offset: u64,
        chunk: &[u8],
        create_if_not_exists: bool,
        truncate: bool,
        safety: Option<PathSafetyContext>,
    ) -> WriteFileChunkResponse {
        let safety = safety.unwrap_or_default();
        let normalized = match self.path_validator.validate(
            path,
            &safety.allowed_roots,
            true,
            FileOperation::Write,
        ) {
            Ok(value) => value,
            Err(err) => return WriteFileChunkResponse::from_error(file_error_from_validation(err)),
        };

        if chunk.len() > self.max_write_bytes {
            return WriteFileChunkResponse::from_error(FileError::file_too_large(format!(
                "chunk size exceeds max_write_bytes={}",
                self.max_write_bytes
            )));
        }

        if truncate && offset > 0 {
            return WriteFileChunkResponse::from_error(FileError::bad_request(
                "truncate upload chunk requires offset=0".to_string(),
            ));
        }

        if let Some(parent) = normalized.parent() {
            if !parent.exists() {
                return WriteFileChunkResponse::from_error(FileError::bad_request(format!(
                    "parent path '{}' does not exist",
                    parent.display()
                )));
            }
        }

        if !normalized.exists() && !create_if_not_exists {
            return WriteFileChunkResponse::from_error(FileError::not_found(format!(
                "target '{}' does not exist",
                normalized.display()
            )));
        }

        let mut options = OpenOptions::new();
        options
            .write(true)
            .create(create_if_not_exists)
            .truncate(truncate);

        let mut file = match options.open(&normalized) {
            Ok(value) => value,
            Err(err) => {
                return WriteFileChunkResponse::from_error(FileError::io(format!(
                    "cannot open '{}': {err}",
                    normalized.display()
                )))
            }
        };

        let current_size = match file.metadata() {
            Ok(value) => value.len(),
            Err(err) => {
                return WriteFileChunkResponse::from_error(FileError::io(format!(
                    "cannot read metadata for '{}': {err}",
                    normalized.display()
                )))
            }
        };

        if offset > current_size {
            return WriteFileChunkResponse::from_error(FileError::bad_request(format!(
                "offset {offset} exceeds file size {current_size}",
            )));
        }

        if let Err(err) = file.seek(SeekFrom::Start(offset)) {
            return WriteFileChunkResponse::from_error(FileError::io(format!(
                "cannot seek '{}' to offset {offset}: {err}",
                normalized.display()
            )));
        }

        if !chunk.is_empty() {
            if let Err(err) = file.write_all(chunk) {
                return WriteFileChunkResponse::from_error(FileError::io(format!(
                    "cannot write '{}': {err}",
                    normalized.display()
                )));
            }
        }

        let total_size = match fs::metadata(&normalized) {
            Ok(value) => value.len(),
            Err(err) => {
                return WriteFileChunkResponse::from_error(FileError::io(format!(
                    "cannot read metadata for '{}': {err}",
                    normalized.display()
                )))
            }
        };

        WriteFileChunkResponse {
            error: Some(Error::ok()),
            path: normalized.to_string_lossy().into_owned(),
            offset,
            written_bytes: chunk.len() as u64,
            total_size,
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

    pub fn rename_file(
        &self,
        source_path: &str,
        target_path: &str,
        safety: Option<PathSafetyContext>,
    ) -> RenameFileResponse {
        let safety = safety.unwrap_or_default();
        let source = match self.path_validator.validate(
            source_path,
            &safety.allowed_roots,
            true,
            FileOperation::Move,
        ) {
            Ok(value) => value,
            Err(err) => return RenameFileResponse::from_error(file_error_from_validation(err)),
        };

        let target = match self.path_validator.validate(
            target_path,
            &safety.allowed_roots,
            true,
            FileOperation::Move,
        ) {
            Ok(value) => value,
            Err(err) => return RenameFileResponse::from_error(file_error_from_validation(err)),
        };

        if source == target {
            return RenameFileResponse::from_error(FileError::bad_request(
                "source_path and target_path cannot be the same".to_string(),
            ));
        }

        let source_metadata = match fs::metadata(&source) {
            Ok(value) => value,
            Err(err) => {
                return RenameFileResponse::from_error(FileError::io(format!(
                    "cannot read metadata for '{}': {err}",
                    source.display()
                )))
            }
        };
        if !source_metadata.is_file() {
            return RenameFileResponse::from_error(FileError::bad_request(format!(
                "'{}' is not a file",
                source.display()
            )));
        }

        if target.exists() {
            return RenameFileResponse::from_error(FileError::bad_request(format!(
                "target '{}' already exists",
                target.display()
            )));
        }

        if let Some(parent) = target.parent() {
            if !parent.exists() {
                return RenameFileResponse::from_error(FileError::bad_request(format!(
                    "parent path '{}' does not exist",
                    parent.display()
                )));
            }
        }

        if let Err(err) = fs::rename(&source, &target) {
            return RenameFileResponse::from_error(FileError::io(format!(
                "cannot rename '{}' -> '{}': {err}",
                source.display(),
                target.display()
            )));
        }

        let moved_bytes = match fs::metadata(&target) {
            Ok(value) => value.len(),
            Err(err) => {
                return RenameFileResponse::from_error(FileError::io(format!(
                    "cannot read metadata for '{}': {err}",
                    target.display()
                )))
            }
        };

        RenameFileResponse {
            error: Some(Error::ok()),
            source_path: source.to_string_lossy().into_owned(),
            target_path: target.to_string_lossy().into_owned(),
            moved_bytes,
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

trait ReadFileChunkResponseExt {
    fn from_error(err: FileError) -> Self;
}

impl ReadFileChunkResponseExt for ReadFileChunkResponse {
    fn from_error(err: FileError) -> Self {
        Self {
            error: Some(err.into_proto()),
            path: String::new(),
            offset: 0,
            chunk: Vec::new(),
            total_size: 0,
            eof: false,
        }
    }
}

trait WriteTextFileResponseExt {
    fn from_error(err: FileError) -> Self;
}

trait WriteFileChunkResponseExt {
    fn from_error(err: FileError) -> Self;
}

impl WriteFileChunkResponseExt for WriteFileChunkResponse {
    fn from_error(err: FileError) -> Self {
        Self {
            error: Some(err.into_proto()),
            path: String::new(),
            offset: 0,
            written_bytes: 0,
            total_size: 0,
        }
    }
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

trait RenameFileResponseExt {
    fn from_error(err: FileError) -> Self;
}

impl RenameFileResponseExt for RenameFileResponse {
    fn from_error(err: FileError) -> Self {
        Self {
            error: Some(err.into_proto()),
            source_path: String::new(),
            target_path: String::new(),
            moved_bytes: 0,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::FileService;
    use crate::security::path_validator::PathValidator;
    use std::fs;
    use std::path::{Path, PathBuf};
    use std::time::{SystemTime, UNIX_EPOCH};

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
            path.push(format!("snowpanel_file_service_{prefix}_{nonce}"));
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

    fn test_service(root: &Path) -> FileService {
        FileService::new(PathValidator::new(vec![root.to_path_buf()]), 16, 32)
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
        assert_eq!(error.message, "ok");
    }

    #[test]
    fn file_operations_preserve_proto_response_fields() {
        let temp = TempDir::new("contract_fields");
        let service = test_service(temp.path());

        let source_file = temp.path().join("alpha.txt");
        fs::write(&source_file, "hello snowpanel").expect("failed to write source file");

        let read_text = service.read_text_file(&path_string(&source_file), 5, "utf-8", None);
        assert_ok(read_text.error);
        assert_eq!(read_text.path, canonical_string(&source_file));
        assert_eq!(read_text.content, "hello");
        assert_eq!(read_text.size, "hello snowpanel".len() as u64);
        assert!(read_text.truncated);
        assert_eq!(read_text.encoding, "utf-8");

        let read_chunk = service.read_file_chunk(&path_string(&source_file), 6, 9, None);
        assert_ok(read_chunk.error);
        assert_eq!(read_chunk.path, canonical_string(&source_file));
        assert_eq!(read_chunk.offset, 6);
        assert_eq!(read_chunk.chunk, b"snowpanel".to_vec());
        assert_eq!(read_chunk.total_size, "hello snowpanel".len() as u64);
        assert!(read_chunk.eof);

        let chunk_file = temp.path().join("chunks.bin");
        let first_write =
            service.write_file_chunk(&path_string(&chunk_file), 0, b"abc", true, true, None);
        assert_ok(first_write.error);
        assert_eq!(first_write.path, canonical_string(&chunk_file));
        assert_eq!(first_write.offset, 0);
        assert_eq!(first_write.written_bytes, 3);
        assert_eq!(first_write.total_size, 3);

        let append_write =
            service.write_file_chunk(&path_string(&chunk_file), 3, b"def", false, false, None);
        assert_ok(append_write.error);
        assert_eq!(append_write.path, canonical_string(&chunk_file));
        assert_eq!(append_write.offset, 3);
        assert_eq!(append_write.written_bytes, 3);
        assert_eq!(append_write.total_size, 6);
        assert_eq!(
            fs::read(&chunk_file).expect("failed to read chunk file"),
            b"abcdef".to_vec()
        );

        let note_file = temp.path().join("note.txt");
        let write_text = service.write_text_file(
            &path_string(&note_file),
            "contract text",
            true,
            true,
            "utf-8",
            None,
        );
        assert_ok(write_text.error);
        assert_eq!(write_text.path, canonical_string(&note_file));
        assert_eq!(write_text.written_bytes, "contract text".len() as u64);

        let cache_dir = temp.path().join("cache");
        let created = service.create_directory(&path_string(&cache_dir), true, None);
        assert_ok(created.error);
        assert_eq!(created.path, canonical_string(&cache_dir));

        let listed = service.list_files(&path_string(temp.path()), None);
        assert_ok(listed.error);
        assert_eq!(listed.current_path, canonical_string(temp.path()));
        let entry_names = listed
            .entries
            .iter()
            .map(|entry| entry.name.as_str())
            .collect::<Vec<_>>();
        assert_eq!(
            entry_names,
            vec!["alpha.txt", "cache", "chunks.bin", "note.txt"]
        );
        let cache_entry = listed
            .entries
            .iter()
            .find(|entry| entry.name == "cache")
            .expect("cache entry should be present");
        assert!(cache_entry.is_dir);

        let renamed_file = temp.path().join("renamed.txt");
        let expected_note_path = canonical_string(&note_file);
        let expected_renamed_path = canonical_string(&renamed_file);
        let renamed =
            service.rename_file(&path_string(&note_file), &path_string(&renamed_file), None);
        assert_ok(renamed.error);
        assert_eq!(renamed.source_path, expected_note_path);
        assert_eq!(renamed.target_path, expected_renamed_path);
        assert_eq!(renamed.moved_bytes, "contract text".len() as u64);

        let expected_deleted_file_path = canonical_string(&renamed_file);
        let deleted_file = service.delete_path(&path_string(&renamed_file), false, None);
        assert_ok(deleted_file.error);
        assert_eq!(deleted_file.path, expected_deleted_file_path);
        assert!(!renamed_file.exists());

        let expected_deleted_dir_path = canonical_string(&cache_dir);
        let deleted_dir = service.delete_path(&path_string(&cache_dir), false, None);
        assert_ok(deleted_dir.error);
        assert_eq!(deleted_dir.path, expected_deleted_dir_path);
        assert!(!cache_dir.exists());
    }

    #[test]
    fn file_operations_return_structured_error_envelopes() {
        let temp = TempDir::new("contract_errors");
        let outside = TempDir::new("contract_outside");
        let service = test_service(temp.path());

        let outside_file = outside.path().join("secret.txt");
        fs::write(&outside_file, "secret").expect("failed to write outside file");

        let response = service.read_text_file(&path_string(&outside_file), 0, "utf-8", None);
        let error = response
            .error
            .expect("response should include error envelope");
        assert_eq!(error.code, 4001);
        assert_eq!(error.message, "unsafe path");
        assert!(error.detail.contains("out of allowed roots"));
        assert!(response.path.is_empty());
        assert!(response.content.is_empty());

        let bad_encoding = service.write_text_file(
            &path_string(&temp.path().join("bad.txt")),
            "hello",
            true,
            true,
            "latin1",
            None,
        );
        let error = bad_encoding
            .error
            .expect("response should include error envelope");
        assert_eq!(error.code, 4006);
        assert_eq!(error.message, "unsupported encoding");
        assert!(bad_encoding.path.is_empty());
        assert_eq!(bad_encoding.written_bytes, 0);
    }
}
