【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮继续推进文件传输链路，完成“下载断点续传后端 Range 支持 + 前端分段重试/续传接入”。

本次核心完成项

1. backend（Go）：
   - `backend/internal/dto/file.go`：为下载请求增加 `offset` / `limit`，为结果增加 `start_offset` / `end_offset`
   - `backend/internal/service/file_service.go`：支持从指定 offset 开始读取，并按 limit 截断返回区间
   - `backend/internal/api/handler/file_handler.go`：
     - 解析 `Range: bytes=<offset>-`
     - 输出 `206 Partial Content`
     - 返回 `Accept-Ranges: bytes` 与 `Content-Range`
     - 下载审计摘要增加 `offset` / `limit`
2. backend 测试：
   - `backend/internal/service/file_service_test.go`：新增 offset + limit 的下载用例
   - `backend/internal/api/handler/file_handler_test.go`：新增 partial download 的 `206` / `Content-Range` 断言
   - 已通过：`cd backend && go test ./internal/service ./internal/api/handler`
3. frontend（React/TS）：
   - `frontend/src/api/files.ts`：
     - 下载改为按 1MB 分段拉取
     - 每段最多重试 3 次
     - 通过 `offset + limit` 查询参数和 `Range` 请求头续传
     - 基于 `Content-Range` 识别总大小并拼接最终 Blob
4. 文档：
   - `docs/api-design.md`、`docs/api-design.zh-CN.md`：补充 `/files/download` 的 Range/206/Content-Range 语义与前端分段重试说明

本轮修改文件

- `backend/internal/api/handler/file_handler.go`
- `backend/internal/api/handler/file_handler_test.go`
- `backend/internal/dto/file.go`
- `backend/internal/service/file_service.go`
- `backend/internal/service/file_service_test.go`
- `frontend/src/api/files.ts`
- `docs/api-design.md`
- `docs/api-design.zh-CN.md`

本地验证

- `cd backend && go test ./internal/service ./internal/api/handler` ✅
- `npm --prefix frontend run build` ✅
- 本机未执行 Rust 编译/测试

commit摘要

已提交并推送：
- `feat(files): add resumable download range handling`

待提交：
- `feat(files): add frontend segmented download retry`

希望接下来的 AI 做什么

1. 提交并推送本轮前端下载分段重试与文档改动。
2. 如继续增强文件链路，可为上传/下载补更细的错误响应示例与前端进度展示。
3. 若要继续完善下载能力，可评估多段 Range、校验 ETag/Last-Modified 等更完整的恢复语义。

by: claude-sonnet-4-6
