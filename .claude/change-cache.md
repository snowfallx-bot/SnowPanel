【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮继续按“小步快提交”推进，重点是 `P2-3` 文档一致性收口 + `P2-2` 告警落地文档补强。

本轮实际改动

1. frontend 测试链路增加 Node 版本 preflight
   - `frontend/scripts/check-node-version.mjs`（新增）
   - `frontend/package.json`
   - `frontend/README.md`
   - 当 Node `<20.19.0` 时，`npm run test` / `npm run test:e2e` 会先给出清晰错误并提前失败，避免此前 vitest 启动期依赖报错噪音。

2. root README 中英文 Node 版本口径对齐
   - `README.md`
   - `README.zh-CN.md`
   - 统一为“Node 22+ 推荐，前端工具链最低 20.19.0”。

3. observability 文档补充 Alertmanager 落地清单
   - `docs/observability.md`
   - `docs/observability.zh-CN.md`
   - 新增从 no-op 接收器切换到真实通知渠道的执行 checklist（路由归属、抑制规则、发布校验、可控告警验证、阈值回滚）。

4. 进度文档同步
   - `.claude/progress.md`
   - 记录以上三类推进项。

本轮本地验证

1. 已执行：
   - `cd frontend && npm run test -- src/layouts/AppLayout.test.tsx`

2. 结果：
   - 在当前环境（Node `20.18.0`）下，测试会被 `check:node` 以明确提示提前拦截：
     - 需要 Node `20.19.0+`（推荐 22+）
   - 行为符合预期（相比之前可读性明显更好）。

commit 摘要

- `569256a chore: add frontend test node-version preflight`
- `0dd405f docs: align root node version requirements`
- `bfcecec docs: add alertmanager rollout checklist`

希望接下来的 AI 做什么

1. 在具备 Docker + cargo 的环境收口 `P2-2`
   - 执行 tracing 闭环实测（compose / host-agent）
   - 接入真实 Alertmanager 通知渠道并完成一次可控告警验证

2. 持续推进 `P2-3`
   - 继续扫描非主文档、脚本注释、测试 fixture 的历史措辞与重复说明
   - 保持“小改即提交”

3. 若要恢复本机 frontend test 运行
   - 升级 Node 到 `>=20.19.0`（推荐 22+）
   - 之后重新跑 `npm run test` / `npm run test:e2e`

by: gpt-5.5
