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

生产环境中 backend 连接宿主机 `core-agent` 时，保持 `CORE_AGENT_AUTH_MODE=token`，设置强 `CORE_AGENT_SHARED_TOKEN`，并在 backend 侧用相同值配置 `BACKEND_AGENT_SHARED_TOKEN`。

生产环境下 `core-agent` 默认按 deny-by-default 校验启动配置，除非显式设置 `CORE_AGENT_ALLOW_UNSAFE_PRODUCTION_CONFIG=true`。生产配置应保持：

- `CORE_AGENT_AUTH_MODE=token`
- `CORE_AGENT_ALLOWED_ROOTS` 限制为具体目录，不能包含 `/`
- 启用服务操作时，`CORE_AGENT_SERVICE_WHITELIST` 非空
- 启用 Cron 操作时，显式配置 `CORE_AGENT_CRON_ALLOWED_COMMANDS`

不需要的操作类别可以直接关闭：

- `CORE_AGENT_ENABLE_FILE_OPS=false`
- `CORE_AGENT_ENABLE_SERVICE_OPS=false`
- `CORE_AGENT_ENABLE_DOCKER_OPS=false`
- `CORE_AGENT_ENABLE_CRON_OPS=false`

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
- 生产环境在 mTLS 支持实现前保持 `CORE_AGENT_AUTH_MODE=token`，并且不要记录或公开 `CORE_AGENT_SHARED_TOKEN`。
- 将 metrics 端点（`CORE_AGENT_METRICS_HOST:CORE_AGENT_METRICS_PORT`，默认 `127.0.0.1:9108`）限制在本地回环或可信采集网络内。
- 将 OTLP 导出目标限制在可信 collector / tracing backend 范围内。
- 收紧 `CORE_AGENT_ALLOWED_ROOTS`、服务白名单、Cron 命令白名单。
- 条件允许时将 `CORE_AGENT_HOST` 绑定到私网地址，而不是公网 `0.0.0.0`。
- 启用具有破坏性的文件、服务、Docker 或 Cron 操作前，先备份宿主机配置与应用数据。

## Systemd 加固

模板已启用 `NoNewPrivileges=true`、`PrivateTmp=true`、`ProtectSystem=full`、`ProtectHome=read-only`、内核/control-group 保护、显式 `ReadWritePaths`，以及 `RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX`。

取舍说明：

- `ProtectSystem=strict` 更强，但在部分发行版上可能与 Docker socket 或 systemd 交互冲突。建议先使用 `full`，再按目标发行版验证更严格配置。
- `ReadWritePaths` 需要与 `CORE_AGENT_ALLOWED_ROOTS` 对齐；启用 Docker 操作时还需要包含 `/run` 或 `/var/run` 等运行时 socket 路径。
- 基础模板暂不限制 `CapabilityBoundingSet`，因为文件所有权与宿主机服务流程在不同发行版上差异较大。请在验证文件、Docker、systemd 操作后再加入主机专用的最小 capability 集合。
