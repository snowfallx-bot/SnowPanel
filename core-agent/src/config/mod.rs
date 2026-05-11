use std::env;

pub struct Config {
    pub app_env: String,
    pub host: String,
    pub port: u16,
    pub metrics_enabled: bool,
    pub metrics_host: String,
    pub metrics_port: u16,
    pub tracing_enabled: bool,
    pub tracing_service_name: String,
    pub tracing_service_version: String,
    pub otlp_endpoint: String,
    pub otlp_insecure: bool,
    pub trace_sample_ratio: f64,
    pub agent_auth: AgentAuthConfig,
    pub allowed_roots: Vec<String>,
    pub max_read_bytes: usize,
    pub max_write_bytes: usize,
    pub service_whitelist: Vec<String>,
    pub cron_allowed_commands: Vec<String>,
    pub cron_allowed_commands_configured: bool,
    pub allow_unsafe_production_config: bool,
    pub feature_gates: OperationFeatureGates,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub enum AgentAuthMode {
    None,
    Token,
    Mtls,
    Unknown(String),
}

#[derive(Clone, Debug)]
pub struct AgentAuthConfig {
    pub mode: AgentAuthMode,
    pub shared_token: String,
    pub tls_ca_file: String,
    pub tls_cert_file: String,
    pub tls_key_file: String,
}

#[derive(Clone, Debug)]
pub struct OperationFeatureGates {
    pub file_ops: bool,
    pub service_ops: bool,
    pub docker_ops: bool,
    pub cron_ops: bool,
}

impl Config {
    pub fn from_env() -> Self {
        let app_env = env::var("APP_ENV").unwrap_or_else(|_| "development".to_string());
        let host = env::var("CORE_AGENT_HOST").unwrap_or_else(|_| "0.0.0.0".to_string());
        let port = env::var("CORE_AGENT_PORT")
            .ok()
            .and_then(|raw| raw.parse::<u16>().ok())
            .unwrap_or(50051);
        let metrics_enabled = env::var("CORE_AGENT_METRICS_ENABLED")
            .ok()
            .map(|raw| parse_bool(&raw))
            .unwrap_or(true);
        let metrics_host =
            env::var("CORE_AGENT_METRICS_HOST").unwrap_or_else(|_| "127.0.0.1".to_string());
        let metrics_port = env::var("CORE_AGENT_METRICS_PORT")
            .ok()
            .and_then(|raw| raw.parse::<u16>().ok())
            .unwrap_or(9108);
        let tracing_enabled = env::var("OTEL_TRACING_ENABLED")
            .ok()
            .map(|raw| parse_bool(&raw))
            .unwrap_or(false);
        let tracing_service_name =
            env::var("OTEL_SERVICE_NAME").unwrap_or_else(|_| "snowpanel-core-agent".to_string());
        let tracing_service_version =
            env::var("OTEL_SERVICE_VERSION").unwrap_or_else(|_| String::new());
        let otlp_endpoint = env::var("OTEL_EXPORTER_OTLP_ENDPOINT").unwrap_or_default();
        let otlp_insecure = env::var("OTEL_EXPORTER_OTLP_INSECURE")
            .ok()
            .map(|raw| parse_bool(&raw))
            .unwrap_or(true);
        let trace_sample_ratio = env::var("OTEL_TRACES_SAMPLER_ARG")
            .ok()
            .and_then(|raw| raw.parse::<f64>().ok())
            .map(clamp_sample_ratio)
            .unwrap_or(1.0);
        let agent_auth = AgentAuthConfig {
            mode: parse_agent_auth_mode(
                &env::var("CORE_AGENT_AUTH_MODE").unwrap_or_else(|_| "none".to_string()),
            ),
            shared_token: env::var("CORE_AGENT_SHARED_TOKEN")
                .unwrap_or_default()
                .trim()
                .to_string(),
            tls_ca_file: env::var("CORE_AGENT_TLS_CA_FILE")
                .unwrap_or_default()
                .trim()
                .to_string(),
            tls_cert_file: env::var("CORE_AGENT_TLS_CERT_FILE")
                .unwrap_or_default()
                .trim()
                .to_string(),
            tls_key_file: env::var("CORE_AGENT_TLS_KEY_FILE")
                .unwrap_or_default()
                .trim()
                .to_string(),
        };
        let allowed_roots = env::var("CORE_AGENT_ALLOWED_ROOTS")
            .unwrap_or_else(|_| "/tmp,/var/tmp,/home".to_string())
            .split(',')
            .map(|item| item.trim().to_string())
            .filter(|item| !item.is_empty())
            .collect::<Vec<_>>();
        let max_read_bytes = env::var("CORE_AGENT_MAX_READ_BYTES")
            .ok()
            .and_then(|raw| raw.parse::<usize>().ok())
            .unwrap_or(1024 * 1024);
        let max_write_bytes = env::var("CORE_AGENT_MAX_WRITE_BYTES")
            .ok()
            .and_then(|raw| raw.parse::<usize>().ok())
            .unwrap_or(1024 * 1024);
        let service_whitelist = env::var("CORE_AGENT_SERVICE_WHITELIST")
            .unwrap_or_else(|_| String::new())
            .split(',')
            .map(|item| item.trim().to_string())
            .filter(|item| !item.is_empty())
            .collect::<Vec<_>>();
        let cron_allowed_commands_raw = env::var("CORE_AGENT_CRON_ALLOWED_COMMANDS");
        let cron_allowed_commands_configured = cron_allowed_commands_raw.is_ok();
        let cron_allowed_commands = cron_allowed_commands_raw
            .unwrap_or_else(|_| "backup,logrotate,cleanup".to_string())
            .split(',')
            .map(|item| item.trim().to_string())
            .filter(|item| !item.is_empty())
            .collect::<Vec<_>>();
        let allow_unsafe_production_config = env::var("CORE_AGENT_ALLOW_UNSAFE_PRODUCTION_CONFIG")
            .ok()
            .map(|raw| parse_bool(&raw))
            .unwrap_or(false);
        let feature_gates = OperationFeatureGates {
            file_ops: env::var("CORE_AGENT_ENABLE_FILE_OPS")
                .ok()
                .map(|raw| parse_bool(&raw))
                .unwrap_or(true),
            service_ops: env::var("CORE_AGENT_ENABLE_SERVICE_OPS")
                .ok()
                .map(|raw| parse_bool(&raw))
                .unwrap_or(true),
            docker_ops: env::var("CORE_AGENT_ENABLE_DOCKER_OPS")
                .ok()
                .map(|raw| parse_bool(&raw))
                .unwrap_or(true),
            cron_ops: env::var("CORE_AGENT_ENABLE_CRON_OPS")
                .ok()
                .map(|raw| parse_bool(&raw))
                .unwrap_or(true),
        };

        Self {
            app_env,
            host,
            port,
            metrics_enabled,
            metrics_host,
            metrics_port,
            tracing_enabled,
            tracing_service_name,
            tracing_service_version,
            otlp_endpoint,
            otlp_insecure,
            trace_sample_ratio,
            agent_auth,
            allowed_roots,
            max_read_bytes,
            max_write_bytes,
            service_whitelist,
            cron_allowed_commands,
            cron_allowed_commands_configured,
            allow_unsafe_production_config,
            feature_gates,
        }
    }

