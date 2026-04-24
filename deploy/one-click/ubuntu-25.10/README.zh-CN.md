# 一键安装（Ubuntu 25.10）

语言: [English](README.md) | **简体中文**

该安装脚本面向 **Ubuntu 25.10**，部署模式为：

- `core-agent` 以宿主机 systemd 服务运行（推荐模式）
- backend/frontend/postgres/redis 通过 docker compose 运行（`docker-compose.host-agent.yml`）

## 脚本

- `install.sh`

## 使用方式

```bash
curl -fsSL https://raw.githubusercontent.com/snowfallx-bot/SnowPanel/main/deploy/one-click/ubuntu-25.10/install.sh -o install.sh
sudo bash install.sh
```

或在本地仓库目录执行：

```bash
sudo bash deploy/one-click/ubuntu-25.10/install.sh
```

## 常用参数

```bash
sudo bash install.sh \
  --install-dir /opt/snowpanel \
  --branch main \
  --backend-port 8080 \
  --frontend-port 5173 \
  --admin-username admin \
  --admin-email admin@example.com \
  --docker-pull-retries 5
```

关键参数：

- `--admin-password`：显式指定初始管理员密码
- `--jwt-secret`：显式指定 JWT 密钥
- `--docker-registry-mirror`：配置 Docker 镜像加速地址（Docker Hub 不稳定地区建议使用）
- `--docker-pull-retries`：镜像拉取与 compose 启动的重试次数
- `--postgres-image` / `--redis-image`：覆盖主运行镜像
- `--postgres-image-fallback` / `--redis-image-fallback`：主镜像拉取失败时使用的回退镜像
- `--force-unsupported`：允许在非 Ubuntu 25.10 上继续（不推荐）

如果你所在区域持续拉取 `*-alpine` 失败，可直接改为非 alpine 标签：

```bash
sudo bash install.sh --postgres-image postgres:16 --redis-image redis:7
```

脚本还包含自动兜底：当主镜像和回退镜像都拉取失败，且 Docker 配置了 `registry-mirrors` 时，会临时移除 mirror 并再重试一次拉取。

每次安装时，脚本还会在 `postgres` healthy 之后以幂等方式重新执行一次基线 schema，方便重复安装和已经保留 `postgres_data` 数据卷的主机继续完成部署。

## 输出结果

脚本会将生成的凭据写入：

- `/root/.snowpanel/installer-output.env`

安装完成后默认访问：

- Frontend：`http://127.0.0.1:<FRONTEND_PORT>`
- Backend 健康检查：`http://127.0.0.1:<BACKEND_PORT>/health`

安装器默认会把 frontend 配置为同源 API 模式，并由 Vite 容器将 `/api`、`/health`、`/ready` 代理到 backend 服务，这样可避免远程浏览器访问时把 `127.0.0.1` 错指到用户自己电脑而导致登录失败。

## 注意事项

- 进入正式生产前，请复核并收紧：
  - `/etc/snowpanel/core-agent.env`
  - `${INSTALL_DIR}/.env`
- 一键安装完成后，如果要重建或查看应用栈日志，请使用 `make up-host-agent` / `make logs-host-agent`。不要退回普通 `docker compose up` / `make up`，否则 backend 会丢失 `docker-compose.host-agent.yml` 覆盖，重新连回已禁用的容器版 `core-agent`。
- 如果 backend 健康检查失败，脚本退出前会自动输出 `docker compose ps`、backend 日志以及 `core-agent` 的状态与日志，便于直接定位原因。
- 对外开放前请先限制 backend/core-agent 端口访问范围。
