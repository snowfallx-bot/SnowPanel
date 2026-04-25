【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮继续按“小步快提交”推进 `P2-3`，重点收口剩余原型措辞与文档命名遗留。

本轮实际改动

1. 清理前端应用壳中的原型文案
   - `frontend/src/layouts/AppLayout.tsx`
   - 将 header 文案从 `Linux Panel Prototype` 调整为 `SnowPanel Operations Console`。

2. 同步 e2e 登录成功后的页面锚点
   - `frontend/e2e/fixtures.ts`
   - 将 `shellMarker` 从 `/linux panel prototype/i` 调整为 `/snowpanel operations console/i`，避免继续依赖原型文案。

3. 清理 proto 文档中的原型期命名
   - `proto/README.md`
   - `Generate Go Stubs` / `Generate Rust Stubs` 统一为 `Bindings`。
   - Rust 生成说明改为当前真实流程：`core-agent` 通过 `build.rs`（`tonic-build` + vendored `protoc`）在构建时生成。

4. 同步路线图与接力进度文档
   - `.claude/progress.md`
   - `docs/roadmap.md`
   - `docs/roadmap.zh-CN.md`
   - 记录本轮“前端原型文案清理 + proto 命名统一”进展，并将 roadmap 中 `placeholder` 相关遗留措辞统一为更准确的“历史遗留措辞”表达。

本轮本地验证

1. 已执行：
   - `cd frontend && npm run build`（通过）
   - `rg -n "Linux Panel Prototype|linux panel prototype" frontend`（无结果）
   - `rg -n "placeholder" docs proto README.md README.zh-CN.md backend/README.md`（无结果）

2. 未通过/受环境影响：
   - `cd frontend && npm run test -- src/layouts/AppLayout.test.tsx`
   - 失败原因为依赖侧 `ERR_REQUIRE_ESM`（`html-encoding-sniffer` -> `@exodus/bytes`），与本轮改动无直接关联。

commit 摘要

- `ac97d99 chore: remove remaining prototype wording from app shell`
- `b767d41 docs: refresh roadmap wording for prototype-trace cleanup`

希望接下来的 AI 做什么

1. 在可执行环境继续推进 `P2-2` 实测收口：
   - `make up-observability` 或 `make up-host-agent-observability`
   - 触发一条 backend -> core-agent 请求
   - 在 Jaeger 确认单条 trace 同时包含 backend HTTP span / gRPC client span / core-agent gRPC server span

2. 继续 `P2-3` 扫描：
   - 重点检查非中文文档、脚本注释、测试基线里是否还存在历史命名或重复说明
   - 保持“小改动即提交”节奏

3. 若要恢复前端单测稳定性，可单开一笔依赖兼容修复：
   - 排查 Vitest + jsdom 依赖链上的 ESM/CJS 冲突
   - 独立提交，避免与功能/文案改动混在同一笔

by: gpt-5.5
