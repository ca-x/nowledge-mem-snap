# Remote Restore Wizard Design

Date: 2026-06-11

## Goal

Add a Restore tab that guides a logged-in user through restoring a portable Nowledge Mem export from an existing S3 or WebDAV target into a selected Nowledge Mem API source.

The feature is a remote data transfer workflow, not the reverse of a backup task. It reuses saved targets and Nowledge Mem API sources, but it does not reuse schedules, backup tasks, retention policies, or backup run history.

## Non-Goals

- Do not accept ad-hoc S3 or WebDAV credentials in the restore wizard.
- Do not support directory source snapshots as importable archives.
- Do not scan an entire bucket or WebDAV root with an empty prefix.
- Do not store restore encryption passwords, API keys, S3 secrets, WebDAV passwords, or archive contents.
- Do not add persistent restore history in the first version.
- Do not add restore scheduling.

## User Flow

1. Open the Restore tab.
2. Select an existing S3 or WebDAV target.
3. Enter a non-empty remote prefix and scan, or manually enter a full object path.
4. Select a `.zip` or `.zip.aes.json` object.
5. Select an existing enabled `nowledgemem_api` source as the destination instance.
6. Configure import options matching the Nowledge Mem Data Import API.
7. Confirm and start restore.
8. Watch step progress until the restore job completes or fails.

## Supported Objects

- `.zip`: treated as a portable Nowledge Mem export and uploaded directly to the destination instance.
- `.zip.aes.json`: decrypted with the supplied password, then uploaded as a ZIP.

Other file types are not shown by object scanning and are rejected by restore start.

## Import Options

The UI mirrors `github.com/lib-x/nowledgemem-go` `UploadImportRequest` fields:

- `mode`
- `include_memories`
- `include_threads`
- `include_messages`
- `include_entities`
- `include_labels`
- `include_sources`
- `include_communities`
- `include_skills`
- `include_edges`
- `include_working_memory`
- `include_working_memory_archive`
- `include_source_files`

The content flags default to enabled. The import mode defaults to the safest append or merge mode supported by the Nowledge Mem API. Destructive modes, if supported by the API, must be visibly marked as dangerous and must not be selected by default.

The implementation should not invent separate `overwrite` or `clear` request fields unless the Nowledge Mem API exposes them.

## Backend Design

Add `internal/restore`.

Responsibilities:

- List candidate restore objects from an existing target.
- Download and validate a selected object.
- Decrypt `.zip.aes.json` envelopes using `archive.Decrypt`.
- Start a Nowledge Mem `Data.UploadImport`.
- Poll `Data.ImportStatus` until the target import completes or fails.
- Keep in-memory restore job state for the current process lifetime.

The package depends on:

- `internal/config` for saved target/source config.
- `internal/storage` for S3/WebDAV filesystem access.
- `internal/archive` for encryption envelope handling.
- `github.com/lib-x/nowledgemem-go` for destination import.

It does not depend on `internal/backup`, `internal/scheduler`, or `internal/tasktimer`.

## API Contract

All endpoints are authenticated under `/api/`.

### `POST /api/restore/objects`

Request:

```json
{
  "target_key": "target-id",
  "prefix": "nowledge-mem/main/"
}
```

Rules:

- `prefix` is required and must not be empty.
- The target must belong to the current tenant.
- Existing target secrets are merged before opening the remote filesystem.
- Only `.zip` and `.zip.aes.json` files are returned.

Response:

```json
{
  "objects": [
    {
      "name": "nowledge-mem/main/backup.zip",
      "size_bytes": 12345,
      "modified_at": "2026-06-11T05:00:00Z",
      "encrypted": false
    }
  ]
}
```

### `POST /api/restore/jobs`

Request:

```json
{
  "target_key": "target-id",
  "object_name": "nowledge-mem/main/backup.zip",
  "destination_source_key": "source-id",
  "encryption_password": "",
  "import_options": {
    "mode": "",
    "include_memories": true,
    "include_threads": true,
    "include_messages": true,
    "include_entities": true,
    "include_labels": true,
    "include_sources": true,
    "include_communities": true,
    "include_skills": true,
    "include_edges": true,
    "include_working_memory": true,
    "include_working_memory_archive": true,
    "include_source_files": true
  }
}
```

