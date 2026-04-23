【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮针对 Ubuntu 25.10 一键安装过程中的 Docker 拉取失败（`descriptor content size of zero`）进行了安装器稳定性修复。

本次核心完成项

1. 一键安装脚本增强（网络抖动容错）：
   - 更新 `deploy/one-click/ubuntu-25.10/install.sh`
   - 新增参数：
     - `--docker-registry-mirror <url>`：配置 Docker daemon 镜像加速
     - `--docker-pull-retries <n>`：控制镜像拉取与 compose 启动重试次数（默认 3）
   - 新增能力：
     - `configure_docker_registry_mirror`：在 `/etc/docker/daemon.json` 合并写入 `registry-mirrors`（保留已有配置；非法 JSON 自动备份并修复）
     - `run_with_retry`：失败后自动重试，并在重试前重启 Docker daemon
     - 在 `compose up` 前，先预拉取 `postgres:16-alpine` 与 `redis:7-alpine` 并重试
     - `docker compose up -d --build` 也纳入重试路径
   - 新增参数校验：
     - `DOCKER_PULL_RETRIES` 必须为正整数
     - `DOCKER_REGISTRY_MIRROR` 必须是 `http://` 或 `https://` URL

2. 一键安装文档同步：
   - 更新 `deploy/one-click/ubuntu-25.10/README.md`
   - 更新 `deploy/one-click/ubuntu-25.10/README.zh-CN.md`
   - 补充了新参数说明与示例（含 `--docker-pull-retries`）。

本轮修改文件

- `deploy/one-click/ubuntu-25.10/install.sh`
- `deploy/one-click/ubuntu-25.10/README.md`
- `deploy/one-click/ubuntu-25.10/README.zh-CN.md`
- `.claude/change-cache.md`

本地验证

- 脚本逻辑与 diff 已检查。
- 由于当前为 Windows 工作环境，无法本机直接执行 Ubuntu bash 安装脚本做端到端验证。

commit摘要

待提交：
- `fix(deploy): harden one-click installer against docker pull failures`

希望接下来的 AI 做什么

1. 在 Ubuntu 25.10 测试机验证以下两种场景：
   - 默认参数安装（无镜像加速）
   - `--docker-registry-mirror` + `--docker-pull-retries 5` 安装
2. 若现场网络仍不稳定，补充“推荐镜像加速配置”到部署文档。
3. 收集安装日志，确认 `core-agent`、`backend /ready`、前端登录全链路通过。

by: gpt-5.4
