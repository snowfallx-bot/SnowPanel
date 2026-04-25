【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮继续按“及时提交（即便小改动）”推进，优先收口 `P2-3` 文案遗留，并补充 `P2-2` 的可执行实测文档路径。

本轮实际改动

1. 清理前端原型文案残留
   - `frontend/src/layouts/AppLayout.tsx`
     - `Linux Panel Prototype` -> `SnowPanel Operations Console`
   - `frontend/e2e/fixtures.ts`
     - 登录成功锚点同步为 `/snowpanel operations console/i`

2. 统一 proto 文档命名
   - `proto/README.md`
   - `Generate ... Stubs` 全部统一为 `Generate ... Bindings`
   - Rust 说明更新为当前真实流程（`build.rs` + `tonic-build` + vendored `protoc`）

3. 收敛 roadmap 历史措辞
   - `docs/roadmap.md`
   - `docs/roadmap.zh-CN.md`
   - 将 `placeholder` 类表述替换为更准确的“历史遗留措辞”，并补入本轮前端文案清理进展

4. 新增 tracing 实测清单（文档化推进 P2-2）
   - `docs/observability.md`
   - `docs/observability.zh-CN.md`
   - 增加 compose / host-agent 两种模式的最小验证步骤：
     - 启动方式
     - 调用经过 core-agent 的接口（示例 `/api/v1/dashboard/summary`）
     - 注入并校验 `X-Request-ID`
     - 在 Jaeger 确认 backend + core-agent spans 同 trace

5. 明确前端 Node 版本门槛并加本地防护
   - `docs/development.md`
   - `docs/development.zh-CN.md`
   - `frontend/README.md`
   - `frontend/package.json`
   - 明确 Node 最低版本 `>=20.19.0`（推荐 22+），并通过 `engines.node` 让不兼容环境尽早告警

6. 同步接力进度
   - `.claude/progress.md`
   - 记录上述文档/前端清理进展

本轮本地验证

1. 已通过：
   - `cd frontend && npm run build`
   - `rg` 扫描主要文档中的 `prototype/placeholder/stubs` 遗留词

2. 已执行但失败（环境兼容）：
   - `cd frontend && npm run test -- src/layouts/AppLayout.test.tsx`
   - 当前失败为工具链启动阶段依赖兼容问题（与 Node `20.18.0` 偏低相关），非本轮业务改动引入

3. 环境能力检查：
   - `docker` 不可用
   - `cargo` 不可用
   - 因此无法在本机完成 `P2-2` tracing 实跑闭环

commit 摘要

- `ac97d99 chore: remove remaining prototype wording from app shell`
- `b767d41 docs: refresh roadmap wording for prototype-trace cleanup`
- `eb4c2fc chore: refresh change cache handoff notes`
- `c629372 docs: add tracing validation checklist for observability`
- `17768da chore: refresh change cache with latest handoff`
- `b0988de docs: clarify frontend node minimum and add engine guard`

希望接下来的 AI 做什么

1. 在具备 Docker + cargo 的环境优先收口 `P2-2`
   - `make up-observability` / `make up-host-agent-observability`
   - 触发一条 backend -> core-agent 请求
   - 在 Jaeger 确认跨服务 trace 串联

2. 继续 `P2-3` 的小步收口
   - 重点扫描非主文档、测试 fixture、脚本注释中的历史命名或重复描述
   - 保持“改完即提交”

3. 若要修复本机前端单测启动问题，建议单独开任务
   - 统一 Node 版本到 `>=20.19.0` 或 22+
   - 再验证 `vitest` / `jsdom` 依赖链启动稳定性

by: gpt-5.5