Rules:

- The target must belong to the current tenant.
- The destination source must belong to the current tenant, be enabled, and have type `nowledgemem_api`.
- `.zip.aes.json` requires a non-empty password.
- The password is used only in memory for this request.
- The job runs asynchronously.

Response:

```json
{
  "job_id": "restore-job-id"
}
```

### `GET /api/restore/jobs/{id}`

Response:

```json
{
  "id": "restore-job-id",
  "state": "importing",
  "stage": "importing",
  "target_key": "target-id",
  "object_name": "nowledge-mem/main/backup.zip",
  "destination_source_key": "source-id",
  "encrypted": false,
  "size_bytes": 12345,
  "mem_job_id": "nowledge-mem-job-id",
  "progress": 0.6,
  "imported": 10,
  "skipped": 1,
  "failed": 0,
  "message": "Importing",
  "error": "",
  "started_at": "2026-06-11T05:01:00Z",
  "finished_at": null
}
```

States:

- `queued`
- `downloading`
- `decrypting`
- `uploading`
- `importing`
- `completed`
- `failed`
- `cancelled`

### Optional `POST /api/restore/jobs/{id}/cancel`

First-version cancellation may only cancel the local restore context and status polling. If Nowledge Mem has no import cancellation API, the UI must state that target-side import cancellation is not guaranteed after upload begins.

## Frontend Design

Add a Restore tab to the dashboard. Keep restore UI outside `main.tsx` except for tab registration and state wiring.

Suggested files:

- `web/src/pages/RestorePage.tsx`
- `web/src/restore/RestoreWizard.tsx`
- `web/src/restore/RestoreTargetStep.tsx`
- `web/src/restore/RestoreObjectStep.tsx`
- `web/src/restore/RestoreDestinationStep.tsx`
- `web/src/restore/RestoreOptionsStep.tsx`
- `web/src/restore/RestoreProgressStep.tsx`
- `web/src/restore/types.ts`

The Restore tab is a guided operation console, not a configuration editor. The UX must answer five questions clearly:

- Where is the archive coming from?
- Which remote object will be restored?
- Which Nowledge Mem instance will receive the data?
- Which import mode and content flags will be used?
- What stage is the restore currently in?

Wizard steps:

1. Remote target
2. Backup object
3. Destination
4. Import options
5. Restore progress

The page uses a visible step progress indicator. On mobile, steps must wrap or stack without overflowing. All new labels, help text, validation messages, and errors go through `web/src/i18n.tsx`.

### Page Layout

Desktop layout:

- Main column: the active wizard step.
- Side column: a sticky restore summary showing target, object, encryption state, destination, import mode, selected content groups, and readiness warnings.
- Footer row: previous, next, scan, and start actions, depending on the active step.

Mobile layout:

- Stepper appears above the active step.
- Restore summary collapses below the step content.
- Buttons wrap cleanly and keep destructive actions visually separated.

### Step UX

Remote target:

- Show existing S3/WebDAV targets as selectable rows or compact cards.
- Show target name, type, remote root prefix, and enabled state.
- If no target exists, show an empty state that tells the user to create a target in the Targets tab.
- Do not expose or edit target secrets here.

Backup object:

- Primary path: enter a non-empty prefix and scan.
- Secondary path: manually enter a full object path.
- Scan results are tabular or list-based, not decorative cards. Show object name, size, modified time, and encrypted status.
- Only `.zip` and `.zip.aes.json` are selectable.
- Selecting `.zip.aes.json` immediately reveals a password field for this restore only.
- Empty prefix scanning is disabled and explained inline.

Destination:

- Show enabled `nowledgemem_api` sources only.
- Use user-facing language such as "Destination Nowledge Mem instance" instead of "source".
- Show source name and API URL, but never show API keys.
- If no destination exists, show an empty state that tells the user to create a Nowledge Mem API source in the Sources tab.

Import options:

- Use a segmented control or radio group for import `mode`; do not use free text.
- Default to the safest append or merge mode supported by the Nowledge Mem API.
- If destructive modes are supported by the API, place them in a danger area, keep them unselected by default, and require explicit confirmation.
- Show import content flags as a checkbox grid.
- Provide "select all", "clear", and "recommended defaults" actions.
- Defaults should restore all content flags unless the API documents a safer default.

