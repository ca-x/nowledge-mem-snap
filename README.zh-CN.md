# Nowledge Mem Snap

[English](README.md) | [简体中文](README.zh-CN.md)

Nowledge Mem Snap 是一个自托管的 [Nowledge Mem](https://mem.nowledge.co) 备份和恢复服务。

它给每个用户提供独立的备份空间，可以定时备份、保存到远端存储，也可以通过向导恢复到指定的 Nowledge Mem 实例。应用级备份使用 Nowledge Mem 的可移植导出和导入 API。目录快照也支持，主要用于运维管理 Docker volume，并且只能访问显式允许的路径。

## 界面截图

PC：

![Nowledge Mem Snap PC 界面](screenshot/README-zh-CN-pc.png)

手机：

![Nowledge Mem Snap 手机界面](screenshot/README-zh-CN-mobile.png)

## 功能

- 通过官方 Data Transfer API 把 Nowledge Mem 备份成可移植 ZIP。
- 从 S3 或 WebDAV 上已有的备份对象恢复到指定 Nowledge Mem 实例。
- 恢复页面提供向导式流程：扫描对象、选择恢复实例、设置导入选项、查看实时进度。
- 每个任务都可以开启备份包加密。恢复加密包时只在启动恢复任务时输入密码，密码不会保存。
- 支持每天、每周、单次备份。单次任务执行后会自动禁用。
- 可以在 UI 里手动运行备份，也可以通过 CLI 触发单次备份。
- 备份可以保存到 S3/R2 兼容存储或 WebDAV，保存前可以先测试连接。
- 可以快照显式允许的本地目录，适合运维管理 Docker volume 的备份。
- 来源、目标、导出选项、保留策略和计划都可以复用到多个任务。
- 支持远端备份清理：保留最近 N 份、保留最近 N 天、保留某日期之后、保留某日期之前。
- 多用户隔离：每个用户的来源、目标、计划、任务、恢复任务和运行历史互不影响。
- 所有配置都通过 Web UI 管理，不需要编辑本地 JSON 配置文件。
- 支持密码登录、可选 OIDC 登录、首次启动设置向导，也支持环境变量初始化管理员。
- 备份和恢复操作会记录到运行历史和轮转日志文件，方便后续排查。

## Docker

```bash
docker compose up -d
```

打开 `http://localhost:14335`。如果需要自动初始化第一个管理员、启用 OIDC，或开放目录 source，请先编辑 `example.env`。如果没有设置管理员初始化环境变量，程序会进入设置向导创建第一个管理员。

GitHub Actions 会自动构建并推送镜像到 Docker Hub 和 GitHub Container Registry：

```bash
docker pull czyt/nowledge-mem-snap:latest
docker pull ghcr.io/ca-x/nowledge-mem-snap:latest
```

镜像标签规则：

- `vX.Y.Z`、`X.Y.Z`、`X.Y`：推送版本 tag 时生成，例如 `v0.1.12`。
- `latest`：最新发布的版本 tag。
- `sha-<commit>`：不可变的 commit 镜像。

常用环境变量：

```bash
DATA_DIR=/app/data
PORT=14335
TZ=UTC

# 可选：反向代理子路径。留空表示挂在域名根路径。
NMEM_SNAP_BASE_PATH=
# 示例：
# NMEM_SNAP_BASE_PATH=/your-prefix

# 数据库选项：sqlite（默认）、postgres、mysql。
NMEM_SNAP_DATABASE_TYPE=sqlite
NMEM_SNAP_DATABASE_DSN=
# NMEM_SNAP_DATABASE_DSN=file:/app/data/data.db?cache=shared&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(10000)
# NMEM_SNAP_DATABASE_TYPE=postgres
# NMEM_SNAP_DATABASE_DSN=postgres://nowledge_mem_snap:nowledge_mem_snap_password@postgres:5432/nowledge_mem_snap?sslmode=disable
# NMEM_SNAP_DATABASE_TYPE=mysql
# NMEM_SNAP_DATABASE_DSN=nowledge_mem_snap:nowledge_mem_snap_password@tcp(mysql:3306)/nowledge_mem_snap?parseTime=true&charset=utf8mb4&loc=Local

# 可选：自动初始化第一个管理员。留空则使用页面设置向导。
NMEM_SNAP_ADMIN_USERNAME=admin
NMEM_SNAP_ADMIN_PASSWORD=change-me
NMEM_SNAP_SESSION_SECRET=change-this-session-secret

# 默认 Nowledge Mem API source。
NMEM_API_URL=http://host.docker.internal:14242
NMEM_API_KEY=nmem_xxx

# 目录 source 默认禁用，必须显式列出允许根目录。
NMEM_SNAP_ALLOWED_SOURCE_ROOTS=mem-data=/mem-data,mem-config=/mem-config

# 可选 OIDC。
NMEM_SNAP_OIDC_ENABLED=true
NMEM_SNAP_OIDC_ISSUER_URL=https://issuer.example.com
NMEM_SNAP_OIDC_CLIENT_ID=nowledge-mem-snap
NMEM_SNAP_OIDC_CLIENT_SECRET=secret
NMEM_SNAP_OIDC_REDIRECT_URL=http://localhost:14335/auth/oidc/callback
# 如果 NMEM_SNAP_BASE_PATH=/your-prefix：
# NMEM_SNAP_OIDC_REDIRECT_URL=https://example.com/your-prefix/auth/oidc/callback
NMEM_SNAP_OIDC_ALLOWED_DOMAINS=example.com

# 日志轮转。默认文件是 DATA_DIR/logs/nowledge-mem-snap.log。
NMEM_SNAP_LOG_LEVEL=info
NMEM_SNAP_LOG_FILE=/app/data/logs/nowledge-mem-snap.log
NMEM_SNAP_LOG_MAX_SIZE_MB=20
NMEM_SNAP_LOG_MAX_BACKUPS=7
NMEM_SNAP_LOG_MAX_AGE_DAYS=30
NMEM_SNAP_LOG_COMPRESS=true
```

如果要备份 Nowledge Mem 官方 Docker 部署目录，可以只读挂载它的宿主机目录：

```yaml
volumes:
  - ./data:/mem-data:ro
  - ./config:/mem-config:ro
environment:
  - NMEM_SNAP_ALLOWED_SOURCE_ROOTS=mem-data=/mem-data,mem-config=/mem-config
```

建议优先使用 API source 做应用级可移植导出，适合跨版本、跨架构恢复。目录 source 更适合运维级目录快照。

子路径托管：把 `NMEM_SNAP_BASE_PATH` 设置成公开访问路径前缀，例如 `/your-prefix`，并让反向代理转发到应用时保留这个前缀。OIDC redirect URL 也必须带同一个前缀。

远端对象位置分两层：

- Target `root_prefix`：bucket 或 WebDAV 账号下的远端根目录/前缀。
- Task `object_prefix`：该任务自己的对象路径模板，例如 `nowledge-mem/{task}/{timestamp}`。

路径模板变量：`{task}` / `{task_name}` 使用任务显示名称，`{task_id}` 使用内部 UUID，`{date}` 使用 UTC `YYYY-MM-DD`，`{timestamp}` 使用 UTC `YYYYMMDDTHHMMSSZ`。

自动清理远端备份时，只会扫描 `target.root_prefix + task.object_prefix` 推导出的稳定任务目录，并且只删除 `.zip` 或 `.zip.aes.json` 备份对象，不会扫描整个 bucket 或 WebDAV 根目录。

远端恢复复用已保存的 S3/WebDAV target 和 Nowledge Mem API source：

- 扫描对象时必须填写非空远端前缀；程序不会扫描整个 bucket 或 WebDAV 根目录。
- 支持恢复 Nowledge Mem 可移植 `.zip` 导出包，以及本应用生成的 `.zip.aes.json` 加密包。
- 加密包只在启动恢复任务时输入密码；密码不会保存。
- 导入内容选项和 `mode` 会按 Nowledge Mem Data Import API 发送。默认模式使用 API 默认值；覆盖、清空这类危险模式默认不选。

时间语义：

- 程序启动时读取 `TZ`。二进制内嵌 IANA timezone data，所以在极简容器里也可以使用 `Asia/Shanghai` 这类时区名。
- 每天、每周定时任务按 `TZ` 计算。
- 单次任务的 `run_at` 使用 `YYYY-MM-DDTHH:MM` 格式，默认按 `TZ` 解释；如果填 RFC3339 且带 offset，则按 offset 解释。单次任务执行后会自动禁用对应 task。
- `keep_days` 按 `TZ` 的本地时间计算。
- 日期型 `keep_after` 会保留该本地日期 00:00 起及之后的备份。
- 日期型 `keep_before` 只保留该本地日期 00:00 之前的备份。

## 本地开发

```bash
npm --prefix web ci
npm --prefix web run build
go generate ./internal/persist/ent
go test ./...
go run .
```

命令行单次备份：

```bash
go run . backup <tenant> <task>
```

默认数据库是 `DATA_DIR/data.db`，SQLite DSN 默认启用 WAL、外键、normal synchronous 和 10 秒 busy timeout。可以通过 `NMEM_SNAP_DATABASE_TYPE` 加 `NMEM_SNAP_DATABASE_DSN` 切换到 PostgreSQL 或 MySQL；随附的 Compose 文件只把 PostgreSQL 和 MySQL 作为注释示例保留，所以 `docker compose up -d` 默认使用 SQLite。

Web UI 按使用流程提供 source、target、schedule、导出选项、备份清理策略、task、恢复、运行历史和设置页面。用户不需要编辑原始 JSON 配置，也不需要输入内部记录标识。

## 技术实现

- 后端使用 Go `net/http`，内嵌静态资源和 React 前端。
- Nowledge Mem 备份和恢复使用 `github.com/lib-x/nowledgemem-go`。
- S3/R2 存储使用 `github.com/fclairamb/afero-s3`；WebDAV 使用 `github.com/lib-x/aferodav` 和本项目的 HTTP WebDAV 适配。
- 配置和用户存储使用 ent ORM。默认数据库是 SQLite，也支持 PostgreSQL 和 MySQL。
- 定时任务使用 `github.com/lib-x/timewheel`，日历时间计算放在 `internal/schedulecalc`。
- 备份包使用 ZIP，可选 AES-GCM 加密和 scrypt 密钥派生。
- 恢复任务以内存异步任务执行：下载远端对象、必要时解密、上传到 Nowledge Mem，并轮询导入状态。
- 日志使用 `slog`，默认写 stdout；启用 `NMEM_SNAP_LOG_FILE` 后同时写入 lumberjack 轮转日志文件。

## GitHub Actions

- `.github/workflows/ci.yml`：安装 Node/Go 依赖，构建嵌入式前端，校验 ent 生成代码，运行 Go 测试，并构建 Go 包。
- `.github/workflows/binary.yml`：推送 `v*` tag 时构建 Linux、Windows、macOS 独立二进制；推送版本 tag 时会创建 draft GitHub Release 并上传二进制压缩包。
- `.github/workflows/docker.yml`：构建 `linux/amd64` 和 `linux/arm64` 多架构 Docker 镜像。
  - 推送 `v*` tag：自动构建并推送语义化版本镜像到 Docker Hub 和 GHCR。
