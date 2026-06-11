# Nowledge Mem Snap

[English](README.md) | [简体中文](README.zh-CN.md)

Nowledge Mem Snap 是一个自托管的 Nowledge Mem 备份服务。

它可以把每个登录用户自己的备份配置同步到 S3 兼容存储和 WebDAV。推荐 source 是 Nowledge Mem 官方 Data Transfer API 导出的可移植 ZIP；目录 source 主要用于 Docker 场景下的运维级目录快照，并且只能访问显式允许的挂载根目录。

## 界面截图

PC：

![Nowledge Mem Snap PC 界面](screenshot/README-zh-CN-pc.png)

手机：

![Nowledge Mem Snap 手机界面](screenshot/README-zh-CN-mobile.png)

## 功能

- 多用户隔离：source、target、schedule、导出选项、备份清理策略、task、运行历史按用户/租户隔离。
- 配置全部存数据库，用户通过 Web UI 表单管理，不需要编辑本地 JSON 配置文件。
- 首次启动设置向导；也支持通过环境变量初始化管理员。
- 密码登录和可选 OIDC 登录。
- 用户资料：昵称、头像 URL、上传图片自动转 base64 存库。
- Source：
  - `nowledgemem_api`：通过 `github.com/lib-x/nowledgemem-go` 导出 Nowledge Mem 可移植 ZIP。
  - `directory`：压缩允许目录，适合挂载 Nowledge Mem Docker 的 `data` / `config` 目录做运维快照。
- 远程 Nowledge Mem source、目录 source，以及 S3/WebDAV target 都可以在 UI 里点击按钮测试。
- Target：
  - S3/R2 兼容存储，基于 `github.com/fclairamb/afero-s3`。
  - WebDAV，基于 `github.com/lib-x/aferodav` 和本项目的 HTTP WebDAV 适配。
- 可复用的 Nowledge Mem 导出选项，用来选择可移植压缩包包含哪些数据。
- 可复用的备份清理策略：不清理、保留最近 N 份、保留最近 N 天、保留某日期之后、保留某日期之前。
- 计划：支持每天、每周、单次执行；执行时间按进程 `TZ` 解释。
- 每个任务可选 AES-GCM 加密备份包。
- 任务只负责组合来源、目标、计划、导出选项和备份清理策略。
- 运行历史按条数和天数自动清理，避免数据库无限膨胀。
- 使用 `slog` 输出结构化日志，并通过 lumberjack 写文件和自动轮转。
- 嵌入式 React UI，使用 `animal-island-ui`。
- 使用 ent ORM，默认 SQLite 数据库，风格对齐 `cfui` 的 `entsqlite` 用法。

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

- `vX.Y.Z`、`X.Y.Z`、`X.Y`：推送版本 tag 时生成，例如 `v0.1.8`。
- `latest`：最新发布的版本 tag。
- `sha-<commit>`：不可变的 commit 镜像。

常用环境变量：

```bash
DATA_DIR=/app/data
PORT=14335
TZ=UTC

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

远端对象位置分两层：

- Target `root_prefix`：bucket 或 WebDAV 账号下的远端根目录/前缀。
- Task `object_prefix`：该任务自己的对象路径模板，例如 `nowledge-mem/{task}/{timestamp}`。

路径模板变量：`{task}` / `{task_name}` 使用任务显示名称，`{task_id}` 使用内部 UUID，`{date}` 使用 UTC `YYYY-MM-DD`，`{timestamp}` 使用 UTC `YYYYMMDDTHHMMSSZ`。

自动清理远端备份时，只会扫描 `target.root_prefix + task.object_prefix` 推导出的稳定任务目录，并且只删除 `.zip` 或 `.zip.aes.json` 备份对象，不会扫描整个 bucket 或 WebDAV 根目录。

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

Web UI 按使用流程提供 source、target、schedule、导出选项、备份清理策略、task、运行历史和设置页面。用户不需要编辑原始 JSON 配置，也不需要输入内部记录标识。

## GitHub Actions

- `.github/workflows/ci.yml`：安装 Node/Go 依赖，构建嵌入式前端，校验 ent 生成代码，运行 Go 测试，并构建 Go 包。
- `.github/workflows/binary.yml`：推送 `v*` tag 时构建 Linux、Windows、macOS 独立二进制；推送版本 tag 时会创建 draft GitHub Release 并上传二进制压缩包。
- `.github/workflows/docker.yml`：构建 `linux/amd64` 和 `linux/arm64` 多架构 Docker 镜像。
  - 推送 `v*` tag：自动构建并推送语义化版本镜像到 Docker Hub 和 GHCR。