Confirm and progress:

- Before starting, show a complete summary: target, object path, encrypted yes/no, destination API URL, import mode, and selected content flags.
- After starting, freeze previous inputs and show restore stages: downloading, decrypting, uploading, importing, completed or failed.
- Show Nowledge Mem import progress plus imported, skipped, and failed counts when available.
- Errors remain visible in the progress step; they are not only transient toast messages.

### Component Boundaries

Keep restore state local to `RestorePage` or `RestoreWizard` unless it becomes shared app state. Presentation components should receive explicit props and avoid reaching into global config directly.

Recommended split:

- `RestorePage`: loads config-derived target/source lists and hosts the wizard.
- `RestoreWizard`: owns wizard step state, draft restore request, and navigation.
- `RestoreStepper`: renders step state and labels.
- `RestoreSummary`: renders sticky/collapsible readiness summary.
- `RestoreTargetStep`: target selection.
- `RestoreObjectStep`: prefix scan, manual object path, object table, encryption password prompt.
- `RestoreDestinationStep`: destination Nowledge Mem instance selection.
- `RestoreOptionsStep`: mode and content flags.
- `RestoreProgressStep`: start status, polling output, terminal state.
- `restoreDefaults.ts`: default import option values and content flag definitions.
- `types.ts`: restore-specific TypeScript contracts.

The dashboard `main.tsx` should only import `RestorePage`, register the tab, and pass current config lists if needed.

### Visual Language

Use the existing Animal Island UI style, but make the Restore page denser and more operational than the CRUD tabs.

- Current step: primary teal.
- Completed step: success green.
- Warning or missing readiness: yellow.
- Dangerous import mode: error red.
- Encrypted object: lock icon plus muted warning text.

Avoid decorative card grids for scan results. Object selection is a comparison task, so it should be compact and scannable.

### Accessibility

- All wizard navigation uses buttons or form controls with visible labels.
- The stepper exposes the active step with `aria-current`.
- Error messages are rendered near the relevant field and should be reachable by screen readers.
- Progress status uses a live region for stage changes.
- Buttons remain keyboard-accessible and maintain focus order across steps.

## Error Handling

User-facing errors should be specific:

- No target selected.
- Prefix is required for scanning.
- Object path is required.
- Object must end with `.zip` or `.zip.aes.json`.
- Encrypted object requires a password.
- Destination source must be an enabled Nowledge Mem API source.
- Remote object cannot be read.
- Encrypted object cannot be decrypted.
- Archive upload failed.
- Destination import failed.

Implementation logs should include tenant, target key, destination source key, object name, job id, stage, and error. Logs must not include secrets or encryption passwords.

## Security

- Reuse current session authentication and tenant isolation.
- Do not expose raw remote filesystem errors with secrets embedded in URLs.
- Validate object names with the same path rules used by storage writes.
- Limit scanning to the provided prefix.
- Avoid logging full credentials, passwords, API keys, or object content.
- Treat Nowledge Mem API responses as untrusted external responses and handle missing fields safely.

## Tests

Backend:

- `archive.Decrypt` round-trips encrypted envelopes and rejects wrong passwords.
- Object listing filters by suffix and rejects empty prefix.
- Restore start rejects wrong file type, missing encrypted password, missing target, and non-Nowledge destination source.
- Restore job uploads plain ZIP to a test Nowledge Mem server and tracks import status.
- Restore job decrypts `.zip.aes.json` before upload.

Frontend:

- Restore wizard validates required selections before advancing.
- Import content flags serialize to API payload.
- Encrypted object selection requires a password.
- Progress step renders state transitions and terminal errors.

Verification:

- `npm --prefix web run build`
- `go test ./...`
- `go build -trimpath ./...`
- `docker compose -f docker-compose.yml config`

## First Implementation Slice

Implement the smallest complete path:

1. `archive.Decrypt`.
2. `internal/restore` object listing and plain ZIP restore.
3. API endpoints for list/start/status.
4. Restore tab and wizard for saved target, object path, destination source, import options, and progress.
5. Add encrypted object support after the plain ZIP path is tested.
