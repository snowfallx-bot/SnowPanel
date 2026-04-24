【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮继续接手 `frontend-e2e` 失败。上一轮已经补了脚本时序和 artifact 上传，但 CI 仍在 `npm run test:e2e` 处退出。此次进一步定位到 e2e 夹具中的登录 fallback 判定逻辑存在脆弱点。

本次核心判断

1. `frontend/e2e/fixtures.ts` 里 `loginAndMaybeRotate()` 的 fallback 依赖页面错误文案出现：
   - 原逻辑通过 `/invalid credentials|invalid credential/i` 判断是否使用备用密码
   - 但后端当前失败文案是 `invalid username or password`
   - 这会导致“主密码失败但 fallback 未触发”，后续直接超时失败

2. 该判断方式本身不稳定：
   - UI 文案可变
   - 渲染时序也可能导致短窗口误判
   - 相比之下，直接基于登录请求响应判断更稳

本轮实际改动

1. 重构 `submitLogin()`：
   - 从“只点按钮”改为“等待并返回 `/api/v1/auth/login` 响应结果”
   - 返回结构：
     - `status`
     - `payload.code/message`（可解析时）

2. `loginAndMaybeRotate()` 改为基于响应驱动 fallback：
   - 首次登录后记录 `primaryAttempt`
   - 若 `status >= 400` 或 `payload.code !== 0`，则使用 `fallbackPassword` 再次提交
   - 不再依赖固定错误文案匹配

本轮修改文件

- `.claude/change-cache.md`
- `frontend/e2e/fixtures.ts`

本地验证

- `frontend` 下 `npm run build` 通过
- 当前环境仍无法完整跑 e2e（无 docker，且本地 Playwright 依赖不完整）
- 因此最终验证仍需看 GitHub Actions 下一次 `frontend-e2e` 实跑结果

commit摘要

- 计划提交：`fix(e2e): drive login fallback by api response`

希望接下来的 AI 做什么

1. 观察下一次 `frontend-e2e` 是否通过。
2. 若仍失败，下载 `frontend-e2e-artifacts` 并基于 trace 精确定位失败步骤。
3. 如果该问题收敛，再继续推进 `P2-1` 剩余项（integration/e2e 分层与覆盖扩展）。

by: gpt-5.4
