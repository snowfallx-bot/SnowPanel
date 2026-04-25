【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮继续按“及时提交（即便小改动）”推进，目标仍是 `P2-3` 持续收口，并补一笔可直接执行的 `P2-2` 文档化实测路径。

本轮实际改动

1. 清理前端应用壳原型文案并同步 e2e 锚点
   - `frontend/src/layouts/AppLayout.tsx`
     - `Linux Panel Prototype` -> `SnowPanel Operations Console`
   - `frontend/e2e/fixtures.ts`
     - 登录后页面锚点改为 `/snowpanel operations console/i`

2. 统一 proto 文档命名，去掉 `Stubs` 遗留
   - `proto/README.md`
   - `Generate Go/Rust Stubs` -> `Generate Go/Rust Bindings`
   - Rust 说明改为当前真实流程：`core-agent` 通过 `build.rs`（`tonic-build` + vendored `protoc`）在构建时生成绑定代码

3. 更新路线图措辞，继续收敛历史命名
   - `docs/roadmap.md`
   - `docs/roadmap.zh-CN.md`
   - 将 `placeholder` 类表达替换为更准确的历史遗留措辞，并写入“前端原型文案已清理”的最新进展

4. 增加 tracing 实测清单（P2-2 文档化推进）
   - `docs/observability.md`
   - `docs/observability.zh-CN.md`
   - 新增 “Tracing Validation Checklist / Tracing 实测清单”：
     - compose / host-agent 启动方式
     - 选择必经 core-agent 的接口（如 `/api/v1/dashboard/summary`）
     - 注入并核对 `X-Request-ID`
     - 在 Jaeger 确认同一 trace 同时包含 backend 与 core-agent spans
     - host-agent 模式下 OTEL 环境变量核对点

5. 同步接力进度
   - `.claude/progress.md`
   - 追加记录以上 4 类推进项

本轮本地验证

1. 已执行并通过：
   - `cd frontend && npm run build`
   - `rg` 复查原型/placeholder/stubs 文案在主要文档中的残留情况

2. 已执行但失败（环境/依赖问题）：
   - `cd frontend && npm run test -- src/layouts/AppLayout.test.tsx`
   - 失败为依赖侧 `ERR_REQUIRE_ESM`（`html-encoding-sniffer` -> `@exodus/bytes`），与本轮功能文案改动无直接关联

3. 环境能力检查：
   - `docker --version`：不可用（命令不存在）
   - `cargo --version`：不可用（命令不存在）
   - 因此本轮无法在本机完成 `P2-2` tracing 实跑验证

commit 摘要

- `ac97d99 chore: remove remaining prototype wording from app shell`
- `b767d41 docs: refresh roadmap wording for prototype-trace cleanup`
- `eb4c2fc chore: refresh change cache handoff notes`
- `c629372 docs: add tracing validation checklist for observability`

希望接下来的 AI 做什么

1. 在具备 Docker + cargo 的环境执行 `P2-2` 实跑闭环
   - `make up-observability` 或 `make up-host-agent-observability`
   - 发起至少一条会经过 core-agent 的真实请求
   - 在 Jaeger 核验 backend/client/server spans 同 trace

2. 继续 `P2-3` 小步清理
   - 重点查漏：非主文档、脚本注释、测试 fixtures 中的历史命名或重复描述
   - 维持“改完就提交”的节奏

3. 如需恢复本机前端单测可执行性，建议单独开任务处理
   - 排查 Vitest + jsdom 的 ESM/CJS 依赖链冲突
   - 与业务改动分离提交

by: gpt-5.5
