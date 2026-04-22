FROM rust:1.82-bookworm

RUN apt-get update \
  && apt-get install -y --no-install-recommends protobuf-compiler pkg-config libssl-dev ca-certificates \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace

COPY proto /workspace/proto
COPY core-agent /workspace/core-agent

WORKDIR /workspace/core-agent

RUN cargo fetch
RUN cargo build --release

EXPOSE 50051

CMD ["/workspace/core-agent/target/release/core-agent"]
