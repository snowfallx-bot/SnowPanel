# Core-Agent Systemd 部署

语言: [English](README.md) | **简体中文**

此目录提供将 `core-agent` 以宿主机 systemd 服务运行的基础模板。

## 文件说明

- `core-agent.service`：systemd unit 模板。
- `core-agent.env.example`：systemd 加载的环境变量模板。

## 安装步骤（Linux 宿主机）

1. 在目标机器构建二进制：
   - `cd core-agent`
   - `cargo build --release`
2. 安装二进制与配置：
   - `sudo install -Dm755 target/release/core-agent /usr/local/bin/core-agent`
   - `sudo install -d -m 750 /etc/snowpanel`
   - `sudo install -Dm640 deploy/core-agent/systemd/core-agent.env.example /etc/snowpanel/core-agent.env`
3. 安装并启动 systemd unit：
   - `sudo install -Dm644 deploy/core-agent/systemd/core-agent.service /etc/systemd/system/core-agent.service`
   - `sudo systemctl daemon-reload`
   - `sudo systemctl enable --now core-agent`
4. 验证：
   - `sudo systemctl status core-agent --no-pager`
   - `ss -lntp | grep 50051`
   - `curl -fsS http://127.0.0.1:9108/metrics | head`

如果还希望宿主机上的 `core-agent` 输出分布式 trace，请在 `/etc/snowpanel/core-agent.env` 中启用：

- `OTEL_TRACING_ENABLED=true`
- `OTEL_SERVICE_NAME=snowpanel-core-agent`
- `OTEL_EXPORTER_OTLP_ENDPOINT=<collector-host>:4317`

## backend 容器 + 宿主机 agent 运行方式

当 backend 在 Docker 中运行、`core-agent` 在宿主机运行时，使用：

- `make up-host-agent`

该覆盖文件会将 backend 指向 `host.docker.internal:50051`，并默认禁用容器版 `core-agent` 服务。

后续如果需要重建或查看日志，也请持续使用：

- `make up-host-agent`
- `make logs-host-agent`

不要再退回普通 `docker compose up` / `make up`，否则 backend 会重新连回容器版 `core-agent`，而不是宿主机 systemd 服务。

## 安全提示

- 将 `50051` 端口限制在可信网络（防火墙/内网）内。
- 将 metrics 端点（`CORE_AGENT_METRICS_HOST:CORE_AGENT_METRICS_PORT`，默认 `127.0.0.1:9108`）限制在本地回环或可信采集网络内。
- 将 OTLP 导出目标限制在可信 collector / tracing backend 范围内。
- 收紧 `CORE_AGENT_ALLOWED_ROOTS`、服务白名单、Cron 命令白名单。
- 条件允许时将 `CORE_AGENT_HOST` 绑定到私网地址，而不是公网 `0.0.0.0`。
