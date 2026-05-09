# 路线图

语言: [English](roadmap.md) | **简体中文**

这份路线图描述的是仓库当前真实状态，而不是最早的初始化计划。

## 已完成的基础能力

- backend 与 core-agent 的真实 gRPC 主链路已经覆盖 dashboard、files、services、Docker、cron
- 宿主机 Agent 运行模式已落地，包含 systemd 模板与 Ubuntu 一键安装
- secrets、bootstrap admin、内部端口暴露、cron allowlist 等安全基线已收口
- 基于 RBAC 的认证/会话模型已落地，前端也已具备权限感知
- Docker / service restart 已接入真实异步任务执行
- 文件能力已扩展到更接近真实运维场景
- CI 已形成分层覆盖：backend tests、compose smoke、backend integration、frontend e2e

## P2 已完成项

### P2-2 生产化观测能力

- Prometheus 指标与基线告警规则已具备
- Alertmanager 基线路由已具备
- OTEL tracing 基线现已具备：
  - backend HTTP spans
  - backend gRPC client spans
  - core-agent gRPC server spans
  - OTel Collector -> Jaeger 管线
- Prometheus/Alertmanager 的 SLO 基线已扩展：
  - backend 可用性 recording rules 与 warning/critical 分级告警
  - core-agent gRPC 错误率 recording rule 与 warning/critical 分级告警
  - Alertmanager warning/critical 双接收器路由基线
- observability 冒烟脚本现已具备：
  - Jaeger 跨服务 trace 校验（`scripts/observability/trace-smoke.ps1`）
  - Alertmanager 合成告警注入校验（`scripts/observability/alertmanager-smoke.ps1`）
  - tracing + alertmanager 一键串行校验（`scripts/observability/full-smoke.ps1`）
- observability 配置校验闸门现已具备：
  - `scripts/observability/validate-config.ps1`（`promtool`/`amtool` 检查，默认 Docker 且支持本地回退 + `promtool test rules`）
  - `ci.yml` 中的 `observability-config` 任务
- Alertmanager 生产落地辅助能力已补齐：
  - 生产接收器模板：`deploy/observability/alertmanager/alertmanager.production.example.yml`
  - 生产配置生成脚本：`scripts/observability/generate-alertmanager-config.ps1`
- SLO burn-rate 已扩展为 5m/30m 双窗口，并具备 warning/critical 分级告警
- compose + host-agent 双模式观测冒烟通过证据已沉淀到：
  - `docs/observability-validation.md`
  - `docs/observability-validation.zh-CN.md`

### P2-3 原型痕迹清理

- backend README 中过时的原型遗留描述已移除
- 前端应用壳与 e2e 页面锚点已不再使用 `Linux Panel Prototype` 文案
- root README 已补上 observability 命令与文档入口
- README / roadmap / observability 文档中的历史原型措辞与重复说明已对齐收口

## 后续加固（Post-P2，非阻塞）

1. 按团队值班制度把最终告警目的地接入到真实 on-call 通道
2. 基于真实流量持续调优去重/升级窗口与 SLO 阈值
3. 若后续排障深度需要，再补浏览器侧 tracing

## 当前不优先的方向

- UI 美化或视觉重做
- 在运维主链路未补齐前继续扩页面
- 在运维治理未固化前过早把项目表述成“全面生产就绪”
