【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮继续按照 `.claude/progress.md` 推进 `P2-1`，没有去扩页面或做 UI，而是把现有 CI 从“单测 + build”为主，往“真实 compose 主链路 smoke integration”推进了一步。

本次核心完成项

1. 新增 compose 级冒烟脚本：
   - 新增 `scripts/ci/compose-smoke.ps1`
   - 这是一个跨平台 PowerShell 7 脚本，目标是给 GitHub Actions 的 Ubuntu runner 直接跑
   - 脚本会：
     - 用 production 模式起 `postgres + redis + core-agent + backend + frontend`
     - 显式注入强 `JWT_SECRET`
     - 显式注入强 `DEFAULT_ADMIN_PASSWORD`
     - 把 `LOGIN_ATTEMPT_STORE` 切到 `redis`
     - 等待 backend `/ready` 和 frontend 启动成功

2. 这条 smoke 覆盖的主链路：
   - 通过 frontend 同源代理登录 `POST /api/v1/auth/login`
   - 断言 bootstrap admin 首次登录 `must_change_password=true`
   - 断言改密前访问 `/api/v1/dashboard/summary` 会被 `403` 拦下
   - 调 `/api/v1/auth/change-password` 完成首次改密
   - 调 `/api/v1/auth/refresh` 验证 refresh rotation
   - 验证旧 refresh token 会失效
   - 调 `/api/v1/dashboard/summary` 验证 backend ↔ core-agent 主链路
   - 调 `/api/v1/files/list`
   - 调 `/api/v1/files/write` / `read` / `rename` / `delete`
   - 调 `/api/v1/auth/logout`
   - 验证 logout 后 access token / refresh token 都失效

3. 把 smoke 接进 CI：
   - 更新 `.github/workflows/ci.yml`
   - 原来的 `container-build` job 被替换成 `compose-smoke`
   - `compose-smoke` 依赖：
     - `backend`
     - `core-agent`
     - `frontend`
   - 这样 CI 现在形成了更清晰的一层：
     - 单元/组件测试先跑
     - 然后再跑 compose 级真实主链路 smoke

4. 同步进度文档：
   - 更新 `.claude/progress.md`
   - `P2-1` 仍未划掉，但“当前已有”现在明确包含：
     - compose-based smoke integration
   - “明显缺失”改成更准确的说法：
     - 仍缺 proto contract tests
     - 仍缺更系统的 backend + core-agent + postgres integration 覆盖
     - 仍缺前端 e2e
     - 仍缺更完整的 CI 分层矩阵

本轮修改文件

- `.claude/change-cache.md`
- `.claude/progress.md`
- `.github/workflows/ci.yml`
- `scripts/ci/compose-smoke.ps1`

本地验证

- 已用 PowerShell parser 对 `scripts/ci/compose-smoke.ps1` 做语法解析，未发现语法错误
- 已人工复查 workflow 与脚本逻辑链路
- 当前本机没有 `docker` / `bash`，所以无法在本地实际起 compose 执行 smoke
- 因此这轮最关键的真实执行验证将依赖 GitHub Actions 里的 `compose-smoke` job

commit摘要

- 计划提交：`test(ci): add compose smoke coverage`

希望接下来的 AI 做什么

1. 下一步先观察 `compose-smoke` 在 CI 上的第一次真实运行结果，优先修复任何环境相关或接口断言问题。
2. 在此基础上继续补 `P2-1` 剩余缺口，优先级建议：
   - proto contract tests
   - 更系统的 backend + core-agent + postgres integration tests
   - 前端 e2e（登录 / 权限隐藏 / 文件浏览）
3. 后续如果要扩 CI 分层，建议把“轻量 smoke”和“更重的 integration/e2e”拆成独立 job，而不是把所有断言继续塞进同一个脚本。

by: gpt-5.4
