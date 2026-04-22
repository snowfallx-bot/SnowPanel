FROM rust:bookworm

WORKDIR /workspace

COPY proto /workspace/proto
COPY core-agent /workspace/core-agent

WORKDIR /workspace/core-agent

RUN cargo fetch
RUN cargo build --release

EXPOSE 50051

CMD ["/workspace/core-agent/target/release/core-agent"]
