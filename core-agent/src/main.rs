mod api;
mod config;
mod cron;
mod docker;
mod file;
mod observability;
mod process;
mod security;
mod service;
mod system;

use anyhow::Result;

#[tokio::main]
async fn main() -> Result<()> {
    let cfg = config::Config::from_env();
    let _tracing_guard = match observability::tracing::TracingGuard::init(&cfg) {
        Ok(guard) => guard,
        Err(err) => {
            eprintln!("otel tracing disabled: {err:#}");
            tracing_subscriber::fmt()
                .with_env_filter(tracing_subscriber::EnvFilter::try_from_default_env().unwrap_or_else(
                    |_| tracing_subscriber::EnvFilter::new("info"),
                ))
                .init();
            observability::tracing::TracingGuard::disabled()
        }
    };

    let addr = cfg.address();
    let metrics_enabled = cfg.metrics_enabled;
    let metrics_addr = cfg.metrics_address();

    let server = api::grpc_server::GrpcServer::new(
        cfg.allowed_roots.clone(),
        cfg.max_read_bytes,
        cfg.max_write_bytes,
        cfg.service_whitelist.clone(),
        cfg.cron_allowed_commands.clone(),
    )?;

    if metrics_enabled {
        tokio::try_join!(
            async { server.run(&addr).await },
            async { observability::metrics::run_metrics_server(&metrics_addr).await }
        )?;
        return Ok(());
    }

    server.run(&addr).await
}
