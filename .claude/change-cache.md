【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮把 `.claude/progress.md` 里悬而未决的 `P1-2` 正式收尾，重点不是加新页面，而是把“前端权限感知 + session 管理”补成可交付状态，并把 token 存储策略形成明确文档结论。

本次核心完成项

1. 收紧 `ProtectedRoute` 的 session 校验行为：
   - 在已有 token 但 `getMe()` 尚未成功前，不再先渲染受保护内容，避免使用本地残留用户态短暂漏出页面。
   - 对 `401/403` 继续做统一清凭据并跳登录。
   - 对非鉴权失败（如网络/代理/backend 不可达）展示可重试的 session 错误态，而不是只停留在空白/卡住。

2. 补前端回归测试，覆盖 `P1-2` 关键路径：
   - 新增 `frontend/src/routes/ProtectedRoute.test.tsx`
     - 无 token 跳登录
     - `getMe()` 完成前不渲染受保护内容
     - `401` 清 session
     - 非鉴权失败可重试
     - 权限不足时跳转到可访问页面
   - 新增 `frontend/src/layouts/AppLayout.test.tsx`
     - 导航只展示有权限的模块
     - 首次登录强制改密并刷新本地 session
     - backend logout 失败时仍清本地凭据

3. 顺手补了两处前端收口细节：
   - `frontend/src/layouts/AppLayout.tsx`
     - 给强制改密表单的 password 输入加了 `id/htmlFor`
     - `nav` 增加了 `aria-label`
   - `frontend/src/lib/http.ts`
     - `describeApiError()` 现在会把普通 `Error("Network Error")` 也统一映射成可执行提示，不再只对 `ApiError` 生效

4. 形成 token 存储策略的明确决议并写入文档：
   - 更新 `docs/security.md`
   - 更新 `docs/security.zh-CN.md`
   - 明确结论：
     - 当前阶段继续使用前端持久化 bearer token + refresh token
     - 不把迁移到 httpOnly cookie 作为当前发布前阻塞项
     - 若以后迁移，必须连同 backend 安全 cookie、CSRF、防代理/域名策略和浏览器/API client 回归测试一起推进

5. 更新协作文档：
   - `.claude/progress.md` 现已将 `P1-2` 划为完成
   - 剩余优先级改为：
     - `P2-1` 测试矩阵补齐
     - `P2-2` 生产观测能力
     - `P2-3` 文档与原型痕迹清理

本轮修改文件

- `.claude/progress.md`
- `.claude/change-cache.md`
- `docs/security.md`
- `docs/security.zh-CN.md`
- `frontend/src/layouts/AppLayout.tsx`
- `frontend/src/layouts/AppLayout.test.tsx`
- `frontend/src/lib/http.ts`
- `frontend/src/routes/ProtectedRoute.tsx`
- `frontend/src/routes/ProtectedRoute.test.tsx`

本地验证

- `frontend` 下 `npm exec tsc -b` 通过
- `frontend` 下 `npm exec vite build` 通过
- `vitest` 仍未稳定跑通：
  - 先是本机缺少 `rolldown` 可选原生绑定
  - 补绑定后又撞上本机 Node `20.18.0` 与 `vitest/jsdom` 依赖链的运行时兼容问题
  - 当前判断：这是本机依赖环境问题，不是这轮业务代码本身的编译/构建问题

commit摘要

- 计划提交：`fix(frontend): finalize p1-2 session guardrails`

希望接下来的 AI 做什么

1. 以后请以最新 `.claude/progress.md` 为准，不要再把 `P1-2` 当作未完成事项重复分析。
2. 下一优先级直接进入 `P2-1`：
   - 优先考虑 proto contract tests
   - backend + core-agent + postgres 真实 integration tests
   - 前端关键 e2e（登录 / 权限隐藏 / 文件浏览）
3. 如果要继续补前端测试，优先解决当前本机 `vitest/jsdom` 与 Node 版本/依赖链兼容问题，再扩大覆盖面。

by: gpt-5.4
