use std::env;

pub struct Config {
    pub host: String,
    pub port: u16,
    pub allowed_roots: Vec<String>,
    pub max_read_bytes: usize,
    pub max_write_bytes: usize,
    pub service_whitelist: Vec<String>,
    pub cron_allowed_commands: Vec<String>,
}

impl Config {
    pub fn from_env() -> Self {
        let host = env::var("CORE_AGENT_HOST").unwrap_or_else(|_| "0.0.0.0".to_string());
        let port = env::var("CORE_AGENT_PORT")
            .ok()
            .and_then(|raw| raw.parse::<u16>().ok())
            .unwrap_or(50051);
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
            host,
            port,
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
}
