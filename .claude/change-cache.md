【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮继续按“小步快提交”推进，核心是把 `P2-2` 从“有文档”再推进到“有可执行脚本工具链”。

本轮实际改动

1. 新增一键 observability 冒烟脚本
   - `scripts/observability/full-smoke.ps1`
   - 串行执行：
     - `trace-smoke.ps1`（backend/core-agent trace 串联验证）
     - `alertmanager-smoke.ps1`（Alertmanager 合成告警注入验证）
   - 支持统一传入 `AccessToken`、backend/jaeger/alertmanager base URL、超时参数等。

2. 脚本文档与执行策略补充
   - `scripts/observability/README.md`
   - 增加 `full-smoke.ps1` 用法和参数说明
   - 增加 Windows `ExecutionPolicy` 受限时的运行方式（`-ExecutionPolicy Bypass`）

3. observability / development 文档入口对齐
   - `docs/observability.md`
   - `docs/observability.zh-CN.md`
   - `docs/development.md`
   - `docs/development.zh-CN.md`
   - 补充 `full-smoke` 入口与 one-shot 说明，保持脚本与文档双向可达。

4. root README 可发现性补齐
   - `README.md`
   - `README.zh-CN.md`
   - 常用命令区增加 `full-smoke` 命令入口。

5. roadmap 状态同步
   - `docs/roadmap.md`
   - `docs/roadmap.zh-CN.md`
   - `P2-2` 进展明确写入 observability 冒烟脚本能力（trace / alertmanager / full）。

6. 接力文档同步
   - `.claude/progress.md`
   - 记录本轮新增脚本与文档入口同步进展。

本轮本地验证

1. 轻量执行检查：
   - `trace-smoke.ps1`、`alertmanager-smoke.ps1`、`full-smoke.ps1` 均可正常启动并进入请求流程。
   - 在不可达地址下按预期报网络不可达错误（用于本机无 Docker 环境下的语法/流程检查）。

2. 环境限制：
   - 当前环境仍缺 `docker` / `cargo`，无法执行真实 Jaeger/Alertmanager 在线验收。

commit 摘要

- `ae05931 feat(observability): add one-shot full smoke runner`
- `2e43703 docs: add observability full-smoke command to root readmes`
- `a3111e5 docs: update roadmap with observability smoke tooling`

希望接下来的 AI 做什么

1. 在具备 Docker + cargo 的环境执行真实 `P2-2` 实跑闭环：
   - `make up-observability` 或 `make up-host-agent-observability`
   - `pwsh -File ./scripts/observability/full-smoke.ps1 -AccessToken "<access_token>"`
   - 在 Jaeger / Alertmanager UI 复核与留证

2. 继续 `P2-3` 小步收口：
   - 扫描非主文档、脚本注释、测试 fixture 的历史措辞或重复说明
   - 继续“改完即提交”

by: gpt-5.5
