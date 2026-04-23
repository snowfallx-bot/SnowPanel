【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮继续推进文件模块主链路，完成“重命名从文本拷贝改为 agent 原子 rename RPC”。

本次核心完成项

1. 协议层：
   - 在 `proto/agent/v1/agent.proto` 新增：
     - `RenameFileRequest(source_path, target_path, safety)`
     - `RenameFileResponse(error, source_path, target_path, moved_bytes)`
     - `FileService.RenameFile` RPC
2. core-agent（Rust）：
   - `core-agent/src/file/service.rs` 新增 `rename_file`：
     - 路径安全校验（source/target 都走 safe-root）
     - same-path 拒绝
     - 仅允许 source 为文件
     - target 已存在拒绝
     - target 父目录不存在拒绝
     - 使用 `fs::rename` 执行原子重命名
   - `core-agent/src/api/grpc_server.rs` 增加 `rename_file` gRPC handler
   - `core-agent/src/security/path_validator.rs` 增加 `FileOperation::Move`，并纳入危险路径拦截集合
3. backend（Go）：
   - `backend/internal/grpcclient/agent_client.go` 新增 `RenameFile` request/result 与 client 调用
   - `backend/internal/service/file_service.go` 删除旧的“list+read+write+delete”重命名流程，改为直连 `agentClient.RenameFile`
4. 测试：
   - `backend/internal/service/file_service_test.go` 重写 rename 用例断言（改为 RPC 调用语义）
   - `backend/internal/service/agent_integration_test.go` 增加 rename 集成测试，并扩展 fake gRPC service
5. 文档：
   - `docs/api-design.md`、`docs/api-design.zh-CN.md` 更新 `/files/rename` 为原子 rename 说明
6. 代码生成：
   - 重新生成：
     - `backend/internal/grpcclient/pb/proto/agent/v1/agent.pb.go`
     - `backend/internal/grpcclient/pb/proto/agent/v1/agent_grpc.pb.go`

本轮修改文件

- `proto/agent/v1/agent.proto`
- `core-agent/src/file/service.rs`
- `core-agent/src/api/grpc_server.rs`
- `core-agent/src/security/path_validator.rs`
- `backend/internal/grpcclient/agent_client.go`
- `backend/internal/grpcclient/pb/proto/agent/v1/agent.pb.go`
- `backend/internal/grpcclient/pb/proto/agent/v1/agent_grpc.pb.go`
- `backend/internal/service/file_service.go`
- `backend/internal/service/file_service_test.go`
- `backend/internal/service/agent_integration_test.go`
- `docs/api-design.md`
- `docs/api-design.zh-CN.md`

本地验证

- `cd backend && go test ./...` ✅
- `cd frontend && npm run build` ✅
- 本机无 `cargo`，未执行 Rust 本地编译/测试；Rust 侧由 CI 继续校验。

commit摘要

待提交：
- `feat(files): switch rename to atomic grpc rpc`

希望接下来的 AI 做什么

1. 文件链路下一优先级：下载断点续传/重试策略（后端与前端联动）。
2. 上传链路补断点续传（offset 校验 + 前端 retry 设计）。
3. 文档补充 upload/download/rename 的错误语义与示例响应。

by: gpt-5.4
