# Nowledge Mem Snap

[English](README.md) | [简体中文](README.zh-CN.md)

Self-hosted backup service for Nowledge Mem.

It backs up each logged-in user's private configuration to S3-compatible storage and WebDAV targets. The preferred source is Nowledge Mem's application-level Data Transfer API. Directory sources are supported for operator-managed Docker volume snapshots, but only under explicitly allowed roots.

## Features

- Multi-user isolation: sources, targets, schedules, export options, backup strategies, tasks, and run history are per user.
- Configuration is stored in the database. Users manage it from the web UI; no local JSON config file is required.
- First-run setup wizard, or admin bootstrap by environment variable.
- Password login and optional OIDC login.
- User profile settings: nickname plus avatar URL or uploaded base64 image.
- Sources:
  - `nowledgemem_api`: portable Mem export ZIP via `github.com/lib-x/nowledgemem-go`.
  - `directory`: ZIP of an allowed directory, intended for mounted Docker volumes.
- Remote Nowledge Mem sources and directory sources can be tested from the UI before saving.
- Targets:
  - S3/R2-compatible storage via `github.com/fclairamb/afero-s3`.
  - WebDAV via `github.com/lib-x/aferodav` and an HTTP WebDAV adapter.
- Reusable export options for Nowledge Mem portable archives.
- Reusable backup cleanup policies: no cleanup, keep latest N, keep recent N days, keep after date, or keep before date.
- Daily, weekly, and one-time schedules. Schedule times use the process `TZ`.
- Optional AES-GCM encrypted backup packages per task.
- Tasks compose a source, target set, schedule, export option, and backup cleanup policy.
- Run history cleanup by count and age.
- Structured `slog` logs to stdout and a rotating file via lumberjack.
- Embedded React UI built with `animal-island-ui`.
- ent ORM + SQLite default database, using the same `entsqlite` style as `cfui`.

## Docker

```bash
docker compose up -d
```

Open `http://localhost:14335`. If no admin env vars are set, the setup wizard creates the first administrator.

Published images are built by GitHub Actions and pushed to Docker Hub and GitHub Container Registry:

```bash
docker pull czyt/nowledge-mem-snap:v0.1.2
docker pull ghcr.io/ca-x/nowledge-mem-snap:v0.1.2
```

Image tags:

- `vX.Y.Z`, `X.Y.Z`, `X.Y`: pushed from version tags such as `v0.1.2`.
- `latest`: latest published version tag.
- `sha-<commit>`: immutable commit image.

Useful environment variables:

```bash
DATA_DIR=/app/data
PORT=14335
TZ=UTC
NMEM_SNAP_DATABASE_URL=

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

Target layout has two levels:

- Target `root_prefix`: the remote root directory/prefix inside the bucket or WebDAV account.
- Task `object_prefix`: the task-specific path template under that root, for example `nowledge-mem/{task}/{timestamp}`.

Path template tokens: `{task}` / `{task_name}` use the task display name, `{task_id}` uses the internal UUID, `{date}` uses UTC `YYYY-MM-DD`, and `{timestamp}` uses UTC `YYYYMMDDTHHMMSSZ`.

Automatic remote cleanup only scans the stable directory derived from the task `object_prefix` under the target `root_prefix`, and only removes backup objects ending in `.zip` or `.zip.aes.json`.

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

The default database is `DATA_DIR/data.db`. You can override the DSN with `NMEM_SNAP_DATABASE_URL` or `DATABASE_URL`.

The web UI follows the setup flow: sources, targets, schedules, export options, backup strategies, tasks, run history, and settings. Users do not edit raw JSON configuration or internal record identifiers.

## GitHub Actions

- `.github/workflows/ci.yml`: installs Node/Go dependencies, builds the embedded web UI, verifies generated ent code, runs Go tests, and builds all Go packages.
- `.github/workflows/binary.yml`: builds standalone binaries for Linux, Windows, and macOS when a `v*` tag is pushed. Version tags create a draft GitHub release and upload binary archives.
- `.github/workflows/docker.yml`: builds multi-arch Docker images for `linux/amd64` and `linux/arm64`.
  - Push tag `v*`: builds and pushes semantic version tags to Docker Hub and GHCR.
