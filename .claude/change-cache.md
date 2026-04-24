【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮接手后，继续按 `.claude/progress.md` 推进 `P2-1`，重点不是再扩 smoke，而是把“proto 契约层”补上一道更靠前的测试与 CI 保护。

本次核心完成项

1. 新增 backend 侧 proto contract tests：
   - 新增 `backend/internal/grpcclient/agent_client_contract_test.go`
   - 这组测试直接复用生成后的 Go proto types 和真实 gRPC server/client 通路，不引入新工具链
   - 当前覆盖点：
     - `HealthService.Check`
     - `SystemService.GetRealtimeResource`
     - `FileService.ListFiles`
     - 结构化 `error.code/message/detail` 的 `AgentError` 映射
     - gRPC transport error -> `AgentError` 映射
     - 生成后的 Go proto descriptor 中关键 message / service 是否存在（`PathSafetyContext`、`HealthService`、`SystemService`、`FileService`）

2. 统一 Go proto 生成入口：
   - 修改 `Makefile`
   - 新增 `proto-go` 目标
   - 约定 Go stubs 统一生成到 backend 实际消费的位置：
     - `backend/internal/grpcclient/pb/proto/agent/v1/agent.pb.go`
     - `backend/internal/grpcclient/pb/proto/agent/v1/agent_grpc.pb.go`

3. 修正文档中的 proto 生成说明：
   - 修改 `proto/README.md`
   - 不再让文档误导到“生成到 proto 目录旁边”
   - 文档现在优先指向 `make proto-go`，并给出与实际产物路径一致的 raw `protoc` 命令

4. 把 proto 契约检查接入 CI：
   - 修改 `.github/workflows/ci.yml`
   - 新增 `proto-contract` job
   - job 会：
     - 安装 `protoc`
     - 安装 `protoc-gen-go` / `protoc-gen-go-grpc`
     - 执行 `make proto-go`
     - 用 `git diff --exit-code` 校验生成产物是否已同步提交
   - `compose-smoke` 现在依赖：
     - `backend`
     - `core-agent`
     - `frontend`
     - `proto-contract`

本轮修改文件

- `.claude/change-cache.md`
- `.github/workflows/ci.yml`
- `Makefile`
- `proto/README.md`
- `backend/internal/grpcclient/agent_client_contract_test.go`

本地验证

1. 已通过：
   - `cd backend && go test ./...`
   - `go test ./internal/grpcclient -run 'Proto|Health|ListFiles' -v`

2. 当前环境限制：
   - 本机缺少 `protoc`，`protoc --version` 返回 command not found
   - 因此 `make proto-go` 无法在本机直接执行
   - 这部分真实验证将依赖新加的 GitHub Actions `proto-contract` job

3. 当前本地 diff 现状：
   - `git diff --stat` 显示改动集中在：
     - `.github/workflows/ci.yml`
     - `Makefile`
     - `proto/README.md`
   - 新增的 `backend/internal/grpcclient/agent_client_contract_test.go` 已参与并通过 backend 测试

commit摘要

- 计划提交：`test(proto): add contract coverage and generated stub check`

希望接下来的 AI 做什么

1. 优先观察 GitHub Actions 上新增的 `proto-contract` 首次运行结果：
   - 重点看 `arduino/setup-protoc@v3`
   - 重点看 `make proto-go` 在 Ubuntu runner 上是否能正确生成到目标路径
   - 重点看 `git diff --exit-code` 是否暴露出 repo 内现有 pb.go 与 proto 的偏差

2. 如果 CI 暴露出生成差异：
   - 不要先改 schema
   - 先把生成产物重新生成并提交，确认只是产物漂移还是工具版本差异

3. 如果 `proto-contract` 跑通，下一步建议继续补 `P2-1`：
   - 更系统的 backend + core-agent + postgres integration 覆盖
   - frontend e2e（登录 / 权限隐藏 / 文件浏览）
   - 然后再考虑是否需要引入更重的 proto 规则工具（如 Buf），而不是现在就上

by: claude-sonnet-4-6
