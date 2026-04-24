【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮在远程登录恢复之后，继续把“宿主机 Agent 模式下误用普通 compose 重建，导致 backend 丢失 override、面板功能全部转圈”的坑做成可复用的防呆，并顺手把前端在后端/agent 异常时的长时间转圈体验收敛掉。

本次核心完成项

1. 新增 host-agent 专用 `make` 目标（`Makefile`）：
   - `make up-host-agent`
   - `make down-host-agent`
   - `make logs-host-agent`
   - 目的：把 `docker compose -f docker-compose.yml -f docker-compose.host-agent.yml ...` 固化成固定入口，避免后续手工命令漏掉 override

2. 文档统一切换到 host-agent 专用命令，并明确写出反例：
   - 更新 `README.md`
   - 更新 `README.zh-CN.md`
   - 更新 `docs/development.md`
   - 更新 `docs/development.zh-CN.md`
   - 更新 `docs/deployment.md`
   - 更新 `docs/deployment.zh-CN.md`
   - 更新 `deploy/core-agent/systemd/README.md`
   - 更新 `deploy/core-agent/systemd/README.zh-CN.md`
   - 更新 `deploy/one-click/ubuntu-25.10/README.md`
   - 更新 `deploy/one-click/ubuntu-25.10/README.zh-CN.md`
   - 核心新增说明：
     - 宿主机 Agent 模式后续重建/看日志请使用 `make up-host-agent` / `make logs-host-agent`
     - 不要退回普通 `docker compose up` / `make up`
     - 否则 backend 会丢掉 `docker-compose.host-agent.yml`，重新连回被禁用的容器版 `core-agent`

3. 前端查询默认策略改为更快失败（`frontend/src/main.tsx`）：
   - 为 React Query 配置全局 `QueryClient` 默认项
   - 当接口返回 `ApiError` 且已有明确 HTTP 状态码或业务错误码时，不再反复重试
   - 其他未知错误最多仅重试 1 次
   - mutation 默认不重试
   - 目的：避免宿主机 agent 不通、backend 返回明确错误时，页面长时间停留在“Loading...”

4. 线上现状和根因已沉淀到文档：
   - 用户已可正常登录面板
   - 宿主机 `core-agent` 已成功监听 `0.0.0.0:50051`
   - 页面“持续 loading”根因已定位为：
     - 手工重建时使用了普通 compose
     - 导致 backend 容器内 `AGENT_TARGET=core-agent:50051`
     - 而不是预期的 `host.docker.internal:50051`

本轮修改文件

- `Makefile`
- `README.md`
- `README.zh-CN.md`
- `docs/development.md`
- `docs/development.zh-CN.md`
- `docs/deployment.md`
- `docs/deployment.zh-CN.md`
- `deploy/core-agent/systemd/README.md`
- `deploy/core-agent/systemd/README.zh-CN.md`
- `deploy/one-click/ubuntu-25.10/README.md`
- `deploy/one-click/ubuntu-25.10/README.zh-CN.md`
- `frontend/src/main.tsx`
- `.claude/change-cache.md`

本地验证

- 已完成 `git diff` 复核。
- 已完成 `frontend` 下 `npm exec tsc -b`。
- 已完成 `frontend` 下 `npm exec vite build`。
- 当前环境没有 `make` 可执行文件，未做 `make` 目标级验证。

commit摘要

- `docs(host-agent): harden host-agent rebuild flow`

希望接下来的 AI 做什么

1. 确认这轮 Makefile + 文档 + frontend 查询策略改动已经提交到远端后，再继续做下一步体验优化。
2. 用户服务器后续重建时统一改用：
   - `make up-host-agent`
   - `make logs-host-agent`
3. 如果后续仍反馈某些页面“无报错但长时间空白”，可继续把各页面的 error state 显式化，而不是只依赖全局 query retry 收敛。

by: gpt-5.4
