【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮针对 Ubuntu 一键安装中镜像拉取异常后的二次故障做了修复（包括 Docker start-limit、镜像变量被污染、mirror 配置干扰）。

本次核心完成项

1. 安装脚本稳定性增强（`deploy/one-click/ubuntu-25.10/install.sh`）：
   - 版本更新为 `0.5.0`
   - 新增 `recover_docker_daemon`：
     - 在重试前执行 `systemctl reset-failed docker`，避免触发 start-limit 后 Docker 无法恢复
   - `run_with_retry` 从“直接 restart”改为“recover 流程”
   - 新增 `docker_pull_image`：
     - 将 `docker pull` 输出重定向到 stderr，避免命令替换捕获进变量导致 `POSTGRES_IMAGE` 被污染
   - 新增 `clear_registry_mirrors_if_present`：
     - 当主镜像+回退镜像都失败时，若检测到 `/etc/docker/daemon.json` 存在 `registry-mirrors`，自动备份并临时移除 mirror 后再重试
   - `pull_image_with_fallback` 已接入上述能力，失败路径更可恢复
   - Docker 初始化后增加 active 校验：
     - `recover_docker_daemon || die ...`

2. 文档同步：
   - 更新 `deploy/one-click/ubuntu-25.10/README.md`
   - 更新 `deploy/one-click/ubuntu-25.10/README.zh-CN.md`
   - 新增说明：主+回退镜像失败且配置了 mirror 时，安装器会临时移除 mirror 并重试。

本轮修改文件

- `deploy/one-click/ubuntu-25.10/install.sh`
- `deploy/one-click/ubuntu-25.10/README.md`
- `deploy/one-click/ubuntu-25.10/README.zh-CN.md`
- `.claude/change-cache.md`

本地验证

- Windows 环境下 `bash -n` 不可执行（`Bash/Service/CreateInstance/E_ACCESSDENIED`），未能进行本地 bash 语法检查。
- 已完成脚本 diff 复核。

commit摘要

待提交：
- `fix(deploy): harden installer docker recovery and mirror fallback`

希望接下来的 AI 做什么

1. 在 Ubuntu 25.10 上跑一次完整安装，确认以下链路：
   - 主镜像失败 -> 回退镜像 ->（必要时）临时移除 mirror -> 成功拉取
2. 若仍失败，采集：
   - `systemctl status docker --no-pager`
   - `journalctl -u docker -n 100 --no-pager`
   - `/etc/docker/daemon.json`
3. 若网络环境长期不稳定，考虑在文档中追加区域化 mirror 推荐列表。

by: gpt-5.4
