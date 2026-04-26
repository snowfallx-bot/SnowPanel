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
    pub allowed_roots: Vec<String>,
    pub max_read_bytes: usize,
    pub max_write_bytes: usize,
    pub service_whitelist: Vec<String>,
    pub cron_allowed_commands: Vec<String>,
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
        let cron_allowed_commands = env::var("CORE_AGENT_CRON_ALLOWED_COMMANDS")
            .unwrap_or_else(|_| "backup,logrotate,cleanup".to_string())
            .split(',')
            .map(|item| item.trim().to_string())
            .filter(|item| !item.is_empty())
            .collect::<Vec<_>>();

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
            allowed_roots,
            max_read_bytes,
            max_write_bytes,
            service_whitelist,
            cron_allowed_commands,
        }
    }

    pub fn address(&self) -> String {
        format!("{}:{}", self.host, self.port)
    }

    pub fn metrics_address(&self) -> String {
        format!("{}:{}", self.metrics_host, self.metrics_port)
    }
}

fn parse_bool(raw: &str) -> bool {
    matches!(
        raw.trim().to_ascii_lowercase().as_str(),
        "1" | "true" | "yes" | "on"
    )
}

fn clamp_sample_ratio(raw: f64) -> f64 {
    raw.clamp(0.0, 1.0)
}
