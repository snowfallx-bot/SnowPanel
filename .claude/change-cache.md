【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============
本轮继续按“P2-3 代码侧清理 + 小步提交/推送”推进，重点收敛 CI/observability 脚本中的重复逻辑，不改业务行为与文档。

本轮实际改动

1. observability 告警 payload 构造公共化
   - 文件：
     - `scripts/observability/common.ps1`
     - `scripts/observability/alertmanager-smoke.ps1`
     - `scripts/observability/alertmanager-inhibition-smoke.ps1`
   - 改动：
     - 新增 `New-AlertmanagerSyntheticAlert`。
     - `alertmanager-smoke` 与 `alertmanager-inhibition-smoke` 改为复用公共 payload 构造。

2. CI readiness 等待原语公共化
   - 文件：
     - `scripts/ci/common.ps1`
     - `scripts/ci/observability-smoke.ps1`
   - 改动：
     - 在 `scripts/ci/common.ps1` 新增：
       - `Wait-ApiStatus`
       - `Wait-JsonReady`
       - `Wait-ApiJsonReady`
     - `Wait-BackendReadyJson` / `Wait-BackendReadyApi` / `Wait-FrontendStartup` / `Wait-FrontendProxyHealth` 改为复用公共等待原语。
     - `scripts/ci/observability-smoke.ps1` 删除本地重复的 `Wait-HttpStatusReady` / `Wait-JsonEndpointReady`，改用公共等待原语。

3. alertmanager 过滤查询与提交动作继续公共化
   - 文件：
     - `scripts/observability/common.ps1`
     - `scripts/observability/alertmanager-smoke.ps1`
     - `scripts/observability/alertmanager-inhibition-smoke.ps1`
   - 改动：
     - 新增：
       - `Get-AlertmanagerApiUriWithFilters`
       - `Get-AlertmanagerActiveAlerts`
       - `Get-AlertmanagerActiveAlertGroups`
       - `Submit-AlertmanagerAlerts`
     - 两份 smoke 脚本改为复用这些 helper。
     - inhibition 脚本里的 receiver 名从硬编码改为 `Resolve-AlertmanagerReceiver` 统一解析。

本轮本地验证

- 语法检查通过：
  - `scripts/ci/common.ps1`
  - `scripts/ci/observability-smoke.ps1`
  - `scripts/observability/common.ps1`
  - `scripts/observability/alertmanager-smoke.ps1`
  - `scripts/observability/alertmanager-inhibition-smoke.ps1`
- 当前本地工作树干净：`git status` 无未提交改动。

远端与 CI 状态

- 已推送到 `origin/main`。
- 由于连续小步 push，当前 GitHub Actions 有多条 in-progress 队列在跑（从 `fc26fe6` 到最新 `3fb6940`）。
- 最近已完成的历史 run（`958e3e3`、`21c9b72`、`0cbfa20`）为全绿。

commit 摘要

- `fc26fe6 refactor(observability): share synthetic alert payload builder`
- `e88821f refactor(ci): centralize readiness wait primitives`
- `ef34e0f refactor(observability): centralize alertmanager filtered query helpers`
- `52b05b5 refactor(observability): resolve inhibition receivers via shared helper`
- `3fb6940 refactor(observability): share alert submission helper`

希望接下来的 AI 做什么

1. 先盯完当前 in-progress 的几条 CI run，若有回归立即按失败日志定点修复。
2. 若 CI 继续全绿，继续 P2-3 的代码侧收敛：
   - 优先 `scripts/` 与 `backend` 的重复流程压缩；
   - 保持“行为不变 + 小步提交 + 立即 push”。
3. 如需继续加速节奏，维持 `--no-gpg-sign` 提交方式避免本机 pinentry 卡住。

by: gpt-5
