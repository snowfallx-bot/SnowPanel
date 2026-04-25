【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮继续按“小步快提交”推进，主线仍是 `P2-2` 可观测性实跑能力落地，并同步 `P2-3` 文档一致性。

本轮实际改动

1. observability 一键脚本能力继续增强
   - `scripts/observability/full-smoke.ps1`
   - 新增双模式：
     - `-AccessToken` 直传 token 模式
     - `-LoginUsername/-LoginPassword` 自动登录取 token 模式
   - 自动登录模式下若用户 `must_change_password=true` 会明确失败并给出指引，避免误判。

2. 新增 CI 端到端 observability 冒烟脚本
   - `scripts/ci/observability-smoke.ps1`（新增）
   - 在 Docker 环境中自动执行：
     - 拉起 compose + observability 栈（`docker-compose.yml` + `docker-compose.observability.yml`）
     - backend readiness / jaeger / alertmanager 等待检查
     - bootstrap admin 登录与改密
     - 调用 `scripts/observability/full-smoke.ps1` 做 tracing + alertmanager 一体化校验
   - 增加 `docker` 前置检查，缺失时直接给出清晰错误。

3. GitHub Actions 手动入口接入
   - `.github/workflows/ci.yml`
   - 新增 `workflow_dispatch` 输入 `run_observability_smoke`
   - 新增 `observability-smoke` 任务（仅手动触发时运行），避免阻塞默认 PR 流水线。

4. 文档入口全面同步
   - `scripts/observability/README.md`
   - `docs/observability.md` / `docs/observability.zh-CN.md`
   - `docs/development.md` / `docs/development.zh-CN.md`
   - `README.md` / `README.zh-CN.md`
   - 补齐 `full-smoke` 自动登录模式、`scripts/ci/observability-smoke.ps1`、ExecutionPolicy 提示等入口。

5. 进度文档同步
   - `.claude/progress.md`
   - 已记录 observability 脚本链路与手动 workflow 状态。

本轮本地验证

1. 已执行脚本级检查（当前机器无 docker）：
   - `full-smoke.ps1` 自动登录模式可正常进入登录流程（不可达地址下按预期网络失败）
   - `scripts/ci/observability-smoke.ps1` 可正常执行前置检查，并在缺少 docker 时明确报错退出

2. 环境限制：
   - 当前环境仍缺 `docker` / `cargo`，无法完成真实 Jaeger/Alertmanager 在线实跑验收

commit 摘要

- `fe712f1 feat(ci): add observability smoke compose runner`
- `c5e6008 ci: add manual observability smoke workflow`
- `b026ae9 feat(observability): support full-smoke login mode`
- （同轮前序）`ae05931 feat(observability): add one-shot full smoke runner`
- （同轮前序）`2e43703 docs: add observability full-smoke command to root readmes`
- （同轮前序）`a3111e5 docs: update roadmap with observability smoke tooling`
- （同轮前序）`1ae67cf chore: refresh change cache after full-smoke rollout`

希望接下来的 AI 做什么

1. 在具备 Docker + cargo 的环境跑通手动 workflow：
   - GitHub Actions 手动触发 `CI`，启用 `run_observability_smoke=true`
   - 收集 Jaeger trace ID 与 Alertmanager 告警验证结果

2. 在验证结果基础上继续收口 `P2-2`：
   - 接入真实通知渠道（webhook/email/slack/wechat）
   - 校准 dedup/escalation 与 SLO/SLI 阈值

3. 持续推进 `P2-3`：
   - 继续扫描非主文档/脚本注释/fixture 的历史措辞与重复说明
   - 保持“改完即提交”

by: gpt-5.5
