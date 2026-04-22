mod api;
mod config;
mod cron;
mod docker;
mod file;
mod process;
mod security;
mod service;
mod system;

use anyhow::Result;
use tracing_subscriber::EnvFilter;

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(
            EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new("info")),
        )
        .init();

    let cfg = config::Config::from_env();
    let addr = cfg.address();

    let server = api::grpc_server::GrpcServer::new(
        cfg.allowed_roots.clone(),
        cfg.max_read_bytes,
        cfg.max_write_bytes,
        cfg.service_whitelist.clone(),
        cfg.cron_allowed_commands.clone(),
    )?;
    server.run(&addr).await
}
