【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮按新目标实现了 Ubuntu 25.10 的“一键安装”sh 脚本（Host-Agent 推荐模式），并补齐部署文档入口。

本次核心完成项

1. 一键安装脚本：
   - 新增 `deploy/one-click/ubuntu-25.10/install.sh`（已设可执行权限）
   - 关键能力：
     - OS 检测：默认只允许 `ubuntu 25.10`，支持 `--force-unsupported`
     - 自动安装依赖：`docker`/`compose plugin`、`rust`、构建依赖
     - 自动获取/更新仓库到 `/opt/snowpanel`（可参数化）
     - 自动准备 `.env`，注入 Host-Agent 模式关键变量
     - 自动生成强 `JWT_SECRET` 与 bootstrap 管理员密码（可显式传参覆盖）
     - 构建并安装 `core-agent` 到 systemd
     - 启动 compose（`docker-compose.yml + docker-compose.host-agent.yml`）
     - 自动做 `/health` 和 `/ready` 验证
     - 输出并落盘安装凭据到 `/root/.snowpanel/installer-output.env`
   - 稳定性补强：
     - 修复 `pipefail` 下随机串生成的潜在 SIGPIPE 失败
     - Docker apt repo 对 codename 不可用时回退 `noble`
     - 新增参数防呆校验（必填值、端口范围、安装目录安全）

2. 文档与入口：
   - 新增：
     - `deploy/one-click/ubuntu-25.10/README.md`
     - `deploy/one-click/ubuntu-25.10/README.zh-CN.md`
   - 更新：
     - `deploy/README.md`
     - `deploy/README.zh-CN.md`
     - `docs/deployment.md`
     - `docs/deployment.zh-CN.md`
   - 已在部署文档中添加“一键安装”入口链接。

本轮修改文件

- `deploy/one-click/ubuntu-25.10/install.sh`
- `deploy/one-click/ubuntu-25.10/README.md`
- `deploy/one-click/ubuntu-25.10/README.zh-CN.md`
- `deploy/README.md`
- `deploy/README.zh-CN.md`
- `docs/deployment.md`
- `docs/deployment.zh-CN.md`

本地验证

- 由于当前执行环境是 Windows，无法在本机直接执行 Ubuntu 25.10 安装脚本做端到端验证（这是当前交付风险点）。
- 文档链接与脚本内容已静态检查；脚本可执行位已设置。

commit摘要

待提交：
- `feat(deploy): add ubuntu 25.10 one-click installer`

希望接下来的 AI 做什么

1. 在全新 Ubuntu 25.10 VM 上执行端到端冒烟：
   - `sudo bash deploy/one-click/ubuntu-25.10/install.sh`
   - 验证 `core-agent` systemd、`docker compose ps`、`/health`、`/ready`
2. 补一份“交付测试 checklist”文档（安装、登录、核心模块 smoke）。
3. 根据真实测试日志修复脚本兼容性问题（若出现 Docker 源、Rust 构建或 systemd 权限差异）。

by: gpt-5.4
