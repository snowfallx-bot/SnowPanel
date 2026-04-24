【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮继续沿着“登录虽恢复，但 agent/backend 异常时前端页面仍容易表现为长时间空白或用户感知不明确”的方向推进，把核心页面的 query 失败状态显式化，并统一补上就地重试入口。

本次核心完成项

1. 新增共享查询错误卡片组件（`frontend/src/components/ui/query-error-card.tsx`）：
   - 统一展示错误标题、错误正文、可选排障提示、`Retry` 按钮
   - 避免每个页面各自写一套红字提示，交互和视觉都更一致

2. 新增通用 API 错误文案映射（`frontend/src/lib/http.ts`）：
   - 新增 `describeApiError`
   - 对以下场景给出更可执行的前端提示：
     - `3001` / `503`：`core-agent` 不可用
     - `401`：会话失效
     - `403`：权限不足
     - `404`：前后端版本/路由不匹配，或代理目标错误
     - `5xx`：backend 内部错误
     - timeout / network error：API 不可达

3. 将核心页面的列表/详情查询切换为显式错误态：
   - 更新 `frontend/src/pages/DashboardPage.tsx`
   - 更新 `frontend/src/pages/ServicesPage.tsx`
   - 更新 `frontend/src/pages/DockerPage.tsx`
   - 更新 `frontend/src/pages/FilesPage.tsx`
   - 更新 `frontend/src/pages/CronPage.tsx`
   - 更新 `frontend/src/pages/TasksPage.tsx`
   - 更新 `frontend/src/pages/AuditLogsPage.tsx`
   - 主要变化：
     - query 失败时不再主要依赖页面底部 `message`
     - 改为在对应内容区直接展示错误卡片
     - 支持页面内直接点击 `Retry`
     - Docker / Tasks 这类双查询页面会分别在各自区块展示错误，不互相遮挡

4. 补充测试覆盖：
   - 新增 `frontend/src/lib/http.test.ts`
   - 更新 `frontend/src/pages/CronPage.test.tsx`
   - 覆盖点：
     - 通用错误映射文案
     - cron 列表首次失败后，点击 `Retry` 能恢复为正常列表

本轮修改文件

- `frontend/src/components/ui/query-error-card.tsx`
- `frontend/src/lib/http.ts`
- `frontend/src/lib/http.test.ts`
- `frontend/src/pages/DashboardPage.tsx`
- `frontend/src/pages/ServicesPage.tsx`
- `frontend/src/pages/DockerPage.tsx`
- `frontend/src/pages/FilesPage.tsx`
- `frontend/src/pages/CronPage.tsx`
- `frontend/src/pages/CronPage.test.tsx`
- `frontend/src/pages/TasksPage.tsx`
- `frontend/src/pages/AuditLogsPage.tsx`
- `.claude/change-cache.md`

本地验证

- 已完成 `frontend` 下 `npm exec tsc -b`。
- 已完成 `frontend` 下 `npm exec vite build`。
- `npm exec vitest run` 仍因当前 Windows/npm 环境缺少 `rolldown` 可选原生绑定而失败，错误与本次代码逻辑无直接关系。
- 已完成 `git diff` 复核。

commit摘要

- `fix(frontend): surface actionable query error states`

希望接下来的 AI 做什么

1. 将这轮前端错误态改动提交并推送到远端。
2. 如果用户继续反馈“页面不再转圈，但不知道下一步怎么修”，可把错误卡片里的 hint 再细化为按页面区分的排障建议。
3. 等本地依赖问题解决后，补跑 `vitest`，确保新加的错误态测试在 CI/本地环境都稳定通过。

by: gpt-5.4