    pub fn address(&self) -> String {
        format!("{}:{}", self.host, self.port)
    }

    pub fn metrics_address(&self) -> String {
        format!("{}:{}", self.metrics_host, self.metrics_port)
    }

    pub fn validate(&self) -> anyhow::Result<()> {
        match self.agent_auth.mode {
            AgentAuthMode::None => {}
            AgentAuthMode::Token => {
                if self.agent_auth.shared_token.trim().is_empty() {
                    anyhow::bail!(
                        "CORE_AGENT_SHARED_TOKEN cannot be empty when CORE_AGENT_AUTH_MODE=token"
                    );
                }
            }
            AgentAuthMode::Mtls => {
                if self.agent_auth.tls_ca_file.trim().is_empty()
                    || self.agent_auth.tls_cert_file.trim().is_empty()
                    || self.agent_auth.tls_key_file.trim().is_empty()
                {
                    anyhow::bail!(
                        "CORE_AGENT_TLS_CA_FILE, CORE_AGENT_TLS_CERT_FILE, and CORE_AGENT_TLS_KEY_FILE are required when CORE_AGENT_AUTH_MODE=mtls"
                    );
                }
                anyhow::bail!("CORE_AGENT_AUTH_MODE=mtls is reserved but not implemented yet");
            }
            AgentAuthMode::Unknown(ref mode) => {
                anyhow::bail!(
                    "CORE_AGENT_AUTH_MODE must be one of: none, token, mtls (got {mode})"
                );
            }
        }

        if self.is_production() && !self.allow_unsafe_production_config {
            self.validate_safe_production_config()?;
        }

        Ok(())
    }

