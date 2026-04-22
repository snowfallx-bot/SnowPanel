use std::fmt::{Display, Formatter};
use std::fs;
use std::path::{Path, PathBuf};

#[derive(Clone, Copy)]
pub enum FileOperation {
    List,
    Read,
    Write,
    Mkdir,
    Delete,
}

#[derive(Debug)]
pub enum PathValidationError {
    EmptyPath,
    InvalidPath(String),
    UnsafePath(String),
    DangerousPath(String),
    IOError(String),
}

impl Display for PathValidationError {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::EmptyPath => write!(f, "path is empty"),
            Self::InvalidPath(msg) => write!(f, "invalid path: {msg}"),
            Self::UnsafePath(msg) => write!(f, "unsafe path: {msg}"),
            Self::DangerousPath(msg) => write!(f, "dangerous path: {msg}"),
            Self::IOError(msg) => write!(f, "io error: {msg}"),
        }
    }
}

impl std::error::Error for PathValidationError {}

#[derive(Clone)]
pub struct PathValidator {
    allowed_roots: Vec<PathBuf>,
}

impl PathValidator {
    pub fn new(allowed_roots: Vec<PathBuf>) -> Self {
        Self { allowed_roots }
    }

    pub fn validate(
        &self,
        raw_path: &str,
        override_roots: &[String],
        enforce_safe_root: bool,
        operation: FileOperation,
    ) -> Result<PathBuf, PathValidationError> {
        let trimmed = raw_path.trim();
        if trimmed.is_empty() {
            return Err(PathValidationError::EmptyPath);
        }

        let raw = PathBuf::from(trimmed);
        if !raw.is_absolute() {
            return Err(PathValidationError::InvalidPath(
                "absolute path is required".to_string(),
            ));
        }

        let normalized = normalize_path(&raw)?;
        if enforce_safe_root {
            let allowed_roots = self.resolve_roots(override_roots)?;
            if allowed_roots.is_empty() {
                return Err(PathValidationError::UnsafePath(
                    "no allowed root configured".to_string(),
                ));
            }

            let allowed = allowed_roots.iter().any(|root| normalized.starts_with(root));
            if !allowed {
                return Err(PathValidationError::UnsafePath(format!(
                    "path '{}' is out of allowed roots",
                    normalized.display()
                )));
            }
        }

        if is_dangerous_target(&normalized, operation) {
            return Err(PathValidationError::DangerousPath(format!(
                "operation is blocked for '{}'",
                normalized.display()
            )));
        }

        Ok(normalized)
    }

    fn resolve_roots(&self, override_roots: &[String]) -> Result<Vec<PathBuf>, PathValidationError> {
        let roots = if override_roots.is_empty() {
            self.allowed_roots.clone()
        } else {
            override_roots.iter().map(PathBuf::from).collect()
        };

        roots
            .into_iter()
            .map(|root| normalize_path(&root))
            .collect::<Result<Vec<_>, _>>()
    }
}

fn normalize_path(path: &Path) -> Result<PathBuf, PathValidationError> {
    if path.exists() {
        return fs::canonicalize(path)
            .map_err(|err| PathValidationError::IOError(format!("canonicalize failed: {err}")));
    }

    let parent = path.parent().ok_or_else(|| {
        PathValidationError::InvalidPath("path has no parent for normalization".to_string())
    })?;
    let file_name = path.file_name().ok_or_else(|| {
        PathValidationError::InvalidPath("path has no filename for normalization".to_string())
    })?;

    let normalized_parent = fs::canonicalize(parent).map_err(|err| {
        PathValidationError::IOError(format!(
            "canonicalize parent '{}' failed: {err}",
            parent.display()
        ))
    })?;

    Ok(normalized_parent.join(file_name))
}

fn is_dangerous_target(path: &Path, operation: FileOperation) -> bool {
    if !matches!(operation, FileOperation::Delete | FileOperation::Write | FileOperation::Mkdir) {
        return false;
    }

    let blocked = [
        Path::new("/"),
        Path::new("/bin"),
        Path::new("/boot"),
        Path::new("/dev"),
        Path::new("/etc"),
        Path::new("/lib"),
        Path::new("/proc"),
        Path::new("/root"),
        Path::new("/sbin"),
        Path::new("/sys"),
        Path::new("/usr"),
    ];

    blocked.iter().any(|item| path == *item)
}

#[cfg(test)]
mod tests {
    use super::{FileOperation, PathValidationError, PathValidator};
    use std::fs;
    use std::path::PathBuf;
    use std::time::{SystemTime, UNIX_EPOCH};

    fn make_temp_dir(prefix: &str) -> PathBuf {
        let mut dir = std::env::temp_dir();
        let nonce = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .expect("system time must be after unix epoch")
            .as_nanos();
        dir.push(format!("snowpanel_{prefix}_{nonce}"));
        fs::create_dir_all(&dir).expect("failed to create temp dir");
        dir
    }

    #[test]
    fn validate_returns_normalized_path_inside_allowed_root() {
        let root = make_temp_dir("path_validator_root");
        let nested = root.join("sub");
        fs::create_dir_all(&nested).expect("failed to create nested dir");

        let validator = PathValidator::new(vec![root.clone()]);
        let output = validator
            .validate(
                nested
                    .to_str()
                    .expect("path should be valid utf8 for this test"),
                &[],
                true,
                FileOperation::List,
            )
            .expect("path should be accepted");

        assert!(
            output.starts_with(&root),
            "normalized path should stay within allowed root"
        );

        fs::remove_dir_all(root).expect("failed to cleanup temp dir");
    }

    #[test]
    fn validate_rejects_relative_path() {
        let root = make_temp_dir("path_validator_relative");
        let validator = PathValidator::new(vec![root.clone()]);

        let result = validator.validate("relative/path.txt", &[], true, FileOperation::Read);
        assert!(
            matches!(result, Err(PathValidationError::InvalidPath(_))),
            "relative path should be rejected"
        );

        fs::remove_dir_all(root).expect("failed to cleanup temp dir");
    }

    #[test]
    fn validate_rejects_path_outside_allowed_root() {
        let root = make_temp_dir("path_validator_allowed");
        let outside = make_temp_dir("path_validator_outside");
        let target = outside.join("file.txt");
        fs::write(&target, "hello").expect("failed to create outside file");

        let validator = PathValidator::new(vec![root.clone()]);
        let result = validator.validate(
            target
                .to_str()
                .expect("path should be valid utf8 for this test"),
            &[],
            true,
            FileOperation::Read,
        );
        assert!(
            matches!(result, Err(PathValidationError::UnsafePath(_))),
            "path outside root should be rejected"
        );

        fs::remove_dir_all(root).expect("failed to cleanup temp dir");
        fs::remove_dir_all(outside).expect("failed to cleanup outside dir");
    }

    #[cfg(unix)]
    #[test]
    fn validate_blocks_dangerous_delete_targets() {
        let validator = PathValidator::new(vec![PathBuf::from("/")]);
        let result = validator.validate("/etc", &[], false, FileOperation::Delete);
        assert!(
            matches!(result, Err(PathValidationError::DangerousPath(_))),
            "dangerous delete target should be blocked"
        );
    }
}
