【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮继续在上一轮 Playwright e2e 基建之上推进 `P2-1`，目标是把前端 e2e 接进独立的 CI job，而不是把浏览器测试继续塞回已有 `compose-smoke`。

本次核心完成项

1. 新增独立 frontend e2e 运行脚本：
   - 新增 `scripts/ci/frontend-e2e.ps1`
   - 该脚本会：
     - 用独立 project name 起 `postgres + redis + core-agent + backend + frontend`
     - 复用 smoke 同一套端口、JWT secret、bootstrap password 约定
     - 等待 backend `/ready` 和 frontend 启动成功
     - 进入 `frontend/` 执行 `npm run test:e2e`
     - 失败时自动输出 compose `ps` 和 `logs`
     - 收尾时执行 `docker compose down -v --remove-orphans`

2. 把 Playwright 接入独立 CI job：
   - 修改 `.github/workflows/ci.yml`
   - 新增 `frontend-e2e` job
   - job 依赖：
     - `backend`
     - `core-agent`
     - `frontend`
     - `compose-smoke`
   - job 会：
     - `npm ci`
     - `npx playwright install --with-deps chromium`
     - 执行 `./scripts/ci/frontend-e2e.ps1`

3. 保持 CI 分层清晰：
   - `compose-smoke` 仍负责 API / auth / files 主链路 smoke
   - `frontend-e2e` 负责浏览器级回归
   - 没有把 Playwright 回塞进 smoke 脚本，避免职责混杂

本轮修改文件

- `.claude/change-cache.md`
- `.github/workflows/ci.yml`
- `scripts/ci/frontend-e2e.ps1`

本地验证

已通过：
- `cd frontend && npm run test`
- `npx playwright test --list`
- `frontend-e2e.ps1` PowerShell 语法解析检查

验证结果：
- Vitest：6 个文件、24 个测试通过
- Playwright：成功发现 3 个 e2e 用例
- `frontend-e2e.ps1`：PowerShell parser 返回 `ok`

当前限制

- 本轮没有在本机完整执行 Linux compose + Playwright 闭环，最终验收依赖 GitHub Actions 上的 `frontend-e2e` job
- 该 job 当前仍复用真实 Linux `/tmp` 路径和 compose 环境，这是符合仓库目标运行面的

commit摘要

- 计划提交：`test(ci): add dedicated frontend e2e job`

希望接下来的 AI 做什么

1. 先观察 `frontend-e2e` 首次 CI 运行结果。
2. 如果失败：
   - 优先看 Playwright 浏览器依赖是否安装齐全
   - 再看 `frontend-e2e.ps1` 的服务等待与环境变量注入
   - 再看具体是哪条 e2e 场景失败（登录 / 权限导航 / 文件页）
3. 如果通过：
   - `P2-1` 可以基本从“只有 smoke 没有浏览器回归”提升到“有独立前端 e2e 层”
   - 下一步可转向更系统的 backend + core-agent + postgres integration，或进入 `P2-2`

by: claude-sonnet-4-6
