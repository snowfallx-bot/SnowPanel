【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮针对“前端页面能打开，但输入正确 admin 密码后返回 `network error`”继续补了一层后端跨域兜底，避免旧前端配置仍直连 `:8080` 时被浏览器拦掉。

本次核心完成项

1. 前端 API 访问策略调整：
   - `frontend/src/lib/http.ts`
     - `VITE_API_BASE_URL` 为空或 `/` 时，默认走同源请求，而不是回退到 `http://127.0.0.1:8080`
     - 目的：避免用户从自己电脑浏览器访问服务器前端时，请求被错误发到“用户自己机器的 127.0.0.1”
   - `frontend/vite.config.ts`
     - 改为从 `vite` 导入 `defineConfig/loadEnv`
     - 新增 Vite 代理：
       - `/api` -> backend
       - `/health` -> backend
       - `/ready` -> backend
     - 代理目标来自 `VITE_API_PROXY_TARGET`，默认 `http://127.0.0.1:8080`

2. Docker / 安装器默认配置修正：
   - `docker-compose.yml`
     - `VITE_API_BASE_URL` 默认改为空
     - 新增 `VITE_API_PROXY_TARGET=${VITE_API_PROXY_TARGET:-http://backend:8080}`
   - `.env.example`
     - `VITE_API_BASE_URL=`
     - `VITE_API_PROXY_TARGET=http://127.0.0.1:8080`
   - `deploy/one-click/ubuntu-25.10/install.sh`
     - 版本更新为 `0.5.2`
     - 安装器生成 `.env` 时：
       - `VITE_API_BASE_URL` 置空
       - `VITE_API_PROXY_TARGET` 设为 `http://backend:8080`
     - 这样一键安装后的远程浏览器登录不再误指向本机 `127.0.0.1`

3. Backend CORS 兜底：
   - `backend/internal/middleware/cors.go`
     - 新增全局 CORS 中间件
     - 对带 `Origin` 的请求回显 `Access-Control-Allow-Origin`
     - 允许 `Authorization`、`Content-Type` 等常用头
     - 自动处理 `OPTIONS` 预检为 `204`
   - `backend/internal/api/router.go`
     - 全局接入 `middleware.CORS()`
   - `backend/internal/middleware/cors_test.go`
     - 新增预检与普通请求场景测试
   - 目的：
     - 即使当前线上 frontend 仍在直连 `http://<server>:8080`
     - backend 也不会因为缺少 CORS 而在浏览器里表现为 `network error`

4. 文档同步：
   - 更新 `frontend/README.md`
   - 更新 `docs/deployment.md`
   - 更新 `docs/deployment.zh-CN.md`
   - 更新 `deploy/one-click/ubuntu-25.10/README.md`
   - 更新 `deploy/one-click/ubuntu-25.10/README.zh-CN.md`
   - 新增说明：frontend 默认走同源 API + Vite proxy，避免远程登录 `network error`

本轮修改文件

- `frontend/src/lib/http.ts`
- `frontend/vite.config.ts`
- `docker-compose.yml`
- `.env.example`
- `deploy/one-click/ubuntu-25.10/install.sh`
- `frontend/README.md`
- `docs/deployment.md`
- `docs/deployment.zh-CN.md`
- `deploy/one-click/ubuntu-25.10/README.md`
- `deploy/one-click/ubuntu-25.10/README.zh-CN.md`
- `backend/internal/middleware/cors.go`
- `backend/internal/middleware/cors_test.go`
- `backend/internal/api/router.go`
- `.claude/change-cache.md`

本地验证

- `bash -n deploy/one-click/ubuntu-25.10/install.sh` 通过。
- `npm exec tsc -b`（`frontend/`）通过。
- `npm exec vite build`（`frontend/`）通过。
- Go 工具链当前环境不可用，未执行 backend 单测；已完成 CORS 中间件 diff 复核。
- `npm test` 未通过，但失败原因是当前 Windows/npm 环境下 `rolldown` 可选原生绑定缺失，与本次改动无直接关系。

commit摘要

待提交：
- `fix(frontend): use same-origin api proxy for remote login`
- `fix(backend): allow cors preflight for browser api access`

希望接下来的 AI 做什么

1. 在服务器上 `git pull` 后重建 frontend，确认远程浏览器登录不再出现 `network error`。
2. 同时重建 backend，使新的 CORS 中间件生效。
3. 若用户需要立刻恢复，可直接在 `/opt/snowpanel/.env` 中设置：
   - `VITE_API_BASE_URL=`
   - `VITE_API_PROXY_TARGET=http://backend:8080`
   然后重建 frontend 容器。
4. 若后续考虑去掉 Vite dev server 作为运行时，再把这套同源代理迁到 Nginx/Caddy 等正式反向代理。

by: gpt-5.4
