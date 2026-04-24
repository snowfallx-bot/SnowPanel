【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮接手的是 `P2-1` 新增的 compose smoke CI 失败问题。GitHub Actions 首次运行 `scripts/ci/compose-smoke.ps1` 时，在通过 frontend 代理执行 `POST /api/v1/auth/login` 这一步返回了 `500`，失败点位于脚本第一个代理登录请求之前。

本次核心判断

1. 失败更像“前端首页已起来，但前端代理到 backend 还未完全可用”的时序问题，而不是 backend `/ready` 本身失败：
   - 现有脚本在登录前只等待了：
     - backend `/ready`
     - frontend `/`
   - 但 frontend 首页返回 `200` 并不代表其 Vite proxy 到 backend 的 `/api` / `/health` 已经可用
   - 因此首次代理登录有机会撞上 proxy 尚未稳定的窗口，表现为 frontend 返回 `500`

2. 本轮修复策略
   - 不改业务接口
   - 先收紧 smoke 脚本启动门槛
   - 在真正发起代理登录前，新增一段“frontend proxy `/health` 可用”的显式等待

本轮实际改动

1. 更新 `scripts/ci/compose-smoke.ps1`
   - 保留原有：
     - backend `/ready` 等待
     - frontend `/` 等待
   - 新增：
     - `Wait-UntilReady -Description "frontend proxy health"`
     - 通过 `GET http://127.0.0.1:<frontend-port>/health`
     - 断言：
       - `code == 0`
       - `database == up`
       - `agent == up`
   - 只有当 frontend 代理层也确认能成功打通 backend 后，才继续执行 `POST /api/v1/auth/login`

本轮修改文件

- `.claude/change-cache.md`
- `scripts/ci/compose-smoke.ps1`

本地验证

- 已再次用 PowerShell parser 对 `scripts/ci/compose-smoke.ps1` 做语法解析，未发现语法错误
- 当前本机仍无 `docker`，无法本地重放 compose smoke
- 因此这次修复是否生效，必须看 GitHub Actions 下一次 `compose-smoke` 真实跑数结果

commit摘要

- 计划提交：`fix(ci): wait for frontend proxy before smoke login`

希望接下来的 AI 做什么

1. 优先观察下一次 `compose-smoke` 结果。
2. 如果仍失败，请优先从 GitHub Actions 日志中核对：
   - frontend 容器日志
   - backend 容器日志
   - `/health` 与 `/api/v1/auth/login` 的代理返回体
3. 如果这次通过，再继续回到 `P2-1` 主线，补 proto contract tests 或更系统的 integration coverage。

by: gpt-5.4
