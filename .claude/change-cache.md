【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮接手的是 `frontend-e2e` CI 失败：`scripts/ci/frontend-e2e.ps1` 里 `npm run test:e2e` 返回 exit code 1，但日志尾部只有脚本抛错，不够定位。处理方向是先补稳定性，再补失败时可观测性。

本次核心判断

1. `frontend-e2e.ps1` 存在与此前 `compose-smoke` 同类的时序风险：
   - 登录前只等待了 backend `/ready` 与 frontend `/`
   - 没有等待 frontend 代理链路可用（`/health` 通过 proxy 能打到 backend）
   - 这会造成 e2e 第一波 API 请求偶发失败

2. e2e 用例本身有两个潜在竞态：
   - `fixtures.ts` 登录 fallback 在检测到“invalid credential”前没有等待，可能因为渲染延迟误判
   - `auth-and-nav.spec.ts` 的“受限导航”用例只 mock 了 `/auth/me`，但真实 `/dashboard/summary` 可能返回 401 并触发全局登出跳转，导致用例不稳定

本轮实际改动

1. `scripts/ci/frontend-e2e.ps1`
   - 新增 `Wait-UntilReady -Description "frontend proxy health"`
   - 在执行 `npm run test:e2e` 前，显式检查：
     - `GET $FrontendBaseUrl/health` 返回 `code == 0`
     - `database == up`
     - `agent == up`

2. `frontend/e2e/fixtures.ts`
   - `loginAndMaybeRotate()` 中：
     - 新增 `shellMarker`（`linux panel prototype`）
     - fallback 分支在判断 `invalid credential` 之前，先等待“登录成功标记”或“凭据错误提示”任一出现
   - 目标：消除提交登录后立刻读取错误提示导致的竞态

3. `frontend/e2e/auth-and-nav.spec.ts`
   - 在“受限导航”测试里新增对 `**/api/v1/dashboard/summary` 的 mock 成功响应
   - 避免假 token 命中真实 backend 401 后触发全局重定向，影响菜单断言

4. `.github/workflows/ci.yml`
   - 在 `frontend-e2e` job 新增失败时 artifact 上传：
     - `frontend/playwright-report`
     - `frontend/test-results`
   - 这样下次即便脚本抛错，也能直接看 trace/screenshot/report 定位具体失败步骤

本轮修改文件

- `.claude/change-cache.md`
- `.github/workflows/ci.yml`
- `scripts/ci/frontend-e2e.ps1`
- `frontend/e2e/fixtures.ts`
- `frontend/e2e/auth-and-nav.spec.ts`

本地验证

- 已通过 PowerShell parser 校验 `scripts/ci/frontend-e2e.ps1` 语法
- `frontend` 下 `npm run build` 通过
- 本机无法完整跑 e2e：
  - 这里没有 docker，无法起 compose 环境
  - 本地前端依赖里缺 `@playwright/test`，`npm exec playwright test --list` 无法执行

commit摘要

- 计划提交：`fix(ci): stabilize frontend e2e startup and auth mocks`

希望接下来的 AI 做什么

1. 观察下一次 `frontend-e2e` job 是否通过。
2. 若仍失败，优先下载 `frontend-e2e-artifacts`，按 trace/screenshot 精确定位具体失败用例与步骤。
3. 若通过，再继续推进 `P2-1` 未完成项（proto contract 更细覆盖、integration/e2e 分层）。

by: gpt-5.4
