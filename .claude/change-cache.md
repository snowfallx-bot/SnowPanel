【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮继续推进“文件模块二进制 + 大文件下载闭环”并完成交付级联动。

本次核心完成项

1. 协议层：
   - 在 `proto/agent/v1/agent.proto` 新增 `ReadFileChunk` RPC。
   - 新增 `ReadFileChunkRequest(path, offset, limit, safety)`。
   - 新增 `ReadFileChunkResponse(error, path, offset, chunk, total_size, eof)`。
2. core-agent：
   - `core-agent/src/file/service.rs` 新增原始字节分块读取能力，保留路径安全校验。
   - 支持 offset/limit 边界处理、EOF 判定、空文件处理与错误映射。
   - `core-agent/src/api/grpc_server.rs` 挂载 `read_file_chunk` gRPC 处理。
3. backend：
   - `backend/internal/grpcclient/agent_client.go` 新增 `ReadFileChunk` client 方法与数据结构。
   - `backend/internal/service/file_service.go` 将下载改为分块流式（callback 写出），不再依赖 `ReadTextFile` 的 UTF-8/8MB 限制。
   - `backend/internal/api/handler/file_handler.go` 改为附件流式响应，首块自动探测 `Content-Type`，并保留下载审计。
   - 更新 DTO：`DownloadFileResult`。
4. frontend：
   - `frontend/src/api/files.ts` 下载 API 解析 `Content-Disposition` 文件名。
   - `frontend/src/pages/FilesPage.tsx` 优先使用后端附件名，自动兼容文本/二进制下载。
5. 文档：
   - `docs/api-design.md`、`docs/api-design.zh-CN.md` 更新下载链路说明为 chunk 流式方案。
6. 测试：
   - 覆盖 service 层分块下载成功/异常路径。
   - 覆盖 handler 层下载流式成功/失败路径。
   - 更新 backend+agent 集成测试到 `ReadFileChunk` 下载路径。
   - 本地验证通过：
     - `cd backend && go test ./...`
     - `cd frontend && npm run build`

本轮修改文件

- `proto/agent/v1/agent.proto`
- `core-agent/src/file/service.rs`
- `core-agent/src/api/grpc_server.rs`
- `backend/internal/grpcclient/agent_client.go`
- `backend/internal/grpcclient/pb/proto/agent/v1/agent.pb.go`
- `backend/internal/grpcclient/pb/proto/agent/v1/agent_grpc.pb.go`
- `backend/internal/service/file_service.go`
- `backend/internal/service/file_service_test.go`
- `backend/internal/service/agent_integration_test.go`
- `backend/internal/api/handler/file_handler.go`
- `backend/internal/api/handler/file_handler_test.go`
- `backend/internal/dto/file.go`
- `frontend/src/api/files.ts`
- `frontend/src/pages/FilesPage.tsx`
- `docs/api-design.md`
- `docs/api-design.zh-CN.md`
- `.gitignore`

commit摘要

待提交：
- `feat(files): stream binary-safe downloads via grpc chunk rpc`

希望接下来的 AI 做什么

继续推进文件模块剩余闭环（优先级从高到低）：
1. 上传链路改为二进制分块上传（当前仍 UTF-8 文本上传）。
2. 重命名从“读写拷贝”升级为 agent 侧原子 rename API。
3. 增加文件下载中断/超时/断点续传策略与对应测试。
4. 在 API 文档补充下载响应头与错误语义示例。

by: gpt-5
