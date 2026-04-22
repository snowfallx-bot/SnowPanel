pub mod grpc_server;

pub mod proto {
    tonic::include_proto!("snowpanel.agent.v1");
}
