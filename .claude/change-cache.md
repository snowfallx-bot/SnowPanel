【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮按用户要求完成 `.claude/progress.md` 中剩余 P2 项并进行标记。

本轮实际改动

1. 完成并标记 `P2-1`
   - `.claude/progress.md` 已将 `P2-1` 标记为完成。
   - 记录当前测试矩阵已覆盖：
     - backend unit tests
     - backend + fake agent integration-style tests
     - proto-contract CI job
     - compose smoke
     - frontend Playwright e2e
     - frontend vitest
     - core-agent Rust unit/contract tests

2. 完成并标记 `P2-2`
   - 新增 `docs/observability.md`
   - 新增 `docs/observability.zh-CN.md`
   - README / README.zh-CN 文档导航已加入 Observability / 可观测性入口。
   - 文档覆盖：
     - `/health`
     - `/ready`
     - `/metrics`
     - backend HTTP Prometheus metrics
     - backend -> core-agent metrics
     - request id / access log
     - core-agent tracing logs
     - audit logs
     - 生产排障检查顺序

3. 完成并标记 `P2-3`
   - `backend/README.md` 移除 grpc transport placeholder 过时描述，改为真实 gRPC client 与 metrics 说明。
   - `docs/deployment.md` / `docs/deployment.zh-CN.md` 将 Compose Prototype / Compose 原型模式改为 Compose Local / Compose 本地模式。
   - `frontend/src/layouts/AppLayout.tsx` 副标题从 `Linux Panel Prototype` 改为 `Linux Server Operations`。
   - `frontend/e2e/fixtures.ts` 更新对应 e2e 断言。
   - `core-agent/src/process/systemd_service.rs` 移除未使用的 `tail_logs_placeholder`。

本轮修改文件

- `.claude/change-cache.md`
- `.claude/progress.md`
- `README.md`
- `README.zh-CN.md`
- `backend/README.md`
- `core-agent/src/process/systemd_service.rs`
- `docs/deployment.md`
- `docs/deployment.zh-CN.md`
- `docs/observability.md`
- `docs/observability.zh-CN.md`
- `frontend/e2e/fixtures.ts`
- `frontend/src/layouts/AppLayout.tsx`

本地验证

- `rg` 扫描已确认以下旧痕迹不存在：
  - `Linux Panel Prototype`
  - `grpc transport placeholder`
  - `tail_logs_placeholder`
  - `Compose Prototype`
  - `Compose 原型`
- `C:\Users\GuaiZai\.cargo\bin\cargo.exe fmt` 通过
- `C:\Users\GuaiZai\.cargo\bin\cargo.exe test` 通过
  - 25 个 core-agent Rust 单元测试全部通过
- `go test ./...` 在 `backend` 目录下通过
- `npm run test -- --run` 在 `frontend` 目录下通过
  - 24 个前端单元测试全部通过
- `npm run build` 在 `frontend` 目录下通过

备注

- 当前 shell 的 `PATH` 仍未包含 `C:\Users\GuaiZai\.cargo\bin`，Rust 命令继续使用完整路径运行。
- `cargo fmt` / `cargo test` 仍会输出 `warn: could not canonicalize path C:\Users\GuaiZai`，但命令成功，不影响测试结果。
- 默认 sandbox 下 `npm run test -- --run` / `npm run build` 因真实路径解析失败，需要提升权限后运行；提升权限后均已通过。

commit摘要

- 建议提交：`docs: mark progress complete`

希望接下来的 AI 做什么

1. 本轮改动应及时提交。
2. 当前 `.claude/progress.md` 内列出的 P0 / P1 / P2 项均已标记完成。
3. 后续新增工作建议另开 progress 项，例如 httpOnly cookie、OpenTelemetry tracing、service logs 查询等。

by: gpt-5.5-codex
