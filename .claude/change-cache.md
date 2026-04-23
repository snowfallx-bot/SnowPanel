【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮继续推进了任务系统，重点是“执行可靠性 + 可观测筛选”。

本次核心完成项

1. 修复任务取消竞态（已提交）：
   - commit: `e6f9faf`
   - 摘要: `fix(tasks): prevent canceled tasks from being overwritten by worker`
   - 结果：任务被取消后，不再被 worker 写回 `running/failed/success`。
2. 增强任务列表筛选（本轮待提交）：
   - backend `/api/v1/tasks` 新增可选查询参数：`status`、`type`。
   - repository 层支持按状态/类型过滤并保持分页语义。
   - frontend Tasks 页新增 Status/Type 下拉筛选器，筛选变化时自动回到第 1 页。
3. 补充测试：
   - 新增任务并发回归测试，覆盖“运行中取消后仍保持 canceled”。
   - 新增任务列表筛选测试，覆盖 status/type 组合过滤。
4. 文档同步：
   - `docs/api-design.md` 与 `docs/api-design.zh-CN.md` 已更新 `/tasks` 筛选参数说明。

本轮修改文件

backend/internal/dto/task.go
backend/internal/repository/task_repository.go
backend/internal/service/task_service.go
backend/internal/service/task_service_test.go
docs/api-design.md
docs/api-design.zh-CN.md
frontend/src/api/tasks.ts
frontend/src/pages/TasksPage.tsx

验证结果

1. `cd backend && go test ./...` 通过。
2. `cd frontend && npm run build` 通过。

commit摘要

已提交：`fix(tasks): prevent canceled tasks from being overwritten by worker`（`e6f9faf`）
待提交：`feat(tasks): support status/type filters in task list`

希望接下来的 AI 做什么

优先继续文件模块“二进制 + 大文件下载”闭环：
- 在 proto 增加文件分块读取 RPC（offset/limit/eof/total_size）。
- core-agent 支持原始字节读取并保留路径安全校验。
- backend `/api/v1/files/download` 改为流式透传，不再受 UTF-8/8MB 限制。
- 前端下载统一走后端附件响应，自动兼容文本与二进制。

by: gpt-5