    pub fn startup_warnings(&self) -> Vec<String> {
        let mut warnings = Vec::new();

        if self.allow_unsafe_production_config && self.is_production() {
            warnings.push(
                "CORE_AGENT_ALLOW_UNSAFE_PRODUCTION_CONFIG=true disables production deny-by-default checks"
                    .to_string(),
            );
        }
        if self.feature_gates.file_ops && self.allowed_roots.iter().any(|item| is_root_path(item)) {
            warnings
                .push("CORE_AGENT_ALLOWED_ROOTS includes /; file operations are broad".to_string());
        }
        if self.feature_gates.service_ops && self.service_whitelist.is_empty() {
            warnings.push(
                "CORE_AGENT_SERVICE_WHITELIST is empty; service operations can target any valid service"
                    .to_string(),
            );
        }
        if self.feature_gates.cron_ops && !self.cron_allowed_commands_configured {
            warnings.push(
                "CORE_AGENT_CRON_ALLOWED_COMMANDS is using defaults; configure an explicit cron allowlist"
                    .to_string(),
            );
        }
        if self.metrics_enabled && !is_loopback_host(&self.metrics_host) {
            warnings.push(
                "core-agent metrics endpoint is exposed on a non-loopback address; restrict it with firewall or trusted scrape network"
                    .to_string(),
            );
        }
        if is_wildcard_host(&self.host) && matches!(self.agent_auth.mode, AgentAuthMode::None) {
            warnings.push(
                "core-agent gRPC is bound to a wildcard address without authentication".to_string(),
            );
        }

        warnings
    }

    fn is_production(&self) -> bool {
        self.app_env.trim().eq_ignore_ascii_case("production")
    }

