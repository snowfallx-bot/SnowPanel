【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮先完成一笔提交，然后继续推进 `P2-3` 的小步清理，优先收掉已经明确过时的 README 描述与 placeholder 痕迹。

本次核心判断

1. 上一笔 `P2-2` tracing 改动已经适合独立提交，不应继续堆在工作区。
2. 在当前环境缺少 `cargo` / `docker`、不适合继续做 tracing 实跑验证时，最划算的继续推进方式是清理 `P2-3` 里已经明确的过时文档和命名痕迹。

本轮实际改动

1. 先提交上一笔 tracing 工作
   - 已提交 commit：
     - `ce43f0d feat(observability): add otel tracing pipeline and jaeger baseline`

2. 清理过时 backend README
   - 更新 `backend/README.md`：
     - 删除 “gRPC Client Skeleton / transport placeholder” 过时段落。
     - 改成当前真实能力说明：auth、RBAC、真实 gRPC 调 core-agent、`/ready`、`/metrics`、observability 说明等。

3. 删除 core-agent 中的 placeholder 命名痕迹
   - 更新 `core-agent/src/process/systemd_service.rs`：
     - 删除未使用的 `tail_logs_placeholder()` 方法，避免继续保留原型占位语义。

4. 补齐仓库根 README 的 observability 入口
   - 更新 `README.md` / `README.zh-CN.md`：
     - 常用命令增加 `make up-observability` / `make up-host-agent-observability`。
     - 文档导航增加 observability 文档入口。

5. 重写 roadmap 文档
   - 更新 `docs/roadmap.md` / `docs/roadmap.zh-CN.md`：
     - 从 “monorepo skeleton / generated stubs” 初始化草案，
     - 改为“已完成基础能力 / 进行中事项 / 下一步优先级”的当前状态版。

6. 修正 architecture 文档中的 Redis 描述
   - 更新 `docs/architecture.md` / `docs/architecture.zh-CN.md`：
     - 不再写成“Redis 仅预留后续使用”，
     - 改为反映当前已用于共享登录失败限流状态。

7. 更新 development 文档中的测试矩阵与 observability 命令
   - 更新 `docs/development.md` / `docs/development.zh-CN.md`：
     - 增加 `make up-observability` / `make up-host-agent-observability`
     - 将测试覆盖说明改为当前真实 CI / test matrix

8. 同步 root README 中的 roadmap 标签
   - 更新 `README.md` / `README.zh-CN.md`：
     - 不再继续把 roadmap 标成“路线图草案”。

9. 补齐 observability 命令文档入口
   - 更新 `README.md` / `README.zh-CN.md`：
     - 增加 `down/logs` 相关 observability 命令，和 `Makefile` 对齐。
   - 更新 `docs/development.md` / `docs/development.zh-CN.md`：
     - 增加 `down/logs` 相关 observability 命令，避免只写 `up` 不写关闭与排障入口。

10. 统一 observability 文档术语
   - 更新 `docs/deployment.md` / `docs/deployment.zh-CN.md`：
     - “Prometheus UI” 调整为可观测性入口表达。
   - 更新 `docs/observability.md` / `docs/observability.zh-CN.md`：
     - “Prometheus Baseline Stack” 调整为 “Observability Baseline Stack / 可观测性基线栈”。

11. 同步进度缓存
   - 更新 `.claude/progress.md`：
     - 标记 `P2-3` 已从“未开始收口”推进到“进行中”。

本轮修改文件

- `.claude/change-cache.md`
- `.claude/progress.md`
- `README.md`
- `README.zh-CN.md`
- `backend/README.md`
- `docs/architecture.md`
- `docs/architecture.zh-CN.md`
- `docs/development.md`
- `docs/development.zh-CN.md`
- `docs/deployment.md`
- `docs/deployment.zh-CN.md`
- `docs/observability.md`
- `docs/observability.zh-CN.md`
- `docs/roadmap.md`
- `docs/roadmap.zh-CN.md`
- `core-agent/src/process/systemd_service.rs`

本地验证

- 已通过：
  - `rg` 复查：已不再存在 `backend/README.md` 中的 `transport placeholder` 文案
  - `rg` 复查：已不再存在 `tail_logs_placeholder`
- 未做 / 受环境限制：
  - `cargo` 相关验证（当前环境无 `cargo`）

commit摘要

- 已提交：`feat(observability): add otel tracing pipeline and jaeger baseline`
- 计划下一笔提交：`docs: refresh backend and observability readmes`
- 计划再下一笔提交：`docs: rewrite roadmap to reflect current project state`
- 计划再下一笔提交：`docs: align architecture docs with current redis usage`
- 计划再下一笔提交：`docs: refresh development guides for current test matrix`
- 计划再下一笔提交：`docs: remove stale roadmap draft labels`
- 计划再下一笔提交：`docs: align observability command docs with make targets`
- 计划再下一笔提交：`docs: normalize observability terminology across docs`

希望接下来的 AI 做什么

1. 在具备 `cargo` 的环境优先验证并修正 Rust 侧：
   - `cd core-agent && cargo fmt --check`
   - `cd core-agent && cargo check`
   - `cd core-agent && cargo test`
2. 在具备 Docker 的环境验证 tracing 闭环：
   - `make up-observability`
   - backend 发起一个会走 core-agent 的真实请求
   - 在 Jaeger UI 确认单条 trace 中同时出现 backend HTTP span、backend gRPC client span、core-agent gRPC server span
3. 继续 `P2-3`：
   - 扫描并更新其他仍落后的 README / docs 文案
   - 收敛少量历史原型措辞与重复说明
4. 若 tracing 基线验证通过，继续 `P2-2` 剩余项：
   - 真实通知渠道接入 Alertmanager
   - 告警去重 / 升级策略
   - SLO/SLI 阈值校准

by: gpt-5.5
