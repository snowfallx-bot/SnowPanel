# SnowPanel

[![CI](https://github.com/snowfallx-bot/SnowPanel/actions/workflows/ci.yml/badge.svg)](https://github.com/snowfallx-bot/SnowPanel/actions/workflows/ci.yml)
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL%202.0-brightgreen.svg)](https://www.mozilla.org/MPL/2.0/)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](backend/go.mod)
[![Gin](https://img.shields.io/badge/Gin-1.10-008ECF)](backend/go.mod)
[![GORM](https://img.shields.io/badge/GORM-1.30-0E5A8A)](backend/go.mod)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white)](docker-compose.yml)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?logo=redis&logoColor=white)](docker-compose.yml)
[![JWT](https://img.shields.io/badge/JWT-v5-000000?logo=jsonwebtokens&logoColor=white)](backend/go.mod)
[![Rust](https://img.shields.io/badge/Rust-Edition%202021-000000?logo=rust&logoColor=white)](core-agent/Cargo.toml)
[![Tokio](https://img.shields.io/badge/Tokio-1.41-333333)](core-agent/Cargo.toml)
[![Tonic](https://img.shields.io/badge/Tonic-0.12-4A5568)](core-agent/Cargo.toml)
[![gRPC](https://img.shields.io/badge/gRPC-API-244C5A?logo=grpc&logoColor=white)](proto/README.md)
[![Protocol Buffers](https://img.shields.io/badge/Protocol_Buffers-Proto3-336791)](proto/agent/v1/agent.proto)
[![React](https://img.shields.io/badge/React-18.3-61DAFB?logo=react&logoColor=black)](frontend/package.json)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.8-3178C6?logo=typescript&logoColor=white)](frontend/package.json)
[![Vite](https://img.shields.io/badge/Vite-5.4-646CFF?logo=vite&logoColor=white)](frontend/package.json)
[![Tailwind CSS](https://img.shields.io/badge/Tailwind_CSS-3.4-06B6D4?logo=tailwindcss&logoColor=white)](frontend/package.json)
[![TanStack Query](https://img.shields.io/badge/TanStack_Query-5.68-FF4154)](frontend/package.json)
[![Zustand](https://img.shields.io/badge/Zustand-5.0-5C4B51)](frontend/package.json)
[![Docker](https://img.shields.io/badge/Docker-Containerized-2496ED?logo=docker&logoColor=white)](docker-compose.yml)
[![Docker Compose](https://img.shields.io/badge/Docker_Compose-v2-1488C6?logo=docker&logoColor=white)](docker-compose.yml)

语言: [English](README.md) | **简体中文**

SnowPanel 是一个 Linux 服务器面板原型项目（风格类似 BT Panel / 1Panel），采用 monorepo 组织。  
项目将 UI/API 层与主机能力层进行清晰拆分，便于独立演进和维护。

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
2. 启动全部服务：
   - `make up`
3. 访问地址：
   - 前端：`http://127.0.0.1:5173`
   - 后端健康检查：`http://127.0.0.1:8080/health`
4. 默认管理员（数据库为空时自动创建）：
   - 用户名：`admin`
   - 密码：`admin123456`
5. 停止服务：
   - `make down`

## 本地开发

1. 仅启动依赖：
   - `docker compose up -d postgres redis`
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

- [Architecture](docs/architecture.md)
- [Development Guide](docs/development.md)
- [Security Notes](docs/security.md)
- [Deployment Notes](docs/deployment.md)
- [Roadmap](docs/roadmap.md)

## 贡献与社区

- [Contributing Guide](CONTRIBUTING.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Security Policy](SECURITY.md)
- [Issue Templates](.github/ISSUE_TEMPLATE)
- [Pull Request Template](.github/pull_request_template.md)

## 许可证

本项目基于 [Mozilla Public License 2.0](LICENSE) 发布。
