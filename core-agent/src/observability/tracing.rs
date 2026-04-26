use anyhow::{anyhow, Context, Result};
use opentelemetry::{global, trace::TracerProvider as _, KeyValue};
use opentelemetry_otlp::{WithExportConfig, WithTonicConfig};
use opentelemetry_sdk::{
    propagation::TraceContextPropagator,
    trace::{Sampler, SdkTracerProvider},
    Resource,
};
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt, EnvFilter};

use crate::config::Config;

pub struct TracingGuard {
    tracer_provider: Option<SdkTracerProvider>,
}

impl TracingGuard {
    pub fn disabled() -> Self {
        Self {
            tracer_provider: None,
        }
    }

    pub fn init(cfg: &Config) -> Result<Self> {
        global::set_text_map_propagator(TraceContextPropagator::new());

        let env_filter =
            EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new("info"));

        if !cfg.tracing_enabled {
            tracing_subscriber::registry()
                .with(env_filter)
                .with(tracing_subscriber::fmt::layer())
                .init();
            return Ok(Self::disabled());
        }

        let endpoint = cfg.otlp_endpoint.trim();
        if endpoint.is_empty() {
            return Err(anyhow!(
                "OTEL_EXPORTER_OTLP_ENDPOINT is required when OTEL_TRACING_ENABLED=true"
            ));
        }

        let mut exporter_builder = opentelemetry_otlp::SpanExporter::builder()
            .with_tonic()
            .with_endpoint(endpoint.to_string());
        if cfg.otlp_insecure {
            exporter_builder = exporter_builder.with_insecure();
        }

        let exporter = exporter_builder
            .build()
            .context("create OTLP trace exporter")?;

        let mut attributes = vec![KeyValue::new(
            "deployment.environment.name",
            cfg.app_env.clone(),
        )];
        if !cfg.tracing_service_version.trim().is_empty() {
            attributes.push(KeyValue::new(
                "service.version",
                cfg.tracing_service_version.clone(),
            ));
        }

        let resource = Resource::builder()
            .with_service_name(cfg.tracing_service_name.clone())
            .with_attributes(attributes)
            .build();

        let tracer_provider = SdkTracerProvider::builder()
            .with_sampler(Sampler::ParentBased(Box::new(Sampler::TraceIdRatioBased(
                cfg.trace_sample_ratio,
            ))))
            .with_resource(resource)
            .with_batch_exporter(exporter)
            .build();

        let tracer = tracer_provider.tracer(cfg.tracing_service_name.clone());

        tracing_subscriber::registry()
            .with(env_filter)
            .with(tracing_subscriber::fmt::layer())
            .with(tracing_opentelemetry::layer().with_tracer(tracer))
            .init();

        Ok(Self {
            tracer_provider: Some(tracer_provider),
        })
    }
}

impl Drop for TracingGuard {
    fn drop(&mut self) {
        if let Some(provider) = self.tracer_provider.take() {
            let _ = provider.shutdown();
        }
    }
}
