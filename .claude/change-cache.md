【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮继续推进“文件模块上传链路”并完成二进制分块上传闭环。

本次核心完成项

1. 协议层：
   - 在 `proto/agent/v1/agent.proto` 新增 `WriteFileChunk` RPC。
   - 新增 `WriteFileChunkRequest(path, offset, chunk, create_if_not_exists, truncate, safety)`。
   - 新增 `WriteFileChunkResponse(error, path, offset, written_bytes, total_size)`。
2. core-agent：
   - `core-agent/src/file/service.rs` 新增 `write_file_chunk` 原始字节分块写入。
   - 保留路径安全校验，增加 offset 越界、chunk 超限、父目录存在性校验。
   - `core-agent/src/api/grpc_server.rs` 挂载 `write_file_chunk` gRPC 处理。
3. backend：
   - `backend/internal/grpcclient/agent_client.go` 新增 `WriteFileChunk` client 方法与模型。
   - `backend/internal/service/file_service.go` 新增 `UploadFile`，将 HTTP 上传流按 chunk 透传到 agent。
   - 新增 `dto.UploadFileRequest/Result`。
   - `backend/internal/api/handler/file_handler.go` 新增 `UploadFile` handler（multipart/form-data：`path` + `file`）并接入审计。
   - `backend/internal/api/router.go` 增加 `POST /api/v1/files/upload`（`files.write`）。
4. frontend：
   - `frontend/src/api/files.ts` 新增 `uploadFile(file, path)`，使用 `FormData` 直传二进制。
   - `frontend/src/pages/FilesPage.tsx` 上传改为通用文件上传，不再做 UTF-8 解码限制。
   - `frontend/src/types/file.ts` 新增上传返回类型。
5. 文档：
   - `docs/api-design.md`、`docs/api-design.zh-CN.md` 新增 `/files/upload`，并更新分块上传说明。
6. 测试：
   - `backend/internal/service/file_service_test.go` 新增上传成功/空文件/偏移异常/读取异常用例。
   - `backend/internal/api/handler/file_handler_test.go` 新增上传成功与缺失文件校验用例。
   - `backend/internal/service/agent_integration_test.go` 新增上传集成用例并扩展 fake gRPC 服务。
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
- `backend/internal/api/router.go`
- `backend/internal/dto/file.go`
- `frontend/src/api/files.ts`
- `frontend/src/pages/FilesPage.tsx`
- `frontend/src/types/file.ts`
- `docs/api-design.md`
- `docs/api-design.zh-CN.md`

commit摘要

待提交：
- `feat(files): add binary chunked upload pipeline via grpc`

希望接下来的 AI 做什么

继续推进文件模块剩余闭环（优先级从高到低）：
1. 重命名从“读写拷贝”升级为 agent 侧原子 rename API。
2. 下载链路增加中断/超时/断点续传策略与测试。
3. 上传链路支持断点续传（offset 校验 + 前端重试策略）。
4. 在 API 文档补充上传/下载错误语义与示例。

by: gpt-5
