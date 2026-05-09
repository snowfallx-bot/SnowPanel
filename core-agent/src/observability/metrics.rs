use std::net::SocketAddr;
use std::sync::Arc;
use std::time::Instant;

use anyhow::{Context, Result};
use axum::extract::State;
use axum::http::header::CONTENT_TYPE;
use axum::http::{HeaderValue, StatusCode};
use axum::routing::get;
use axum::{
    response::{IntoResponse, Response},
    Router,
};
use once_cell::sync::Lazy;
use prometheus::{CounterVec, Encoder, GaugeVec, HistogramVec, TextEncoder};
use tracing::info;

static GRPC_REQUESTS_TOTAL: Lazy<CounterVec> = Lazy::new(|| {
    let metric = CounterVec::new(
        prometheus::Opts::new(
            "snowpanel_core_agent_grpc_requests_total",
            "Total number of gRPC requests handled by core-agent.",
        ),
        &["grpc_method", "outcome"],
    )
    .expect("create grpc request total metric");
    prometheus::default_registry()
        .register(Box::new(metric.clone()))
        .expect("register grpc request total metric");
    metric
});

static GRPC_REQUEST_DURATION: Lazy<HistogramVec> = Lazy::new(|| {
    let metric = HistogramVec::new(
        prometheus::HistogramOpts::new(
            "snowpanel_core_agent_grpc_request_duration_seconds",
            "Latency of gRPC requests handled by core-agent.",
        ),
        &["grpc_method", "outcome"],
    )
    .expect("create grpc request duration metric");
    prometheus::default_registry()
        .register(Box::new(metric.clone()))
        .expect("register grpc request duration metric");
    metric
});

static GRPC_REQUESTS_IN_FLIGHT: Lazy<GaugeVec> = Lazy::new(|| {
    let metric = GaugeVec::new(
        prometheus::Opts::new(
            "snowpanel_core_agent_grpc_requests_in_flight",
            "Current number of in-flight gRPC requests handled by core-agent.",
        ),
        &["grpc_method"],
    )
    .expect("create grpc requests in flight metric");
    prometheus::default_registry()
        .register(Box::new(metric.clone()))
        .expect("register grpc requests in flight metric");
    metric
});

pub struct GrpcCallGuard {
    method: String,
    started_at: Instant,
}

pub fn start_grpc_call(method: &str) -> GrpcCallGuard {
    let normalized = normalize_method_label(method);
    GRPC_REQUESTS_IN_FLIGHT
        .with_label_values(&[normalized.as_str()])
        .inc();

    GrpcCallGuard {
        method: normalized,
        started_at: Instant::now(),
    }
}

impl GrpcCallGuard {
    pub fn finish(self, outcome: &str) {
        let normalized_outcome = normalize_outcome_label(outcome);
        GRPC_REQUESTS_TOTAL
            .with_label_values(&[self.method.as_str(), normalized_outcome])
            .inc();
        GRPC_REQUEST_DURATION
            .with_label_values(&[self.method.as_str(), normalized_outcome])
            .observe(self.started_at.elapsed().as_secs_f64());
        GRPC_REQUESTS_IN_FLIGHT
            .with_label_values(&[self.method.as_str()])
            .dec();
    }
}

#[derive(Clone)]
struct MetricsState;

pub async fn run_metrics_server(addr: &str) -> Result<()> {
    let socket_addr: SocketAddr = addr
        .parse()
        .with_context(|| format!("invalid metrics listen address: {addr}"))?;

    let app = Router::new()
        .route("/metrics", get(metrics_handler))
        .with_state(Arc::new(MetricsState));

    let listener = tokio::net::TcpListener::bind(socket_addr)
        .await
        .with_context(|| format!("bind metrics listener failed: {socket_addr}"))?;

    info!("core-agent metrics server listening on {}", socket_addr);

    axum::serve(listener, app)
        .await
        .with_context(|| format!("metrics server terminated unexpectedly on {socket_addr}"))?;

    Ok(())
}

async fn metrics_handler(State(_state): State<Arc<MetricsState>>) -> Response {
    let metric_families = prometheus::gather();
    let mut buffer = Vec::new();
    let encoder = TextEncoder::new();

    if let Err(err) = encoder.encode(&metric_families, &mut buffer) {
        return (
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("encode metrics failed: {err}"),
        )
            .into_response();
    }

    let payload = match String::from_utf8(buffer) {
        Ok(value) => value,
        Err(err) => {
            return (
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("metrics payload is not valid utf-8: {err}"),
            )
                .into_response();
        }
    };

    let mut response = payload.into_response();
    *response.status_mut() = StatusCode::OK;
    if let Ok(content_type) = HeaderValue::from_str(encoder.format_type()) {
        response.headers_mut().insert(CONTENT_TYPE, content_type);
    }
    response
}

fn normalize_method_label(raw: &str) -> String {
    let trimmed = raw.trim();
    if trimmed.is_empty() {
        return "unknown".to_string();
    }
    trimmed.to_string()
}

fn normalize_outcome_label(raw: &str) -> &'static str {
    match raw {
        "ok" => "ok",
        "error" => "error",
        _ => "unknown",
    }
}
