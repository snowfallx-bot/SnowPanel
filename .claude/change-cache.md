【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮继续按“小步快提交”推进，核心是把 observability 验证链路进一步工程化，并收口 CI/文档一致性。

本轮实际改动

1. `full-smoke.ps1` 支持自动登录模式
   - 文件：`scripts/observability/full-smoke.ps1`
   - 新增参数集：
     - token 模式：`-AccessToken`
     - 登录模式：`-LoginUsername/-LoginPassword`
   - 自动登录后可直接串行执行 tracing + alertmanager 验证，减少手工取 token 成本。

2. 新增 CI 端到端 observability 冒烟脚本
   - 文件：`scripts/ci/observability-smoke.ps1`
   - 能力：
     - 拉起 compose + observability 栈
     - 等待 backend/jaeger/alertmanager 就绪
     - 完成 bootstrap admin 改密并拿 token
     - 调用 `scripts/observability/full-smoke.ps1`
   - 增加 `docker` 前置检查，缺失时直接明确失败。

3. workflow 结构优化（避免影响默认 CI）
   - `ci.yml` 回归 push/PR 主流水线职责
   - 新增独立手动 workflow：`.github/workflows/observability-smoke.yml`
   - 通过 `workflow_dispatch` 按需触发 observability 冒烟，不阻塞默认 PR 路径。

4. 术语与文档入口收口
   - `ci.yml` 中 `Proto Stubs` 步骤命名改为 `Proto Bindings`
   - root README（中英文）补充 `Observability Smoke` 手动 workflow 入口
   - development/observability/scripts 文档同步 `full-smoke` 自动登录与 CI 脚本入口
   - 新增 `scripts/ci/README.md`，汇总 CI 脚本职责与入口

5. 进度文档同步
   - `.claude/progress.md` 已同步上述进展

本轮本地验证

1. 脚本执行检查：
   - `full-smoke.ps1` 登录模式可正常进入登录流程（不可达地址下按预期网络失败）
   - `scripts/ci/observability-smoke.ps1` 在无 docker 环境下会给出清晰前置错误并退出

2. 环境限制：
   - 当前环境依然没有 `docker` / `cargo`，无法完成真实在线 Jaeger/Alertmanager 验证

commit 摘要

- `b026ae9 feat(observability): support full-smoke login mode`
- `fe712f1 feat(ci): add observability smoke compose runner`
- `c5e6008 ci: add manual observability smoke workflow`
- `4631054 ci: split observability smoke into dedicated manual workflow`
- `7b88619 chore(ci): rename proto stubs steps to bindings`
- `180af40 docs: add observability workflow entry to root readmes`
- `00bdf6b docs(ci): add script index and development cross-links`

希望接下来的 AI 做什么

1. 在具备 Docker + cargo 的环境执行真实验收
   - 手动触发 GitHub Actions `Observability Smoke`
   - 或本地跑 `pwsh -File ./scripts/ci/observability-smoke.ps1`
   - 记录 Jaeger trace 与 Alertmanager 验证结果

2. 继续 `P2-2` 收口
   - 接入真实通知渠道（webhook/email/slack/wechat）
   - 校准告警 dedup/escalation 与 SLO/SLI 阈值

3. 持续 `P2-3`
   - 扫描非主文档、脚本注释、测试 fixture 的历史措辞和重复描述
   - 继续小改即提交

by: gpt-5.5
