# Nowledge Mem Snap

[English](README.md) | [简体中文](README.zh-CN.md)

Self-hosted backup and restore service for [Nowledge Mem](https://mem.nowledge.co).

Nowledge Mem Snap gives each user a private workspace for scheduled backups, remote storage, and guided restore into a Nowledge Mem instance. It uses Nowledge Mem's portable export/import API for application-level backups. Directory snapshots are also available for operator-managed Docker volumes, but only from paths you explicitly allow.

## Screenshots

Desktop:

![Nowledge Mem Snap desktop UI](screenshot/README-en-pc.png)

Mobile:

![Nowledge Mem Snap mobile UI](screenshot/README-en-mobile.png)

## Features

- Back up Nowledge Mem as a portable ZIP through the official Data Transfer API.
- Restore from an existing backup object on S3, WebDAV, Google Cloud Storage, or SFTP into a selected Nowledge Mem instance.
- Use a step-by-step restore wizard with object scanning, destination selection, import options, and live progress.
- Optionally encrypt backup packages per task. Restore passwords are entered only when starting a restore and are not stored.
- Schedule daily, weekly, or one-time backups. One-time tasks are disabled automatically after they run.
- Run backups manually from the UI or with the CLI.
- Store backups on S3/R2-compatible storage, WebDAV, Google Cloud Storage, or SFTP, and test connections before saving them.
- Snapshot explicitly allowed local directories, useful for operator-managed Docker volume backups.
- Reuse export presets, retention policies, backup sources, storage targets, and schedules across tasks.
- Clean up remote backups by keeping the latest N, keeping recent N days, keeping after a date, or keeping before a date.
- Keep each user's sources, targets, schedules, tasks, restore jobs, and run history isolated.
- Manage everything from the web UI. No local JSON config file is required.
- Use password login, optional OIDC login, first-run setup, and optional admin bootstrap from environment variables.
- Track backup and restore operations in run history and rotating log files.

## Docker

```bash
docker compose up -d
```

Open `http://localhost:14335`. Edit `example.env` before starting if you want to bootstrap the first admin, enable OIDC, or expose directory sources. If no admin env vars are set, the setup wizard creates the first administrator.

Published images are built by GitHub Actions and pushed to Docker Hub and GitHub Container Registry:

```bash
docker pull czyt/nowledge-mem-snap:latest
docker pull ghcr.io/ca-x/nowledge-mem-snap:latest
```

Image tags:

- `vX.Y.Z`, `X.Y.Z`, `X.Y`: pushed from version tags such as `v0.1.12`.
- `latest`: latest published version tag.
- `sha-<commit>`: immutable commit image.

Useful environment variables:

```bash
DATA_DIR=/app/data
PORT=14335
TZ=UTC

# Optional reverse-proxy subpath. Empty means serving at the domain root.
NMEM_SNAP_BASE_PATH=
# Example:
# NMEM_SNAP_BASE_PATH=/your-prefix

# Database options: sqlite (default), postgres, mysql.
NMEM_SNAP_DATABASE_TYPE=sqlite
NMEM_SNAP_DATABASE_DSN=
# NMEM_SNAP_DATABASE_DSN=file:/app/data/data.db?cache=shared&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(10000)
# NMEM_SNAP_DATABASE_TYPE=postgres
# NMEM_SNAP_DATABASE_DSN=postgres://nowledge_mem_snap:nowledge_mem_snap_password@postgres:5432/nowledge_mem_snap?sslmode=disable
# NMEM_SNAP_DATABASE_TYPE=mysql
# NMEM_SNAP_DATABASE_DSN=nowledge_mem_snap:nowledge_mem_snap_password@tcp(mysql:3306)/nowledge_mem_snap?parseTime=true&charset=utf8mb4&loc=Local

# Optional bootstrap. If omitted, use the setup wizard.
NMEM_SNAP_ADMIN_USERNAME=admin
NMEM_SNAP_ADMIN_PASSWORD=change-me
NMEM_SNAP_SESSION_SECRET=change-this-session-secret

# Default local Nowledge Mem API source.
NMEM_API_URL=http://host.docker.internal:14242
NMEM_API_KEY=nmem_xxx

# Directory sources are disabled unless roots are explicitly listed.
NMEM_SNAP_ALLOWED_SOURCE_ROOTS=mem-data=/mem-data,mem-config=/mem-config

# Optional OIDC.
NMEM_SNAP_OIDC_ENABLED=true
NMEM_SNAP_OIDC_ISSUER_URL=https://issuer.example.com
NMEM_SNAP_OIDC_CLIENT_ID=nowledge-mem-snap
NMEM_SNAP_OIDC_CLIENT_SECRET=secret
NMEM_SNAP_OIDC_REDIRECT_URL=http://localhost:14335/auth/oidc/callback
# With NMEM_SNAP_BASE_PATH=/your-prefix:
# NMEM_SNAP_OIDC_REDIRECT_URL=https://example.com/your-prefix/auth/oidc/callback
NMEM_SNAP_OIDC_ALLOWED_DOMAINS=example.com

# Rotating file logs. Default file is DATA_DIR/logs/nowledge-mem-snap.log.
NMEM_SNAP_LOG_LEVEL=info
NMEM_SNAP_LOG_FILE=/app/data/logs/nowledge-mem-snap.log
NMEM_SNAP_LOG_MAX_SIZE_MB=20
NMEM_SNAP_LOG_MAX_BACKUPS=7
NMEM_SNAP_LOG_MAX_AGE_DAYS=30
NMEM_SNAP_LOG_COMPRESS=true
```

For Nowledge Mem's official Docker layout, mount its host directories read-only:

```yaml
volumes:
  - ./data:/mem-data:ro
  - ./config:/mem-config:ro
environment:
  - NMEM_SNAP_ALLOWED_SOURCE_ROOTS=mem-data=/mem-data,mem-config=/mem-config
```

Use the API source for portable app exports and cross-version/cross-architecture restores. Use directory sources for operator-level snapshots of mounted directories.

Subpath hosting: set `NMEM_SNAP_BASE_PATH` to the public path prefix, for example `/your-prefix`, and configure the reverse proxy to preserve that prefix when forwarding requests to the app. The same prefix must appear in OIDC redirect URLs.

Target layout has two levels:

- Target `root_prefix`: the remote root directory/prefix inside the bucket or WebDAV/SFTP account.
- Task `object_prefix`: the task-specific path template under that root, for example `nowledge-mem/{task}/{timestamp}`.

Supported target types are S3/R2 (`s3`), WebDAV (`webdav`), Google Cloud Storage (`gcs`), and SFTP (`sftp`). GCS and SFTP use Afero's upstream experimental backends. GCS can use service account JSON in `credentials_json` or the target-specific `credentials_json_env`. SFTP supports password or private-key auth, and its `root_prefix` may be relative or absolute; `host_key_sha256` is required unless `insecure_ignore_host_key` is explicitly enabled.

Path template tokens: `{task}` / `{task_name}` use the task display name, `{task_id}` uses the internal UUID, `{date}` uses UTC `YYYY-MM-DD`, and `{timestamp}` uses UTC `YYYYMMDDTHHMMSSZ`.

Automatic remote cleanup only scans the stable directory derived from the task `object_prefix` under the target `root_prefix`, and only removes backup objects ending in `.zip` or `.zip.aes.json`.

Remote restore uses the same saved S3/WebDAV/GCS/SFTP targets and Nowledge Mem API sources:

- Scan requires a non-empty remote prefix; the app will not scan an entire bucket or remote storage root.
- Supported restore objects are portable `.zip` exports and encrypted `.zip.aes.json` packages created by this app.
- Encrypted packages ask for the password only when starting the restore job; the password is not stored.
- Import content flags and `mode` are sent to the Nowledge Mem Data Import API. The default mode is the API default; overwrite/clear-style modes are never selected by default.

Time semantics:

- `TZ` is loaded at process start. The binary embeds IANA timezone data, so names such as `Asia/Shanghai` work even in minimal containers.
- Daily and weekly schedules use `TZ`.
- One-time schedules use `run_at` in `YYYY-MM-DDTHH:MM` form and are interpreted in `TZ` unless an RFC3339 offset is provided. After a one-time schedule runs, the task is automatically disabled.
- `keep_days` uses local time in `TZ`.
- Date-only `keep_after` keeps backups on or after local midnight for that date.
- Date-only `keep_before` keeps only backups before local midnight for that date.

## Local Development

```bash
npm --prefix web ci
npm --prefix web run build
go generate ./internal/persist/ent
go test ./...
go run .
```

CLI one-shot backup:

```bash
go run . backup <tenant> <task>
```

The default database is `DATA_DIR/data.db` with SQLite WAL, foreign keys, normal synchronous mode, and a 10s busy timeout. Use `NMEM_SNAP_DATABASE_TYPE` plus `NMEM_SNAP_DATABASE_DSN` to switch to PostgreSQL or MySQL. The bundled Compose file keeps PostgreSQL and MySQL as commented examples so `docker compose up -d` starts with SQLite by default.

The web UI follows the setup flow: sources, targets, schedules, export options, backup strategies, tasks, restore, run history, and settings. Users do not edit raw JSON configuration or internal record identifiers.

## Technical implementation

- Go backend with `net/http`, embedded static assets, and a React UI.
- Nowledge Mem backup and restore use `github.com/lib-x/nowledgemem-go`.
- S3/R2 storage uses `github.com/fclairamb/afero-s3`; WebDAV uses `github.com/lib-x/aferodav` with this project's HTTP WebDAV adapter; GCS and SFTP use Afero's official experimental `gcsfs` and `sftpfs` packages.
- Configuration and users are stored with ent ORM. SQLite is the default database; PostgreSQL and MySQL are optional.
- Scheduled tasks run through `github.com/lib-x/timewheel` with calendar-time calculation in `internal/schedulecalc`.
- Backup packages use ZIP, optional AES-GCM encryption, and scrypt key derivation.
- Restore jobs run asynchronously in memory, download the selected object, decrypt it when needed, upload it to Nowledge Mem, and poll import status.
- Logs use `slog` and write to stdout plus a rotating file through lumberjack when `NMEM_SNAP_LOG_FILE` is enabled.

## GitHub Actions

- `.github/workflows/ci.yml`: installs Node/Go dependencies, builds the embedded web UI, verifies generated ent code, runs Go tests, and builds all Go packages.
- `.github/workflows/binary.yml`: builds standalone binaries for Linux, Windows, and macOS when a `v*` tag is pushed. Version tags create a draft GitHub release and upload binary archives.
- `.github/workflows/docker.yml`: builds multi-arch Docker images for `linux/amd64` and `linux/arm64`.
  - Push tag `v*`: builds and pushes semantic version tags to Docker Hub and GHCR.