    fn validate_safe_production_config(&self) -> anyhow::Result<()> {
        if matches!(self.agent_auth.mode, AgentAuthMode::None) {
            anyhow::bail!(
                "CORE_AGENT_AUTH_MODE must not be none in production unless CORE_AGENT_ALLOW_UNSAFE_PRODUCTION_CONFIG=true"
            );
        }

        if self.feature_gates.file_ops {
            if self.allowed_roots.is_empty() {
                anyhow::bail!(
                    "CORE_AGENT_ALLOWED_ROOTS must be non-empty in production when CORE_AGENT_ENABLE_FILE_OPS=true"
                );
            }
            if self.allowed_roots.iter().any(|item| is_root_path(item)) {
                anyhow::bail!(
                    "CORE_AGENT_ALLOWED_ROOTS must not include / in production unless CORE_AGENT_ALLOW_UNSAFE_PRODUCTION_CONFIG=true"
                );
            }
        }

        if self.feature_gates.service_ops && self.service_whitelist.is_empty() {
            anyhow::bail!(
                "CORE_AGENT_SERVICE_WHITELIST must be non-empty in production when CORE_AGENT_ENABLE_SERVICE_OPS=true"
            );
        }

        if self.feature_gates.cron_ops {
            if !self.cron_allowed_commands_configured {
                anyhow::bail!(
                    "CORE_AGENT_CRON_ALLOWED_COMMANDS must be explicitly configured in production when CORE_AGENT_ENABLE_CRON_OPS=true"
                );
            }
            if self.cron_allowed_commands.is_empty() {
                anyhow::bail!(
                    "CORE_AGENT_CRON_ALLOWED_COMMANDS must be non-empty in production when CORE_AGENT_ENABLE_CRON_OPS=true"
                );
            }
        }

        Ok(())
    }
}

fn parse_bool(raw: &str) -> bool {
    matches!(
        raw.trim().to_ascii_lowercase().as_str(),
        "1" | "true" | "yes" | "on"
    )
}

fn parse_agent_auth_mode(raw: &str) -> AgentAuthMode {
    match raw.trim().to_ascii_lowercase().as_str() {
        "token" => AgentAuthMode::Token,
        "mtls" => AgentAuthMode::Mtls,
        "" | "none" => AgentAuthMode::None,
        other => AgentAuthMode::Unknown(other.to_string()),
    }
}

fn clamp_sample_ratio(raw: f64) -> f64 {
    raw.clamp(0.0, 1.0)
}

fn is_root_path(raw: &str) -> bool {
    let normalized = raw.trim().replace('\\', "/");
    normalized == "/"
}

fn is_wildcard_host(raw: &str) -> bool {
    let host = raw.trim().trim_matches(['[', ']']);
    host == "0.0.0.0" || host == "::"
}

fn is_loopback_host(raw: &str) -> bool {
    let host = raw.trim().trim_matches(['[', ']']);
    host.eq_ignore_ascii_case("localhost") || host == "127.0.0.1" || host == "::1"
}

#[cfg(test)]
mod tests {
    use super::{AgentAuthConfig, AgentAuthMode, Config, OperationFeatureGates};

    fn base_config(agent_auth: AgentAuthConfig) -> Config {
        Config {
            app_env: "development".to_string(),
            host: "127.0.0.1".to_string(),
            port: 50051,
            metrics_enabled: false,
            metrics_host: "127.0.0.1".to_string(),
            metrics_port: 9108,
            tracing_enabled: false,
            tracing_service_name: "snowpanel-core-agent".to_string(),
            tracing_service_version: String::new(),
            otlp_endpoint: String::new(),
            otlp_insecure: true,
            trace_sample_ratio: 1.0,
            agent_auth,
            allowed_roots: vec!["/tmp".to_string()],
            max_read_bytes: 1024,
            max_write_bytes: 1024,
            service_whitelist: Vec::new(),
            cron_allowed_commands: vec!["backup".to_string()],
            cron_allowed_commands_configured: true,
            allow_unsafe_production_config: false,
            feature_gates: OperationFeatureGates {
                file_ops: true,
                service_ops: true,
                docker_ops: true,
                cron_ops: true,
            },
        }
    }

    fn token_auth_config() -> AgentAuthConfig {
        AgentAuthConfig {
            mode: AgentAuthMode::Token,
            shared_token: "agent-secret-token".to_string(),
            tls_ca_file: String::new(),
            tls_cert_file: String::new(),
            tls_key_file: String::new(),
        }
    }

    fn production_config() -> Config {
        let mut cfg = base_config(token_auth_config());
        cfg.app_env = "production".to_string();
        cfg.service_whitelist = vec!["sshd.service".to_string()];
        cfg
    }

    #[test]
    fn validate_rejects_missing_token_when_token_mode_enabled() {
        let cfg = base_config(AgentAuthConfig {
            mode: AgentAuthMode::Token,
            shared_token: String::new(),
            tls_ca_file: String::new(),
            tls_cert_file: String::new(),
            tls_key_file: String::new(),
        });

        assert!(cfg.validate().is_err());
    }

