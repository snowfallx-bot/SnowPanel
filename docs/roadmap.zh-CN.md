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

## 进行中

### P2-2 生产化观测能力

- Prometheus 指标与基线告警规则已具备
- Alertmanager 基线路由已具备
- OTEL tracing 基线现已具备：
  - backend HTTP spans
  - backend gRPC client spans
  - core-agent gRPC server spans
  - OTel Collector -> Jaeger 管线

仍待完成：

- 在 compose / host-agent 模式下做 tracing 端到端实测
- 接入真实告警通知渠道
- 校准告警去重、升级策略与 SLO/SLI 阈值

### P2-3 原型痕迹清理

- backend README 中过时的 placeholder 描述已移除
- root README 已补上 observability 命令与文档入口
- 仍有少量历史措辞与重复说明需要继续清理

## 下一步优先级

1. 在具备 Docker 的环境验证 tracing 闭环
2. 完成生产化告警投递与阈值调优
3. 继续清理过时文档、placeholder 措辞与重复说明

## 当前不优先的方向

- UI 美化或视觉重做
- 在运维主链路未补齐前继续扩页面
- 过早把项目表述成“生产就绪”
