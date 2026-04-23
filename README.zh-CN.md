# SnowPanel | 雪面板

[![CI](https://github.com/snowfallx-bot/SnowPanel/actions/workflows/ci.yml/badge.svg)](https://github.com/snowfallx-bot/SnowPanel/actions/workflows/ci.yml)
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL%202.0-brightgreen.svg)](https://www.mozilla.org/MPL/2.0/)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](backend/go.mod)
[![Rust](https://img.shields.io/badge/Rust-Edition%202021-000000?logo=rust&logoColor=white)](core-agent/Cargo.toml)
[![React](https://img.shields.io/badge/React-18.3-61DAFB?logo=react&logoColor=black)](frontend/package.json)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.8-3178C6?logo=typescript&logoColor=white)](frontend/package.json)
[![Docker Compose](https://img.shields.io/badge/Docker_Compose-v2-1488C6?logo=docker&logoColor=white)](docker-compose.yml)

> “苍山负雪，明烛天南。”

[English](README.md) | **简体中文**

SnowPanel 是一个 Linux 服务器运维面板，采用 Vibe Coding 而成。

**注意！本项目为 GuaiTech 的 AI 编程实验性项目，所有代码均为 AI 生成，我们不能保证代码的安全性与健壮性，请谨慎在生产环境部署！**

## 架构概览

| 组件 | 技术栈 | 职责 |
| --- | --- | --- |
| `frontend` | React + TypeScript + Vite | 管理后台 UI |
| `backend` | Go + Gin + GORM + JWT + RBAC | HTTP API 与业务逻辑 |
| `core-agent` | Rust + gRPC | 主机能力层（文件、服务、Docker、Cron） |
| `proto` | Protocol Buffers | 共享 gRPC 协议定义 |
| `docs` | Markdown | 架构、开发、安全、部署文档 |

## 仓库结构

```text
.
├── backend      # Go API 服务
├── core-agent   # Rust gRPC Agent
├── deploy       # 部署说明
├── docs         # 架构与开发文档
├── frontend     # React 管理界面
└── proto        # 共享 gRPC 协议
```

## 环境要求

- Docker + Docker Compose v2（推荐）
- 可选本地工具链：
  - Go `1.25+`
  - Rust stable
  - Node.js `22+`

## 快速开始

1. 复制环境变量文件：
   - macOS/Linux: `cp .env.example .env`
   - PowerShell: `Copy-Item .env.example .env`
   - 生产环境提示：设置 `APP_ENV=production`，并显式提供强 `JWT_SECRET` 与 `DEFAULT_ADMIN_PASSWORD`
   - 多实例提示：设置 `LOGIN_ATTEMPT_STORE=redis` 以在 backend 多副本间共享登录锁定状态
2. 启动全部服务：
   - `make up`
3. 访问地址：
   - 前端：`http://127.0.0.1:5173`
   - 后端健康检查：`http://127.0.0.1:8080/health`
   - 后端就绪检查：`http://127.0.0.1:8080/ready`
4. 初始化管理员（仅在数据库为空的首次启动时创建）：
   - 用户名：`DEFAULT_ADMIN_USERNAME`（默认：`admin`）
   - 密码：
     - 若配置了 `DEFAULT_ADMIN_PASSWORD`，使用该值
     - 若开发环境留空，backend 会生成一次性随机密码并写入日志（`docker compose logs backend`）
5. 停止服务：
   - `make down`

## 本地开发

1. 仅启动依赖：
   - 安全默认（不向宿主机暴露 DB/Redis 端口）：`docker compose up -d postgres redis`
   - 本地二进制调试（按需暴露 DB/Redis 端口）：`docker compose -f docker-compose.yml -f docker-compose.local.yml up -d postgres redis`
   - backend 容器 + 宿主机 core-agent：`docker compose -f docker-compose.yml -f docker-compose.host-agent.yml up -d --build`
2. 分别启动各服务：
   - `make agent`
   - `make backend`
   - `make frontend`

常用命令：

- `make up`: 构建并启动所有服务
- `make down`: 停止所有服务
- `make logs`: 查看 compose 日志
- `make lint`: 运行基础静态检查
- `make test`: backend/core-agent 测试 + frontend test/build 检查

## 文档导航

- [架构说明](docs/architecture.zh-CN.md) | [Architecture](docs/architecture.md)
- [开发指南](docs/development.zh-CN.md) | [Development Guide](docs/development.md)
- [API 设计](docs/api-design.zh-CN.md) | [API Design](docs/api-design.md)
- [安全说明](docs/security.zh-CN.md) | [Security Notes](docs/security.md)
- [部署指南](docs/deployment.zh-CN.md) | [Deployment Guide](docs/deployment.md)
- [路线图草案](docs/roadmap.zh-CN.md) | [Roadmap](docs/roadmap.md)

## 贡献与社区

- [Contributing Guide](CONTRIBUTING.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Security Policy](SECURITY.md)
- [Issue Templates](.github/ISSUE_TEMPLATE)
- [Pull Request Template](.github/pull_request_template.md)

## 许可证

本项目基于 [Mozilla Public License 2.0](LICENSE) 发布。