    #[test]
    fn validate_allows_token_mode_with_token() {
        let cfg = base_config(token_auth_config());

        cfg.validate()
            .expect("token mode with a token should be valid");
    }

    #[test]
    fn production_rejects_none_auth() {
        let mut cfg = production_config();
        cfg.agent_auth.mode = AgentAuthMode::None;
        cfg.agent_auth.shared_token = String::new();

        let err = cfg
            .validate()
            .expect_err("production none auth should be rejected")
            .to_string();

        assert!(err.contains("CORE_AGENT_AUTH_MODE must not be none"));
    }

    #[test]
    fn production_override_allows_unsafe_config() {
        let mut cfg = production_config();
        cfg.agent_auth.mode = AgentAuthMode::None;
        cfg.agent_auth.shared_token = String::new();
        cfg.allowed_roots = vec!["/".to_string()];
        cfg.service_whitelist = Vec::new();
        cfg.cron_allowed_commands_configured = false;
        cfg.allow_unsafe_production_config = true;

        cfg.validate()
            .expect("explicit unsafe production override should bypass deny-by-default checks");
    }

    #[test]
    fn production_rejects_root_allowed_root_when_file_ops_enabled() {
        let mut cfg = production_config();
        cfg.allowed_roots = vec!["/".to_string()];

        let err = cfg
            .validate()
            .expect_err("root allowed root should be rejected")
            .to_string();

        assert!(err.contains("CORE_AGENT_ALLOWED_ROOTS must not include /"));
    }

    #[test]
    fn production_allows_root_allowed_root_when_file_ops_disabled() {
        let mut cfg = production_config();
        cfg.allowed_roots = vec!["/".to_string()];
        cfg.feature_gates.file_ops = false;

        cfg.validate()
            .expect("disabled file ops should skip file allowed-root validation");
    }

    #[test]
    fn production_rejects_empty_service_whitelist_when_service_ops_enabled() {
        let mut cfg = production_config();
        cfg.service_whitelist = Vec::new();

        let err = cfg
            .validate()
            .expect_err("empty service whitelist should be rejected")
            .to_string();

        assert!(err.contains("CORE_AGENT_SERVICE_WHITELIST must be non-empty"));
    }

    #[test]
    fn production_allows_empty_service_whitelist_when_service_ops_disabled() {
        let mut cfg = production_config();
        cfg.service_whitelist = Vec::new();
        cfg.feature_gates.service_ops = false;

        cfg.validate()
            .expect("disabled service ops should skip service whitelist validation");
    }

    #[test]
    fn production_rejects_default_cron_allowlist_when_cron_ops_enabled() {
        let mut cfg = production_config();
        cfg.cron_allowed_commands_configured = false;

        let err = cfg
            .validate()
            .expect_err("implicit cron allowlist should be rejected")
            .to_string();

        assert!(err.contains("CORE_AGENT_CRON_ALLOWED_COMMANDS must be explicitly configured"));
    }

    #[test]
    fn production_allows_default_cron_allowlist_when_cron_ops_disabled() {
        let mut cfg = production_config();
        cfg.cron_allowed_commands_configured = false;
        cfg.feature_gates.cron_ops = false;

        cfg.validate()
            .expect("disabled cron ops should skip cron allowlist validation");
    }

    #[test]
    fn startup_warnings_include_non_loopback_metrics_and_wildcard_unauthenticated_grpc() {
        let mut cfg = base_config(AgentAuthConfig {
            mode: AgentAuthMode::None,
            shared_token: String::new(),
            tls_ca_file: String::new(),
            tls_cert_file: String::new(),
            tls_key_file: String::new(),
        });
        cfg.host = "0.0.0.0".to_string();
        cfg.metrics_enabled = true;
        cfg.metrics_host = "0.0.0.0".to_string();

        let warnings = cfg.startup_warnings();

        assert!(warnings
            .iter()
            .any(|item| item.contains("metrics endpoint")));
        assert!(warnings
            .iter()
            .any(|item| item.contains("without authentication")));
    }
}
