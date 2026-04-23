【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮针对安装阶段 `docker pull postgres:16-alpine` 持续出现 `content size of zero` 的问题，新增了镜像回退机制，避免因单一 tag 异常导致一键安装中断。

本次核心完成项

1. Compose 镜像可配置化：
   - 更新 `docker-compose.yml`
   - 将镜像改为环境变量可覆盖：
     - `postgres` 使用 `${POSTGRES_IMAGE:-postgres:16-alpine}`
     - `redis` 使用 `${REDIS_IMAGE:-redis:7-alpine}`

2. 一键安装脚本容错增强：
   - 更新 `deploy/one-click/ubuntu-25.10/install.sh`
   - 版本号升级为 `0.4.0`
   - 新增参数：
     - `--postgres-image`
     - `--redis-image`
     - `--postgres-image-fallback`
     - `--redis-image-fallback`
   - 默认策略：
     - 主镜像：`postgres:16-alpine` / `redis:7-alpine`
     - 回退镜像：`postgres:16` / `redis:7`
   - 新增 `pull_image_with_fallback`：
     - 先按重试策略拉主镜像
     - 失败后清理疑似损坏本地引用
     - 自动切换回退镜像继续拉取
   - 选定最终镜像后自动写入 `.env` 的 `POSTGRES_IMAGE`/`REDIS_IMAGE`

3. 文档同步：
   - 更新 `deploy/one-click/ubuntu-25.10/README.md`
   - 更新 `deploy/one-click/ubuntu-25.10/README.zh-CN.md`
   - 增加新参数说明与“直接改用非 alpine tag”示例。

本轮修改文件

- `docker-compose.yml`
- `deploy/one-click/ubuntu-25.10/install.sh`
- `deploy/one-click/ubuntu-25.10/README.md`
- `deploy/one-click/ubuntu-25.10/README.zh-CN.md`
- `.claude/change-cache.md`

本地验证

- `docker compose -f docker-compose.yml -f docker-compose.host-agent.yml config` 通过。
- 由于当前环境为 Windows，未在本机完成 Ubuntu 端到端安装执行。

commit摘要

待提交：
- `fix(deploy): add runtime image fallback for one-click installer`

希望接下来的 AI 做什么

1. 在 Ubuntu 25.10 测试机执行安装脚本，确认主镜像失败时能自动回退并继续完成安装。
2. 若仍失败，收集 `/etc/docker/daemon.json` 与 `journalctl -u docker`，评估是否需要脚本内增加“临时禁用 registry mirror 重试”。
3. 补充交付测试记录（安装耗时、登录验证、health/ready、compose ps）。

by: gpt-5.4
